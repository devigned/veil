package cgo

import (
	"go/ast"
	"go/token"
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

// CGoType returns the selector expression for the Struct aliased package and type
func (s Struct) CGoType() ast.Expr {
	return CGoType(s.Named)
}

// CGoType returns the selector expression for the aliased package and type
func CGoType(n *types.Named) ast.Expr {
	pkgPathIdent := NewIdent(PkgPathAliasFromString(n.Obj().Pkg().Path()))
	typeIdent := NewIdent(n.Obj().Name())
	return &ast.SelectorExpr{
		X:   pkgPathIdent,
		Sel: typeIdent,
	}
}

// ToAst returns the go/ast representation of the CGo wrapper of the Array type
func (s Struct) ToAst() []ast.Decl {
	decls := []ast.Decl{s.NewAst(), s.StringAst(), s.DestroyAst()}
	decls = append(decls, s.FieldAccessorsAst()...)
	return decls
}

// NewAst produces the []ast.Decl to construct a slice type and increment it's reference count
func (s Struct) NewAst() ast.Decl {
	functionName := s.CGoName() + "_new"
	return NewAst(functionName, s.CGoType())
}

// StringAst produces the []ast.Decl to provide a string representation of the slice
func (s Struct) StringAst() ast.Decl {
	functionName := s.CGoName() + "_str"
	return StringAst(functionName, s.CGoType())
}

// DestroyAst produces the []ast.Decl to destruct a slice type and decrement it's reference count
func (s Struct) DestroyAst() ast.Decl {
	return DestroyAst(s.CGoName() + "_destroy")
}

func (s Struct) FieldAccessorsAst() []ast.Decl {
	var accessors []ast.Decl
	for i := 0; i < s.Struct().NumFields(); i++ {
		field := s.Struct().Field(i)
		if !field.Exported() {
			continue
		}
		accessors = append(accessors, s.Getter(field), s.Setter(field))
	}

	return accessors
}

func (s Struct) Getter(field *types.Var) ast.Decl {
	functionName := s.CGoFieldName(field) + "_get"
	selfIdent := NewIdent("self")
	localVarIdent := NewIdent("value")
	fieldIdent := NewIdent(field.Name())

	castExpression := CastUnsafePtr(DeRef(s.CGoType()), selfIdent)

	results := &ast.FieldList{}
	body := &ast.BlockStmt{}
	if basic, ok := field.Type().(*types.Basic); ok {
		results.List = []*ast.Field{{Type: NewIdent(basic.Name())}}
		body.List = []ast.Stmt{
			&ast.AssignStmt{
				Lhs: []ast.Expr{localVarIdent},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{
					&ast.SelectorExpr{
						X:   castExpression,
						Sel: fieldIdent,
					},
				},
			},
			Return(localVarIdent),
		}

	} else {
		results.List = []*ast.Field{{Type: unsafePointer}}
		body.List = []ast.Stmt{
			&ast.AssignStmt{
				Lhs: []ast.Expr{localVarIdent},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{
					&ast.SelectorExpr{
						X:   castExpression,
						Sel: fieldIdent,
					},
				},
			},
			Return(UnsafePointerToTarget(Ref(localVarIdent))),
		}
	}

	funcDecl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: ExportComments(functionName),
		},
		Name: NewIdent(functionName),
		Type: &ast.FuncType{
			Params:  InstanceMethodParams(),
			Results: results,
		},
		Body: body,
	}

	return funcDecl
}

func (s Struct) Setter(field *types.Var) ast.Decl {
	functionName := s.CGoFieldName(field) + "_set"
	selfIdent := NewIdent("self")
	localVarIdent := NewIdent("value")
	fieldIdent := NewIdent(field.Name())

	castExpression := CastUnsafePtr(DeRef(s.CGoType()), selfIdent)

	var params *ast.FieldList
	body := &ast.BlockStmt{}
	if basic, ok := field.Type().(*types.Basic); ok {
		basicTypeIdent := NewIdent(basic.Name())
		params = InstanceMethodParams(&ast.Field{Type: basicTypeIdent, Names: []*ast.Ident{localVarIdent}})
		body.List = []ast.Stmt{
			&ast.AssignStmt{
				Lhs: []ast.Expr{
					&ast.SelectorExpr{
						X:   castExpression,
						Sel: fieldIdent,
					},
				},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{localVarIdent},
			},
		}

	} else {
		params = InstanceMethodParams(&ast.Field{Type: unsafePointer, Names: []*ast.Ident{localVarIdent}})
		rhs := CastExpr(field.Type(), localVarIdent)
		body.List = []ast.Stmt{
			&ast.AssignStmt{
				Lhs: []ast.Expr{
					&ast.SelectorExpr{
						X:   castExpression,
						Sel: fieldIdent,
					},
				},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{rhs},
			},
		}
	}

	funcDecl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: ExportComments(functionName),
		},
		Name: NewIdent(functionName),
		Type: &ast.FuncType{
			Params: params,
		},
		Body: body,
	}

	return funcDecl
}

func (s Struct) CGoFieldName(field *types.Var) string {
	return s.CGoName() + "_" + field.Name()
}
