package cgo

import (
	"go/types"
)

// Inerface is a helpful facade over types.Named which is intended to only contain an Interface
type Interface struct {
	*Named
}

func NewInterface(named *types.Named) *Interface {
	if _, ok := named.Underlying().(*types.Interface); !ok {
		panic("only interfaces belong in Interface")
	}
	return &Interface{NewNamed(named)}
}
