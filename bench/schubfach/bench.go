package schubfach

/*
#include <stdint.h>
#include "../bench.h"

void schubfachRaw(uint64_t*, int64_t*, double);

void
schubfachBenchShortRaw(uint64_t *dp, int64_t *pp, int count, double *f, int nf)
{
	benchShortRaw(dp, pp, count, f, nf, schubfachRaw);
}
*/
import "C"

func BenchShortRaw(dp *uint64, pp *int64, count int, f []float64) {
	C.schubfachBenchShortRaw((*C.uint64_t)(dp), (*C.int64_t)(pp), C.int(count), (*C.double)(&f[0]), C.int(len(f)))
}
