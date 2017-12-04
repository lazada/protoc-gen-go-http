package descriptor

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	gendesc "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
	options "google.golang.org/genproto/googleapis/api/annotations"
)

// Registry is a registry of information extracted from plugin.CodeGeneratorRequest.
type Registry struct {
	// msgs is a mapping from fully-qualified message name to descriptor
	msgs map[string]*Message

	// enums is a mapping from fully-qualified enum name to descriptor
	enums map[string]*Enum

	// files is a mapping from file path to descriptor
	files map[string]*File

	// prefix is a prefix to be inserted to golang package paths generated from proto package names.
	prefix string

	// pkgMap is a user-specified mapping from file path to proto package.
	pkgMap map[string]string

	// pkgAliases is a mapping from package aliases to package paths in go which are already taken.
	pkgAliases map[string]string
}

// NewRegistry returns a new Registry.
func NewRegistry() *Registry {
	return &Registry{
		msgs:       make(map[string]*Message),
		enums:      make(map[string]*Enum),
		files:      make(map[string]*File),
		pkgMap:     make(map[string]string),
		pkgAliases: make(map[string]string),
	}
}

// Load loads definitions of services, methods, messages, enumerations and fields from "req".
func (r *Registry) Load(req *plugin_go.CodeGeneratorRequest) error {
	for _, file := range req.GetProtoFile() {
		r.loadFile(file)
	}

	var targetPkg string
	for _, name := range req.FileToGenerate {
		target := r.files[name]
		if target == nil {
			return fmt.Errorf("no such file: %s", name)
		}
		name := packageIdentityName(target.FileDescriptorProto)
		if targetPkg == "" {
			targetPkg = name
		} else {
			if targetPkg != name {
				return fmt.Errorf("inconsistent package names: %s %s", targetPkg, name)
			}
		}

		if err := r.loadServices(target); err != nil {
			return err
		}
	}
	return nil
}

// loadFile loads messages, enumerations and fields from "file".
// It does not loads services and methods in "file".  You need to call
// loadServices after loadFiles is called for all files to load services and methods.
func (r *Registry) loadFile(file *gendesc.FileDescriptorProto) {
	pkg := GoPackage{
		Path: r.goPackagePath(file),
		Name: defaultGoPackageName(file),
	}
	if err := r.ReserveGoPackageAlias(pkg.Name, pkg.Path); err != nil {
		for i := 0; ; i++ {
			alias := fmt.Sprintf("%s_%d", pkg.Name, i)
			if err := r.ReserveGoPackageAlias(alias, pkg.Path); err == nil {
				pkg.Alias = alias
				break
			}
		}
	}
	f := &File{
		FileDescriptorProto: file,
		GoPkg:               pkg,
	}

	r.files[file.GetName()] = f
	r.registerMsg(f, nil, file.GetMessageType())
	r.registerEnum(f, nil, file.GetEnumType())
}

func (r *Registry) registerMsg(file *File, outerPath []string, msgs []*gendesc.DescriptorProto) {
	for i, md := range msgs {
		m := &Message{
			File:            file,
			Outers:          outerPath,
			DescriptorProto: md,
			Index:           i,
		}
		for _, fd := range md.GetField() {
			m.Fields = append(m.Fields, &Field{
				Message:              m,
				FieldDescriptorProto: fd,
			})
		}
		file.Messages = append(file.Messages, m)
		r.msgs[m.FQMN()] = m
		glog.V(1).Infof("register name: %s", m.FQMN())

		var outers []string
		outers = append(outers, outerPath...)
		outers = append(outers, m.GetName())
		r.registerMsg(file, outers, m.GetNestedType())
		r.registerEnum(file, outers, m.GetEnumType())
	}
}

func (r *Registry) registerEnum(file *File, outerPath []string, enums []*gendesc.EnumDescriptorProto) {
	for i, ed := range enums {
		e := &Enum{
			File:                file,
			Outers:              outerPath,
			EnumDescriptorProto: ed,
			Index:               i,
		}
		file.Enums = append(file.Enums, e)
		r.enums[e.FQEN()] = e
		glog.V(1).Infof("register enum name: %s", e.FQEN())
	}
}

