package gen

import "github.com/devigned/veil/core"

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

func fetchProject(pkgPath string) (string, error) {

	return "", nil
}

func (*Generator) bindingGen() error {
	return nil
}

// Execute builds the target projects
func (*Generator) Execute() error {
	return core.NewSystemError("this stuff is totally broken")
}
