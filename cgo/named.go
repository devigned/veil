package cgo

import (
	"go/ast"
	"go/types"
	"strings"
)

type Named struct {
	*types.Named
}

func NewNamed(named *types.Named) *Named {
	return &Named{named}
}

// ToAst returns the go/ast representation of the CGo wrapper of the named type
func (n Named) ToAst() []ast.Decl {
	decls := []ast.Decl{n.NewAst(), n.StringAst()}
	decls = append(decls, n.MethodAsts()...)
	return decls
}

// NewAst produces the []ast.Decl to construct a named type and increment it's reference count
func (n Named) NewAst() ast.Decl {
	functionName := n.NewMethodName()
	return NewAst(functionName, n.CTypeName())
}

// StringAst produces the []ast.Decl to provide a string representation of the named type
func (n Named) StringAst() ast.Decl {
	functionName := n.ToStringMethodName()
	return StringAst(functionName, n.CTypeName())
}

func (n Named) MethodAsts() []ast.Decl {
	results := []ast.Decl{}
	for _, fun := range n.ExportedMethods() {
		results = append(results, fun.ToAst()...)
	}
	return results
}

// CTypeName returns the selector expression for the Named aliased package and type
func (n Named) CTypeName() ast.Expr {
	return cTypeName(n.Named)
}

// CTypeName returns the selector expression for the aliased package and type
func cTypeName(n *types.Named) ast.Expr {
	pkgPathIdent := NewIdent(PkgPathAliasFromString(n.Obj().Pkg().Path()))
	typeIdent := NewIdent(n.Obj().Name())
	return &ast.SelectorExpr{
		X:   pkgPathIdent,
		Sel: typeIdent,
	}
}

func (n Named) NewMethodName() string {
	return n.CName() + "_new"
}

func (n Named) ToStringMethodName() string {
	return n.CName() + "_str"
}

// Methods returns the list of methods decorated on the named type
func (n Named) ExportedMethods() []*Func {
	var methods []*Func
	numMethods := n.NumMethods()
	for i := 0; i < numMethods; i++ {
		meth := n.Method(i)
		fun := NewBoundFunc(meth, &n)
		if fun.IsExportable() {
			methods = append(methods, fun)
		}
	}
	return methods
}

func (n Named) CShortName() string {
	return n.Obj().Name()
}

// CName returns the fully resolved name to the named type
func (n Named) CName() string {
	return strings.Join([]string{PkgPathAliasFromString(n.PackagePath()), n.Obj().Name()}, "_")
}

func (n Named) PackagePath() string {
	return n.Obj().Pkg().Path()
}

func (n Named) ExportName() string {
	return n.CName()
}

func (n Named) IsExportable() bool {
	return true
}
