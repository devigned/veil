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

	"github.com/devigned/veil/core"
	"github.com/devigned/veil/golang/cgo"
	"github.com/emirpasic/gods/sets/hashset"
	"github.com/marstr/collection"
	"go/ast"
)

// Package is a container for ast.Types and Docs
type Package struct {
	pkg              *types.Package
	doc              *doc.Package
	funcs            *hashset.Set
	namedStructs     *hashset.Set
	exportedAstables *hashset.Set
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
		funcs:            hashset.New(),
		namedStructs:     hashset.New(),
		exportedAstables: hashset.New(),
	}

	if err = veilPkg.build(); err != nil {
		return nil, err
	}

	return veilPkg, nil
}

func (p Package) Funcs() []types.Func {
	values := p.funcs.Values()
	output := make([]types.Func, len(values))
	for i, item := range values {
		output[i] = item.(types.Func)
	}
	return output
}

func (p Package) Structs() []cgo.StructWrapper {
	values := p.namedStructs.Values()
	output := make([]cgo.StructWrapper, len(values))
	for i, item := range values {
		output[i] = item.(cgo.StructWrapper)
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
			funcWrapper := cgo.FuncWrapper{obj}
			p.funcs.Add(funcWrapper)
			p.exportedAstables.Add(funcWrapper)
			for _, astTransformable := range funcExportedTypes(obj) {
				p.exportedAstables.Add(astTransformable)
			}
		case *types.TypeName:
			named := obj.Type().(*types.Named)
			switch named.Underlying().(type) {
			case *types.Struct:
				structWapper := &cgo.StructWrapper{named}
				p.namedStructs.Add(structWapper)
				p.exportedAstables.Add(structWapper)
				for _, v := range structWapper.Methods() {
					for _, astTransformable := range funcExportedTypes(v) {
						p.exportedAstables.Add(astTransformable)
					}
				}

				for i := 0; i < structWapper.Struct().NumFields(); i++ {
					field := structWapper.Struct().Field(i)
					if astTransformable, ok := shouldWrapField(field); ok {
						p.exportedAstables.Add(astTransformable)
					}
				}
			default:
				return core.NewSystemError("I don't know how to handle type names that arn't structs: ", obj)
			}
		}
	}

	return nil
}

func (pkg Package) ToCgoAst() []ast.Decl {
	decls := []ast.Decl{}

	for _, t := range pkg.exportedAstables.Values() {
		transformer := t.(cgo.AstTransformer)
		for _, d := range transformer.ToCgoAst() {
			decls = append(decls, d)
		}
	}

	return decls
}

func funcExportedTypes(fun *types.Func) []cgo.AstTransformer {
	typs := []cgo.AstTransformer{}
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

func shouldWrapType(t types.Type) (cgo.AstTransformer, bool) {
	underlying := t.Underlying()
	switch u := underlying.(type) {
	case *types.Basic:
		return nil, false
	case *types.Pointer:
		return shouldWrapType(u.Elem())
	case *types.Slice:
		return cgo.NewSliceWrapper(u.Elem()), true
	case *types.Array:
		return cgo.NewArrayWrapper(u.Elem(), u.Len()), true
	default:
		return nil, false
	}
}

func shouldWrapField(f *types.Var) (cgo.AstTransformer, bool) {
	if f.Exported() {
		return shouldWrapType(f.Type())
	}
	return nil, false
}
