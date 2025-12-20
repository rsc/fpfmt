// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fpfmt

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"iter"
	"math"
	"math/big"
	"math/rand/v2"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "rsc.io/fpfmt/bench"
	"rsc.io/fpfmt/bench/abseil"
	"rsc.io/fpfmt/bench/dblconv"
	"rsc.io/fpfmt/bench/dmg"
	"rsc.io/fpfmt/bench/dragonbox"
	"rsc.io/fpfmt/bench/fast_float"
	"rsc.io/fpfmt/bench/go125"
	"rsc.io/fpfmt/bench/libc"
	"rsc.io/fpfmt/bench/ryu"
	"rsc.io/fpfmt/bench/uscalec"
)

type fixedFn struct {
	name string
	fn   func(dst []byte, count int, f []float64, digits int)
}

type shortFn struct {
	name string
	fn   func(dst []byte, count int, f []float64)
}

type shortRawFn struct {
	name string
	fn   func(dp *uint64, pp *int64, count int, f []float64)
}

type parseFn struct {
	name string
	fn   func(count int, text []byte) float64
}

type parseRawFn struct {
	name string
	fn   func(count int, raw []int64) float64
}

var fixeds = []fixedFn{
	{"uscale", fixedLoop},
	{"uscalet", fixedTruncLoop},
	{"uscalec", uscalec.BenchFixed},
	{"dmg1991", dmg.BenchFixed1991},
	{"dmg1997", dmg.BenchFixed19970128},
	{"dmg2016", dmg.BenchFixed20161215},
	{"dmg2017", dmg.BenchFixed20170421},
	{"dmg2025", dmg.BenchFixed20251117},
	{"dblconv", dblconv.BenchFixed},
	{"libc", libc.BenchFixed},
	{"go125", go125.BenchFixed},
	{"go125unopt", go125.BenchFixedUnopt},
	{"ryu", ryu.BenchFixed},
}

var shorts = []shortFn{
	{"uscale", shortLoop},
	{"uscalec", uscalec.BenchShort},
	{"dmg1991", dmg.BenchShort1991},
	{"dmg1997", dmg.BenchShort19970128},
	{"dmg2016", dmg.BenchShort20161215},
	{"dmg2017", dmg.BenchShort20170421},
	{"dmg2025", dmg.BenchShort20251117},
	{"dblconv", dblconv.BenchShort},
	{"dragonbox", dragonbox.BenchShort},
	{"go125", go125.BenchShort},
	{"go125unopt", go125.BenchShortUnopt},
	{"ryu", ryu.BenchShort},
}

var shortRaws = []shortRawFn{
	{"uscale", shortRawLoop},
	{"uscalet", shortRawTruncLoop},
	{"uscalec", uscalec.BenchShortRaw},
	// {"schubfach", schubfach.BenchShortRaw},
	{"dragonbox", dragonbox.BenchShortRaw},
	{"ryu", ryu.BenchShortRaw},
}

var parses = []parseFn{
	{"uscale", parseLoop},
	{"dmg1991", dmg.BenchParse1991},
	{"dmg1997", dmg.BenchParse19970128},
	{"dmg2016", dmg.BenchParse20161215},
	{"dmg2017", dmg.BenchParse20170421},
	{"dmg2025", dmg.BenchParse20251117},
	{"abseil", abseil.BenchParse},
	{"fast_float", fast_float.BenchParse},
	{"dblconv", dblconv.BenchParse},
	{"libc", libc.BenchParse},
	{"uscalec", uscalec.BenchParse},
}

var parseRaws = []parseRawFn{
	{"uscale", parseRawLoop},
	{"uscalec", uscalec.BenchParseRaw},
	{"fast_float", fast_float.BenchParseRaw},
}

func fixedLoop(dst []byte, count int, fs []float64, digits int) {
	for range count {
		for _, f := range fs {
			d, p := FixedWidth(f, digits)
			efmt(dst, d, p, digits)
		}
	}
}

func fixedTruncLoop(dst []byte, count int, fs []float64, digits int) {
	for range count {
		for _, f := range fs {
			d, p := FixedWidthTrunc(f, digits)
			efmt(dst, d, p, digits)
		}
	}
}

func shortLoop(dst []byte, count int, fs []float64) {
	for range count {
		for _, f := range fs {
			d, p := Short(f)
			efmt(dst, d, p, countDigits(d))
		}
	}
}

func shortRawLoop(dp *uint64, pp *int64, count int, fs []float64) {
	for range count {
		for _, f := range fs {
			d, p := Short(f)
			*dp = d
			*pp = int64(p)
		}
	}
}

func shortRawTruncLoop(dp *uint64, pp *int64, count int, fs []float64) {
	for range count {
		for _, f := range fs {
			d, p := ShortTrunc(f)
			*dp = d
			*pp = int64(p)
		}
	}
}

func parseLoop(count int, text []byte) float64 {
	var total float64
	for range count {
		total = 0
		start := 0
		for i, c := range text {
			if c == '\x00' || c == '\n' {
				total += parseText(text[start:i])
				if c == '\x00' {
					break
				}
				start = i + 1
			}
		}
	}
	return total
}

func parseRawLoop(count int, raw []int64) float64 {
	var total float64
	for range count {
		total = 0
		for i := 0; i < len(raw); i += 2 {
			total += Parse(uint64(raw[i]), int(raw[i+1]))
		}
	}
	return total
}

//go:embed test.ivy
var testIvy string

var ivyRE = regexp.MustCompile(`\(([0-9]+) ftoa ([^ ]+)\) is ([0-9]+) (-?[0-9]+)`)

func TestFixedWidth(t *testing.T) {
	var have, want [2000]byte
	var floats [1]float64

	for _, impl := range fixeds {
		t.Run(impl.name, func(t *testing.T) {
			fail := 0
			for tt := range ivyTests() {
				if tt.n > 18 {
					continue
				}
				floats[0] = tt.f
				want := want[:efmt(want[:], tt.d, tt.p, tt.n)]
				if impl.name == "dblconv" {
					if w, ok := roundHalfUp[string(want)]; ok {
						want = []byte(w)
					}
				}
				clear(have[:])
				impl.fn(have[:], 1, floats[:], tt.n)
				have := have[:bytes.IndexByte(have[:], 0)]
				if string(have) != string(want) {
					t.Fatalf("fixed(%#x, %d) = %s want %s", tt.f, tt.n, have, want)
					if fail++; fail >= 100 {
						t.Fatalf("too many failures")
					}
				}
			}
		})
	}
}

var roundHalfUp = map[string]string{
	"7.65171411354557812e+13": "7.65171411354557813e+13",
	"6.6357334387500562e+14":  "6.6357334387500563e+14",
	"5.37278223945945312e+12": "5.37278223945945313e+12",
	"1.234e+06":               "1.235e+06",
	"1.234567890123456e+15":   "1.234567890123457e+15",
	"9.76562e+18":             "9.76563e+18",
	"1.95312e+19":             "1.95313e+19",
	"7.45058059692382812e-09": "7.45058059692382813e-09",
}

func TestShort(t *testing.T) {
	var have [2000]byte
	var floats [1]float64
	for _, impl := range shorts {
		t.Run(impl.name, func(t *testing.T) {
			fail := 0
			for f := range testFloats() {
				if f == 0 || math.IsInf(f, 0) {
					continue
				}
				floats[0] = f
				want := strconv.FormatFloat(f, 'e', -1, 64)
				clear(have[:])
				impl.fn(have[:], 1, floats[:])
				have := have[:bytes.IndexByte(have[:], 0)]
				if string(have) != want {
					t.Fatalf("short(%#x) = %s want %s", f, have, want)
					if fail++; fail >= 100 {
						t.Fatalf("too many failures")
					}
				}
			}
		})
	}
}

func TestShortRaw(t *testing.T) {
	var have [2000]byte
	var floats [1]float64
	var d uint64
	var p int64
	for _, impl := range shortRaws {
		t.Run(impl.name, func(t *testing.T) {
			fail := 0
			for f := range testFloats() {
				if f == 0 || math.IsInf(f, 0) {
					continue
				}
				floats[0] = f
				want := strconv.FormatFloat(f, 'e', -1, 64)
				clear(have[:])
				impl.fn(&d, &p, 1, floats[:])
				have := have[:efmt(have[:], d, int(p), countDigits(d))]
				if string(have) != want {
					t.Fatalf("shortRaw(%#x) = %s want %s", f, have, want)
					if fail++; fail >= 100 {
						t.Fatalf("too many failures")
					}
				}
			}
		})
	}
}

func TestParse(t *testing.T) {
	for _, impl := range parses {
		t.Run(impl.name, func(t *testing.T) {
			fail := 0
			for tt := range parseTests() {
				have := impl.fn(1, []byte(tt.s+"\x00"))
				if have != tt.want {
					t.Errorf("parse(%s) = %v want %v", tt.s, have, tt.want)
					if fail++; fail >= 100 {
						t.Fatalf("too many failures")
					}
				}
				s := tt.s + "\n" + tt.s + "\n" + tt.s
				have = impl.fn(1, []byte(s+"\x00"))
				if have != tt.want+tt.want+tt.want {
					t.Errorf("parse(%q) = %v want %v", s, have, tt.want+tt.want+tt.want)
					if fail++; fail >= 100 {
						t.Fatalf("too many failures")
					}
				}
			}
		})
	}
}

func TestParseRaw(t *testing.T) {
	for _, impl := range parseRaws {
		t.Run(impl.name, func(t *testing.T) {
			fail := 0
			for line := range strings.Lines(testIvy) {
				m := ivyRE.FindStringSubmatch(line)
				if m == nil {
					t.Fatalf("bad line: %s", line)
				}
				d, _ := strconv.ParseUint(m[3], 10, 64)
				p, _ := strconv.Atoi(m[4])
				s := fmt.Sprintf("%de%d", d, p)
				want, _ := strconv.ParseFloat(s, 64)
				have := impl.fn(1, []int64{int64(d), int64(p)})
				if have != want {
					t.Errorf("parseRaw(%s) = %v want %v", s, have, want)
					if fail++; fail >= 100 {
						t.Fatalf("too many failures")
					}
				}
				have = impl.fn(1, []int64{int64(d), int64(p), int64(d), int64(p), int64(d), int64(p)})
				if have != want+want+want {
					t.Errorf("parse(%q) = %v want %v", s, have, want+want+want)
					if fail++; fail >= 100 {
						t.Fatalf("too many failures")
					}
				}
			}
		})
	}
}

var randFloats = computeRandFloats()

func computeRandFloats() []float64 {
	var fs []float64
	var seed [32]byte
	r := rand.New(rand.NewChaCha8(seed))
	for range 10000 {
		x := r.Uint64N(1<<63 - 1<<52)
		f := math.Float64frombits(x)
		fs = append(fs, f)
	}
	return fs
}

func BenchmarkFixedWidth(b *testing.B) {
	fs := randFloats
	var dst [1000]byte
	for _, n := range []int{2, 5, 10, 15, 17, 18} {
		for _, impl := range fixeds {
			b.Run(fmt.Sprintf("impl=%s/n=%d", impl.name, n), func(b *testing.B) {
				if b.N >= 1<<31 {
					b.Fatalf("b.N too big")
				}
				impl.fn(dst[:], b.N, fs, n)
			})
		}
	}
}

func BenchmarkShort(b *testing.B) {
	fs := randFloats
	var dst [1000]byte
	for _, impl := range shorts {
		b.Run(fmt.Sprintf("impl=%s", impl.name), func(b *testing.B) {
			if b.N >= 1<<31 {
				b.Fatalf("b.N too big")
			}
			impl.fn(dst[:], b.N, fs)
		})
	}
}

func BenchmarkShortRaw(b *testing.B) {
	fs := randFloats
	var d uint64
	var p int64
	for _, impl := range shortRaws {
		b.Run(fmt.Sprintf("impl=%s", impl.name), func(b *testing.B) {
			if b.N >= 1<<31 {
				b.Fatalf("b.N too big")
			}
			impl.fn(&d, &p, b.N, fs)
		})
	}
}

func randParses() []byte {
	var buf bytes.Buffer

	for range 10000 {
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		n := rand.N(uint64(1e19))
		p := rand.N(600) - 300
		fmt.Fprintf(&buf, "%d.%018de%d", n/1e18, n%1e18, p)
	}
	buf.WriteByte(0)
	return buf.Bytes()
}

func randParseRaws() []int64 {
	var raws []int64
	for range 10000 {
		n := rand.N(uint64(1e19))
		p := rand.N(int64(600)) - 300
		raws = append(raws, int64(n), p-18)
	}
	return raws
}

func BenchmarkParse(b *testing.B) {
	ps := randParses()
	for _, impl := range parses {
		b.Run(fmt.Sprintf("impl=%s", impl.name), func(b *testing.B) {
			if b.N >= 1<<31 {
				b.Fatalf("b.N too big")
			}
			impl.fn(b.N, ps)
		})
	}
}

func BenchmarkParseRaw(b *testing.B) {
	raws := randParseRaws()
	for _, impl := range parseRaws {
		b.Run(fmt.Sprintf("impl=%s", impl.name), func(b *testing.B) {
			if b.N >= 1<<31 {
				b.Fatalf("b.N too big")
			}
			impl.fn(b.N, raws)
		})
	}
}

var scatterFlag = flag.String("scatter", "", "write scatterplot data to `file`")

var scatterReps = map[string]int{
	"uscale":     10,
	"uscalec":    10,
	"uscalet":    10,
	"dmg1997":    1,
	"dmg2017":    10,
	"dblconv":    10,
	"dragonbox":  10,
	"ryu":        10,
	"libc":       1,
	"fast_float": 10,
	"abseil":     10,
}

type scatterplot struct {
	scat  string
	name  string
	t     *testing.T
	out   *os.File
	fs    []float64
	ts    []time.Duration
	batch []float64
}

const TimeBatch = 10

func newScatter(t *testing.T, scat, name string) *scatterplot {
	if *scatterFlag == "" {
		t.Skip("skipping without -scatter")
	}
	out, err := os.OpenFile(*scatterFlag, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		t.Fatal(err)
	}
	var fs []float64
	switch scat {
	default:
		t.Fatalf("unknown scat %q", scat)
	case "rand":
		fs = randFloats
	case "hard":
		fs = hardFloats()
	}
	ts := make([]time.Duration, len(fs))
	batch := make([]float64, 1000)
	return &scatterplot{
		scat:  scat,
		name:  name,
		t:     t,
		out:   out,
		fs:    fs,
		ts:    ts,
		batch: batch,
	}
}

func (p *scatterplot) run(name string, do func(int, []float64)) {
	reps := scatterReps[name]
	if reps == 0 {
		return
	}
	var times [TimeBatch]time.Duration
	p.t.Run("impl="+name, func(t *testing.T) {
		print(p.scat, " ", p.name, " ", name)
		for fi, f := range p.fs {
			if fi%100 == 0 {
				print(".")
			}
			for i := range p.batch {
				p.batch[i] = f
			}
			for i := range times {
				start := time.Now()
				do(reps, p.batch)
				times[i] = time.Since(start) / time.Duration(reps)
			}
			slices.Sort(times[:])
			p.ts[fi] = times[len(times)/2]
		}
		var buf bytes.Buffer
		for fi, f := range p.fs {
			fmt.Fprintf(&buf, "%s %s %s %#016x %d\n", p.scat, p.name, name, math.Float64bits(f), p.ts[fi].Nanoseconds())
		}
		p.out.Write(buf.Bytes())
		print("\n")
	})
}

