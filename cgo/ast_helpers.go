package cgo

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

const (
	GET_REF_FUNC_NAME         = "cgo_get_ref"
	GET_UUID_FROM_PTR_NAME    = "cgo_get_uuid_from_ptr"
	INCREMENT_REF_FUNC_NAME   = "cgo_incref"
	DECREMENT_REF_FUNC_NAME   = "cgo_decref"
	ERROR_TO_STRING_FUNC_NAME = "cgo_error_to_string"
	IS_ERROR_NIL_FUNC_NAME    = "cgo_is_error_nil"
	CFREE_FUNC_NAME           = "cgo_cfree"
	COBJECT_STRUCT_TYPE_NAME  = "cobject"
	REFS_VAR_NAME             = "refs"
	REFS_STRUCT_FIELD_NAME    = "refs"
)

var (
	unsafePointer = &ast.SelectorExpr{
		X:   NewIdent("unsafe"),
		Sel: NewIdent("Pointer"),
	}

	uuidType = &ast.SelectorExpr{
		X:   NewIdent("uuid"),
		Sel: NewIdent("UUID"),
	}

	cBytesType = &ast.SelectorExpr{
		X:   NewIdent("C"),
		Sel: NewIdent("CBytes"),
	}

	cStringType = &ast.SelectorExpr{
		X:   NewIdent("C"),
		Sel: NewIdent("CString"),
	}

	charStarType = DeRef(&ast.SelectorExpr{
		X:   NewIdent("C"),
		Sel: NewIdent("char"),
	})
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
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: NewIdent(REFS_VAR_NAME),
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
								Names: []*ast.Ident{NewIdent(REFS_STRUCT_FIELD_NAME)},
								Type: &ast.MapType{
									Key:   unsafePointer,
									Value: uuidType,
								},
							},
							{
								Names: []*ast.Ident{NewIdent("ptrs")},
								Type: &ast.MapType{
									Key:   uuidType,
									Value: NewIdent(COBJECT_STRUCT_TYPE_NAME),
								},
							},
						},
					},
				},
			},
		},
	}
}

// NewAst produces the []ast.Decl to construct a slice type and increment it's reference count
func NewAst(functionName string, goType ast.Expr) ast.Decl {
	localVarIdent := NewIdent("o")
	target := &ast.UnaryExpr{
		Op: token.AND,
		X:  localVarIdent,
	}

	funcDecl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: ExportComments(functionName),
		},
		Name: NewIdent(functionName),
		Type: &ast.FuncType{
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: unsafePointer},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				DeclareVar(localVarIdent, goType),
				Return(UuidToCBytes(IncrementRefCall(target))),
			},
		},
	}

	return funcDecl
}

// StringAst produces the []ast.Decl to provide a string representation of the slice
func StringAst(functionName string, goType ast.Expr) ast.Decl {
	selfIdent := NewIdent("self")
	deRef := DeRef(CastUnsafePtrOfTypeUuid(DeRef(goType), selfIdent))
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
						Type:  unsafePointer,
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: charStarType},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				Return(&ast.CallExpr{
					Fun: cStringType,
					Args: []ast.Expr{
						sprintf,
					},
				}),
			},
		},
	}

	return funcDecl
}

func refLockUnlockDefer() []ast.Stmt {
	refsVar := NewIdent(REFS_VAR_NAME)
	return []ast.Stmt{
		// refs.Lock()
		&ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   refsVar,
					Sel: NewIdent("Lock"),
				},
			},
		},
		// defer refs.Unlock()
		&ast.DeferStmt{
			Call: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   refsVar,
					Sel: NewIdent("Unlock"),
				},
			},
		},
	}
}

func UuidToCBytes(uuidExpr ast.Expr) ast.Expr {
	return &ast.CallExpr{
		Fun: cBytesType,
		Args: []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   uuidExpr,
					Sel: NewIdent("Bytes"),
				},
			},
		},
	}
}

