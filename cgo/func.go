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
	if !f.Exported() {
		return false
	}

	for _, v := range allVars(f.Func) {
		if !ShouldGenerate(v) {
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

func (f Func) PackagePath() string {
	return f.Pkg().Path()
}

func (f Func) CGoName() string {
	splitNames := strings.Split(f.Name(), ".")
	pkgName := PkgPathAliasFromString(f.PackagePath())
	return pkgName + "_" + splitNames[len(splitNames)-1]
}

func (f Func) AliasedGoName() ast.Expr {
	splitNames := strings.Split(f.Name(), ".")
	pkgName := PkgPathAliasFromString(f.PackagePath())
	return &ast.SelectorExpr{
		X:   NewIdent(pkgName),
		Sel: NewIdent(splitNames[len(splitNames)-1]),
	}
}
