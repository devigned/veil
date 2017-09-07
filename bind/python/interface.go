package python

import (
	"github.com/devigned/veil/cgo"
	"go/ast"
)

type Interface struct {
	*cgo.Interface
	binder  *Binder
	Methods []*Func
}

func (iface Interface) Name() string {
	return iface.Interface.Name()
}

func (iface Interface) CName() string {
	return iface.Interface.CName()
}

// ToAst returns the go/ast representation of the CGo wrapper of the named type
func (iface Interface) ToAst() []ast.Decl {
	decls := []ast.Decl{}
	return decls
}
