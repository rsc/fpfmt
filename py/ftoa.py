# Copyright 2025 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

import math
from scale1 import scale as scale1
from scale2 import scale as scale2
from scale3 import scale as scale3
from scale4 import scale as scale4
from scale5 import scale as scale5
from scale6 import scale as scale6
from scale7 import scale as scale7
from scale8 import scale as scale8

# unpack returns m, e such that f = m * 2**e
# and the mantissa m ∈ [2**63, 2**64).
def unpack(f: float) -> (int, int):
	fr, fe = math.frexp(f)
	return int(fr*(1<<64)), fe-64

assert unpack(1.5) == (3<<62, -63)

# pack returns f = m * 2**e.
def pack(m: int, e: int) -> float:
	return math.ldexp(m, e)

assert pack(3<<62, -63) == 1.5

# round(x, xt) rounds x/2 to the nearest integer, using xt to decide
# whether x/2 is an exact half integer. Half integers round to even.
def round(x: int, xt: int) -> int:
	return (x + (xt | ((x>>1)&1))) >> 1

# scale(m, e, p) returns trunc(x * 2**e * 10**p),
# requiring that 2**e * 10**p < 1.
# It is a variable so we can assign and test different implementations.
# See scale*.py.
scale = scale1

# FixedWidth computes the fixed-width formatting of f to n digits.
# It returns integers d, p where f ≈ d * 10**p and d has n digits.
# n can be at most 18.
def FixedWidth(f: float, n: int) -> (int, int):
	if n > 18:
		raise ValueError("too many digits")
	m, e = unpack(f)

	# Estimate scaling constant p.
	p = n - 1 - math.floor(math.log10(2)*(e+63))

	# r = f * 10**p.
	r = scale(m, e, p)

	# Round and correct extra digit or rollover.
	d = r.round()
	if d >= 10**n:
		d, p = (r/10).round(), p-1

	return d, -p

assert FixedWidth(1.234, 4) == (1234, -3)

