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

func bool2[T ~int | ~uint64](b bool) T {
	if b {
		return 1
	}
	return 0
}

func unpack64(f float64) (uint64, int) {
	b := math.Float64bits(f)
	m := 1<<63 | (b&(1<<52-1))<<11
	e := int((b>>52)&(1<<11-1)) - 1086
	if e == -1086 {
		m &^= 1 << 63
		e = -1085
		s := 64 - bits.Len64(m)
		m, e = m<<s, e-s
	}
	return m, e
}

type unrounded uint64

func unround(x float64) unrounded {
	return unrounded(math.Floor(4*x)) | bool2[unrounded](math.Floor(4*x) != 4*x)
}

func (u unrounded) String() string {
	return fmt.Sprintf("⟨%d.%d%s⟩", u>>2, 5*((u>>1)&1), "+"[1-u&1:])
}

func (r unrounded) floor() uint64         { return uint64((r + 0) >> 2) }
func (r unrounded) roundHalfDown() uint64 { return uint64((r + 1) >> 2) }
func (r unrounded) round() uint64         { return uint64((r + 1 + (r>>2)&1) >> 2) }
func (r unrounded) roundHalfUp() uint64   { return uint64((r + 2) >> 2) }
func (r unrounded) ceil() uint64          { return uint64((r + 3) >> 2) }
func (r unrounded) nudge(δ int) unrounded { return r + unrounded(δ) }

func (r unrounded) div(d uint64) unrounded {
	u := uint64(r)
	return unrounded(u/d) | r&1 | bool2[unrounded](u%d != 0)
}

