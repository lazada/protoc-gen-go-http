package example

import (
	"net/http"

	"github.com/lazada/protoc-gen-go-http/codec"
)

type HTTPExampleServer struct {
	srv ExampleServer
	cdc codec.Codec
}

func NewHTTPExampleServer(srv ExampleServer, cdc codec.Codec) *HTTPExampleServer {
	return &HTTPExampleServer{
		srv: srv,
		cdc: cdc,
	}
}

func (s *HTTPExampleServer) GetPerson(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	arg := Query{}
	err := s.cdc.ReadRequest(r, &arg)
	if err != nil {
		s.cdc.WriteError(w, err)
		return
	}
	grpcResp, err := s.srv.GetPerson(r.Context(), &arg)
	if err != nil {
		s.cdc.WriteError(w, err)
		return
	}
	s.cdc.WriteResponse(w, grpcResp)
}
