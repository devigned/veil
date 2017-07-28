package cgo

import (
	"go/ast"
	"go/types"
)

// Struct is a helpful facade over types.Named which is intended to only contain a struct
type Struct struct {
	*types.Named
}

// Struct returns the underlying struct
func (s Struct) Struct() *types.Struct {
	return s.Named.Underlying().(*types.Struct)
}

// Methods returns the list of methods decorated on the struct
func (s Struct) Methods() []*types.Func {
	var methods []*types.Func
	for i := 0; i < s.Named.NumMethods(); i++ {
		meth := s.Named.Method(i)
		methods = append(methods, meth)
	}
	return methods
}

// Underlying returns the underlying type
func (s Struct) Underlying() types.Type { return s.Named }

// Underlying returns the string representation of the type (types.Type)
func (s Struct) String() string { return types.TypeString(s.Named, nil) }

// ToAst returns the go/ast representation of the CGo wrapper of the Array type
func (s Struct) ToAst() []ast.Decl {
	return nil
}
