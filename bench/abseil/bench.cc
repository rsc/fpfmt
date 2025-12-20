#include "absl_strings_charconv.h"

extern "C" {

double
abslstrtod(char *s, char *e)
{
	double d;
	absl::from_chars(s, e, d);
	return d;
}

} // extern "C"