package rpc

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"moul.io/http2curl"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Version ...
type Version string

// json rpc version
const (
	JSONRPCVersion1 Version = "1.0"
	JSONRPCVersion2 Version = "2.0"
)

// Client ...
type Client struct {
	version        Version
	Client         *http.Client
	Req            *http.Request
	idCounter      uint64
	URL            string
	User           string
	Pass           string
	enableMaxBatch bool
	maxBatchNum    int
	ResultHandler  func(result []byte, destination interface{}) error // 用以更灵活的支持各式返回结果,目前仅不支持批量请求，需要时请自行修改BatchSyncCall并充分测试
}

// Dial ...
func Dial(url string, user string, pass string, certs []byte, version Version) (*Client, error) {
	c, err := DialWithoutAuth(url, certs, version)
	if err != nil {
		return nil, err
	}
	c.Req.SetBasicAuth(user, pass)
	c.User = user
	c.Pass = pass
	return c, nil
}

// DialWithoutAuth ...
func DialWithoutAuth(url string, certs []byte, version Version) (*Client, error) {
	req, err := http.NewRequest("POST", url, nil)
	if err = errors.WithStack(err); err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	ts := DefaultTs

	// Configure TLS if needed.
	var tlsConfig *tls.Config
	if certs != nil {
		if len(certs) > 0 {
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM(certs)
			tlsConfig = &tls.Config{
				RootCAs: pool,
			}
		}
		ts.TLSClientConfig = tlsConfig
	}

	c := Client{
		version: version,
		Client: &http.Client{
			Timeout:   time.Second * 60,
			Transport: ts,
		},
		Req:           req,
		idCounter:     uint64(0),
		URL:           url,
		ResultHandler: DefaultHandler,
	}
	return &c, nil
}

// SetTransport set the Transport
func (c *Client) SetTransport(ts http.RoundTripper) {
	c.Client.Transport = ts
}

// SetTimeout set http timeout
func (c *Client) SetTimeout(timeout time.Duration) *Client {
	c.Client.Timeout = timeout
	return c
}

// SetMaxBatchNum set max batch num
func (c *Client) SetMaxBatchNum(maxBatchNum int) *Client {
	if maxBatchNum > 0 {
		c.enableMaxBatch = true
		c.maxBatchNum = maxBatchNum
	}
	return c
}

// SetResultHandler ..
func (c *Client) SetResultHandler(handler func([]byte, interface{}) error) {
	c.ResultHandler = handler
}

// DialInsecureSkipVerify make client ignore server's certificate chain and host name
func DialInsecureSkipVerify(url string, user string, pass string, version Version) (*Client, error) {
	req, err := http.NewRequest("POST", url, nil)
	if err = errors.WithStack(err); err != nil {
		return nil, err
	}
	req.SetBasicAuth(user, pass)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	ts := DefaultTs

	// Configure TLS if needed.
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	ts.TLSClientConfig = tlsConfig

	c := Client{
		version: version,
		Client: &http.Client{
			Timeout:   time.Second * 60,
			Transport: ts,
		},
		Req:           req,
		idCounter:     uint64(0),
		URL:           url,
		User:          user,
		Pass:          pass,
		ResultHandler: DefaultHandler,
	}
	return &c, nil
}

// DefaultHandler default way to unmarshal
func DefaultHandler(res []byte, target interface{}) error {
	resMsg := jsonRPCReceiveMessage{}
	err := json.Unmarshal(res, &resMsg)
	if err != nil {
		return errors.Wrapf(err, "unmarshaling result: %s", string(res))
	}
	if resMsg.Error != nil {
		return errors.WithStack(resMsg.Error)
	}
	err = json.Unmarshal(resMsg.Result, &target)
	return errors.Wrapf(err, "unmarshaling josn rpc result: %s", string(res))
}

// SyncCallObject ...
func (c *Client) SyncCallObject(res interface{}, method string, params interface{}) error {
	if params == nil {
		params = new(emptyStruct)
	}
	msg, err := c.newMessage(method, params)
	if err != nil {
		return err
	}
	buf, err := c.syncRequest(msg)
	if err != nil {
		return err
	}
	return c.ResultHandler(buf, res)
}

// SyncCall ...
func (c *Client) SyncCall(res interface{}, method string, params ...interface{}) error {
	return c.SyncCallObject(res, method, params)
}

func (c *Client) syncRequest(msg *jsonRPCSendMessage) (buf []byte, err error) {
	body, err := json.Marshal(msg)
	if err = errors.WithStack(err); err != nil {
		return nil, err
	}
	req := c.Req.WithContext(context.Background())
	req.Body = io.NopCloser(bytes.NewBuffer(body))
	req.ContentLength = int64(len(body))

	command, _ := http2curl.GetCurlCommand(req)
	logrus.WithField("tags", "request").Debug(command)

	res, err := c.Client.Do(req)
	if err = errors.WithStack(err); err != nil {
		return nil, err
	}
	defer res.Body.Close()
	buf, err = io.ReadAll(res.Body)
	if res.StatusCode != 200 {
		bodyStr := string(buf)
		if len(buf) > 500 {
			bodyStr = string(buf[:150])
			bodyStr = strings.ToValidUTF8(bodyStr, "") + "   凸(゜皿゜メ)"
		}
		return nil, errors.Errorf("http status code err: %d, msg: %s", res.StatusCode, bodyStr)
	}
	return
}

