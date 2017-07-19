package cgo

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
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

// Return takes an expression and returns a return statement containing the expression
func Return(expression ast.Expr) *ast.ReturnStmt {
	return &ast.ReturnStmt{
		Results: []ast.Expr{
			expression,
		},
	}
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

// InstanceMethodParams returns a constructed field list for an instance method
func InstanceMethodParams(selfTypeIdent *ast.Ident, fields ...*ast.Field) *ast.FieldList {
	tmpFields := []*ast.Field{
		{
			Names: []*ast.Ident{NewIdent("self")},
			Type:  selfTypeIdent,
		},
	}
	tmpFields = append(tmpFields, fields...)
	params := &ast.FieldList{
		List: tmpFields,
	}
	return params
}

func FuncAst(f *Func) *ast.FuncDecl {
	fun := f.Func
	functionName := f.CGoName()
	sig := fun.Type().(*types.Signature)
	return &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: ExportComments(functionName),
		},
		Type: &ast.FuncType{
			Params: ParamsAst(sig.Params()),
		},
		Name: NewIdent(functionName),
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun:  NewIdent(f.AliasedGoName()),
						Args: ParamIdents(sig.Params()),
					},
				},
			},
		},
	}
}

func ParamIdents(funcParams *types.Tuple) []ast.Expr {
	if funcParams == nil || funcParams.Len() <= 0 {
		return []ast.Expr{}
	}

	args := make([]ast.Expr, funcParams.Len())
	for i := 0; i < funcParams.Len(); i++ {
		args[i] = NewIdent(funcParams.At(i).Name())
	}
	return args
}

func ParamsAst(funcParams *types.Tuple) *ast.FieldList {
	if funcParams == nil || funcParams.Len() <= 0 {
		return &ast.FieldList{}
	}

	fields := make([]*ast.Field, funcParams.Len())
	for i := 0; i < funcParams.Len(); i++ {
		p := funcParams.At(i)
		switch named := p.Type().(type) {
		case *types.Named:
			pkgAlias := PkgPathAliasFromString(p.Pkg().Path())
			fields[i] = &ast.Field{
				Type:  NewIdent(pkgAlias + "." + named.Obj().Name()),
				Names: []*ast.Ident{NewIdent(p.Name())},
			}
		default:
			fields[i] = &ast.Field{
				Type:  NewIdent(p.Type().String()),
				Names: []*ast.Ident{NewIdent(p.Name())},
			}

		}
	}
	return &ast.FieldList{List: fields}
}

func PkgPathAliasFromString(path string) string {
	splits := strings.FieldsFunc(path, splitPkgPath)
	return strings.Join(splits, "_")
}

func splitPkgPath(r rune) bool {
	return r == '.' || r == '/'
}
