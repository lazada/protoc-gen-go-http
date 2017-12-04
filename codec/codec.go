package codec

import (
	"net/http"
)

type Codec interface {
	Route(r *http.Request) (route string, err error)
	ReadRequest(r *http.Request, out interface{}) error
	WriteResponse(w http.ResponseWriter, resp interface{}) error
	WriteError(w http.ResponseWriter, err error) error
}

type CodecBuilder func() Codec
