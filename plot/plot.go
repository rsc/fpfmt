// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"image"
	"log"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	png   = flag.Bool("png", false, "generate PNG files")
	graph = flag.String("graph", "", "regexp pattern to filter graph names")
)

var graphs = []struct {
	data  string
	svg   string
	op    string
	title string
}{
	{"linux-ryzen.out", "fpfmt-ryzen-fixed6", "fixed6", "Fmt(FixedWidth(f, 6)) [Linux, AMD Ryzen 9]"},
	{"linux-ryzen.out", "fpfmt-ryzen-fixed17", "fixed17", "Fmt(FixedWidth(f, 17)) [Linux, AMD Ryzen 9]"},
	{"linux-ryzen.out", "fpfmt-ryzen-short", "short", "Fmt(Short(f)) [Linux, AMD Ryzen 9]"},
	{"linux-ryzen.out", "fpfmt-ryzen-shortraw", "shortraw", "Short(f) [Linux, AMD Ryzen 9]"},
	{"linux-ryzen.out", "fpfmt-ryzen-parse", "parse", "ParseText(s) [Linux, AMD Ryzen 9]"},
	{"linux-ryzen.out", "fpfmt-ryzen-parseraw", "parseraw", "Parse(d, p) [Linux, AMD Ryzen 9]"},

	{"darwin-m4.out", "fpfmt-apple-fixed6", "fixed6", "Fmt(FixedWidth(f, 6)) [macOS, Apple M4]"},
	{"darwin-m4.out", "fpfmt-apple-fixed17", "fixed17", "Fmt(FixedWidth(f, 17)) [macOS, Apple M4]"},
	{"darwin-m4.out", "fpfmt-apple-short", "short", "Fmt(Short(f)) [macOS, Apple M4]"},
	{"darwin-m4.out", "fpfmt-apple-shortraw", "shortraw", "Short(f) [macOS, Apple M4]"},
	{"darwin-m4.out", "fpfmt-apple-parse", "parse", "ParseText(s) [macOS, Apple M4]"},
	{"darwin-m4.out", "fpfmt-apple-parseraw", "parseraw", "Parse(d, p) [macOS, Apple M4]"},
}

func main() {
	flag.Parse()

	var graphRE *regexp.Regexp
	if *graph != "" {
		var err error
		graphRE, err = regexp.Compile(*graph)
		if err != nil {
			log.Fatalf("invalid -graph regexp: %v", err)
		}
	}

	var data []Mark
	lastData := ""

	algs := []string{"uscale", "uscalec", "fast_float", "abseil", "ryu", "dragonbox", "go125", "dmg2017", "dmg1997", "dblconv", "libc", "schubfach"}
	for _, gg := range graphs {
		if graphRE != nil && !graphRE.MatchString(gg.svg) {
			continue
		}
		if gg.data != lastData {
			data = load(gg.data)
			lastData = gg.data
		}
		writeGraphs(gg.svg+"-cdf", cdf(data, algs, gg.op, gg.title))
	}

	algs = []string{"libc", "dblconv", "dmg1997", "dmg2017", "ryu", "dragonbox", "abseil", "fast_float", "uscale", "uscalec", "schubfach"}
	for _, gg := range graphs {
		if graphRE != nil && !graphRE.MatchString(gg.svg) {
			continue
		}
		if gg.data != lastData {
			data = load(gg.data)
			lastData = gg.data
		}
		writeGraphs(gg.svg+"-scat", scatter(data, algs, gg.op, gg.title))
	}
}

func writeGraphs(name string, g *Graph) {
	if g == nil {
		return
	}
	if err := os.WriteFile(name+".svg", []byte(g.SVG()), 0666); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(name+"-big.svg", []byte(g.BigSVG()), 0666); err != nil {
		log.Fatal(err)
	}
	eps := []byte(g.EPS())
	if err := os.WriteFile(name+".eps", eps, 0666); err != nil {
		log.Fatal(err)
	}
	if *png {
		pngs, err := EPSToPNGs(name, eps, g.BBox.Dx(), g.BBox.Dy())
		if err != nil {
			log.Fatal(err)
		}
		for file, data := range pngs {
			if err := os.WriteFile(file, data, 0666); err != nil {
				log.Fatal(err)
			}
		}
	}
}

