package cgo

import (
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
	*Named
}

func NewStruct(named *types.Named) *Struct {
	if _, ok := named.Underlying().(*types.Struct); !ok {
		panic("only structs belong in structs")
	}
	return &Struct{NewNamed(named)}
}

// Struct returns the underlying struct
func (s Struct) Struct() *types.Struct {
	return s.Underlying().(*types.Struct)
}

// ToAst returns the go/ast representation of the CGo wrapper of the Array type
func (s Struct) ToAst() []ast.Decl {
	decls := []ast.Decl{s.NewAst(), s.StringAst()}
	decls = append(decls, s.FieldAccessorsAst()...)
	decls = append(decls, s.MethodAsts()...)
	return decls
}

func (s Struct) FieldAccessorsAst() []ast.Decl {
	var accessors []ast.Decl
	for i := 0; i < s.Struct().NumFields(); i++ {
		field := s.Struct().Field(i)
		if ShouldGenerateField(field) {
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
	castExpression := CastUnsafePtrOfTypeUuid(DeRef(s.CTypeName()), selfIdent)

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
	transformedLocalVarIdent := NewIdent("val")
	fieldIdent := NewIdent(field.Name())
	castExpression := CastUnsafePtrOfTypeUuid(DeRef(s.CTypeName()), selfIdent)
	typedField := UnsafePtrOrBasic(field, field.Type())
	typedField.Names = []*ast.Ident{localVarIdent}
	params := InstanceMethodParams(typedField)
	firstAssignmentCastRhs := CastExpr(field.Type(), localVarIdent)
	secondAssignment := ast.Expr(transformedLocalVarIdent)

	if isStringPointer(field.Type()) {
		strPtrCast := CastExpr(field.Type(), localVarIdent).(*ast.UnaryExpr)
		firstAssignmentCastRhs = strPtrCast.X
		secondAssignment = Ref(transformedLocalVarIdent)
	}

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
					Lhs: []ast.Expr{transformedLocalVarIdent},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{firstAssignmentCastRhs},
				},
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						&ast.SelectorExpr{
							X:   castExpression,
							Sel: fieldIdent,
						},
					},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{secondAssignment},
				},
			},
		},
	}

	return funcDecl
}

func (s Struct) FieldName(field *types.Var) string {
	return s.CName() + "_" + field.Name()
}

func (s Struct) IsConstructor(f *Func) bool {
	matches := constructorName.FindStringSubmatch(f.Name())
	if len(matches) > 1 && strings.HasPrefix(matches[1], s.Obj().Name()) {
		return true
	}
	return false
}

func (s Struct) ConstructorName(f *Func) string {
	return strings.Replace(f.Name(), s.Obj().Name(), "", 1)
}

func isStringPointer(t types.Type) bool {
	if ptr, ok := t.(*types.Pointer); ok {
		if basic, okB := ptr.Elem().(*types.Basic); okB && basic.Kind() == types.String {
			return true
		}
	}
	return false
}