func GetUuidFromPtr() ast.Decl {
	bytes := NewIdent("bytes")
	uid := NewIdent("uid")

	// func cgo_get_uuid_from_ptr(self unsafe.Pointer) uuid.UUID
	return &ast.FuncDecl{
		Name: NewIdent(GET_UUID_FROM_PTR_NAME),
		Type: &ast.FuncType{
			Params: InstanceMethodParams(),
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: uuidType,
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				// bytes := C.GoBytes(self, 16)
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						bytes,
					},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   NewIdent("C"),
								Sel: NewIdent("GoBytes"),
							},
							Args: []ast.Expr{
								NewIdent("self"),
								&ast.BasicLit{
									Kind:  token.INT,
									Value: "16",
								},
							},
						},
					},
				},
				// uid, _ := uuid.FromBytes(bytes)
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						uid,
						NewIdent("_"),
					},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   NewIdent("uuid"),
								Sel: NewIdent("FromBytes"),
							},
							Args: []ast.Expr{
								bytes,
							},
						},
					},
				},
				// return uid
				Return(uid),
			},
		},
	}
}

func Init() ast.Decl {
	refsVar := NewIdent(REFS_VAR_NAME)
	refsField := NewIdent(REFS_STRUCT_FIELD_NAME)
	statements := refLockUnlockDefer()
	statements = append(statements,
		// refs.refs = make(map[unsafe.Pointer]uuid.UUID
		&ast.AssignStmt{
			Lhs: []ast.Expr{
				&ast.SelectorExpr{
					X:   refsVar,
					Sel: refsField,
				},
			},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: NewIdent("make"),
					Args: []ast.Expr{
						&ast.MapType{
							Key:   unsafePointer,
							Value: uuidType,
						},
					},
				},
			},
		},
		// refs.ptrs = make(map[uuid.UUID]cobject)
		&ast.AssignStmt{
			Lhs: []ast.Expr{
				&ast.SelectorExpr{
					X:   refsVar,
					Sel: NewIdent("ptrs"),
				},
			},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: NewIdent("make"),
					Args: []ast.Expr{
						&ast.MapType{
							Key:   uuidType,
							Value: NewIdent(COBJECT_STRUCT_TYPE_NAME),
						},
					},
				},
			},
		})
	return &ast.FuncDecl{
		Name: NewIdent("init"),
		Type: &ast.FuncType{},
		Body: &ast.BlockStmt{
			List: statements,
		},
	}
}

func DecrementRef() ast.Decl {
	ptr := NewIdent("ptr")
	refsType := NewIdent(REFS_VAR_NAME)
	refsField := NewIdent(REFS_STRUCT_FIELD_NAME)
	uid := NewIdent("uid")
	ok := NewIdent("ok")
	cobj := NewIdent("cobj")
	ptrs := NewIdent("ptrs")
	cnt := NewIdent("cnt")
	del := NewIdent("delete")

	statements := refLockUnlockDefer()
	statements = append(statements,
		// uid := cgo_get_uuid_from_ptr(ptr)
		&ast.AssignStmt{
			Lhs: []ast.Expr{
				uid,
			},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: NewIdent(GET_UUID_FROM_PTR_NAME),
					Args: []ast.Expr{
						ptr,
					},
				},
			},
		},
		// cobj, ok := refs.ptrs[uid]
		&ast.AssignStmt{
			Lhs: []ast.Expr{
				cobj,
				ok,
			},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				&ast.IndexExpr{
					X: &ast.SelectorExpr{
						X:   refsType,
						Sel: ptrs,
					},
					Index: uid,
				},
			},
		},
		// if !ok {
		&ast.IfStmt{
			Cond: &ast.UnaryExpr{
				Op: token.NOT,
				X:  ok,
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					// panic("decref untracted object")
					&ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: NewIdent("panic"),
							Args: []ast.Expr{
								&ast.BasicLit{Value: "\"decref untracked object!\"", Kind: token.STRING},
							},
						},
					},
				},
			},
		},
		// }
		// if cobj.cnt -1 <= 0 {
		&ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X: &ast.BinaryExpr{
					X: &ast.SelectorExpr{
						X:   cobj,
						Sel: cnt,
					},
					Op: token.SUB,
					Y:  &ast.BasicLit{Value: "1", Kind: token.INT},
				},
				Op: token.LEQ,
				Y:  &ast.BasicLit{Value: "0", Kind: token.INT},
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					// delete(refs.ptrs, uid)
					&ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: del,
							Args: []ast.Expr{
								&ast.SelectorExpr{
									X:   refsType,
									Sel: ptrs,
								},
								uid,
							},
						},
					},
					// delete(cobj.refs, ptr)
					&ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: del,
							Args: []ast.Expr{
								&ast.SelectorExpr{
									X:   refsType,
									Sel: refsField,
								},
								&ast.SelectorExpr{
									X:   cobj,
									Sel: ptr,
								},
							},
						},
					},
					&ast.ReturnStmt{},
				},
			},
		},
		// }
		// refs.ptrs[uid] = cobject{cobj.ptr, cobj.cnt - 1}
		&ast.AssignStmt{
			Lhs: []ast.Expr{
				&ast.IndexExpr{
					X: &ast.SelectorExpr{
						X:   refsType,
						Sel: ptrs,
					},
					Index: uid,
				},
			},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				&ast.CompositeLit{
					Type: NewIdent(COBJECT_STRUCT_TYPE_NAME),
					Elts: []ast.Expr{
						&ast.SelectorExpr{
							X:   cobj,
							Sel: ptr,
						},
						&ast.BinaryExpr{
							X: &ast.SelectorExpr{
								X:   cobj,
								Sel: NewIdent("cnt"),
							},
							Op: token.SUB,
							Y:  &ast.BasicLit{Value: "1", Kind: token.INT},
						},
					},
				},
			},
		})

	return &ast.FuncDecl{
		Doc:  &ast.CommentGroup{List: ExportComments(DECREMENT_REF_FUNC_NAME)},
		Name: NewIdent(DECREMENT_REF_FUNC_NAME),
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
			List: statements,
		},
	}
}

