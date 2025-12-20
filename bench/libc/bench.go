package libc

/*
#include <stdio.h>
#include <stdlib.h>
#include "../bench.h"

static void
libcFixed(char *dst, double f, int prec)
{
	snprintf(dst, 100, "%.*e", prec-1, f);
}

void
libcBenchFixed(char *dst, int count, double *f, int nf, int digits)
{
	benchFixed(dst, count, f, nf, digits, libcFixed);
}

static double
libcStrtod(char *s, char *e)
{
	return strtod(s, 0);
}

double
libcBenchParse(int count, char *s)
{
	return benchParse(count, s, libcStrtod);
}
*/
import "C"
import "unsafe"

func BenchFixed(dst []byte, count int, f []float64, digits int) {
	C.libcBenchFixed((*C.char)(unsafe.Pointer(&dst[0])), C.int(count), (*C.double)(&f[0]), C.int(len(f)), C.int(digits))
}

func BenchParse(count int, text []byte) float64 {
	return float64(C.libcBenchParse(C.int(count), (*C.char)(unsafe.Pointer(&text[0]))))
}
