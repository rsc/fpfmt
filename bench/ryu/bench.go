package ryu

/*
#cgo CFLAGS: -I..

#include <stdint.h>
#include <string.h>
#include "ryu/ryu.h"
#include "../bench.h"

static void
fixed(char *dst, double f, int digits)
{
	d2exp_buffered(f, digits-1, dst);
}

void
ryuBenchFixed(char *dst, int count, double *f, int nf, int digits)
{
	benchFixed(dst, count, f, nf, digits, fixed);
}

static void
short1(char *dst, double f)
{
	d2s_buffered(f, dst);
}

void
ryuBenchShort(char *dst, int count, double *f, int nf)
{
	benchShort(dst, count, f, nf, short1);
}

void d2s_raw(double, uint64_t*, int64_t*);

static void
shortRaw(uint64_t *dp, int64_t *pp, double f)
{
	d2s_raw(f, dp, pp);
}

void
ryuBenchShortRaw(uint64_t *dp, int64_t *pp, int count, double *f, int nf)
{
	benchShortRaw(dp, pp, count, f, nf, shortRaw);
}
*/
import "C"
import (
	"bytes"
	"unsafe"
)

func BenchFixed(dst []byte, count int, f []float64, digits int) {
	C.ryuBenchFixed((*C.char)(unsafe.Pointer(&dst[0])), C.int(count), (*C.double)(&f[0]), C.int(len(f)), C.int(digits))
}

func BenchShort(dst []byte, count int, f []float64) {
	C.ryuBenchShort((*C.char)(unsafe.Pointer(&dst[0])), C.int(count), (*C.double)(&f[0]), C.int(len(f)))
	fixup(dst)
}

func BenchShortRaw(dp *uint64, pp *int64, count int, f []float64) {
	C.ryuBenchShortRaw((*C.uint64_t)(dp), (*C.int64_t)(pp), C.int(count), (*C.double)(&f[0]), C.int(len(f)))
}

func fixup(b []byte) {
	for i,c:=range b{
		if c == 0 {
			b[i] = 'e'
			b[i+1] = '+'
			b[i+2] = '0'
			b[i+3] = '0'
			b[i+4] = 0
			break
		}
		if c == 'E' {
			b[i] = 'e'
			break
		}
	}
	i := bytes.IndexByte(b, 0)
	if i >= 2 && (b[i-2] == 'e') {
		b[i+2] = 0
		b[i+1] = b[i-1]
		b[i] = '0'
		b[i-1] = '+'
		return
	}
	if i >= 2 && (b[i-2] == '-' || b[i-2] == '+') {
		b[i] = b[i-1]
		b[i-1] = '0'
		b[i+1] = 0
		return
	}
	if i >= 3 && b[i-3] == 'e' && b[i-2] != '+' && b[i-2] != '-' {
		b[i+1] = 0
		b[i] = b[i-1]
		b[i-1] = b[i-2]
		b[i-2] = '+'
		return
	}
	if i >= 4 && b[i-4] == 'e' && b[i-3] != '+' && b[i-3] != '-' {
		b[i+1] = 0
		b[i] = b[i-1]
		b[i-1] = b[i-2]
		b[i-2] = b[i-3]
		b[i-3] = '+'
		return
	}
}
