package descriptor

import (
	"fmt"
	"strings"

	gendesc "github.com/golang/protobuf/protoc-gen-go/descriptor"
	gogen "github.com/golang/protobuf/protoc-gen-go/generator"
)

// GoPackage represents a golang package
type GoPackage struct {
	// Path is the package path to the package.
	Path string
	// Name is the package name of the package
	Name string
	// Alias is an alias of the package unique within the current invokation of grpc-gateway generator.
	Alias string
}

// Standard returns whether the import is a golang standard package.
func (p GoPackage) Standard() bool {
	return !strings.Contains(p.Path, ".")
}

// String returns a string representation of this package in the form of import line in golang.
func (p GoPackage) String() string {
	if p.Alias == "" {
		return fmt.Sprintf("%q", p.Path)
	}
	return fmt.Sprintf("%s %q", p.Alias, p.Path)
}

// File wraps gendesc.FileDescriptorProto for richer features.
type File struct {
	*gendesc.FileDescriptorProto
	// GoPkg is the go package of the go file generated from this file..
	GoPkg GoPackage
	// Messages is the list of messages defined in this file.
	Messages []*Message
	// Enums is the list of enums defined in this file.
	Enums []*Enum
	// Services is the list of services defined in this file.
	Services []*Service
}

// proto2 determines if the syntax of the file is proto2.
func (f *File) proto2() bool {
	return f.Syntax == nil || f.GetSyntax() == "proto2"
}

// Message describes a protocol buffer message types
type Message struct {
	// File is the file where the message is defined
	File *File
	// Outers is a list of outer messages if this message is a nested type.
	Outers []string
	*gendesc.DescriptorProto
	Fields []*Field

	// Index is proto path index of this message in File.
	Index int
}

// FQMN returns a fully qualified message name of this message.
func (m *Message) FQMN() string {
	components := []string{""}
	if m.File.Package != nil {
		components = append(components, m.File.GetPackage())
	}
	components = append(components, m.Outers...)
	components = append(components, m.GetName())
	return strings.Join(components, ".")
}

// GoType returns a go type name for the message type.
// It prefixes the type name with the package alias if
// its belonging package is not "currentPackage".
func (m *Message) GoType(currentPackage string) string {
	var components []string
	components = append(components, m.Outers...)
	components = append(components, m.GetName())

	name := strings.Join(components, "_")
	if m.File.GoPkg.Path == currentPackage {
		return name
	}
	pkg := m.File.GoPkg.Name
	if alias := m.File.GoPkg.Alias; alias != "" {
		pkg = alias
	}
	return fmt.Sprintf("%s.%s", pkg, name)
}

// Enum describes protocol buffer enum types.
type Enum struct {
	// File is the file where the enum is defined
	File *File
	// Outers is a list of outer messages if this enum is a nested type.
	Outers []string
	*gendesc.EnumDescriptorProto

	Index int
}

// FQEN returns a fully qualified enum name of this enum.
func (e *Enum) FQEN() string {
	components := []string{""}
	if e.File.Package != nil {
		components = append(components, e.File.GetPackage())
	}
	components = append(components, e.Outers...)
	components = append(components, e.GetName())
	return strings.Join(components, ".")
}

// Service wraps gendesc.ServiceDescriptorProto for richer features.
type Service struct {
	// File is the file where this service is defined.
	File *File
	*gendesc.ServiceDescriptorProto
	// Methods is the list of methods defined in this service.
	Methods []*Method
}

// Method wraps gendesc.MethodDescriptorProto for richer features.
type Method struct {
	// Service is the service which this method belongs to.
	Service *Service
	*gendesc.MethodDescriptorProto

	// RequestType is the message type of requests to this method.
	RequestType *Message
	// ResponseType is the message type of responses from this method.
	ResponseType *Message
}

// Field wraps gendesc.FieldDescriptorProto for richer features.
type Field struct {
	// Message is the message type which this field belongs to.
	Message *Message
	// FieldMessage is the message type of the field.
	FieldMessage *Message
	*gendesc.FieldDescriptorProto
}

// FieldPath is a path to a field from a request message.
type FieldPath []FieldPathComponent

// String returns a string representation of the field path.
func (p FieldPath) String() string {
	var components []string
	for _, c := range p {
		components = append(components, c.Name)
	}
	return strings.Join(components, ".")
}

// IsNestedProto3 indicates whether the FieldPath is a nested Proto3 path.
func (p FieldPath) IsNestedProto3() bool {
	if len(p) > 1 && !p[0].Target.Message.File.proto2() {
		return true
	}
	return false
}