var scatters = []string{"rand", "hard"}

func TestScatterFixed(t *testing.T) {
	var dst [1000]byte
	for _, scat := range scatters {
		t.Run("scatter="+scat, func(t *testing.T) {
			for _, digits := range []int{2, 4, 8, 16, 17, 18} {
				t.Run(fmt.Sprintf("digits=%d", digits), func(t *testing.T) {
					p := newScatter(t, scat, fmt.Sprintf("fixed%d", digits))
					for _, impl := range fixeds {
						p.run(impl.name, func(reps int, batch []float64) {
							impl.fn(dst[:], reps, batch, digits)
						})
					}
				})
			}
		})
	}
}

func TestScatterShort(t *testing.T) {
	var dst [1000]byte
	for _, scat := range scatters {
		t.Run("scatter="+scat, func(t *testing.T) {
			p := newScatter(t, scat, "short")
			for _, impl := range shorts {
				p.run(impl.name, func(reps int, batch []float64) {
					impl.fn(dst[:], reps, batch)
				})
			}
		})
	}
}

func TestScatterShortRaw(t *testing.T) {
	var dm uint64
	var dp int64
	for _, scat := range scatters {
		t.Run("scatter="+scat, func(t *testing.T) {
			p := newScatter(t, scat, "shortraw")
			for _, impl := range shortRaws {
				p.run(impl.name, func(reps int, batch []float64) {
					impl.fn(&dm, &dp, reps, batch)
				})
			}
		})
	}
}

type scatterplotParse struct {
	scat  string
	name  string
	t     *testing.T
	out   *os.File
	fs    []float64
	ts    []time.Duration
	batch []float64
	reps  int
}

func newScatterParse(t *testing.T, scat, name string) *scatterplotParse {
	if *scatterFlag == "" {
		t.Skip("skipping without -scatter")
	}
	out, err := os.OpenFile(*scatterFlag, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		t.Fatal(err)
	}
	var fs []float64
	switch scat {
	default:
		t.Fatalf("unknown scat %q", scat)
	case "rand":
		fs = randFloats
	case "hard":
		fs = hardFloats()
	}
	ts := make([]time.Duration, len(fs))
	batch := make([]float64, 1000)
	return &scatterplotParse{
		scat:  scat,
		name:  name,
		t:     t,
		out:   out,
		fs:    fs,
		ts:    ts,
		batch: batch,
	}
}

func (p *scatterplotParse) run(name string, do func(int, []byte)) {
	reps := scatterReps[name]
	if reps == 0 {
		return
	}
	var times [TimeBatch]time.Duration
	var batch bytes.Buffer
	raws := randParseRaws()
	var floats []float64
	p.t.Run("impl="+name, func(t *testing.T) {
		print(p.scat, " ", p.name, " ", name)
		for i := 0; i < len(raws); i += 2 {
			if (i/2)%100 == 0 {
				print(".")
			}
			s := fmt.Sprintf("%de%d", uint64(raws[i]), raws[i+1])
			f, _ := strconv.ParseFloat(s, 64)
			floats = append(floats, f)
			batch.Reset()
			batch.WriteString(s)
			for range 1000 - 1 {
				batch.WriteByte('\n')
				batch.WriteString(s)
			}
			batch.WriteByte(0)
			for i := range times {
				start := time.Now()
				do(reps, batch.Bytes())
				times[i] = time.Since(start) / time.Duration(reps)
			}
			slices.Sort(times[:])
			p.ts[i/2] = times[len(times)/2]
		}
		var buf bytes.Buffer
		for fi, f := range floats {
			fmt.Fprintf(&buf, "%s %s %s %#016x %d\n", p.scat, p.name, name, math.Float64bits(f), p.ts[fi].Nanoseconds())
		}
		p.out.Write(buf.Bytes())
		print("\n")
	})
}

func (p *scatterplotParse) runRaw(name string, do func(int, []int64)) {
	reps := 10 * scatterReps[name]
	if reps == 0 {
		return
	}
	var times [TimeBatch]time.Duration
	var batch []int64
	raws := randParseRaws()
	var floats []float64
	p.t.Run("impl="+name, func(t *testing.T) {
		print(p.scat, " ", p.name, " ", name)
		for i := 0; i < len(raws); i += 2 {
			if (i/2)%100 == 0 {
				print(".")
			}
			s := fmt.Sprintf("%de%d", uint64(raws[i]), raws[i+1])
			f, _ := strconv.ParseFloat(s, 64)
			floats = append(floats, f)
			batch = batch[:0]
			for range 1000 {
				batch = append(batch, int64(raws[i]), int64(raws[i+1]))
			}
			for i := range times {
				start := time.Now()
				do(reps, batch)
				times[i] = time.Since(start) / time.Duration(reps)
			}
			slices.Sort(times[:])
			p.ts[i/2] = times[len(times)/2]
		}
		var buf bytes.Buffer
		for fi, f := range floats {
			fmt.Fprintf(&buf, "%s %s %s %#016x %d\n", p.scat, p.name, name, math.Float64bits(f), p.ts[fi].Nanoseconds())
		}
		p.out.Write(buf.Bytes())
		print("\n")
	})
}

func TestScatterParse(t *testing.T) {
	for _, scat := range scatters {
		t.Run("scatter="+scat, func(t *testing.T) {
			p := newScatterParse(t, scat, "parse")
			for _, impl := range parses {
				p.run(impl.name, func(reps int, batch []byte) {
					impl.fn(reps, batch)
				})
			}
		})
	}
}

func TestScatterParseRaw(t *testing.T) {
	for _, scat := range scatters {
		t.Run("scatter="+scat, func(t *testing.T) {
			p := newScatterParse(t, scat, "parseraw")
			for _, impl := range parseRaws {
				p.runRaw(impl.name, func(reps int, batch []int64) {
					impl.fn(reps, batch)
				})
			}
		})
	}
}

type ivyTest struct {
	n int
	f float64
	d uint64
	p int
}

func ivyTests() iter.Seq[ivyTest] {
	return func(yield func(ivyTest) bool) {
		if !yield(ivyTest{7, 0x1.18352262653f8p+19, 5738651, -1}) {
			return
		}
		for line := range strings.Lines(testIvy) {
			m := ivyRE.FindStringSubmatch(line)
			if m == nil {
				panic("bad ivy line")
			}
			n, _ := strconv.Atoi(m[1])
			f, _ := strconv.ParseFloat(m[2], 64)
			d, _ := strconv.ParseUint(m[3], 10, 64)
			p, _ := strconv.Atoi(m[4])
			if !yield(ivyTest{n, f, d, p}) {
				break
			}
		}
	}
}

func testFloats() iter.Seq[float64] {
	return func(yield func(float64) bool) {
		for f := range someTestFloats() {
			if !yield(f) || !yield(math.Nextafter(f, math.Inf(-1))) || !yield(math.Nextafter(f, math.Inf(+1))) {
				return
			}
		}
	}
}

func someTestFloats() iter.Seq[float64] {
	return func(yield func(float64) bool) {
		// Powers of 2
		for e := -1074; e <= 1024; e++ {
			if !yield(math.Ldexp(1, e)) {
				return
			}
		}

		// Powers of 10.
		for p := -308; p <= 308; p++ {
			if !yield(math.Pow10(p)) {
				return
			}
		}

		// Unique inputs in test file.
		last := -1.0
		for tt := range ivyTests() {
			if tt.f != last {
				last = tt.f
				if !yield(last) {
					return
				}
			}
		}
	}
}

func hardFloats() []float64 {
	var out []float64
	for _, tab := range hard {
		pe := -int(math.Floor(math.Log2(math.Pow10(tab.p))))
		for exp := pe - 3; exp <= pe+64; exp += 3 {
			if f := math.Ldexp(float64(tab.xmin), exp); f != 0 && !math.IsInf(f, 0) {
				out = append(out, f)
			}
			if f := math.Ldexp(float64(tab.xmax), exp); f != 0 && !math.IsInf(f, 0) {
				out = append(out, f)
			}
		}
	}
	return out
}

type parseTest struct {
	d    uint64
	p    int
	s    string
	want float64
}

var longparseFlag = flag.Bool("longparse", false, "test long parse cases")

func near(f float64) []parseTest {
	if !*longparseFlag {
		// For testing strtod and libc only;
		// our parser does not handle long parses.
		return nil
	}

	var all []parseTest
	for _, f := range []float64{math.Nextafter(f, math.Inf(-1)), f, math.Nextafter(f, math.Inf(+1))} {
		fm, fe := math.Frexp(f)
		m := int64(fm * (1 << 55))
		e := fe - 55
		bm := big.NewInt(m + 2)
		var s string
		var p int
		if e >= 0 {
			bm.Lsh(bm, uint(e))
			s = bm.String()
			p = 0
		} else {
			bm.Mul(bm, new(big.Int).Exp(big.NewInt(5), big.NewInt(int64(-e)), nil))
			s = bm.String()
			p = e
		}
		up := math.Nextafter(f, math.Inf(+1))
		//down := math.Nextafter(f, math.Inf(-1))
		want := f
		if m&4 != 0 {
			// f is odd, so halfway rounds away
			want = up
		}
		all = append(all, parseTest{s: fmt.Sprintf("%se%d", s, p), want: want}, parseTest{s: fmt.Sprintf("%s1e%d", s, p-1), want: up})
	}
	return all
}

func parseTests() iter.Seq[parseTest] {
	const Nudge = 3
	return func(yield func(parseTest) bool) {
		for _, v := range []float64{1e300, 1e-300, 1.23456789e300, 1.23456789e-300} {
			for _, tt := range near(v) {
				if !yield(tt) {
					return
				}
			}
		}
		for f := range testFloats() {
			if math.IsInf(f, 0) { // TODO drop from testFloats
				continue
			}
			d, p := FixedWidth(f, 17)
			if !yield(parseTest{d, p, fmt.Sprintf("%de%d", d, p), f}) {
				return
			}
			if p > -300 {
				for i := -Nudge; i <= Nudge; i++ {
					d1 := d + uint64(i)
					s := fmt.Sprintf("%de%d", d1, p)
					f, _ := strconv.ParseFloat(s, 64)
					if math.IsInf(f, 0) {
						continue
					}
					if !yield(parseTest{d, p, fmt.Sprintf("%de%d", d1, p), f}) {
						return
					}
				}
			}
			d, p = FixedWidth(f, 18)
			if !yield(parseTest{d, p, fmt.Sprintf("%de%d", d, p), f}) {
				return
			}
			if p > -300 {
				for i := -Nudge; i <= Nudge; i++ {
					d1 := d + uint64(i*20)
					s := fmt.Sprintf("%de%d", d1, p)
					f, _ := strconv.ParseFloat(s, 64)
					if math.IsInf(f, 0) {
						continue
					}
					if !yield(parseTest{d, p, fmt.Sprintf("%de%d", d1, p), f}) {
						return
					}
				}
			}
			s := fmt.Sprintf("%.*e", 19-1, f)
			t := s[0:1] + s[2:]
			e := strings.Index(t, "e")
			d, err := strconv.ParseUint(t[:e], 10, 64)
			if err != nil {
				panic(err)
			}
			p64, err := strconv.ParseInt(t[e+1:], 10, 64)
			if err != nil {
				panic(err)
			}
			p = int(p64) - (e - 1)
			if !yield(parseTest{d, p, fmt.Sprintf("%de%d", d, p), f}) {
				return
			}
		}
	}
}

