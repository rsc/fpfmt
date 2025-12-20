package dblconv

/*
#include "../bench.h"

void dblconvFixed(char *dst, double f, int prec);
void dblconvShort(char*, double);
double dblconvStrtod(char*, char*);

void
dblconvBenchFixed(char *dst, int count, double *f, int nf, int digits)
{
	benchFixed(dst, count, f, nf, digits, dblconvFixed);
}

void
dblconvBenchShort(char *dst, int count, double *f, int nf)
{
	benchShort(dst, count, f, nf, dblconvShort);
}

double
dblconvBenchParse(int count, char *s)
{
	return benchParse(count, s, dblconvStrtod);
}

*/
import "C"

import (
	"bytes"
	"unsafe"
)

func BenchFixed(dst []byte, count int, f []float64, digits int) {
	C.dblconvBenchFixed((*C.char)(unsafe.Pointer(&dst[0])), C.int(count), (*C.double)(&f[0]), C.int(len(f)), C.int(digits))
	fixup(dst)
}

func BenchShort(dst []byte, count int, f []float64) {
	C.dblconvBenchShort((*C.char)(unsafe.Pointer(&dst[0])), C.int(count), (*C.double)(&f[0]), C.int(len(f)))
	fixup(dst)
}

func BenchParse(count int, text []byte) float64 {
	return float64(C.dblconvBenchParse(C.int(count), (*C.char)(unsafe.Pointer(&text[0]))))
}

func fixup(b []byte) {
	end := bytes.IndexByte(b, 0)
	if b[0] == '0' && b[1] == '.' {
		i := 2
		for b[i] == '0' {
			i++
		}
		n := 1 + copy(b[1:], b[i:end])
		p := i - 1
		b[0] = b[1]
		b[1] = '.'
		if n == 2 {
			n--
		}
		b[n] = 'e'
		b[n+1] = '-'
		b[n+2] = '0' + byte(p/10)
		b[n+3] = '0' + byte(p%10)
		b[n+4] = 0
		return
	}
	i := bytes.IndexByte(b, 'e')
	if i < 0 {
		j := bytes.IndexByte(b, '.')
		p := 0
		if j < 0 {
			copy(b[2:], b[1:end])
			b[1] = '.'
			p = end - 1
			end++
		} else if j > 1 {
			copy(b[2:], b[1:j])
			b[1] = '.'
			p = j - 1
		}
		for end > 0 && b[end-1] == '0' {
			end--
		}
		if end > 0 && b[end-1] == '.' {
			end--
		}
		b[end] = 'e'
		b[end+1] = '+'
		b[end+2] = '0' + byte(p/10)
		b[end+3] = '0' + byte(p%10)
		b[end+4] = 0
		return
	}
	if i := end; i >= 2 && (b[i-2] == '-' || b[i-2] == '+') {
		b[i] = b[i-1]
		b[i-1] = '0'
		b[i+1] = 0
	}
}
