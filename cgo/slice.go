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
		s.ItemDeleteAst(),
		s.LenAst(),
		s.ItemInsertAst(),
	}
}

func (s Slice) ExportName() string {
	return s.CGoName()
}

func (s Slice) MethodName() string {
	pkgAlias, name := s.ElementPackageAliasAndPath(nil)
	return pkgAlias + "_" + name
}

func (s Slice) ElementName() string {
	pkgAlias, name := s.ElementPackageAliasAndPath(nil)
	if pkgAlias == "" {
		return name
	} else {
		return pkgAlias + "." + name
	}
}

func (s Slice) ElementPackageAliasAndPath(typ types.Type) (string, string) {
	if typ == nil {
		typ = s.elem
	}

	objToString := func(typeName *types.TypeName) (string, string) {
		return PkgPathAliasFromString(typeName.Pkg().Path()), typeName.Name()
	}
	switch t := typ.(type) {
	case *types.Basic:
		return "", t.Name()
	case *types.Named:
		obj := t.Obj()
		return objToString(obj)
	case *types.Pointer:
		return s.ElementPackageAliasAndPath(t.Elem())
	default:
		return "", t.String()
	}
}

func (s Slice) Elem() types.Type {
	return s.elem
}

func (s Slice) GoName() string {
	if _, ok := s.elem.(*types.Pointer); ok {
		return "[]*" + s.ElementName()
	} else {
		return "[]" + s.ElementName()
	}
}

func (s Slice) CGoName() string {
	return "slice_of_" + s.MethodName()
}

// NewAst produces the []ast.Decl to construct a slice type and increment it's reference count
func (s Slice) NewAst() ast.Decl {
	functionName := s.CGoName() + "_new"
	goType := &ast.ArrayType{
		Elt: NewIdent(s.ElementName()),
	}
	return NewAst(functionName, goType)
}

// StringAst produces the []ast.Decl to provide a string representation of the slice
func (s Slice) StringAst() ast.Decl {
	functionName := s.CGoName() + "_str"
	goTypeIdent := NewIdent(s.GoName())
	return StringAst(functionName, goTypeIdent)
}

func (s Slice) LenAst() ast.Decl {
	functionName := s.CGoName() + "_len"
	selfIdent := NewIdent("self")
	goTypeIdent := NewIdent(s.GoName())
	itemsIdent := NewIdent("items")

	castExpression := CastUnsafePtrOfTypeUuid(DeRef(goTypeIdent), selfIdent)

	funcDecl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: ExportComments(functionName),
		},
		Name: NewIdent(functionName),
		Type: &ast.FuncType{
			Params: InstanceMethodParams(),
			Results: &ast.FieldList{
				List: []*ast.Field{{Type: NewIdent("int")}},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{itemsIdent},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{castExpression},
				},
				Return(&ast.CallExpr{
					Fun:  NewIdent("len"),
					Args: []ast.Expr{DeRef(itemsIdent)},
				}),
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
	itemsIdent := NewIdent("items")

	castExpression := CastUnsafePtrOfTypeUuid(DeRef(goTypeIdent), selfIdent)
	itemField := &ast.Field{
		Type:  TypeToArgumentTypeExpr(s.Elem()),
		Names: []*ast.Ident{NewIdent("item")},
	}

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
	itemField := &ast.Field{
		Type:  TypeToArgumentTypeExpr(s.Elem()),
		Names: []*ast.Ident{NewIdent("item")},
	}

	rhsCast := CastExpr(s.elem, itemIdent)

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
					Rhs: []ast.Expr{rhsCast},
					Tok: token.ASSIGN,
				},
			},
		},
	}

	return funcDecl
}

// ItemDeleteAst returns a function declaration which deletes an item from the slice
func (s Slice) ItemDeleteAst() ast.Decl {
	functionName := s.CGoName() + "_item_del"
	selfIdent := NewIdent("self")
	indexIdent := NewIdent("i")
	indexTypeIdent := NewIdent("int")
	goTypeIdent := NewIdent(s.GoName())
	itemsIdent := NewIdent("items")

	castExpression := CastUnsafePtrOfTypeUuid(DeRef(goTypeIdent), selfIdent)

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
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				// items := (*[]type)(cgo_get_ref(cgo_get_uuid_from_ptr(self)))
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						itemsIdent,
					},
					Rhs: []ast.Expr{
						castExpression,
					},
					Tok: token.DEFINE,
				},
				// *items = append(*items[:i], *items[i+1:]...)
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						DeRef(itemsIdent),
					},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: NewIdent("append"),
							Args: []ast.Expr{
								&ast.SliceExpr{
									X:    DeRef(itemsIdent),
									High: indexIdent,
								},
								&ast.SliceExpr{
									X: DeRef(itemsIdent),
									Low: &ast.BinaryExpr{
										X:  indexIdent,
										Op: token.ADD,
										Y: &ast.BasicLit{
											Kind:  token.INT,
											Value: "1",
										},
									},
								},
							},
							Ellipsis: token.Pos(-1),
						},
					},
				},
			},
		},
	}

	return funcDecl
}

// ItemInsertAst returns a function declaration which inserts an item into the slice
func (s Slice) ItemInsertAst() ast.Decl {
	functionName := s.CGoName() + "_item_insert"
	selfIdent := NewIdent("self")
	indexIdent := NewIdent("i")
	indexTypeIdent := NewIdent("int")
	goTypeIdent := NewIdent(s.GoName())
	itemsIdent := NewIdent("items")
	itemIdent := NewIdent("item")

	castExpression := CastUnsafePtrOfTypeUuid(DeRef(goTypeIdent), selfIdent)
	itemField := &ast.Field{
		Type:  TypeToArgumentTypeExpr(s.Elem()),
		Names: []*ast.Ident{NewIdent("item")},
	}

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
				// items := (*[]type)(cgo_get_ref(cgo_get_uuid_from_ptr(self)))
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						itemsIdent,
					},
					Rhs: []ast.Expr{
						castExpression,
					},
					Tok: token.DEFINE,
				},
				// *items = append(a[:i], append([]T{x}, a[i:]...)...)
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						DeRef(itemsIdent),
					},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: NewIdent("append"),
							Args: []ast.Expr{
								&ast.SliceExpr{
									X:    DeRef(itemsIdent),
									High: indexIdent,
								},
								&ast.CallExpr{
									Fun: NewIdent("append"),
									Args: []ast.Expr{
										&ast.CompositeLit{
											Type: goTypeIdent,
											Elts: []ast.Expr{
												CastExpr(s.elem, itemIdent),
											},
										},
										&ast.SliceExpr{
											X:   DeRef(itemsIdent),
											Low: indexIdent,
										},
									},
									Ellipsis: token.Pos(-1),
								},
							},
							Ellipsis: token.Pos(-1),
						},
					},
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
	itemField := &ast.Field{
		Type:  TypeToArgumentTypeExpr(s.Elem()),
		Names: []*ast.Ident{NewIdent("item")},
	}

	funcDecl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: ExportComments(functionName),
		},
		Name: NewIdent(functionName),
		Type: &ast.FuncType{
			Params: InstanceMethodParams(itemField),
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