// BatchSyncCall batch SyncCall and will cut batch request when enableMaxBatch is true and 0 < maxBatchNum < len(batch request)
func (c *Client) BatchSyncCall(batch []BatchElem) (err error) {
	totalLength := len(batch)
	if totalLength == 0 {
		return
	}
	requestList := make([]*jsonRPCSendMessage, totalLength)
	for i := range requestList {
		requestList[i], err = c.newMessage(batch[i].Method, batch[i].Args)
		if err != nil {
			return
		}
	}
	batchNum := len(requestList)
	var buf []byte
	responseList := make([]*jsonRPCReceiveMessage, 0, totalLength)
	if !c.enableMaxBatch || c.maxBatchNum <= 0 || batchNum <= c.maxBatchNum {
		buf, err = c.batchSyncRequest(requestList)
		if err != nil {
			return
		}
		err = json.Unmarshal(buf, &responseList)
		if err != nil {
			err = errors.Errorf("can not paste the content into []*jsonRPCReceiveMessage: %s", string(buf))
			return
		}
	} else {
		for i, j := 0, c.maxBatchNum; true; {
			if batchNum < j {
				j = batchNum
			}

			logrus.Debugf("try batch [%d,%d], total %d", i, j, batchNum)
			buf, err = c.batchSyncRequest(requestList[i:j])
			if err != nil {
				return
			}
			tempResMsgs := make([]*jsonRPCReceiveMessage, 0)
			err = json.Unmarshal(buf, &tempResMsgs)
			if err != nil {
				err = errors.Errorf("can not paste the content into []*jsonRPCReceiveMessage: %s", string(buf))
				return
			}
			responseList = append(responseList, tempResMsgs...)
			i += c.maxBatchNum
			j += c.maxBatchNum
			if i >= batchNum {
				break
			}
		}
	}

	return handleBatchResult(batch, requestList, responseList)
}

func handleBatchResult(batch []BatchElem, requestList []*jsonRPCSendMessage, responseList []*jsonRPCReceiveMessage) error {
	responseMap, err := getResponseMap(responseList)
	if err != nil {
		return err
	}

	var elem *BatchElem
	var req *jsonRPCSendMessage
	var res *jsonRPCReceiveMessage
	var ok bool
	for i := range batch {
		elem = &batch[i]
		req = requestList[i]
		res, ok = responseMap[req.ID]
		if !ok {
			return errors.Errorf("can not found result, resuest id %d, method %s, params %v", req.ID, req.Method, req.Params)
		}
		if res == nil {
			elem.Error = errors.New("not found response")
			return elem.Error
		}
		if res.Error != nil {
			elem.Error = errors.WithStack(res.Error)
			return elem.Error
		}
		if len(res.Result) == 0 {
			elem.Error = errors.New("not found")
			return elem.Error
		}
		elem.Error = errors.WithStack(json.Unmarshal(res.Result, elem.Result))
		if elem.Error != nil {
			return elem.Error
		}
	}
	return nil
}

func (c *Client) batchSyncRequest(msg []*jsonRPCSendMessage) (buf []byte, err error) {
	body, err := json.Marshal(msg)
	if err = errors.WithStack(err); err != nil {
		return nil, err
	}
	req := c.Req.WithContext(context.Background())
	req.Body = io.NopCloser(bytes.NewBuffer(body))
	req.ContentLength = int64(len(body))

	if len(msg) <= 5 {
		command, _ := http2curl.GetCurlCommand(req)
		logrus.WithField("tags", "request").Debug(command)
	}

	res, err := c.Client.Do(req)
	if err = errors.WithStack(err); err != nil {
		return nil, err
	}
	defer res.Body.Close()
	buf, err = io.ReadAll(res.Body)
	if res.StatusCode != 200 {
		bodyStr := string(buf)
		if len(buf) > 500 {
			bodyStr = string(buf[:150])
			bodyStr = strings.ToValidUTF8(bodyStr, "") + "   凸(゜皿゜メ)"
		}
		return nil, errors.Errorf("http status code err: %d, msg: %s", res.StatusCode, bodyStr)
	}
	return
}

func (c *Client) newMessage(method string, param interface{}) (*jsonRPCSendMessage, error) {
	params, err := json.Marshal(param)
	if err = errors.WithStack(err); err != nil {
		return nil, err
	}
	return &jsonRPCSendMessage{Version: string(c.version), ID: c.nextID(), Method: method, Params: params}, nil
}

func (c *Client) nextID() uint64 {
	return atomic.AddUint64(&c.idCounter, 1)
}

func getResponseMap(responseList []*jsonRPCReceiveMessage) (result map[uint64]*jsonRPCReceiveMessage, err error) {
	length := len(responseList)
	result = make(map[uint64]*jsonRPCReceiveMessage, length)
	for i := 0; i < length; i++ {
		responseList[i].id, err = strconv.ParseUint(string(responseList[i].ID), 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "can not parse %s as uint64", string(responseList[i].ID))
		}
		result[responseList[i].id] = responseList[i]
	}
	return result, nil
}