func (r unrounded) rsh(s int) unrounded {
	return r>>s | r&1 | bool2[unrounded](r&((1<<s)-1) != 0)
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

// FixedWidth returns the n-digit decimal form of f as d * 10^p.
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

// FixedWidth returns the n-digit decimal form of f as d * 10^p.
func FixedWidthTrunc(f float64, n int) (d uint64, p int) {
	if n > 18 {
		panic("too many digits")
	}
	m, e := unpack64(f)
	p = n - 1 - log10Pow2(e+63)
	u := uscaleTrunc(m, prescaleTrunc(e, p))
	d = u.round()
	if d >= uint64pow10[n] {
		d, p = u.div(10).round(), p-1
	}
	return d, -p
}

// Parse rounds d * 10^p to the nearest float64 f.
// d can have at most 19 digits.
func xParse(d uint64, p int) float64 {
	if d > 1e19 {
		panic("too many digits")
	}
	b := bits.Len64(d)
	e := 53 - b - log2Pow10(p)
	if e > 1074 {
		e = 1074
	}
	u := uscale(d<<(64-b), prescale(e-(64-b), p, log2Pow10(p)))
	m := u.round()
	if m >= 1<<53 {
		m, e = u.rsh(1).round(), e-1
	}
	if m&(1<<52) == 0 {
		return math.Float64frombits(m)
	}
	return math.Float64frombits((m &^ (1 << 52)) | uint64(1075-e)<<52)
}

func unmin(x uint64) unrounded {
	return unrounded(x<<2 - 2)
}

func pack64(m uint64, e int) float64 {
	if m&(1<<52) == 0 {
		return math.Float64frombits(m)
	}
	return math.Float64frombits((m &^ (1 << 52)) | uint64(1075+e)<<52)
}

func Parse(d uint64, p int) float64 {
	if d > 1e19 {
		panic("too many digits")
	}
	b := bits.Len64(d)
	lp := log2Pow10(p)
	e := min(53-b-lp, 1074)
	u := uscale(d<<(64-b), prescale(e-(64-b), p, lp))
	s := bool2[int](u >= unmin(1<<53))
	u = (u >> s) | u&1
	e = e - s
	m := u.round()
	return pack64(m, -e)
}

func parseText(s []byte) float64 {
	d := uint64(0)
	dp := 0
	i := 0
	for ; i < len(s) && s[i] != 'e'; i++ {
		if s[i] == '.' {
			dp = i + 1
			continue
		}
		d = d*10 + uint64(s[i]) - '0'
	}
	if dp > 0 {
		dp = i - dp
	}
	p := 0
	if i < len(s) {
		sign := +1
		i++
		if i < len(s) {
			if s[i] == '-' {
				sign = -1
				i++
			} else if s[i] == '+' {
				i++
			}
		}
		for ; i < len(s); i++ {
			p = p*10 + int(s[i]) - '0'
		}
		p *= sign
	}
	if d > 1e19 {
		println("PARSE", s, d)
	}
	return Parse(d, p-dp)
}

// Short computes the shortest formatting of f,
// using as few digits as possible that will still round trip
// back to the original float64.
func Short(f float64) (d uint64, p int) {
	m, e := unpack64(f)

	var min uint64
	b := 11
	if m == 1<<63 && e > -1085 {
		p = -skewed(e + b)
		min = m - 1<<(b-2)
	} else {
		if e < -1085 {
			b = 11 + (-1085 - e)
		}
		p = -log10Pow2(e + b)
		min = m - 1<<(b-1)
	}
	max := m + 1<<(b-1)

	odd := int(m>>b) & 1
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

// ShortTrunc computes the shortest formatting of f,
// using as few digits as possible that will still round trip
// back to the original float64.
func ShortTrunc(f float64) (d uint64, p int) {
	m, e := unpack64(f)

	b := 11
	var min uint64
	var b1, b2 int
	if m == 1<<63 && e > -1085 {
		p = -skewed(e + b)
		min = m - 1<<(b-2)
		b1 = b - 2
		b2 = b - 1
	} else {
		if e < -1085 {
			b = 11 + (-1085 - e)
		}
		p = -log10Pow2(e + b)
		min = m - 1<<(b-1)
		b1 = b - 1
		b2 = b - 1
	}

	odd := int(m>>b) & 1
	pre := prescaleTrunc(e, p)
	umin, um, umax := uscale3(min, b1, b2, pre)
	dmin := umin.nudge(+odd).ceil()
	dmax := umax.nudge(-odd).floor()

	d0 := dmax / 10 * 10
	if d0 >= dmin {
		return trimZeros(dmax/10, -(p - 1))
	}
	if dmin == dmax {
		return dmin, -p
	}
	return um.round(), -p
}

// skewed computes the skewed footprint of m * 2**e,
// which is ⌊log₁₀ 3/4 * 2**e⌋ = ⌊e*(log₁₀ 2)-(log₁₀ 4/3)⌋.
// It is valid for e ∈ [-2985, 2936].
func skewed(e int) int {
	return (e*631305 - 261663) >> 21
}

func trimZeros(x uint64, p int) (uint64, int) {
	const (
		maxUint64 = ^uint64(0)

		div1e8m  = 0xc767074b22e90e21
		div1e8le = maxUint64 / 100000000

		div1e4m  = 0xd288ce703afb7e91
		div1e4le = maxUint64 / 10000

		div1e2m  = 0x8f5c28f5c28f5c29
		div1e2le = maxUint64 / 100

		div1e1m  = 0xcccccccccccccccd
		div1e1le = maxUint64 / 10
	)

	// Cut 1 zero, or else return.
	if d := bits.RotateLeft64(x*div1e1m, -1); d <= div1e1le {
		x = d
		p += 1
	} else {
		return x, p
	}

	// Cut 8 zeros, then 4, then 2, then 1.
	if d := bits.RotateLeft64(x*div1e8m, -8); d <= div1e8le {
		x = d
		p += 8
	}
	if d := bits.RotateLeft64(x*div1e4m, -4); d <= div1e4le {
		x = d
		p += 4
	}
	if d := bits.RotateLeft64(x*div1e2m, -2); d <= div1e2le {
		x = d
		p += 2
	}
	if d := bits.RotateLeft64(x*div1e1m, -1); d <= div1e1le {
		x = d
		p += 1
	}
	return x, p
}

// A pmHiLo represents hi<<64 - lo.
type pmHiLo struct {
	hi uint64
	lo uint64
}

type scalers struct {
	pmHi uint64
	pmLo uint64
	s    int
}

// prescale
func prescale(e, p, lp int) scalers {
	if p < pow10Min || p > pow10Max {
		panic("prescale")
	}
	pm := pow10Tab[p-pow10Min]
	s := -(e + lp + 3)
	return scalers{pm.hi, pm.lo, s}
}

// uscale returns unroundedOf(x * 2**e * 10**p).
// The caller should pass c = prescale(e, p)
// and must have left-justified x so its high bit is set.
func uscale(x uint64, c scalers) unrounded {
	hi, mid1 := bits.Mul64(x, c.pmHi)
	sticky := uint64(1)
	if hi&(1<<(c.s&63)-1) == 0 {
		mid2, _ := bits.Mul64(x, c.pmLo)
		sticky = bool2[uint64](mid1-mid2 > 1)
		hi -= bool2[uint64](mid1 < mid2)
	}
	return unrounded(hi>>c.s | sticky)
}

// prescaleTrunc
func prescaleTrunc(e, p int) scalers {
	if p < pow10Min || p > pow10Max {
		panic("prescale")
	}
	pm := pow10Tab[p-pow10Min]
	s := -(e + log2Pow10(p) + 3)
	if pm.lo != 0 {
		pm.hi--
		pm.lo = -pm.lo
	}
	return scalers{pm.hi, pm.lo, s}
}

// uscale returns unroundedOf(x * 2**e * 10**p).
// The caller should pass c = prescale(e, p)
// and must have left-justified x so its high bit is set.
func uscaleTrunc(x uint64, c scalers) unrounded {
	const N = 11
	const N2 = 15
	if x&(1<<N) != 0 {
		_, dx, _ := uscale3(x-(1<<N), N, 0, c)
		dx2, _, _ := uscale3(x, 0, 0, c)
		if dx != dx2 {
			fmt.Printf("bad uscale3: x=%#x s=%d pm=%#x %#x have %#x want %#x\n", x, c.s, c.pmHi, c.pmLo, uint64(dx), uint64(dx2))
			panic("uscale3")
		}
		if x&(1<<N2) != 0 {
			dx1a, dx1b, dx3 := uscale3(x-(1<<N)-(1<<N2), N, N2, c)
			if dx3 != dx {
				fmt.Printf("bad uscale3x: x=%#x s=%d pm=%#x %#x have %#x %#x %#x want %#x\n", x, c.s, c.pmHi, c.pmLo, uint64(dx1a), uint64(dx1b), uint64(dx3), uint64(dx))
			}
		}
		return dx
	}
	dx, _, _ := uscale3(x, 0, 0, c)
	return dx
}

func uscale3(x uint64, b1, b2 int, c scalers) (dx, dxb1, dxb2 unrounded) {
	hi, mid := bits.Mul64(x, c.pmHi)
	hi1 := hi + c.pmHi>>(64-b1)
	hi2 := hi1 + c.pmHi>>(64-b2)
	if (hi+1)&(1<<c.s-1) > 1 && (hi1+2)&(1<<c.s-1) > 2 && (hi2+3)&(1<<c.s-1) > 3 {
		dx = unrounded(hi>>c.s | 1)
		dxb1 = unrounded(hi1>>c.s | 1)
		dxb2 = unrounded(hi2>>c.s | 1)
		return
	}

	midX, lo := bits.Mul64(x, c.pmLo)
	mid, carry := bits.Add64(mid, midX, 0)
	hi += carry
	dx = unrounded(hi>>c.s | bool2[uint64](hi&(1<<c.s-1)|mid != 0))
	//fmt.Printf("\nstep1 %#x %#x %#x => %#x\n", hi, mid, lo, uint64(dx))

	lo1, carry1 := bits.Add64(lo, c.pmLo<<b1, 0)
	mid1, carry1m := bits.Add64(mid, c.pmHi<<b1|c.pmLo>>(64-b1), carry1)
	hi1 = hi1 + carry + carry1m
	dxb1 = unrounded(hi1>>c.s | bool2[uint64](hi1&(1<<c.s-1)|mid1 != 0))
	//fmt.Printf("step2 %#x %#x %#x => %#x\n", hi, mid, lo, uint64(dxb1))

	_, carry2 := bits.Add64(lo1, c.pmLo<<b2, 0)
	mid2, carry2m := bits.Add64(mid1, c.pmHi<<b2|c.pmLo>>(64-b2), carry2)
	hi2 = hi2 + carry + carry1m + carry2m
	dxb2 = unrounded(hi2>>c.s | bool2[uint64](hi2&(1<<c.s-1)|mid2 != 0))
	//fmt.Printf("step3 %#x %#x %#x => %#x\n", hi, mid, lo, uint64(dxb2))
	return
}

// smalls is the formatting of 00..99 concatenated,
// a lookup table for formatting [0, 99].
const smalls = "00010203040506070809" +
	"10111213141516171819" +
	"20212223242526272829" +
	"30313233343536373839" +
	"40414243444546474849" +
	"50515253545556575859" +
	"60616263646566676869" +
	"70717273747576777879" +
	"80818283848586878889" +
	"90919293949596979899"

func efmt(dst []byte, dm uint64, dp int, nd int) int {
	formatBase10(dst[1:nd+1], dm)
	dp += nd - 1
	dst[0] = dst[1]
	n := nd
	if n > 1 {
		dst[1] = '.'
		n++
	}
	dst[n] = 'e'
	if dp < 0 {
		dst[n+1] = '-'
		dp = -dp
	} else {
		dst[n+1] = '+'
	}
	if dp < 100 {
		dst[n+2] = smalls[dp*2]
		dst[n+3] = smalls[dp*2+1]
		return n + 4
	}
	dst[n+2] = byte('0' + dp/100)
	dst[n+3] = smalls[(dp%100)*2]
	dst[n+4] = smalls[(dp%100)*2+1]
	return n + 5
}

func countDigits(d uint64) int {
	nd := log10Pow2(bits.Len64(d))
	return nd + bool2[int](d >= uint64pow10[nd])
}

func AppendFloat(dst []byte, f float64, fmt byte, prec, bitSize int) []byte {
	var buf [24]byte
	var d uint64
	var p, nd int
	if prec < 0 {
		d, p = Short(f)
		nd = countDigits(d)
	} else {
		d, p = FixedWidth(f, prec)
		nd = prec
	}
	i := efmt(buf[:], d, p, nd)
	return append(dst, buf[:i]...)
}

const host64bit = bits.UintSize == 64

// formatBase10 formats the decimal representation of u into the tail of a
// and returns the offset of the first byte written to a. That is, after
//
//	i := formatBase10(a, u)
//
// the decimal representation is in a[i:].
func formatBase10(a []byte, u uint64) {
	nd := len(a)
	for nd >= 9 {
		x := u % 1e8
		u /= 1e8
		y := x % 1e4
		x /= 1e4
		x1, x0 := (x/100)*2, (x%100)*2
		y1, y0 := (y/100)*2, (y%100)*2
		a[nd-1] = smalls[y0+1]
		a[nd-2] = smalls[y0]
		a[nd-3] = smalls[y1+1]
		a[nd-4] = smalls[y1]
		a[nd-5] = smalls[x0+1]
		a[nd-6] = smalls[x0]
		a[nd-7] = smalls[x1+1]
		a[nd-8] = smalls[x1]
		nd -= 8
	}

	d := uint32(u)
	for nd >= 5 {
		x := d % 1e4
		d /= 1e4
		x1, x0 := (x/100)*2, (x%100)*2
		a[nd-1] = smalls[x0+1]
		a[nd-2] = smalls[x0]
		a[nd-3] = smalls[x1+1]
		a[nd-4] = smalls[x1]
		nd -= 4
	}
	if nd >= 3 {
		x := (d % 1e2) * 2
		d /= 1e2
		a[nd-1] = smalls[x+1]
		a[nd-2] = smalls[x]
		nd -= 2
	}
	if nd >= 2 && d < 100 {
		a[nd-1] = smalls[d*2+1]
		a[nd-2] = smalls[d*2]
		return
	}
	a[0] = byte('0' + d)
}
