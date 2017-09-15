package python

import (
	"github.com/devigned/veil/cgo"
	"strings"
	"github.com/devigned/veil/core"
)

type List struct {
	*cgo.Slice
	MethodPrefix string
	InputFormat  func() string
	OutputFormat func(string) string
}

func (l List) ListTypeName() string {
	return l.Name() + "List"
}

func (l List) Name() string {
	typeString := l.Slice.ElementPackageAliasAndPath(nil)
	typeString = strings.Replace(typeString, "[]", "SliceOf", -1)
	splits := strings.Split(typeString, ".")
	if len(splits) > 1 {
		return splits[len(splits)-1]
	} else {
		return core.ToCap(splits[0])
	}
}
