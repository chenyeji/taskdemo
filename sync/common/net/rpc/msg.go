package rpc

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

// DefaultTs ...
var DefaultTs = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}).DialContext,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

// BatchElem ...
type BatchElem struct {
	Method string
	Args   interface{}
	Result interface{}
	Error  error
}

type jsonRPCSendMessage struct {
	Version string          `json:"jsonrpc"`
	ID      uint64          `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCReceiveMessage struct {
	Version string          `json:"jsonrpc"`
	ID      json.Number     `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonError      `json:"error,omitempty"`
	id      uint64
}

type jsonError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (err *jsonError) Error() string {
	return fmt.Sprintf("json-rpc error code: %d, msg: %s", err.Code, err.Message)
}

type emptyStruct struct {
}
