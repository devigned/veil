package cgo

import (
	"go/ast"
	"go/types"
	"strings"
)

// FuncWrapper is a wrapper for a Function
type Func struct {
	*types.Func
	BoundRecv *Named
}

func NewFunc(fun *types.Func) *Func {
	return &Func{Func: fun, BoundRecv: nil}
}

func NewBoundFunc(fun *types.Func, boundRecv *Named) *Func {
	return &Func{Func: fun, BoundRecv: boundRecv}
}

// Underlying returns the underlying type
func (f Func) Underlying() types.Type {
	return f.Type()
}

func (f Func) Signature() *types.Signature {
	sig, _ := f.Underlying().(*types.Signature)
	return sig
}

func (f Func) IsExportable() bool {
	if !f.Exported() {
		return false
	}

	for _, v := range allVars(&f) {
		if !ShouldGenerate(v) {
			return false
		}
	}
	return true
}

// Underlying returns the string representation of the type (types.Type)
func (f Func) String() string {
	return f.FullName() + ": " + types.TypeString(f.Underlying(), nil)
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
	return f.CName()
}

func (f Func) PackagePath() string {
	return f.Pkg().Path()
}

func (f Func) CName() string {
	splitNames := strings.Split(f.Name(), ".")
	pkgName := PkgPathAliasFromString(f.PackagePath())
	if f.BoundRecv == nil {
		return strings.Join([]string{pkgName, splitNames[len(splitNames)-1]}, "_")
	} else {
		boundName := f.BoundRecv.CShortName()
		return strings.Join([]string{pkgName, boundName, splitNames[len(splitNames)-1]}, "_")
	}

}

func (f Func) AliasedGoName() ast.Expr {
	splitNames := strings.Split(f.Name(), ".")
	pkgName := PkgPathAliasFromString(f.PackagePath())
	return &ast.SelectorExpr{
		X:   NewIdent(pkgName),
		Sel: NewIdent(splitNames[len(splitNames)-1]),
	}
}
