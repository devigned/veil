package golang

import (
	"go/build"
	"go/doc"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path"

	"github.com/devigned/veil/bind/cgo"
	"github.com/devigned/veil/core"
	"github.com/emirpasic/gods/sets/hashset"
	"github.com/marstr/collection"
)

// Package is a container for ast.Types and Docs
type Package struct {
	pkg           *types.Package
	doc           *doc.Package
	funcs         map[string]*types.Func
	namedStructs  map[string]*NamedStruct
	exportedTypes *hashset.Set
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
		pkg:           typesPkg,
		doc:           docPkg,
		funcs:         make(map[string]*types.Func),
		namedStructs:  make(map[string]*NamedStruct),
		exportedTypes: hashset.New(),
	}

	if err = veilPkg.build(); err != nil {
		return nil, err
	}

	return veilPkg, nil
}

func (p Package) FuncsByName() map[string]*types.Func {
	return p.funcs
}

func (p Package) StructsByName() map[string]*NamedStruct {
	return p.namedStructs
}

func (p Package) ExportedTypes() []types.Type {
	values := p.exportedTypes.Values()
	output := make([]types.Type, len(values))
	for i, item := range values {
		output[i] = item.(types.Type)
	}
	return output
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
			for _, t := range funcExportedTypes(obj) {
				p.exportedTypes.Add(t)
			}
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
				for _, v := range namedStruct.Methods() {
					for _, t := range funcExportedTypes(v) {
						p.exportedTypes.Add(t)
					}
				}

				for i := 0; i < namedStruct.Struct().NumFields(); i++ {
					field := namedStruct.Struct().Field(i)
					if typ, ok := shouldWrapField(field); ok {
						p.exportedTypes.Add(typ)
					}
				}
			default:
				return core.NewSystemError("I don't know how to handle type names that arn't structs: ", obj)
			}
		}
	}

	return nil
}

func funcExportedTypes(fun *types.Func) []types.Type {
	typs := []types.Type{}
	sig := fun.Type().(*types.Signature)
	params := sig.Params()
	for i := 0; i < params.Len(); i++ {
		param := params.At(i)
		paramType := param.Type()
		if typ, ok := shouldWrapType(paramType); ok {
			typs = append(typs, typ)
		}
	}

	results := sig.Results()
	for i := 0; i < results.Len(); i++ {
		param := results.At(i)
		paramType := param.Type()
		if typ, ok := shouldWrapType(paramType); ok {
			typs = append(typs, typ)
		}
	}
	return typs
}

func shouldWrapType(t types.Type) (types.Type, bool) {
	underlying := t.Underlying()
	switch u := underlying.(type) {
	case *types.Basic:
		return t, false
	case *types.Pointer:
		return u.Elem(), true
	case *types.Slice:
		return cgo.NewSliceWrapper(u.Elem()), true
	case *types.Array:
		return cgo.NewArrayWrapper(u.Elem(), u.Len()), true
	default:
		return t, true
	}
}

func shouldWrapField(f *types.Var) (types.Type, bool) {
	if f.Exported() {
		return shouldWrapType(f.Type())
	}
	return nil, false
}
