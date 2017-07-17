package cgo

import (
	"go/ast"
	"go/token"
	"go/types"
)

// ArrayWrapper is a wrapper for the
type SliceWrapper struct {
	elem types.Type
}

// NewSliceWrapper wraps types.Slice to provide a consistent comparison
func NewSliceWrapper(elem types.Type) SliceWrapper {
	return SliceWrapper{
		elem: elem,
	}
}

// Underlying returns the underlying type of the Slice (types.Type)
func (t SliceWrapper) Underlying() types.Type {
	return t
}

// Underlying returns the string representation of the type (types.Type)
func (t SliceWrapper) String() string {
	return types.TypeString(types.NewSlice(t.elem), nil)
}

// ToCgoAst returns the go/ast representation of the CGo wrapper of the Slice type
func (s SliceWrapper) ToCgoAst() []ast.Decl {
	decls := s.NewAst()
	decls = append(decls, s.StringAst()...)
	return decls
}

func (s SliceWrapper) GoName() string {
	return "[]" + s.elem.String()
}

func (s SliceWrapper) CGoName() string {
	return "slice_of_" + s.elem.String()
}

// NewAst produces the []ast.Decl to construct a slice type and increment it's reference count
func (s SliceWrapper) NewAst() []ast.Decl {
	functionName := s.CGoName() + "_new"
	localVarIdent := NewIdent("o")
	goTypeIdent := NewIdent(s.GoName())
	target := &ast.UnaryExpr{
		Op: token.AND,
		X:  localVarIdent,
	}

	goType := &ast.ArrayType{
		Elt: NewIdent(s.elem.String()),
	}

	funcDecl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: ExportComments(functionName),
		},
		Name: NewIdent(functionName),
		Type: &ast.FuncType{
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: goTypeIdent},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				DeclareVar(localVarIdent, goType),
				IncrementRef(target),
				CastReturn(goTypeIdent, target),
			},
		},
	}

	return []ast.Decl{funcDecl}
}

// StringAst produces the []ast.Decl to provide a string representation of the slice
func (s SliceWrapper) StringAst() []ast.Decl {
	functionName := s.CGoName() + "_str"
	selfIdent := NewIdent("self")
	goTypeIdent := NewIdent(s.GoName())
	stringIdent := NewIdent("string")

	castExpression := CastUnsafePtr(DeRef(goTypeIdent), selfIdent)
	deRef := DeRef(castExpression)
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
						Type:  goTypeIdent,
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: stringIdent},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				Return(sprintf),
			},
		},
	}

	return []ast.Decl{funcDecl}
}
