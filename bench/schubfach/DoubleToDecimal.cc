#include <stdio.h>
#include <stdint.h>
#include <string.h>
#include <string>

typedef unsigned char byte;

typedef uint32_t uint32;
typedef uint64_t uint64;
typedef int64_t int64;

static uint64_t rotateRight64(uint64_t x, int s) {
	return (x>>s) | (x<<(64-s));
}

static const int DoubleSIZE = 64;
static const int LongSIZE = 64;
static int numberOfLeadingZeros(int64 f) {
    return __builtin_clzll(f);
}
static int64 multiplyHigh(int64 x, int64 y) {
    return ((__int128)x * y) >> 64;
}

static int64 doubleToRawLongBits(double f) {
    int64 i;
    memcpy(&i, &f, sizeof f);
    return i;
}

static void trimZeros(uint64_t *xp, int64_t *pp) {
	uint64_t x = *xp;
	int p = *pp;

	const uint64_t maxUint64 = ~0ULL;
	const uint64_t div1e8m = 0xc767074b22e90e21ULL;
	const uint64_t div1e4m = 0xd288ce703afb7e91ULL;
	const uint64_t div1e2m = 0x8f5c28f5c28f5c29ULL;
	const uint64_t div1e1m = 0xcccccccccccccccdULL;
	const uint64_t div1e8le = maxUint64 / 100000000;
	const uint64_t div1e4le = maxUint64 / 10000;
	const uint64_t div1e2le = maxUint64 / 100;
	const uint64_t div1e1le = maxUint64 / 10;

	uint64_t d;

	// Cut 1 zero, or else return.
	if ((d = rotateRight64(x*div1e1m, 1)) <= div1e1le) {
		x = d;
		p++;
	} else {
		*xp = x;
		*pp = p;
		return;
	}

	// Cut 8 zeros, then 4, then 2, then 1.
	if ((d = rotateRight64(x*div1e8m, 8)) <= div1e8le) {
		x = d;
		p += 8;
	}
	if ((d = rotateRight64(x*div1e4m, 4)) <= div1e4le) {
		x = d;
		p += 4;
	}
	if ((d = rotateRight64(x*div1e2m, 2)) <= div1e2le) {
		x = d;
		p += 2;
	}
	if ((d = rotateRight64(x*div1e1m, 1)) <= div1e1le) {
		x = d;
		p += 1;
	}
	*xp = x;
	*pp = p;
}



// https://github.com/c4f7fcce9cb06515/Schubfach/blob/3c92d3c9b1fead540616c918cdfef432bca53dfa/todec/src/math/FloatToDecimal.java

/*
 * Copyright 2018-2020 Raffaello Giulietti
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 */

#include "MathUtils.h"

// package math;
//
// import java.io.IOException;
//
// import static java.lang.Double.*;
// import static java.lang.Long.*;
// import static java.lang.Math.multiplyHigh;
// import static math.MathUtils.*;

#undef NAN

/**
 * This class exposes a method to render a {@code double} as a string.
 *
 * @author Raffaello Giulietti
 */
class DoubleToDecimal {
public:
    /*
    For full details about this code see the following references:

    [1] Giulietti, "The Schubfach way to render doubles",
        https://drive.google.com/open?id=1luHhyQF9zKlM8yJ1nebU0OgVYhfC6CBN

    [2] IEEE Computer Society, "IEEE Standard for Floating-Point Arithmetic"

    [3] Bouvier & Zimmermann, "Division-Free Binary-to-Decimal Conversion"

    Divisions are avoided altogether for the benefit of those architectures
    that do not provide specific machine instructions or where they are slow.
    This is discussed in section 10 of [1].
     */

    // The precision in bits.
    static const int P = 53;

    // Exponent width in bits.
    static const int W = (DoubleSIZE - 1) - (P - 1);

    // Minimum value of the exponent: -(2^(W-1)) - P + 3.
    static const int Q_MIN = (-(1 << (W - 1))) - P + 3;

    // Maximum value of the exponent: 2^(W-1) - P.
    static const int Q_MAX = (1 << (W - 1)) - P;

