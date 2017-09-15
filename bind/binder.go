package bind

import (
	"bufio"
	"fmt"
	"github.com/devigned/veil/bind/python"
	"github.com/devigned/veil/cgo"
	"github.com/devigned/veil/core"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"path"
)

var (
	registry = map[string]func(*cgo.Package) core.Binder{"py3": python.NewBinder}
)

type wrapper struct {
	binder core.Binder
	pkg    *cgo.Package
}

func (b wrapper) Bind(outDir, libName string) error {
	code := toCodeFile(b.pkg)
	mainFile := path.Join(outDir, "main.go")
	f, err := os.Create(mainFile)
	if err != nil {
		return core.NewSystemErrorF("Unable to create %s", path.Join(outDir, "main.go"))
	}

	defer f.Close()

	w := bufio.NewWriter(f)
	printer.Fprint(w, &token.FileSet{}, code)
	w.Flush()

	buildSharedLib(outDir, libName)
	b.binder.Bind(outDir, libName)
	return nil
}

// NewBinder is a factory method for creating a new binder for a given target
func NewBinder(pkg *cgo.Package, target string) (core.Binder, error) {
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

func buildSharedLib(outDir, libName string) error {
	cmd := exec.Command("go", "build", "-o", libName, "-buildmode", "c-shared", ".")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = outDir

	if err := cmd.Run(); err != nil {
		return core.NewSystemErrorF("error building CGo shared library: %v\n", err)
	}

	return nil
}

// toCodeFile generates a CGo wrapper around the pkg
func toCodeFile(pkg *cgo.Package) *ast.File {
	cImport := cgo.Imports("C")

	cImport.Doc = &ast.CommentGroup{
		List: append(cgo.IncludeComments("<stdlib.h>"), cgo.RawComments(pkg.CDefinitions()...)...),
	}

	declarations := []ast.Decl{
		cImport,
		cgo.Imports("fmt", "sync", "unsafe", "github.com/satori/go.uuid"), //, "strconv", "strings", "os"
		cgo.ImportsFromMap(pkg.ImportAliases()),
		cgo.RefsStruct(),
		cgo.CObjectStruct(),
		cgo.DecrementRef(),
		cgo.IncrementRef(),
		cgo.GetRef(),
		cgo.GetUuidFromPtr(),
		cgo.Init(),
		cgo.ErrorToString(),
		cgo.CFree(),
		cgo.IsErrorNil(),
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
