import os
from cffi import FFI

ffibuilder = FFI()
here = os.path.dirname(os.path.abspath(__file__))
header_path = os.path.join(here, "_hello.h")
ffibuilder.set_source("_hello_example", open(header_path, "r").read(),
                      libraries=[os.path.join(here, "hello")])
ffibuilder.cdef("""
    int GetMagicNumber();
""")

if __name__ == "__main__":
    ffibuilder.compile(verbose=True)