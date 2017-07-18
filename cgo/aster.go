package cgo

import (
	"go/ast"
)

type AstTransformer interface {
	ToCgoAst() []ast.Decl
}
