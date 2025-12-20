#include <stdint.h>
#include <stdio.h>
#include "fast_float.h"
#include "decimal_to_binary.h"

extern "C" {

double fast_float_strtod(char *s, char *e) {
	double d;
	fast_float::from_chars(s, e, d);
	return d;
}

double fast_float_parse(uint64_t d, int p) {
	double f;
	fast_float::to_float(0, fast_float::compute_float<fast_float::binary_format<double>>(p, d), f);
	return f;
}

} // extern "C"
