package bind

import (
	"fmt"
	"io"

	"github.com/devigned/veil/golang"
)

// Binder is the interface for any object that will create a binding for a golang.Package
type Binder interface {
	Bind(writer io.Writer) error
}

// NewBinder is a factory method for creating a new binder for a given target
func NewBinder(pkg *golang.Package, target string) Binder {
	switch target {
	case "py3":
		return NewPy3Binder(pkg)
	default:
		panic(fmt.Sprintf("I don't know how to create a binder for %s", target))
	}
}
