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
		//typedef struct ReturnType_2 { void* r0; void* r1; } ReturnType_2;
		//typedef ReturnType_2 FuncPtr_2_2(void *bytes, void *handle);
		//inline ReturnType_2 CallHandleFunc_2_2(void *bytes, void *handle, FuncPtr_2_2 *fn) { return fn(bytes, handle); }
		returnTypeDefName := fmt.Sprintf(""+"ReturnType_%d", resLen)
		returnTypeDef := fmt.Sprintf("//typedef struct %s {%s;} %s;",
			returnTypeDefName,
			strings.Join(paddedReturnVoidPtrs("r", resLen), "; "),
			returnTypeDefName)

		funcArgs := strings.Join(voidPtrs("arg", paramLen), ", ")
		funcPtrDefName := f.CallbackFuncPtrName()
		funcPtrDef := fmt.Sprintf("//typedef struct %s* %s(%s);",
			returnTypeDefName,
			funcPtrDefName,
			funcArgs)

		callHandleFuncDef := fmt.Sprintf("//inline struct %s* %s(%s, %s *fn){ return fn(%s); }",
			returnTypeDefName,
			f.CallbackFuncName(),
			funcArgs,
			funcPtrDefName,
			strings.Join(argNames("arg", paramLen), ", "))

		return returnTypeDef, funcPtrDef, callHandleFuncDef
	} else if resLen == 1 {
		//typedef void* FuncPtr_1_2(void *bytes, void *handle);
		//inline void* CallHandleFunc_1_2(void *bytes, void *handle, FuncPtr_1_2 *fn) { return fn(bytes, handle); }
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

func paddedReturnVoidPtrs(prefix string, length int) []string {
	results := make([]string, length+1)
	for i := 0; i < length; i++ {
		results[i] = fmt.Sprintf("void *%s%d", prefix, i)
	}
	results[length] = "void *empty"
	return results
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

/*
InterfaceCallbackAst produces proxy functions for interface methods which act as a C bridge between
Golang and the hosting language. It translates Golang Args to C args, calls a function callback into
the hosting language providing arguments, an object handle, and captures a return. The return struct
is then transformed back into Golang and C arguments are freed.

The function looks like the following.

func (iface veil_io_Reader_helper) Read(p []byte) (n int, err error) {
	fun, ok := iface.callbacks["Read"]
	if ok {
		arg0 := C.CBytes(cgo_incref(unsafe.Pointer(&p)).Bytes())
		res := C.CallHandleFunc_2_1(arg0, iface.handle, (*C.FuncPtr_2_1)(fun))
		cgo_decref(arg0)
		var r0 int
		if res.r0 == nil {
			panic("result: 0 is nil and must have a value")
		} else {
			r0 = *(*int)(res.r0)
		}
		var r1 error
		if res.r1 == nil {
			r1 = nil
		} else {
			r1 = *(*error)(res.r1)
		}
		return r0, r1
	} else {
		panic("can't find registerd method: Read")
	}
}
*/
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

	reqArgAssignments := make([]ast.Stmt, len(params))
	for i := 0; i < len(params); i++ {
		p := sig.Params().At(i)
		reqArgAssignments[i] = &ast.AssignStmt{
			Lhs: []ast.Expr{NewIdent(fmt.Sprintf("arg%d", i))},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{CastOut(p.Type(), params[i].Names[0])},
		}
	}

	freeArgStmts := make([]ast.Stmt, len(params))
	for i := 0; i < len(params); i++ {
		argIdent := NewIdent(fmt.Sprintf("arg%d", i))
		freeArgStmts[i] = &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun:  NewIdent("cgo_decref"),
				Args: []ast.Expr{argIdent},
			},
		}
	}

	callArgs := make([]ast.Expr, len(params)+2)
	for i := 0; i < len(params); i++ {
		callArgs[i] = NewIdent(fmt.Sprintf("arg%d", i))
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

	resultHandlers := []ast.Stmt{}
	for idx, res := range results {
		resIdxIdent := NewIdent(fmt.Sprintf("r%d", idx))
		// define result variable
		decl := &ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.TypeSpec{
						Name: resIdxIdent,
						Type: res.Type,
					},
				},
			},
		}

		var elseExpr ast.Expr = CastUnsafePtr(DeRef(results[idx].Type), &ast.SelectorExpr{
			X:   resIdent,
			Sel: resIdxIdent,
		})
		if _, ok := results[idx].Type.(*ast.StarExpr); !ok {
			elseExpr = DeRef(elseExpr)
		}

		var trueBody []ast.Stmt
		if isTypeNilable(sig.Results().At(idx).Type()) {
			trueBody = []ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{resIdxIdent},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{NewIdent("nil")},
				},
			}
		} else {
			trueBody = []ast.Stmt{
				&ast.ExprStmt{
					X: Panic(fmt.Sprintf("result: %d is nil and must have a value", idx)),
				},
			}
		}

		ifStmt := &ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X: &ast.SelectorExpr{
					X:   resIdent,
					Sel: resIdxIdent,
				},
				Op: token.EQL,
				Y:  NewIdent("nil"),
			},
			Body: &ast.BlockStmt{
				List: trueBody,
			},
			Else: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.AssignStmt{
						Lhs: []ast.Expr{resIdxIdent},
						Tok: token.ASSIGN,
						Rhs: []ast.Expr{elseExpr},
					},
				},
			},
		}

		resultHandlers = append(resultHandlers, decl, ifStmt)
	}

	resultExprs := make([]ast.Expr, len(results))
	for i := 0; i < len(results); i++ {
		resultExprs[i] = NewIdent(fmt.Sprintf("r%d", i))
	}

	// arg0 := C.CBytes(cgo_incref(unsafe.Pointer(&p)).Bytes()) ...
	ifStmtBody := reqArgAssignments
	// res := C.CallHandleFunc_2_1(arg0, iface.handle, (*C.FuncPtr_2_1)(fun))
	ifStmtBody = append(ifStmtBody,
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
		})
	// cgo_decref(arg0) ...
	ifStmtBody = append(ifStmtBody, freeArgStmts...)
	/*
		var r0 int
		if res.r0 == nil {
			panic("result: 0 is nil and must have a value")
		} else {
			r0 = *(*int)(res.r0)
		} ...
	*/
	ifStmtBody = append(ifStmtBody, resultHandlers...)
	// return r0, r1
	ifStmtBody = append(ifStmtBody, Return(resultExprs...))

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
				List: ifStmtBody,
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

func isTypeNilable(t types.Type) bool {
	nilable := func(typ types.Type) bool {
		switch typ.(type) {
		case *types.Pointer, *types.Interface:
			return true
		default:
			return false
		}
	}

	nilableType := t
	if nilable(nilableType) {
		return true
	}

	parentEqUnderlying := nilableType == nilableType.Underlying()
	for !parentEqUnderlying {
		underlying := nilableType.Underlying()
		if nilable(underlying) {
			return true
		}
		nilableType = underlying
		parentEqUnderlying = nilableType == nilableType.Underlying()
	}

	return false
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