    // 10^(E_MIN - 1) <= MIN_VALUE < 10^E_MIN
    static const int E_MIN = -323;

    // 10^(E_MAX - 1) <= MAX_VALUE < 10^E_MAX
    static const int E_MAX = 309;

    // Threshold to detect tiny values, as in section 8.1.1 of [1]
    static const int64 C_TINY = 3;

    // The minimum and maximum k, as in section 8 of [1]
    static const int K_MIN = -324;
    static const int K_MAX = 292;

    // H is as in section 8 of [1].
    static const int H = 17;

    // Minimum value of the significand of a normal value: 2^(P-1).
    static const int64 C_MIN = 1LL << (P - 1);

    // Mask to extract the biased exponent.
    static const int BQ_MASK = (1 << W) - 1;

    // Mask to extract the fraction bits.
    static const int64 T_MASK = (1LL << (P - 1)) - 1;

    // Used in rop().
    static const int64 MASK_63 = (1ULL << 63) - 1;

    // Used for left-to-tight digit extraction.
    static const int MASK_28 = (1 << 28) - 1;

    static const int NON_SPECIAL = 0;
    static const int PLUS_ZERO = 1;
    static const int MINUS_ZERO = 2;
    static const int PLUS_INF = 3;
    static const int MINUS_INF = 4;
    static const int NAN = 5;

    /*
    Room for the int64er of the forms
        -ddddd.dddddddddddd         H + 2 characters
        -0.00ddddddddddddddddd      H + 5 characters
        -d.ddddddddddddddddE-eee    H + 7 characters
    where there are H digits d
     */
    static const int MAX_CHARS = H + 7;

    // Numerical results are created here...
    byte bytes[MAX_CHARS];

    // Index into bytes of rightmost valid character.
    int index;

    DoubleToDecimal() {
    }

