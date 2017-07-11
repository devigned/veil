package main

import "C"

import (
	"github.com/devigned/veil/_examples/helloworld"
)

//export GetMagicNumber
func GetMagicNumber() int {
	return helloworld.GetMagicNumber()
}

func main() {
}
