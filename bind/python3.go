package bind

import (
	"fmt"
	"io"

	"github.com/devigned/veil/golang"
)

// Py3Binder contains the data for generating a python 3 binding
type Py3Binder struct {
	pkg *golang.Package
}

// NewPy3Binder creates a new Binder for Python 3
func NewPy3Binder(pkg *golang.Package) Binder {
	return &Py3Binder{
		pkg: pkg,
	}
}

// Bind is the Python 3 implementation of Bind
func (p Py3Binder) Bind(w io.Writer) error {
	fmt.Println("doing some python 3 binding")
	return nil
}
