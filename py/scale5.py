# Copyright 2025 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

import math
from rough import Rough
from scale1 import scale as scale1
import scale2
from scale3 import leftmost

def split(pm):
	pmHi = (Rough.value(pm) >> 64).ceil()
	pmLo = (pmHi<<64) - pm
	return pmHi, pmLo

def pow10(p):
	pm, pe = scale2.pow10(p)
	pmHi, pmLo = split(pm)
	return pmHi, pmLo, pe

assert split(0xe45c10c42a2b3b058cb89a7db77c506b) == (0xe45c10c42a2b3b06, 0x734765824883af95)

def mulHi(x, y):
	return (x*y) >> 64

def mulLo(x, y):
	return (x*y) & ((1<<64)-1)

# scale(x, e, p) returns Rough.value(x * 2**e * 10**p).
def scale(x: int, e: int, p: int) -> Rough:
	x, e = leftmost(x, e)
	pmHi, pmLo, pe = pow10(p)

	s = -(e+pe) - 64 - 2
	hi, mid1 = mulHi(x, pmHi), mulLo(x, pmHi)
	mid2, lo = mulHi(x, pmLo), mulLo(x, pmLo)
	w = ((hi<<128) + (mid1<<64) - (mid2<<64) - lo) >> 64
	return Rough((w>>s) | (w&((1<<s)-1) != 0))

assert scale(0x123456, -10, 2) == scale1(0x123456, -10, 2)
assert scale(0x123456, 10, -2) == scale1(0x123456, 10, -2)

# scale is inaccurate for a few 64-bit inputs looking for 64-bit outputs,
# but we don't care since we are not trying for 64-bit outputs.
# assert scale(0x89acc4afe3aed480, -290, 87) != scale1.scale(0x89acc4afe3aed480, -290, 87)