    /**
     * Returns a string rendering of the {@code double} argument.
     *
     * <p>The characters of the result are all drawn from the ASCII set.
     * <ul>
     * <li> Any NaN, whether quiet or signaling, is rendered as
     * {@code "NaN"}, regardless of the sign bit.
     * <li> The infinities +&infin; and -&infin; are rendered as
     * {@code "Infinity"} and {@code "-Infinity"}, respectively.
     * <li> The positive and negative zeroes are rendered as
     * {@code "0.0"} and {@code "-0.0"}, respectively.
     * <li> A finite negative {@code v} is rendered as the sign
     * '{@code -}' followed by the rendering of the magnitude -{@code v}.
     * <li> A finite positive {@code v} is rendered in two stages:
     * <ul>
     * <li> <em>Selection of a decimal</em>: A well-defined
     * decimal <i>d</i><sub><code>v</code></sub> is selected
     * to represent {@code v}.
     * <li> <em>Formatting as a string</em>: The decimal
     * <i>d</i><sub><code>v</code></sub> is formatted as a string,
     * either in plain or in computerized scientific notation,
     * depending on its value.
     * </ul>
     * </ul>
     *
     * <p>A <em>decimal</em> is a number of the form
     * <i>d</i>&times;10<sup><i>i</i></sup>
     * for some (unique) integers <i>d</i> &gt; 0 and <i>i</i> such that
     * <i>d</i> is not a multiple of 10.
     * These integers are the <em>significand</em> and
     * the <em>exponent</em>, respectively, of the decimal.
     * The <em>length</em> of the decimal is the (unique)
     * integer <i>n</i> meeting
     * 10<sup><i>n</i>-1</sup> &le; <i>d</i> &lt; 10<sup><i>n</i></sup>.
     *
     * <p>The decimal <i>d</i><sub><code>v</code></sub>
     * for a finite positive {@code v} is defined as follows:
     * <ul>
     * <li>Let <i>R</i> be the set of all decimals that round to {@code v}
     * according to the usual round-to-closest rule of
     * IEEE 754 floating-point arithmetic.
     * <li>Let <i>m</i> be the minimal length over all decimals in <i>R</i>.
     * <li>When <i>m</i> &ge; 2, let <i>T</i> be the set of all decimals
     * in <i>R</i> with length <i>m</i>.
     * Otherwise, let <i>T</i> be the set of all decimals
     * in <i>R</i> with length 1 or 2.
     * <li>Define <i>d</i><sub><code>v</code></sub> as
     * the decimal in <i>T</i> that is closest to {@code v}.
     * Or if there are two such decimals in <i>T</i>,
     * select the one with the even significand (there is exactly one).
     * </ul>
     *
     * <p>The (uniquely) selected decimal <i>d</i><sub><code>v</code></sub>
     * is then formatted.
     *
     * <p>Let <i>d</i>, <i>i</i> and <i>n</i> be the significand, exponent and
     * length of <i>d</i><sub><code>v</code></sub>, respectively.
     * Further, let <i>e</i> = <i>n</i> + <i>i</i> - 1 and let
     * <i>d</i><sub>1</sub>&hellip;<i>d</i><sub><i>n</i></sub>
     * be the usual decimal expansion of the significand.
     * Note that <i>d</i><sub>1</sub> &ne; 0 &ne; <i>d</i><sub><i>n</i></sub>.
     * <ul>
     * <li>Case -3 &le; <i>e</i> &lt; 0:
     * <i>d</i><sub><code>v</code></sub> is formatted as
     * <code>0.0</code>&hellip;<code>0</code><!--
     * --><i>d</i><sub>1</sub>&hellip;<i>d</i><sub><i>n</i></sub>,
     * where there are exactly -(<i>n</i> + <i>i</i>) zeroes between
     * the decimal point and <i>d</i><sub>1</sub>.
     * For example, 123 &times; 10<sup>-4</sup> is formatted as
     * {@code 0.0123}.
     * <li>Case 0 &le; <i>e</i> &lt; 7:
     * <ul>
     * <li>Subcase <i>i</i> &ge; 0:
     * <i>d</i><sub><code>v</code></sub> is formatted as
     * <i>d</i><sub>1</sub>&hellip;<i>d</i><sub><i>n</i></sub><!--
     * --><code>0</code>&hellip;<code>0.0</code>,
     * where there are exactly <i>i</i> zeroes
     * between <i>d</i><sub><i>n</i></sub> and the decimal point.
     * For example, 123 &times; 10<sup>2</sup> is formatted as
     * {@code 12300.0}.
     * <li>Subcase <i>i</i> &lt; 0:
     * <i>d</i><sub><code>v</code></sub> is formatted as
     * <i>d</i><sub>1</sub>&hellip;<!--
     * --><i>d</i><sub><i>n</i>+<i>i</i></sub>.<!--
     * --><i>d</i><sub><i>n</i>+<i>i</i>+1</sub>&hellip;<!--
     * --><i>d</i><sub><i>n</i></sub>.
     * There are exactly -<i>i</i> digits to the right of
     * the decimal point.
     * For example, 123 &times; 10<sup>-1</sup> is formatted as
     * {@code 12.3}.
     * </ul>
     * <li>Case <i>e</i> &lt; -3 or <i>e</i> &ge; 7:
     * computerized scientific notation is used to format
     * <i>d</i><sub><code>v</code></sub>.
     * Here <i>e</i> is formatted as by {@link Integer#toString(int)}.
     * <ul>
     * <li>Subcase <i>n</i> = 1:
     * <i>d</i><sub><code>v</code></sub> is formatted as
     * <i>d</i><sub>1</sub><code>.0E</code><i>e</i>.
     * For example, 1 &times; 10<sup>23</sup> is formatted as
     * {@code 1.0E23}.
     * <li>Subcase <i>n</i> &gt; 1:
     * <i>d</i><sub><code>v</code></sub> is formatted as
     * <i>d</i><sub>1</sub><code>.</code><i>d</i><sub>2</sub><!--
     * -->&hellip;<i>d</i><sub><i>n</i></sub><code>E</code><i>e</i>.
     * For example, 123 &times; 10<sup>-21</sup> is formatted as
     * {@code 1.23E-19}.
     * </ul>
     * </ul>
     *
     * @param v the {@code double} to be rendered.
     * @return a string rendering of the argument.
     */
    std::string toString(double v) {
        return toDecimalString(v);
    }

