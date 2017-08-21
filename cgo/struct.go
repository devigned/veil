package cgo

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"regexp"
	"strings"
)

var (
	constructorName = regexp.MustCompile(`^New([A-Z]\w+)`)
)

// Struct is a helpful facade over types.Named which is intended to only contain a struct
type Struct struct {
	*types.Named
}

func NewStruct(named *types.Named) *Struct {
	if _, ok := named.Underlying().(*types.Struct); !ok {
		panic("only structs belong in structs")
	}
	return &Struct{named}
}

// Struct returns the underlying struct
func (s Struct) Struct() *types.Struct {
	if _, ok := s.Named.Underlying().(*types.Struct); !ok {
		fmt.Println(s.Named)
	}
	return s.Named.Underlying().(*types.Struct)
}

// Methods returns the list of methods decorated on the struct
func (s Struct) Methods() []*types.Func {
	var methods []*types.Func
	for i := 0; i < s.Named.NumMethods(); i++ {
		meth := s.Named.Method(i)
		methods = append(methods, meth)
	}
	return methods
}

// Underlying returns the underlying type
func (s Struct) Underlying() types.Type { return s.Named }

// Underlying returns the string representation of the type (types.Type)
func (s Struct) String() string { return types.TypeString(s.Named, nil) }

// CGoName returns the fully resolved name to the struct
func (s Struct) CGoName() string {
	return PkgPathAliasFromString(s.Named.Obj().Pkg().Path()) + "_" + s.Named.Obj().Name()
}

// CGoType returns the selector expression for the Struct aliased package and type
func (s Struct) CGoType() ast.Expr {
	return CGoType(s.Named)
}

// CGoType returns the selector expression for the aliased package and type
func CGoType(n *types.Named) ast.Expr {
	pkgPathIdent := NewIdent(PkgPathAliasFromString(n.Obj().Pkg().Path()))
	typeIdent := NewIdent(n.Obj().Name())
	return &ast.SelectorExpr{
		X:   pkgPathIdent,
		Sel: typeIdent,
	}
}

func (s Struct) NewMethodName() string {
	return s.CGoName() + "_new"
}

func (s Struct) ToStringMethodName() string {
	return s.CGoName() + "_str"
}

// ToAst returns the go/ast representation of the CGo wrapper of the Array type
func (s Struct) ToAst() []ast.Decl {
	decls := []ast.Decl{s.NewAst(), s.StringAst()}
	decls = append(decls, s.FieldAccessorsAst()...)
	return decls
}

func (s Struct) ExportName() string {
	return s.CGoName()
}

// NewAst produces the []ast.Decl to construct a slice type and increment it's reference count
func (s Struct) NewAst() ast.Decl {
	functionName := s.NewMethodName()
	return NewAst(functionName, s.CGoType())
}

// StringAst produces the []ast.Decl to provide a string representation of the slice
func (s Struct) StringAst() ast.Decl {
	functionName := s.ToStringMethodName()
	return StringAst(functionName, s.CGoType())
}

func (s Struct) FieldAccessorsAst() []ast.Decl {
	var accessors []ast.Decl
	for i := 0; i < s.Struct().NumFields(); i++ {
		field := s.Struct().Field(i)
		if ShouldGenerate(field) {
			accessors = append(accessors, s.Getter(field), s.Setter(field))
		}
	}

	return accessors
}

func (s Struct) Getter(field *types.Var) ast.Decl {
	functionName := s.FieldName(field) + "_get"
	selfIdent := NewIdent("self")
	localVarIdent := NewIdent("value")
	fieldIdent := NewIdent(field.Name())
	castExpression := CastUnsafePtrOfTypeUuid(DeRef(s.CGoType()), selfIdent)

	assignment := &ast.AssignStmt{
		Lhs: []ast.Expr{localVarIdent},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			&ast.SelectorExpr{
				X:   castExpression,
				Sel: fieldIdent,
			},
		},
	}

	funcDecl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: ExportComments(functionName),
		},
		Name: NewIdent(functionName),
		Type: &ast.FuncType{
			Params: InstanceMethodParams(),
			Results: &ast.FieldList{
				List: []*ast.Field{{Type: TypeToArgumentTypeExpr(field.Type())}},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				assignment,
				Return(CastOut(field.Type(), localVarIdent)),
			},
		},
	}

	return funcDecl
}

func (s Struct) Setter(field *types.Var) ast.Decl {
	functionName := s.FieldName(field) + "_set"
	selfIdent := NewIdent("self")
	localVarIdent := NewIdent("value")
	fieldIdent := NewIdent(field.Name())
	castExpression := CastUnsafePtrOfTypeUuid(DeRef(s.CGoType()), selfIdent)

	typedField := UnsafePtrOrBasic(field, field.Type())
	typedField.Names = []*ast.Ident{localVarIdent}
	params := InstanceMethodParams(typedField)
	rhs := CastExpr(field.Type(), localVarIdent)

	funcDecl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: ExportComments(functionName),
		},
		Name: NewIdent(functionName),
		Type: &ast.FuncType{
			Params: params,
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						&ast.SelectorExpr{
							X:   castExpression,
							Sel: fieldIdent,
						},
					},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{rhs},
				},
			},
		},
	}

	return funcDecl
}

func (s Struct) FieldName(field *types.Var) string {
	return s.CGoName() + "_" + field.Name()
}

func (s Struct) IsConstructor(f Func) bool {
	matches := constructorName.FindStringSubmatch(f.Name())
	if len(matches) > 1 && strings.HasPrefix(matches[1], s.Named.Obj().Name()) {
		return true
	}
	return false
}

func (s Struct) ConstructorName(f Func) string {
	return strings.Replace(f.Name(), s.Named.Obj().Name(), "", 1)
}

func ShouldGenerate(f *types.Var) bool {
	return f.Exported() && !strings.Contains(f.Type().String(), "/vendor/")
}
