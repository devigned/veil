from _hello_example import ffi, lib

if __name__ == "__main__":
  for i in range(1000000):
    lib.GetMagicNumber()
  print(lib.GetMagicNumber())