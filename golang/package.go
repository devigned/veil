package golang

import (
	"fmt"
	"go/build"
	"go/doc"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path"

	"github.com/devigned/veil/core"
	"github.com/marstr/collection"
)

// Package is a container for ast.Types and Docs
type Package struct {
	pkg          *types.Package
	doc          *doc.Package
	funcs        map[string]*types.Func
	namedStructs map[string]*NamedStruct
}

// NewPackage constructs a Package from pkgPath using the specified working directory
func NewPackage(pkgPath string, workDir string) (*Package, error) {
	cmd := exec.Command("go", "install", pkgPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = workDir

	if err := cmd.Run(); err != nil {
		return nil, core.NewSystemErrorF("error installing [%s]: %v\n", pkgPath, err)
	}

	buildPkg, err := build.Import(pkgPath, workDir, 0)
	if err != nil {
		return nil, core.NewSystemErrorF("error resolving import path [%s]: %v\n", pkgPath, err)
	}

	typesPkg, err := importer.Default().Import(buildPkg.ImportPath)
	if err != nil {
		return nil, core.NewSystemErrorF("error importing package [%v]: %v\n", buildPkg.ImportPath, err)
	}

	fset := token.NewFileSet()
	astPkgs, err := parser.ParseDir(fset, buildPkg.Dir, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	astPkg, ok := astPkgs[typesPkg.Name()]
	if !ok {
		return nil, core.NewSystemErrorF("could not find AST for package %q", typesPkg.Name())
	}

	docPkg := doc.New(astPkg, buildPkg.ImportPath, 0)

	veilPkg := &Package{
		pkg:          typesPkg,
		doc:          docPkg,
		funcs:        make(map[string]*types.Func),
		namedStructs: make(map[string]*NamedStruct),
	}

	if err = veilPkg.build(); err != nil {
		return nil, err
	}

	return veilPkg, nil
}

func (p Package) Funcs() map[string]*types.Func {
	return p.funcs
}

func (p Package) Structs() map[string]*NamedStruct {
	return p.namedStructs
}

func (p Package) Name() string {
	return p.pkg.Name()
}

func (p *Package) build() error {

	scope := p.pkg.Scope()
	exportedObjects := collection.AsEnumerable(scope.Names()).Enumerate(nil).
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
			switch named.Underlying().(type) {
			case *types.Struct:
				pkgName := p.pkg.Name()
				pkgPath := p.pkg.Path()
				namedStruct := &NamedStruct{
					named: named,
				}
				p.namedStructs[path.Join(pkgPath, pkgName, obj.Name())] = namedStruct
			default:
				fmt.Println("default: ", obj)
			}
		}
	}

	return nil
}