    /**
     * Appends the rendering of the {@code v} to {@code app}.
     *
     * <p>The outcome is the same as if {@code v} were first
     * {@link #toString(double) rendered} and the resulting string were then
     * {@link Appendable#append(CharSequence) appended} to {@code app}.
     *
     * @param v the {@code double} whose rendering is appended.
     * @param app the {@link Appendable} to append to.
     * @throws IOException If an I/O error occurs
     */
    int appendTo(double v, char *p) {
        return appendDecimalTo(v, p);
    }

    std::string toDecimalString(double v) {
        switch (toDecimal(v)) {
            case NON_SPECIAL: return charsToString();
            case PLUS_ZERO: return "0.0";
            case MINUS_ZERO: return "-0.0";
            case PLUS_INF: return "Infinity";
            case MINUS_INF: return "-Infinity";
            default: return "NaN";
        }
    }

    static int append(char *app, const char *s) {
        int n = strlen(s);
        memmove(app, s, n+1);
        return n;
    }

    int appendDecimalTo(double v, char *app) {
        switch (toDecimal(v)) {
            case NON_SPECIAL:
                memcpy(app, bytes, index+1);
                return index+1;
            case PLUS_ZERO: return append(app, "0.0");
            case MINUS_ZERO: return append(app, "-0.0");
            case PLUS_INF: return append(app, "Infinity");
            case MINUS_INF: return append(app, "-Infinity");
            default: return append(app, "NaN");
        }
    }

    /*
    Returns
        PLUS_ZERO       iff v is 0.0
        MINUS_ZERO      iff v is -0.0
        PLUS_INF        iff v is POSITIVE_INFINITY
        MINUS_INF       iff v is NEGATIVE_INFINITY
        NAN             iff v is NaN
     */
    int toDecimal(double v) {
        /*
        For full details see references [2] and [1].

        For finite v != 0, determine integers c and q such that
            |v| = c 2^q    and
            Q_MIN <= q <= Q_MAX    and
                either    2^(P-1) <= c < 2^P                 (normal)
                or        0 < c < 2^(P-1)  and  q = Q_MIN    (subnormal)
         */
        int64 bits = doubleToRawLongBits(v);
        int64 t = bits & T_MASK;
        int bq = (int) (((uint64)bits >> (P - 1)) & BQ_MASK);
        // fprintf(stderr, "toDecimal bits %g %016llx %016lld %d\n", v, bits, t, bq);
        if (bq < BQ_MASK) {
            index = -1;
            if (bits < 0) {
                append('-');
            }
            if (bq != 0) {
                // normal value. Here mq = -q
                int mq = -Q_MIN + 1 - bq;
                int64 c = C_MIN | t;
// fprintf(stderr, "normal mq=%d c=%lld P=%d\n", mq,  c, P);
                // The fast path discussed in section 8.2 of [1].
                if ((0 < mq) & (mq < P)) {
                    int64 f = c >> mq;
                    if ((f << mq) == c) {
                        return toChars(f, 0);
                    }
                }
                return toDecimal(-mq, c, 0);
            }
            if (t != 0) {
                // subnormal value
                return t < C_TINY
                       ? toDecimal(Q_MIN, 10 * t, -1)
                       : toDecimal(Q_MIN, t, 0);
            }
            return bits == 0 ? PLUS_ZERO : MINUS_ZERO;
        }
        if (t != 0) {
            return NAN;
        }
        return bits > 0 ? PLUS_INF : MINUS_INF;
    }

