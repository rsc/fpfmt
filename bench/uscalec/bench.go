package uscalec

/*
#include <stdint.h>

void uscalecBenchFixed(char*, int, double*, int, int);
void uscalecBenchShort(char*, int, double*, int);
void uscalecBenchShortRaw(uint64_t*, int64_t*, int, double*, int);
double uscalecBenchParseRaw(int, int64_t*, int);
double uscalecBenchParse(int, char*);
*/
import "C"
import "unsafe"

func BenchFixed(dst []byte, count int, f []float64, digits int) {
	C.uscalecBenchFixed((*C.char)(unsafe.Pointer(&dst[0])), C.int(count), (*C.double)(&f[0]), C.int(len(f)), C.int(digits))
}

func BenchShort(dst []byte, count int, f []float64) {
	C.uscalecBenchShort((*C.char)(unsafe.Pointer(&dst[0])), C.int(count), (*C.double)(&f[0]), C.int(len(f)))
}

func BenchShortRaw(dp *uint64, pp *int64, count int, f []float64) {
	C.uscalecBenchShortRaw((*C.uint64_t)(dp), (*C.int64_t)(pp), C.int(count), (*C.double)(&f[0]), C.int(len(f)))
}

func BenchParse(count int, text []byte) float64 {
	return float64(C.uscalecBenchParse(C.int(count), (*C.char)(unsafe.Pointer(&text[0]))))
}

func BenchParseRaw(count int, raw []int64) float64 {
	return float64(C.uscalecBenchParseRaw(C.int(count), (*C.int64_t)(&raw[0]), C.int(len(raw))))
}
