package cgo

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

var (
	strCFuncPtrMapType = &ast.MapType{
		Key:   NewIdent("string"),
		Value: unsafePointer,
	}
)

// Inerface is a helpful facade over types.Named which is intended to only contain an Interface
type Interface struct {
	named *Named
}

func NewInterface(named *types.Named) *Interface {
	if _, ok := named.Underlying().(*types.Interface); !ok {
		panic("only interfaces belong in Interface")
	}
	return &Interface{NewNamed(named)}
}

func (iface Interface) ExportedMethods() []*Func {
	var methods []*Func
	underlyingIface := iface.Interface()
	numMethods := underlyingIface.NumMethods()
	for i := 0; i < numMethods; i++ {
		meth := underlyingIface.Method(i)
		fun := NewBoundFunc(meth, iface.named)
		if fun.IsExportable() {
			methods = append(methods, fun)
		}
	}
	return methods
}

func (iface Interface) Interface() *types.Interface {
	return iface.named.Underlying().(*types.Interface)
}

// ToAst returns the go/ast representation of the CGo wrapper of the named type
func (iface Interface) ToAst() []ast.Decl {
	decls := []ast.Decl{
		iface.HelperStructAst(),
		iface.NewAst(),
		iface.StringAst(),
		iface.HelperCallbackRegistrationAst(),
	}
	decls = append(decls, iface.MethodAsts()...)
	return decls
}

func (iface Interface) Underlying() types.Type {
	return iface.named.Underlying()
}

func (iface Interface) ExportName() string {
	return iface.named.CName()
}

func (iface Interface) IsExportable() bool {
	return true
}

func (iface Interface) Name() string {
	return iface.named.Obj().Name()
}

func (iface Interface) CName() string {
	return iface.named.CName()
}

func (iface Interface) CDefs() (retTypes []string, funcPtrs []string, calls []string) {
	retTypes = []string{}
	funcPtrs = []string{}
	calls = []string{}
	for _, method := range iface.ExportedMethods() {
		r, f, c := method.CDefs()
		if r != "" {
			retTypes = append(retTypes, r)
		}
		funcPtrs = append(funcPtrs, f)
		calls = append(calls, c)
	}
	return retTypes, funcPtrs, calls
}

func (f Func) CDefs() (retTypes string, funcPtrs string, calls string) {
	sig := f.Signature()
	resLen := sig.Results().Len()
	paramLen := sig.Params().Len() + 1
	if resLen > 1 {
		//struct ReturnType_2 { void* r0; void* r1; };
		//typedef ReturnType_2 FuncPtr_2_2(void *bytes, void *handle);
		//inline ReturnType_2 CallHandleFunc_2_2(void *bytes, void *handle, FuncPtr_2_2 *fn) { return fn(bytes, handle); }
		returnTypeDefName := fmt.Sprintf(""+"ReturnType_%d", resLen)
		returnTypeDef := fmt.Sprintf("//struct %s {%s;};",
			returnTypeDefName,
			strings.Join(voidPtrs("r", resLen), "; "))

		funcArgs := strings.Join(voidPtrs("arg", paramLen), ", ")
		funcPtrDefName := f.CallbackFuncPtrName()
		funcPtrDef := fmt.Sprintf("//typedef struct %s %s(%s);",
			returnTypeDefName,
			funcPtrDefName,
			funcArgs)

		callHandleFuncDef := fmt.Sprintf("//inline struct %s %s(%s, %s *fn){ return fn(%s); }",
			returnTypeDefName,
			f.CallbackFuncName(),
			funcArgs,
			funcPtrDefName,
			strings.Join(argNames("arg", paramLen), ", "))

		return returnTypeDef, funcPtrDef, callHandleFuncDef
	} else if resLen == 1 {
		//typedef void FuncPtr_1_2(void *bytes, void *handle);
		//inline void CallHandleFunc_1_2(void *bytes, void *handle, FuncPtr_1_2 *fn) { return fn(bytes, handle); }
		funcArgs := strings.Join(voidPtrs("arg", paramLen), ", ")
		funcPtrDefName := f.CallbackFuncPtrName()
		funcPtrDef := fmt.Sprintf("//typedef void* %s(%s);",
			funcPtrDefName,
			funcArgs)

		callHandleFuncDef := fmt.Sprintf("//inline void* %s(%s, %s *fn){ return fn(%s); }",
			f.CallbackFuncName(),
			funcArgs,
			funcPtrDefName,
			strings.Join(argNames("arg", paramLen), ", "))

		return "", funcPtrDef, callHandleFuncDef
	} else {
		//typedef void FuncPtr_0_2(void *bytes, void *handle);
		//inline void CallHandleFunc_0_2(void *bytes, void *handle, FuncPtr_0_2 *fn) { return fn(bytes, handle); }
		funcArgs := strings.Join(voidPtrs("arg", paramLen), ", ")
		funcPtrDefName := f.CallbackFuncPtrName()
		funcPtrDef := fmt.Sprintf("//typedef void %s(%s);",
			funcPtrDefName,
			funcArgs)

		callHandleFuncDef := fmt.Sprintf("//inline void %s(%s, %s *fn){ return fn(%s); }",
			f.CallbackFuncName(),
			funcArgs,
			funcPtrDefName,
			strings.Join(argNames("arg", paramLen), ", "))

		return "", funcPtrDef, callHandleFuncDef
	}
}

