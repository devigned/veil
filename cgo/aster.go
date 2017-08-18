package cgo

import (
	"go/ast"
)

type AstTransformer interface {
	ToAst() []ast.Decl
	ExportName() string
}
