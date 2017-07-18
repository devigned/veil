package bind

import (
	"fmt"

	"github.com/devigned/veil/cgo"
)

// Py3Binder contains the data for generating a python 3 binding
type Py3Binder struct {
	pkg *cgo.Package
}

// NewPy3Binder creates a new Binder for Python 3
func NewPy3Binder(pkg *cgo.Package) Bindable {
	return &Py3Binder{
		pkg: pkg,
	}
}

// Bind is the Python 3 implementation of Bind
func (p Py3Binder) Bind(outDir string) error {
	fmt.Println("doing some python 3 binding")
	return nil
}
