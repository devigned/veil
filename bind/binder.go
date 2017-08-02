package bind

import (
	"fmt"
	"go/ast"

	"bufio"
	"github.com/devigned/veil/cgo"
	"github.com/devigned/veil/core"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path"
)

var (
	registry = map[string]func(*cgo.Package) Bindable{"py3": NewPy3Binder}
)

// Bindable is the interface for any object that will create a binding for a golang.Package
type Bindable interface {
	Bind(outDir string) error
}

type wrapper struct {
	binder Bindable
	pkg    *cgo.Package
}

func (b wrapper) Bind(outDir string) error {
	code := toCodeFile(b.pkg)
	mainFile := path.Join(outDir, "main.go")
	f, err := os.Create(mainFile)
	if err != nil {
		return core.NewSystemErrorF("Unable to create %s", path.Join(outDir, "main.go"))
	}

	defer f.Close()

	w := bufio.NewWriter(f)
	printer.Fprint(w, &token.FileSet{}, code)
	defer w.Flush()

	b.binder.Bind(outDir)
	return nil
}

// NewBinder is a factory method for creating a new binder for a given target
func NewBinder(pkg *cgo.Package, target string) (Bindable, error) {
	binderFactory, ok := registry[target]
	if !ok {
		return nil, core.NewSystemError(fmt.Sprintf("I don't know how to create a binder for %s", target))
	}

	bindable := wrapper{
		binder: binderFactory(pkg),
		pkg:    pkg,
	}

	return bindable, nil
}

// toCodeFile generates a CGo wrapper around the pkg
func toCodeFile(pkg *cgo.Package) *ast.File {

	// printPracticeAst()
	declarations := []ast.Decl{
		cgo.Imports("C"),
		cgo.Imports("fmt", "sync", "unsafe"), //, "strconv", "strings", "os"
		cgo.ImportsFromMap(pkg.ImportAliases()),
		cgo.RefsStruct(),
		cgo.CObjectStruct(),
		cgo.DecrementRef(),
		cgo.IncrementRef(),
	}

	declarations = append(declarations, pkg.ToAst()...)
	declarations = append(declarations, cgo.MainFunc())

	mainFile := &ast.File{
		Name: &ast.Ident{
			Name: "main",
		},
		Decls: declarations,
	}

	return mainFile
}

func printPracticeAst() {
	src := `

	package main

		`

	fset := token.NewFileSet() // positions are relative to fset
	f, _ := parser.ParseFile(fset, "src.go", src, parser.ParseComments)
	ast.Print(fset, f)
}
