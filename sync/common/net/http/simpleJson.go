package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"moul.io/http2curl"
)

type resultHandler func(body []byte, destination interface{}) error

// SimpleJSON ...
type SimpleJSON struct {
	client *http.Client
	url    string

	handler resultHandler // 用以更灵活的支持各式返回结果,目前仅不支持批量请求，需要时请自行修改BatchSyncCall并充分测试
}

// NewSimpleJSON ...
func NewSimpleJSON(url string) *SimpleJSON {
	ts := DefaultTs
	return &SimpleJSON{
		client: &http.Client{
			Timeout:   5 * time.Second,
			Transport: ts,
		},
		url:     url,
		handler: DefaultSimpleJSONHandler,
	}
}

func (s *SimpleJSON) GetURL() string {
	return s.url
}

// SetTimeout ste http timeout
func (s *SimpleJSON) SetTimeout(timeout time.Duration) *SimpleJSON {
	s.client.Timeout = timeout
	return s
}

// SetTransport set the Transport
func (s *SimpleJSON) SetTransport(ts http.RoundTripper) {
	s.client.Transport = ts
}

func (s *SimpleJSON) SetResultHandler(handler resultHandler) *SimpleJSON {
	s.handler = handler
	return s
}

// Get ...
func (s *SimpleJSON) Get(tail string, out interface{}) error {
	req, err := http.NewRequest("GET", s.url+tail, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	command, _ := http2curl.GetCurlCommand(req)
	logrus.WithField("tags", "request").Debug(command)

	resp, err := s.client.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	return handleResponseForSimpleJSON(resp, out, s.handler)
}

func (s *SimpleJSON) GetWithHeader(hKey, hValue, tail string, out interface{}) error {
	req, err := http.NewRequest("GET", s.url+tail, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	req.Header.Set(hKey, hValue)

	command, _ := http2curl.GetCurlCommand(req)
	logrus.WithField("tags", "request").Debug(command)

	resp, err := s.client.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	return handleResponseForSimpleJSON(resp, out, s.handler)
}

// Post ...
func (s *SimpleJSON) Post(tail string, in, out interface{}) error {
	marshal, err := json.Marshal(in)
	if err != nil {
		return errors.WithStack(err)
	}

	req, err := http.NewRequest("POST", s.url+tail, bytes.NewReader(marshal))
	if err != nil {
		return errors.WithStack(err)
	}
	req.Header.Set("Content-Type", "application/json")

	command, _ := http2curl.GetCurlCommand(req)
	logrus.WithField("tags", "request").Debug(command)

	resp, err := s.client.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	return handleResponseForSimpleJSON(resp, out, s.handler)
}

// PostString ...
func (s *SimpleJSON) PostString(tail, in string, out interface{}) error {
	req, err := http.NewRequest("POST", s.url+tail, strings.NewReader(in))
	if err != nil {
		return errors.WithStack(err)
	}
	req.Header.Set("Content-Type", "application/json")

	command, _ := http2curl.GetCurlCommand(req)
	logrus.WithField("tags", "request").Debug(command)

	resp, err := s.client.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	return handleResponseForSimpleJSON(resp, out, s.handler)
}

func (s *SimpleJSON) PostShortConn(tail string, in, out interface{}) error {
	marshal, err := json.Marshal(in)
	if err != nil {
		return errors.WithStack(err)
	}

	req, err := http.NewRequest("POST", s.url+tail, bytes.NewReader(marshal))
	if err != nil {
		return errors.WithStack(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Close = true

	command, _ := http2curl.GetCurlCommand(req)
	logrus.WithField("tags", "request").Debug(command)

	resp, err := s.client.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}

	return handleResponseForSimpleJSON(resp, out, s.handler)
}

func handleResponseForSimpleJSON(resp *http.Response, out interface{}, handler resultHandler) error {
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.WithStack(err)
	}
	if resp.StatusCode != 200 {
		bodyStr := string(bodyBytes)
		if len(bodyBytes) > 500 {
			bodyStr = string(bodyBytes[:150])
			bodyStr = strings.ToValidUTF8(bodyStr, "") + "   凸(゜皿゜メ)"
		}
		return errors.Errorf("http status %d != 200\n%s", resp.StatusCode, bodyStr)
	}
	return handler(bodyBytes, out)
}

func DefaultSimpleJSONHandler(body []byte, out interface{}) error {
	err := json.Unmarshal(body, out)
	return errors.WithStack(err)
}