func IncrementRef() ast.Decl {
	ptr := NewIdent("ptr")
	refsType := NewIdent(REFS_VAR_NAME)
	refsField := NewIdent(REFS_STRUCT_FIELD_NAME)
	uid := NewIdent("uid")
	ok := NewIdent("ok")
	s := NewIdent("s")
	ptrs := NewIdent("ptrs")

	statements := refLockUnlockDefer()
	statements = append(statements,
		// uid, ok := refs.refs[ptr]
		&ast.AssignStmt{
			Lhs: []ast.Expr{
				uid,
				ok,
			},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				&ast.IndexExpr{
					X: &ast.SelectorExpr{
						X:   refsType,
						Sel: refsField,
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
					// s := refs.ptrs[uid]
					&ast.AssignStmt{
						Lhs: []ast.Expr{s},
						Tok: token.DEFINE,
						Rhs: []ast.Expr{
							&ast.IndexExpr{
								X: &ast.SelectorExpr{
									X:   refsType,
									Sel: ptrs,
								},
								Index: uid,
							},
						},
					},
					// refs.ptrs[uid] = cobject{s.ptr, s.cnt + 1}
					&ast.AssignStmt{
						Lhs: []ast.Expr{
							&ast.IndexExpr{
								X: &ast.SelectorExpr{
									X:   refsType,
									Sel: ptrs,
								},
								Index: uid,
							},
						},
						Tok: token.ASSIGN,
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
					// uid = uuid.NewV4()
					&ast.AssignStmt{
						Lhs: []ast.Expr{uid},
						Tok: token.ASSIGN,
						Rhs: []ast.Expr{
							&ast.CallExpr{
								Fun: &ast.SelectorExpr{
									X:   NewIdent("uuid"),
									Sel: NewIdent("NewV4"),
								},
							},
						},
					},
					// refs.refs[ptr] = uid
					&ast.AssignStmt{
						Lhs: []ast.Expr{
							&ast.IndexExpr{
								X: &ast.SelectorExpr{
									X:   refsType,
									Sel: refsField,
								},
								Index: ptr,
							},
						},
						Tok: token.ASSIGN,
						Rhs: []ast.Expr{
							uid,
						},
					},
					// refs.ptrs[uid] = cobject{ptr, 1}
					&ast.AssignStmt{
						Lhs: []ast.Expr{
							&ast.IndexExpr{
								X: &ast.SelectorExpr{
									X:   refsType,
									Sel: ptrs,
								},
								Index: uid,
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
		Return(uid),
	)

	return &ast.FuncDecl{
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
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: uuidType,
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: statements,
		},
	}
}

func GetRef() ast.Decl {
	uid := NewIdent("uid")
	cobj := NewIdent("cobj")
	ok := NewIdent("ok")
	refs := NewIdent(REFS_VAR_NAME)
	statements := refLockUnlockDefer()
	statements = append(statements,
		&ast.IfStmt{
			Init: &ast.AssignStmt{
				Lhs: []ast.Expr{
					cobj,
					ok,
				},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{
					&ast.IndexExpr{
						X: &ast.SelectorExpr{
							X:   refs,
							Sel: NewIdent("ptrs"),
						},
						Index: uid,
					},
				},
			},
			Cond: ok,
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					Return(&ast.SelectorExpr{
						X:   cobj,
						Sel: NewIdent("ptr"),
					}),
				},
			},
			Else: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: NewIdent("panic"),
							Args: []ast.Expr{
								&ast.BasicLit{
									Kind:  token.STRING,
									Value: "\"ref untracked object!\"",
								},
							},
						},
					},
				},
			},
		})
	return &ast.FuncDecl{
		Name: NewIdent(GET_REF_FUNC_NAME),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{uid},
						Type:  uuidType,
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: unsafePointer,
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: statements,
		},
	}
}

