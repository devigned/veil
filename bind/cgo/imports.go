package cgo

import (
	"go/ast"
	"go/token"
	"github.com/marstr/collection"
)

// Imports creates a GenDecl for a series of imports
func Imports(imports ...string) ast.Decl {
	objs := collection.AsEnumerable(imports).Enumerate(nil).
		Select(func(item interface{}) interface{} {
		return &ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind: token.STRING,
				Value: "\"" + item.(string) + "\"",
			},
		}
	})

	var specs []ast.Spec
	for i := range objs {
		specs = append(specs, i.(ast.Spec))
	}

	var decl ast.Decl
	decl = &ast.GenDecl{
		Tok: token.IMPORT,
		Specs: specs,
		Lparen: token.Pos(1),
	}

	return decl
}