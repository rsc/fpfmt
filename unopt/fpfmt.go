// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package fpfmt implements floating point formatting
// algorithm and benchmarks to compare with other algorithms.
package fpfmt

import (
	"fmt"
	"math"
	"math/bits"
)

// bool2 converts b to an integer: 1 for true, 0 for false.
func bool2[T ~int | ~uint64](b bool) T {
	if b {
		return 1
	}
	return 0
}

// unpack64 returns m, e such that f = m * 2**e.
// The caller is expected to have handled 0, NaN, and ±Inf already.
// To unpack a float32 f, use unpack64(float64(f)).
func unpack64(f float64) (uint64, int) {
	const shift = 64 - 53
	const minExp = -(1074 + shift)
	b := math.Float64bits(f)
	m := 1<<63 | (b&(1<<52-1))<<shift
	e := int((b >> 52) & (1<<shift - 1))
	if e == 0 {
		m &^= 1 << 63
		e = minExp
		s := 64 - bits.Len64(m)
		return m << s, e - s
	}
	return m, (e - 1) + minExp
}

// pack64 takes m, e and returns f = m * 2**e.
// It assumes the caller has provided a 53-bit mantissa m
// and an exponent that is in range for the mantissa.
func pack64(m uint64, e int) float64 {
	if m&(1<<52) == 0 {
		return math.Float64frombits(m)
	}
	return math.Float64frombits(m&^(1<<52) | uint64(1075+e)<<52)
}

// An unrounded represents an unrounded value.
type unrounded uint64

func unround(x float64) unrounded {
	return unrounded(math.Floor(4*x)) | bool2[unrounded](math.Floor(4*x) != 4*x)
}

func (u unrounded) String() string {
	return fmt.Sprintf("⟨%d.%d%s⟩", u>>2, 5*((u>>1)&1), "+"[1-u&1:])
}

func (u unrounded) floor() uint64         { return uint64((u + 0) >> 2) }
func (u unrounded) roundHalfDown() uint64 { return uint64((u + 1) >> 2) }
func (u unrounded) round() uint64         { return uint64((u + 1 + (u>>2)&1) >> 2) }
func (u unrounded) roundHalfUp() uint64   { return uint64((u + 2) >> 2) }
func (u unrounded) ceil() uint64          { return uint64((u + 3) >> 2) }
func (u unrounded) nudge(δ int) unrounded { return u + unrounded(δ) }

func (u unrounded) div(d uint64) unrounded {
	x := uint64(u)
	return unrounded(x/d) | u&1 | bool2[unrounded](x%d != 0)
}

func (u unrounded) rsh(s int) unrounded {
	return u>>s | u&1 | bool2[unrounded](u&((1<<s)-1) != 0)
}

// log10Pow2(x) returns ⌊log₁₀ 2**x⌋ = ⌊x * log₁₀ 2⌋.
func log10Pow2(x int) int {
	// log₁₀ 2 ≈ 0.30102999566 ≈ 78913 / 2^18
	return (x * 78913) >> 18
}

// log2Pow10(x) returns ⌊log₂ 10**x⌋ = ⌊x * log₂ 10⌋.
func log2Pow10(x int) int {
	// log₂ 10 ≈ 3.32192809489 ≈ 108853 / 2^15
	return (x * 108853) >> 15
}

var uint64pow10 = [...]uint64{
	1, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9,
	1e10, 1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19,
}

// FixedWidth returns the n-digit decimal form of f as d * 10**p.
// n can be at most 18.
func FixedWidth(f float64, n int) (d uint64, p int) {
	if n > 18 {
		panic("too many digits")
	}
	m, e := unpack64(f)
	p = n - 1 - log10Pow2(e+63)
	u := uscale(m, prescale(e, p, log2Pow10(p)))
	d = u.round()
	if d >= uint64pow10[n] {
		d, p = u.div(10).round(), p-1
	}
	return d, -p
}

