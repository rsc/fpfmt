# Copyright 2025 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

import math
from rough import Rough
from scale1 import scale as scale1
from scale2 import pow10

def leftmost(x: int, e: int) -> (int, int):
	s = 64 - x.bit_length()
	return x << s, e - s

def scale(x: int, e: int, p: int) -> Rough:
	x, e = leftmost(x, e)
	pm, pe = pow10(p)
	return Rough.value((x*pm)>>64) >> -(e+pe+64)

assert scale(0x123456, -10, 2) == scale1(0x123456, -10, 2)
assert scale(0x123456, 10, -2) == scale1(0x123456, 10, -2)

# scale is inaccurate for a few 64-bit inputs looking for 64-bit outputs,
# but we don't care since we are not trying for 64-bit outputs.
# assert scale(0x89acc4afe3aed480, -290, 87) != scale1.scale(0x89acc4afe3aed480, -290, 87)
