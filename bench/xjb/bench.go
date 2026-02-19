package xjb

/*
#include <stdint.h>

void xjbBenchShort(char*, int, double*, int);
*/
import "C"
import "unsafe"

func BenchShort(dst []byte, count int, f []float64) {
	C.xjbBenchShort((*C.char)(unsafe.Pointer(&dst[0])), C.int(count), (*C.double)(&f[0]), C.int(len(f)))
}
