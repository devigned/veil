package python

import (
	"fmt"
	"github.com/devigned/veil/cgo"
	"github.com/devigned/veil/core"
	"go/types"
	"strconv"
)

const (
	STRING_OUTPUT_TRANSFORM = "_CffiHelper.c2py_string(%s)"
	STRING_INPUT_TRANSFORM  = "%s = _CffiHelper.py2c_string(%s)"
	STRUCT_INPUT_TRANSFORM  = "%s = _CffiHelper.py2c_veil_object(%s)"
	STRUCT_OUTPUT_TRANSFORM = "%s(uuid_ptr=%s, tracked=%s)"
)

type Param struct {
	underlying  *types.Var
	binder      *Binder
	DefaultName string
}

func (p Param) Name() string {
	name := p.DefaultName
	if p.underlying.Name() != "" {
		name = p.underlying.Name()
	}
	return core.ToSnake(name)
}

func (p Param) IsError() bool {
	return cgo.ImplementsError(p.underlying.Type())
}

func (p Param) ReturnFormatWithName(varName string) string {
	return p.ReturnFormatWithNameAndTracked(varName, true)
}

func (p Param) ReturnFormatWithNameAndTracked(varName string, tracked bool) string {
	return p.returnFormatWithTypeAndNameAndTracked(p.underlying.Type(), varName, tracked)
}

func (p Param) returnFormatWithTypeAndNameAndTracked(typ types.Type, varName string, tracked bool) string {
	trackedBoolStr := core.ToCap(strconv.FormatBool(tracked))
	switch t := typ.(type) {
	case *types.Basic:
		if t.Kind() == types.String {
			return fmt.Sprintf(STRING_OUTPUT_TRANSFORM, varName)
		}
		return varName
	case *types.Named:
		if cgo.ImplementsError(t) {
			return fmt.Sprintf(STRUCT_OUTPUT_TRANSFORM, "VeilError", varName, trackedBoolStr)
		} else if _, ok := t.Underlying().(*types.Struct); ok {
			class := p.binder.NewClass(cgo.NewStruct(t))
			return fmt.Sprintf(STRUCT_OUTPUT_TRANSFORM, class.Name(), varName, trackedBoolStr)
		} else {
			return varName
		}
	case *types.Slice:
		slice := p.binder.NewList(cgo.NewSlice(t.Elem()))
		return fmt.Sprintf(STRUCT_OUTPUT_TRANSFORM, slice.ListTypeName(), varName, trackedBoolStr)
	case *types.Pointer:
		return p.returnFormatWithTypeAndNameAndTracked(t.Elem(), varName, tracked)
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
	case *types.Named, *types.Slice, *types.Interface:
		return fmt.Sprintf(STRUCT_INPUT_TRANSFORM, varName, varName)
	case *types.Pointer:
		if _, ok := t.Elem().(*types.Named); ok {
			return fmt.Sprintf(STRUCT_INPUT_TRANSFORM, varName, varName)
		}
	}
	return ""
}

func (p Param) ReturnFormatUntracked() string {
	return p.ReturnFormatWithNameAndTracked(p.Name(), false)
}

func (p Param) ReturnFormat() string {
	return p.ReturnFormatWithName(p.Name())
}

func (p Param) InputFormat() string {
	return InputFormat(p.Name(), p.underlying.Type())
}

func (p Param) InputFormatWithName(name string) string {
	return InputFormat(name, p.underlying.Type())
}
