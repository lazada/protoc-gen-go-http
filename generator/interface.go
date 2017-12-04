// Package generator provides an abstract interface to code generators.
package generator

import (
	"github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/lazada/protoc-gen-go-http/descriptor"
)

// Generator is an abstraction of code generators.
type Generator interface {
	// Generate generates output files from input .proto files.
	Generate(targets []*descriptor.File) ([]*plugin_go.CodeGeneratorResponse_File, error)
}
