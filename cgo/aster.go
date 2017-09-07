package cgo

import (
	"go/ast"
	"go/types"
)

type AstTransformer interface {
	ToAst() []ast.Decl
	Underlying() types.Type
	Exportable
}

type Aliased interface {
	Alias() string
	Path() string
}

type Exportable interface {
	IsExportable() bool
	ExportName() string
}
