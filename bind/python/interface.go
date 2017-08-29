package python

import (
	"github.com/devigned/veil/cgo"
)

type Interface struct {
	*cgo.Interface
	binder  *Binder
	Methods []*Func
}

func (i Interface) Name() string {
	return i.Named.Obj().Name()
}

func (i Interface) NewMethodName() string {
	return i.NewMethodName()
}
