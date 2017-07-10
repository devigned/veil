package cgo

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewIdent(t *testing.T) {
	subject := NewIdent("something")
	assert.Equal(t, "something", subject.Name)
}
