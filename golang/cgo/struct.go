package cgo

import (
	"go/ast"
	"go/types"
)

// NamedStruct is a helpful facade over types.Named which is intended to only contain a struct
type StructWrapper struct {
	*types.Named
}

// Struct returns the underlying struct
func (sw StructWrapper) Struct() *types.Struct {
	return sw.Named.Underlying().(*types.Struct)
}

// Methods returns the list of methods decorated on the struct
func (sw StructWrapper) Methods() []*types.Func {
	var methods []*types.Func
	for i := 0; i < sw.Named.NumMethods(); i++ {
		meth := sw.Named.Method(i)
		methods = append(methods, meth)
	}
	return methods
}

// ToCgoAst returns the go/ast representation of the CGo wrapper of the Array type
func (s StructWrapper) ToCgoAst() []ast.Decl {
	return nil
}

// Underlying returns the underlying type
func (sw StructWrapper) Underlying() types.Type { return sw.Named }

// Underlying returns the string representation of the type (types.Type)
func (sw StructWrapper) String() string { return types.TypeString(sw.Named, nil) }
