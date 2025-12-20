# Copyright 2025 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

from scale1 import scale as scale1
import math
from rough import Rough

def pow10(p):
    return pow10Tab[p-pow10Min], pe10(p)

def pe10(p):
    return int(math.floor(math.log2(10) * p)) - 127

def pm10(p):
    return scale1(1, -pe10(p), p).ceil()

pow10Min = -400
pow10Max = 400
pow10Tab = [pm10(p) for p in range(pow10Min, pow10Max+1)]

assert pow10(0) == (2**127, -127)
assert pow10(25) == (0x84595161401484a00000000000000000, -44)
assert pow10(72) == (0x90e40fbeea1d3a4abc8955e946fe31ce, 112)
assert pow10(-44) == (0xe45c10c42a2b3b058cb89a7db77c506b, -274)

pm, pe = pow10(40)
assert pm * 2**pe == 10**40

# scale(x, e, p) returns (x * 2**e * 10**p).
def scale(x: int, e: int, p: int) -> Rough:
	pm, pe = pow10(p)
	b = x.bit_length()
	return Rough.value((x*pm)>>b) >> -(e+pe+b)

assert scale(0x123456, -10, 2) == scale1(0x123456, -10, 2)
assert scale(0x123456, 10, -2) == scale1(0x123456, 10, -2)

# scale is inaccurate for a few 64-bit inputs looking for 64-bit outputs,
# but we don't care since we are not trying for 64-bit outputs.
# print(scale(0x89acc4afe3aed480, -290, 87), scale1.scale(0x89acc4afe3aed480, -290, 87))
# assert scale(0x89acc4afe3aed480, -290, 87) != scale1.scale(0x89acc4afe3aed480, -290, 87)

