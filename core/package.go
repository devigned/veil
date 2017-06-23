package core

import (
	"go/build"
	"go/doc"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
)

// Package is a container for ast.Types and Docs
type Package struct {
	pkg     *types.Package
	doc     *doc.Package
	funcs   map[string]*types.Func
	structs map[string]*types.Struct
	objects map[string]interface{}
	consts  map[string]types.Const
}

// NewPackage constructs a Package from pkgPath using the specified working directory
func NewPackage(pkgPath string, workDir string) (*Package, error) {
	cmd := exec.Command("go", "get", pkgPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = workDir

	if err := cmd.Run(); err != nil {
		return nil, NewSystemErrorF("error installing [%s]: %v\n", pkgPath, err)
	}

	buildPkg, err := build.Import(pkgPath, workDir, 0)
	if err != nil {
		return nil, NewSystemErrorF("error resolving import path [%s]: %v\n", pkgPath, err)
	}

	typesPkg, err := importer.Default().Import(buildPkg.ImportPath)
	if err != nil {
		return nil, NewSystemErrorF("error importing package [%v]: %v\n", buildPkg.ImportPath, err)
	}

	fset := token.NewFileSet()
	astPkgs, err := parser.ParseDir(fset, buildPkg.Dir, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	astPkg, ok := astPkgs[typesPkg.Name()]
	if !ok {
		return nil, NewSystemErrorF("could not find AST for package %q", typesPkg.Name())
	}

	docPkg := doc.New(astPkg, buildPkg.ImportPath, 0)

	veilPkg := &Package{
		pkg:     typesPkg,
		doc:     docPkg,
		funcs:   make(map[string]*types.Func),
		structs: make(map[string]*types.Struct),
	}

	if err = veilPkg.build(); err != nil {
		return nil, err
	}

	return veilPkg, nil
}

func (p Package) GetFuncs() map[string]*types.Func {
	return p.funcs
}

func (p Package) GetStruct() map[string]*types.Struct {
	return p.structs
}

func (p *Package) build() error {

	scope := p.pkg.Scope()
	var scopeNames CollectionStringSlice = scope.Names()
	exportedObjects := scopeNames.Enumerate().
		Where(func(name interface{}) bool {
			return scope.Lookup(name.(string)).Exported()
		}).
		Select(func(name interface{}) interface{} {
			return scope.Lookup(name.(string))
		})

	for obj := range exportedObjects {
		switch obj := obj.(type) {
		case *types.Func:
			p.funcs[obj.FullName()] = obj
		case *types.TypeName:
			named := obj.Type().(*types.Named)
			switch typ := named.Underlying().(type) {
			case *types.Struct:
				p.structs[obj.Name()] = typ
			}
		}
	}

	return nil
}
