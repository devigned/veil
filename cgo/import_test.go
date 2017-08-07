package cgo

import (
	"github.com/stretchr/testify/assert"
	"go/ast"
	"testing"
)

func TestSingleImport(t *testing.T) {
	subject := Imports("C")
	assert.NotNil(t, subject)
	assert.Equal(t, 1, len(subject.Specs))
	importSpec := subject.Specs[0].(*ast.ImportSpec)
	assert.Equal(t, "\"C\"", importSpec.Path.Value)
}

func TestMultipleImports(t *testing.T) {
	subject := Imports("foo", "bar", "baz")
	assert.NotNil(t, subject)
	assert.Equal(t, 3, len(subject.Specs))
}
