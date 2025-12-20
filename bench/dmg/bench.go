package dmg

/*
#cgo CFLAGS: -DIEEE_8087 -DLong=int -w

#include <string.h>
#include <stdlib.h>
#include "../bench.h"

char *dtoa1991(double dd, int mode, int ndigits, int *decpt, int *sign, char **rve);
char *dtoa19970128(double dd, int mode, int ndigits, int *decpt, int *sign, char **rve);
char *dtoa20161215(double dd, int mode, int ndigits, int *decpt, int *sign, char **rve);
char *dtoa20170421(double dd, int mode, int ndigits, int *decpt, int *sign, char **rve);
char *dtoa20251117(double dd, int mode, int ndigits, int *decpt, int *sign, char **rve);

double strtod1991(const char*, char**);
double strtod19970128(const char*, char**);
double strtod20161215(const char*, char**);
double strtod20170421(const char*, char**);
double strtod20251117(const char*, char**);

static void
dmgFormat(int mode, char *buf, double f, int prec, char *(*dtoa)(double, int, int, int*, int*, char**))
{
	char *p;
	int exp, neg, ns;

	strcpy(buf+1, dtoa(f, mode, prec, &exp, &neg, 0));
	ns = strlen(buf+1);
	exp--;
	while(ns < prec)
		buf[1+ns++] = '0';
	buf[0] = buf[1];
	p = buf+ns;
	if(ns > 1) {
		buf[1] = '.';
		p++;
	}
	*p++ = 'e';
	if(exp<0) {
		*p++ = '-';
		exp = -exp;
	}else{
		*p++ = '+';
	}
	if(exp >= 100) {
		*p++ = (exp/100)+'0';
		*p++ = (exp/10)%10+'0';
		*p++ = exp%10+'0';
	} else {
		*p++ = (exp/10)+'0';
		*p++ = exp%10+'0';
	}
	*p = '\0';
}

#define DO(N) \
	static void fixed##N(char *dst, double f, int digits) { \
		dmgFormat(2, dst, f, digits, dtoa##N); \
	} \
	static void short##N(char *dst, double f) { \
		dmgFormat(0, dst, f, 0, dtoa##N); \
	} \
	static double parse##N(char *s, char *end) { \
		return strtod##N(s, &end); \
	} \
	void dmgBenchFixed##N(char *dst, int count, double *f, int nf, int digits) { \
		benchFixed(dst, count, f, nf, digits, fixed##N); \
	} \
	void dmgBenchShort##N(char *dst, int count, double *f, int nf) { \
		benchShort(dst, count, f, nf, short##N); \
	} \
	double dmgBenchParse##N(int count, char *s) { \
		return benchParse(count, s, parse##N); \
	}

DO(1991);
DO(19970128);
DO(20161215);
DO(20170421);
DO(20251117);

*/
import "C"
import "unsafe"

func BenchFixed1991(dst []byte, count int, f []float64, digits int) { C.dmgBenchFixed1991((*C.char)(unsafe.Pointer(&dst[0])), C.int(count), (*C.double)(&f[0]), C.int(len(f)), C.int(digits)) }
func BenchFixed19970128(dst []byte, count int, f []float64, digits int) { C.dmgBenchFixed19970128((*C.char)(unsafe.Pointer(&dst[0])), C.int(count), (*C.double)(&f[0]), C.int(len(f)), C.int(digits)) }
func BenchFixed20161215(dst []byte, count int, f []float64, digits int) { C.dmgBenchFixed20161215((*C.char)(unsafe.Pointer(&dst[0])), C.int(count), (*C.double)(&f[0]), C.int(len(f)), C.int(digits)) }
func BenchFixed20170421(dst []byte, count int, f []float64, digits int) { C.dmgBenchFixed20170421((*C.char)(unsafe.Pointer(&dst[0])), C.int(count), (*C.double)(&f[0]), C.int(len(f)), C.int(digits)) }
func BenchFixed20251117(dst []byte, count int, f []float64, digits int) { C.dmgBenchFixed20251117((*C.char)(unsafe.Pointer(&dst[0])), C.int(count), (*C.double)(&f[0]), C.int(len(f)), C.int(digits)) }

func BenchShort1991(dst []byte, count int, f []float64) { C.dmgBenchShort1991((*C.char)(unsafe.Pointer(&dst[0])), C.int(count), (*C.double)(&f[0]), C.int(len(f))) }
func BenchShort19970128(dst []byte, count int, f []float64) { C.dmgBenchShort19970128((*C.char)(unsafe.Pointer(&dst[0])), C.int(count), (*C.double)(&f[0]), C.int(len(f))) }
func BenchShort20161215(dst []byte, count int, f []float64) { C.dmgBenchShort20161215((*C.char)(unsafe.Pointer(&dst[0])), C.int(count), (*C.double)(&f[0]), C.int(len(f))) }
func BenchShort20170421(dst []byte, count int, f []float64) { C.dmgBenchShort20170421((*C.char)(unsafe.Pointer(&dst[0])), C.int(count), (*C.double)(&f[0]), C.int(len(f))) }
func BenchShort20251117(dst []byte, count int, f []float64) { C.dmgBenchShort20251117((*C.char)(unsafe.Pointer(&dst[0])), C.int(count), (*C.double)(&f[0]), C.int(len(f))) }

func BenchParse1991(count int, text []byte) float64 { return float64(C.dmgBenchParse1991(C.int(count), (*C.char)(unsafe.Pointer(&text[0])))) }
func BenchParse19970128(count int, text []byte) float64 { return float64(C.dmgBenchParse19970128(C.int(count), (*C.char)(unsafe.Pointer(&text[0])))) }
func BenchParse20161215(count int, text []byte) float64 { return float64(C.dmgBenchParse20161215(C.int(count), (*C.char)(unsafe.Pointer(&text[0])))) }
func BenchParse20170421(count int, text []byte) float64 { return float64(C.dmgBenchParse20170421(C.int(count), (*C.char)(unsafe.Pointer(&text[0])))) }
func BenchParse20251117(count int, text []byte) float64 { return float64(C.dmgBenchParse20251117(C.int(count), (*C.char)(unsafe.Pointer(&text[0])))) }
