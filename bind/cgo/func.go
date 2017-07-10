package cgo

import (
	"go/ast"
	"go/types"
)

func ToDecls(a []interface{}) []ast.Decl {
	var decls []ast.Decl
	//for _, item := range a {
	//	decls = append(decls, item.(ast.Decl))
	//}
	return decls
}

// Func transforms a function into a cgo declaration
func Func(fun *types.Func) ast.Decl {
	//typ := fun.Type().(*types.Signature)
	return nil
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
