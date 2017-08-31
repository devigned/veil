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
	*Named
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
		fun := NewBoundFunc(meth, iface.Named)
		if fun.IsExportable() {
			methods = append(methods, fun)
		}
	}
	return methods
}

func (iface Interface) Interface() *types.Interface {
	return iface.Underlying().(*types.Interface)
}

// ToAst returns the go/ast representation of the CGo wrapper of the named type
func (iface Interface) ToAst() []ast.Decl {
	decls := []ast.Decl{iface.HelperStructAst(), iface.NewAst(), iface.StringAst()}
	// decls = append(decls, iface.MethodAsts()...)
	return decls
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
	paramLen := sig.Params().Len()
	if resLen > 1 {
		//struct ReturnType_2 { void* r0; void* r1; };
		//typedef ReturnType_2 FuncPtr_2_2(void *bytes, void *handle);
		//inline ReturnType_2 CallHandleFunc_2_2(void *bytes, void *handle, FuncPtr_2_2 *fn) { return fn(bytes, handle); }
		returnTypeDefName := fmt.Sprintf(""+"ReturnType_%d", resLen)
		returnTypeDef := fmt.Sprintf("//struct %s {%s;};",
			returnTypeDefName,
			strings.Join(voidPtrs("r", resLen), "; "))

		funcArgs := strings.Join(voidPtrs("arg", paramLen), ", ")
		funcPtrDefName := fmt.Sprintf("FuncPtr_%d_%d", resLen, paramLen)
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
		funcPtrDefName := fmt.Sprintf("FuncPtr_%d_%d", resLen, paramLen)
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
		funcPtrDefName := fmt.Sprintf("FuncPtr_%d_%d", resLen, paramLen)
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
	functionName := iface.NewMethodName()
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
	functionName := iface.ToStringMethodName()
	return StringAst(functionName, iface.helperStructName())
}

// CTypeName returns the selector expression for the Named aliased package and type
func (iface Interface) CTypeName() ast.Expr {
	pkgPathIdent := NewIdent(PkgPathAliasFromString(iface.Obj().Pkg().Path()))
	typeIdent := NewIdent(iface.Obj().Name() + "_helper")
	return &ast.SelectorExpr{
		X:   pkgPathIdent,
		Sel: typeIdent,
	}
}

func (iface Interface) MethodsAsts() []ast.Decl {
	return []ast.Decl{}
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

func (iface Interface) helperStructName() *ast.Ident {
	return NewIdent(iface.CName() + "_helper")
}