// Parse rounds d * 10**p to the nearest float64 f.
// d can have at most 19 digits.
func Parse(d uint64, p int) float64 {
	if d > 1e19 {
		panic("too many digits")
	}
	b := bits.Len64(d)
	e := min(1074, 53-b-log2Pow10(p))
	u := uscale(d<<(64-b), prescale(e-(64-b), p, log2Pow10(p)))
	m := u.round()
	if m >= 1<<53 {
		m, e = u.rsh(1).round(), e-1
	}
	return pack64(m, -e)
}

// ParseText parses a decimal string s
// and returns the nearest floating point value.
// It returns 0, false if the input s is malformed.
func ParseText(s []byte) (f float64, ok bool) {
	isDigit := func(c byte) bool { return c-'0' <= 9 }

	// Read digits.
	const maxDigits = 19
	d := uint64(0) // decimal value of digits
	frac := 0      // count of digits after decimal point
	i := 0
	for ; i < len(s) && isDigit(s[i]); i++ {
		d = d*10 + uint64(s[i]) - '0'
	}
	if i > maxDigits {
		return // too many digits
	}
	if i < len(s) && s[i] == '.' {
		i++
		for ; i < len(s) && isDigit(s[i]); i++ {
			d = d*10 + uint64(s[i]) - '0'
			frac++
		}
		if i == 1 || i > maxDigits+1 {
			return // no digits or too many digits
		}
	}
	if i == 0 {
		return // no digits
	}

	// Read exponent.
	p := 0
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		sign := +1
		if i < len(s) {
			if s[i] == '-' {
				sign = -1
				i++
			} else if s[i] == '+' {
				i++
			}
		}
		if i >= len(s) || len(s)-i > 3 {
			return // missing or too large exponent
		}
		for ; i < len(s) && isDigit(s[i]); i++ {
			p = p*10 + int(s[i]) - '0'
		}
		p *= sign
	}
	if i != len(s) {
		return // junk on end
	}
	return Parse(d, p-frac), true
}

// Short computes the shortest formatting of f,
// using as few digits as possible that will still round trip
// back to the original float64.
func Short(f float64) (d uint64, p int) {
	const minExp = -1085

	m, e := unpack64(f)

	var min uint64
	z := 11 // extra zero bits at bottom of m; 11 for 53-bit m
	if m == 1<<63 && e > minExp {
		p = -skewed(e + z)
		min = m - 1<<(z-2) // min = m - 1/4 * 2**(e+z)
	} else {
		if e < minExp {
			z = 11 + (minExp - e)
		}
		p = -log10Pow2(e + z)
		min = m - 1<<(z-1) // min = m - 1/2 * 2**(e+z)
	}
	max := m + 1<<(z-1) // max = m + 1/2 * 2**(e+z)
	odd := int(m>>z) & 1

	pre := prescale(e, p, log2Pow10(p))
	dmin := uscale(min, pre).nudge(+odd).ceil()
	dmax := uscale(max, pre).nudge(-odd).floor()

	if d = dmax / 10; d*10 >= dmin {
		return trimZeros(d, -(p - 1))
	}
	if d = dmin; d < dmax {
		d = uscale(m, pre).round()
	}
	return d, -p
}

// skewed computes the skewed footprint of m * 2**e,
// which is ⌊log₁₀ 3/4 * 2**e⌋ = ⌊e*(log₁₀ 2)-(log₁₀ 4/3)⌋.
func skewed(e int) int {
	return (e*631305 - 261663) >> 21
}

// trimZeros removes trailing zeros from x * 10**p.
// If x ends in k zeros, trimZeros returns x/10**k, p+k.
// It assumes that x ends in at most 16 zeros.
func trimZeros(x uint64, p int) (uint64, int) {
	if x%10 != 0 {
		return x, p
	}
	x /= 10
	p += 1

	if x%100000000 == 0 {
		x /= 100000000
		p += 8
	}
	if x%10000 == 0 {
		x /= 10000
		p += 4
	}
	if x%100 == 0 {
		x /= 100
		p += 2
	}
	if x%10 == 0 {
		x /= 10
		p += 1
	}
	return x, p
}

// A scaler holds derived scaling constants for a given e, p pair.
type scaler struct {
	pm pmHiLo
	s  int
}

