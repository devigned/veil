package cgo

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

const (
	INCREMENT_REF_FUNC_NAME  = "__cgo_incref"
	COBJECT_STRUCT_TYPE_NAME = "__cobject"
)

// CObjectStruct produces an AST struct which will represent a C exposed Object
func CObjectStruct() ast.Decl {
	return &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: NewIdent(COBJECT_STRUCT_TYPE_NAME),
				Type: &ast.StructType{
					Fields: &ast.FieldList{
						List: []*ast.Field{
							{
								Names: []*ast.Ident{NewIdent("ptr")},
								Type: &ast.SelectorExpr{
									X:   NewIdent("unsafe"),
									Sel: NewIdent("Pointer"),
								},
							},
							{
								Names: []*ast.Ident{NewIdent("cnt")},
								Type:  NewIdent("int32"),
							},
						},
					},
				},
			},
		},
	}
}

// RefsStruct produces an AST struct which will keep track of references to pointers used by the host CFFI
func RefsStruct() ast.Decl {
	return &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: NewIdent("refs"),
				Type: &ast.StructType{
					Fields: &ast.FieldList{
						List: []*ast.Field{
							{
								Type: &ast.SelectorExpr{
									X:   NewIdent("sync"),
									Sel: NewIdent("Mutex"),
								},
							},
							{
								Names: []*ast.Ident{NewIdent("next")},
								Type:  NewIdent("int32"),
							},
							{
								Names: []*ast.Ident{NewIdent("refs")},
								Type: &ast.MapType{
									Key: &ast.SelectorExpr{
										X:   NewIdent("unsafe"),
										Sel: NewIdent("Pointer"),
									},
									Value: NewIdent("int32"),
								},
							},
							{
								Names: []*ast.Ident{NewIdent("ptrs")},
								Type: &ast.MapType{
									Key:   NewIdent("int32"),
									Value: NewIdent("cobject"),
								},
							},
						},
					},
				},
			},
		},
	}
}

func IncrementRef() ast.Decl {
	ptr := NewIdent("ptr")
	refs := NewIdent("refs")
	num := NewIdent("num")
	ok := NewIdent("ok")
	s := NewIdent("s")
	ptrs := NewIdent("ptrs")
	next := NewIdent("next")

	return &ast.FuncDecl{
		Doc:  &ast.CommentGroup{List: ExportComments(INCREMENT_REF_FUNC_NAME)},
		Name: NewIdent(INCREMENT_REF_FUNC_NAME),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ptr},
						Type: &ast.SelectorExpr{
							X:   NewIdent("unsafe"),
							Sel: NewIdent("Pointer"),
						},
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				// refs.Lock()
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   refs,
							Sel: NewIdent("Lock"),
						},
					},
				},
				// num, ok := refs.refs[ptr]
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						num,
						ok,
					},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.IndexExpr{
							X: &ast.SelectorExpr{
								X:   refs,
								Sel: refs,
							},
							Index: ptr,
						},
					},
				},
				// if ok {
				&ast.IfStmt{
					Cond: ok,
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							// s := refs.ptrs[num]
							&ast.AssignStmt{
								Lhs: []ast.Expr{s},
								Tok: token.DEFINE,
								Rhs: []ast.Expr{
									&ast.IndexExpr{
										X: &ast.SelectorExpr{
											X:   refs,
											Sel: ptrs,
										},
										Index: num,
									},
								},
							},
							// refs.ptrs[num] = cobjects{s.ptr, s.cnt + 1}
							&ast.AssignStmt{
								Lhs: []ast.Expr{
									&ast.IndexExpr{
										X: &ast.SelectorExpr{
											X:   refs,
											Sel: ptrs,
										},
										Index: num,
									},
								},
								Tok: token.EQL,
								Rhs: []ast.Expr{
									&ast.CompositeLit{
										Type: NewIdent(COBJECT_STRUCT_TYPE_NAME),
										Elts: []ast.Expr{
											&ast.SelectorExpr{
												X:   s,
												Sel: ptr,
											},
											&ast.BinaryExpr{
												X: &ast.SelectorExpr{
													X:   s,
													Sel: NewIdent("cnt"),
												},
												Op: token.ADD,
												Y:  &ast.BasicLit{Value: "1", Kind: token.INT},
											},
										},
									},
								},
							},
						},
					},
					// } else {
					Else: &ast.BlockStmt{
						List: []ast.Stmt{
							// num = refs.next
							&ast.AssignStmt{
								Lhs: []ast.Expr{
									num,
								},
								Tok: token.EQL,
								Rhs: []ast.Expr{
									&ast.SelectorExpr{
										X:   refs,
										Sel: next,
									},
								},
							},
							// refs.next--
							&ast.IncDecStmt{
								X: &ast.SelectorExpr{
									X:   refs,
									Sel: next,
								},
								Tok: token.DEC,
							},
							// if refs.next > 0 {
							&ast.IfStmt{
								Cond: &ast.BinaryExpr{
									X: &ast.SelectorExpr{
										X:   refs,
										Sel: next,
									},
									Op: token.LSS,
									Y:  &ast.BasicLit{Value: "0", Kind: token.INT},
								},
								Body: &ast.BlockStmt{
									List: []ast.Stmt{
										// panic("refs.next underflow")
										&ast.ExprStmt{
											X: &ast.CallExpr{
												Fun: NewIdent("panic"),
												Args: []ast.Expr{
													&ast.BasicLit{Value: "refs.next underflow", Kind: token.STRING},
												},
											},
										},
									},
								},
							},
							// }
							// refs.refs[ptr] = num
							&ast.AssignStmt{
								Lhs: []ast.Expr{
									&ast.IndexExpr{
										X: &ast.SelectorExpr{
											X:   refs,
											Sel: refs,
										},
										Index: ptr,
									},
								},
								Tok: token.ASSIGN,
								Rhs: []ast.Expr{
									num,
								},
							},
							// refs.ptrs[num] = cobject{ptr, 1}
							&ast.AssignStmt{
								Lhs: []ast.Expr{
									&ast.IndexExpr{
										X: &ast.SelectorExpr{
											X:   refs,
											Sel: ptrs,
										},
										Index: num,
									},
								},
								Tok: token.ASSIGN,
								Rhs: []ast.Expr{
									&ast.CompositeLit{
										Type: NewIdent(COBJECT_STRUCT_TYPE_NAME),
										Elts: []ast.Expr{
											ptr,
											&ast.BasicLit{Value: "1", Kind: token.INT},
										},
									},
								},
							},
						},
					},
				},
				// refs.Unlock()
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   refs,
							Sel: NewIdent("Unlock"),
						},
					},
				},
			},
		},
	}
}