func ErrorToString() ast.Decl {
	self := NewIdent("self")
	err := NewIdent("err")

	// func cgo_error_to_string(self unsafe.Pointer) string {
	return &ast.FuncDecl{
		Doc:  &ast.CommentGroup{List: ExportComments(ERROR_TO_STRING_FUNC_NAME)},
		Name: NewIdent(ERROR_TO_STRING_FUNC_NAME),
		Type: &ast.FuncType{
			Params: InstanceMethodParams(),
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: DeRef(&ast.SelectorExpr{
							X:   NewIdent("C"),
							Sel: NewIdent("char"),
						}),
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				// err := *(*error)(cgo_get_ref(cgo_get_uuid_from_ptr(ptr)))
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						err,
					},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						DeRef(CastUnsafePtrOfTypeUuid(DeRef(NewIdent("error")), self)),
					},
				},
				// return C.CString(err.Error())
				Return(ToCString(
					&ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   err,
							Sel: NewIdent("Error"),
						},
					})),
			},
		},
	}
	// }
}

func IsErrorNil() ast.Decl {
	self := NewIdent("self")
	err := NewIdent("err")

	// func cgo_error_is_nil(self unsafe.Pointer) string {
	return &ast.FuncDecl{
		Doc:  &ast.CommentGroup{List: ExportComments(IS_ERROR_NIL_FUNC_NAME)},
		Name: NewIdent(IS_ERROR_NIL_FUNC_NAME),
		Type: &ast.FuncType{
			Params: InstanceMethodParams(),
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: NewIdent("bool"),
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				// err := *(*error)(cgo_get_ref(cgo_get_uuid_from_ptr(ptr)))
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						err,
					},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						DeRef(CastUnsafePtrOfTypeUuid(DeRef(NewIdent("error")), self)),
					},
				},
				// return (*(*error)(self)) == nil
				Return(&ast.BinaryExpr{
					Op: token.EQL,
					X:  err,
					Y:  NewIdent("nil"),
				}),
			},
		},
	}
	// }
}

// CFree takes an unsafe pointer and frees the C memory associated with that pointer
func CFree() ast.Decl {
	self := NewIdent("self")

	// func cgo_cfree(self unsafe.Pointer) {
	return &ast.FuncDecl{
		Doc:  &ast.CommentGroup{List: ExportComments(CFREE_FUNC_NAME)},
		Name: NewIdent(CFREE_FUNC_NAME),
		Type: &ast.FuncType{
			Params: InstanceMethodParams(),
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				// C.free(self)
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   NewIdent("C"),
							Sel: NewIdent("free"),
						},
						Args: []ast.Expr{
							self,
						},
					},
				},
			},
		},
	}
	// }
}

func ToCString(targets ...ast.Expr) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   NewIdent("C"),
			Sel: NewIdent("CString"),
		},
		Args: targets,
	}
}

func ToGoString(targets ...ast.Expr) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   NewIdent("C"),
			Sel: NewIdent("GoString"),
		},
		Args: targets,
	}
}

func ToUnsafePointer(targets ...ast.Expr) ast.Expr {
	return &ast.CallExpr{
		Fun:  unsafePointer,
		Args: targets,
	}
}

// IncrementRefCall takes a target expression to increment it's cgo pointer ref and returns the expression
func IncrementRefCall(target ast.Expr) ast.Expr {
	return &ast.CallExpr{
		Fun:  NewIdent(INCREMENT_REF_FUNC_NAME),
		Args: []ast.Expr{ToUnsafePointer(target)},
	}
}

