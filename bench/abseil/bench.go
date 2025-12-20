package abseil

/*
#cgo CXXFLAGS: -std=c++17

#include <stdlib.h>
#include <string.h>
#include "../bench.h"

double abslstrtod(char*, char*);

double
abslBenchParse(int count, char *s)
{
	return benchParse(count, s, abslstrtod);
}
*/
import "C"
import "unsafe"

func BenchParse(count int, text []byte) float64 {
	return float64(C.abslBenchParse(C.int(count), (*C.char)(unsafe.Pointer(&text[0]))))
}
