#!/usr/bin/env python

import tempfile
import os
from shutil import copyfile
from subprocess import call


def generate(package):
    temp_dir = tempfile.mkdtemp(prefix="veil_")
    cmd = ["./bin/github.com/devigned/veil", "generate", "-p", package, "-o", temp_dir, "-n", "libGen"]
    call(cmd)
    return temp_dir


def copy_test(test_path, tmp_dir):
    copyfile(test_path, os.path.join(tmp_dir, os.path.basename(test_path)))


def register_helloworld_python():
    tmp_dir = generate("github.com/devigned/veil/_examples/helloworld")
    copy_test("./_examples/helloworld/python/hello_test.py", tmp_dir)
    return tmp_dir


def register_tests():
    return [
        register_helloworld_python()
    ]


if __name__ == "__main__":
    test_dirs = register_tests()
    call(["nosetests"] + ["-w{}".format(i) for i in test_dirs])
