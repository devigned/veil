package helloworld

import (
	"fmt"
)

var (
	magicNumber = 0
)

// Hello is a complex structure
type Hello struct {
	World  World
	Foo    int
	Bar    string
	Buzz   []string
	Baz    float32
	secret notExported
}

// World is a nested struct
type World struct {
	Something string
	blah      float64
}

// notExported is a struct field not to be exported
type notExported struct {
	something string
}

func init() {
	magicNumber = 42
}

func privateFunc() (int, error) {
	return magicNumber, nil
}

// GetMagicNumber returns 42 when properly initialized
func GetMagicNumber() int {
	magicNum, _ := privateFunc()
	return magicNum
}

// PublicUnbound returns the meaning of everything
func PublicUnbound(arg1 int) int {
	return arg1
}

// PublicUnboundError returns an error
func PublicUnboundError(arg1 int) (int, error) {
	return 0, fmt.Errorf("public unbound error given: %s", arg1)
}

// PublicBound returns the meaning of everything
func (h *Hello) PublicBound(arg1 int) (string, error) {
	return h.Bar, nil
}

// NewHello constructs a new instance of Hello
func NewHelloPtr(world World, foo int, bar string, buzz []string, baz float32) *Hello {
	return &Hello{
		World:  world,
		Foo:    foo,
		Bar:    bar,
		Buzz:   buzz,
		Baz:    baz,
		secret: notExported{},
	}
}

// NewHello constructs a new instance of Hello
func NewHello(world World, foo int, bar string, buzz []string, baz float32) Hello {
	return Hello{
		World:  world,
		Foo:    foo,
		Bar:    bar,
		Buzz:   buzz,
		Baz:    baz,
		secret: notExported{},
	}
}
