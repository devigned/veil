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
func (f Func) Underlying() types.Type {
	return f.Func.Type()
}

func (f Func) Signature() *types.Signature {
	sig, _ := f.Underlying().(*types.Signature)
	return sig
}

func (f Func) IsExportable() bool {
	for i := 0; i < f.Signature().Params().Len(); i++ {
		param := f.Signature().Params().At(i)
		switch param.Type().(type) {
		case *types.Chan:
			return false
		}
	}

	for i := 0; i < f.Signature().Results().Len(); i++ {
		result := f.Signature().Results().At(i)
		switch result.Type().(type) {
		case *types.Chan:
			return false
		}
	}

	return true
}

// Underlying returns the string representation of the type (types.Type)
func (f Func) String() string {
	return f.Func.FullName() + ": " + types.TypeString(f.Underlying(), nil)
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

// ToAst returns the go/ast representation of the CGo wrapper of the Func type
func (f Func) ToAst() []ast.Decl {
	return []ast.Decl{
		FuncAst(&f),
	}
}

func (f Func) ExportName() string {
	return f.CGoName()
}

func (f Func) CGoName() string {
	splitNames := strings.Split(f.Name(), ".")
	pkgName := PkgPathAliasFromString(f.Pkg().Path())
	return pkgName + "_" + splitNames[len(splitNames)-1]
}

func (f Func) AliasedGoName() ast.Expr {
	splitNames := strings.Split(f.Name(), ".")
	pkgName := PkgPathAliasFromString(f.Pkg().Path())
	return &ast.SelectorExpr{
		X:   NewIdent(pkgName),
		Sel: NewIdent(splitNames[len(splitNames)-1]),
	}
}