// RHS is a right-hand-side expression in go to be used to assign a value to the target field.
// It starts with "msgExpr", which is the go expression of the method request object.
func (p FieldPath) RHS(msgExpr string) string {
	l := len(p)
	if l == 0 {
		return msgExpr
	}
	components := []string{msgExpr}
	for i, c := range p {
		if i == l-1 {
			components = append(components, c.RHS())
			continue
		}
		components = append(components, c.LHS())
	}
	return strings.Join(components, ".")
}

// FieldPathComponent is a path component in FieldPath
type FieldPathComponent struct {
	// Name is a name of the proto field which this component corresponds to.
	// TODO(yugui) is this necessary?
	Name string
	// Target is the proto field which this component corresponds to.
	Target *Field
}

// RHS returns a right-hand-side expression in go for this field.
func (c FieldPathComponent) RHS() string {
	return gogen.CamelCase(c.Name)
}

// LHS returns a left-hand-side expression in go for this field.
func (c FieldPathComponent) LHS() string {
	if c.Target.Message.File.proto2() {
		return fmt.Sprintf("Get%s()", gogen.CamelCase(c.Name))
	}
	return gogen.CamelCase(c.Name)
}

var (
	proto3ConvertFuncs = map[gendesc.FieldDescriptorProto_Type]string{
		gendesc.FieldDescriptorProto_TYPE_DOUBLE:  "runtime.Float64",
		gendesc.FieldDescriptorProto_TYPE_FLOAT:   "runtime.Float32",
		gendesc.FieldDescriptorProto_TYPE_INT64:   "runtime.Int64",
		gendesc.FieldDescriptorProto_TYPE_UINT64:  "runtime.Uint64",
		gendesc.FieldDescriptorProto_TYPE_INT32:   "runtime.Int32",
		gendesc.FieldDescriptorProto_TYPE_FIXED64: "runtime.Uint64",
		gendesc.FieldDescriptorProto_TYPE_FIXED32: "runtime.Uint32",
		gendesc.FieldDescriptorProto_TYPE_BOOL:    "runtime.Bool",
		gendesc.FieldDescriptorProto_TYPE_STRING:  "runtime.String",
		// FieldDescriptorProto_TYPE_GROUP
		// FieldDescriptorProto_TYPE_MESSAGE
		// FieldDescriptorProto_TYPE_BYTES
		// TODO(yugui) Handle bytes
		gendesc.FieldDescriptorProto_TYPE_UINT32: "runtime.Uint32",
		// FieldDescriptorProto_TYPE_ENUM
		// TODO(yugui) Handle Enum
		gendesc.FieldDescriptorProto_TYPE_SFIXED32: "runtime.Int32",
		gendesc.FieldDescriptorProto_TYPE_SFIXED64: "runtime.Int64",
		gendesc.FieldDescriptorProto_TYPE_SINT32:   "runtime.Int32",
		gendesc.FieldDescriptorProto_TYPE_SINT64:   "runtime.Int64",
	}

	proto2ConvertFuncs = map[gendesc.FieldDescriptorProto_Type]string{
		gendesc.FieldDescriptorProto_TYPE_DOUBLE:  "runtime.Float64P",
		gendesc.FieldDescriptorProto_TYPE_FLOAT:   "runtime.Float32P",
		gendesc.FieldDescriptorProto_TYPE_INT64:   "runtime.Int64P",
		gendesc.FieldDescriptorProto_TYPE_UINT64:  "runtime.Uint64P",
		gendesc.FieldDescriptorProto_TYPE_INT32:   "runtime.Int32P",
		gendesc.FieldDescriptorProto_TYPE_FIXED64: "runtime.Uint64P",
		gendesc.FieldDescriptorProto_TYPE_FIXED32: "runtime.Uint32P",
		gendesc.FieldDescriptorProto_TYPE_BOOL:    "runtime.BoolP",
		gendesc.FieldDescriptorProto_TYPE_STRING:  "runtime.StringP",
		// FieldDescriptorProto_TYPE_GROUP
		// FieldDescriptorProto_TYPE_MESSAGE
		// FieldDescriptorProto_TYPE_BYTES
		// TODO(yugui) Handle bytes
		gendesc.FieldDescriptorProto_TYPE_UINT32: "runtime.Uint32P",
		// FieldDescriptorProto_TYPE_ENUM
		// TODO(yugui) Handle Enum
		gendesc.FieldDescriptorProto_TYPE_SFIXED32: "runtime.Int32P",
		gendesc.FieldDescriptorProto_TYPE_SFIXED64: "runtime.Int64P",
		gendesc.FieldDescriptorProto_TYPE_SINT32:   "runtime.Int32P",
		gendesc.FieldDescriptorProto_TYPE_SINT64:   "runtime.Int64P",
	}
)
