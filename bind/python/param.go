package python

import (
	"github.com/devigned/veil/core"
	"fmt"
	"go/types"
	"github.com/devigned/veil/cgo"
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
		class := p.binder.NewClass(&cgo.Struct{Named: t})
		return fmt.Sprintf(STRUCT_OUTPUT_TRANSFORM, class.Name(), varName)
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