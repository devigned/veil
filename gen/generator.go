package gen

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/devigned/veil/core"
)

// Generator generates libraries in other languages by creating bindings in those languages
// to a Golang project
type Generator struct {
	PkgPath string
	OutDir  string
	Targets []string
}

// NewGenerator constructs a new Generator instance
func NewGenerator(pkgPath string, outDir string, targets []string) *Generator {
	return &Generator{
		PkgPath: pkgPath,
		OutDir:  outDir,
		Targets: targets,
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

func (g Generator) bindingGen() error {
	return nil
}

// Execute builds the target projects
func (g Generator) Execute() error {
	outDir, err := createOutputDir(g.OutDir)
	if err != nil {
		return err
	}

	pkg, err := core.NewPackage(g.PkgPath, outDir)
	if err != nil {
		return err
	}

	for key := range pkg.GetFuncs() {
		fmt.Println("func: ", key)
	}

	for key := range pkg.GetStruct() {
		fmt.Println("struct: ", key)
	}

	return nil
}
