package bind

import (
	"bufio"
	"fmt"
	"github.com/devigned/veil/cgo"
	"github.com/devigned/veil/core"
	"go/types"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"text/template"
	"unicode"
)

const (
	RETURN_VAR_NAME  = "cret"
	CFFI_HELPER_NAME = "_CffiHelper"
	HEADER_FILE_NAME = "output.h"
	PYTHON_FILE_NAME = "generated.py"
	PYTHON_TEMPLATE  = `import os
import sys
import cffi as _cffi_backend

_PY3 = sys.version_info[0] == 3

ffi = _cffi_backend.FFI()
ffi.cdef("""{{.CDef}}""")

{{ $cret := .ReturnVarName -}}
{{ $cffiHelperName := .CffiHelperName -}}

class _CffiHelper(object):

    here = os.path.dirname(os.path.abspath(__file__))
    lib = ffi.dlopen(os.path.join(here, "output"))

    @staticmethod
    def error_string(ptr):
        return _CffiHelper.c2py_string(_CffiHelper.lib.cgo_error_to_string(ptr))

    @staticmethod
    def cgo_free(ptr):
        return _CffiHelper.lib.cgo_cfree(ptr)

    @staticmethod
    def cgo_decref(ptr):
        return _CffiHelper.lib.cgo_decref(ptr)

    @staticmethod
    def handle_error(err):
        ptr = ffi.cast("void *", err)
        if not _CffiHelper.lib.cgo_is_error_nil(ptr):
            raise Exception(_CffiHelper.error_string(ptr))

    @staticmethod
    def c2py_string(s):
        pystr = ffi.string(s)
        _CffiHelper.lib.cgo_cfree(s)
        if _PY3:
            pystr = pystr.decode('utf-8')
        return pystr


class VeilObject(object):
    def __init__(self, uuid_ptr):
        self._uuid_ptr = uuid_ptr

    def __del__(self):
        _CffiHelper.cgo_decref(self._uuid_ptr)


class VeilError(Exception):
    def __init__(self, uuid_ptr):
        self.veil_obj = VeilObject(uuid_ptr=uuid_ptr)
        message = _CffiHelper.error_string(uuid_ptr)
        super(VeilError, self).__init__(message)

    @staticmethod
    def is_nil(uuid_ptr):
        return _CffiHelper.lib.cgo_is_error_nil(uuid_ptr)


# Globally defined functions
{{range $_, $func := .Funcs}}
def {{$func.Name}}({{$func.PrintArgs}}):
    {{ range $_, $inTrx := $func.InputTransforms -}}
      {{ $inTrx }}
    {{ end -}}
    {{$cret}} = _CffiHelper.lib.{{$func.Call -}}
    {{ range $idx, $result := $func.Results -}}
		{{if $result.IsError -}}
			if not VeilError.is_nil(cret.r1):
				{{ printf "raise VeilError(%s.r%d)" $cret $idx -}}
		{{end}}
    {{ end -}}
    return {{$func.PrintReturns}}

{{end}}

{{range $_, $class := .Classes}}
class {{$class.Name}}(VeilObject):

		def __init__(self, uuid_ptr=None):
			if uuid_ptr is None:
				uuid_ptr = _CffiHelper.lib.{{$class.NewMethodName}}()
			super({{$class.Name}}, self).__init__(uuid_ptr)

		def __go_str__(self):
			cret = _CffiHelper.lib.{{$class.ToStringMethodName}}(self._uuid_ptr)
			return _CffiHelper.c2py_string(cret)

		{{if $class.Constructors}}# Constructors{{end}}

		{{range $_, $func := $class.Constructors }}
		@staticmethod
		def {{$func.Name}}({{$func.PrintArgs}}):
			# TODO: Add constructor logic
			pass

		{{end}}

		# Properties

		{{ range $_, $field := $class.Fields -}}
		@property
		def {{$field.Name}}(self):
			cret = _CffiHelper.lib.{{$class.MethodName $field}}_get(self._uuid_ptr)
			return cret

		@{{$field.Name}}.setter
		def {{$field.Name}}(self, value):
			_CffiHelper.lib.{{$class.MethodName $field}}_set(self._uuid_ptr, value)
    {{ end -}}

{{end}}

`
	STRING_INPUT_TRANSFORM = "%s = ffi.new(\"char[]\", %s.encode(\"utf-8\"))"

	STRING_OUTPUT_TRANSFORM = "_CffiHelper.c2py_string(%s)"
)

