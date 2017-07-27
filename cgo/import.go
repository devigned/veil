package cgo

import (
	"github.com/marstr/collection"
	"go/ast"
	"go/token"
)

// Imports creates a GenDecl for a series of imports
func Imports(imports ...string) ast.Decl {
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

	var decl ast.Decl
	decl = &ast.GenDecl{
		Tok:    token.IMPORT,
		Specs:  specs,
		Lparen: token.Pos(1),
	}

	return decl
}

func AliasImports(imports map[string]string) ast.Decl {
	specs := []ast.Spec{}

	for k, v := range imports {
		spec := &ast.ImportSpec{
			Name: NewIdent(k),
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: "\"" + v + "\"",
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
