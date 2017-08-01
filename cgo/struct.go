package cgo

import (
	"go/ast"
	"go/types"
)

// Struct is a helpful facade over types.Named which is intended to only contain a struct
type Struct struct {
	*types.Named
}

// Struct returns the underlying struct
func (s Struct) Struct() *types.Struct {
	return s.Named.Underlying().(*types.Struct)
}

// Methods returns the list of methods decorated on the struct
func (s Struct) Methods() []*types.Func {
	var methods []*types.Func
	for i := 0; i < s.Named.NumMethods(); i++ {
		meth := s.Named.Method(i)
		methods = append(methods, meth)
	}
	return methods
}

// Underlying returns the underlying type
func (s Struct) Underlying() types.Type { return s.Named }

// Underlying returns the string representation of the type (types.Type)
func (s Struct) String() string { return types.TypeString(s.Named, nil) }

// CGoName returns the fully resolved name to the struct
func (s Struct) CGoName() string {
	return PkgPathAliasFromString(s.Named.Obj().Pkg().Path()) + "_" + s.Named.Obj().Name()
}

func (s Struct) GoType() ast.Expr {
	pkgPathIdent := NewIdent(PkgPathAliasFromString(s.Named.Obj().Pkg().Path()))
	typeIdent := NewIdent(s.Named.Obj().Name())
	return &ast.SelectorExpr{
		X:   pkgPathIdent,
		Sel: typeIdent,
	}
}

// ToAst returns the go/ast representation of the CGo wrapper of the Array type
func (s Struct) ToAst() []ast.Decl {
	return []ast.Decl{s.NewAst(), s.StringAst(), s.DestroyAst()}
}

// NewAst produces the []ast.Decl to construct a slice type and increment it's reference count
func (s Struct) NewAst() ast.Decl {
	functionName := s.CGoName() + "_new"
	return NewAst(functionName, s.GoType())
}

// StringAst produces the []ast.Decl to provide a string representation of the slice
func (s Struct) StringAst() ast.Decl {
	functionName := s.CGoName() + "_str"
	return StringAst(functionName, s.GoType())
}

// DestroyAst produces the []ast.Decl to destruct a slice type and decrement it's reference count
func (s Struct) DestroyAst() ast.Decl {
	return DestroyAst(s.CGoName() + "_destroy")
}
