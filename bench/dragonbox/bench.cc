#include "dragonbox_to_chars.h"

extern "C" {
#include "../bench.h"

static void
dragonboxShort(char *dst, double f)
{
	jkj::dragonbox::to_chars(f, dst);
}

static void
dragonboxShortRaw(uint64_t *dp, int64_t *pp, double f)
{
	auto v = jkj::dragonbox::to_decimal(f);
	*dp = v.significand;
	*pp = v.exponent;
}

void
dragonboxBenchShort(char *dst, int count, double *f, int nf)
{
	benchShort(dst, count, f, nf, dragonboxShort);
}

void
dragonboxBenchShortRaw(uint64_t *dp, int64_t *pp, int count, double *f, int nf)
{
	benchShortRaw(dp, pp, count, f, nf, dragonboxShortRaw);
}

} // extern "C"
