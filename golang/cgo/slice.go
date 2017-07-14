package cgo

import (
	"go/ast"
	"go/types"
)

// ArrayWrapper is a wrapper for the
type SliceWrapper struct {
	elem types.Type
}

// NewSliceWrapper wraps types.Slice to provide a consistent comparison
func NewSliceWrapper(elem types.Type) SliceWrapper {
	return SliceWrapper{
		elem: elem,
	}
}

// Underlying returns the underlying type of the Slice (types.Type)
func (t SliceWrapper) Underlying() types.Type { return t }

// Underlying returns the string representation of the type (types.Type)
func (t SliceWrapper) String() string { return types.TypeString(types.NewSlice(t.elem), nil) }

// ToCgoAst returns the go/ast representation of the CGo wrapper of the Slice type
func (s SliceWrapper) ToCgoAst() []ast.Decl {
	return nil
}
