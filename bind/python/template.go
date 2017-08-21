package python

import (
	"strings"
	"text/template"
)

const (
	PYTHON_TEMPLATE = `import os
import sys
import uuid
import cffi as _cffi_backend
from collections import MutableSequence
from abc import abstractmethod

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

    def go_uuid(self):
    	ba = bytearray(16)
    	ffi.memmove(ba, self._uuid_ptr, 16)
    	return uuid.UUID(bytes=ba)

    def uuid_ptr(self):
        return self._uuid_ptr


class VeilList(MutableSequence):
    def __init__(self, data=None, uuid_ptr=None):
        if uuid_ptr is None:
            uuid_ptr = self.__get_method__("new")()
        self._veil_obj = VeilObject(uuid_ptr)
        super(VeilList, self).__init__()


		@abstractmethod
    def __go_slice_type__(self):
        raise NotImplementedError("__go_slice_type__ is not implemented on VeilList and should "
                                  "only be implemented in the inheriting object.")

    def __go_type_input_transform__(self, value):
    	return value

    def __go_type_output_transform__(self, value):
    	return value

    def __len__(self):
        """List length"""
        return self.__get_method__("len")(self._veil_obj.uuid_ptr())

    def __getitem__(self, idx):
        """Get a list item"""
        if idx >= self.__len__():
        	raise IndexError
        value = self.__get_method__("item")(self._veil_obj.uuid_ptr(), idx)
        return self.__go_type_output_transform__(value)

    def __delitem__(self, idx):
        """Delete an item"""
        self.__get_method__("item_del")(self._veil_obj.uuid_ptr(), idx)

    def __setitem__(self, idx, val):
    	val = self.__go_type_input_transform__(val)
    	self.__get_method__("item_set")(self._veil_obj.uuid_ptr(), idx, val)

    def insert(self, idx, val):
    	val = self.__go_type_input_transform__(val)
    	self.__get_method__("item_insert")(self._veil_obj.uuid_ptr(), idx, val)

    def __go_str__(self):
        cret = self.__get_method__("str")(self._veil_obj.uuid_ptr())
        return _CffiHelper.c2py_string(cret)

    def __get_method__(self, method_name):
        return getattr(_CffiHelper.lib, self.__go_slice_type__() + "_" + method_name)


class VeilError(Exception):
    def __init__(self, uuid_ptr):
        self.veil_obj = VeilObject(uuid_ptr=uuid_ptr)
        message = _CffiHelper.error_string(uuid_ptr)
        super(VeilError, self).__init__(message)

    @staticmethod
    def is_nil(uuid_ptr):
        return _CffiHelper.lib.cgo_is_error_nil(uuid_ptr)

{{range $_, $listType := .Lists}}
class {{$listType.SliceType}}List(VeilList):
	def __init__(self, data=None, uuid_ptr=None):
		super({{$listType.SliceType}}List, self).__init__(data=data, uuid_ptr=uuid_ptr)

	def __go_slice_type__(self):
		return "{{$listType.MethodPrefix}}"

	def __go_type_input_transform__(self, value):
		{{call $listType.InputFormat "value"}}
		return value

	def __go_type_output_transform__(self, value):
		return {{call $listType.OutputFormat "value"}}

{{end}}


# Globally defined functions
{{range $_, $func := .Funcs}}
def {{$func.Name}}({{$func.PrintArgs}}):
    {{ range $_, $inTrx := $func.InputTransforms -}}
      {{ $inTrx }}
    {{ end -}}
    {{$cret}} = _CffiHelper.lib.{{$func.Call -}}
    {{ range $idx, $result := $func.Results }}
		{{if $result.IsError -}}
			if not VeilError.is_nil(cret.r1):
				{{ printf "raise VeilError(%s.r%d)" $cret $idx -}}
		{{end}}
	{{ end }}

    {{$func.PrintReturns}}
{{end -}}

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
			{{$func.PrintReturns}}

		{{end -}}

		{{if $class.Methods}}# Methods{{end}}
		{{range $_, $func := $class.Methods }}
		def {{$func.Name}}(self{{if $func.PrintArgs}}, {{end}}{{$func.PrintArgs}}):
			{{ range $_, $inTrx := $func.InputTransforms -}}
			  {{ $inTrx }}
			{{ end -}}
			{{$cret}} = _CffiHelper.lib.{{$func.Call -}}
			{{ range $idx, $result := $func.Results -}}
				{{if $result.IsError }}
			if not VeilError.is_nil(cret.r1):
				{{ printf "raise VeilError(%s.r%d)" $cret $idx -}}
				{{end}}
			{{ end -}}
			{{$func.PrintReturns}}

		{{end -}}

		{{if $class.Fields}}# Properties{{end}}
		{{ range $_, $field := $class.Fields -}}
		@property
		def {{$field.Name}}(self):
			cret = _CffiHelper.lib.{{$class.MethodName $field}}_get(self._uuid_ptr)
			return {{$field.ReturnFormat "cret"}}

		@{{$field.Name}}.setter
		def {{$field.Name}}(self, value):
			{{with $format := $field.InputFormat "value"}}{{if $format}}{{$format}}{{end}}{{end}}
			_CffiHelper.lib.{{$class.MethodName $field}}_set(self._uuid_ptr, value)
    {{ end -}}

{{end}}

`
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
