import os
import subprocess
import re
from cffi import FFI

here = os.path.dirname(os.path.abspath(__file__))
header_path = os.path.join(here, "_hello.h")

ffi_builder = FFI()
expanded_header = subprocess.Popen([
    'cc', '-E', header_path],
    stdout=subprocess.PIPE).communicate()[0].decode()

patterns = '.+?_check_for_64_bit_pointer_matching_GoInt|.+?_Complex'
filtered_header = [line for line in expanded_header.split("\n")
                   if re.match(patterns, line) is None]
ffi_builder.cdef("\n".join(filtered_header))


# python <--> cffi helper.
class FFIHelper(object):
    here = os.path.dirname(os.path.abspath(__file__))
    lib = ffi_builder.dlopen(os.path.join(here, "_hello.so"))

    @staticmethod
    def get_magic_number():
        return FFIHelper.lib.GetMagicNumber()


def main():
    print(FFIHelper.get_magic_number())


if __name__ == "__main__":
    main()
