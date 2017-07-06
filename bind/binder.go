package bind

import (
	"fmt"
	"go/ast"

	"github.com/devigned/veil/core"
	"github.com/devigned/veil/golang"
	"go/token"
	"os"
	"bufio"
	"go/printer"
	"go/parser"
	"github.com/devigned/veil/bind/cgo"
)

var (
	registry = map[string]func(*golang.Package) Bindable{"py3": NewPy3Binder}
	rewriteRule = ptrTo("a[b:len(a)] -> a[b:]")
	rewrite    func(*ast.File) *ast.File
	fileSet = token.NewFileSet()
)

func ptrTo(s string) *string {
	return &s;
}

// Bindable is the interface for any object that will create a binding for a golang.Package
type Bindable interface {
	Bind(outDir string) error
}

// NewBinder is a factory method for creating a new binder for a given target
func NewBinder(pkg *golang.Package, target string) (Bindable, error) {
	bindable, ok := registry[target]

	if !ok {
		return nil, core.NewSystemError(fmt.Sprintf("I don't know how to create a binder for %s", target))
	}

	cgoAst(pkg)
	return bindable(pkg), nil
}

// cgoAst generates a map of file names and io.Writers which are the cgo substrate for targets to bind.
// The cgo layer is intended to normalize types from Go into more standard C types and provide a standard
// layer to build FFI language bindings.
func cgoAst(pkg *golang.Package) *ast.File {
	src := `

package main

import "C"

//export cgo_type_0x3129483107
// cgo_type_0x3129483107 wraps []string
type cgo_type_0x3129483107 unsafe.Pointer

//export cgo_func_0x3129483107_eface
func cgo_func_0x3129483107_eface(self cgo_type_0x3129483107) interface{} {
	var v interface{} = *(*[]string)(unsafe.Pointer(self))
	return v
}
	`

	fset := token.NewFileSet() // positions are relative to fset
	f, _ := parser.ParseFile(fset, "src.go", src, parser.ParseComments)
	ast.Print(fset, f)

	mainFile := &ast.File{
		Name: &ast.Ident{
			Name: "main",
		},
		Decls: []ast.Decl{
			cgo.Imports("C"),
			cgo.Imports("fmt", "sync", "unsafe", "strconv", "strings", "os"),
			cgo.WrapType("something", "unsafe.Pointer", "//export something", "// something wraps []string"),
			cgo.ArrayConstructor("string", "foo_bar"),
			&ast.FuncDecl{
				Name: &ast.Ident{
					Name: "main",
				},
				Type: &ast.FuncType{},
				Body: &ast.BlockStmt{},
			},
		},
	}

	// Print the AST.
	//ast.Print(&token.FileSet{}, mainFile)
	writer := bufio.NewWriter(os.Stdout)
	printer.Fprint(writer, &token.FileSet{}, mainFile)
	defer writer.Flush()
	return mainFile
}