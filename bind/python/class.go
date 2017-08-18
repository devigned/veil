package python

import (
	"github.com/devigned/veil/cgo"
	"go/types"
)

type Class struct {
	*cgo.Struct
	binder       *Binder
	Fields       []*Param
	Constructors []*PyFunc
}

func (c Class) Name() string {
	return c.Struct.Named.Obj().Name()
}

func (c Class) MethodName(p *Param) string {
	return c.FieldName(p.underlying)
}

func (c Class) NewMethodName() string {
	return c.Struct.NewMethodName()
}

func (p *Binder) NewParam(v *types.Var) *Param {
	return &Param{
		underlying: v,
		binder:     p,
	}
}
