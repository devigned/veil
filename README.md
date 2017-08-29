# Veil
Veil is a code generator, which exposes Golang packages via a generated C ABI that is consumed by 
host languages through FFI. Currently, Veil supports Python, but could be used with any FFI 
implementation.

Veil produces a C ABI from a Golang package by parsing the AST exposed by the Golang package 
and building a CGo wrapper AST for the exposed functionality in the package. The CGo
wrapper AST is written to `main.go`, compiled, and used as the basis for FFI binding from the 
consuming language.

Veil has been built to comply with the Golang specification for Go code consumed via a C bridge 
(see: https://golang.org/cmd/cgo/#hdr-Passing_pointers). The Veil bridge provides a 16 byte UUID as an
exposed pointer to the consuming language, which maps to the Golang pointer on the Go side of the
C bridge. No Golang pointers are shared across the Go to C bridge.

**This is a work in progress. Please don't use this if you expect stability.**

## Other Languages
Adding support for other languages is a future goal of the project. If you are interested, visit
[./bind/python](./bind/python) and take a look at what the Python the binder does. 
Pull requests are welcome.

## Running Veil
- `make`
- `./bin/github.com/devigned/veil generate -p github.com/devigned/veil/_examples/helloworld`
- `cd ./output`
- run some python...
```python
import generated
print(generated.get_magic_number())
```

## License
MIT License

Copyright (c) 2017 David Justice

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
