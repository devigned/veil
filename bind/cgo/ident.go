package cgo

import (
	"go/ast"
)

func NewIdent(name string) *ast.Ident {
	return &ast.Ident{
		Name: name,
	}
}
