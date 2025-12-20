#include "double-to-string.h"
#include "string-to-double.h"
#include <stdlib.h>
#include <stdio.h>

extern "C" {
#include "../bench.h"

void
dblconvFixed(char *dst, double f, int prec)
{
	double_conversion::StringBuilder b(dst, 100);
	b.Reset();
	if(!double_conversion::DoubleToStringConverter::EcmaScriptConverter().ToExponential(f, prec-1, &b))
		abort();
	dst[b.position()] = 0;
}

void
dblconvShort(char *dst, double f)
{
	double_conversion::StringBuilder b(dst, 100);
	b.Reset();
	if(!double_conversion::DoubleToStringConverter::EcmaScriptConverter().ToShortest(f, &b))
		abort();
	dst[b.position()] = 0;
}

static double_conversion::StringToDoubleConverter dstr(double_conversion::StringToDoubleConverter::NO_FLAGS, 0, 0, "", "");

double
dblconvStrtod(char *s, char *e)
{
	int n;
	return dstr.StringToDouble(s, e-s, &n);
}

} // extern "C"

