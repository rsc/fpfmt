// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package go125

import "math"

func ryu(tmp *[32]byte, f float64, prec int) int {
	flt := &float64info
	bits := math.Float64bits(f)
	exp := int(bits>>flt.mantbits) & (1<<flt.expbits - 1)
	mant := bits & (uint64(1)<<flt.mantbits - 1)

	switch exp {
	case 0:
		exp++
	default:
		mant |= uint64(1) << flt.mantbits
	}
	exp += flt.bias

	var d decimalSlice
	var buf [32]byte
	d.d = buf[:]
	ryuFtoaFixed64(&d, mant, exp-int(flt.mantbits), prec)
	return len(formatDigits(tmp[:0], false, false, d, prec-1, 'e'))
}

func ryuShort(tmp *[32]byte, f float64) int {
	flt := &float64info
	bits := math.Float64bits(f)
	exp := int(bits>>flt.mantbits) & (1<<flt.expbits - 1)
	mant := bits & (uint64(1)<<flt.mantbits - 1)

	switch exp {
	case 0:
		exp++
	default:
		mant |= uint64(1) << flt.mantbits
	}
	exp += flt.bias

	var d decimalSlice
	var buf [32]byte
	d.d = buf[:]
	ryuFtoaShortest(&d, mant, exp-int(flt.mantbits), flt)
	prec := max(d.nd-1, 0)
	return len(formatDigits(tmp[:0], false, false, d, prec, 'e'))
}

func BenchFixed(dst []byte, count int, fs []float64, digits int) {
	var tmp [32]byte
	i := 0
	for range count {
		for _, f := range fs {
			i = ryu(&tmp, f, digits)
		}
	}
	copy(dst, tmp[:i])
	dst[i] = 0
}

func BenchShort(dst []byte, count int, fs []float64) {
	var tmp [32]byte
	i := 0
	for range count {
		for _, f := range fs {
			i = ryuShort(&tmp, f)
		}
	}
	copy(dst, tmp[:i])
	dst[i] = 0
}

func unopt(tmp *[32]byte, f float64, prec int) int {
	flt := &float64info
	bits := math.Float64bits(f)
	exp := int(bits>>flt.mantbits) & (1<<flt.expbits - 1)
	mant := bits & (uint64(1)<<flt.mantbits - 1)

	switch exp {
	case 0:
		exp++
	default:
		mant |= uint64(1) << flt.mantbits
	}
	exp += flt.bias

	return len(bigFtoa(tmp[:0], prec-1, 'e', false, mant, exp, flt))
}

func BenchFixedUnopt(dst []byte, count int, fs []float64, prec int) {
	var tmp [32]byte
	i := 0
	for range count {
		for _, f := range fs {
			i = unopt(&tmp, f, prec)
		}
	}
	copy(dst, tmp[:i])
	dst[i] = 0
}

func BenchShortUnopt(dst []byte, count int, fs []float64) {
	var tmp [32]byte
	i := 0
	for range count {
		for _, f := range fs {
			i = unopt(&tmp, f, -1)
		}
	}
	copy(dst, tmp[:i])
	dst[i] = 0
}