// IncrementRefCall takes a target expression to increment it's cgo pointer ref and returns the expression
func IncrementRefCall(target ast.Expr) *ast.ExprStmt {
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

// DecrementRefCall takes a target expression to decrement it's cgo pointer ref and returns the expression
func DecrementRefCall(target ast.Expr) *ast.ExprStmt {
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

// FuncAst returns an FuncDecl which wraps the func
func FuncAst(f *Func) *ast.FuncDecl {
	fun := f.Func
	functionName := f.CGoName()
	sig := fun.Type().(*types.Signature)
	functionCall := &ast.CallExpr{
		Fun:  NewIdent(f.AliasedGoName()),
		Args: ParamIdents(sig.Params()),
	}

	funcDecl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: ExportComments(functionName),
		},
		Name: NewIdent(functionName),
		Body: &ast.BlockStmt{List: []ast.Stmt{}},
	}

	if sig.Results().Len() > 0 {
		// signature will return
		funcDecl.Body.List = append(funcDecl.Body.List, Return(functionCall))

		funcDecl.Type = &ast.FuncType{
			Params:  Fields(sig.Params()),
			Results: Fields(sig.Results()),
		}
	} else {
		funcDecl.Body.List = append(funcDecl.Body.List, &ast.ExprStmt{
			X: functionCall,
		})

		funcDecl.Type = &ast.FuncType{
			Params: Fields(sig.Params()),
		}
	}

	return funcDecl
}

// ParamIdents transforms parameter tuples into a slice of AST expressions
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

// Fields transforms parameters into a list of AST fields
func Fields(funcParams *types.Tuple) *ast.FieldList {
	if funcParams == nil || funcParams.Len() <= 0 {
		return &ast.FieldList{}
	}

	fields := make([]*ast.Field, funcParams.Len())
	for i := 0; i < funcParams.Len(); i++ {
		p := funcParams.At(i)
		switch t := p.Type().(type) {
		case *types.Pointer:
			fields[i] = VarToField(p, t.Elem())
		default:
			fields[i] = VarToField(p, t)
		}
	}
	return &ast.FieldList{List: fields}
}

// VarToField transforms a Var into an AST field
func VarToField(p *types.Var, t types.Type) *ast.Field {
	switch typ := t.(type) {
	case *types.Named:
		return NamedToField(p, typ)
	default:
		name := p.Name()
		typeName := p.Type().String()
		return &ast.Field{
			Type:  NewIdent(typeName),
			Names: []*ast.Ident{NewIdent(name)},
		}

	}
}

// NameToField transforms a Var that's a Named type into an AST Field
func NamedToField(p *types.Var, named *types.Named) *ast.Field {
	pkg := p.Pkg()
	if pkg == nil {
		pkg = named.Obj().Pkg()
	}

	if pkg != nil {
		path := pkg.Path()
		pkgAlias := PkgPathAliasFromString(path)
		typeName := pkgAlias + "." + named.Obj().Name()
		nameIdent := NewIdent(p.Name())
		return &ast.Field{
			Type:  NewIdent(typeName),
			Names: []*ast.Ident{nameIdent},
		}
	} else {
		fmt.Println(p, named)
		typeIdnet := NewIdent(p.Type().String())
		nameIdent := NewIdent(p.Name())
		return &ast.Field{
			Type:  typeIdnet,
			Names: []*ast.Ident{nameIdent},
		}
	}
}

// PkgPathAliasFromString takes a golang path as a string and returns an import alias for that path
func PkgPathAliasFromString(path string) string {
	splits := strings.FieldsFunc(path, splitPkgPath)
	return strings.Join(splits, "_")
}

func splitPkgPath(r rune) bool {
	return r == '.' || r == '/'
}