# Short computes the shortest formatting of f,
# using as few digits as possible that will still round trip
# back to the original float64.
def Short(f: float) -> (int, int):
	# Unpack f into m * 2**e.
	m, e = unpack(f)

	# Compute min and max allowable decimal
	# as extended-precision mantissas,
	# along with the exact scaling constant p.
	b = 11 # unused bits at bottom of m
	if m == 2**63 and e > -1085:
		# Asymmetric spacing around exponent change.
		p = -math.floor(math.log10(3/4) + math.log10(2) * (e+b))
		min, max = m - (1<<(b-2)), m + (1<<(b-1))
	else:
		# Symmetric spacing.
		if e < -1085:
			# Subnormals have fewer mantissa bits.
			b = 11 + (-1085 - e)
		p = -math.floor(math.log10(2) * (e+b))
		min, max = m - (1<<(b-1)), m + (1<<(b-1))

	# Convert min and max to allowable decimals.
	# If m is odd, disallow exactly min or max.
	odd = (m>>b) & 1
	dmin = scale(min, e, p).nudge(+odd).ceil()
	dmax = scale(max, e, p).nudge(-odd).floor()

	# Find the best decimal out of at most nine choices in [dmin, dmax].
	# If one ends in zero, use it to get a shorter decimal.
	d0 = (dmax // 10) * 10
	if d0 >= dmin:
		# Remove trailing zeros.
		while d0 % 10 == 0:
			d0, p = d0 // 10, p - 1
		return d0, -p

	# If there is only one choice in [dmin, dmax], use it.
	if dmin == dmax:
		return dmin, -p

	# Otherwise, use the correctly-rounded decimal.
	return scale(m, e, p).round(), -p

assert Short(1.112536929253601e-308) == (1112536929253601, -323)
assert Short(8e-323) == (8, -323)
assert Short(1.0000000000000001e+23) == (10000000000000001, 7)
assert Short(1.234) == (1234, -3)

# Parse rounds d * 10^p to the nearest float64 f.
# d can have at most 19 digits.
def Parse(d: int, p: int) -> float:
	if d > 10**19:
		raise ValueError("too many digits")

	# Estimate scaling constant e.
	e = 53 - d.bit_length() - math.floor(math.log2(10)*p)
	if e > 1074:
		e = 1074

	# Scale d * 10**p by 2**e.
	r = scale(d, e, p)

	# Round and correct extra bit or rollover.
	m = r.round()
	if m >= 2**53:
		m, e = (r>>1).round(), e - 1

	return pack(m, -e)

assert Parse(7205759403792795, 1) == 72057594037927950.0

# Testing

import re

# shortRef is a reference implementation of shortFormat.
# It parses the Python str(f) into (d, p).

# unStr parses a floating-point string into d, p.
def parseFloatString(s):
	floatRE = r"(\d+)\.?(\d*)e?([+\-]?\d*)"
	m = re.match(floatRE, s)
	d, p = (int(m[1]) * 10**len(m[2]) + int('0'+m[2])), int(float(m[3]+'.0'))-len(m[2])
	return d, p

# shortRef f is a reference implementation of Short, using str(f).
def shortRef(f):
	d, p = parseFloatString(str(f))
	while d != 0 and d%10 == 0:
		d, p = d//10, p+1
	return d, p

assert shortRef(1.234) == (1234, -3)

def readTestFile():
	ivyRE = r"\((\d+) ftoa ([^()]+)\) is (\d+) (-?\d+)"
	with open('../test.ivy', 'r') as file:
		for line in file:
			m = re.match(ivyRE, line)
			n = int(m[1])
			f = float.fromhex(m[2])
			want = (int(m[3]), int(m[4]))
			yield n, f, want

def testFixedWidth():
	total, bad, stopped = 0, 0, ''
	for n, f, want in readTestFile():
		if n > 18:
			continue
		have = FixedWidth(f, n)
		total += 1
		if have != want:
			print(f'FixedWidth({f}, {n}) = {have}, want {want}')
			bad += 1
			if bad > 100:
				stopped = ' (stopped early)'
				break
	if bad > 0:
		print(f'FAIL FixedWidth {bad}/{total} incorrect{stopped}')
	else:
		print(f'PASS FixedWidth')

def shortTests():
	# Powers of two.
	for e in range(-1074, 1024):
		yield math.ldexp(1, e)

	# Powers of ten.
	for p in range(-308, 308):
		if p >= 0:
			yield float(10**p)
		else:
			yield 1/10**p

	# Unique inputs in test file.
	last = None
	for _, f, _ in readTestFile():
		if f != last:
			last = f
			yield f

def allShortTests():
	for f in shortTests():
		yield f
		g = math.nextafter(f, float('-inf'))
		if g+g != g:  # not zero or infinity
			yield g
		g = math.nextafter(f, float('+inf'))
		if g+g != g:
			yield g

def testShort():
	total, bad, stopped = 0, 0, ''
	for f in allShortTests():
		if bad >= 100:
			stopped = ' (stopped early)'
			break
		have = Short(f)
		want = shortRef(f)
		total += 1
		if have != want:
			print(f'Short({f}) = {have}, want {want}')
			bad += 1
	if bad > 0:
		print(f'FAIL Short {bad}/{total} incorrect{stopped}')
	else:
		print(f'PASS Short')

def allParseTests():
	for f in allShortTests():
		d, p = shortRef(f)
		yield d, p, f
		d, p = parseFloatString(f'{f:.17g}')
		yield d, p, f
		d, p = parseFloatString(f'{f:.18g}')
		yield d, p, f
		d, p = parseFloatString(f'{f:.19g}')
		yield d, p, f

def testParse():
	total, bad, stopped = 0, 0, ''
	for d, p, want in allParseTests():
		if bad >= 100:
			stopped = ' (stopped early)'
			break
		have = Parse(d, p)
		total += 1
		if have != want:
			print(f'Parse({d}, {p}) = {have}, want {want}')
			bad += 1
	if bad > 0:
		print(f'FAIL Parse {bad}/{total} incorrect{stopped}')
	else:
		print(f'PASS Parse')

def runTests():
	global scale
	testRE = r"^test[A-Z]"
	scaleRE = r"^scale[0-9]+"
	tests = []
	scales = []
	for name, fn in globals().items():
		if re.match(testRE, name):
			tests.append(fn)
		if re.match(scaleRE, name):
			scales.append((name, fn))
	for name, sc in scales:
		scale = sc
		print(f'Testing {name}...')
		for fn in tests:
			fn()

if __name__ == '__main__':
	runTests()

