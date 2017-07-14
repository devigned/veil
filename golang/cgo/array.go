package cgo

import (
	"go/ast"
	"go/types"
)

// ArrayWrapper is a wrapper for the
type ArrayWrapper struct {
	elem types.Type
	len  int64
}

// NewArrayWrapper wraps types.Array to provide a consistent comparison
func NewArrayWrapper(elem types.Type, len int64) *ArrayWrapper {
	return &ArrayWrapper{
		elem: elem,
		len:  len,
	}
}

// Underlying returns the underlying type of the Array (types.Type)
func (t ArrayWrapper) Underlying() types.Type { return t }

// Underlying returns the string representation of the type (types.Type)
func (t ArrayWrapper) String() string { return types.TypeString(types.NewArray(t.elem, t.len), nil) }

// ToCgoAst returns the go/ast representation of the CGo wrapper of the Array type
func (s ArrayWrapper) ToCgoAst() []ast.Decl {
	return nil
}
