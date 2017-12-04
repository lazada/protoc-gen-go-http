package example

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/lazada/protoc-gen-go-http/codec"
)

type options struct {
	routes      map[string]http.HandlerFunc
	withSwagger bool
}

func (o *options) validate() error {
	if o.routes == nil {
		return errors.New("nil routes were provided")
	}

	return nil
}

type option func(*options)

// WithRoutes sets handlers to specific routes.
// Set the handler to nil to delete a route.
func WithRoutes(routes map[string]http.HandlerFunc) option {
	return func(opts *options) {
		opts.routes = routes
	}
}

func WithSwagger() option {
	return func(opts *options) {
		opts.withSwagger = true
	}
}

type ExampleRouter struct {
	srv          *httpExampleServer
	codecBuilder codec.CodecBuilder
	routes       map[string]http.HandlerFunc
}

func NewExampleRouter(srv ExampleServer, codecBuilder codec.CodecBuilder, opts ...option) (*ExampleRouter, error) {
	out := &ExampleRouter{
		srv:          newHTTPExampleServer(srv, codecBuilder()),
		codecBuilder: codecBuilder,
		routes:       make(map[string]http.HandlerFunc),
	}

	defaultOptions := &options{
		routes: make(map[string]http.HandlerFunc),
	}

	for _, opt := range opts {
		opt(defaultOptions)
	}

	if err := defaultOptions.validate(); err != nil {
		return nil, err
	}

	out.routes = map[string]http.HandlerFunc{
		"/example/getperson": out.srv.GetPerson,
	}

	for route, handler := range defaultOptions.routes {
		if handler == nil {
			continue
		}
		out.routes[route] = handler
	}

	return out, nil
}

func (s *ExampleRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := s.codecBuilder()

	route, err := c.Route(r)
	if err != nil {
		c.WriteError(w, err)
		return
	}

	handler, ok := s.routes[route]
	if !ok {
		c.WriteError(w, fmt.Errorf("no handler for route %s", route))
		return
	}

	handler(w, r)
}

type httpExampleServer struct {
	srv ExampleServer
	cdc codec.Codec
}

func newHTTPExampleServer(srv ExampleServer, cdc codec.Codec) *httpExampleServer {
	return &httpExampleServer{
		srv: srv,
		cdc: cdc,
	}
}

func (s *httpExampleServer) GetPerson(w http.ResponseWriter, r *http.Request) {
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
