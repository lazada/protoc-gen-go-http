package generator

import (
	"strings"
	"text/template"
)

var (
	RouterTemplate = template.Must(template.New(`file`).Funcs(template.FuncMap{
		`lower`: strings.ToLower,
	}).Parse(`
package {{ .Package }}

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/lazada/protoc-gen-go-http/codec"
)

type options struct {
	routes		map[string]http.HandlerFunc
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

{{ range $sIdx, $service := .Services }}

type {{ $service.Name }}Router struct {
	srv				*http{{ $service.Name }}Server
	codecBuilder	codec.CodecBuilder
	routes			map[string]http.HandlerFunc
}

func New{{ $service.Name }}Router(srv {{ $service.Name }}Server, codecBuilder codec.CodecBuilder, opts ...option) (*{{ $service.Name }}Router, error) {
	out := &{{ $service.Name }}Router{
		srv:			newHTTP{{ $service.Name }}Server(srv, codecBuilder()),
		codecBuilder:	codecBuilder,
		routes:			make(map[string]http.HandlerFunc),
	}

	defaultOptions := &options{
		routes:	make(map[string]http.HandlerFunc),
	}

	for _, opt := range opts {
		opt(defaultOptions)
	}

	if err := defaultOptions.validate(); err != nil {
		return nil, err
	}

	out.routes = map[string]http.HandlerFunc{
		{{ range $hIdx, $handler := $service.Handlers }}"/{{ lower $service.Name }}/{{ lower $handler.Name }}": out.srv.{{ $handler.Name }},
		{{ end }}
	}

	for route, handler := range defaultOptions.routes {
		if handler == nil {
			continue
		}
		out.routes[route] = handler
	}

	return out, nil
}

func (s *{{ $service.Name }}Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

type http{{ $service.Name }}Server struct {
	srv		{{ $service.Name }}Server
	cdc	codec.Codec
}

func newHTTP{{ $service.Name }}Server(srv {{ $service.Name }}Server, cdc codec.Codec) *http{{ $service.Name }}Server {
	return &http{{ $service.Name }}Server{
		srv: srv,
		cdc: cdc,
	}
}

{{ range $hIdx, $handler := $service.Handlers }}
func (s *http{{ $service.Name }}Server) {{ $handler.Name }}(w http.ResponseWriter, r *http.Request) {
    defer r.Body.Close()
	arg := {{ $handler.Arg }}{}
	err := s.cdc.ReadRequest(r, &arg)
    if err != nil {
        s.cdc.WriteError(w, err)
		return
    }

	grpcResp, err := s.srv.{{ $handler.Name }}(r.Context(), &arg)
	if err != nil {
        s.cdc.WriteError(w, err)
		return
	}

	s.cdc.WriteResponse(w, grpcResp)
}
{{ end }}

{{ end }}
`))
)
