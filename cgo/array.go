package cgo

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

// ArrayWrapper is a wrapper for the
type Array struct {
	elem types.Type
	len  int64
}

// NewArrayWrapper wraps types.Array to provide a consistent comparison
func NewArray(elem types.Type, len int64) *Array {
	return &Array{
		elem: elem,
		len:  len,
	}
}

// Underlying returns the underlying type of the Array (types.Type)
func (a Array) Underlying() types.Type { return a }

// Underlying returns the string representation of the type (types.Type)
func (a Array) String() string { return types.TypeString(types.NewArray(a.elem, a.len), nil) }

// ToAst returns the go/ast representation of the CGo wrapper of the Array type
func (a Array) ToAst() []ast.Decl {
	return nil
}

func ArrayConstructor(goType, cType string) *ast.FuncDecl {
	funcName := fmt.Sprintf("%s_new", cType)
	varIdent := NewIdent("o")
	unsafePtrSelector := &ast.SelectorExpr{
		X:   NewIdent("unsafe"),
		Sel: NewIdent("Pointer"),
	}
	varRefExpr := &ast.UnaryExpr{
		Op: token.AND,
		X:  varIdent,
	}

	funcDecl := ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: []*ast.Comment{
				{Text: fmt.Sprintf("//export %s", funcName)},
			},
		},
		Name: NewIdent(funcName),
		Type: &ast.FuncType{
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: NewIdent(cType),
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.DeclStmt{
					Decl: &ast.GenDecl{
						Tok: token.VAR,
						Specs: []ast.Spec{
							&ast.ValueSpec{
								Names: []*ast.Ident{varIdent},
								Type: &ast.ArrayType{
									Elt: NewIdent(goType),
								},
							},
						},
					},
				},
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: NewIdent(INCREMENT_REF_FUNC_NAME),
						Args: []ast.Expr{
							&ast.CallExpr{
								Fun: unsafePtrSelector,
								Args: []ast.Expr{
									varRefExpr,
								},
							},
						},
					},
				},
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.ParenExpr{
								X: NewIdent(cType),
							},
							Args: []ast.Expr{
								&ast.CallExpr{
									Fun: unsafePtrSelector,
									Args: []ast.Expr{
										varRefExpr,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return &funcDecl
}