func (f Func) CallbackFuncName() string {
	sig := f.Signature()
	return fmt.Sprintf("CallHandleFunc_%d_%d", sig.Results().Len(), sig.Params().Len())
}

func (f Func) CallbackFuncPtrName() string {
	sig := f.Signature()
	return fmt.Sprintf("FuncPtr_%d_%d", sig.Results().Len(), sig.Params().Len())
}

func voidPtrs(prefix string, length int) []string {
	results := make([]string, length)
	for i := 0; i < length; i++ {
		results[i] = fmt.Sprintf("void *%s%d", prefix, i)
	}
	return results
}

func argNames(prefix string, length int) []string {
	results := make([]string, length)
	for i := 0; i < length; i++ {
		results[i] = fmt.Sprintf("%s%d", prefix, i)
	}
	return results
}

// NewAst produces the []ast.Decl to construct a named type and increment it's reference count
func (iface Interface) NewAst() ast.Decl {
	functionName := iface.named.NewMethodName()
	handleIdent := NewIdent("handle")
	structInitialization := func(localVar *ast.Ident) []ast.Stmt {
		return []ast.Stmt{
			&ast.AssignStmt{
				Lhs: []ast.Expr{
					&ast.SelectorExpr{
						X:   localVar,
						Sel: handleIdent,
					},
				},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{
					handleIdent,
				},
			},
			&ast.AssignStmt{
				Lhs: []ast.Expr{
					&ast.SelectorExpr{
						X:   localVar,
						Sel: NewIdent("callbacks"),
					},
				},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{
					&ast.CompositeLit{
						Type: strCFuncPtrMapType,
					},
				},
			},
		}
	}

	params := []*ast.Field{
		{
			Names: []*ast.Ident{handleIdent},
			Type:  unsafePointer,
		},
	}
	return NewAstWithInitialization(functionName, iface.helperStructName(), params, structInitialization)
}

// StringAst produces the []ast.Decl to provide a string representation of the named type
func (iface Interface) StringAst() ast.Decl {
	functionName := iface.named.ToStringMethodName()
	return StringAst(functionName, iface.helperStructName())
}

// CTypeName returns the selector expression for the Named aliased package and type
func (iface Interface) CTypeName() ast.Expr {
	pkgPathIdent := NewIdent(PkgPathAliasFromString(iface.named.Obj().Pkg().Path()))
	typeIdent := NewIdent(iface.named.Obj().Name() + "_helper")
	return &ast.SelectorExpr{
		X:   pkgPathIdent,
		Sel: typeIdent,
	}
}

func (iface Interface) MethodAsts() []ast.Decl {
	methods := iface.ExportedMethods()
	asts := make([]ast.Decl, len(methods))
	for idx, meth := range methods {
		asts[idx] = meth.InterfaceCallbackAst(iface)
	}
	return asts
}

