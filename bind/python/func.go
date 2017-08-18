package python

import (
	"fmt"
	"github.com/devigned/veil/cgo"
	"strings"
)

type PyFunc struct {
	fun     cgo.Func
	Name    string
	Params  []*Param
	Results []*Param
}

func (f PyFunc) InputTransforms() []string {
	inputTranforms := []string{}
	for _, param := range f.Params {
		varName := param.Name()
		if format := param.InputFormat(varName); format != "" {
			inputTranforms = append(inputTranforms, format)
		}
	}
	return inputTranforms
}

func (f PyFunc) Call() string {
	return f.fun.CGoName() + "(" + f.PrintArgs() + ")"
}

func (f PyFunc) PrintArgs() string {
	names := make([]string, len(f.Params))
	for i := 0; i < len(names); i++ {
		names[i] = f.Params[i].Name()
	}
	return strings.Join(names, ", ")
}

func (f PyFunc) PrintReturns() string {
	returns := ""
	if len(f.Results) > 1 {
		names := []string{}
		for i := 0; i < len(f.Results); i++ {
			result := f.Results[i]
			if !cgo.ImplementsError(result.underlying.Type()) {
				names = append(names, result.ReturnFormat(fmt.Sprintf(RETURN_VAR_NAME+".r%d", i)))
			}
		}
		returns = strings.Join(names, ", ")
	} else {
		if !cgo.ImplementsError(f.Results[0].underlying.Type()) {
			result := f.Results[0]
			returns = result.ReturnFormat(RETURN_VAR_NAME)
		}
	}

	if returns != "" {
		return "return " + returns
	} else {
		return ""
	}
}
