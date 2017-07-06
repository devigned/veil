package main

//#include <stdlib.h>
//#include <string.h>
//#include <complex.h>
import "C"

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

var _ = unsafe.Pointer(nil)
var _ = fmt.Sprintf

// --- begin cgo helpers ---

//export _cgo_GoString
func _cgo_GoString(str *C.char) string {
	return C.GoString(str)
}

//export _cgo_CString
func _cgo_CString(s string) *C.char {
	return C.CString(s)
}

//export _cgo_ErrorIsNil
func _cgo_ErrorIsNil(err error) bool {
	return err == nil
}

//export _cgo_ErrorString
func _cgo_ErrorString(err error) *C.char {
	return C.CString(err.Error())
}

//export _cgo_FreeCString
func _cgo_FreeCString(cs *C.char) {
	C.free(unsafe.Pointer(cs))
}

// --- end cgo helpers ---

// --- begin cref helpers ---

type cobject struct {
	ptr unsafe.Pointer
	cnt int32
}

// refs stores Go objects that have been passed to another language.
var refs struct {
	sync.Mutex
	next int32 // next reference number to use for Go object, always negative
	refs map[unsafe.Pointer]int32
	ptrs map[int32]cobject
}

//export cgo_incref
func cgo_incref(ptr unsafe.Pointer) {
	refs.Lock()
	num, ok := refs.refs[ptr]
	if ok {
		s := refs.ptrs[num]
		refs.ptrs[num] = cobject{s.ptr, s.cnt + 1}
	} else {
		num = refs.next
		refs.next--
		if refs.next > 0 {
			panic("refs.next underflow")
		}
		refs.refs[ptr] = num
		refs.ptrs[num] = cobject{ptr, 1}
	}
	refs.Unlock()
}

//export cgo_decref
func cgo_decref(ptr unsafe.Pointer) {
	refs.Lock()
	num, ok := refs.refs[ptr]
	if !ok {
		panic("cgopy: decref untracked object")
	}
	s := refs.ptrs[num]
	if s.cnt-1 <= 0 {
		delete(refs.ptrs, num)
		delete(refs.refs, ptr)
		refs.Unlock()
		return
	}
	refs.ptrs[num] = cobject{s.ptr, s.cnt - 1}
	refs.Unlock()
}

func cgoCheckGoVersion() {
	godebug := os.Getenv("GODEBUG")
	cgocheck := -1
	var err error
	if godebug != "" {
		const prefix = "cgocheck="
		for _, option := range strings.Split(godebug, ",") {
			if !strings.HasPrefix(option, prefix) {
				continue
			}
			cgocheck, err = strconv.Atoi(option[len(prefix):])
			if err != nil {
				cgocheck = -1
				fmt.Fprintf(os.Stderr, "gopy: invalid cgocheck value %q (expected an integer)\n", option)
			}
		}
	}

	if cgocheck != 0 {
		fmt.Fprintf(os.Stderr, "gopy: GODEBUG=cgocheck=0 should be set for Go>=1.6\n")
	}
}

func init() {
	cgoCheckGoVersion()
	refs.Lock()
	refs.next = -24 // Go objects get negative reference numbers. Arbitrary starting point.
	refs.refs = make(map[unsafe.Pointer]int32)
	refs.ptrs = make(map[int32]cobject)
	refs.Unlock()

	// make sure cgo is used and cgo hooks are run
	str := C.CString("foo")
	C.free(unsafe.Pointer(str))
}

// --- end cref helpers ---

func main() {}
