package cgo

import (
	"go/ast"
	"go/types"
	"strings"
)

// FuncWrapper is a wrapper for a Function
type Func struct {
	*types.Func
}

// Underlying returns the underlying type
func (t Func) Underlying() types.Type {
	return t.Func.Type()
}

// Underlying returns the string representation of the type (types.Type)
func (t Func) String() string {
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

// ToCgoAst returns the go/ast representation of the CGo wrapper of the Func type
func (f Func) ToCgoAst() []ast.Decl {
	return []ast.Decl{
		FuncAst(&f),
	}
}

func (f Func) CGoName() string {
	splitNames := strings.Split(f.Name(), ".")
	pkgName := PkgPathAliasFromString(f.Pkg().Path())
	return pkgName + "_" + splitNames[len(splitNames)-1]
}

func (f Func) AliasedGoName() string {
	splitNames := strings.Split(f.Name(), ".")
	pkgName := PkgPathAliasFromString(f.Pkg().Path())
	return pkgName + "." + splitNames[len(splitNames)-1]
}
