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
	"github.com/emirpasic/gods/sets/hashset"
	"github.com/marstr/collection"
	"go/ast"
	"strings"
)

// Package is a container for ast.Types and Docs
type Package struct {
	pkg              *types.Package
	doc              *doc.Package
	exportedAstables *hashset.Set
	packageAliases   map[string]string
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
		pkg:              typesPkg,
		doc:              docPkg,
		exportedAstables: hashset.New(),
		packageAliases:   map[string]string{},
	}

	if err = veilPkg.build(); err != nil {
		return nil, err
	}

	return veilPkg, nil
}

func (p Package) Funcs() []Func {
	values := p.exportedAstables.Values()
	output := []Func{}
	for _, item := range values {
		if cast, ok := item.(Func); ok {
			output = append(output, cast)
		}
	}
	return output
}

func (p Package) Structs() []*Struct {
	values := p.exportedAstables.Values()
	output := []*Struct{}
	for _, item := range values {
		if cast, ok := item.(*Struct); ok {
			output = append(output, cast)
		}
	}
	return output
}

func (p Package) ExportedTypes() []types.Type {
	values := p.exportedAstables.Values()
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
	shouldExport := hashset.New()
	addExport := func(item AstTransformer) {
		if !shouldExport.Contains(item.ExportName()) {
			shouldExport.Add(item.ExportName())
			p.exportedAstables.Add(item)
		}
	}

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
			funcWrapper := Func{obj}
			if funcWrapper.IsExportable() {
				p.exportedAstables.Add(funcWrapper)
				for _, astTransformable := range funcExportedTypes(obj) {
					addExport(astTransformable)
				}
			}
		case *types.TypeName:
			named := obj.Type().(*types.Named)
			switch named.Underlying().(type) {
			case *types.Struct:
				structWapper := NewStruct(named)
				addExport(structWapper)
				for _, v := range structWapper.Methods() {
					for _, astTransformable := range funcExportedTypes(v) {
						addExport(astTransformable)
					}
				}

				for i := 0; i < structWapper.Struct().NumFields(); i++ {
					field := structWapper.Struct().Field(i)
					if astTransformable, ok := shouldWrapField(field); ok {
						if slice, ok := field.Type().(*types.Slice); ok {
							if typ, ok := shouldWrapType(slice.Elem()); ok {
								addExport(typ)
							}
						}
						addExport(astTransformable)
					}
				}
			default:
				return core.NewSystemError("I don't know how to handle type names that aren't structs: ", obj)
			}
		}
	}

	for _, item := range p.exportedAstables.Values() {

		addObjAlias := func(typeName *types.TypeName) {
			path := typeName.Pkg().Path()
			alias := PkgPathAliasFromString(path)
			p.packageAliases[alias] = path
		}

		var addNamedOrPtr func(typ types.Type)
		addNamedOrPtr = func(typ types.Type) {
			if named, ok := typ.(*types.Named); ok {
				addObjAlias(named.Obj())
			}
			if ptr, ok := typ.(*types.Pointer); ok {
				addNamedOrPtr(ptr.Elem())
			}
		}

		t := item.(types.Type)
		underlying := t.Underlying()
		switch typ := underlying.(type) {
		case *types.Named:
			addObjAlias(typ.Obj())
		case Slice:
			addNamedOrPtr(typ.Elem())
		case Func:
			params := typ.Signature().Params()
			for i := 0; i < params.Len(); i++ {
				param := params.At(i)
				addNamedOrPtr(param.Type())
			}
			results := typ.Signature().Results()
			for i := 0; i < results.Len(); i++ {
				result := results.At(i)
				addNamedOrPtr(result.Type())
			}
		}
	}

	return nil
}

func (p Package) ImportAliases() map[string]string {
	return p.packageAliases
}

func (p Package) ToAst() []ast.Decl {
	decls := []ast.Decl{}

	for _, t := range p.exportedAstables.Values() {
		transformer := t.(AstTransformer)
		for _, d := range transformer.ToAst() {
			decls = append(decls, d)
		}
	}

	return decls
}

func (p Package) IsConstructor(f Func) bool {
	for _, s := range p.Structs() {
		if s.IsConstructor(f) {
			return true
		}
	}
	return false
}

func funcExportedTypes(fun *types.Func) []AstTransformer {
	typs := []AstTransformer{}
	sig := fun.Type().(*types.Signature)
	vars := []*types.Var{}

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

	for _, v := range vars {
		paramType := v.Type()
		if slice, ok := paramType.(*types.Slice); ok {
			if typ, ok := shouldWrapType(slice.Elem()); ok {
				typs = append(typs, typ)
			}
		}
		if typ, ok := shouldWrapType(paramType); ok {
			typs = append(typs, typ)
		}
	}
	return typs
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
		if !ImplementsError(u) && !strings.Contains(u.String(), "/vendor/") {
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
