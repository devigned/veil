package bind

import (
	"bufio"
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
	HEADER_FILE_NAME = "output.h"
	PYTHON_FILE_NAME = "generated.py"
	PYTHON_TEMPLATE  = `import os
import sys
import cffi as _cffi_backend

_PY3 = sys.version_info[0] == 3

ffi = _cffi_backend.FFI()
ffi.cdef("""{{.CDef}}""")


class _CffiHelper(object):

    here = os.path.dirname(os.path.abspath(__file__))
    lib = ffi.dlopen(os.path.join(here, "output"))

    @staticmethod
    def error_string(err):
        ptr = ffi.cast("void *", err)
        return _CffiHelper.lib.cgo_error_to_string(ptr)

    @staticmethod
    def c2py_string(s):
        pystr = ffi.string(s)
        _CffiHelper.lib.cgo_cfree(s)
        if _PY3:
            pystr = pystr.decode('utf-8')
        return pystr


# Globally defined functions
{{range $_, $func := .Funcs}}
def {{$func.Name}}({{$func.PrintArgs}}):
    {{ range $_, $inTrx := $func.InputTransforms -}}
      {{ $inTrx }}
    {{ end -}}
    return _CffiHelper.lib.{{$func.Call}}

{{end}}

`
)

var (
	startCGoDefine  = regexp.MustCompile(`^#define GO_CGO_PROLOGUE_H`)
	sizeOfRemove    = regexp.MustCompile(`_check_for_64_bit_pointer_matching_GoInt`)
	complexRemove   = regexp.MustCompile(`_Complex`)
	endif           = regexp.MustCompile(`^#endif`)
	endOfCGoDefine  = regexp.MustCompile(`^#ifdef __cplusplus`)
	extern          = regexp.MustCompile(`^extern`)
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
	CDef  string
	Funcs []*PyFunc
}

type PyParam struct {
	underlying     *types.Var
	InputTransform string
}

func NewPyParam(v *types.Var) *PyParam {
	return &PyParam{underlying: v}
}

func (p PyParam) HasInputTransform() bool {
	return p.InputTransform != ""
}

func (p PyParam) Name() string {
	return ToSnake(p.underlying.Name())
}

type PyFunc struct {
	fun    cgo.Func
	Name   string
	Params []*PyParam
}

func (f PyFunc) InputTransforms() []string {
	// TODO: add input transformations
	return []string{}
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
		CDef:  strings.Join(cdefText, "\n"),
		Funcs: p.Funcs(),
	}

	pythonFilePath := path.Join(outDir, PYTHON_FILE_NAME)
	f, err := os.Create(pythonFilePath)
	if err != nil {
		return core.NewSystemErrorF("Unable to create %s", path.Join(outDir, PYTHON_FILE_NAME))
	}

	w := bufio.NewWriter(f)
	pythonTemplate.Execute(w, data)
	w.Flush()
	f.Close()

	PyFormat(pythonFilePath)

	return nil
}

func (p Py3Binder) Funcs() []*PyFunc {
	funcs := make([]*PyFunc, len(p.pkg.Funcs()))

	for idx, f := range p.pkg.Funcs() {

		pyParams := make([]*PyParam, f.Signature().Params().Len())
		for i := 0; i < f.Signature().Params().Len(); i++ {
			param := f.Signature().Params().At(i)
			pyParams[i] = NewPyParam(param)
		}

		funcs[idx] = &PyFunc{
			fun:    f,
			Name:   ToSnake(f.Name()),
			Params: pyParams,
		}
	}
	return funcs
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

			if !recording && (startCGoDefine.MatchString(text) || extern.MatchString(text)) {
				recording = true
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
