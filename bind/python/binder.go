package python

import (
	"bufio"
	"fmt"
	"github.com/devigned/veil/cgo"
	"github.com/devigned/veil/core"
	"github.com/emirpasic/gods/sets/hashset"
	"go/token"
	"go/types"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
)

const (
	RETURN_VAR_NAME  = "cret"
	CFFI_HELPER_NAME = "_CffiHelper"
	HEADER_FILE_NAME = "output.h"
	FILE_NAME        = "generated.py"
)

var (
	startCGoDefine  = regexp.MustCompile(`^typedef|^struct`)
	sizeOfRemove    = regexp.MustCompile(`_check_for_64_bit_pointer_matching_GoInt`)
	complexRemove   = regexp.MustCompile(`_Complex`)
	endif           = regexp.MustCompile(`^#endif`)
	pounds          = regexp.MustCompile(`^#line|#ifndef|^#define|^#ifdef`)
	inline          = regexp.MustCompile(`^inline`)
	endOfCGoDefine  = regexp.MustCompile(`^#ifdef __cplusplus`)
	extern          = regexp.MustCompile(`^extern \w`)
	sizeTypeReplace = regexp.MustCompile(`__SIZE_TYPE__`)
	removeFilters   = []*regexp.Regexp{sizeOfRemove, complexRemove, pounds, inline}
	replaceFilters  = map[string]*regexp.Regexp{"size_t": sizeTypeReplace}
	reserved_words  = hashset.New()
)

func init() {
	words := []string{"False", "True", "None", "and", "as", "assert", "break", "class",
		"continue", "def", "del", "elif", "else", "except", "finally", "for", "from", "global",
		"if", "import", "in", "is", "lambda", "nonlocal", "not", "or", "pass", "raise", "return",
		"try", "while", "with", "yield"}
	for _, word := range words {
		reserved_words.Add(word)
	}
}

// Binder contains the data for generating a python 3 binding
type Binder struct {
	pkg *cgo.Package
}

type TemplateData struct {
	CDef           string
	Funcs          []*Func
	Constructors   map[string]*Func
	Classes        []*Class
	Lists          []*List
	Interfaces     []*Interface
	CffiHelperName string
	ReturnVarName  string
	LibName        string
}

// NewBinder creates a new Binder for Python
func NewBinder(pkg *cgo.Package) core.Binder {
	return &Binder{
		pkg: pkg,
	}
}

func (p Binder) NewList(slice *cgo.Slice) *List {
	v := types.NewVar(token.Pos(0), nil, "value", slice.Elem())
	return &List{
		Slice:        slice,
		MethodPrefix: slice.CGoName(),
		InputFormat: func() string {
			return InputFormat("value", slice.Elem())
		},
		OutputFormat: p.NewParam(v, "value").ReturnFormatWithName,
	}
}

func (p Binder) NewClass(s *cgo.Struct) *Class {
	fields := []*Param{}
	for i := 0; i < s.Struct().NumFields(); i++ {
		field := s.Struct().Field(i)
		param := p.NewParam(s.Struct().Field(i), fmt.Sprintf("param_%d", i))
		if cgo.ShouldGenerateField(field) && !IsReservedWord(param.Name()) {
			fields = append(fields, param)
		}
	}

	constructors := []*Func{}
	for _, f := range p.pkg.Funcs() {
		if s.IsConstructor(f) {
			constructors = append(constructors, p.ToConstructor(s, f))
		}
	}

	methods := []*Func{}
	for _, f := range s.ExportedMethods() {
		fun := p.ToFunc(f)
		if !IsReservedWord(fun.Name) {
			methods = append(methods, fun)
		}
	}

	return &Class{
		binder:       &p,
		Struct:       s,
		Fields:       fields,
		Constructors: constructors,
		Methods:      methods,
	}
}

func (p Binder) NewInterface(i *cgo.Interface) *Interface {
	methods := []*Func{}
	for _, f := range i.ExportedMethods() {
		fun := p.ToFunc(f)
		if !IsReservedWord(fun.Name) {
			methods = append(methods, fun)
		}
	}

	return &Interface{
		binder:    &p,
		Interface: i,
		Methods:   methods,
	}
}

