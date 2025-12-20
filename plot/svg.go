// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore

package main

import (
	"image"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"rsc.io/fpfmt/svg"
)

var graphs = []struct {
	data  string
	svg   string
	op    string
	title string
}{
	{"linux-ryzen.out", "fpfmt-ryzen-fixed2", "fixed2", "Fmt(FixedWidth(f, 2)) [Linux, AMD Ryzen 9]"},
	{"linux-ryzen.out", "fpfmt-ryzen-fixed4", "fixed4", "Fmt(FixedWidth(f, 4)) [Linux, AMD Ryzen 9]"},
	{"linux-ryzen.out", "fpfmt-ryzen-fixed8", "fixed8", "Fmt(FixedWidth(f, 8)) [Linux, AMD Ryzen 9]"},
	{"linux-ryzen.out", "fpfmt-ryzen-fixed16", "fixed16", "Fmt(FixedWidth(f, 16)) [Linux, AMD Ryzen 9]"},
	{"linux-ryzen.out", "fpfmt-ryzen-fixed17", "fixed17", "Fmt(FixedWidth(f, 17)) [Linux, AMD Ryzen 9]"},
	{"linux-ryzen.out", "fpfmt-ryzen-fixed18", "fixed18", "Fmt(FixedWidth(f, 18)) [Linux, AMD Ryzen 9]"},
	{"linux-ryzen.out", "fpfmt-ryzen-short", "short", "Fmt(Short(f)) [Linux, AMD Ryzen 9]"},
	{"linux-ryzen.out", "fpfmt-ryzen-shortraw", "shortraw", "Short(f) [Linux, AMD Ryzen 9]"},
	{"linux-ryzen.out", "fpfmt-ryzen-parse", "parse", "Parse(Unfmt(s)) [Linux, AMD Ryzen 9]"},
	{"linux-ryzen.out", "fpfmt-ryzen-parseraw", "parseraw", "Parse(d, p) [Linux, AMD Ryzen 9]"},

	{"darwin-m4.out", "fpfmt-apple-fixed2", "fixed2", "Fmt(FixedWidth(f, 2)) [macOS, Apple M4]"},
	{"darwin-m4.out", "fpfmt-apple-fixed4", "fixed4", "Fmt(FixedWidth(f, 4)) [macOS, Apple M4]"},
	{"darwin-m4.out", "fpfmt-apple-fixed8", "fixed8", "Fmt(FixedWidth(f, 8)) [macOS, Apple M4]"},
	{"darwin-m4.out", "fpfmt-apple-fixed16", "fixed16", "Fmt(FixedWidth(f, 16)) [macOS, Apple M4]"},
	{"darwin-m4.out", "fpfmt-apple-fixed17", "fixed17", "Fmt(FixedWidth(f, 17)) [macOS, Apple M4]"},
	{"darwin-m4.out", "fpfmt-apple-fixed18", "fixed18", "Fmt(FixedWidth(f, 18)) [macOS, Apple M4]"},
	{"darwin-m4.out", "fpfmt-apple-short", "short", "Fmt(Short(f)) [macOS, Apple M4]"},
	{"darwin-m4.out", "fpfmt-apple-shortraw", "shortraw", "Short(f) [macOS, Apple M4]"},
	{"darwin-m4.out", "fpfmt-apple-parse", "parse", "Parse(Unfmt(s)) [macOS, Apple M4]"},
	{"darwin-m4.out", "fpfmt-apple-parseraw", "parseraw", "Parse(d, p) [macOS, Apple M4]"},
}

func main() {
	var data []Mark
	lastData := ""

	algs := []string{"uscale", "uscalec", "fast_float", "abseil", "ryu", "dragonbox", "go125", "dmg2017", "dmg1997", "dblconv", "libc"}
	for _, gg := range graphs {
		if gg.data != lastData {
			data = load(gg.data)
			lastData = gg.data
		}
		svg := cdf(data, algs, gg.op, gg.title)
		if err := os.WriteFile(gg.svg+"-cdf.svg", []byte(svg), 0666); err != nil {
			log.Fatal(err)
		}
	}

	algs = []string{"libc", "dblconv", "dmg1997", "dmg2017", "ryu", "dragonbox", "abseil", "fast_float", "uscale", "uscalec"}
	for _, gg := range graphs {
		if gg.data != lastData {
			data = load(gg.data)
			lastData = gg.data
		}
		svg := scatter(data, algs, gg.op, gg.title)
		if err := os.WriteFile(gg.svg+"-scat.svg", []byte(svg), 0666); err != nil {
			log.Fatal(err)
		}
	}

}

func cdf(data []Mark, algs []string, op, title string) string {
	const (
		Width  = 400
		Height = 300
	)
	g := &svg.Graph{
		Style: svg.Style,
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

	byAlg := make(map[string]*svg.Line)
	for _, alg := range algs {
		l := &svg.Line{Name: alg}
		byAlg[alg] = l
		g.Lines = append(g.Lines, l)
	}

	for _, m := range data {
		l := byAlg[m.Alg]
		if m.Op != op || l == nil {
			continue
		}
		x := m.T
		l.Points = append(l.Points, svg.Point{x, 0})
		g.X.Min = min(g.X.Min, x)
		g.X.Max = max(g.X.Max, x)
	}

	for _, l := range g.Lines {
		if len(l.Points) < 2 {
			continue
		}
		sort.Slice(l.Points, func(i, j int) bool { return l.Points[i].X < l.Points[j].X })
		const N = 1000
		if len(l.Points) > 2*N {
			trim := make([]svg.Point, N+1)
			for i := range N + 1 {
				trim[i] = l.Points[(len(l.Points)-1)*i/N]
			}
			l.Points = trim
		}
		for i := range l.Points {
			l.Points[i].Y = float64(i) / float64(len(l.Points)-1)
		}
	}
	g.Y.Min = 0
	g.Y.Max = 1
	g.Y.AutoScale()
	g.X.AutoTimeLogScale(g)
	return g.SVG()
}

func scatter(data []Mark, algs []string, op, title string) string {
	const (
		Width  = 600
		Height = 300
	)
	g := &svg.Graph{
		Style:   svg.Style,
		LegendX: Width - 95,
		LegendY: 60,
		BBox:    image.Rect(0, 0, Width, Height),
		DBox:    image.Rect(80, 40, Width-100, Height-50),
		Title:   title,
	}
	g.X.Title = "log₂(f)"
	g.Y.Title = "Time"
	g.X.Min = -1024
	g.X.Max = +1024
	g.X.Ticks = []svg.Tick{
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

	byAlg := make(map[string]*svg.Scatter)
	for _, alg := range algs {
		s := &svg.Scatter{Name: alg}
		byAlg[alg] = s
		g.Scatters = append(g.Scatters, s)
	}

	for _, m := range data {
		s := byAlg[m.Alg]
		if m.Dist != "rand" || m.Op != op || s == nil {
			continue
		}
		y := float64(m.T)
		s.Points = append(s.Points, svg.Point{math.Log2(m.F), y})
		g.Y.Min = min(g.Y.Min, y)
		g.Y.Max = max(g.Y.Max, y)
	}
	if g.Y.Max < g.Y.Min {
		return ""
	}

	g.Y.AutoTimeLogScale(g)
	return g.SVG()
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
