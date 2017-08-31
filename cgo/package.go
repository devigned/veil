package cgo

import (
	"go/build"
	"go/doc"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"

	"github.com/devigned/veil/core"
	"github.com/emirpasic/gods/maps"
	"github.com/emirpasic/gods/maps/treemap"
	"github.com/emirpasic/gods/sets/treeset"
	"github.com/marstr/collection"
	"go/ast"
	"strings"
)

// Package is a container for ast.Types and Docs
type Package struct {
	pkg            *types.Package
	doc            *doc.Package
	symbols        *treemap.Map
	packageAliases *treemap.Map
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
		pkg:            typesPkg,
		doc:            docPkg,
		packageAliases: treemap.NewWithStringComparator(),
		symbols:        treemap.NewWithStringComparator(),
	}

	if err = veilPkg.build(); err != nil {
		return nil, err
	}

	return veilPkg, nil
}

func (p Package) AstTransformers() []AstTransformer {
	v := make([]AstTransformer, p.symbols.Size())
	for idx, item := range p.symbols.Values() {
		v[idx] = item.(AstTransformer)
	}
	return v
}

func (p Package) Funcs() []*Func {
	keysValues := p.symbols.Select(func(key, value interface{}) bool {
		_, ok := value.(*Func)
		return ok
	})
	v := make([]*Func, keysValues.Size())
	for idx, item := range keysValues.Values() {
		v[idx] = item.(*Func)
	}
	return v
}

func (p Package) Structs() []*Struct {
	keysValues := p.symbols.Select(func(key, value interface{}) bool {
		_, ok := value.(*Struct)
		return ok
	})
	v := make([]*Struct, keysValues.Size())
	for idx, item := range keysValues.Values() {
		v[idx] = item.(*Struct)
	}
	return v
}

func (p Package) Interfaces() []*Interface {
	keysValues := p.symbols.Select(func(key, value interface{}) bool {
		_, ok := value.(*Interface)
		return ok
	})
	v := make([]*Interface, keysValues.Size())
	for idx, item := range keysValues.Values() {
		v[idx] = item.(*Interface)
	}
	return v
}

func (p Package) ExportedTypes() []types.Type {
	values := p.AstTransformers()
	output := make([]types.Type, len(values))
	for i, item := range values {
		output[i] = item.Underlying()
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
		if err := p.addExportedObject(obj); err != nil {
			return err
		}
	}

	for _, aster := range p.AstTransformers() {
		if item, ok := aster.(Packaged); ok {
			path := item.PackagePath()
			p.packageAliases.Put(PkgPathAliasFromString(path), path)
		}
	}

	return nil
}

func (p Package) addExportedObject(obj interface{}) error {
	addExport := func(item AstTransformer) bool {
		exportName := item.ExportName()
		if _, ok := p.symbols.Get(exportName); ok || !item.IsExportable() {
			// already registered this symbol
			return false
		} else {
			p.symbols.Put(exportName, item)
			return true
		}
	}

	handleNamed := func(named *types.Named) error {
		switch named.Underlying().(type) {
		case *types.Struct:
			structWapper := NewStruct(named)
			if addExport(structWapper) {
				for _, method := range structWapper.ExportedMethods() {
					for _, v := range allVars(method) {
						if err := p.addExportedObject(v.Type()); err != nil {
							return err
						}
					}
				}

				for i := 0; i < structWapper.Struct().NumFields(); i++ {
					field := structWapper.Struct().Field(i)
					if field.Exported() {
						if err := p.addExportedObject(field.Type()); err != nil {
							return err
						}
					}
				}
			}
		case *types.Map:
			// Todo: handle maps
		case *types.Basic:
			// Todo: should this be handled differently?
		case *types.Interface:
			if !ImplementsError(named) {
				addExport(NewInterface(named))
			}
		case *types.Slice:
			addExport(NewNamed(named))
		default:
			return core.NewSystemError("I don't know how to handle named types like: ", obj)
		}
		return nil
	}

	switch t := obj.(type) {
	case *types.Func:
		funcWrapper := NewFunc(t)
		if addExport(funcWrapper) {
			for _, v := range allVars(funcWrapper) {
				if err := p.addExportedObject(v.Type()); err != nil {
					return err
				}
			}
		}
	case *types.Slice:
		addExport(NewSlice(t.Elem()))
		if err := p.addExportedObject(t.Elem()); err != nil {
			return err
		}
	case *types.TypeName:
		named := t.Type().(*types.Named)
		if t.Exported() {
			if err := handleNamed(named); err != nil {
				return err
			}
		}
	case *types.Named:
		if err := handleNamed(t); err != nil {
			return err
		}
	case *types.Pointer:
		if err := p.addExportedObject(t.Elem()); err != nil {
			return err
		}
	}
	return nil
}

func (p Package) ImportAliases() maps.Map {
	return p.packageAliases
}

func (p Package) ToAst() []ast.Decl {
	decls := []ast.Decl{}

	for _, t := range p.AstTransformers() {
		for _, d := range t.ToAst() {
			decls = append(decls, d)
		}
	}

	return decls
}

func (p Package) CDefinitions() []string {
	retTypes := []string{}
	funcPtrs := []string{}
	calls := []string{}
	for _, iface := range p.Interfaces() {
		r, f, c := iface.CDefs()
		retTypes = append(retTypes, r...)
		funcPtrs = append(funcPtrs, f...)
		calls = append(calls, c...)
	}
	retTypes = uniqStrings(retTypes...)
	funcPtrs = uniqStrings(funcPtrs...)
	calls = uniqStrings(calls...)
	return append(append(retTypes, funcPtrs...), calls...)
}

func uniqStrings(items ...string) []string {
	set := treeset.NewWithStringComparator()

	for _, item := range items {
		set.Add(item)
	}

	slice := make([]string, set.Size())
	for idx, def := range set.Values() {
		slice[idx] = def.(string)
	}

	return slice
}

func (p Package) IsConstructor(f *Func) bool {
	for _, s := range p.Structs() {
		if s.IsConstructor(f) {
			return true
		}
	}
	return false
}

func allVars(fun *Func) []*types.Var {
	vars := []*types.Var{}
	sig := fun.Type().(*types.Signature)
	params := sig.Params()
	for i := 0; i < params.Len(); i++ {
		param := params.At(i)
		vars = append(vars, param)
	}

	results := sig.Results()
	for i := 0; i < results.Len(); i++ {
		param := results.At(i)
		vars = append(vars, param)
	}

	return vars
}

func shouldWrapType(t types.Type) (AstTransformer, bool) {
	switch u := t.(type) {
	case *types.Basic:
		return nil, false
	case *types.Pointer:
		return shouldWrapType(u.Elem())
	case *types.Slice:
		return NewSlice(u.Elem()), true
	case *types.Named:
		_, isStruct := u.Underlying().(*types.Struct)
		if !ImplementsError(u) && isStruct && !strings.Contains(u.String(), "/vendor/") {
			return NewStruct(u), true
		} else {
			return nil, false
		}
	//case *types.Array:
	//	return NewArray(u.Elem(), u.Len()), true
	default:
		return nil, false
	}
}

func shouldWrapField(f *types.Var) (AstTransformer, bool) {
	if f.Exported() {
		return shouldWrapType(f.Type())
	}
	return nil, false
}
