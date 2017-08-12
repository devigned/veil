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
func (s Slice) Underlying() types.Type {
	return s
}

// Underlying returns the string representation of the type (types.Type)
func (s Slice) String() string {
	return types.TypeString(types.NewSlice(s.elem), nil)
}

// ToAst returns the go/ast representation of the CGo wrapper of the Slice type
func (s Slice) ToAst() []ast.Decl {
	return []ast.Decl{
		s.NewAst(),
		s.StringAst(),
		s.ItemAst(),
		s.ItemSetAst(),
		s.ItemAppendAst(),
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
	goType := &ast.ArrayType{
		Elt: NewIdent(s.elem.String()),
	}
	return NewAst(functionName, goType)
}

// StringAst produces the []ast.Decl to provide a string representation of the slice
func (s Slice) StringAst() ast.Decl {
	functionName := s.CGoName() + "_str"
	goTypeIdent := NewIdent(s.GoName())
	return StringAst(functionName, goTypeIdent)
}

func (s Slice) ItemAst() ast.Decl {
	functionName := s.CGoName() + "_item"
	selfIdent := NewIdent("self")
	indexIdent := NewIdent("i")
	indexTypeIdent := NewIdent("int")
	goTypeIdent := NewIdent(s.GoName())
	itemsIdent := NewIdent("items")

	castExpression := CastUnsafePtrOfTypeUuid(DeRef(goTypeIdent), selfIdent)
	itemField := VarToField(types.NewVar(0, nil, "", s.elem), s.elem)

	funcDecl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: ExportComments(functionName),
		},
		Name: NewIdent(functionName),
		Type: &ast.FuncType{
			Params: InstanceMethodParams(
				[]*ast.Field{
					{
						Names: []*ast.Ident{indexIdent},
						Type:  indexTypeIdent,
					},
				}...),
			Results: &ast.FieldList{
				List: []*ast.Field{
					itemField,
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
				Return(CastOut(s.elem, &ast.IndexExpr{
					X: &ast.ParenExpr{
						X: &ast.StarExpr{
							X: itemsIdent,
						},
					},
					Index: indexIdent,
				})),
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
	itemsIdent := NewIdent("items")
	itemIdent := NewIdent("item")

	castExpression := CastUnsafePtrOfTypeUuid(DeRef(goTypeIdent), selfIdent)
	itemField := VarToField(types.NewVar(0, nil, itemIdent.Name, s.elem), s.elem)

	funcDecl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: ExportComments(functionName),
		},
		Name: NewIdent(functionName),
		Type: &ast.FuncType{
			Params: InstanceMethodParams(
				[]*ast.Field{
					{
						Names: []*ast.Ident{indexIdent},
						Type:  indexTypeIdent,
					},
					itemField,
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
						CastExpr(s.elem, itemIdent),
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
	itemsIdent := NewIdent("items")
	itemIdent := NewIdent("item")

	castExpression := CastUnsafePtrOfTypeUuid(DeRef(goTypeIdent), selfIdent)
	field := VarToField(types.NewVar(0, nil, itemIdent.Name, s.elem), s.elem)

	funcDecl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: ExportComments(functionName),
		},
		Name: NewIdent(functionName),
		Type: &ast.FuncType{
			Params: InstanceMethodParams(field),
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
								CastExpr(s.elem, itemIdent),
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
