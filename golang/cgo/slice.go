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
	return decls
}

func (s SliceWrapper) GoName() string {
	return "[]" + s.elem.String()
}

func (s SliceWrapper) CGoName() string {
	return "slice_of_" + s.elem.String()
}

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

	exportComment := []*ast.Comment{
		{
			Text:  "//export " + functionName,
			Slash: token.Pos(1),
		},
	}

	funcDecl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: exportComment,
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