    int toDecimalRaw(double v, uint64_t *dp, int64_t *pp) {
        /*
        For full details see references [2] and [1].

        For finite v != 0, determine integers c and q such that
            |v| = c 2^q    and
            Q_MIN <= q <= Q_MAX    and
                either    2^(P-1) <= c < 2^P                 (normal)
                or        0 < c < 2^(P-1)  and  q = Q_MIN    (subnormal)
         */
        int64 bits = doubleToRawLongBits(v);
        int64 t = bits & T_MASK;
        int bq = (int) (((uint64)bits >> (P - 1)) & BQ_MASK);
        // fprintf(stderr, "toDecimal bits %g %016llx %016lld %d\n", v, bits, t, bq);
        if (bq < BQ_MASK) {
            index = -1;
            if (bits < 0) {
                append('-');
            }
            if (bq != 0) {
                // normal value. Here mq = -q
                int mq = -Q_MIN + 1 - bq;
                int64 c = C_MIN | t;
// fprintf(stderr, "normal mq=%d c=%lld P=%d\n", mq,  c, P);
                // The fast path discussed in section 8.2 of [1].
                if (0 && (0 < mq) & (mq < P)) {
                    int64 f = c >> mq;
                    if ((f << mq) == c) {
                        return toChars(f, 0);
                    }
                }
                toDecimalRaw(-mq, c, 0, dp, pp);
                trimZeros(dp, pp);
                return 0;
            }
            if (t != 0) {
                // subnormal value
                t < C_TINY
                       ? toDecimalRaw(Q_MIN, 10 * t, -1, dp, pp)
                       : toDecimalRaw(Q_MIN, t, 0, dp, pp);
		trimZeros(dp, pp);
		return 0;
            }
            return bits == 0 ? PLUS_ZERO : MINUS_ZERO;
        }
        if (t != 0) {
            return NAN;
        }
        return bits > 0 ? PLUS_INF : MINUS_INF;
    }

    int toDecimal(int q, int64 c, int dk) {
//fprintf(stderr, "toDecimal2 q=%d c=%lld dk=%d\n", q,  c, dk);
        /*
        The skeleton corresponds to figure 4 of [1].
        The efficient computations are those summarized in figure 7.

        Here's a correspondence between Java names and names in [1],
        expressed as approximate LaTeX source code and informally.
        Other names are identical.
        cb:     \bar{c}     "c-bar"
        cbr:    \bar{c}_r   "c-bar-r"
        cbl:    \bar{c}_l   "c-bar-l"

        vb:     \bar{v}     "v-bar"
        vbr:    \bar{v}_r   "v-bar-r"
        vbl:    \bar{v}_l   "v-bar-l"

        rop:    r_o'        "r-o-prime"
         */
        int out = (int) c & 0x1;
        int64 cb = c << 2;
        int64 cbr = cb + 2;
        int64 cbl;
        int k;
        /*
        flog10pow2(e) = floor(log_10(2^e))
        flog10threeQuartersPow2(e) = floor(log_10(3/4 2^e))
        flog2pow10(e) = floor(log_2(10^e))
         */
        if ((c != C_MIN) | (q == Q_MIN)) {
            // regular spacing
            cbl = cb - 2;
            k = MathUtils::flog10pow2(q);
        } else {
            // irregular spacing
            cbl = cb - 1;
            k = MathUtils::flog10threeQuartersPow2(q);
        }
        int h = q + MathUtils::flog2pow10(-k) + 2;

        // g1 and g0 are as in section 9.9.3 of [1], so g = g1 2^63 + g0
        int64 g1 = MathUtils::gg1(k);
        int64 g0 = MathUtils::gg0(k);

        int64 vb = rop(g1, g0, cb << h);
        int64 vbl = rop(g1, g0, cbl << h);
        int64 vbr = rop(g1, g0, cbr << h);

//fprintf(stderr, "vb=%lld.%d vbl=%lld.%d vbr=%lld.%d k=%d\n", vb>>2, (int)(vb&3), vbl>>2, (int)(vbl&3), vbr>>2, (int)(vbr&3), k);

        int64 s = vb >> 2;
        if (s >= 10) {
            /*
            For n = 17, m = 1 the table in section 10 of [1] shows
                s' = floor(s / 10) = floor(s 115292150460684698 / 2^60)
                   = floor(s 115292150460684698 2^4 / 2^64)

            sp10 = 10 s'
            tp10 = 10 t'
            upin    iff    u' = sp10 10^k in Rv
            wpin    iff    w' = tp10 10^k in Rv
            See section 9.4 of [1].
             */
            int64 sp10 = 10 * multiplyHigh(s, 115292150460684698LL << 4);
            int64 tp10 = sp10 + 10;
            bool upin = (vbl + out) <= (sp10 << 2);
            bool wpin = ((tp10 << 2) + out) <= vbr;
//fprintf(stderr, "s=%lld out=%d sp10=%lld tp10=%lld upin=%d wpin=%d\n", s, out, sp10, tp10, upin, wpin);
            if (upin != wpin) {
                return toChars(upin ? sp10 : tp10, k+dk);
            }
        }

        /*
        10 <= s < 100    or    s >= 100  and  u', w' not in Rv
        uin    iff    u = s 10^k in Rv
        win    iff    w = t 10^k in Rv
        See section 9.4 of [1].
         */
        int64 t = s + 1;
        bool uin = (vbl + out) <= (s << 2);
        bool win = ((t << 2) + out) <= vbr;
        if (uin != win) {
            // Exactly one of u or w lies in Rv.
            return toChars(uin ? s : t, k + dk);
        }
        /*
        Both u and w lie in Rv: determine the one closest to v.
        See section 9.4 of [1].
         */
        int64 cmp = vb - ((s + t) << 1);
        return toChars(cmp < 0 || (cmp == 0 && (s & 0x1) == 0) ? s : t, k + dk);
    }