// DecrementRefCall takes a target expression to decrement it's cgo pointer ref and returns the expression
func DecrementRefCall(target ast.Expr) *ast.ExprStmt {
	return &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun:  NewIdent(DECREMENT_REF_FUNC_NAME),
			Args: []ast.Expr{target},
		},
	}
}

// NewIdent takes a name as string and returns an *ast.Ident in that name
func NewIdent(name string) *ast.Ident {
	return &ast.Ident{
		Name: name,
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

// CastUnsafePtr take a cast type and target expression and returns a cast expression
func CastUnsafePtr(castType, target ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: &ast.ParenExpr{
			X: castType,
		},
		Args: []ast.Expr{target},
	}
}

func CastUnsafePtrOfTypeUuid(castType, target ast.Expr) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.ParenExpr{
			X: castType,
		},
		Args: []ast.Expr{
			&ast.CallExpr{
				Fun: NewIdent(GET_REF_FUNC_NAME),
				Args: []ast.Expr{
					&ast.CallExpr{
						Fun: NewIdent(GET_UUID_FROM_PTR_NAME),
						Args: []ast.Expr{
							target,
						},
					},
				},
			},
		},
	}
}

// DeRef takes an expression and prefaces the expression with a *
func DeRef(expr ast.Expr) *ast.StarExpr {
	return &ast.StarExpr{X: expr}
}

// Ref takes an expression and prefaces the expression with a &
func Ref(expr ast.Expr) ast.Expr {
	if star, ok := expr.(*ast.StarExpr); ok {
		return star.X
	} else {
		return &ast.UnaryExpr{
			X:  expr,
			Op: token.AND,
		}
	}
}

// Return takes an expression and returns a return statement containing the expression
func Return(expressions ...ast.Expr) *ast.ReturnStmt {
	return &ast.ReturnStmt{
		Results: expressions,
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

// IncludeComments takes includeNames and returns AST comments for each include
func IncludeComments(includeNames ...string) []*ast.Comment {
	comments := make([]*ast.Comment, len(includeNames))
	for i := 0; i < len(comments); i++ {
		comments[i] = &ast.Comment{
			Text:  "//#include " + includeNames[i],
			Slash: token.Pos(1),
		}
	}
	return comments
}

// InstanceMethodParams returns a constructed field list for an instance method
func InstanceMethodParams(fields ...*ast.Field) *ast.FieldList {
	tmpFields := []*ast.Field{
		{
			Names: []*ast.Ident{NewIdent("self")},
			Type:  unsafePointer,
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
	if f.BoundRecv != nil {
		return buildBoundMethod(f)
	} else {
		return buildUnboundMethod(f)
	}
}

func buildBoundMethod(f *Func) *ast.FuncDecl {
	castSelfIdent := NewIdent("castSelf")
	functionName := f.CName()
	sig := f.Signature()
	args := []*ast.Field{}
	for i := 0; i < sig.Params().Len(); i++ {
		param := sig.Params().At(i)
		typedField := UnsafePtrOrBasic(param, param.Type())
		typedField.Names = []*ast.Ident{NewIdent(param.Name())}
		args = append(args, typedField)
	}
	params := InstanceMethodParams(args...)

	castExpression := CastUnsafePtrOfTypeUuid(DeRef(f.BoundRecv.CTypeName()), NewIdent("self"))

	selfCastAssign := &ast.AssignStmt{
		Lhs: []ast.Expr{castSelfIdent},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{castExpression},
	}

	callArgs := make([]ast.Expr, len(params.List)-1)
	assignStmts := make([]ast.Stmt, len(params.List)-1)
	for i := 0; i < len(args); i++ {
		param := sig.Params().At(i)
		varIdent := NewIdent(fmt.Sprintf("castArg%d", i+1))
		assignStmts[i] = &ast.AssignStmt{
			Lhs: []ast.Expr{varIdent},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{CastExpr(param.Type(), NewIdent(param.Name()))},
		}
		callArgs[i] = varIdent
	}

	functionCall := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   castSelfIdent,
			Sel: NewIdent(f.Name()),
		},
		Args: callArgs,
	}

	assign, returnStmt, results := buildFuncResults(sig, functionCall)
	var bodyStmts []ast.Stmt
	if returnStmt == nil {
		bodyStmts = append(assignStmts, selfCastAssign, assign)
	} else {
		bodyStmts = append(assignStmts, selfCastAssign, assign, returnStmt)
	}

	funcDecl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: ExportComments(functionName),
		},
		Name: NewIdent(functionName),
		Type: &ast.FuncType{
			Params: params,
		},
		Body: &ast.BlockStmt{List: bodyStmts},
	}

	if results != nil {
		funcDecl.Type.Results = results
	}

	return funcDecl
}

