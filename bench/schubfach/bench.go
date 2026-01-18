package schubfach

/*
#cgo CFLAGS: -Wall -O2

#include <stdint.h>

void schubfachBenchShort(char*, int, double*, int);
void schubfachBenchShortRaw(uint64_t*, int64_t*, int, double*, int);
*/
import "C"

import (
	"unsafe"
)

func BenchShort(dst []byte, count int, f []float64) {
	C.schubfachBenchShort((*C.char)(unsafe.Pointer(&dst[0])), C.int(count), (*C.double)(&f[0]), C.int(len(f)))
	//fixup(dst)
}

func BenchShortRaw(dp *uint64, pp *int64, count int, f []float64) {
	C.schubfachBenchShortRaw((*C.uint64_t)(dp), (*C.int64_t)(pp), C.int(count), (*C.double)(&f[0]), C.int(len(f)))
}