    int toDecimalRaw(int q, int64 c, int dk, uint64_t *dp, int64_t *pp) {
//fprintf(stderr, "toDecimal2 q=%d c=%lld dk=%d\n", q,  c, dk);
        /*
        The skeleton corresponds to figure 4 of [1].
        The efficient computations are those summarized in figure 7.

        Here's a correspondence between Java names and names in [1],
        expressed as approximate LaTeX source code and informally.
        Other names are identical.
        cb:     \bar{c}     "c-bar"
        cbr:    \bar{c}_r   "c-bar-r"
        cbl:    \bar{c}_l   "c-bar-l"

        vb:     \bar{v}     "v-bar"
        vbr:    \bar{v}_r   "v-bar-r"
        vbl:    \bar{v}_l   "v-bar-l"

        rop:    r_o'        "r-o-prime"
         */
        int out = (int) c & 0x1;
        int64 cb = c << 2;
        int64 cbr = cb + 2;
        int64 cbl;
        int k;
        /*
        flog10pow2(e) = floor(log_10(2^e))
        flog10threeQuartersPow2(e) = floor(log_10(3/4 2^e))
        flog2pow10(e) = floor(log_2(10^e))
         */
        if ((c != C_MIN) | (q == Q_MIN)) {
            // regular spacing
            cbl = cb - 2;
            k = MathUtils::flog10pow2(q);
        } else {
            // irregular spacing
            cbl = cb - 1;
            k = MathUtils::flog10threeQuartersPow2(q);
        }
        int h = q + MathUtils::flog2pow10(-k) + 2;

        // g1 and g0 are as in section 9.9.3 of [1], so g = g1 2^63 + g0
        int64 g1 = MathUtils::gg1(k);
        int64 g0 = MathUtils::gg0(k);

        int64 vb = rop(g1, g0, cb << h);
        int64 vbl = rop(g1, g0, cbl << h);
        int64 vbr = rop(g1, g0, cbr << h);

//fprintf(stderr, "vb=%lld.%d vbl=%lld.%d vbr=%lld.%d k=%d\n", vb>>2, (int)(vb&3), vbl>>2, (int)(vbl&3), vbr>>2, (int)(vbr&3), k);

        int64 s = vb >> 2;
        if (s >= 10) {
            /*
            For n = 17, m = 1 the table in section 10 of [1] shows
                s' = floor(s / 10) = floor(s 115292150460684698 / 2^60)
                   = floor(s 115292150460684698 2^4 / 2^64)

            sp10 = 10 s'
            tp10 = 10 t'
            upin    iff    u' = sp10 10^k in Rv
            wpin    iff    w' = tp10 10^k in Rv
            See section 9.4 of [1].
             */
            int64 sp10 = 10 * multiplyHigh(s, 115292150460684698LL << 4);
            int64 tp10 = sp10 + 10;
            bool upin = (vbl + out) <= (sp10 << 2);
            bool wpin = ((tp10 << 2) + out) <= vbr;
//fprintf(stderr, "s=%lld out=%d sp10=%lld tp10=%lld upin=%d wpin=%d\n", s, out, sp10, tp10, upin, wpin);
            if (upin != wpin) {
                *dp = upin ? sp10 : tp10;
                *pp = k + dk;
                return 0;
            }
        }

        /*
        10 <= s < 100    or    s >= 100  and  u', w' not in Rv
        uin    iff    u = s 10^k in Rv
        win    iff    w = t 10^k in Rv
        See section 9.4 of [1].
         */
        int64 t = s + 1;
        bool uin = (vbl + out) <= (s << 2);
        bool win = ((t << 2) + out) <= vbr;
        if (uin != win) {
            // Exactly one of u or w lies in Rv.
            *dp = uin ? s : t;
            *pp = k + dk;
            return 0;
        }
        /*
        Both u and w lie in Rv: determine the one closest to v.
        See section 9.4 of [1].
         */
        int64 cmp = vb - ((s + t) << 1);
        *dp = cmp < 0 || (cmp == 0 && (s & 0x1) == 0) ? s : t;
        *pp = k + dk;
        return 0;
    }

