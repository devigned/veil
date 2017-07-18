package cgo

import (
	"go/ast"
	"go/types"
)

// ArrayWrapper is a wrapper for the
type Array struct {
	elem types.Type
	len  int64
}

// NewArrayWrapper wraps types.Array to provide a consistent comparison
func NewArray(elem types.Type, len int64) *Array {
	return &Array{
		elem: elem,
		len:  len,
	}
}

// Underlying returns the underlying type of the Array (types.Type)
func (t Array) Underlying() types.Type { return t }

// Underlying returns the string representation of the type (types.Type)
func (t Array) String() string { return types.TypeString(types.NewArray(t.elem, t.len), nil) }

// ToCgoAst returns the go/ast representation of the CGo wrapper of the Array type
func (s Array) ToCgoAst() []ast.Decl {
	return nil
}