// LookupMsg looks up a message type by "name".
// It tries to resolve "name" from "location" if "name" is a relative message name.
func (r *Registry) LookupMsg(location, name string) (*Message, error) {
	glog.V(1).Infof("lookup %s from %s", name, location)
	if strings.HasPrefix(name, ".") {
		m, ok := r.msgs[name]
		if !ok {
			return nil, fmt.Errorf("no message found: %s", name)
		}
		return m, nil
	}

	if !strings.HasPrefix(location, ".") {
		location = fmt.Sprintf(".%s", location)
	}
	components := strings.Split(location, ".")
	for len(components) > 0 {
		fqmn := strings.Join(append(components, name), ".")
		if m, ok := r.msgs[fqmn]; ok {
			return m, nil
		}
		components = components[:len(components)-1]
	}
	return nil, fmt.Errorf("no message found: %s", name)
}

// LookupEnum looks up a enum type by "name".
// It tries to resolve "name" from "location" if "name" is a relative enum name.
func (r *Registry) LookupEnum(location, name string) (*Enum, error) {
	glog.V(1).Infof("lookup enum %s from %s", name, location)
	if strings.HasPrefix(name, ".") {
		e, ok := r.enums[name]
		if !ok {
			return nil, fmt.Errorf("no enum found: %s", name)
		}
		return e, nil
	}

	if !strings.HasPrefix(location, ".") {
		location = fmt.Sprintf(".%s", location)
	}
	components := strings.Split(location, ".")
	for len(components) > 0 {
		fqen := strings.Join(append(components, name), ".")
		if e, ok := r.enums[fqen]; ok {
			return e, nil
		}
		components = components[:len(components)-1]
	}
	return nil, fmt.Errorf("no enum found: %s", name)
}

// LookupFile looks up a file by name.
func (r *Registry) LookupFile(name string) (*File, error) {
	f, ok := r.files[name]
	if !ok {
		return nil, fmt.Errorf("no such file given: %s", name)
	}
	return f, nil
}

// AddPkgMap adds a mapping from a .proto file to proto package name.
func (r *Registry) AddPkgMap(file, protoPkg string) {
	r.pkgMap[file] = protoPkg
}

// SetPrefix registers the prefix to be added to go package paths generated from proto package names.
func (r *Registry) SetPrefix(prefix string) {
	r.prefix = prefix
}

// ReserveGoPackageAlias reserves the unique alias of go package.
// If succeeded, the alias will be never used for other packages in generated go files.
// If failed, the alias is already taken by another package, so you need to use another
// alias for the package in your go files.
func (r *Registry) ReserveGoPackageAlias(alias, pkgpath string) error {
	if taken, ok := r.pkgAliases[alias]; ok {
		if taken == pkgpath {
			return nil
		}
		return fmt.Errorf("package name %s is already taken. Use another alias", alias)
	}
	r.pkgAliases[alias] = pkgpath
	return nil
}

// goPackagePath returns the go package path which go files generated from "f" should have.
// It respects the mapping registered by AddPkgMap if exists. Or use go_package as import path
// if it includes a slash,  Otherwide, it generates a path from the file name of "f".
func (r *Registry) goPackagePath(f *gendesc.FileDescriptorProto) string {
	name := f.GetName()
	if pkg, ok := r.pkgMap[name]; ok {
		return path.Join(r.prefix, pkg)
	}

	gopkg := f.Options.GetGoPackage()
	idx := strings.LastIndex(gopkg, "/")
	if idx >= 0 {
		if sc := strings.LastIndex(gopkg, ";"); sc > 0 {
			gopkg = gopkg[:sc+1-1]
		}
		return gopkg
	}

	return path.Join(r.prefix, path.Dir(name))
}

// GetAllFQMNs returns a list of all FQMNs
func (r *Registry) GetAllFQMNs() []string {
	var keys []string
	for k := range r.msgs {
		keys = append(keys, k)
	}
	return keys
}

// GetAllFQENs returns a list of all FQENs
func (r *Registry) GetAllFQENs() []string {
	var keys []string
	for k := range r.enums {
		keys = append(keys, k)
	}
	return keys
}

