package python

type List struct {
	SliceType    string
	MethodPrefix string
	InputFormat  func() string
	OutputFormat func(string) string
}
