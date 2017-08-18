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
	registry = map[string]func(*cgo.Package) core.Bindable{"py3": python.NewBinder}
)

type wrapper struct {
	binder core.Bindable
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
	w.Flush()

	buildSharedLib(outDir)
	b.binder.Bind(outDir)
	return nil
}

// NewBinder is a factory method for creating a new binder for a given target
func NewBinder(pkg *cgo.Package, target string) (core.Bindable, error) {
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

func buildSharedLib(outDir string) error {
	cmd := exec.Command("go", "build", "-buildmode", "c-shared", ".")
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
		List: cgo.IncludeComments("<stdlib.h>"),
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
