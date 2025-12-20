#include "bench.h"

void
benchFixed(char *dst, int count, double *f, int nf, int digits, void (*fn)(char*, double, int))
{
	for(int i = 0; i < count; i++)
		for(int j = 0; j < nf; j++)
			fn(dst, f[j], digits);
}

void
benchShort(char *dst, int count, double *f, int nf, void (*fn)(char*, double))
{
	for(int i = 0; i < count; i++)
		for(int j = 0; j < nf; j++)
			fn(dst, f[j]);
}

void
benchShortRaw(uint64_t *dp, int64_t *pp, int count, double *f, int nf, void (*fn)(uint64_t*, int64_t*, double))
{
	for(int i = 0; i < count; i++)
		for(int j = 0; j < nf; j++)
			fn(dp, pp, f[j]);
}

double
benchParse(int count, char *text, double (*fn)(char *start, char *end))
{
	double total = 0;

	for(int i = 0; i < count; i++) {
		int start = 0;
		total = 0;
		for(int j = 0;; j++) {
			int c = text[j];
			if(c == '\0' || c == '\n') {
				total += fn(text+start, text+j);
				if(c == '\0')
					break;
				start = j+1;
			}
		}
	}
	return total;
}

double
benchParseRaw(int count, int64_t *raw, int nraw, double (*fn)(uint64_t, int))
{
	double total = 0;

	for(int i = 0; i < count; i++) {
		total = 0;
		for(int j = 0; j < nraw; j += 2)
			total += fn(raw[j], raw[j+1]);
	}
	return total;
}
