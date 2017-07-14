package cgo

import (
	"fmt"
	"github.com/marstr/collection"
	"go/ast"
	"go/token"
	"strings"
)

const (
	incrementRefFuncName = "cgo_incref"
)

func WrapType(typeName string, selectionExpr string, comments ...string) ast.Decl {
	objs := collection.AsEnumerable(comments).Enumerate(nil).
		Select(func(item interface{}) interface{} {
			return &ast.Comment{
				Text:  item.(string),
				Slash: token.Pos(1),
			}
		})

	var cs []*ast.Comment
	for i := range objs {
		cs = append(cs, i.(*ast.Comment))
	}

	selections := strings.Split(selectionExpr, ".")

	var decl ast.Decl
	decl = &ast.GenDecl{
		Doc: &ast.CommentGroup{
			List: cs,
		},
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: NewIdent(typeName),
				Type: &ast.SelectorExpr{
					X:   NewIdent(selections[0]),
					Sel: NewIdent(selections[1]),
				},
			},
		},
	}

	return decl
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
						Fun: NewIdent(incrementRefFuncName),
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
