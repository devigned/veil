package golang

import (
	"go/types"
)

// NamedStruct is a helpful facade over types.Named which is intended to only contain a struct
type NamedStruct struct {
	named *types.Named
}

// Struct returns the underlying struct
func (n NamedStruct) Struct() *types.Struct {
	return n.named.Underlying().(*types.Struct)
}

// Methods returns the list of methods decorated on the struct
func (n NamedStruct) Methods() []*types.Func {
	var methods []*types.Func
	for i := 0; i < n.named.NumMethods(); i++ {
		meth := n.named.Method(i)
		methods = append(methods, meth)
	}
	return methods
}
