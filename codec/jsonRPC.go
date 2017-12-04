package codec

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/sourcegraph/jsonrpc2"
)

// define list of json-rpc v2 error code
const (
	E_PARSE       int64 = -32700
	E_INVALID_REQ int64 = -32600
	E_NO_METHOD   int64 = -32601
	E_BAD_PARAMS  int64 = -32602
	E_INTERNAL    int64 = -32603
	E_SERVER      int64 = -32000
)

type jsonRPCoptions struct {
	errorClassifier func(error) int64
}

type jsonRPCoption func(*jsonRPCoptions)

func WithErrorClassifier(errorClassifier func(error) int64) jsonRPCoption {
	return func(opts *jsonRPCoptions) {
		opts.errorClassifier = errorClassifier
	}
}

type JsonRPCCodec struct {
	jsonrpcRequest  *jsonrpc2.Request
	errorClassifier func(error) int64
}

func NewJsonRPCCodec(opts ...jsonRPCoption) Codec {
	defaultOptions := &jsonRPCoptions{}

	for _, opt := range opts {
		opt(defaultOptions)
	}

	return &JsonRPCCodec{
		jsonrpcRequest:  &jsonrpc2.Request{},
		errorClassifier: defaultOptions.errorClassifier,
	}
}

func (c *JsonRPCCodec) Route(r *http.Request) (route string, err error) {
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return "", nil
	}

	c.jsonrpcRequest.UnmarshalJSON(body)

	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	return c.jsonrpcRequest.Method, nil
}

func (c *JsonRPCCodec) ReadRequest(r *http.Request, out interface{}) error {
	if c.jsonrpcRequest.Params == nil {
		return errors.New("request contains nil params")
	}

	var (
		paramsBytes = []byte(*c.jsonrpcRequest.Params)
		decoder     = json.NewDecoder(ioutil.NopCloser(bytes.NewBuffer(paramsBytes)))
	)

	return decoder.Decode(out)
}

func (c *JsonRPCCodec) WriteResponse(w http.ResponseWriter, grpcResp interface{}) error {
	grpcRespBin, err := json.Marshal(grpcResp)
	if err != nil {
		return err
	}

	finResp := &jsonrpc2.Response{
		ID:     c.jsonrpcRequest.ID,
		Meta:   c.jsonrpcRequest.Meta,
		Result: (*json.RawMessage)(&grpcRespBin),
	}

	finRespBin, err := finResp.MarshalJSON()
	if err != nil {
		return err
	}
	_, err = w.Write(finRespBin)

	return err
}

func (c *JsonRPCCodec) WriteError(w http.ResponseWriter, err error) error {
	finResp := &jsonrpc2.Response{
		ID:   c.jsonrpcRequest.ID,
		Meta: c.jsonrpcRequest.Meta,
		Error: &jsonrpc2.Error{
			Message: err.Error(),
		},
	}

	if c.errorClassifier != nil {
		finResp.Error.Code = c.errorClassifier(err)
	} else {
		finResp.Error.Code = E_INTERNAL
	}

	bResp, err := finResp.MarshalJSON()
	if err != nil {
		// ?
	}

	_, err = w.Write(bResp)

	return err
}