// A pmHiLo represents hi<<64 + lo.
type pmHiLo struct {
	hi uint64
	lo uint64
}

// prescale returns the scaling constants for e, p.
// lp must be log2Pow10(p).
func prescale(e, p, lp int) scaler {
	if p < pow10Min || p > pow10Max {
		panic("prescale")
	}
	return scaler{pm: pow10Tab[p-pow10Min], s: -(e + lp + 3)}
}

/*
// uscale0 returns unround(x * 2**e * 10**p).
// The caller should pass c = prescale(e, p)
// and must have left-justified x so its high bit is set.
func uscale0(x uint64, c scaler) unrounded {
	b := func() *big.Int { return new(big.Int) }
	bn := func(x uint64) *big.Int { return b().SetUint64(x) }
	mul := func(factors ...*big.Int) *big.Int {
		b := bn(1)
		for _, f := range factors {
			b = b.Mul(b, f)
		}
		return b
	}
	pow := func(base uint64, exp int) *big.Int {
		return b().Exp(bn(base), bn(uint64(exp)), nil)
	}

	num := mul(bn(4), bn(x), pow(2, max(0, c.e)), pow(10, max(0, c.p)))
	denom := mul(pow(2, max(0, -c.e)), pow(10, max(0, -c.p)))
	r := new(big.Rat).SetFrac(num, denom)
	return unrounded(b().Div(r.Num(), r.Denom()).Uint64()) | bool2[unrounded](!r.IsInt())
}
*/

// uscale returns unround(x * 2**e * 10**p).
// The caller should pass c = prescale(e, p, log2Pow10(p))
// and should have left-justified x so its high bit is set.
func uscale(x uint64, c scaler) unrounded {
	hi, mid := bits.Mul64(x, c.pm.hi)
	mid2, _ := bits.Mul64(x, c.pm.lo)
	mid, carry := bits.Add64(mid, mid2, 0)
	hi += carry
	sticky := bool2[unrounded](mid != 0 || hi&((1<<c.s)-1) != 0)
	return unrounded(hi>>c.s) | sticky
}

// Fmt formats d, p into s in exponential notation.
// The caller must pass nd set to the number of digits in d.
// It returns the number of bytes written to s.
func Fmt(s []byte, d uint64, p, nd int) int {
	// Put digits into s, leaving room for decimal point.
	formatBase10(s[1:nd+1], d)
	p += nd - 1

	// Move first digit up and insert decimal point.
	s[0] = s[1]
	n := nd
	if n > 1 {
		s[1] = '.'
		n++
	}

	// Add 2- or 3-digit exponent.
	s[n] = 'e'
	if p < 0 {
		s[n+1] = '-'
		p = -p
	} else {
		s[n+1] = '+'
	}
	if p < 100 {
		s[n+2] = i2a[p*2]
		s[n+3] = i2a[p*2+1]
		return n + 4
	}
	s[n+2] = byte('0' + p/100)
	s[n+3] = i2a[(p%100)*2]
	s[n+4] = i2a[(p%100)*2+1]
	return n + 5
}

// Digits returns the number of decimal digits in d.
func Digits(d uint64) int {
	nd := log10Pow2(bits.Len64(d))
	return nd + bool2[int](d >= uint64pow10[nd])
}

// i2a is the formatting of 00..99 concatenated,
// a lookup table for formatting [0, 99].
const i2a = "00010203040506070809" +
	"10111213141516171819" +
	"20212223242526272829" +
	"30313233343536373839" +
	"40414243444546474849" +
	"50515253545556575859" +
	"60616263646566676869" +
	"70717273747576777879" +
	"80818283848586878889" +
	"90919293949596979899"

// formatBase10 formats the decimal representation of u into a.
// The caller is responsible for ensuring that a is big enough to hold u.
// If a is too big, leading zeros will be filled in as needed.
func formatBase10(a []byte, u uint64) {
	for nd := len(a) - 1; nd >= 0; nd-- {
		a[nd] = byte(u%10 + '0')
		u /= 10
	}
}
