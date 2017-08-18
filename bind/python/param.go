package python

import (
	"fmt"
	"github.com/devigned/veil/cgo"
	"github.com/devigned/veil/core"
	"go/types"
)

const (
	STRING_OUTPUT_TRANSFORM = "_CffiHelper.c2py_string(%s)"
	STRING_INPUT_TRANSFORM  = "%s = ffi.new(\"char[]\", %s.encode(\"utf-8\"))"
	STRUCT_INPUT_TRANSFORM  = "%s = %s.uuid_ptr()"
	STRUCT_OUTPUT_TRANSFORM = "%s(%s)"
)

type Param struct {
	underlying *types.Var
	binder     *Binder
}

func (p Param) Name() string {
	return core.ToSnake(p.underlying.Name())
}

func (p Param) IsError() bool {
	return cgo.ImplementsError(p.underlying.Type())
}

func (p Param) ReturnFormat(varName string) string {
	switch t := p.underlying.Type().(type) {
	case *types.Basic:
		if t.Kind() == types.String {
			return fmt.Sprintf(STRING_OUTPUT_TRANSFORM, varName)
		}
		return varName
	case *types.Named:
		if cgo.ImplementsError(t) {
			return fmt.Sprintf(STRUCT_OUTPUT_TRANSFORM, "VeilError", varName)
		} else {
			class := p.binder.NewClass(&cgo.Struct{Named: t})
			return fmt.Sprintf(STRUCT_OUTPUT_TRANSFORM, class.Name(), varName)
		}
	case *types.Pointer:
		if named, ok := t.Elem().(*types.Named); ok {
			class := p.binder.NewClass(&cgo.Struct{Named: named})
			return fmt.Sprintf(STRUCT_OUTPUT_TRANSFORM, class.Name(), varName)
		}
		return varName
	default:
		return varName
	}
}

func (p Param) InputFormat(varName string) string {
	switch t := p.underlying.Type().(type) {
	case *types.Basic:
		if t.Kind() == types.String {
			return fmt.Sprintf(STRING_INPUT_TRANSFORM, varName, varName)
		}
	case *types.Named:
		return fmt.Sprintf(STRUCT_INPUT_TRANSFORM, varName, varName)
	case *types.Pointer:
		if _, ok := t.Elem().(*types.Named); ok {
			return fmt.Sprintf(STRUCT_INPUT_TRANSFORM, varName, varName)
		}
	}
	return ""
}
