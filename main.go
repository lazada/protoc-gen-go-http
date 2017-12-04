package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/lazada/protoc-gen-go-http/descriptor"
	"github.com/lazada/protoc-gen-go-http/generator"
)

var (
	importPrefix      = flag.String("import_prefix", "", "prefix to be added to go package paths for imported proto files")
	useRequestContext = flag.Bool("request_context", false, "determine whether to use http.Request's context or not")
	allowDeleteBody   = flag.Bool("allow_delete_body", false, "unless set, HTTP DELETE methods may not have a body")
)

func parseReq(r io.Reader) (*plugin_go.CodeGeneratorRequest, error) {
	log.Println("Parsing code generator request")
	input, err := ioutil.ReadAll(r)
	if err != nil {
		log.Printf("Failed to read code generator request: %v", err)
		return nil, err
	}
	req := new(plugin_go.CodeGeneratorRequest)
	if err = proto.Unmarshal(input, req); err != nil {
		log.Printf("Failed to unmarshal code generator request: %v", err)
		return nil, err
	}
	log.Println("Parsed code generator request")
	return req, nil
}

func main() {
	flag.Parse()
	var (
		reg      = descriptor.NewRegistry()
		req, err = parseReq(os.Stdin)
	)
	if err != nil {
		log.Panic(err)
	}
	processParameters(req, reg)

	g := generator.New(reg, *useRequestContext)

	reg.SetPrefix(*importPrefix)
	if err := reg.Load(req); err != nil {
		emitError(err)
		return
	}

	var targets []*descriptor.File
	for _, target := range req.FileToGenerate {
		f, err := reg.LookupFile(target)
		if err != nil {
			log.Panic(err)
		}
		targets = append(targets, f)
	}

	out, err := g.Generate(targets)
	log.Println("Processed code generator request")
	if err != nil {
		emitError(err)
		return
	}
	emitFiles(out)
}

func emitFiles(out []*plugin_go.CodeGeneratorResponse_File) {
	emitResp(&plugin_go.CodeGeneratorResponse{File: out})
}

func emitError(err error) {
	emitResp(&plugin_go.CodeGeneratorResponse{Error: proto.String(err.Error())})
}

func emitResp(resp *plugin_go.CodeGeneratorResponse) {
	buf, err := proto.Marshal(resp)
	if err != nil {
		log.Panic(err)
	}
	if _, err := os.Stdout.Write(buf); err != nil {
		log.Panic(err)
	}
}

func processParameters(req *plugin_go.CodeGeneratorRequest, reg *descriptor.Registry) {
	if req.Parameter != nil {
		for _, p := range strings.Split(req.GetParameter(), ",") {
			spec := strings.SplitN(p, "=", 2)
			if len(spec) == 1 {
				if err := flag.CommandLine.Set(spec[0], ""); err != nil {
					log.Panicf("Cannot set flag %s", p)
				}
				continue
			}
			name, value := spec[0], spec[1]
			if strings.HasPrefix(name, "M") {
				reg.AddPkgMap(name[1:], value)
				continue
			}
			if err := flag.CommandLine.Set(name, value); err != nil {
				log.Panicf("Cannot set flag %s", p)
			}
		}
	}
}
