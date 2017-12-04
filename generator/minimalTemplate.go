package generator

import (
	"strings"
	"text/template"
)

var (
	MinimalTemplate = template.Must(template.New(`file`).Funcs(template.FuncMap{
		`lower`: strings.ToLower,
	}).Parse(`
package {{ .Package }}

import (
	"net/http"

	"github.com/lazada/protoc-gen-go-http/codec"
)

{{ range $sIdx, $service := .Services }}

type HTTP{{ $service.Name }}Server struct {
	srv		{{ $service.Name }}Server
	cdc	codec.Codec
}

func NewHTTP{{ $service.Name }}Server(srv {{ $service.Name }}Server, cdc codec.Codec) *HTTP{{ $service.Name }}Server {
	return &HTTP{{ $service.Name }}Server{
		srv: srv,
		cdc: cdc,
	}
}

{{ range $hIdx, $handler := $service.Handlers }}
func (s *HTTP{{ $service.Name }}Server) {{ $handler.Name }}(w http.ResponseWriter, r *http.Request) {
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
