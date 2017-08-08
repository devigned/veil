package bind

import (
	"bufio"
	"github.com/devigned/veil/cgo"
	"github.com/devigned/veil/core"
	"os"
	"path"
	"regexp"
	"strings"
	"text/template"
)

const (
	HEADER_FILE_NAME = "output.h"
	PYTHON_FILE_NAME = "generated.py"
	PYTHON_TEMPLATE  = `
import os
import sys
import cffi as _cffi_backend

_PY3 = sys.version_info[0] == 3

ffi = _cffi_backend.FFI()
ffi.cdef("""
{{.CDef}}
""")
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
	if tmpl, err := template.New("codeTemplate").Parse(PYTHON_TEMPLATE); err != nil {
		panic(err)

	} else {
		pythonTemplate = tmpl
	}
}

// Py3Binder contains the data for generating a python 3 binding
type Py3Binder struct {
	pkg *cgo.Package
}

type PyTemplateData struct {
	CDef string
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

	data := PyTemplateData{CDef: strings.Join(cdefText, "\n")}

	f, err := os.Create(path.Join(outDir, PYTHON_FILE_NAME))
	if err != nil {
		return core.NewSystemErrorF("Unable to create %s", path.Join(outDir, PYTHON_FILE_NAME))
	}

	defer f.Close()

	w := bufio.NewWriter(f)
	pythonTemplate.Execute(w, data)
	w.Flush()

	return nil
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