func (f Func) InterfaceCallbackAst(iface Interface) ast.Decl {
	sig := f.Signature()
	params := make([]*ast.Field, sig.Params().Len())
	for i := 0; i < len(params); i++ {
		p := sig.Params().At(i)
		params[i] = &ast.Field{
			Names: []*ast.Ident{NewIdent(p.Name())},
			Type:  TypeExpression(p.Type()),
		}
	}

	results := make([]*ast.Field, sig.Results().Len())
	for i := 0; i < len(results); i++ {
		r := sig.Results().At(i)
		results[i] = &ast.Field{
			Names: []*ast.Ident{NewIdent(r.Name())},
			Type:  TypeExpression(r.Type()),
		}
	}

	recvIdent := NewIdent("iface")
	funIdent := NewIdent("fun")
	okIdent := NewIdent("ok")
	resIdent := NewIdent("res")
	callbackMap := func(idx ast.Expr) ast.Expr {
		return &ast.IndexExpr{
			X: &ast.SelectorExpr{
				X:   recvIdent,
				Sel: NewIdent("callbacks"),
			},
			Index: idx,
		}
	}

	//func (r readerHelper) Read(bytes []byte) (int, error) {
	//	fun, ok := r.callbacks["read"]
	//	if !ok {
	//		fmt.Println("didn't find read method!!")
	//		return 0, nil
	//	} else {
	//		fmt.Println("read callback ptr: ", fun)
	//
	//		r0 := C.Call_HandleFunc(C.CBytes(cgo_incref(unsafe.Pointer(&bytes)).Bytes()), r.handle, fun)
	//		fmt.Println(r0)
	//		return 42, nil
	//	}
	//}

	callArgs := make([]ast.Expr, len(params)+2)
	for i := 0; i < len(params); i++ {
		p := sig.Params().At(i)
		callArgs[i] = CastOut(p.Type(), params[i].Names[0])
	}

	callArgs[len(params)] = &ast.SelectorExpr{
		X:   recvIdent,
		Sel: NewIdent("handle"),
	}

	callArgs[len(params)+1] = &ast.CallExpr{
		Fun: DeRef(&ast.SelectorExpr{
			X:   NewIdent("C"),
			Sel: NewIdent(f.CallbackFuncPtrName()),
		}),
		Args: []ast.Expr{funIdent},
	}

	resultStmts := make([]ast.Expr, len(results))
	for i := 0; i < len(results); i++ {
		var expr ast.Expr = CastUnsafePtr(DeRef(results[i].Type), &ast.SelectorExpr{
			X:   resIdent,
			Sel: NewIdent(fmt.Sprintf("r%d", i)),
		})
		if _, ok := results[i].Type.(*ast.StarExpr); !ok {
			expr = DeRef(expr)
		}
		resultStmts[i] = expr
	}

	body := []ast.Stmt{
		// fun, ok := r.callbacks["read"]
		&ast.AssignStmt{
			Lhs: []ast.Expr{funIdent, okIdent},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{callbackMap(&ast.BasicLit{
				Kind:  token.STRING,
				Value: "\"" + f.Name() + "\"",
			})},
		},
		&ast.IfStmt{
			// if ok {
			Cond: okIdent,
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.AssignStmt{
						Lhs: []ast.Expr{resIdent},
						Tok: token.DEFINE,
						Rhs: []ast.Expr{
							&ast.CallExpr{
								Fun: &ast.SelectorExpr{
									X:   NewIdent("C"),
									Sel: NewIdent(f.CallbackFuncName()),
								},
								Args: callArgs,
							},
						},
					},
					Return(resultStmts...),
				},
			},
			// } else {
			Else: &ast.BlockStmt{
				List: []ast.Stmt{
					// panic("
					&ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: NewIdent("panic"),
							Args: []ast.Expr{
								&ast.BasicLit{
									Kind:  token.STRING,
									Value: "\"can't find registerd method: " + f.Name() + "\"",
								},
							},
						},
					},
				},
			},
		},
	}

	funcDecl := &ast.FuncDecl{
		Name: NewIdent(f.Name()),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: params,
			},
			Results: &ast.FieldList{
				List: results,
			},
		},
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{recvIdent},
					Type:  iface.helperStructName(),
				},
			},
		},
		Body: &ast.BlockStmt{
			List: body,
		},
	}
	return funcDecl
}

func (iface Interface) HelperStructAst() ast.Decl {
	return &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: iface.helperStructName(),
				Type: &ast.StructType{
					Fields: &ast.FieldList{
						List: []*ast.Field{
							{
								Names: []*ast.Ident{NewIdent("handle")},
								Type:  unsafePointer,
							},
							{
								Names: []*ast.Ident{NewIdent("callbacks")},
								Type:  strCFuncPtrMapType,
							},
						},
					},
				},
			},
		},
	}
}

func (iface Interface) HelperCallbackRegistrationAst() ast.Decl {
	funcName := iface.named.CName() + "_register_callback"
	selfIdent := NewIdent("self")
	helperIdent := NewIdent("helper")
	methodNameIdent := NewIdent("methodName")
	funcPtrIdent := NewIdent("funcPtr")
	strIdent := NewIdent("strMethodName")

	castExpression := CastUnsafePtrOfTypeUuid(DeRef(iface.helperStructName()), selfIdent)

	selfCastAssign := &ast.AssignStmt{
		Lhs: []ast.Expr{helperIdent},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{castExpression},
	}

	//func veil_reader_helper_register_callback(self unsafe.Pointer, methodName *C.char, cfn unsafe.Pointer) {
	funcDecl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: ExportComments(funcName),
		},
		Name: NewIdent(funcName),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{selfIdent},
						Type:  unsafePointer,
					},
					{
						Names: []*ast.Ident{methodNameIdent},
						Type:  charStarType,
					},
					{
						Names: []*ast.Ident{funcPtrIdent},
						Type:  unsafePointer,
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				// helper := (*readerHelper)(cgo_get_ref(cgo_get_uuid_from_ptr(self)))
				selfCastAssign,
				// strMethodName := C.GoString(methodName)
				&ast.AssignStmt{
					Lhs: []ast.Expr{strIdent},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{ToGoString(methodNameIdent)},
				},
				// helper.callbacks[str] = cfn
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						&ast.IndexExpr{
							X: &ast.SelectorExpr{
								X:   helperIdent,
								Sel: NewIdent("callbacks"),
							},
							Index: strIdent,
						},
					},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{funcPtrIdent},
				},
			},
		},
	}
	return funcDecl
}

func (iface Interface) helperStructName() *ast.Ident {
	return NewIdent(iface.named.CName() + "_helper")
}
