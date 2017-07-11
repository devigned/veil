import os
from cffi import FFI

here = os.path.dirname(os.path.abspath(__file__))
header_path = os.path.join(here, "_hello.h")

ffibuilder = FFI()
ffibuilder.cdef("""
int GetMagicNumber();
""")

# python <--> cffi helper.
class cffi_helper(object):
    here = os.path.dirname(os.path.abspath(__file__))
    lib = ffibuilder.dlopen(os.path.join(here, "_hello.so"))

    @staticmethod
    def get_magic_number():
        return cffi_helper.lib.GetMagicNumber()

def main():
  print(cffi_helper.get_magic_number())
  # j = 0
  # for i in range(10000000):
  #   j += 1

if __name__ == "__main__":
    main()