// mix 53 64 exactInfo@ seq -400 400
var hard = []struct {
	p        int
	xmin     uint64
	xminBits float64
	xmax     uint64
	xmaxBits float64
}{
	{-400, 0x109e9ca62cbb67, -51.7934203833, 0x1ff2cac926600d, -50.6240861809},
	{-399, 0x125b143979cd57, -53.3594768483, 0x176a77ee0304c2, -53.4169548651},
	{-398, 0x12fda93f0ca1f5, -51.7886705965, 0x1d1c70a81f10cb, -52.4096436275},
	{-397, 0x12fda93f0ca1f5, -52.4667425016, 0x1e1e76e76b05f9, -51.5354087348},
	{-396, 0x12fda93f0ca1f5, -52.1448144067, 0x1e6cecdf37b23b, -51.8910704471},
	{-395, 0x1a18c9cbca3587, -53.7863369393, 0x1adf5c98d8e872, -52.1083032161},
	{-394, 0x1c0226a79e0041, -52.1775502301, 0x157f7d4713ed28, -53.1083032161},
	{-393, 0x1e287ffb8664c5, -51.9333610188, 0x1132ca9f432420, -53.1083032161},
	{-392, 0x1e2241066608f3, -53.39746261, 0x15829cc1a41b11, -53.9434129196},
	{-391, 0x1fdbc847599eed, -53.6565479809, 0x120f8e3fbcef1d, -53.6330781148},
	{-390, 0x131d782acf5f5b, -54.0715854802, 0x1ba4209b0fddf5, -50.8842388265},
	{-389, 0x197ca039147f24, -53.334619886, 0x1a5939f9df71de, -52.2418696893},
	{-388, 0x19508178ebe832, -53.3108673538, 0x1ab1777a309fc2, -52.2531425717},
	{-387, 0x16a528b980a505, -52.7161222327, 0x155ac5fb5a1968, -52.2531425717},
	{-386, 0x121dba2e008404, -52.7161222327, 0x1d1f405433cfcd, -50.6471198291},
	{-385, 0x104df45c9a106a, -53.5461972312, 0x1e6fccf64ccc21, -48.9656592807},
	{-384, 0x116437da3def60, -53.1311597319, 0x1f294f4a0f60c5, -48.6464135416},
	{-383, 0x116437da3def60, -52.8092316371, 0x1d218c98d49f99, -51.1487500818},
	{-382, 0x18608ae5396385, -54.6901324237, 0x1ac10bbf07018f, -53.0400959056},
	{-381, 0x1199a1a2bc9dfc, -52.3288087772, 0x1f277427b6290e, -53.2904604631},
	{-380, 0x1da83b62d85fb7, -52.8871509818, 0x1f277427b6290e, -52.9685323682},
	{-379, 0x117eb1e0bbddf6, -52.5811360498, 0x1f277427b6290e, -52.6466042733},
	{-378, 0x117eb1e0bbddf6, -53.2592079549, 0x195a8d9e82d7be, -51.7087917879},
	{-377, 0x1fab260509f3c5, -51.5119262434, 0x109b34ae24e8c3, -51.1594851446},
	{-376, 0x15a87b6690ff2f, -52.5274282143, 0x174ba190301532, -52.4515322419},
	{-375, 0x115395eba7328c, -53.5274282143, 0x1ba0870b19e1d5, -53.4331660963},
	{-374, 0x1e29f69d829681, -49.6378849187, 0x107b1b65844ba8, -51.4795274356},
	{-373, 0x1cf55eba423cf3, -52.4485860533, 0x107b1b65844ba8, -51.1575993407},
	{-372, 0x1f7438fe683a02, -52.3701549365, 0x1578cfedd045c6, -53.0315138141},
	{-371, 0x1f7438fe683a02, -52.0482268416, 0x112d7324a69e38, -53.0315138141},
	{-370, 0x1fddcd5b71406b, -47.1701958075, 0x103efda368da68, -48.2324912042},
	{-369, 0x1ffeb80389c789, -46.2946585716, 0x1003939d6b032c, -48.9313216695},
	{-368, 0x196e1ab8d0e5ed, -52.7839531036, 0x1a5bc2d0c842dd, -57.2001997276},
	{-367, 0x1e251007648856, -51.7671588154, 0x1a5bc2d0c842dd, -56.8782716327},
	{-366, 0x1099fcaae3bd1a, -51.736186943, 0x1a5bc2d0c842dd, -57.5563435378},
	{-365, 0x1bba341e94e788, -52.7881824014, 0x1a5bc2d0c842dd, -57.2344154429},
	{-364, 0x162e9018771fa0, -52.7881824014, 0x1a5bc2d0c842dd, -56.912487348},
	{-363, 0x1e29da24a72dd7, -50.9821271956, 0x10b4cae1448cf4, -51.3858144334},
	{-362, 0x1e29da24a72dd7, -50.6601991007, 0x1009b8865649e0, -51.1227800276},
	{-361, 0x149e75675fd356, -56.6072821794, 0x1f4f9b16440496, -51.4250104888},
	{-360, 0x15756c2ef0e083, -52.9666969404, 0x171038ecea7cda, -52.0872759554},
	{-359, 0x15c79521bc662e, -53.8554230809, 0x171038ecea7cda, -51.7653478605},
	{-358, 0x199f3e3d9c718c, -51.8096825997, 0x1857bb95437733, -54.6004940563},
	{-357, 0x13fc63edf329e6, -54.0348508032, 0x12b4e1459a2f8d, -52.8055289145},
	{-356, 0x187eac2dadc99e, -52.3693289549, 0x1ef4375c71145c, -53.6077084911},
	{-355, 0x187eac2dadc99e, -52.0474008601, 0x13dd3d0d9e1b2a, -52.9388328574},
	{-354, 0x1e2f8b2f16f17f, -51.6028960261, 0x10f6fe159796d0, -50.1925130079},
	{-353, 0x19d8558e3b1d14, -52.1855369054, 0x179fffb0123b83, -53.2030408212},
	{-352, 0x106dae9f2e8531, -48.3969321128, 0x1fb5a64d064c0f, -48.3468709434},
	{-351, 0x106dae9f2e8531, -49.0750040179, 0x1fe8c1484900e4, -47.5883205032},
	{-350, 0x106dae9f2e8531, -48.753075923, 0x1f90e058a99974, -47.8466590107},
	{-349, 0x19f25cf205ed2e, -52.4648468493, 0x182bb1d9e538ba, -52.8407759185},
	{-348, 0x1e6cf7e593a13c, -53.1669030145, 0x1fc6efd34a2109, -52.7695881647},
	{-347, 0x11b6fe5ddc60d2, -53.2344432966, 0x1845689212706d, -51.7082849425},
	{-346, 0x11b6fe5ddc60d2, -52.9125152017, 0x1fd138e9c29db7, -52.2307124421},
	{-345, 0x11b6fe5ddc60d2, -52.5905871069, 0x1ec4ad0a5dce6b, -52.6148624258},
	{-344, 0x1bc8a0cf54957f, -50.8419540903, 0x1276016c9eaf0d, -54.029899925},
	{-343, 0x1bb5bb789c2329, -54.3915364823, 0x189d573b7e3ebc, -53.2929343309},
	{-342, 0x18066bb85ba52f, -52.6447493762, 0x14ee077b3dc0c2, -55.2006422541},
	{-341, 0x1f4e6b0e1f7cf6, -52.227877764, 0x11d5a33e1fdc55, -52.9004612791},
	{-340, 0x1f4e6b0e1f7cf6, -51.9059496691, 0x148764ce1fc942, -53.633779512},
	{-339, 0x1f4e6b0e1f7cf6, -51.5840215742, 0x1452dcddbfdcd4, -51.763455838},
	{-338, 0x1d1c1b6ad929bc, -52.9438861942, 0x1b99e3b8762743, -52.721666069},
	{-337, 0x1796eddf8e5515, -52.3194279243, 0x1614b62d2b529c, -52.721666069},
	{-336, 0x1e36be204e8777, -52.3689689246, 0x11aa2b5755dbb0, -52.721666069},
	{-335, 0x105a8ce7561082, -49.7887354987, 0x1f9cd9915df971, -51.1436495055},
	{-334, 0x11ff48a1c8bcf6, -52.0156719323, 0x1e0c2f7b1516d4, -54.982541232},
	{-333, 0x1ea068ee4c5ec5, -52.3758430436, 0x10d6d5bb5a2d14, -52.6870565514},
	{-332, 0x1485e8b11643b9, -52.4848340452, 0x1af155f8904820, -52.6870565514},
	{-331, 0x1b320fbde51fd4, -53.3075036425, 0x140475266c9451, -51.4465272921},
	{-330, 0x1b7fbb777e22ac, -50.8137194937, 0x145220e0059729, -55.9682884257},
	{-329, 0x176f4e7de36ad7, -51.4837133173, 0x15acefdde3d470, -56.5532509264},
	{-328, 0x1f0ca8dd0b98c6, -52.595328458, 0x1273d24e1a1827, -51.795154213},
	{-327, 0x1fef87edabea55, -48.6058990869, 0x1138e66b0749be, -51.5727617917},
	{-326, 0x1fa41ed137e7eb, -48.1208820273, 0x10baee76995d94, -52.2926538725},
	{-325, 0x1eee73e7501b63, -50.8771499172, 0x13aebe312ce690, -51.7362605239},
	{-324, 0x1a6e919e753975, -51.9531143769, 0x1c1e601d7729ce, -55.1622634455},
	{-323, 0x154233b6803197, -51.2738052325, 0x1c1e601d7729ce, -55.8403353506},
	{-322, 0x1c29f4483d3fbc, -53.0540653388, 0x15084ee0a1c3f1, -52.3648549262},
	{-321, 0x10f87fa2fb4954, -51.4160374788, 0x1edf291c01917a, -51.9373005597},
	{-320, 0x14d583764b120e, -53.2700641482, 0x18b287499adac8, -51.9373005597},
	{-319, 0x1989f62ba813fb, -52.5111540053, 0x102110c0ee1021, -54.4472547468},
	{-318, 0x1989f62ba813fb, -52.1892259104, 0x1202d83cacddb3, -58.9759098257},
	{-317, 0x1f3ce047549501, -52.1990596716, 0x1202d83cacddb3, -58.6539817308},
	{-316, 0x12be202acc5967, -53.6140971709, 0x1fa200e49cd52e, -50.9930441307},
	{-315, 0x18fd8039107734, -52.8771315768, 0x1ce593b3bca2c1, -52.6194005768},
	{-314, 0x1c6cf879cee406, -51.4998420872, 0x158e07f8520a62, -56.2311765422},
	{-313, 0x1e3449e44c1947, -51.0038464043, 0x145e7c5bfe918c, -52.370075539},
	{-312, 0x1b5bc7a05090df, -53.8785637637, 0x145e7c5bfe918c, -52.0481474441},
	{-311, 0x149bd822c7f01e, -51.8194516199, 0x1eac686d62899b, -52.6386481052},
	{-310, 0x1a023154020a52, -53.6820672492, 0x1711a8f49b9ea6, -52.0680139487},
	{-309, 0x14ce8ddcce6ea8, -53.6820672492, 0x19ab7ab0356c7b, -51.794002711},
	{-308, 0x1d668beb5e4c64, -50.9376083243, 0x1251d7b55dd962, -51.026240341},
	{-307, 0x1a92e0afdb148b, -52.2521966405, 0x1251d7b55dd962, -51.7043122461},
	{-306, 0x1a24ed45951734, -53.2160476346, 0x19490671091c86, -52.3081773814},
	{-305, 0x17c99acb302ce3, -52.0761835533, 0x19490671091c86, -51.9862492865},
	{-304, 0x1171b5c834652f, -53.2017144353, 0x1f030c37bad434, -48.7802242459},
	{-303, 0x1307af08f3571c, -52.7542554584, 0x1f42317087e5ab, -51.3691134259},
	{-302, 0x1d167587a59677, -52.9938116659, 0x1292676a905643, -50.7192283779},
	{-301, 0x18c18504670ea2, -54.9479713284, 0x13ce8579a0769f, -52.8819669218},
	{-300, 0x1eb26c543cfcc3, -53.4641872853, 0x14cc6d3eafccbd, -51.6015361275},
	{-299, 0x1eb26c543cfcc3, -53.1422591904, 0x17f6e6a9a8ce41, -54.5774383805},
	{-298, 0x1d59b7feec26a9, -54.2822105319, 0x129415546575d9, -53.2687986413},
	{-297, 0x100c8aaa2defca, -53.2754890013, 0x1d21e553b3f3cd, -51.6849490482},
	{-296, 0x100c8aaa2defca, -52.9535609064, 0x1077b887bc2975, -53.0981321781},
	{-295, 0x1d960223068592, -49.8844721959, 0x1077b887bc2975, -53.7762040832},
	{-294, 0x1d9e94f28624c3, -51.1741253292, 0x13c2dd6fae9826, -53.1912415825},
	{-293, 0x15b95274a69ae3, -51.7131477102, 0x1a59273f937588, -52.4542759883},
	{-292, 0x1cf7189b88ce84, -50.976182116, 0x131b6118b141e7, -51.4934299208},
	{-291, 0x19770adb8cf4cd, -52.1359090696, 0x16eda7b73b1be2, -51.9084674201},
	{-290, 0x19fa990c70286d, -56.0343466929, 0x12d99a003fa817, -52.8069169929},
	{-289, 0x12f23c139a43c6, -52.0299381131, 0x1c52b80a0cc9fa, -54.15346954},
	{-288, 0x1c8dd69ee63f9e, -50.7366859858, 0x12d4acc92d88f4, -52.603161606},
	{-287, 0x1c5e8af49eae4e, -53.1214420573, 0x12d4acc92d88f4, -52.2812335111},
	{-286, 0x1c5e8af49eae4e, -52.7995139624, 0x1a698e9c63e67c, -52.7537255489},
	{-285, 0x12965072f4a20a, -52.3232633887, 0x1cdb9d3cce698d, -53.0567568815},
	{-284, 0x188ddd9bfdced9, -53.1234443814, 0x1e856ac506fba8, -55.6244831236},
	{-283, 0x188ddd9bfdced9, -52.8015162865, 0x1d30c6580ced2c, -52.6869638373},
	{-282, 0x11c9edcb2b6626, -52.772014232, 0x1b0fbf4349a2cc, -52.1391839459},
	{-281, 0x11dd4cb546b4dd, -53.8089870384, 0x12040a897d524b, -52.7182731203},
	{-280, 0x11cdcd939742b1, -52.3159048963, 0x11eccbd6f62709, -55.4760350438},
	{-279, 0x18e6b9c1272fb3, -52.2958707125, 0x11eccbd6f62709, -56.1541069489},
	{-278, 0x1ead0ee84ca3fb, -52.3724780732, 0x11eccbd6f62709, -55.832178854},
	{-277, 0x1479a60da10cd3, -55.0244009854, 0x1ebfe34096827e, -52.7445418432},
	{-276, 0x1a987e90d4cdeb, -53.3480818861, 0x148c7a65eaeb56, -52.1524821078},
	{-275, 0x1546cba710a4bc, -53.3480818861, 0x19de2d4faf1485, -51.9623745392},
	{-274, 0x1370a496d144d2, -51.736201224, 0x171cf2b75004a6, -54.1936277716},
	{-273, 0x1b79d1a45c195d, -53.1736355015, 0x156dcd797236c8, -52.2882072889},
	{-272, 0x199bdd6a0935d7, -51.5249470508, 0x138fd93f1f5342, -54.4126688494},
	{-271, 0x1c4fd4d47a4c60, -51.6258617195, 0x138fd93f1f5342, -54.0907407545},
	{-270, 0x19bcae7b3df3be, -51.4711384115, 0x138fd93f1f5342, -54.7688126597},
	{-269, 0x1f177ce2d7d4da, -52.0963946065, 0x14e68cd905cb89, -53.3203549964},
	{-268, 0x11b01101431145, -54.7317365495, 0x15f8b620f15ef5, -52.4787973756},
	{-267, 0x15724635bad288, -51.9733205092, 0x15f8b620f15ef5, -53.1568692807},
	{-266, 0x1128382afbdba0, -51.9733205092, 0x1a42c42bb055dd, -53.7086808766},
	{-265, 0x1496a9cd2e3ac0, -51.3883580085, 0x1db135cde2b4fd, -54.9685101773},
	{-264, 0x15f60a74759400, -51.9733205092, 0x199313d80ca93d, -53.8844411987},
	{-263, 0x13bca798ebe98b, -56.2784511775, 0x1f6980172d68ef, -52.4566994208},
	{-262, 0x13bca798ebe98b, -55.9565230827, 0x19213345bded8c, -52.4566994208},
	{-261, 0x1472dbb636af73, -52.5323743605, 0x1f3bf30fdab775, -51.6312395756},
	{-260, 0x1b43cf9d9e3f44, -52.7954087663, 0x1bb6bd9c71ff7e, -52.7661034413},
	{-259, 0x1dad576166a56f, -51.1432869033, 0x112ac28fa8e65e, -50.8708636403},
	{-258, 0x11055d520eedaf, -55.552583125, 0x16be48d7473a79, -52.275540241},
	{-257, 0x1c3b5f2d22c62b, -51.2064118506, 0x1114529de61df5, -52.6180190564},
	{-256, 0x1c3b5f2d22c62b, -50.8844837557, 0x1237cf975bdbb0, -52.2029815572},
	{-255, 0x13abfea4fc3e41, -51.4519059961, 0x1f552405595a86, -55.1691198846},
	{-254, 0x13171f05ef4a07, -53.8529701924, 0x1f0ab435d2e069, -53.3658900301},
	{-253, 0x10b33462c1c58d, -52.9651319832, 0x1a42deef77d775, -54.7123119439},
	{-252, 0x10b33462c1c58d, -52.6432038883, 0x11ab41789f84db, -53.8331909651},
	{-251, 0x10b33462c1c58d, -53.3212757934, 0x1f9c4007f2c8e1, -54.3175559615},
	{-250, 0x1cde1e4e4a2526, -50.7972280976, 0x12f7599e5e7887, -54.7325934608},
	{-249, 0x1bfc04755e7751, -52.0175421932, 0x1949ccd328a0b4, -53.9956278666},
	{-248, 0x188b6346103e58, -51.8451657635, 0x1638606a6698e9, -53.7568868988},
	{-247, 0x11840bcb3bb8ba, -50.4399015062, 0x1f92bac0e4c3f6, -54.5950489138},
	{-246, 0x1a6d1eb8ee319e, -55.6398437005, 0x1781c76c643d7f, -52.9048236945},
	{-245, 0x1c25b6e53cd7fc, -52.0565023409, 0x1e8b3525b3737e, -55.4202314946},
	{-244, 0x1376a69a05213d, -52.0390891805, 0x1e8b3525b3737e, -55.0983033997},
	{-243, 0x12ec8ee75a06fc, -50.9964541779, 0x1e8b3525b3737e, -54.7763753049},
	{-242, 0x1015dfde3fb923, -51.9089913367, 0x1f746182e6cd5d, -48.1098554255},
	{-241, 0x10a7361344fbe8, -51.5370225593, 0x1f746182e6cd5d, -47.7879273306},
	{-240, 0x122ac6a0a858a0, -51.0895635823, 0x1fb98ffb19d859, -50.9985976553},
	{-239, 0x18cd2469253c4a, -53.8227263958, 0x19613ffc14ad14, -51.9985976553},
	{-238, 0x15f95d06783f63, -51.6429929814, 0x1ba0ebcbd23931, -52.800750504},
	{-237, 0x183c62ef02a34f, -52.5701444786, 0x132c49d564f21a, -52.3446271246},
	{-236, 0x13638258cee90c, -53.5701444786, 0x1ca6d27e626fae, -51.7948869267},
	{-235, 0x1cff2cea3f2dfe, -54.204446973, 0x12defab703cb94, -50.9532018493},
	{-234, 0x136c583064fc14, -52.4472240611, 0x1800bf7fcda2d9, -52.4322155787},
	{-233, 0x174ed03a12c818, -51.8622615604, 0x1dd4738e5254df, -52.6184759403},
	{-232, 0x1e940800fcc206, -53.3686632286, 0x1d9497681985d2, -52.5737198242},
	{-231, 0x1e940800fcc206, -53.0467351337, 0x17aa12b9ae04a8, -52.5737198242},
	{-230, 0x1a233c8c3a2259, -53.9681303868, 0x1141a5a2b4e2ff, -53.1733068322},
	{-229, 0x1536b559fb78e7, -52.2892817407, 0x1b9c3c3787d198, -53.1733068322},
	{-228, 0x11c7d1621ef374, -50.3584383234, 0x1d178daa5fbb37, -53.7648759749},
	{-227, 0x11904b42b92492, -51.7947474962, 0x17460aeeb2fc2c, -53.7648759749},
	{-226, 0x1ff62add89c00a, -49.9841623093, 0x105b2c38dc19b1, -48.2505669489},
	{-225, 0x1606adcf3d8837, -55.2557849882, 0x1f7a84d997bbb0, -50.9787945553},
	{-224, 0x102707fe6052d3, -55.3813158703, 0x1fefb6588b9880, -48.5795951201},
	{-223, 0x107236af382f18, -56.0333925668, 0x1fa487a7b3bc3b, -49.2574319242},
	{-222, 0x12cbac35f71140, -55.518819394, 0x18a77ca2f4d677, -51.3327075813},
	{-221, 0x1073bf3d918f2a, -52.0789905455, 0x1d5d78cd255aeb, -53.9700012603},
	{-220, 0x18e862e9b696e6, -53.2879000703, 0x155e5d3bb8d37d, -52.0994541369},
	{-219, 0x18e862e9b696e6, -52.9659719754, 0x10050fc4671b77, -53.6752445777},
	{-218, 0x105efaac0be0f2, -55.3304625138, 0x1f5649b984abf8, -51.1808341997},
	{-217, 0x105efaac0be0f2, -56.0085344189, 0x1dfb1fc8074d0f, -52.2116428107},
	{-216, 0x1d81c48cf38db6, -51.3165399366, 0x11fd13119dfb09, -52.62668031},
	{-215, 0x11cc882d2fae7f, -52.9074326133, 0x1dca94e3990085, -59.245979748},
	{-214, 0x1c7a737b7f7d98, -52.9074326133, 0x1dca94e3990085, -59.9240516531},
	{-213, 0x1f7bf879dcdc07, -48.1559001626, 0x104804a46f8aae, -48.9421371437},
	{-212, 0x160c20b7dcd37a, -51.8153891841, 0x1825efc4d271c2, -54.2749785955},
	{-211, 0x1e8d64e115baf9, -54.5742718535, 0x11be7aa88f288b, -53.230479279},
	{-210, 0x11581fa3d81ef6, -53.3921590333, 0x1ad0e57f32419b, -51.6712362366},
	{-209, 0x1d534494a630cd, -51.6591874449, 0x1016f0191e275d, -52.0862737359},
	{-208, 0x1908e26bd15981, -52.0762261011, 0x112988a3535d30, -52.6712362366},
	{-207, 0x1908e26bd15981, -51.7542980062, 0x17be617a966e5a, -54.7018059502},
	{-206, 0x1382e7f5c31cbe, -52.954600326, 0x1bf9daff69bff6, -52.1516250914},
	{-205, 0x18d92d2bcc7a81, -62.5131574306, 0x155914d825770c, -51.6357366157},
	{-204, 0x198c653c877b32, -53.6311424762, 0x16bf84f99b786e, -52.0512854792},
	{-203, 0x18fd052f25143e, -55.6159334843, 0x16bf84f99b786e, -51.7293573843},
	{-202, 0x18fd052f25143e, -55.2940053894, 0x14b2847f99af2b, -51.7416021444},
	{-201, 0x104e4f67de8015, -50.3756666008, 0x1f4c09c3ddffc8, -48.3302754816},
	{-200, 0x112a02be2d944c, -49.9797379245, 0x1ee2681ef516e9, -50.9838305243},
	{-199, 0x17ae0c186d8709, -59.8583542983, 0x18b52018c41254, -50.9838305243},
	{-198, 0x17ae0c186d8709, -60.5364262034, 0x1e0f61b7744db6, -50.983062277},
	{-197, 0x1f92bacb3cb40c, -59.7994606093, 0x13ead5c8fdb27f, -51.9846632541},
	{-196, 0x1fc0af700b2e1a, -48.7318541028, 0x1085e18d7d61e4, -49.6595349796},
	{-195, 0x1edacf1a3f1ec9, -50.0844979131, 0x1085e18d7d61e4, -50.3376068847},
	{-194, 0x12f379dcb1cd12, -53.7201384505, 0x198f6daee3f43a, -51.6447736996},
	{-193, 0x12f379dcb1cd12, -53.3982103556, 0x1dec7b13dc1051, -51.8763517157},
	{-192, 0x1325ee5a121794, -50.5288499125, 0x1dec7b13dc1051, -52.5544236208},
	{-191, 0x1d0b90f4b4e024, -54.039703712, 0x10479cb8a8d06c, -50.9523617111},
	{-190, 0x109cf5899748dd, -53.6822018107, 0x1080829947cb62, -52.6135822807},
	{-189, 0x1df2b4be301bbf, -52.7566911982, 0x13ab45fdf6c3e2, -51.7201569702},
	{-188, 0x15e7140b4e37d3, -54.8326581099, 0x1a39b2a7f3afd8, -50.983191376},
	{-187, 0x1d341ab9bd9fc4, -54.0956925157, 0x1e116da5deb7c5, -53.7037029723},
	{-186, 0x17045a9c8a7636, -53.9872678404, 0x1a79a64d0ed63a, -51.862278708},
	{-185, 0x18f90f4dbe453b, -52.9549908094, 0x11263c88ef0927, -53.8036234533},
	{-184, 0x17687ebffb3937, -53.1706169884, 0x11263c88ef0927, -53.4816953584},
	{-183, 0x1cbd9c24dc7aac, -51.9652000482, 0x1942c86451166d, -54.2309261639},
	{-182, 0x1b5947717185c6, -52.5955325618, 0x18909e0a9b9bfa, -52.7967698211},
	{-181, 0x10f74cbf087f06, -52.311668766, 0x18909e0a9b9bfa, -52.4748417262},
	{-180, 0x10f74cbf087f06, -52.9897406711, 0x1f873427cf3c19, -59.1028245324},
	{-179, 0x1365cd292317a6, -54.010735344, 0x1d1fced18d7a26, -51.6471190491},
	{-178, 0x1746f6315d4f94, -53.4257728433, 0x1b2f3a4d705e2f, -51.2974277456},
	{-177, 0x1a5ea5c99b8cda, -52.3601735346, 0x11d9f55d03f903, -55.6555865478},
	{-176, 0x1ff8358d0fc98e, -53.8862612096, 0x177385207835b7, -52.3173878539},
	{-175, 0x14e2a9d0c1634d, -53.0857796764, 0x162b17789ccc82, -53.9104627169},
	{-174, 0x1138b3b6f27986, -55.2853870153, 0x1c65e8e22288b3, -51.2438185274},
	{-173, 0x1df8ebc304ecfa, -52.7956055074, 0x1374f18c627c8f, -54.6452618291},
	{-172, 0x1fc2b6d42b55ce, -53.0110667295, 0x170887aeaf4e37, -52.6266021676},
	{-171, 0x1fc2b6d42b55ce, -52.6891386346, 0x18c75de961b622, -52.9333107097},
	{-170, 0x1efad528beec29, -48.7129999984, 0x1039021db5b22f, -50.6018135338},
	{-169, 0x1ec713652b5b9b, -51.802772522, 0x114de01fb0be10, -50.1867760345},
	{-168, 0x13c3afa57a786b, -52.9148278947, 0x1c6a9fb552d773, -51.2636029599},
	{-167, 0x1a5a3f874df5e4, -53.1778623005, 0x1c7ee92d4702cf, -54.9523373689},
	{-166, 0x143988904125f2, -52.458766214, 0x1882dbdc333fc8, -52.5137424136},
	{-165, 0x1b2642311eb67f, -52.2769036486, 0x1bb2e683764234, -53.5708968482},
	{-164, 0x1a999ddec72aca, -51.6872476074, 0x137fc2394ab0af, -53.7734566962},
	{-163, 0x1a3d40ed583810, -53.6059008093, 0x19e0e3fbe94556, -52.159181704},
	{-162, 0x17a6c3ba8db121, -52.936207923, 0x14b3e996543778, -52.159181704},
	{-161, 0x1a999ddec72aca, -51.7214633228, 0x15e1da719e6822, -54.2850826643},
	{-160, 0x1d63f64322e181, -57.5647528736, 0x1cbf7d4033dd86, -51.9049226424},
	{-159, 0x100fd149576383, -46.910880728, 0x1fecf70ba54fd3, -46.170645387},
	{-158, 0x1030b649ed9782, -47.5774569943, 0x1fd634594433ab, -47.6715458582},
	{-157, 0x130155d9f0647a, -53.570997582, 0x1bfe6cc48fc6bb, -53.5306844809},
	{-156, 0x11007c76e030d5, -54.5915845806, 0x19fd93617f9316, -52.5406576427},
	{-155, 0x11007c76e030d5, -55.2696564857, 0x152ebcda257d8a, -52.3006526687},
	{-154, 0x1621614b848c15, -52.6455609829, 0x17bf2f4477ab2a, -52.3958523571},
	{-153, 0x1d93e5783f78d9, -54.6063066192, 0x17bf2f4477ab2a, -52.0739242622},
	{-152, 0x11bf234826154f, -56.0213441185, 0x1dea9109530931, -50.6209741849},
	{-151, 0x17a9846032c714, -55.2843785243, 0x169d4f2f04dd43, -51.6731442251},
	{-150, 0x13594bfa3afcca, -52.356432356, 0x1bf9bcc62a915e, -52.930981197},
	{-149, 0x1ef5465d2b2e10, -52.356432356, 0x1de9112bfd443f, -53.8416884287},
	{-148, 0x133416ef024187, -51.5344047848, 0x1de9112bfd443f, -53.5197603338},
	{-147, 0x133416ef024187, -51.2124766899, 0x1a7d21dca8c7de, -55.0758724836},
	{-146, 0x1d9f45f479f271, -50.5652482666, 0x1011f2d73116f4, -52.5270639779},
	{-145, 0x1f47a4e15b3a88, -47.7468800331, 0x1011f2d73116f4, -53.205135883},
	{-144, 0x1fe9aca55031cc, -45.3781376419, 0x1011f2d73116f4, -52.8832077881},
	{-143, 0x1ffd8bfe990279, -45.3821339664, 0x1011f2d73116f4, -52.5612796932},
	{-142, 0x1fb97e0b2e86f9, -48.4627917332, 0x10b681ac997408, -53.1827680699},
	{-141, 0x1bf17204bbec05, -50.8685833338, 0x1491daad0ba280, -52.5612796932},
	{-140, 0x199379e16ce9c2, -52.9931473133, 0x1491daad0ba280, -52.2393515983},
	{-139, 0x160467739b5dbb, -55.1341505688, 0x1dcc852ff08b5a, -52.8045342397},
	{-138, 0x1d008dbbe7a4ac, -52.1293406233, 0x168c61c0f6a22f, -52.0827654551},
	{-137, 0x1fcbd03f395aa6, -51.3876492648, 0x149ec631f282be, -58.059051973},
	{-136, 0x14cb1ac9353a73, -54.4380591379, 0x1419c86c2a5b9f, -52.7585476087},
	{-135, 0x1ee9620b99fb95, -50.492228321, 0x1014a05688494c, -52.7585476087},
	{-134, 0x1c71fac192adee, -52.5500668046, 0x16434d8c0d7b9d, -57.1616437811},
	{-133, 0x1448835240109e, -54.6103212616, 0x19fa93778b6047, -51.5480280563},
	{-132, 0x1448835240109e, -54.2883931667, 0x1ade36f84f8535, -51.6142954288},
	{-131, 0x1dd1bd7460525a, -51.7796610109, 0x19a827ea4ff80f, -55.8212240029},
	{-130, 0x155ba43d76721b, -55.0697975227, 0x11320eb36617d0, -52.6142954288},
	{-129, 0x155ba43d76721b, -54.7478694278, 0x1a923b1acbf11d, -52.8264209649},
	{-128, 0x1ac5f1d13f538b, -52.3307530902, 0x1541c8e23cc0e4, -52.8264209649},
	{-127, 0x1fdc523fd1da84, -48.7894933557, 0x105bf1a058f8f0, -50.2854150337},
	{-126, 0x1f25f7211e6f76, -58.261551891, 0x119e79c49a56d9, -52.9819388948},
	{-125, 0x11defb6574366e, -53.9095061683, 0x1a4d75d67a927b, -51.5157701367},
	{-124, 0x1efc48e1b6db87, -51.2938842012, 0x1170d301b5bf8d, -49.7484889237},
	{-123, 0x12975eedd41d51, -51.7089217005, 0x1f216674079388, -52.9076445301},
	{-122, 0x179b61f04f1967, -52.6189636787, 0x166ceff8d8b662, -52.4403017776},
	{-121, 0x1ccd09835b12e4, -52.2594892804, 0x1da03259af7b1b, -53.9898396319},
	{-120, 0x170a6e02af4250, -52.2594892804, 0x1abee4995992d1, -53.2545250069},
	{-119, 0x1885d0a559c8ea, -53.2619778377, 0x1b7c95eaaed61e, -52.2557644748},
	{-118, 0x16a9c9a2ae9ddd, -55.6853416672, 0x1e735b3003e352, -51.6131488186},
	{-117, 0x1f4f26e0908b81, -50.2726521411, 0x1121d9d088659b, -50.0704024535},
	{-116, 0x1181f1d2ffc06a, -53.8872385225, 0x1d6058283b5601, -53.1193317205},
	{-115, 0x1c031c84cc6710, -52.8872385225, 0x12df2d766eaf5b, -52.361922932},
	{-114, 0x1476a3ef068752, -52.9093181361, 0x195443230f0325, -55.0250885793},
	{-113, 0x1f91783aa7d48b, -53.1834317343, 0x195443230f0325, -54.7031604844},
	{-112, 0x11b170b812ef3f, -54.4598744671, 0x1e8b49abf04f98, -51.4200176419},
	{-111, 0x1c4f1ac01e4b98, -54.4598744671, 0x177782fbe8bcb2, -52.0295734016},
	{-110, 0x11225a6b7f9684, -51.1257226276, 0x1bc01996c8a253, -52.3838963885},
	{-109, 0x1d0dd23d39f144, -54.7352370732, 0x1bc01996c8a253, -52.0619682936},
	{-108, 0x102a24dcb03946, -48.8976686114, 0x1fc678f1e974b7, -48.2131429896},
	{-107, 0x17f924134f162e, -52.7891824224, 0x15e32fd5ccfe46, -52.3102568035},
	{-106, 0x10929a6dc3f389, -53.5855757336, 0x141da8b31f5007, -52.833881213},
	{-105, 0x13e31fb6eb243e, -54.0006132328, 0x1ebed383316581, -50.9804551926},
	{-104, 0x1a842a493985a8, -53.2636476387, 0x1c353ac6cfdf32, -52.3888354537},
	{-103, 0x1808a7b4dc67a5, -53.6392587824, 0x1c353ac6cfdf32, -52.0669073588},
	{-102, 0x165d39adae37d3, -52.9446258663, 0x19b415bc0a9777, -55.0796296463},
	{-101, 0x11e42e248b5fdc, -52.9446258663, 0x1d0af1ca66f71b, -52.0813477548},
	{-100, 0x19ee1f3aff15ee, -56.4118972181, 0x1d0af1ca66f71b, -51.7594196599},
	{-99, 0x137fc4c2a28446, -53.5968421852, 0x169c97520a6573, -53.1937063312},
	{-98, 0x1f32d46a9da070, -52.5968421852, 0x169c97520a6573, -52.8717782363},
	{-97, 0x135952a4b4d5c4, -51.3786890893, 0x1c895834fecac9, -51.9448356362},
	{-96, 0x1327778a47d9d2, -55.0020361026, 0x1c895834fecac9, -52.6229075413},
	{-95, 0x1cb13ab0892df1, -57.3279807382, 0x12eba3d0f84516, -51.8639857341},
	{-94, 0x132b7496a27d56, -53.7388345833, 0x1ca1467f1e9fe1, -53.3689271255},
	{-93, 0x136b455c4cb596, -51.6978238605, 0x16e76b98e54cb4, -54.3689271255},
	{-92, 0x1eebdef88d406d, -50.1163497211, 0x1278e61571b913, -51.4964892599},
	{-91, 0x1bac942c0bc1bf, -53.4083289726, 0x151c74aacb1ca8, -50.9819160871},
	{-90, 0x1a13bca3fcfe0d, -51.8768442193, 0x13839d22bc58f6, -53.2788639685},
	{-89, 0x14dc96e99731a4, -51.8768442193, 0x18bac2dd22255f, -54.0417365433},
	{-88, 0x1e48ce4c82bfa4, -50.1884405802, 0x11288f4a017576, -51.759816615},
	{-87, 0x1edb3979a4ee39, -49.8827855368, 0x1003b8efbd184c, -51.5374241937},
	{-86, 0x1cd08c114d9cda, -54.330744304, 0x124d65a445d2a0, -52.0228510209},
	{-85, 0x14b6f6b07847c3, -51.9834482522, 0x189c164c174bf6, -53.5434811622},
	{-84, 0x10925ef3936c9c, -51.9834482522, 0x1c8135e7b65029, -50.9145309742},
	{-83, 0x10925ef3936c9c, -52.6615201573, 0x1ffd1b35f2b3d3, -48.9236783789},
	{-82, 0x113c10d45be198, -52.283008534, 0x1f5369552a3ed7, -48.597431298},
	{-81, 0x15363c190e9f80, -51.6615201573, 0x1ffc0abc36fac3, -51.8486540008},
	{-80, 0x1766558b03b40e, -55.0894996738, 0x1bc6886eeddd2a, -51.7035881307},
	{-79, 0x15a6412fd93d36, -52.286946493, 0x192669e62e2ae6, -52.9270509695},
	{-78, 0x17bff336d8ff06, -58.0239012664, 0x1a8ce0958356c6, -51.5883578622},
	{-77, 0x17bff336d8ff06, -58.7019731715, 0x177b0b29be2538, -51.580047852},
	{-76, 0x10adbf0f6a1147, -53.534175487, 0x1ed2275e47ecc5, -53.6381701788},
	{-75, 0x10adbf0f6a1147, -53.2122473921, 0x11fc397875834e, -53.0177033089},
	{-74, 0x1d7fc08b53f17d, -52.4801166144, 0x17267175012666, -51.9097565457},
	{-73, 0x1dc5c95b76a94c, -53.4656605031, 0x1e0bd22b99611b, -54.5394346579},
	{-72, 0x1ffc2f5c460888, -44.2560129882, 0x100804718c132f, -46.7101157732},
	{-71, 0x1ff31398d39795, -44.0710467182, 0x100804718c132f, -47.3881876784},
	{-70, 0x1ff3d502470231, -48.392394316, 0x103209bb3a8178, -47.0515630723},
	{-69, 0x1ea3b4607fbba4, -55.4806832513, 0x176ad56230f2d3, -52.0351358031},
	{-68, 0x1e82149cd1c111, -50.6573269743, 0x1031f663e22a02, -51.6484709246},
	{-67, 0x12ae6acd14c46e, -52.6662377233, 0x1fa32fe9c9b2d6, -52.6309202651},
	{-66, 0x14b334dac8f84f, -54.1174790018, 0x1cc65d1199c7d3, -51.0010874808},
	{-65, 0x159c2f57040a2a, -52.4538883503, 0x10cb8422df45ff, -54.5311960262},
	{-64, 0x105f12e47d015f, -50.0596310912, 0x1f8c50d66d05f1, -48.7954910865},
	{-63, 0x105f12e47d015f, -49.7377029963, 0x1ffac9f48ab0fd, -49.1434341885},
	{-62, 0x11559b88845bce, -50.3333127413, 0x1ec8b69fc9dc55, -50.583705211},
	{-61, 0x14c9c849c4a726, -52.7233515707, 0x177a674e6c2741, -52.2255463808},
	{-60, 0x10a16d07d085b8, -52.7233515707, 0x1fcb1dd2546a1d, -52.3841233535},
	{-59, 0x1a50bc003e4f49, -53.4800114127, 0x18d8ffc2f1adf6, -52.188483425},
	{-58, 0x1a50bc003e4f49, -53.1580833178, 0x1efdc9815f51d0, -55.6388537618},
	{-57, 0x1a50bc003e4f49, -52.8361552229, 0x1387aecdd964cb, -54.3785337527},
	{-56, 0x163e80e201c297, -54.8800344728, 0x10d0dcb9b106ff, -52.6975921098},
	{-55, 0x1c8bb292164ed1, -51.5482398836, 0x12385a8233abd8, -51.5468358766},
	{-54, 0x16d62874dea574, -51.5482398836, 0x1487417b8928a6, -54.5412334723},
	{-53, 0x1da9b810af046e, -53.0920042806, 0x1117305994f34d, -51.569288678},
	{-52, 0x1eeb92bb25db71, -49.5412822765, 0x123ade3d6bae30, -52.1542511787},
	{-51, 0x1c0f5b3bef301d, -57.2421287604, 0x123ade3d6bae30, -51.8323230838},
	{-50, 0x1c0f5b3bef301d, -56.9202006655, 0x19ced6491c3832, -51.8493916605},
	{-49, 0x13de005bd620df, -55.5413500628, 0x19ced6491c3832, -52.5274635656},
	{-48, 0x13de005bd620df, -55.2194219679, 0x1c97dec59f6d4e, -52.6196519388},
	{-47, 0x13de005bd620df, -54.897493873, 0x12e6b22554b745, -52.561347642},
	{-46, 0x13de005bd620df, -55.5755657782, 0x164b834a31e4e2, -52.4916322002},
	{-45, 0x15e5fe15eabdd6, -52.1188605591, 0x1cc901acb6e2d3, -53.9956056191},
	{-44, 0x15e5fe15eabdd6, -51.7969324642, 0x1937cdf2c3620b, -54.8820570071},
	{-43, 0x1b9b1da5d6bf16, -52.884541575, 0x16d47e3fb00500, -52.4916322002},
	{-42, 0x1b9b1da5d6bf16, -52.5626134801, 0x1169ee9806d4ed, -54.7132357299},
	{-41, 0x120204e58bd4ce, -53.9307464024, 0x19d2dabd47bf73, -52.9647538251},
	{-40, 0x1cd007d5ac87b0, -53.9307464024, 0x1638d9c2922e7d, -53.5021972837},
	{-39, 0x1763561174706f, -51.4001191199, 0x1469f3d67f7c12, -56.4624054899},
	{-38, 0x1bba4aae6e1ca3, -52.500698216, 0x1469f3d67f7c12, -56.1404773951},
	{-37, 0x191b91aeb78a56, -52.8776618228, 0x14f5c7fdb49268, -52.2860103789},
	{-36, 0x191b91aeb78a56, -52.555733728, 0x184ce54c2deb6d, -52.6988251045},
	{-35, 0x1dce69dd72000a, -53.425562778, 0x184ce54c2deb6d, -52.3768970096},
	{-34, 0x17d854b1280008, -54.425562778, 0x1256d01fe3eb6b, -52.921822337},
	{-33, 0x16612ce4e272cb, -52.4706862805, 0x10dfa8539e5e2e, -54.5491775902},
	{-32, 0x1673656b3fac76, -54.3962459227, 0x16859df19ce621, -52.937574002},
	{-31, 0x1f71ffb1056361, -55.6951636955, 0x1221a59845e12c, -52.0393053509},
	{-30, 0x1f71ffb1056361, -55.3732356006, 0x106133e4c8dc7b, -48.895718761},
	{-29, 0x1fdd940e0fd3d3, -47.6352701507, 0x102fdef429e2ec, -48.5908641795},
	{-28, 0x18eccd0678bda2, -53.6881190763, 0x1a77748b708a1a, -52.067445853},
	{-27, 0x1f03da2a578b4d, -52.053622648, 0x1e86d2a35a688c, -55.5835341052},
	{-26, 0x1884dd3747c0fd, -54.4874874177, 0x1807d5b04a9e3c, -51.8232360072},
	{-25, 0x1fec6d2229cb29, -53.1413117766, 0x10233e3e6b714f, -50.2866511397},
	{-24, 0x1989f0e8216f54, -54.1413117766, 0x15fdae4e97418e, -51.9189193552},
	{-23, 0x1173b881009176, -51.8193836817, 0x1eb78a8f178a49, -52.4043461824},
	{-22, 0x10f0cf064dd592, 0, 0x13150f8e6c7ea5, -51.0824180875},
	{-21, 0x10f0cf064dd592, 0, 0x11220440f54898, -48.7604899926},
	{-20, 0x1043561a882930, 0, 0x1003f5a37d72b5, -46.4385618977},
	{-19, 0x100f4b6d667579, 0, 0x1004a60e838ce3, -44.1166338029},
	{-18, 0x1004e2e45fb7ee, 0, 0x1002fc357e5f96, -41.794705708},
	{-17, 0x1000002ea9cf74, -39.4727776131, 0x10000eb3cb11a2, -39.4727776131},
	{-16, 0x100007713a708b, 0, 0x100000f48a8526, -37.1508495182},
	{-15, 0x100000563d26fe, 0, 0x10000548a8183e, -34.8289214233},
	{-14, 0x10000071bcfcbc, -32.5069933284, 0x1000003abd5140, -32.5069933284},
	{-13, 0x100000563d26fe, 0, 0x1000003100439b, -30.1850652335},
	{-12, 0x10000005cab5dc, -27.8631371386, 0x100000069d66a5, -27.8631371386},
	{-11, 0x10000000311663, -25.5412090438, 0x1000000092caaa, -25.5412090438},
	{-10, 0x10000000ac7203, 0, 0x1000000045d49f, -23.2192809489},
	{-9, 0x1000000009e31e, -20.897352854, 0x10000000072d91, -20.897352854},
	{-8, 0x10000000058d67, 0, 0x1000000000224d, -18.5754247591},
	{-7, 0x1000000001f9e0, 0, 0x10000000009175, -16.2534966642},
	{-6, 0x10000000002049, -13.9315685693, 0x100000000002e7, -13.9315685693},
	{-5, 0x10000000000563, 0, 0x100000000007a8, -11.6096404744},
	{-4, 0x10000000000131, -9.28771237955, 0x10000000000242, -9.28771237955},
	{-3, 0x10000000000025, -6.96578428466, 0x10000000000060, -6.96578428466},
	{-2, 0x1000000000000b, -4.64385618977, 0x10000000000016, -4.64385618977},
	{-1, 0x10000000000004, 0, 0x10000000000001, -2.32192809489},
	{0, 0x10000000000000, 0, 0x10000000000000, 0},
	{1, 0x10000000000000, 0, 0x10000000000000, 0},
	{2, 0x10000000000000, 0, 0x10000000000000, 0},
	{3, 0x10000000000000, 0, 0x10000000000000, 0},
	{4, 0x10000000000000, 0, 0x10000000000000, 0},
	{5, 0x10000000000000, 0, 0x10000000000001, -1},
	{6, 0x10000000000000, 0, 0x10000000000007, -3},
	{7, 0x10000000000000, 0, 0x1000000000001b, -6},
	{8, 0x10000000000000, 0, 0x100000000000df, -8},
	{9, 0x10000000000000, 0, 0x10000000000393, -10},
	{10, 0x10000000000000, 0, 0x100000000000b7, -13},
	{11, 0x10000000000000, 0, 0x1000000000468b, -15},
	{12, 0x10000000000000, 0, 0x1000000001c14f, -17},
	{13, 0x10000000000000, 0, 0x100000000cc043, -20},
	{14, 0x10000000000000, 0, 0x100000003f59a7, -22},
	{15, 0x10000000000000, 0, 0x10000000bfdebb, -24},
	{16, 0x10000000000000, 0, 0x10000003265fbf, -27},
	{17, 0x10000000000000, 0, 0x1000001a3adff3, -29},
	{18, 0x10000000000000, 0, 0x100000053ef997, -31},
	{19, 0x10000000000000, 0, 0x1000039aa631eb, -34},
	{20, 0x10000000000000, 0, 0x10000652213d2f, -36},
	{21, 0x10000000000000, 0, 0x10000476d372a3, -38},
	{22, 0x10000000000000, 0, 0x100040e490b087, -41},
	{23, 0x10000000000000, 0, 0x1007a6941cf01b, -43},
	{24, 0x10000000000000, 0, 0x100e548405c99f, -45},
	{25, 0x10000000000000, 0, 0x10a2ddb4012853, -48},
	{26, 0x10000000000000, 0, 0x13ba2c57336e77, -50},
	{27, 0x10000000000000, 0, 0x18bed5ab0a494b, -52},
	{28, 0x158165ff8d41a6, -52.4150374993, 0x11bfc45568750f, -55},
	{29, 0x11f4a521ab90a3, -52.0458036896, 0x1c6606ef0d8818, -54},
	{30, 0x1b39fabe9d6262, -51.524266569, 0x13b6b76a53c934, -51.5737352453},
	{31, 0x1b2dae8672e9b1, -55.0692626624, 0x18b0b17d133432, -53.0574854947},
	{32, 0x116aeb419df679, -50.6086221377, 0x1e0c8da47fd507, -51.0705566423},
	{33, 0x11fdcb2518d59a, -54.5395441037, 0x1a07a224b2d538, -51.3866709456},
	{34, 0x1d2b4adc88b3d6, -51.6921232638, 0x1108bc92266a6b, -50.9858378309},
	{35, 0x1e2620eae320d6, -55.1327695311, 0x15e35be31a87c9, -53.158662934},
	{36, 0x13bdd4c1b06583, -54.1456581417, 0x15e35be31a87c9, -52.8367348391},
	{37, 0x13bdd4c1b06583, -53.8237300468, 0x1d5afc5d18a989, -53.3569175819},
	{38, 0x100779b9aefdb4, -47.0112311904, 0x1fe7f0ef7457de, -45.4684009053},
	{39, 0x100779b9aefdb4, -46.6893030955, 0x1fea7b74ec0951, -45.7390538731},
	{40, 0x1981ee0941780e, -51.8744341385, 0x1918b666f4c6e2, -54.4645530477},
	{41, 0x19eb25ab8e293a, -51.4374248193, 0x16571c61ae48b1, -52.7884859758},
	{42, 0x1dabc316b77142, -51.7521606512, 0x16571c61ae48b1, -52.4665578809},
	{43, 0x11c7836f977a6c, -51.4853402492, 0x1dc97b2ce860ec, -51.7295922867},
	{44, 0x1e7d819bc80dfb, -51.0135995733, 0x1446150d9b9b74, -52.3553934033},
	{45, 0x1e7d819bc80dfb, -50.6916714785, 0x103810d7afaf90, -52.3553934033},
	{46, 0x12af076163189e, -52.2388100636, 0x11738c1c896417, -55.9475664454},
	{47, 0x1e6309ea8f3c66, -51.8949813154, 0x1bec1360dbd358, -55.9475664454},
	{48, 0x1e398ad2edf940, -49.8312061435, 0x11d1fa969c4a67, -51.923450423},
	{49, 0x1fd12ae94a80e3, -50.0248958639, 0x11d1fa969c4a67, -51.6015223281},
	{50, 0x14df8b0891225e, -52.7219274812, 0x1ecd763068e637, -52.6475393013},
	{51, 0x16c1f4cad8ed45, -56.0650333517, 0x18a45e8d20b82c, -52.6475393013},
	{52, 0x1fdc0d8a2f0525, -47.2082027449, 0x1034c0ba030429, -49.0922019273},
	{53, 0x10c62189813817, -53.14609827, 0x13c9fdf38dafef, -53.2488065181},
	{54, 0x1555b39c39a672, -54.0502185744, 0x1ce92218fe8ca5, -51.7701485687},
	{55, 0x13d1d0b678ab9b, -52.6096629442, 0x16d99681faa149, -56.2728258523},
	{56, 0x181c6a0b402fd5, -51.8575974144, 0x14b449d796df2c, -53.3981858231},
	{57, 0x181c6a0b402fd5, -51.5356693195, 0x10903b12df18f0, -53.3981858231},
	{58, 0x1d853e1b256442, -55.0501480815, 0x1f52da2071cb11, -53.2726716307},
	{59, 0x14bb380d3d4550, -51.7490697649, 0x18567017d612ee, -54.7702252875},
	{60, 0x172a4342d2ad9b, -52.4450232037, 0x137859acab4258, -55.7702252875},
	{61, 0x13456486b06d23, -52.0945900811, 0x1f26f5e11203c0, -54.7702252875},
	{62, 0x1f69d83d8cb31b, -50.3429611489, 0x10a231e4ef060f, -51.0774522692},
	{63, 0x1fef9cf68211d1, -49.3812458963, 0x111bdb8c639134, -51.7148821898},
	{64, 0x10a7098588f449, -52.6857879468, 0x1841a3fcd1a699, -56.1770029151},
	{65, 0x102e19ce07e298, -47.2103790774, 0x1fdc3e741a58e9, -52.1276961544},
	{66, 0x13ecd9f65d00ad, -52.9404353128, 0x1906a6410d89cc, -52.0716676666},
	{67, 0x13ecd9f65d00ad, -52.6185072179, 0x11fb00162a9dca, -54.2160428324},
	{68, 0x1e5314d6d2e52c, -53.1972240151, 0x198fc54bdf5715, -54.6437639843},
	{69, 0x15a7da9f0844a6, -52.2273825247, 0x1add13857c5d3a, -53.2581849613},
	{70, 0x1fcc4409a8d0d9, -53.8634068537, 0x1c8223b18b2e6f, -53.9613812633},
	{71, 0x1970366e20a714, -53.8634068537, 0x1c8223b18b2e6f, -53.6394531684},
	{72, 0x1f7dc27f8e420f, -51.5228820233, 0x126b8f947f2c22, -50.8401981502},
	{73, 0x1f5542ece076c2, -53.1171129274, 0x126b8f947f2c22, -50.5182700553},
	{74, 0x1d00eb2c4979ea, -50.7854545683, 0x14a4501cf9cbab, -60.2247810718},
	{75, 0x161f7b22a0e0b1, -51.4628339593, 0x160499b881ea50, -60.8097435725},
	{76, 0x160f5a495b19aa, -52.4661610629, 0x15f9d927a8baf6, -52.4551004273},
	{77, 0x160f5a495b19aa, -52.144232968, 0x15fe262e66011a, -52.8664698121},
	{78, 0x160f5a495b19aa, -52.8223048731, 0x13ccdac60de9d9, -54.1996536753},
	{79, 0x1aec85d8aff529, -51.9377215038, 0x166786d2159587, -52.1298041121},
	{80, 0x1b837df6706378, -52.4354528322, 0x1cb16e31f14016, -53.3514469148},
	{81, 0x182c2c733e519d, -52.3780118464, 0x13d969e3dbe723, -57.0571791017},
	{82, 0x1ebc832a0bbfba, -51.3640019654, 0x13d969e3dbe723, -56.7352510068},
	{83, 0x1ae48c2f3c4223, -54.8579430548, 0x199c8f30f71846, -53.1090715133},
	{84, 0x1a206552c2cb9e, -51.4916697494, 0x15007159ae242a, -55.6185720624},
	{85, 0x1576c666da4113, -52.2373626183, 0x10cd277af1b688, -55.6185720624},
	{86, 0x10c908eadebf8e, -49.93522377, 0x1fc5195145f066, -48.7922151732},
	{87, 0x10c908eadebf8e, -50.6132956751, 0x1ff18c58171be1, -50.150887524},
	{88, 0x1bb29b1624934a, -51.8573272143, 0x143cd00316e8d2, -53.5198507938},
	{89, 0x1db8afe340775f, -51.5817071685, 0x1030a668df20a8, -53.5198507938},
	{90, 0x1146fe1075e1ef, -57.1100640553, 0x1e349d8290bec2, -51.1507860656},
	{91, 0x1f1f7bcc38a0c8, -52.0608957282, 0x16e22db4568793, -53.6290209747},
	{92, 0x18e5fca36080a0, -52.0608957282, 0x16e22db4568793, -53.3070928798},
	{93, 0x1d1275fe0969ee, -52.4023944266, 0x16e22db4568793, -52.9851647849},
	{94, 0x1741f7fe6dee58, -53.4023944266, 0x1682636a3f20ce, -51.985986448},
	{95, 0x10cde266a0e9eb, -51.3718637928, 0x1e9d727c869dae, -50.0554512305},
	{96, 0x12204395694a47, -53.259539958, 0x1aa64ef02d7c9a, -53.8963496356},
	{97, 0x1d37a84f08a4c5, -52.6405652037, 0x1c237b6f27bf24, -53.0665459605},
	{98, 0x175fb9d8d3b704, -52.6405652037, 0x13e9ac2dc8cd53, -54.8567626135},
	{99, 0x16ae83e9d18847, -52.720351129, 0x13e9ac2dc8cd53, -54.5348345187},
	{100, 0x1900be8227a792, -55.9332933883, 0x15c949a26c50f5, -51.709794284},
	{101, 0x14a54961ab9761, -53.3676358221, 0x1d5c33a2a3b7c3, -54.1591925294},
	{102, 0x17acda2e8e8d4d, -51.8151159718, 0x177cf61bb62c9c, -54.1591925294},
	{103, 0x12d3f280569d6d, -56.1044461591, 0x12a40e6d7e3cbc, -52.3359826565},
	{104, 0x12d3f280569d6d, -55.7825180642, 0x1df99a57dd7b54, -52.4459333877},
	{105, 0x176fb8e5a01057, -52.4142271393, 0x114ab05e93771f, -54.9542704648},
	{106, 0x1aaa93eda04a75, -55.2247982966, 0x1fab333e1a8f24, -51.8983694772},
	{107, 0x177fdb9a9da28b, -53.9006180844, 0x1955c298153f50, -51.8983694772},
	{108, 0x1a086f10395c13, -54.3171566045, 0x14449bacddcc40, -51.8983694772},
	{109, 0x10244fe3005f82, -50.8436784407, 0x1f7e54c363c283, -48.7477778681},
	{110, 0x10244fe3005f82, -50.5217503459, 0x1ef5b0cc782b6f, -48.9068988953},
	{111, 0x1142b93994a1f3, -53.9727384669, 0x1de8c956e0040a, -53.0424581147},
	{112, 0x1e2591fb967d23, -54.6238393308, 0x194c203a96c42e, -52.1119512877},
	{113, 0x107ce929cbdb41, -49.7914274598, 0x1f3279712ea488, -49.0455813274},
	{114, 0x194800ba26678a, -52.7116215392, 0x1566d8ec8d06c6, -52.4172890273},
	{115, 0x194800ba26678a, -53.3896934443, 0x159a2783ce70ab, -52.0897994268},
	{116, 0x11604f904bd23f, -51.647005527, 0x1b64ecb3e7b6c0, -52.4172890273},
	{117, 0x1f7ef51c3323dd, -52.2827543996, 0x1a8c8e5000ca44, -53.6299734928},
	{118, 0x1737ce2b47f8da, -54.003102329, 0x1de14e74b99bae, -51.9197539287},
	{119, 0x197aeaaf58844f, -51.0144650206, 0x16e5133cfc7737, -55.2559964254},
	{120, 0x11d2dbc0ddf643, -52.0794647908, 0x1e86c451509ef4, -54.5190308312},
	{121, 0x1b7e622ae7474e, -51.5395216381, 0x10c7933bdcfb4d, -52.1422989887},
	{122, 0x1cb4efd3de372a, -53.1156585342, 0x165f6efa7bf9bc, -52.4053333945},
	{123, 0x1cb4efd3de372a, -52.7937304393, 0x17a3bbf2c2d2d2, -52.6445513994},
	{124, 0x151bb241543aaf, -52.3060027377, 0x1e64f64a27f1ec, -53.5093625838},
	{125, 0x108d00b580b806, -51.2151629099, 0x1f3bd9070e9b3b, -50.1861937286},
	{126, 0x1ba02b92d34cbe, -54.2434347287, 0x1ffb1c623c89f5, -52.1299441461},
	{127, 0x1ba02b92d34cbe, -53.9215066339, 0x1a9aedf0b510c1, -53.6217366711},
	{128, 0x10c78cdeeb3a0b, -54.5344341713, 0x13a6c22393ad6c, -52.4123569737},
	{129, 0x1657afe96c564f, -52.3882347184, 0x166ed3a8d43b8e, -52.5883760905},
	{130, 0x1d1729c25a62d3, -53.5212580399, 0x166ed3a8d43b8e, -52.2664479956},
	{131, 0x1d1729c25a62d3, -54.1993299451, 0x1841a859fc37c2, -52.1227121526},
	{132, 0x19b2b23e14bb09, -55.2802460487, 0x1841a859fc37c2, -51.8007840577},
	{133, 0x19b2b23e14bb09, -54.9583179538, 0x1e42d1ef424588, -53.085876997},
	{134, 0x13a5550e3aad21, -54.8436984566, 0x1cc5947095c21f, -52.7121515753},
	{135, 0x17da90ec941d2f, -52.6928349036, 0x1ee0325fc27a26, -52.8869464289},
	{136, 0x131540bd434a8c, -52.6928349036, 0x18fab98e82e259, -57.1250744101},
	{137, 0x119ffdb7e8dec2, -54.7612568344, 0x1795e344fe5f34, -51.3055834869},
	{138, 0x119ffdb7e8dec2, -54.4393287395, 0x1f4efbcade5c53, -49.9256634275},
	{139, 0x1c332f8ca7cad0, -53.4393287395, 0x151111b8d15cf1, -52.2187848875},
	{140, 0x1c443de6cb608c, -52.3248466112, 0x151111b8d15cf1, -52.8968567926},
	{141, 0x11479d3b24d574, -49.9977236262, 0x1e9d0ff4df80ea, -49.995133179},
	{142, 0x1e2d0a59349af1, -53.279547837, 0x1047006737544e, -52.2545487553},
	{143, 0x1f874543446dab, -51.7407175248, 0x12a7f6dc7161f9, -51.1288106234},
	{144, 0x19390435d057bc, -51.7407175248, 0x1a6125e6f00297, -57.8848989242},
	{145, 0x125997298b3d87, -54.8686116742, 0x1b8947980fad72, -51.3785592186},
	{146, 0x1f249776988ce3, -49.6329456218, 0x10890a6087fca7, -51.085630394},
	{147, 0x1b27bd46aa8506, -53.0431337817, 0x11408b9c5708e1, -53.8318496121},
	{148, 0x11c3505acd6bf6, -55.2907589383, 0x1899e58e5a959d, -51.2688330527},
	{149, 0x1550c6d35ce7f4, -54.7057964376, 0x1c275c06ea119b, -50.9647768864},
	{150, 0x1ca2605c23b834, -51.672738888, 0x15431bad72f1e3, -54.4271892534},
	{151, 0x1b130d9623a06f, -52.6333050573, 0x11027c8ac25b1c, -54.4271892534},
	{152, 0x15a8d7ab4fb38c, -52.6333050573, 0x166cb2759647ff, -55.8112757375},
	{153, 0x10b730b4074c14, -52.5557248853, 0x1f319f603f95b6, -50.6443128546},
	{154, 0x10b730b4074c14, -52.2337967904, 0x1e5ba9bda59eb5, -50.3033353374},
	{155, 0x1c22dd1dce98a5, -52.7544477399, 0x1d8705e85e512a, -52.9566796931},
	{156, 0x10e1b7deaf2863, -54.1694852392, 0x1c12eda87dc298, -50.6933586706},
	{157, 0x1201d531cbe6d0, -53.7544477399, 0x1f7345a1d3fddf, -50.397595436},
	{158, 0x1201d531cbe6d0, -53.432519645, 0x1be60dc2857af3, -53.6686729902},
	{159, 0x1e7c00cdbdc6ad, -50.5540609472, 0x13c87121732846, -52.2238416437},
	{160, 0x1745b0508c6606, -52.3906177687, 0x138c9f0274431f, -56.4192081831},
	{161, 0x1120ec8799dd42, -53.4818411409, 0x1acfb6730374b6, -51.6128655231},
	{162, 0x1b67e0d8f62ed0, -53.4818411409, 0x1ff3309bb19d7d, -52.4145761954},
	{163, 0x15ecb3e0c4f240, -53.4818411409, 0x1ff3309bb19d7d, -52.0926481005},
	{164, 0x144b59d0c4bbf5, -51.406482747, 0x1e51d68bb16732, -55.1730811664},
	{165, 0x15a5b522f3fbb0, -51.9914452477, 0x178e0df0c5288b, -53.3821327347},
	{166, 0x1214b46e4a0e7e, -53.6843269058, 0x1d076773404298, -51.6548578835},
	{167, 0x1214b46e4a0e7e, -53.3623988109, 0x11092c4f2febc9, -52.9477022624},
	{168, 0x13203c8d643133, -52.3036414925, 0x11742f8ed3f9ab, -55.3630807931},
	{169, 0x1d4389b02cef18, -52.487991678, 0x10ee8048710cba, -50.4799753477},
	{170, 0x1ef4fcea9ea391, -52.9083982509, 0x120f77c4bcda60, -50.0649378484},
	{171, 0x1e0dd0872ecbd9, -53.1440629563, 0x120f77c4bcda60, -50.7430097535},
	{172, 0x1e0dd0872ecbd9, -52.8221348614, 0x14ec0b28e85fcf, -54.8506819784},
	{173, 0x1034bc0b0dfe4d, -54.0107475037, 0x19a35a46c2c151, -52.7497148089},
	{174, 0x1ca39f316d4832, -52.9409298099, 0x15dd8162140ce9, -54.6350082205},
	{175, 0x16e94c278aa028, -52.9409298099, 0x1b97d46bf6b4f3, -55.8159133458},
	{176, 0x1a8ea6882a55a1, -52.1057505076, 0x17f27a0b56ff7a, -53.4463144464},
	{177, 0x11ce9a6d116ee5, -53.6124099973, 0x17f27a0b56ff7a, -53.1243863515},
	{178, 0x11ce9a6d116ee5, -54.2904819024, 0x1811e1547d400c, -52.6844966804},
	{179, 0x1dbe6ba997db43, -54.3064524521, 0x1200d94881d635, -53.2668527504},
	{180, 0x106ee26cfe9bb5, -49.3697606457, 0x1f9fdc96151c7e, -50.572434993},
	{181, 0x1f0690cc1cb67b, -51.128982768, 0x1642e4d63f96a3, -54.0904321993},
	{182, 0x1ec1c1ed7d42b0, -51.1920830199, 0x1642e4d63f96a3, -53.7685041044},
	{183, 0x13b92262252ac1, -52.409194107, 0x1dae8672ff7384, -53.0315385103},
	{184, 0x19b2c4d2a82336, -59.6981272671, 0x1cc868eab152d3, -51.6898641987},
	{185, 0x19b2c4d2a82336, -59.3761991722, 0x1c2a7b4c49161a, -51.6912657148},
	{186, 0x1fae335650508d, -51.8640171972, 0x10e649fa924076, -52.1063032141},
	{187, 0x11e1b3786ab491, -53.1495291926, 0x14fb7d9517590b, -53.4507371263},
	{188, 0x1f175a7134d7b0, -55.5574485321, 0x1cc15431648ef7, -53.1974060401},
	{189, 0x18df7b8dc3dfc0, -55.5574485321, 0x1cc15431648ef7, -52.8754779452},
	{190, 0x1529829e361cf8, -51.0181683854, 0x1cc15431648ef7, -53.5535498504},
	{191, 0x10ee02182b4a60, -51.0181683854, 0x1fedf495ecace9, -56.1705528009},
	{192, 0x14d9c6292bef2f, -54.2847366459, 0x16285cd9817b74, -52.8644016054},
	{193, 0x1ebf42e134cc8e, -51.8004449055, 0x11fcced1457ed1, -53.438505816},
	{194, 0x1272c1ba52e122, -52.2154824048, 0x1ffa9786b7be25, -50.1082676579},
	{195, 0x1272c1ba52e122, -51.8935543099, 0x1e9ff7e1777b05, -52.2956312211},
	{196, 0x161d0cac70caa7, -53.0549795351, 0x11252169cab245, -53.5130257181},
	{197, 0x161d0cac70caa7, -52.7330514403, 0x122383aa5250bf, -54.1199831216},
	{198, 0x1c92f2bcf205c5, -53.2206382062, 0x122383aa5250bf, -53.7980550267},
	{199, 0x10641ee18e97ce, -49.3099698372, 0x1f93c17564c779, -47.9251454499},
	{200, 0x102c2c1f672524, -49.0074070672, 0x1fcbb4378c3a23, -47.610602358},
	{201, 0x1f404b99afac5e, -52.1171550415, 0x1d80b58874170e, -54.7033059864},
	{202, 0x1f404b99afac5e, -52.7952269466, 0x1aba3262ee707b, -57.7068772042},
	{203, 0x18189dd665e451, -53.226330721, 0x1392849fa4a86e, -53.6581554711},
	{204, 0x16e55f3db8e690, -51.7714581054, 0x194bdc6f12e212, -55.2772900998},
	{205, 0x113a347714d259, -54.1300589664, 0x10234feffc1f72, -52.4844830521},
	{206, 0x1b9053f1baea28, -54.1300589664, 0x15f08065522715, -52.0304577099},
	{207, 0x160d0ff4958820, -54.1300589664, 0x1e3566610a3a21, -51.8132220574},
	{208, 0x1591127e070559, -51.5482763508, 0x1db968ea7bb75a, -53.5535917843},
	{209, 0x1591127e070559, -52.2263482559, 0x1e4e32abf3877c, -51.9650834862},
	{210, 0x102023d4ab72ed, -54.0439046409, 0x139efd80a3a6fb, -52.5204793407},
	{211, 0x102023d4ab72ed, -53.721976546, 0x1c98e777727b20, -53.1372235895},
	{212, 0x1eca212efb87f6, -55.1368045315, 0x18f187458e1066, -52.0782460792},
	{213, 0x1bfbc0880ee2b1, -51.3601034662, 0x114df520b439aa, -54.4742806359},
	{214, 0x1370b768932845, -54.2543035221, 0x1e5665b1aa961e, -51.6005795864},
	{215, 0x19cdd0dfd6ffec, -51.5059128237, 0x159ea710cb0b0a, -56.0571330544},
	{216, 0x174b1e30696cfe, -52.6013000221, 0x13f22ff12ca916, -52.3051770561},
	{217, 0x1a9cf820cc9ad4, -53.9743289177, 0x1a996df95fcfcb, -52.0769550007},
	{218, 0x1a9c42e5b6d89f, -58.8386348136, 0x1a996df95fcfcb, -52.7550269059},
	{219, 0x1ff33e8dcd1212, -51.7182715904, 0x1545473da09f2c, -51.7444294886},
	{220, 0x1ee2fafdeb7083, -51.3271890859, 0x110438fe1a18f0, -51.7444294886},
	{221, 0x1f8412ca6063fc, -53.0039408617, 0x10632131a52577, -54.0092286422},
	{222, 0x1936756eb38330, -53.0039408617, 0x16b0be8d520643, -54.4278021264},
	{223, 0x17b307b445d1d5, -55.1853198249, 0x15ae75665e3ab1, -52.7994889546},
	{224, 0x12f59fc36b0e44, -56.1853198249, 0x13a9e31876a38d, -52.3457511192},
	{225, 0x1fdcfc783f4be1, -48.800334976, 0x100dff2ca1b21a, -47.4154507827},
	{226, 0x1e38f4d92450dc, -53.5233353797, 0x1783086f4f8963, -51.602104424},
	{227, 0x101b043131debc, -50.6791122057, 0x1fd44e75348375, -49.1959078064},
	{228, 0x101b043131debc, -50.3571841108, 0x1fd44e75348375, -48.8739797116},
	{229, 0x1051fd6be09d70, -50.016147193, 0x1f9d553a85c4c1, -48.5451874047},
	{230, 0x107df801393600, -50.6791122057, 0x1f45600fd493a1, -49.2123440528},
	{231, 0x1212d1bfc6b76d, -52.9666253379, 0x1ea650d47a5cda, -57.8354627332},
	{232, 0x1e55587b2adc91, -52.6645837363, 0x18850d76c84a48, -57.8354627332},
	{233, 0x1a9797fde3515e, -53.0663331623, 0x1dee03ea3c5998, -52.5866314189},
	{234, 0x1a9797fde3515e, -52.7444050675, 0x19449a5ca3661f, -56.8576375922},
	{235, 0x1e9090e1a3131b, -50.361272291, 0x1436e1e3b5eb4c, -56.8576375922},
	{236, 0x195d7ffb1113c9, -51.3874761461, 0x1436e1e3b5eb4c, -56.5357094973},
	{237, 0x1646545373fb7e, -53.4498272448, 0x192bb4be35b875, -51.9672680184},
	{238, 0x1646545373fb7e, -53.1278991499, 0x19a91cece6ad07, -54.3870975551},
	{239, 0x1c15823b1f74ce, -53.4659453086, 0x19a91cece6ad07, -54.0651694602},
	{240, 0x1c488e84a1701c, -51.885721044, 0x19a91cece6ad07, -54.7432413653},
	{241, 0x16a07203b459b0, -51.885721044, 0x1c7d2b2d5d383d, -56.2825959475},
	{242, 0x14723ee06d6d92, -52.9558828754, 0x1c7d2b2d5d383d, -55.9606678526},
	{243, 0x18cea526fb45ce, -51.5467729696, 0x11d467e94b856e, -53.7087098303},
	{244, 0x1e8b28de5cbb1a, -51.8114103249, 0x11d467e94b856e, -53.3867817355},
	{245, 0x1f91170f353174, -52.076370624, 0x11d467e94b856e, -53.0648536406},
	{246, 0x15affdaa4ea67a, -52.9051614081, 0x18d0d0fd6fb79f, -52.2100405723},
	{247, 0x115997bb721ec8, -52.9051614081, 0x1d2736ec4c3f51, -52.0783383577},
	{248, 0x1dfa896359ca26, -50.9465460491, 0x108645446493f3, -50.6259556999},
	{249, 0x19df16bdb2697c, -52.2300483648, 0x1660d785ee1b92, -53.0017398181},
	{250, 0x110fe35a753277, -54.0647416258, 0x1660d785ee1b92, -52.6798117232},
	{251, 0x1b4c9ef721ea58, -53.0647416258, 0x19ca6b64d258dd, -52.4728272758},
	{252, 0x1d1c100d1aff52, -51.7396810526, 0x1ab223efcee35a, -62.1734556591},
	{253, 0x1e16ee5d60cf47, -53.157502641, 0x1f11ccada69f3c, -52.7355149669},
	{254, 0x1e16ee5d60cf47, -52.8355745461, 0x18db0a24854c30, -52.7355149669},
	{255, 0x1e16ee5d60cf47, -53.5136464512, 0x178bc71acbbe1e, -51.5981781198},
	{256, 0x1af33a37ac0b43, -52.6038909399, 0x12d638e23c9818, -51.5981781198},
	{257, 0x1c34b579f459ab, -52.4892946292, 0x1417b42484e680, -51.1831406205},
	{258, 0x128bd38d75b33c, -52.7949014739, 0x1ee97e824ef362, -50.6837673128},
	{259, 0x10b10b328387b6, -52.6249764724, 0x1fd83c7368631f, -47.998457388},
	{260, 0x128bd38d75b33c, -52.1510452841, 0x1dbcb55dcd0505, -50.4791874529},
	{261, 0x17c0747bd76fa1, -54.4606249563, 0x11d463488c26e2, -52.8963022724},
	{262, 0x17c0747bd76fa1, -54.1386968614, 0x1306d6a8f077f6, -51.7792268509},
	{263, 0x17c0747bd76fa1, -53.8167687665, 0x18b8da5249bfd2, -51.9271182906},
	{264, 0x17c0747bd76fa1, -53.4948406716, 0x1d4743a6912c82, -52.1362398755},
	{265, 0x1f0ab69c49b721, -52.6742810861, 0x1076325b652821, -54.4491436231},
	{266, 0x1f6b0ca196a28e, -52.9021607703, 0x1076325b652821, -54.1272155282},
	{267, 0x19227081454ed8, -52.9021607703, 0x1d076a9c07cf8d, -57.749976002},
	{268, 0x1ecbfe7cbe1a27, -50.737528741, 0x12b17d4d3f3698, -51.2679486537},
	{269, 0x1d122c7f26f2e5, -52.683260048, 0x1788e42d0b919e, -56.4354109373},
	{270, 0x1d122c7f26f2e5, -52.3613319531, 0x1599216d203a02, -51.8986353359},
	{271, 0x1bbd90a3bccad9, -51.8843870426, 0x110f4dff9476c4, -51.2855931447},
	{272, 0x1eb8e782c7aa76, -56.1095463965, 0x17e41ac12e109d, -52.0049999186},
	{273, 0x1e0af69d1a9d0c, -51.8463240272, 0x173629db810333, -54.2814732299},
	{274, 0x106ac0e2be7d0f, -49.328396617, 0x1faa8b35c21fb3, -50.779993755},
	{275, 0x1f06308743bc2e, -53.2127370943, 0x1cd5d28b2a562c, -52.6655563527},
	{276, 0x16279b2ab47e87, -52.6371276788, 0x11bd2ab91e7b4e, -54.2697873701},
	{277, 0x11a9c1c6580ce4, -50.9119393618, 0x1f0fe500a6f363, -52.3388510725},
	{278, 0x1bd27348cf6589, -52.3422330507, 0x1bacff2a7f2b2c, -53.4520780343},
	{279, 0x190df35e29146b, -51.9549609825, 0x1623ff553288f0, -53.4520780343},
	{280, 0x190df35e29146b, -52.6330328876, 0x16b93023ca3e6f, -56.2166949361},
	{281, 0x1b80cd061d237c, -50.2179140757, 0x122dc01ca1cb8c, -56.2166949361},
	{282, 0x1c45621da55c5d, -51.2725172451, 0x122dc01ca1cb8c, -55.8947668412},
	{283, 0x1de368232b681c, -52.3021079285, 0x1569cc27ade30a, -53.0304134759},
	{284, 0x16a97621e84e5d, -55.3284044085, 0x1569cc27ade30a, -52.708485381},
	{285, 0x12212b4e5371e4, -55.3284044085, 0x19f216fb42bf83, -52.4342764665},
	{286, 0x17d173e4ab5cb9, -54.3803694788, 0x19f216fb42bf83, -53.1123483716},
	{287, 0x17d173e4ab5cb9, -54.0584413839, 0x1985299057abc1, -53.2704673857},
	{288, 0x1d433a23df5bae, -56.9296095335, 0x15c718fccffbd4, -51.9035678782},
	{289, 0x18f98963bf2ce3, -53.2257253143, 0x1eb11fb93f6b47, -55.1179912018},
	{290, 0x1baecf224bb682, -52.1316975095, 0x10d9b828199006, -52.7361048495},
	{291, 0x1ce2ef5ead3204, -52.7070349956, 0x10d9b828199006, -52.4141767546},
	{292, 0x19ff6ecdc33ccc, -51.7668316183, 0x10d9b828199006, -53.0922486597},
	{293, 0x1b89d58173370c, -52.0878887933, 0x17bc2d88765dc9, -53.5102000737},
	{294, 0x1f86569e251e5a, -52.3857757055, 0x13bfac6bc4767b, -57.1547120226},
	{295, 0x1d2b67c744fcc7, -52.3990655671, 0x13bfac6bc4767b, -56.8327839277},
	{296, 0x1edf4f4f0468dc, -52.3660676763, 0x11401311090834, -51.2867311966},
	{297, 0x1ddf785deca326, -54.6341824385, 0x178fc5b77ad5ad, -52.0828991237},
	{298, 0x1ddf785deca326, -54.3122543436, 0x1e55f024bd2e13, -53.5944984963},
	{299, 0x1aa231ee38d6ab, -50.704010426, 0x123390160b1ba5, -55.0095359956},
	{300, 0x1a6d90d8cb076d, -51.8714396305, 0x1844c01d6424dc, -54.2725704014},
	{301, 0x18b35042df1ef9, -55.9139826891, 0x17d62ff7e92abf, -52.7765977082},
	{302, 0x14738d3e440f5c, -53.4811198716, 0x161bef61fd424b, -51.7643457641},
	{303, 0x1f6560470befd4, -50.8287467224, 0x10a305dd99520f, -50.7527072272},
	{304, 0x1fdf0ffcebbef9, -50.5509370758, 0x111cb593792134, -50.3901371479},
	{305, 0x15062ed5a648c3, -52.3006177892, 0x1828190bc01e42, -52.7632070489},
	{306, 0x1f50347ec5192a, -53.4343412438, 0x1828190bc01e42, -52.441278954},
	{307, 0x13f35deacbefed, -51.9143948, 0x1c735cb729819a, -55.4622945339},
	{308, 0x13f35deacbefed, -52.5924667051, 0x12c5376392047f, -54.4900019642},
	{309, 0x16ff1ead93305e, -54.7027690081, 0x1d16a03321b140, -51.7403043798},
	{310, 0x1813ee3dd23507, -52.8653763127, 0x15ea4f1d542bb5, -54.6002640483},
	{311, 0x1c0744a396a24e, -54.1002085581, 0x1dd0fbe8dd0643, -53.6315215267},
	{312, 0x1a3d8d5e503e59, -52.0196150669, 0x14090caba5e21b, -51.8942964522},
	{313, 0x1f954c7b05032d, -52.1935573917, 0x10073d561e4e7c, -51.8942964522},
	{314, 0x1d59f6cc17fc16, -51.9792195612, 0x1323a6c3e60c39, -53.2219474231},
	{315, 0x177b2bd6799678, -51.9792195612, 0x1ee13caf22d775, -57.1265561765},
	{316, 0x14e04264eddd1f, -55.2468143966, 0x1323a6c3e60c39, -52.5780912334},
	{317, 0x16f5c9f2f73e33, -52.6276370489, 0x142e6a8aeabcc3, -54.9662554023},
	{318, 0x1d5d881f4da271, -54.6111376706, 0x15fe99ed0fae2a, -52.0482165413},
	{319, 0x1d5d881f4da271, -54.2892095757, 0x1a6ac2719b40bb, -53.4706766374},
	{320, 0x165c314aa13a56, -51.5958617293, 0x15223527af66fc, -54.4706766374},
	{321, 0x1ec90778575a21, -49.7424332253, 0x11ea99b7776b45, -51.0780875154},
	{322, 0x1d6f29c8b17f2d, -54.9669460679, 0x13e83904bd93a2, -50.6041563271},
	{323, 0x178c216d5acc24, -54.9669460679, 0x13bc8debc60786, -51.6760380229},
	{324, 0x1b5bb4eeef90c2, -52.0858589744, 0x14a29ab3436bbc, -51.2963209057},
	{325, 0x1c13bf2820adba, -53.593012505, 0x185b2cedb20cbb, -52.2307889866},
	{326, 0x131430d4bc626f, -51.8155998567, 0x1eb73d349b82e0, -52.5428166581},
	{327, 0x19016c3c77870a, -53.7280032369, 0x1823c27e1ae3f6, -52.0169806394},
	{328, 0x192dc1628a0e0e, -52.5148227097, 0x187c6cca3ff1fe, -52.7605055368},
	{329, 0x1deff96c8805a4, -53.3049533001, 0x187c6cca3ff1fe, -52.4385774419},
	{330, 0x1d221b25cb549f, -51.6800873291, 0x12c440bb0e4e01, -52.2455044887},
	{331, 0x10a8606321f262, -53.543913429, 0x14d706039a8287, -52.9119750416},
	{332, 0x19d0df7e849a2f, -52.1447907886, 0x1dff851efd2a54, -54.407339614},
	{333, 0x1f78362109b391, -51.8936763666, 0x13abf885fb530e, -53.5397624174},
	{334, 0x1e4adbb8ffac2d, -54.7244594818, 0x13abf885fb530e, -53.2178343225},
	{335, 0x1e4adbb8ffac2d, -54.4025313869, 0x15cbbf902f64e1, -53.3507023025},
	{336, 0x1f1ab91751cfd1, -50.9742291601, 0x114a4d479bd03a, -50.19356106},
	{337, 0x1eb85c59feb46f, -50.5982498305, 0x10e7f08a48b4d8, -49.9040544428},
	{338, 0x1eb948e9fd79ac, -52.5166172042, 0x126f8a5f9797e6, -52.684780016},
	{339, 0x14471ffc5a71cb, -54.1632869371, 0x11d25880abf9ef, -52.7543025383},
	{340, 0x1038e6637b8e3c, -54.1632869371, 0x11d25880abf9ef, -52.4323744434},
	{341, 0x1d851bec1b48c3, -53.9365863692, 0x17abba3663a159, -53.5884270927},
	{342, 0x1c4b97b4a8c024, -53.2260012145, 0x1a1ec2a548b297, -52.4416484999},
	{343, 0x16a2dfc3ba3350, -53.2260012145, 0x1a1ec2a548b297, -52.119720405},
	{344, 0x113c3633e0206f, -49.8975276307, 0x1f2bc1ba1a1d8b, -50.7102324258},
	{345, 0x126c1f0d3bf1c9, -53.6620560869, 0x1e93cd4d6c34de, -53.5393781854},
	{346, 0x126c1f0d3bf1c9, -53.340127992, 0x1c25774095c10d, -53.9157365931},
	{347, 0x126c1f0d3bf1c9, -53.0181998971, 0x1e4bd9298c7374, -52.1123830613},
	{348, 0x16c2fc27f913c1, -53.0296419907, 0x159e1ef4122f84, -52.0882546001},
	{349, 0x1d3f38a464eecf, -53.9459596578, 0x114b4bf674f2d0, -52.0882546001},
	{350, 0x107adbba885dec, -52.4542843818, 0x1da770c25b3941, -52.6701966522},
	{351, 0x1c576f3b79a806, -52.7414698862, 0x17b92701e29434, -52.6701966522},
	{352, 0x16ac58fc615338, -53.7414698862, 0x131adec84b8062, -51.7866747415},
	{353, 0x12237a63810f60, -53.7414698862, 0x17280b2ec68bb9, -52.9991355103},
	{354, 0x1e067a6175fe45, -53.3281142368, 0x118585965dbd4f, -53.734762025},
	{355, 0x1f8192b43129a1, -51.6661587365, 0x1088cab48ba067, -52.2141126989},
	{356, 0x1fd1933ec594aa, -52.2699877127, 0x1088cab48ba067, -51.892184604},
	{357, 0x1a4ea5029709dd, -51.4288996241, 0x135758c7ba153a, -55.0100260528},
	{358, 0x1ad91ea463db0f, -52.474827228, 0x135758c7ba153a, -55.6880979579},
	{359, 0x157a7ee9e97c0c, -52.474827228, 0x1692d260050d6b, -52.4468860936},
	{360, 0x195610830d2782, -51.1472674469, 0x119eed50c5d096, -59.1510660802},
	{361, 0x1f7f9a79c4f3fb, -54.1900509138, 0x167580eea80f26, -51.5483211653},
	{362, 0x1fcf1f85ae7612, -52.3499845814, 0x1f30156ddb71e4, -54.0786657922},
	{363, 0x1fcf1f85ae7612, -52.0280564865, 0x15c4f4641e1c1b, -53.5785971007},
	{364, 0x12401e69c061dd, -55.3192322325, 0x1ba2590564fd2d, -52.4639658716},
	{365, 0x1d18ec19b20ba1, -52.6483498796, 0x1635f22d6c284b, -51.7826127637},
	{366, 0x1a22cbcf5f5b97, -52.0126310076, 0x11c4c1bdf0203c, -51.7826127637},
	{367, 0x1e62a3fa21afcc, -55.2095683653, 0x1f49ab511dec43, -52.086461059},
	{368, 0x1e063aa4233136, -52.5992670439, 0x1ebf0d50202e62, -53.3590441214},
	{369, 0x1af31fb5b992f9, -52.1945563148, 0x1898d7734cf1e8, -53.3590441214},
	{370, 0x111c6321346569, -52.6357388387, 0x13ad79290a5b20, -54.3590441214},
	{371, 0x1df1b4a35cfe5f, -54.0645672311, 0x10416555997a0b, -53.3055990455},
	{372, 0x11740193a69aae, -53.1800553178, 0x17c8c9e15fa6bf, -51.6989391729},
	{373, 0x1beccf52a42ab0, -53.1800553178, 0x144b2ff70b2169, -52.2771876246},
	{374, 0x1c71f224fdc32c, -51.2278041724, 0x148dc16037eda7, -53.1411497602},
	{375, 0x13997b021579ee, -52.6729230217, 0x178093c1300611, -55.9609579238},
	{376, 0x1e9cf2c623bf8f, -51.2052426024, 0x12bba9fbb2a0a8, -50.9473426834},
	{377, 0x125e2b43af0c89, -51.6202801017, 0x1edb4740d0ccf9, -53.2258911089},
	{378, 0x127719db27784d, -53.606676808, 0x18af6c33da3d94, -53.2258911089},
	{379, 0x1c56debcb1f755, -55.3461100361, 0x16506992f00996, -51.9272187114},
	{380, 0x1a495ee335ee1d, -52.0575449464, 0x1442e9b974005e, -53.0688396316},
	{381, 0x15077f1c2b24e4, -52.0575449464, 0x1984c9807ec997, -53.4914563469},
	{382, 0x10d2cc1688ea50, -52.0575449464, 0x1db97c8621042b, -54.290940845},
	{383, 0x136755c67422af, -53.1363590782, 0x1db97c8621042b, -54.9690127501},
	{384, 0x1f0bbc70b9d118, -52.1363590782, 0x1db97c8621042b, -54.6470846552},
	{385, 0x18d6305a2e40e0, -52.1363590782, 0x1db97c8621042b, -54.3251565604},
	{386, 0x13de8d14f1cd80, -53.1363590782, 0x18ad82279ae0b6, -51.9357697079},
	{387, 0x1e48ec2148695f, -52.5877172151, 0x16c12020240c3a, -52.8450483601},
	{388, 0x1e1f3004917494, -49.3887248157, 0x123419b35009c8, -52.8450483601},
	{389, 0x1eb513f3390575, -51.7722860981, 0x123419b35009c8, -53.5231202652},
	{390, 0x143ea823dd6aa7, -52.2605979339, 0x12e2492e29d4bd, -58.4177752866},
	{391, 0x1d28bdf8d4ef43, -50.9326179701, 0x16a9249dcbcc16, -57.8328127859},
	{392, 0x160712e80eeb93, -52.3516872266, 0x1f0ef31a0b90cc, -51.8968303784},
	{393, 0x10275221a48a9f, -52.4772181087, 0x1fb7b03322cc2d, -48.7543665454},
	{394, 0x119f42533f22dc, -52.0297591317, 0x1f9a135f69fcef, -52.0481790281},
	{395, 0x1863894c98c0ae, -54.5875253811, 0x1a2c95b2773a6a, -51.819181042},
	{396, 0x15c57a5459a13d, -53.5159981779, 0x14f077c1f8fb88, -51.819181042},
	{397, 0x116ac8437ae764, -53.5159981779, 0x1da5dbe3b66f3a, -51.6872399072},
	{398, 0x1402a2951a972b, -52.2771517301, 0x1fb8f0f1cfc8a6, -53.7811399256},
	{399, 0x1cb27c0b352f98, -52.5569410228, 0x1709177bb53039, -58.3916335911},
	{400, 0x1245dd7d7babe2, -51.5381053909, 0x1709177bb53039, -58.0697054962},
}

var pow10Tests = []struct {
	p  int
	pm pmHiLo
	pe int
}{
	{0, pmHiLo{1 << 63, 0}, 1},
	{25, pmHiLo{0x84595161401484a0, 0}, -44 + 128},
	{72, pmHiLo{0x90e40fbeea1d3a4a + 1, (1 << 64) - 0xbc8955e946fe31ce}, 112 + 128},
	{-44, pmHiLo{0xe45c10c42a2b3b05 + 1, (1 << 64) - 0x8cb89a7db77c506b}, -274 + 128},
}

func TestPow10(t *testing.T) {
	for _, tt := range pow10Tests {
		c := prescale(0, tt.p, log2Pow10(tt.p))
		pm := pmHiLo{c.pmHi, c.pmLo}
		pe := -(c.s + 2)
		if pm != tt.pm || pe != tt.pe {
			t.Errorf("pow10(%d) = %#x, %d, want %#x, %d", tt.p, pm, pe, tt.pm, tt.pe)
		}
	}
}