    /*
    Computes rop(cp g 2^(-127)), where g = g1 2^63 + g0
    See section 9.10 and figure 5 of [1].
     */
    static int64 rop(int64 g1, int64 g0, int64 cp) {
        int64 x1 = multiplyHigh(g0, cp);
        int64 y0 = g1 * cp;
        int64 y1 = multiplyHigh(g1, cp);
        int64 z = (int64)((uint64)y0 >> 1) + x1;
        int64 vbp = y1 + (int64)((uint64)z >> 63);

        // NOTE: The Java code uses z & MASK_63,
        // but Apple Clang 17.0.0 miscompiles the
        // addition after that, so we use z<<1>>1.
        return vbp | ((((uint64)z<<1>>1) + MASK_63) >> 63);
    }

    /*
    Formats the decimal f 10^e.
     */
    int toChars(int64 f, int e) {
// fprintf(stderr, "toChars f=%lld e=%d\n", f, e);

        /*
        For details not discussed here see section 10 of [1].

        Determine len such that
            10^(len-1) <= f < 10^len
         */
        int len = MathUtils::flog10pow2(LongSIZE - numberOfLeadingZeros(f));
        if (f >= MathUtils::pow10(len)) {
            len += 1;
        }

        /*
        Let fp and ep be the original f and e, respectively.
        Transform f and e to ensure
            10^(H-1) <= f < 10^H
            fp 10^ep = f 10^(e-H) = 0.f 10^e
         */
        f *= MathUtils::pow10(H - len);
        e += len;

        /*
        The toChars?() methods perform left-to-right digits extraction
        using ints, provided that the arguments are limited to 8 digits.
        Therefore, split the H = 17 digits of f into:
            h = the most significant digit of f
            m = the next 8 most significant digits of f
            l = the last 8, least significant digits of f

        For n = 17, m = 8 the table in section 10 of [1] shows
            floor(f / 10^8) = floor(193428131138340668 f / 2^84) =
            floor(floor(193428131138340668 f / 2^64) / 2^20)
        and for n = 9, m = 8
            floor(hm / 10^8) = floor(1441151881 hm / 2^57)
         */
        int64 hm = (uint64)multiplyHigh(f, 193428131138340668LL) >> 20;
        int l = (int) (f - 100000000LL * hm);
        int h = (int) ((uint64)(hm * 1441151881LL) >> 57);
        int m = (int) (hm - 100000000 * h);

        if (0 && 0 < e && e <= 7) {
            return toChars1(h, m, l, e);
        }
        if (0 && -3 < e && e <= 0) {
            return toChars2(h, m, l, e);
        }
        return toChars3(h, m, l, e);
    }