func cdf(data []Mark, algs []string, op, title string) *Graph {
	const (
		Width  = 400
		Height = 300
	)
	g := &Graph{
		Style: Style,
		BBox:  image.Rect(0, 0, Width, Height),
		DBox:  image.Rect(65, 40, Width-20, Height-50),
		Title: title,
	}
	g.X.Min = math.Inf(+1)
	g.X.Max = math.Inf(-1)
	g.X.Title = "Time"
	g.Y.Title = "Fraction Completed"
	g.LegendX = Width - 120
	g.LegendY = 60

	byAlg := make(map[string]*Line)
	for _, alg := range algs {
		l := &Line{Name: alg}
		byAlg[alg] = l
		g.Lines = append(g.Lines, l)
	}

	for _, m := range data {
		l := byAlg[m.Alg]
		if m.Op != op || l == nil {
			continue
		}
		x := m.T
		l.Points = append(l.Points, Point{x, 0})
		g.X.Min = min(g.X.Min, x)
		g.X.Max = max(g.X.Max, x)
	}

	total := make(map[string]float64)
	for _, l := range g.Lines {
		if len(l.Points) < 2 {
			continue
		}
		sort.Slice(l.Points, func(i, j int) bool { return l.Points[i].X < l.Points[j].X })
		if len(l.Points) > 1000 {
			n := len(l.Points)
			l.Points = l.Points[n/1000 : len(l.Points)-n/1000]
		}
		const N = 1000
		if len(l.Points) > 2*N {
			trim := make([]Point, N+1)
			for i := range N + 1 {
				trim[i] = l.Points[(len(l.Points)-1)*i/N]
			}
			l.Points = trim
		}
		for i := range l.Points {
			l.Points[i].Y = float64(i) / float64(len(l.Points)-1)
			total[l.Name] += l.Points[i].X
		}
	}

	key := func(l *Line) float64 {
		if len(l.Points) == 0 {
			return 0
		}
		return l.Points[len(l.Points)*3/4].X
	}
	sort.Slice(g.Lines, func(i, j int) bool {
		return total[g.Lines[i].Name] > total[g.Lines[j].Name]
		return key(g.Lines[i]) < key(g.Lines[j])
	})

	g.Y.Min = 0
	g.Y.Max = 1
	g.Y.AutoScale()
	g.X.AutoTimeLogScale(g)
	return g
}

func scatter(data []Mark, algs []string, op, title string) *Graph {
	const (
		Width  = 600
		Height = 300
	)
	g := &Graph{
		Style:   Style,
		LegendX: Width - 95,
		LegendY: 60,
		BBox:    image.Rect(0, 0, Width, Height),
		DBox:    image.Rect(80, 40, Width-100, Height-50),
		Title:   title,
	}
	g.X.Title = "log₂ f"
	g.Y.Title = "Time"
	g.X.Min = -1024
	g.X.Max = +1024
	g.X.Ticks = []Tick{
		{At: -1000, Label: "−1000"},
		{At: -900},
		{At: -800},
		{At: -700},
		{At: -600},
		{At: -500, Label: "−500"},
		{At: -400},
		{At: -300},
		{At: -200},
		{At: -100},
		{At: 0, Label: "0"},
		{At: +100},
		{At: +200},
		{At: +300},
		{At: +400},
		{At: +500, Label: "+500"},
		{At: +600},
		{At: +700},
		{At: +800},
		{At: +900},
		{At: +1000, Label: "+1000"},
	}
	g.Y.Min = math.Inf(+1)
	g.Y.Max = math.Inf(-1)

	total := make(map[string]float64)
	byAlg := make(map[string]*Scatter)
	for _, alg := range algs {
		s := &Scatter{Name: alg}
		byAlg[alg] = s
		g.Scatters = append(g.Scatters, s)
	}

	for _, m := range data {
		s := byAlg[m.Alg]
		if m.Dist != "rand" || m.Op != op || s == nil {
			continue
		}
		y := float64(m.T)
		s.Points = append(s.Points, Point{math.Log2(m.F), y})
		g.Y.Min = min(g.Y.Min, y)
		g.Y.Max = max(g.Y.Max, y)
		total[s.Name] += y
	}
	if g.Y.Max < g.Y.Min {
		return nil
	}

	sort.Slice(g.Scatters, func(i, j int) bool {
		return total[g.Scatters[i].Name] > total[g.Scatters[j].Name]
	})

	g.Y.AutoTimeLogScale(g)
	return g
}

type Mark struct {
	Dist string
	Op   string
	Alg  string
	F    float64
	T    float64 // nanoseconds
}

func load(file string) []Mark {
	data, err := os.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}
	var out []Mark
	for line := range strings.Lines(string(data)) {
		fields := strings.Fields(line)
		if len(fields) != 5 || line[0] == '#' {
			continue
		}
		u, err := strconv.ParseUint(fields[3], 0, 64)
		if err != nil {
			log.Fatal(err)
		}
		f := math.Float64frombits(u)
		ns, err := strconv.ParseUint(fields[4], 0, 64)
		if err != nil {
			log.Fatal(err)
		}
		out = append(out, Mark{fields[0], fields[1], fields[2], f, float64(ns) / 1000})
	}
	return out
}
