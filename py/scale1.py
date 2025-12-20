# Copyright 2025 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

from rough import Rough

# scale(x, e, p) returns ext(x * 2**e * 10**p).
def scale(x: int, e: int, p: int) -> Rough:
	num = x * 2**max(e, 0) * 10**max(p, 0)
	denom = 2**max(-e, 0) * 10**max(-p, 0)
	return Rough.value(num) / denom

assert scale(12, -2, 0) == Rough.value(12/4)
assert scale(12, -4, 0) == Rough.value(12/16)
assert scale(12, 0, 2) == Rough.value(12*100)
assert scale(12, 0, -2) == Rough.value(12/100)
assert scale(9, -3, 2) == Rough.value(9/8*100)
assert scale(9, 4, -2) == Rough.value(9*16/100)
