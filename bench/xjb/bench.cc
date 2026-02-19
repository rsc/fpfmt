#include <string.h>
#include "xjb.h"

extern "C" {
#include "../bench.h"

void xjbShort(char *dst, double f) {
	xjb::to_string(f, dst);
	if (dst[1] != '.' && dst[1] != 'e' && dst[1] != '\0') {
		int n = strlen(dst);
		int i = 1;
		while (i < n && dst[i] != '.')
			i++;
		memmove(dst+2, dst+1, i-1);
		dst[1] = '.';
		if (i == n)
			n++;
		while(n>0 && dst[n-1] == '0')
			n--;
		if(i-1 == 0) {
			dst[n] = 0;
		} else {
			if(n == 2)
				n--;
			dst[n] = 'e';
			dst[n+1] = '+';
			dst[n+2] = (i-1)/10 + '0';
			dst[n+3] = (i-1)%10 + '0';
			dst[n+4] = 0;
		}
	} else if (dst[0] == '0' && dst[1] == '.') {
		int i = 2;
		int n = strlen(dst);
		while (i < n && dst[i] == '0')
			i++;
		memmove(dst+1, dst+i, n-i);
		dst[0] = dst[1];
		dst[1] = '.';
		n -= i-1;
		if(n == 2)
			n--;
		dst[n] = 'e';
		dst[n+1] = '-';
		int e = i-1;
		dst[n+2] = e/10 + '0';
		dst[n+3] = e%10 + '0';
		dst[n+4] = 0;
	} else if (strchr(dst, 'e') == 0) {
		int n = strlen(dst);
		while(n > 0 && dst[n-1] == '0')
			n--;
		if(n > 0 && dst[n-1] == '.')
			n--;
		strcpy(dst+n, "e+00");
	}
}

void xjbBenchShort(char *dst, int count, double *f, int nf) {
	benchShort(dst, count, f, nf, xjbShort);
}

} // extern "C"
