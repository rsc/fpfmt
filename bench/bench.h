#include <stdint.h>

void benchFixed(char*, int, double*, int, int, void (*)(char*, double, int));
void benchShort(char*, int, double*, int, void (*)(char*, double));
void benchShortRaw(uint64_t*, int64_t*, int, double*, int, void (*)(uint64_t*, int64_t*, double));
double benchParse(int, char*, double (*)(char*, char*));
double benchParseRaw(int, int64_t*, int, double (*)(uint64_t, int));
