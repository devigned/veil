package cgo

import (
	"go/ast"
	"go/types"
)

// FuncWrapper is a wrapper for a Function
type FuncWrapper struct {
	*types.Func
}

// ToCgoAst returns the go/ast representation of the CGo wrapper of the Func type
func (s FuncWrapper) ToCgoAst() []ast.Decl {
	return nil
}

// Underlying returns the underlying type
func (t FuncWrapper) Underlying() types.Type { return t.Func.Type() }

// Underlying returns the string representation of the type (types.Type)
func (t FuncWrapper) String() string {
	return t.Func.FullName() + ": " + types.TypeString(t.Underlying(), nil)
}

func MainFunc() *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: &ast.Ident{
			Name: "main",
		},
		Type: &ast.FuncType{},
		Body: &ast.BlockStmt{},
	}
}