func (p *Binder) NewParam(v *types.Var, defaultName string) *Param {
	return &Param{
		underlying:  v,
		binder:      p,
		DefaultName: defaultName,
	}
}

// Bind is the Python 3 implementation of Bind
func (p Binder) Bind(outDir, libName string) error {
	headerPath := path.Join(outDir, fmt.Sprintf("%s.h", libName))
	cdefText, err := p.cDefText(headerPath)
	if err != nil {
		fmt.Println(err)
		return core.NewSystemErrorF("Failed to generate Python CDefs: %v", err)
	}

	data := TemplateData{
		CDef:           strings.Join(cdefText, "\n"),
		Funcs:          p.Funcs(),
		Classes:        p.Classes(),
		Lists:          p.Lists(),
		Interfaces:     p.Interfaces(),
		CffiHelperName: CFFI_HELPER_NAME,
		ReturnVarName:  RETURN_VAR_NAME,
		LibName:        libName,
	}

	pythonFilePath := path.Join(outDir, FILE_NAME)
	fmt.Println("foo", pythonFilePath)
	f, err := os.Create(pythonFilePath)
	if err != nil {
		return core.NewSystemErrorF("Unable to create %s", path.Join(outDir, FILE_NAME))
	}

	w := bufio.NewWriter(f)
	err = pythonTemplate.Execute(w, data)
	w.Flush()
	f.Close()
	if err != nil {
		panic(err)
	}

	Format(pythonFilePath)

	return nil
}

func (p Binder) Lists() []*List {
	lists := []*List{}
	for _, exp := range p.pkg.ExportedTypes() {
		if slice, ok := exp.(cgo.Slice); ok {
			lists = append(lists, p.NewList(&slice))
		}
	}
	return lists
}

func (p Binder) Classes() []*Class {
	classes := make([]*Class, len(p.pkg.Structs()))
	for idx, s := range p.pkg.Structs() {
		classes[idx] = p.NewClass(s)
	}
	return classes
}

func (p Binder) Interfaces() []*Interface {
	interfaces := make([]*Interface, len(p.pkg.Interfaces()))
	for idx, i := range p.pkg.Interfaces() {
		interfaces[idx] = p.NewInterface(i)
	}
	return interfaces
}

func (p Binder) Funcs() []*Func {
	funcs := []*Func{}
	for _, f := range p.pkg.Funcs() {
		if IsReservedWord(f.Name()) {
			continue
		}
		if p.pkg.IsConstructor(f) {
			continue
		}
		funcs = append(funcs, p.ToFunc(f))
	}
	return funcs
}

func (p Binder) ToConstructor(class *cgo.Struct, f *cgo.Func) *Func {
	fun := p.ToGenericFunc(f)
	fun.Name = core.ToSnake(class.ConstructorName(f))
	return fun
}

func (p Binder) ToFunc(f *cgo.Func) *Func {
	fun := p.ToGenericFunc(f)
	fun.Name = core.ToSnake(f.Name())
	return fun
}

func (p Binder) ToGenericFunc(f *cgo.Func) *Func {
	pyParams := make([]*Param, f.Signature().Params().Len())
	for i := 0; i < f.Signature().Params().Len(); i++ {
		param := f.Signature().Params().At(i)
		pyParams[i] = p.NewParam(param, fmt.Sprintf("param_%d", i))
	}

	pyResults := make([]*Param, f.Signature().Results().Len())
	for i := 0; i < f.Signature().Results().Len(); i++ {
		param := f.Signature().Results().At(i)
		pyResults[i] = p.NewParam(param, fmt.Sprintf("r_%d", i))
	}
	return &Func{
		fun:     f,
		Name:    core.ToSnake(f.Name()),
		Params:  pyParams,
		Results: pyResults,
	}
}

func (p Binder) cDefText(headerPath string) ([]string, error) {
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

func Format(path string) {
	which := exec.Command("which", "yapf")
	if err := which.Run(); err == nil {
		cmd := exec.Command("yapf", "-i", "--style={based_on_style: pep8, column_limit: 100}", path)
		err = cmd.Run()
	} else {
		log.Println("To format your Python code run `pip install yapf`")
	}
}

func IsReservedWord(word string) bool {
	return reserved_words.Contains(word)
}
