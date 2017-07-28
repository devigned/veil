package main

import (
	"C"
)
import (
	"fmt"
	"sync"
	"unsafe"
)
import (
	github_com_devigned_veil__examples_helloworld "github.com/devigned/veil/_examples/helloworld"
)

var refs struct {
	sync.Mutex
	next	int32
	refs	map[unsafe.Pointer]int32
	ptrs	map[int32]cobject
}

type cobject struct {
	ptr	unsafe.Pointer
	cnt	int32
}
//export cgo_decref
func cgo_decref(ptr unsafe.Pointer) {
	refs.Lock()
	num, ok := refs.refs[ptr]
	if !ok {
		panic("decref untracked object!")
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
		if refs.next < 0 {
			panic("refs.next underflow")
		}
		refs.refs[ptr] = num
		refs.ptrs[num] = cobject{ptr, 1}
	}
	refs.Unlock()
}
//export github_com_devigned_veil__examples_helloworld_GetMagicNumber
func github_com_devigned_veil__examples_helloworld_GetMagicNumber() ( int) {
	return github_com_devigned_veil__examples_helloworld.GetMagicNumber()
}
//export slice_of_string_new
func slice_of_string_new() []string {
	var o []string
	cgo_incref(unsafe.Pointer(&o))
	return ([]string)(unsafe.Pointer(&o))
}
//export slice_of_string_str
func slice_of_string_str(self []string) string {
	return fmt.Sprintf("%#v", *(*[]string)(unsafe.Pointer(self)))
}
//export slice_of_string_item
func slice_of_string_item(self []string, i int) string {
	items := (*[]string)(unsafe.Pointer(self))
	return (*items)[i]
}
//export slice_of_string_item_set
func slice_of_string_item_set(self []string, i int, item string) {
	items := (*[]string)(unsafe.Pointer(self))
	(*items)[i] = item
}
//export slice_of_string_item_append
func slice_of_string_item_append(self []string, item string) {
	items := (*[]string)(unsafe.Pointer(self))
	*items = append(*items, item)
}
//export slice_of_string_destroy
func slice_of_string_destroy(self []string) {
	cgo_decref(unsafe.Pointer(&self))
}
//export github_com_devigned_veil__examples_helloworld_NewHello
func github_com_devigned_veil__examples_helloworld_NewHello(world github_com_devigned_veil__examples_helloworld.World, foo int, bar string, buzz []string, baz float32) ( github_com_devigned_veil__examples_helloworld.Hello) {
	return github_com_devigned_veil__examples_helloworld.NewHello(world, foo, bar, buzz, baz)
}
//export github_com_devigned_veil__examples_helloworld_NewHelloPtr
func github_com_devigned_veil__examples_helloworld_NewHelloPtr(world github_com_devigned_veil__examples_helloworld.World, foo int, bar string, buzz []string, baz float32) ( github_com_devigned_veil__examples_helloworld.Hello) {
	return github_com_devigned_veil__examples_helloworld.NewHelloPtr(world, foo, bar, buzz, baz)
}
//export github_com_devigned_veil__examples_helloworld_PublicUnbound
func github_com_devigned_veil__examples_helloworld_PublicUnbound(arg1 int) ( int) {
	return github_com_devigned_veil__examples_helloworld.PublicUnbound(arg1)
}
//export github_com_devigned_veil__examples_helloworld_PublicUnboundError
func github_com_devigned_veil__examples_helloworld_PublicUnboundError(arg1 int) ( int,  error) {
	return github_com_devigned_veil__examples_helloworld.PublicUnboundError(arg1)
}
func main() {
}