package python

import (
	"fmt"
	"github.com/devigned/veil/cgo"
	"strings"
)

type Func struct {
	fun     *cgo.Func
	Name    string
	Params  []*Param
	Results []*Param
}

func (f Func) InputTransforms() []string {
	inputTranforms := []string{}
	for _, param := range f.Params {
		if format := param.InputFormat(); format != "" {
			inputTranforms = append(inputTranforms, format)
		}
	}
	return inputTranforms
}

func (f Func) Call() string {
	if f.IsBound() {
		return f.fun.CName() + "(" + f.PrintArgs() + ")"
	} else {
		return f.fun.CName() + "(self.uuid_ptr(), " + f.PrintArgs() + ")"
	}
}

func (f Func) PrintArgs() string {
	names := make([]string, len(f.Params))
	for i := 0; i < len(names); i++ {
		names[i] = f.Params[i].Name()
	}
	return strings.Join(names, ", ")
}

func (f Func) PrintReturns() string {
	returns := ""
	if len(f.Results) > 1 {
		names := []string{}
		for i := 0; i < len(f.Results); i++ {
			result := f.Results[i]
			if !cgo.ImplementsError(result.underlying.Type()) {
				names = append(names, result.ReturnFormatWithName(fmt.Sprintf(RETURN_VAR_NAME+".r%d", i)))
			}
		}
		returns = strings.Join(names, ", ")
	} else if len(f.Results) == 1 {
		if !cgo.ImplementsError(f.Results[0].underlying.Type()) {
			result := f.Results[0]
			returns = result.ReturnFormatWithName(RETURN_VAR_NAME)
		}
	}

	if returns != "" {
		return "return " + returns
	} else {
		return ""
	}
}

// ResultsLength returns the length of the results array
func (f Func) ResultsLength() int {
	return len(f.Results)
}

// IsBound returns true if the function is bound to a named type
func (f Func) IsBound() bool {
	return f.fun.BoundRecv == nil
}

func (f Func) RegistrationName() string {
	return f.fun.Name()
}

func (f Func) CallbackAttribute() string {
	voidPtrs := make([]string, f.ResultsLength())
	for i := 0; i < f.ResultsLength(); i++ {
		voidPtrs[i] = "void*"
	}
	return fmt.Sprintf("@ffi.callback(\"void*(%s)\")", strings.Join(voidPtrs, ", "))
}