var (
	startCGoDefine  = regexp.MustCompile(`^typedef`)
	sizeOfRemove    = regexp.MustCompile(`_check_for_64_bit_pointer_matching_GoInt`)
	complexRemove   = regexp.MustCompile(`_Complex`)
	endif           = regexp.MustCompile(`^#endif`)
	endOfCGoDefine  = regexp.MustCompile(`^#ifdef __cplusplus`)
	extern          = regexp.MustCompile(`^extern \w`)
	sizeTypeReplace = regexp.MustCompile(`__SIZE_TYPE__`)
	removeFilters   = []*regexp.Regexp{sizeOfRemove, complexRemove}
	replaceFilters  = map[string]*regexp.Regexp{"size_t": sizeTypeReplace}
)

var pythonTemplate *template.Template

func init() {
	replacedTabsTemplate := removeTabs(PYTHON_TEMPLATE)
	if tmpl, err := template.New("codeTemplate").Parse(replacedTabsTemplate); err != nil {
		panic(err)

	} else {
		pythonTemplate = tmpl
	}
}

func removeTabs(src string) string {
	return strings.Replace(src, "\t", "  ", -1)
}

// Py3Binder contains the data for generating a python 3 binding
type Py3Binder struct {
	pkg *cgo.Package
}

type PyTemplateData struct {
	CDef           string
	Funcs          []*PyFunc
	Constructors   map[string]*PyFunc
	Classes        []*PyClass
	CffiHelperName string
	ReturnVarName  string
}

type PyParam struct {
	underlying *types.Var
}

type PyClass struct {
	*cgo.Struct
	Fields       []*PyParam
	Constructors []*PyFunc
}

func (p Py3Binder) NewPyClass(s *cgo.Struct) *PyClass {
	fields := make([]*PyParam, s.Struct().NumFields())
	for i := 0; i < s.Struct().NumFields(); i++ {
		fields[i] = NewPyParam(s.Struct().Field(i))
	}

	constructors := []*PyFunc{}
	for _, f := range p.pkg.Funcs() {
		if s.IsConstructor(f) {
			constructors = append(constructors, ToPyFunc(f))
		}
	}

	return &PyClass{
		Struct:       s,
		Fields:       fields,
		Constructors: constructors,
	}
}

func (c PyClass) Name() string {
	return c.Struct.Named.Obj().Name()
}

func (c PyClass) MethodName(p *PyParam) string {
	return c.CGoFieldName(p.underlying)
}

func (c PyClass) NewMethodName() string {
	return c.Struct.NewMethodName()
}

func NewPyParam(v *types.Var) *PyParam {
	return &PyParam{underlying: v}
}

func (p PyParam) Name() string {
	return ToSnake(p.underlying.Name())
}

func (p PyParam) IsError() bool {
	return cgo.ImplementsError(p.underlying.Type())
}

func (p PyParam) ReturnFormat(varName string) string {
	switch t := p.underlying.Type().(type) {
	case *types.Basic:
		if t.Kind() == types.String {
			return fmt.Sprintf(STRING_OUTPUT_TRANSFORM, varName)
		}
	}
	return varName
}

type PyFunc struct {
	fun     cgo.Func
	Name    string
	Params  []*PyParam
	Results []*PyParam
}

