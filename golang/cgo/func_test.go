package cgo

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMainFunc(t *testing.T) {
	subject := MainFunc()
	assert.Equal(t, subject.Name.Name, "main")
}
