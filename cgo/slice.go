package cgo

import (
	"go/ast"
	"go/token"
	"go/types"
)

// ArrayWrapper is a wrapper for the
type Slice struct {
	elem types.Type
}

// NewSliceWrapper wraps types.Slice to provide a consistent comparison
func NewSlice(elem types.Type) Slice {
	return Slice{
		elem: elem,
	}
}

// Underlying returns the underlying type of the Slice (types.Type)
func (t Slice) Underlying() types.Type {
	return t
}

// Underlying returns the string representation of the type (types.Type)
func (t Slice) String() string {
	return types.TypeString(types.NewSlice(t.elem), nil)
}

// ToCgoAst returns the go/ast representation of the CGo wrapper of the Slice type
func (s Slice) ToCgoAst() []ast.Decl {
	return []ast.Decl{
		s.NewAst(),
		s.StringAst(),
		s.ItemAst(),
		s.ItemSetAst(),
		s.ItemAppendAst(),
		s.DestroyAst(),
	}
}

func (s Slice) GoName() string {
	return "[]" + s.elem.String()
}

func (s Slice) CGoName() string {
	return "slice_of_" + s.elem.String()
}

// NewAst produces the []ast.Decl to construct a slice type and increment it's reference count
func (s Slice) NewAst() ast.Decl {
	functionName := s.CGoName() + "_new"
	localVarIdent := NewIdent("o")
	goTypeIdent := NewIdent(s.GoName())
	target := &ast.UnaryExpr{
		Op: token.AND,
		X:  localVarIdent,
	}

	goType := &ast.ArrayType{
		Elt: NewIdent(s.elem.String()),
	}

	funcDecl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: ExportComments(functionName),
		},
		Name: NewIdent(functionName),
		Type: &ast.FuncType{
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: goTypeIdent},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				DeclareVar(localVarIdent, goType),
				IncrementRefCall(target),
				CastReturn(goTypeIdent, target),
			},
		},
	}

	return funcDecl
}

// StringAst produces the []ast.Decl to provide a string representation of the slice
func (s Slice) StringAst() ast.Decl {
	functionName := s.CGoName() + "_str"
	selfIdent := NewIdent("self")
	goTypeIdent := NewIdent(s.GoName())
	stringIdent := NewIdent("string")

	castExpression := CastUnsafePtr(DeRef(goTypeIdent), selfIdent)
	deRef := DeRef(castExpression)
	sprintf := FormatSprintf("%#v", deRef)

	funcDecl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: ExportComments(functionName),
		},
		Name: NewIdent(functionName),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{selfIdent},
						Type:  goTypeIdent,
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: stringIdent},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				Return(sprintf),
			},
		},
	}

	return funcDecl
}

func (s Slice) ItemAst() ast.Decl {
	functionName := s.CGoName() + "_item"
	selfIdent := NewIdent("self")
	indexIdent := NewIdent("i")
	indexTypeIdent := NewIdent("int")
	goTypeIdent := NewIdent(s.GoName())
	elementTypeIdent := NewIdent(s.elem.String())
	itemsIdent := NewIdent("items")

	castExpression := CastUnsafePtr(DeRef(goTypeIdent), selfIdent)

	funcDecl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: ExportComments(functionName),
		},
		Name: NewIdent(functionName),
		Type: &ast.FuncType{
			Params: InstanceMethodParams(goTypeIdent,
				[]*ast.Field{
					{
						Names: []*ast.Ident{indexIdent},
						Type:  indexTypeIdent,
					},
				}...),
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: elementTypeIdent},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						itemsIdent,
					},
					Rhs: []ast.Expr{
						castExpression,
					},
					Tok: token.DEFINE,
				},
				Return(&ast.IndexExpr{
					X: &ast.ParenExpr{
						X: &ast.StarExpr{
							X: itemsIdent,
						},
					},
					Index: indexIdent,
				}),
			},
		},
	}

	return funcDecl
}

func (s Slice) ItemSetAst() ast.Decl {
	functionName := s.CGoName() + "_item_set"
	selfIdent := NewIdent("self")
	indexIdent := NewIdent("i")
	indexTypeIdent := NewIdent("int")
	goTypeIdent := NewIdent(s.GoName())
	elementTypeIdent := NewIdent(s.elem.String())
	itemsIdent := NewIdent("items")
	itemIdent := NewIdent("item")

	castExpression := CastUnsafePtr(DeRef(goTypeIdent), selfIdent)

	funcDecl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: ExportComments(functionName),
		},
		Name: NewIdent(functionName),
		Type: &ast.FuncType{
			Params: InstanceMethodParams(goTypeIdent,
				[]*ast.Field{
					{
						Names: []*ast.Ident{indexIdent},
						Type:  indexTypeIdent,
					},
					{
						Names: []*ast.Ident{itemIdent},
						Type:  elementTypeIdent,
					},
				}...),
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						itemsIdent,
					},
					Rhs: []ast.Expr{
						castExpression,
					},
					Tok: token.DEFINE,
				},
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						&ast.IndexExpr{
							X: &ast.ParenExpr{
								X: &ast.StarExpr{
									X: itemsIdent,
								},
							},
							Index: indexIdent,
						},
					},
					Rhs: []ast.Expr{
						itemIdent,
					},
					Tok: token.ASSIGN,
				},
			},
		},
	}

	return funcDecl
}

// ItemAppendAst returns a function declaration which appends an item to the slice
func (s Slice) ItemAppendAst() ast.Decl {
	functionName := s.CGoName() + "_item_append"
	selfIdent := NewIdent("self")
	goTypeIdent := NewIdent(s.GoName())
	elementTypeIdent := NewIdent(s.elem.String())
	itemsIdent := NewIdent("items")
	itemIdent := NewIdent("item")

	castExpression := CastUnsafePtr(DeRef(goTypeIdent), selfIdent)

	funcDecl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: ExportComments(functionName),
		},
		Name: NewIdent(functionName),
		Type: &ast.FuncType{
			Params: InstanceMethodParams(goTypeIdent,
				[]*ast.Field{
					{
						Names: []*ast.Ident{itemIdent},
						Type:  elementTypeIdent,
					},
				}...),
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						itemsIdent,
					},
					Rhs: []ast.Expr{
						castExpression,
					},
					Tok: token.DEFINE,
				},
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						DeRef(itemsIdent),
					},
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: NewIdent("append"),
							Args: []ast.Expr{
								DeRef(itemsIdent),
								itemIdent,
							},
						},
					},
					Tok: token.ASSIGN,
				},
			},
		},
	}

	return funcDecl
}

// DestroyAst produces the []ast.Decl to destruct a slice type and decrement it's reference count
func (s Slice) DestroyAst() ast.Decl {
	functionName := s.CGoName() + "_destroy"
	goTypeIdent := NewIdent(s.GoName())
	target := &ast.UnaryExpr{
		Op: token.AND,
		X:  NewIdent("self"),
	}

	funcDecl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: ExportComments(functionName),
		},
		Name: NewIdent(functionName),
		Type: &ast.FuncType{
			Params: InstanceMethodParams(goTypeIdent),
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				DecrementRefCall(target),
			},
		},
	}

	return funcDecl
}
