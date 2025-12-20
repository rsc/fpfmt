# Copyright 2025 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

import math
from rough import Rough
from scale1 import scale as scale1
from scale3 import leftmost
from scale5 import pow10, mulHi, mulLo

# scale(x, e, p) returns Rough.value(x * 2**e * 10**p).
def scale(x: int, e: int, p: int) -> Rough:
	x, e = leftmost(x, e)
	pmHi, pmLo, pe = pow10(p)

	s = -(e+pe) - 128 - 2
	hi, mid1 = mulHi(x, pmHi), mulLo(x, pmHi)
	mid2, lo = mulHi(x, pmLo), mulLo(x, pmLo)
	mid = mid1 - mid2 - (lo > 0)
	hi -= mid < 0
	return Rough((hi>>s) | (hi&((1<<s)-1) != 0) | (mid != 0))

assert scale(0x123456, -10, 2) == scale1(0x123456, -10, 2)
assert scale(0x123456, 10, -2) == scale1(0x123456, 10, -2)

# scale is inaccurate for a few 64-bit inputs looking for 64-bit outputs,
# but we don't care since we are not trying for 64-bit outputs.
# assert scale(0x89acc4afe3aed480, -290, 87) != scale1.scale(0x89acc4afe3aed480, -290, 87)
