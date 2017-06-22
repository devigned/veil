package helloworld

import (
	"fmt"
)

// Hello is a complex structure
type Hello struct {
	World World
	Foo   int
	Bar   string
	Buzz  []string
	Baz   float32
}

// World is a nested struct
type World struct {
	Something string
	blah      float64
}

func privateFunc(arg1 int) (int, error) {
	return 42, nil
}

// PublicUnbound returns the meaning of everything
func PublicUnbound(arg1 int) (int, error) {
	return 42, nil
}

// PublicUnboundError returns an error
func PublicUnboundError(arg1 int) (int, error) {
	return 0, fmt.Errorf("public unbound error")
}

// PublicBound returns the meaning of everything
func (h *Hello) PublicBound(arg1 int) (string, error) {
	return "42", nil
}

// NewHello constructs a new instance of Hello
func NewHello(world World, foo int, bar string, buzz []string, baz float32) *Hello {
	return &Hello{
		World: world,
		Foo:   foo,
		Bar:   bar,
		Buzz:  buzz,
		Baz:   baz,
	}
}
