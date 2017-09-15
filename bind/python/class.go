package python

import (
	"github.com/devigned/veil/cgo"
)

type Class struct {
	*cgo.Struct
	binder       *Binder
	Fields       []*Param
	Constructors []*Func
	Methods      []*Func
}

func (c Class) Name() string {
	return c.Named.Obj().Name()
}

func (c Class) MethodName(p *Param) string {
	return c.FieldName(p.underlying)
}

func (c Class) NewMethodName() string {
	return c.Struct.NewMethodName()
}