// loadServices registers services and their methods from "targetFile" to "r".
// It must be called after loadFile is called for all files so that loadServices
// can resolve names of message types and their fields.
func (r *Registry) loadServices(file *File) error {
	glog.V(1).Infof("Loading services from %s", file.GetName())
	var svcs []*Service
	for _, sd := range file.GetService() {
		glog.V(2).Infof("Registering %s", sd.GetName())
		svc := &Service{
			File:                   file,
			ServiceDescriptorProto: sd,
		}
		for _, md := range sd.GetMethod() {
			glog.V(2).Infof("Processing %s.%s", sd.GetName(), md.GetName())
			opts, err := extractAPIOptions(md)
			if err != nil {
				glog.Errorf("Failed to extract ApiMethodOptions from %s.%s: %v", svc.GetName(), md.GetName(), err)
				return err
			}
			if opts == nil {
				glog.V(1).Infof("Found non-target method: %s.%s", svc.GetName(), md.GetName())
			}
			meth, err := r.newMethod(svc, md, opts)
			if err != nil {
				return err
			}
			svc.Methods = append(svc.Methods, meth)
		}
		if len(svc.Methods) == 0 {
			continue
		}
		glog.V(2).Infof("Registered %s with %d method(s)", svc.GetName(), len(svc.Methods))
		svcs = append(svcs, svc)
	}
	file.Services = svcs
	return nil
}

func (r *Registry) newMethod(svc *Service, md *gendesc.MethodDescriptorProto, opts *options.HttpRule) (*Method, error) {
	requestType, err := r.LookupMsg(svc.File.GetPackage(), md.GetInputType())
	if err != nil {
		return nil, err
	}
	responseType, err := r.LookupMsg(svc.File.GetPackage(), md.GetOutputType())
	if err != nil {
		return nil, err
	}
	meth := &Method{
		Service:               svc,
		MethodDescriptorProto: md,
		RequestType:           requestType,
		ResponseType:          responseType,
	}

	return meth, nil
}

func extractAPIOptions(meth *gendesc.MethodDescriptorProto) (*options.HttpRule, error) {
	if meth.Options == nil {
		return nil, nil
	}
	if !proto.HasExtension(meth.Options, options.E_Http) {
		return nil, nil
	}
	ext, err := proto.GetExtension(meth.Options, options.E_Http)
	if err != nil {
		return nil, err
	}
	opts, ok := ext.(*options.HttpRule)
	if !ok {
		return nil, fmt.Errorf("extension is %T; want an HttpRule", ext)
	}
	return opts, nil
}

// sanitizePackageName replaces unallowed character in package name
// with allowed character.
func sanitizePackageName(pkgName string) string {
	pkgName = strings.Replace(pkgName, ".", "_", -1)
	pkgName = strings.Replace(pkgName, "-", "_", -1)
	return pkgName
}

// defaultGoPackageName returns the default go package name to be used for go files generated from "f".
// You might need to use an unique alias for the package when you import it.  Use ReserveGoPackageAlias to
// get a unique alias.
func defaultGoPackageName(f *gendesc.FileDescriptorProto) string {
	name := packageIdentityName(f)
	return sanitizePackageName(name)
}

// packageIdentityName returns the identity of packages.
// protoc-gen-grpc-gateway rejects CodeGenerationRequests which contains more than one packages
// as protoc-gen-go-http does.
func packageIdentityName(f *gendesc.FileDescriptorProto) string {
	if f.Options != nil && f.Options.GoPackage != nil {
		gopkg := f.Options.GetGoPackage()
		idx := strings.LastIndex(gopkg, "/")
		if idx < 0 {
			gopkg = gopkg[idx+1:]
		}

		gopkg = gopkg[idx+1:]
		// package name is overrided with the string after the
		// ';' character
		sc := strings.IndexByte(gopkg, ';')
		if sc < 0 {
			return sanitizePackageName(gopkg)

		}
		return sanitizePackageName(gopkg[sc+1:])
	}

	if f.Package == nil {
		base := filepath.Base(f.GetName())
		ext := filepath.Ext(base)
		return strings.TrimSuffix(base, ext)
	}
	return f.GetPackage()
}
