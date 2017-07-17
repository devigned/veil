package cgo

import (
	"go/ast"
	"go/token"
)

// IncrementRef takes a target expression to increment it's cgo pointer ref and returns the expression
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

// IncrementRef takes a target expression to decrement it's cgo pointer ref and returns the expression
func DecrementRef(target ast.Expr) *ast.ExprStmt {
	return &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: NewIdent("cgo_decref"),
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

// NewIdent takes a name as string and returns an *ast.Ident in that name
func NewIdent(name string) *ast.Ident {
	return &ast.Ident{
		Name: name,
	}
}

// UnsafePtrSelector is a commonly used selector expression (unsafe.Pointer)
func UnsafePtrSelector() *ast.SelectorExpr {
	return &ast.SelectorExpr{
		X:   NewIdent("unsafe"),
		Sel: NewIdent("Pointer"),
	}
}

// DeclareVar declares a local variable
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

// CastReturn returns a return expression with a cast version of the target expression
func CastReturn(castType, target ast.Expr) *ast.ReturnStmt {
	return Return(CastUnsafePtr(castType, target))
}

// CastUnsafePtr take a cast type and target expression and returns a cast expression
func CastUnsafePtr(castType, target ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
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
	}
}

// DeRef takes an expression and prefaces the expression with a *
func DeRef(expr ast.Expr) *ast.StarExpr {
	return &ast.StarExpr{X: expr}
}

// Return takes a slice of expressions and returns a return statement containing the expressions
func ReturnGroup(expressions []ast.Expr) *ast.ReturnStmt {
	return &ast.ReturnStmt{
		Results: expressions,
	}
}

// Return takes an expression and returns a return statement containing the expression
func Return(expression ast.Expr) *ast.ReturnStmt {
	return &ast.ReturnStmt{
		Results: []ast.Expr{
			expression,
		},
	}
}

func ReturnCastDeref(castType, target ast.Expr) *ast.ReturnStmt {
	castExpression := CastUnsafePtr(castType, target)
	deRef := DeRef(castExpression)
	return Return(deRef)
}

// FormatSprintf takes a format and a target expression and returns a fmt.Sprintf expression
func FormatSprintf(format string, target ast.Expr) *ast.CallExpr {
	fmtSprintf := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   NewIdent("fmt"),
			Sel: NewIdent("Sprintf"),
		},
		Args: []ast.Expr{
			&ast.BasicLit{
				Kind:  token.STRING,
				Value: "\"" + format + "\"",
			},
			target,
		},
	}

	return fmtSprintf
}

// ExportComments takes a name to export as string and returns a comment group
func ExportComments(exportName string) []*ast.Comment {
	return []*ast.Comment{
		{
			Text:  "//export " + exportName,
			Slash: token.Pos(1),
		},
	}
}