func (f PyFunc) InputTransforms() []string {
	inputTranforms := []string{}
	for _, param := range f.Params {
		switch t := param.underlying.Type().(type) {
		case *types.Basic:
			if t.Kind() == types.String {
				varName := param.Name()
				inputTranforms = append(inputTranforms,
					fmt.Sprintf(STRING_INPUT_TRANSFORM, varName, varName))
			}
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
	if len(f.Results) > 1 {
		names := []string{}
		for i := 0; i < len(f.Results); i++ {
			result := f.Results[i]
			if !cgo.ImplementsError(result.underlying.Type()) {
				names = append(names,
					result.ReturnFormat(fmt.Sprintf(RETURN_VAR_NAME+".r%d", i)))
			}
		}
		return strings.Join(names, ", ")
	} else {
		result := f.Results[0]
		return result.ReturnFormat(RETURN_VAR_NAME)
	}
}

// NewPy3Binder creates a new Binder for Python 3
func NewPy3Binder(pkg *cgo.Package) Bindable {
	return &Py3Binder{
		pkg: pkg,
	}
}

// Bind is the Python 3 implementation of Bind
func (p Py3Binder) Bind(outDir string) error {
	headerPath := path.Join(outDir, HEADER_FILE_NAME)
	cdefText, err := p.cDefText(headerPath)
	if err != nil {
		return core.NewSystemErrorF("Failed to generate Python CDefs: %v", err)
	}

	data := PyTemplateData{
		CDef:           strings.Join(cdefText, "\n"),
		Funcs:          p.Funcs(),
		Classes:        p.Classes(),
		CffiHelperName: CFFI_HELPER_NAME,
		ReturnVarName:  RETURN_VAR_NAME,
	}

	pythonFilePath := path.Join(outDir, PYTHON_FILE_NAME)
	f, err := os.Create(pythonFilePath)
	if err != nil {
		return core.NewSystemErrorF("Unable to create %s", path.Join(outDir, PYTHON_FILE_NAME))
	}

	w := bufio.NewWriter(f)
	err = pythonTemplate.Execute(w, data)
	w.Flush()
	f.Close()
	if err != nil {
		panic(err)
	}

	PyFormat(pythonFilePath)

	return nil
}

func (p Py3Binder) Classes() []*PyClass {
	classes := make([]*PyClass, len(p.pkg.Structs()))
	for idx, s := range p.pkg.Structs() {
		classes[idx] = p.NewPyClass(s)
	}
	return classes
}

func (p Py3Binder) Funcs() []*PyFunc {
	funcs := []*PyFunc{}
	for _, f := range p.pkg.Funcs() {
		if p.pkg.IsConstructor(f) {
			continue
		}
		funcs = append(funcs, ToPyFunc(f))
	}
	return funcs
}

func ToPyFunc(f cgo.Func) *PyFunc {
	pyParams := make([]*PyParam, f.Signature().Params().Len())
	for i := 0; i < f.Signature().Params().Len(); i++ {
		param := f.Signature().Params().At(i)
		pyParams[i] = NewPyParam(param)
	}

	pyResults := make([]*PyParam, f.Signature().Results().Len())
	for i := 0; i < f.Signature().Results().Len(); i++ {
		param := f.Signature().Results().At(i)
		pyResults[i] = NewPyParam(param)
	}

	return &PyFunc{
		fun:     f,
		Name:    ToSnake(f.Name()),
		Params:  pyParams,
		Results: pyResults,
	}
}

func (p Py3Binder) cDefText(headerPath string) ([]string, error) {
	if file, err := os.Open(headerPath); err == nil {
		defer file.Close()

		filteredHeaders := []string{}
		recording := false

		// create a new scanner and read the file line by line
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			text := scanner.Text()

			if !recording && (startCGoDefine.MatchString(text) || extern.MatchString(text)) {
				recording = true
			}

			if recording {
				if endif.MatchString(text) || endOfCGoDefine.MatchString(text) {
					recording = false
					continue
				}

				matched := false
				for _, filter := range removeFilters {
					if filter.MatchString(text) {
						matched = true
						break
					}
				}

				if !matched {
					for key, value := range replaceFilters {
						if value.MatchString(text) {
							text = value.ReplaceAllString(text, key)
						}
					}
					text = removeTabs(text)
					filteredHeaders = append(filteredHeaders, text)
				}
			}
		}

		// check for errors
		if err = scanner.Err(); err != nil {
			return nil, core.NewSystemError(err)
		}

		return filteredHeaders, nil

	} else {
		return nil, core.NewSystemError(err)
	}
}

// ToSnake convert the given string to snake case following the Golang format:
// acronyms are converted to lower-case and preceded by an underscore.
// via: https://gist.github.com/elwinar/14e1e897fdbe4d3432e1
func ToSnake(in string) string {
	runes := []rune(in)
	length := len(runes)

	var out []rune
	for i := 0; i < length; i++ {
		if i > 0 && unicode.IsUpper(runes[i]) && ((i+1 < length && unicode.IsLower(runes[i+1])) || unicode.IsLower(runes[i-1])) {
			out = append(out, '_')
		}
		out = append(out, unicode.ToLower(runes[i]))
	}

	return string(out)
}

func PyFormat(path string) {
	which := exec.Command("which", "yapf")
	if err := which.Run(); err == nil {
		cmd := exec.Command("yapf", "-i", "--style={based_on_style: pep8, column_limit: 100}", path)
		err = cmd.Run()
	} else {
		log.Println("To format your Python code run `pip install yapf`")
	}
}
