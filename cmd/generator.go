package cmd

import (
	"os"
	"path/filepath"

	"github.com/devigned/veil/bind"
	"github.com/devigned/veil/cgo"
	"github.com/devigned/veil/core"
)

// Generator generates libraries in other languages by creating bindings in those languages
// to a Golang project
type Generator struct {
	PkgPath string
	OutDir  string
	Targets []string
	LibName string
}

// NewGenerator constructs a new Generator instance
func NewGenerator(pkgPath string, outDir string, libName string, targets []string) *Generator {
	return &Generator{
		PkgPath: pkgPath,
		OutDir:  outDir,
		Targets: targets,
		LibName: libName,
	}
}

func createOutputDir(outDir string) (string, error) {
	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		return "", core.NewSystemError("Could not create output directory: %v", err)
	}

	outDir, err = filepath.Abs(outDir)
	if err != nil {
		return "", core.NewSystemError("Could not infer absolute path to output directory: %v", err)
	}

	return outDir, nil
}

// Execute builds the target projects
func (g Generator) Execute() error {
	outDir, err := createOutputDir(g.OutDir)
	if err != nil {
		return err
	}

	pkg, err := cgo.NewPackage(g.PkgPath, outDir)
	if err != nil {
		return err
	}

	for _, target := range g.Targets {
		binder, err := bind.NewBinder(pkg, target)
		if err != nil {
			return err
		}
		binder.Bind(outDir, g.LibName)
	}

	return nil
}
