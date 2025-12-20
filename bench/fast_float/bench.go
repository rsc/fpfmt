package fast_float

/*
#include <string.h>
#include <stdlib.h>
#include "../bench.h"

double fast_float_strtod(char*, char*);
double fast_float_parse(uint64_t, int);

double
fast_floatBenchParse(int count, char *s)
{
	return benchParse(count, s, fast_float_strtod);
}

double
fast_floatBenchParseRaw(int count, int64_t *raw, int nraw)
{
	return benchParseRaw(count, raw, nraw, fast_float_parse);
}
*/
import "C"
import "unsafe"

func BenchParse(count int, text []byte) float64 {
	return float64(C.fast_floatBenchParse(C.int(count), (*C.char)(unsafe.Pointer(&text[0]))))
}

func BenchParseRaw(count int, raw []int64) float64 {
	return float64(C.fast_floatBenchParseRaw(C.int(count), (*C.int64_t)(&raw[0]), C.int(len(raw))))
}