func buildUnboundMethod(f *Func) *ast.FuncDecl {
	functionName := f.CName()
	sig := f.Signature()

	functionCall := &ast.CallExpr{
		Fun:  f.AliasedGoName(),
		Args: ParamIdents(sig.Params()),
	}

	funcDecl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: ExportComments(functionName),
		},
		Name: NewIdent(functionName),
		Body: &ast.BlockStmt{List: []ast.Stmt{}},
	}

	params := Fields(sig.Params())

	assign, returnStmt, results := buildFuncResults(sig, functionCall)
	if returnStmt == nil {
		funcDecl.Body.List = append(funcDecl.Body.List, assign)
	} else {
		funcDecl.Body.List = append(funcDecl.Body.List, assign, returnStmt)
	}

	funcDecl.Type = &ast.FuncType{
		Params: params,
	}

	if results != nil {
		funcDecl.Type.Results = results
	}

	return funcDecl
}

func buildFuncResults(sig *types.Signature, functionCall ast.Expr) (ast.Stmt, *ast.ReturnStmt, *ast.FieldList) {
	if sig.Results().Len() > 0 {
		resultNames := make([]ast.Expr, sig.Results().Len())
		for i := 0; i < sig.Results().Len(); i++ {
			resultNames[i] = NewIdent(fmt.Sprintf("r%d", i))
		}

		assign := &ast.AssignStmt{
			Lhs: resultNames,
			Tok: token.DEFINE,
			Rhs: []ast.Expr{functionCall},
		}

		resultExprs := make([]ast.Expr, sig.Results().Len())
		for i := 0; i < sig.Results().Len(); i++ {
			result := sig.Results().At(i)
			resultExprs[i] = CastOut(result.Type(), resultNames[i])
		}

		return assign, Return(resultExprs...), Fields(sig.Results())
	} else {
		return &ast.ExprStmt{X: functionCall}, nil, nil
	}
}

func CastOut(t types.Type, name ast.Expr) ast.Expr {
	switch typ := t.(type) {
	case *types.Basic:
		if typ.Kind() == types.String {
			return ToCString(name)
		} else {
			return name
		}
	case *types.Pointer:
		// already have a pointer, so just count the reference
		return UuidToCBytes(IncrementRefCall(name))
	default:
		return UuidToCBytes(IncrementRefCall(Ref(name)))
	}
}

// ParamIdents transforms parameter tuples into a slice of AST expressions
func ParamIdents(funcParams *types.Tuple) []ast.Expr {
	if funcParams == nil || funcParams.Len() <= 0 {
		return []ast.Expr{}
	}

	args := make([]ast.Expr, funcParams.Len())
	for i := 0; i < funcParams.Len(); i++ {
		param := funcParams.At(i)
		args[i] = ParamExpr(param, param.Type())
	}
	return args
}

func ParamExpr(param *types.Var, t types.Type) ast.Expr {
	return CastExpr(t, NewIdent(param.Name()))
}

func CastExpr(t types.Type, ident ast.Expr) ast.Expr {
	switch t := t.(type) {
	case *types.Pointer:
		return Ref(CastExpr(t.Elem(), ident))
	case *types.Named:
		pkg := t.Obj().Pkg()
		typeName := t.Obj().Name()
		castExpr := DeRef(CastUnsafePtrOfTypeUuid(
			DeRef(&ast.SelectorExpr{
				X:   NewIdent(PkgPathAliasFromString(pkg.Path())),
				Sel: NewIdent(typeName),
			}),
			ident))
		return castExpr
	case *types.Slice:
		slice := NewSlice(t.Elem())
		goTypeExpr := slice.GoTypeExpr()
		castExpr := DeRef(CastUnsafePtrOfTypeUuid(DeRef(goTypeExpr), ident))
		return castExpr
	case *types.Basic:
		if t.Kind() == types.String {
			return ToGoString(ident)
		} else {
			return ident
		}
	default:
		return ident
	}
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
			fields[i] = UnsafePtrOrBasic(p, t.Elem())
		default:
			fields[i] = UnsafePtrOrBasic(p, t)
		}
	}
	return &ast.FieldList{List: fields}
}

