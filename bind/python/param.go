package python

import (
	"fmt"
	"github.com/devigned/veil/cgo"
	"github.com/devigned/veil/core"
	"go/types"
)

const (
	STRING_OUTPUT_TRANSFORM = "_CffiHelper.c2py_string(%s)"
	STRING_INPUT_TRANSFORM  = "%s = _CffiHelper.py2c_string(%s)"
	STRUCT_INPUT_TRANSFORM  = "%s = _CffiHelper.py2c_veil_object(%s)"
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
	return p.returnFormat(p.underlying.Type(), varName)
}

func (p Param) returnFormat(typ types.Type, varName string) string {
	switch t := typ.(type) {
	case *types.Basic:
		if t.Kind() == types.String {
			return fmt.Sprintf(STRING_OUTPUT_TRANSFORM, varName)
		}
		return varName
	case *types.Named:
		if cgo.ImplementsError(t) {
			return fmt.Sprintf(STRUCT_OUTPUT_TRANSFORM, "VeilError", varName)
		} else if _, ok := t.Underlying().(*types.Struct); ok {
			class := p.binder.NewClass(cgo.NewStruct(t))
			return fmt.Sprintf(STRUCT_OUTPUT_TRANSFORM, class.Name(), varName)
		} else {
			return varName
		}
	case *types.Pointer:
		return p.returnFormat(t.Elem(), varName)
	default:
		return varName
	}
}

func InputFormat(varName string, typ types.Type) string {
	switch t := typ.(type) {
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

func (p Param) InputFormat() string {
	return InputFormat(p.Name(), p.underlying.Type())
}

func (p Param) InputFormatForName(name string) string {
	return InputFormat(name, p.underlying.Type())
}
