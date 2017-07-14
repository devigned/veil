package cgo

import (
	"go/ast"
	"go/token"
)

func IncrementRef(target ast.Expr) *ast.ExprStmt {
	return &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: NewIdent("cgo_incref"),
			Args: []ast.Expr{
				&ast.CallExpr{
					Fun: UnsafePtrSelector(),
					Args: []ast.Expr{
						target,
					},
				},
			},
		},
	}
}

func NewIdent(name string) *ast.Ident {
	return &ast.Ident{
		Name: name,
	}
}

func UnsafePtrSelector() *ast.SelectorExpr {
	return &ast.SelectorExpr{
		X:   NewIdent("unsafe"),
		Sel: NewIdent("Pointer"),
	}
}

func DeclareVar(name *ast.Ident, t ast.Expr) *ast.DeclStmt {
	return &ast.DeclStmt{
		Decl: &ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{
				&ast.ValueSpec{
					Names: []*ast.Ident{name},
					Type:  t,
				},
			},
		},
	}
}

func CastReturn(castType, target ast.Expr) *ast.ReturnStmt {
	return &ast.ReturnStmt{
		Results: []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.ParenExpr{
					X: castType,
				},
				Args: []ast.Expr{
					&ast.CallExpr{
						Fun: UnsafePtrSelector(),
						Args: []ast.Expr{
							target,
						},
					},
				},
			},
		},
	}
}
