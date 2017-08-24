package cgo

import (
	"github.com/emirpasic/gods/maps"
	"github.com/marstr/collection"
	"go/ast"
	"go/token"
)

// Imports creates a GenDecl for a series of imports
func Imports(imports ...string) *ast.GenDecl {
	objs := collection.AsEnumerable(imports).Enumerate(nil).
		Select(func(item interface{}) interface{} {
			return &ast.ImportSpec{
				Path: &ast.BasicLit{
					Kind:  token.STRING,
					Value: "\"" + item.(string) + "\"",
				},
			}
		})

	var specs []ast.Spec
	for item := range objs {
		specs = append(specs, item.(ast.Spec))
	}

	return &ast.GenDecl{
		Tok:    token.IMPORT,
		Specs:  specs,
		Lparen: token.Pos(1),
	}
}

// ImportsFromMap create import ASTs from alias keys and package path values
func ImportsFromMap(imports maps.Map) ast.Decl {
	specs := []ast.Spec{}

	for _, k := range imports.Keys() {
		value, _ := imports.Get(k)
		spec := &ast.ImportSpec{
			Name: NewIdent(k.(string)),
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: "\"" + value.(string) + "\"",
			},
		}
		specs = append(specs, spec)
	}

	return &ast.GenDecl{
		Tok:    token.IMPORT,
		Specs:  specs,
		Lparen: token.Pos(1),
	}
}