    int toChars1(int h, int m, int l, int e) {
        /*
        0 < e <= 7: plain format without leading zeroes.
        Left-to-right digits extraction:
        algorithm 1 in [3], with b = 10, k = 8, n = 28.
         */
        appendDigit(h);
        int y = yy(m);
        int t;
        int i = 1;
        for (; i < e; ++i) {
            t = 10 * y;
            appendDigit((uint32)t >> 28);
            y = t & MASK_28;
        }
        append('.');
        for (; i <= 8; ++i) {
            t = 10 * y;
            appendDigit((uint32)t >> 28);
            y = t & MASK_28;
        }
        lowDigits(l);
        return NON_SPECIAL;
    }

    int toChars2(int h, int m, int l, int e) {
        // -3 < e <= 0: plain format with leading zeroes.
        appendDigit(0);
        append('.');
        for (; e < 0; ++e) {
            appendDigit(0);
        }
        appendDigit(h);
        append8Digits(m);
        lowDigits(l);
        return NON_SPECIAL;
    }

    int toChars3(int h, int m, int l, int e) {
        // -3 >= e | e > 7: computerized scientific notation
        appendDigit(h);
        if ((m|l) != 0) {
            append('.');
            append8Digits(m);
            lowDigits(l);
        }
        exponent(e - 1);
        return NON_SPECIAL;
    }

    void lowDigits(int l) {
        if (l != 0) {
            append8Digits(l);
        }
        removeTrailingZeroes();
    }

    void append8Digits(int m) {
        /*
        Left-to-right digits extraction:
        algorithm 1 in [3], with b = 10, k = 8, n = 28.
         */
        int y = yy(m);
        for (int i = 0; i < 8; ++i) {
            int t = 10 * y;
            appendDigit((uint32)t >> 28);
            y = t & MASK_28;
        }
    }

    void removeTrailingZeroes() {
        while (bytes[index] == '0') {
            --index;
        }
        // ... but do not remove the one directly to the right of '.'
        if (bytes[index] == '.') {
            ++index;
        }
    }

    int yy(int a) {
        /*
        Algorithm 1 in [3] needs computation of
            floor((a + 1) 2^n / b^k) - 1
        with a < 10^8, b = 10, k = 8, n = 28.
        Noting that
            (a + 1) 2^n <= 10^8 2^28 < 10^17
        For n = 17, m = 8 the table in section 10 of [1] leads to:
         */
        return (int) ((uint64)multiplyHigh(
                (int64) (a + 1) << 28,
                193428131138340668LL) >> 20) - 1;
    }

    void exponent(int e) {
        append('e');
        if (e < 0) {
            append('-');
            e = -e;
        } else {
            append('+');
        }
        if (0 && e < 10) {
            appendDigit(e);
            return;
        }
        int d;
        if (e >= 100) {
            /*
            For n = 3, m = 2 the table in section 10 of [1] shows
                floor(e / 100) = floor(1311 e / 2^17)
             */
            d = (uint32)(e * 1311) >> 17;
            appendDigit(d);
            e -= 100 * d;
        }
        /*
        For n = 2, m = 1 the table in section 10 of [1] shows
            floor(e / 10) = floor(103 e / 2^10)
         */
        d = (uint32)(e * 103) >> 10;
        appendDigit(d);
        appendDigit(e - 10 * d);
    }

    void append(int c) {
        bytes[++index] = (byte) c;
    }

    void appendDigit(int d) {
        bytes[++index] = (byte) ('0' + d);
    }

    // Using the deprecated constructor enhances performance.
    // @SuppressWarnings("deprecation")
    std::string charsToString() {
        return std::string((const char*)bytes, index+1);
    }

};

extern "C" {
#include "../bench.h"

static DoubleToDecimal schubfach;

static void
schubfachShort(char *dst, double f)
{
	schubfach.appendDecimalTo(f, dst);
}

static void
schubfachShortRaw(uint64_t *dp, int64_t *pp, double f)
{
	schubfach.toDecimalRaw(f, dp, pp);
}

void
schubfachBenchShort(char *dst, int count, double *f, int nf)
{
	benchShort(dst, count, f, nf, schubfachShort);
}

void
schubfachBenchShortRaw(uint64_t *dp, int64_t *pp, int count, double *f, int nf)
{
	benchShortRaw(dp, pp, count, f, nf, schubfachShortRaw);
}

} // extern "C"
