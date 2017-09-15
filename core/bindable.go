package core

// Bindable is the interface for any object that will create a binding for a golang.Package
type Binder interface {
	Bind(outDir, libName string) error
}
