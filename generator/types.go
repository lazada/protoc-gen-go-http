package generator

type templateFileInfo struct {
	Package  string
	Services []*templateService
}

type templateService struct {
	Name     string
	Handlers []*templateHandler
}

type templateHandler struct {
	Name    string
	Service string
	Arg     string
}
