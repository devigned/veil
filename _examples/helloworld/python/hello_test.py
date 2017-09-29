import generated
import unittest
import sys

_PY3 = sys.version_info[0] == 3


class TestHello(unittest.TestCase):
    def test_get_magic_number(self):
        self.assertEqual(generated.get_magic_number(), 42)

    def test_does_not_throw_error(self):
        self.assertIsNotNone(generated.public_unbound_error(100))

    def test_throws_when_1000(self):
        with self.assertRaises(generated.VeilError):
            generated.public_unbound_error(1000)

    def test_string_list(self):
        strings = generated.StringList()
        strings.append("foo")
        strings.append("bar")
        self.assertEqual(len(strings), 2)
        strings[1] = "bazz"
        self.assertEqual(strings[1], "bazz")
        del strings[1]
        self.assertEqual(len(strings), 1)

    def test_tuple_return(self):
        ret = generated.public_multi_return(42, "Hello world!")
        self.assertTupleEqual(ret, (42, "Hello world!"))

    def test_struct_construction(self):
        some_string = "some cool string"
        hello_obj = generated.Hello()
        hello_obj.foo = 150
        hello_obj.bar = some_string
        self.assertEqual(hello_obj.foo, 150)
        self.assertEqual(hello_obj.bar, some_string)
        self.assertIsNotNone(hello_obj.__go_str__())


class StringReader(generated.Reader):
    def __init__(self, s):
        super(StringReader, self).__init__()
        self.s = s

    def read(self, p):
        if _PY3:
            utf8_bytes = self.s.encode("utf-8")
        else:
            utf8_bytes = bytearray(self.s)

        if len(p) < len(utf8_bytes):
            raise Exception("buffer is not large enough")

        for idx, byte in enumerate(utf8_bytes):
            p[idx] = byte

        return len(utf8_bytes), None


class TestInterface(unittest.TestCase):
    def test_string_reader(self):
        reader = StringReader("hello world!")
        hello = generated.Hello()
        self.assertEqual(hello.public_interface(reader), 12)
