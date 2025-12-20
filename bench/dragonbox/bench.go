package dragonbox

/*
#cgo CFLAGS: -Wno-shift-count-overflow

#include <stdint.h>

void dragonboxBenchShort(char*, int, double*, int);
void dragonboxBenchShortRaw(uint64_t*, int64_t*, int, double*, int);
*/
import "C"

import (
	"bytes"
	"unsafe"
)

func BenchShort(dst []byte, count int, f []float64) {
	C.dragonboxBenchShort((*C.char)(unsafe.Pointer(&dst[0])), C.int(count), (*C.double)(&f[0]), C.int(len(f)))
	fixup(dst)
}

func BenchShortRaw(dp *uint64, pp *int64, count int, f []float64) {
	C.dragonboxBenchShortRaw((*C.uint64_t)(dp), (*C.int64_t)(pp), C.int(count), (*C.double)(&f[0]), C.int(len(f)))
}

func fixup(b []byte) {
	end := bytes.IndexByte(b, 0)
	if i := bytes.IndexByte(b[:end], 'E'); i >= 0 {
		b[i] = 'e'
	}
	if i := end; i >= 2 && (b[i-2] == 'e') {
		b[i+2] = 0
		b[i+1] = b[i-1]
		b[i] = '0'
		b[i-1] = '+'
		return
	}
	if i := end; i >= 2 && (b[i-2] == '-' || b[i-2] == '+') {
		b[i] = b[i-1]
		b[i-1] = '0'
		b[i+1] = 0
		return
	}
	if i := end; i >= 3 && b[i-3] == 'e' && b[i-2] != '+' && b[i-2] != '-' {
		b[i+1] = 0
		b[i] = b[i-1]
		b[i-1] = b[i-2]
		b[i-2] = '+'
		return
	}
	if i := end; i >= 4 && b[i-4] == 'e' && b[i-3] != '+' && b[i-3] != '-' {
		b[i+1] = 0
		b[i] = b[i-1]
		b[i-1] = b[i-2]
		b[i-2] = b[i-3]
		b[i-3] = '+'
		return
	}
}