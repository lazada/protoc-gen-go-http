package generator

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"log"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/lazada/protoc-gen-go-http/descriptor"
	options "google.golang.org/genproto/googleapis/api/annotations"
)

var (
	errNoTargetService = errors.New("no target service defined in the file")
)

const (
	minimal = iota
	router
)

type generator struct {
	reg               *descriptor.Registry
	useRequestContext bool
	withRouter        bool
}

// New returns a new generator which generates plugin files.
func New(reg *descriptor.Registry, useRequestContext bool) Generator {
	return &generator{
		reg:               reg,
		useRequestContext: useRequestContext,
		withRouter:        true,
	}
}

func (g *generator) Generate(targets []*descriptor.File) (files []*plugin_go.CodeGeneratorResponse_File, err error) {
	minimalFiles, err := g.buildFiles(targets, minimal)
	if err != nil {
		return nil, err
	}
	files = append(files, minimalFiles...)

	if g.withRouter {
		routerFiles, err := g.buildFiles(targets, router)
		if err != nil {
			return nil, err
		}
		files = append(files, routerFiles...)
	}

	return files, nil
}

func (g *generator) buildFiles(targets []*descriptor.File, mode int) (files []*plugin_go.CodeGeneratorResponse_File, err error) {
	var (
		fromTemplate *template.Template
		fileName     string
	)

	switch mode {
	case minimal:
		fromTemplate, fileName = MinimalTemplate, "%s.pb.http.go"
	case router:
		fromTemplate, fileName = RouterTemplate, "%s.pb.http.router.go"
	}

	for _, file := range targets {
		log.Printf("Processing %s", file.GetName())

		minimalCode, err := g.generateFrom(file, fromTemplate)
		if err == errNoTargetService {
			log.Printf("%s: %v", file.GetName(), err)
			continue
		}
		if err != nil {
			return nil, err
		}

		formatted, err := format.Source([]byte(minimalCode))
		if err != nil {
			log.Printf("%v: %s", err, minimalCode)
			return nil, err
		}

		var (
			name   = file.GetName()
			ext    = filepath.Ext(name)
			base   = strings.TrimSuffix(name, ext)
			output = fmt.Sprintf(fileName, base)
		)
		files = append(files, &plugin_go.CodeGeneratorResponse_File{
			Name:    proto.String(output),
			Content: proto.String(string(formatted)),
		})
	}

	return files, nil
}

func (g *generator) generateFrom(file *descriptor.File, t *template.Template) (string, error) {
	pkgSeen := make(map[string]bool)
	var imports []descriptor.GoPackage
	tFileInfo := &templateFileInfo{
		Package: file.GoPkg.Name,
	}

	for _, svc := range file.Services {
		tService := &templateService{Name: svc.GetName()}
		tFileInfo.Services = append(tFileInfo.Services, tService)

		for _, m := range svc.Methods {
			if m.GetServerStreaming() || m.GetClientStreaming() {
				continue
			}

			tService.Handlers = append(tService.Handlers, &templateHandler{
				Name: m.GetName(),
				Arg:  m.GetInputType()[1:],
			})

			g.markSeen(file, m, pkgSeen, imports)
		}
	}

	buf := bytes.NewBuffer([]byte{})
	t.Execute(buf, tFileInfo)

	return buf.String(), nil
}

func (g *generator) markSeen(file *descriptor.File, m *descriptor.Method, pkgSeen map[string]bool, imports []descriptor.GoPackage) {
	pkg := m.RequestType.File.GoPkg
	if m.Options == nil || !proto.HasExtension(m.Options, options.E_Http) ||
		pkg == file.GoPkg || pkgSeen[pkg.Path] {
		return
	}
	pkgSeen[pkg.Path] = true
	imports = append(imports, pkg)
}