// UnsafePtrOrBasic returns a Basic typed field or an unsafe pointer if not a Basic type
func UnsafePtrOrBasic(p *types.Var, t types.Type) *ast.Field {
	defaultTransform := func() *ast.Field {
		return &ast.Field{
			Type:  unsafePointer,
			Names: []*ast.Ident{NewIdent(p.Name())},
		}
	}
	switch typ := t.(type) {
	case *types.Basic:
		return VarToField(p, t)
	//case *types.Named, *types.Interface:
	//	if ImplementsError(t) {
	//		return &ast.Field{
	//			Type: NewIdent("error"),
	//		}
	//	} else {
	//		return returnDefault()
	//	}
	case *types.Pointer:
		if basic, ok := typ.Elem().(*types.Basic); ok {
			return VarToField(p, basic)
		}
		return defaultTransform()
	default:
		return defaultTransform()
	}
}

func TypeToArgumentTypeExpr(t types.Type) ast.Expr {
	if basic, ok := t.(*types.Basic); ok {
		if basic.Kind() == types.String {
			return charStarType
		} else {
			return NewIdent(basic.Name())
		}
	} else {
		return unsafePointer
	}
}

// VarToField transforms a Var into an AST field
func VarToField(p *types.Var, t types.Type) *ast.Field {
	name := p.Name()
	typeName := t.String()
	defaultAction := func() *ast.Field {
		return &ast.Field{
			Type:  NewIdent(typeName),
			Names: []*ast.Ident{NewIdent(name)},
		}
	}

	switch typ := t.(type) {
	case *types.Named:
		return NamedToField(p, typ)
	case *types.Pointer:
		if named, ok := typ.Elem().(*types.Named); ok {
			return NamedToField(p, named)
		} else if basic, ok := typ.Elem().(*types.Named); ok {
			return &ast.Field{
				Type:  NewIdent(basic.String()),
				Names: []*ast.Ident{NewIdent(name)},
			}
		} else {
			return defaultAction()
		}
	case *types.Basic:
		if typ.Kind() == types.String {
			return &ast.Field{
				Type:  charStarType,
				Names: []*ast.Ident{NewIdent(name)},
			}
		} else {
			return defaultAction()
		}
	default:
		return defaultAction()
	}
}

func ShouldGenerateField(v *types.Var) bool {
	if !v.Exported() {
		return false
	}
	return shouldGenerate(v, v.Type())
}

func ShouldGenerate(v *types.Var) bool {
	return shouldGenerate(v, v.Type())
}

func shouldGenerate(v *types.Var, t types.Type) bool {
	if strings.Contains(t.String(), "/vendor/") {
		return false
	}

	supportedType := true
	switch typ := t.(type) {
	case *types.Chan, *types.Map, *types.Signature:
		supportedType = false
	case *types.Interface:
		if !ImplementsError(typ) {
			supportedType = false
		}
	case *types.Pointer:
		return shouldGenerate(v, typ.Elem())
	case *types.Named:
		return shouldGenerate(v, typ.Underlying())
	}
	return supportedType
}

type hasMethods interface {
	NumMethods() int
	Method(int) *types.Func
}

// ImplementsError returns true if a type has an Error() string function signature
func ImplementsError(t types.Type) bool {
	isError := func(fun *types.Func) bool {
		if sig, ok := fun.Type().(*types.Signature); ok {
			if fun.Name() != "Error" {
				return false
			}

			if sig.Params().Len() != 0 {
				return false
			}

			if sig.Results().Len() == 1 {
				result := sig.Results().At(0)
				if obj, ok := result.Type().(*types.Basic); ok {
					return obj.Kind() == types.String
				}
			}
		}

		return false
	}

	hasErrorMethod := func(obj hasMethods) bool {
		numMethods := obj.NumMethods()
		for i := 0; i < numMethods; i++ {
			if isError(obj.Method(i)) {
				return true
			}
		}
		return false
	}

	if obj, ok := t.Underlying().(hasMethods); ok {
		return hasErrorMethod(obj)
	}

	return false
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
	return r == '.' || r == '/' || r == '-'
}
