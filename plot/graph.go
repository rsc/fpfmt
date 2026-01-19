// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Graph struct {
	Style string

	BBox     image.Rectangle // bounding box of graph
	DBox     image.Rectangle // bounding box of data
	LegendX  int
	LegendY  int
	Title    string
	X        Axis
	Y        Axis
	Scatters []*Scatter
	Lines    []*Line

	inner string
}

type Label struct {
	At   Point
	Text string
}

type Axis struct {
	Title string
	Min   float64
	Max   float64
	Ticks []Tick
}

type Scatter struct {
	Name   string
	Legend Point
	Points []Point
}

type Line struct {
	Name   string
	Legend Point
	Points []Point
}

type Point struct {
	X float64
	Y float64
}

type Tick struct {
	At    float64
	Label string
	Minor bool
}

func (a *Axis) AutoScale() {
	p := int(math.Floor(math.Log10(a.Max-a.Min))) - 1
	scale := math.Pow10(p)
	a.Min -= math.Mod(a.Min, scale)
	a.Max += scale - math.Mod(a.Max, scale)
	imin := int(math.Round(math.Mod(a.Min, scale*10) / scale))
	isize := int(math.Round((a.Max - a.Min) / scale))
	for i := 0; i <= isize; i++ {
		x := a.Min + float64(i)*scale
		label := ""
		if (imin+i)%5 == 0 {
			label = fmt.Sprint(x)
		}
		a.Ticks = append(a.Ticks, Tick{At: x, Label: label})
	}
}

func (g *Graph) minX() float64 {
	x := math.Inf(+1)
	for _, s := range g.Scatters {
		for _, p := range s.Points {
			x = min(x, p.X)
		}
	}
	for _, l := range g.Lines {
		for _, p := range l.Points {
			x = min(x, p.X)
		}
	}
	return x
}

func (g *Graph) minY() float64 {
	y := math.Inf(+1)
	for _, s := range g.Scatters {
		for _, p := range s.Points {
			y = min(y, p.Y)
		}
	}
	for _, l := range g.Lines {
		for _, p := range l.Points {
			y = min(y, p.Y)
		}
	}
	return y
}

func (a *Axis) AutoLogScale(g *Graph) {
	if a.Min == 0 {
		if a == &g.X {
			a.Min = g.minX()
		} else {
			a.Min = g.minY()
		}
	}
	a.Min = math.Log10(a.Min)
	a.Max = math.Log10(a.Max)
	a.Min -= (a.Max - a.Min) / 50
	lmin := math.Floor(a.Min)
	lmax := math.Ceil(a.Max)
	numLabel := 0
	for p := int(math.Round(lmin)); p <= int(math.Round(lmax)); p++ {
		for m := range 9 {
			x := math.Pow10(p) * float64(m+1)
			if math.Log10(x) < a.Min || a.Max < math.Log10(x) {
				continue
			}
			label := ""
			if m == 0 || m == 2 {
				label = fmt.Sprint(x)
				numLabel++
			}
			a.Ticks = append(a.Ticks, Tick{At: math.Log10(x), Label: label})
		}
	}
	if numLabel < 2 {
		ti := 0
		for p := int(math.Round(lmin)); p <= int(math.Round(lmax)); p++ {
			for m := range 9 {
				x := math.Pow10(p) * float64(m+1)
				if math.Log10(x) < a.Min || a.Max < math.Log10(x) {
					continue
				}
				a.Ticks[ti].Label = fmt.Sprint(x)
				ti++
			}
		}
	}
	if a == &g.X {
		for _, s := range g.Scatters {
			for i := range s.Points {
				p := &s.Points[i]
				p.X = math.Log10(p.X)
			}
		}
		for _, l := range g.Lines {
			for i := range l.Points {
				p := &l.Points[i]
				p.X = math.Log10(p.X)
			}
		}
	} else {
		for _, s := range g.Scatters {
			for i := range s.Points {
				p := &s.Points[i]
				p.Y = math.Log10(p.Y)
			}
		}
		for _, l := range g.Lines {
			for i := range l.Points {
				p := &l.Points[i]
				p.Y = math.Log10(p.Y)
			}
		}
	}
}

func (a *Axis) AutoTimeScale() {
	a.AutoScale()
	for i := range a.Ticks {
		t := &a.Ticks[i]
		if t.Label != "" {
			t.Label = time.Duration(t.At).String()
		}
	}
}

func (a *Axis) AutoTimeLogScale(g *Graph) {
	a.AutoLogScale(g)
	for i := range a.Ticks {
		t := &a.Ticks[i]
		if t.Label != "" {
			f, err := strconv.ParseFloat(t.Label, 64)
			if err == nil {
				t.Label = time.Duration(f).String()
			}
		}
	}
}

type attrString string

func (g *Graph) attrs(list ...any) attrString {
	var buf bytes.Buffer
	for i := 0; i < len(list); i += 2 {
		name := list[i].(string)
		val := list[i+1]
		fmt.Fprintf(&buf, " %v='%v'", name, val)
	}
	return attrString(buf.String())
}

func (g *Graph) elem(tag string, attrs attrString, body string) string {
	return "<" + tag + string(attrs) + ">" + body + "</" + tag + ">"
}

func (g *Graph) line(class string, x1, y1, x2, y2 float64) string {
	return g.elem("line", g.attrs("x1", x1, "y1", y1, "x2", x2, "y2", y2, "class", class), "")
}

func (g *Graph) rect(class string, x1, y1, x2, y2 float64) string {
	return g.elem("rect", g.attrs("x", x1, "y", y1, "width", x2-x1, "height", y2-y1, "class", class), "")
}

func (g *Graph) text(class string, x, y float64, body string) string {
	return g.elem("text", g.attrs("x", x, "y", y, "class", class), body)
}

func (g *Graph) dataX(x float64) float64 {
	return float64(g.DBox.Min.X) + (x-g.X.Min)/(g.X.Max-g.X.Min)*float64(g.DBox.Dx())
}

func (g *Graph) dataY(y float64) float64 {
	return float64(g.DBox.Min.Y) + (g.Y.Max-y)/(g.Y.Max-g.Y.Min)*float64(g.DBox.Dy())
}

func (g *Graph) drawAxes() {
	g.inner += g.line("axis", float64(g.DBox.Min.X), float64(g.DBox.Max.Y), float64(g.DBox.Max.X), float64(g.DBox.Max.Y))
	g.inner += g.line("axis", float64(g.DBox.Min.X), float64(g.DBox.Min.Y), float64(g.DBox.Min.X), float64(g.DBox.Max.Y))

	g.inner += g.text("xaxis", float64(g.DBox.Min.X+g.DBox.Max.X)/2, float64(g.BBox.Max.Y)-10, g.X.Title)
	g.inner += g.text("yaxis", float64(g.BBox.Min.X)+5, float64(g.DBox.Min.Y+g.DBox.Max.Y)/2, g.Y.Title)

	g.inner += g.text("title", float64(g.DBox.Min.X+g.DBox.Max.X)/2, float64(g.BBox.Min.Y)+10, g.Title)

	var ticks bytes.Buffer
	for _, t := range g.X.Ticks {
		x := g.dataX(t.At)
		y := float64(g.DBox.Max.Y)
		tickLen := 10.0
		if t.Label == "" {
			tickLen = 5
		}
		fmt.Fprintf(&ticks, "M %v %v l %v %v ", x, y, 0, tickLen)
		if t.Label != "" {
			g.inner += g.text("xtick", x, y+tickLen+2, t.Label)
		}
	}
	for _, t := range g.Y.Ticks {
		x := float64(g.DBox.Min.X)
		y := g.dataY(t.At)
		tickLen := 10.0
		if t.Label == "" {
			tickLen = 5
		}
		fmt.Fprintf(&ticks, "M %v %v l %v %v ", x, y, -tickLen, 0)
		if t.Label != "" {
			g.inner += g.text("ytick", x-tickLen-2, y, t.Label)
		}
	}
	g.inner += g.elem("path", g.attrs("class", "tick", "d", ticks.String()), "")
}

func (g *Graph) drawLegend() {
	lx := g.DBox.Min.X
	if g.LegendX != 0 {
		lx = g.LegendX
	}
	ly := g.DBox.Min.Y
	if g.LegendY != 0 {
		ly = g.LegendY
	}
	y := float64(ly) + 10
	x := float64(lx) + 20
	draw := func(name string) {
		g.inner += g.elem("path", g.attrs("class", "legend "+name, "d", fmt.Sprintf("M %.3f %.3f l -15 0", x, y)), "")
		g.inner += g.text("legend "+name, x+2, y+0.5, name)
		y += 20
	}
	for _, s := range g.Scatters {
		if len(s.Points) >= 2 {
			draw(s.Name)
		}
	}
	for _, l := range g.Lines {
		if len(l.Points) >= 2 {
			draw(l.Name)
		}
	}
}

func (g *Graph) drawScatter(s *Scatter) {
	if len(s.Points) == 0 {
		return
	}
	var path bytes.Buffer
	for _, p := range s.Points {
		x, y := g.dataX(p.X), g.dataY(p.Y)
		r := 0.5
		fmt.Fprintf(&path, "M %.3f %.3f a %v %v 0 1 0 0 %v a %v %v 0 1 0 0 %v ", x, y, r, r, 2*r, r, r, -2*r)
	}
	g.inner += g.elem("path", g.attrs("class", "scat "+s.Name, "d", path.String()), "")
}

func (g *Graph) drawLine(l *Line) {
	if len(l.Points) < 2 {
		return
	}
	var path bytes.Buffer
	p := l.Points[0]
	fmt.Fprintf(&path, "M %.3f %.3f ", g.dataX(p.X), g.dataY(p.Y))
	for _, p := range l.Points[1:] {
		fmt.Fprintf(&path, "L %.3f %.3f ", g.dataX(p.X), g.dataY(p.Y))
	}
	g.inner += g.elem("path", g.attrs("class", "line "+l.Name, "d", path.String()), "")
}

func (g *Graph) drawBBox() {
	return

	var path bytes.Buffer
	fmt.Fprintf(&path, "M %d %d L %d %d L %d %d L %d %d Z",
		g.BBox.Min.X, g.BBox.Min.Y,
		g.BBox.Min.X, g.BBox.Max.Y,
		g.BBox.Max.X, g.BBox.Max.Y,
		g.BBox.Max.X, g.BBox.Min.Y,
	)
	g.inner += g.elem("path", g.attrs("class", "bbox", "d", path.String()), "")
}

func (g *Graph) draw() {
	g.inner = ""
	for _, s := range g.Scatters {
		g.drawScatter(s)
	}
	for _, l := range g.Lines {
		g.drawLine(l)
	}
	g.drawLegend()
	g.drawAxes()
}

func (g *Graph) svg(big bool) string {
	g.draw()
	//g.drawBBox()
	w := g.BBox.Dx()
	h := g.BBox.Dy()
	inner := g.inner
	if big {
		w *= 2
		h *= 2
		inner = g.elem("g", g.attrs("transform", "scale(2)"), inner)
	}
	return svgHeader + g.elem("svg",
		g.attrs(
			"width", w,
			"height", h,
			"version", "1.1",
			"xmlns", "http://www.w3.org/2000/svg",
		),
		"<defs><style type='text/css'><![CDATA["+g.Style+"]]></style></defs>"+inner,
	)
}

func (g *Graph) BigSVG() string { return g.svg(true) }
func (g *Graph) SVG() string    { return g.svg(false) }

const svgHeader = `<?xml version="1.0" standalone="no"?>
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
`

func (g *Graph) EPS() string {
	g.draw()
	var buf bytes.Buffer
	// EPS header
	fmt.Fprintf(&buf, "%%!PS-Adobe-3.0 EPSF-3.0\n")
	fmt.Fprintf(&buf, "%%%%BoundingBox: %d %d %d %d\n", g.BBox.Min.X, g.BBox.Min.Y, g.BBox.Max.X, g.BBox.Max.Y)
	fmt.Fprintf(&buf, "%%%%Creator: svg.Graph.EPS\n")
	fmt.Fprintf(&buf, "%%%%EndComments\n\n")

	// Setup - re-encode fonts to ISOLatin1Encoding for proper µ symbol
	fmt.Fprintf(&buf, "/MinionPro-Regular findfont\n")
	fmt.Fprintf(&buf, "dup length dict begin\n")
	fmt.Fprintf(&buf, "  {1 index /FID ne {def} {pop pop} ifelse} forall\n")
	fmt.Fprintf(&buf, "  /Encoding ISOLatin1Encoding def\n")
	fmt.Fprintf(&buf, "  currentdict\n")
	fmt.Fprintf(&buf, "end\n")
	fmt.Fprintf(&buf, "/MinionPro-Regular-ISOLatin1 exch definefont pop\n")
	fmt.Fprintf(&buf, "/MinionPro-Bold findfont\n")
	fmt.Fprintf(&buf, "dup length dict begin\n")
	fmt.Fprintf(&buf, "  {1 index /FID ne {def} {pop pop} ifelse} forall\n")
	fmt.Fprintf(&buf, "  /Encoding ISOLatin1Encoding def\n")
	fmt.Fprintf(&buf, "  currentdict\n")
	fmt.Fprintf(&buf, "end\n")
	fmt.Fprintf(&buf, "/MinionPro-Bold-ISOLatin1 exch definefont pop\n")
	fmt.Fprintf(&buf, "/MinionPro-Regular-ISOLatin1 findfont 16 scalefont setfont\n")
	fmt.Fprintf(&buf, "1 setlinewidth\n")
	fmt.Fprintf(&buf, "0 setlinejoin\n")  // 0 = miter join, matches SVG default
	fmt.Fprintf(&buf, "0 setlinecap\n\n") // 0 = butt cap, matches SVG default

	// Helper procedures
	fmt.Fprintf(&buf, "/M { moveto } def\n")
	fmt.Fprintf(&buf, "/L { lineto } def\n")
	fmt.Fprintf(&buf, "/l { rlineto } def\n")
	fmt.Fprintf(&buf, "/a { 0 360 arc } def\n")
	fmt.Fprintf(&buf, "/S { stroke } def\n")
	fmt.Fprintf(&buf, "/F { fill } def\n")
	fmt.Fprintf(&buf, "/Z { closepath } def\n\n")

	// Color setup - convert SVG inner content to PostScript
	buf.WriteString(g.epsInner())

	fmt.Fprintf(&buf, "%%%%EOF\n")
	return buf.String()
}

func (g *Graph) epsInner() string {
	var buf bytes.Buffer

	// Parse the SVG inner content and convert to PostScript
	// This is a simplified conversion - handles the main elements
	s := g.inner

	// Set up clipping region for data box to prevent scatter points from covering axes
	fmt.Fprintf(&buf, "gsave\n")
	fmt.Fprintf(&buf, "newpath\n")

	if false {
		fmt.Fprintf(&buf, "%.3f %.3f M\n", float64(g.DBox.Min.X), g.flipY(float64(g.DBox.Max.Y)))
		fmt.Fprintf(&buf, "%.3f %.3f L\n", float64(g.DBox.Max.X), g.flipY(float64(g.DBox.Max.Y)))
		fmt.Fprintf(&buf, "%.3f %.3f L\n", float64(g.DBox.Max.X), g.flipY(float64(g.DBox.Min.Y)))
		fmt.Fprintf(&buf, "%.3f %.3f L\n", float64(g.DBox.Min.X), g.flipY(float64(g.DBox.Min.Y)))
		fmt.Fprintf(&buf, "closepath clip newpath\n\n")
	}

	// Process path elements (scatter and line data) - these should be clipped
	s2 := g.inner
	for {
		start := strings.Index(s2, "<path")
		if start == -1 {
			break
		}
		end := strings.Index(s2[start:], "</path>")
		if end == -1 {
			break
		}
		path := s2[start : start+end+7]
		s2 = s2[start+end+7:]

		d := g.extractStrAttr(path, "d")
		class := g.extractStrAttr(path, "class")

		// Only clip scatter and line data, not tick marks, legend, or bbox
		if strings.Contains(class, "scat ") || strings.Contains(class, "line ") {
			if d != "" {
				g.epsSetColor(&buf, class)
				g.epsSetLineWidth(&buf, class)
				g.epsPath(&buf, d)
			}
		}
	}

	// End clipping region
	fmt.Fprintf(&buf, "grestore\n\n")

	// Process line elements (axes) - these should NOT be clipped
	for {
		start := strings.Index(s, "<line")
		if start == -1 {
			break
		}
		end := strings.Index(s[start:], "</line>")
		if end == -1 {
			break
		}
		line := s[start : start+end+7]
		s = s[start+end+7:]

		x1 := g.extractAttr(line, "x1")
		y1 := g.extractAttr(line, "y1")
		x2 := g.extractAttr(line, "x2")
		y2 := g.extractAttr(line, "y2")
		class := g.extractStrAttr(line, "class")

		g.epsSetColor(&buf, class)
		fmt.Fprintf(&buf, "1 setlinewidth\n")
		fmt.Fprintf(&buf, "%.3f %.3f M %.3f %.3f L S\n", x1, g.flipY(y1), x2, g.flipY(y2))
	}

	// Process remaining path elements (ticks, legend, bbox) - these should NOT be clipped
	s = g.inner
	for {
		start := strings.Index(s, "<path")
		if start == -1 {
			break
		}
		end := strings.Index(s[start:], "</path>")
		if end == -1 {
			break
		}
		path := s[start : start+end+7]
		s = s[start+end+7:]

		d := g.extractStrAttr(path, "d")
		class := g.extractStrAttr(path, "class")

		// Skip scatter and line data (already processed above)
		if strings.Contains(class, "scat ") || strings.Contains(class, "line ") {
			continue
		}

		if d != "" {
			g.epsSetColor(&buf, class)
			g.epsSetLineWidth(&buf, class)
			g.epsPath(&buf, d)
		}
	}

	// Process text elements
	s = g.inner
	for {
		start := strings.Index(s, "<text")
		if start == -1 {
			break
		}
		end := strings.Index(s[start:], "</text>")
		if end == -1 {
			break
		}
		textElem := s[start : start+end+7]
		s = s[start+end+7:]

		x := g.extractAttr(textElem, "x")
		y := g.extractAttr(textElem, "y")
		class := g.extractStrAttr(textElem, "class")

		// Extract text content
		textStart := strings.Index(textElem, ">")
		textEnd := strings.Index(textElem, "</text>")
		if textStart != -1 && textEnd != -1 {
			text := textElem[textStart+1 : textEnd]
			g.epsText(&buf, class, x, g.flipY(y), text)
		}
	}

	return buf.String()
}

func (g *Graph) flipY(y float64) float64 {
	return float64(g.BBox.Max.Y-g.BBox.Min.Y) - y
}

func (g *Graph) extractAttr(elem, attr string) float64 {
	pattern := attr + "='"
	start := strings.Index(elem, pattern)
	if start == -1 {
		return 0
	}
	start += len(pattern)
	end := strings.Index(elem[start:], "'")
	if end == -1 {
		return 0
	}
	val := elem[start : start+end]
	f, _ := strconv.ParseFloat(val, 64)
	return f
}

func (g *Graph) extractStrAttr(elem, attr string) string {
	pattern := attr + "='"
	start := strings.Index(elem, pattern)
	if start == -1 {
		return ""
	}
	start += len(pattern)
	end := strings.Index(elem[start:], "'")
	if end == -1 {
		return ""
	}
	return elem[start : start+end]
}

func (g *Graph) epsSetColor(buf *bytes.Buffer, class string) {
	// Map CSS classes to PostScript colors
	colors := map[string]string{
		"uscale":     "0.533 0.533 0.533", // #888
		"uscalec":    "0 0 0",             // #000
		"uscalet":    "1 1 0",             // yellow
		"dragonbox":  "0.533 0.533 1",     // #88f
		"abseil":     "0.533 0.533 1",     // #88f
		"ryu":        "0 0 1",             // #00f
		"fast_float": "0 0 1",             // #00f
		"dblconv":    "1 0.647 0",         // orange
		"dmg1997":    "1 0.533 0.533",     // #f88
		"dmg2017":    "1 0 0",             // #f00
		"libc":       "1 0.867 0",         // #fd0
		"go125":      "0.933 0.510 0.933", // violet
		"schubfach":  "1 0.867 0",         // #fd0
	}

	// Extract the name from compound classes like "line dragonbox" or "scat ryu"
	parts := strings.Fields(class)
	for _, part := range parts {
		if color, ok := colors[part]; ok {
			fmt.Fprintf(buf, "%s setrgbcolor\n", color)
			return
		}
	}

	// Default to black
	fmt.Fprintf(buf, "0 0 0 setrgbcolor\n")
}

func (g *Graph) epsSetLineWidth(buf *bytes.Buffer, class string) {
	if strings.Contains(class, "line ") || strings.Contains(class, "legend") {
		fmt.Fprintf(buf, "2 setlinewidth\n")
	} else if strings.Contains(class, "tick") {
		fmt.Fprintf(buf, "1 setlinewidth\n")
	} else {
		fmt.Fprintf(buf, "1 setlinewidth\n")
	}
}

func (g *Graph) epsPath(buf *bytes.Buffer, d string) {
	// Parse SVG path data and convert to PostScript
	parts := strings.Fields(d)
	fill := false
	lastX, lastY := 0.0, 0.0

	for i := 0; i < len(parts); {
		cmd := parts[i]
		i++

		switch cmd {
		case "M":
			if i+1 < len(parts) {
				x, _ := strconv.ParseFloat(parts[i], 64)
				y, _ := strconv.ParseFloat(parts[i+1], 64)
				lastX, lastY = x, y
				fmt.Fprintf(buf, "%.3f %.3f M\n", x, g.flipY(y))
				i += 2
			}
		case "L":
			if i+1 < len(parts) {
				x, _ := strconv.ParseFloat(parts[i], 64)
				y, _ := strconv.ParseFloat(parts[i+1], 64)
				lastX, lastY = x, y
				fmt.Fprintf(buf, "%.3f %.3f L\n", x, g.flipY(y))
				i += 2
			}
		case "l":
			if i+1 < len(parts) {
				dx, _ := strconv.ParseFloat(parts[i], 64)
				dy, _ := strconv.ParseFloat(parts[i+1], 64)
				fmt.Fprintf(buf, "%.3f %.3f l\n", dx, -dy)
				i += 2
			}
		case "a":
			// SVG arc for scatter points - draw a filled circle
			if i+6 < len(parts) {
				rx, _ := strconv.ParseFloat(parts[i], 64)
				// Skip arc parameters (rx ry x-axis-rotation large-arc sweep dx dy)
				i += 7
				if rx > 0 {
					// Draw a filled circle at lastX, lastY with radius rx
					// Use newpath to ensure each circle is independent
					// Scale radius slightly smaller for EPS to match SVG appearance
					fmt.Fprintf(buf, "newpath %.3f %.3f %.3f 0 360 arc F\n", lastX, g.flipY(lastY), rx*0.8)
					fill = true
				}
			}
		case "Z":
			fmt.Fprintf(buf, "Z\n")
		}
	}

	if !fill {
		fmt.Fprintf(buf, "S\n")
	}
}

func (g *Graph) epsText(buf *bytes.Buffer, class string, x, y float64, text string) {
	fmt.Fprintf(buf, "0 0 0 setrgbcolor\n")

	// Replace UTF-8 characters with PostScript equivalents
	text = strings.ReplaceAll(text, "µ", "\\265") // micro sign
	text = strings.ReplaceAll(text, "−", "-")     // minus sign U+2212 -> hyphen-minus

	// Handle text anchor based on class
	if strings.Contains(class, "xtick") {
		// Center aligned, hanging baseline - move down slightly
		fmt.Fprintf(buf, "%.3f %.3f M\n", x, y-13)
		fmt.Fprintf(buf, "(%s) dup stringwidth pop 2 div neg 0 rmoveto show\n", text)
	} else if strings.Contains(class, "ytick") {
		// Right aligned, vertically centered - offset down by ~1/3 font height
		fmt.Fprintf(buf, "%.3f %.3f M\n", x, y-5)
		fmt.Fprintf(buf, "(%s) dup stringwidth pop neg 0 rmoveto show\n", text)
	} else if strings.Contains(class, "xaxis") {
		// Center aligned, X-axis label - use bold font
		fmt.Fprintf(buf, "/MinionPro-Bold-ISOLatin1 findfont 16 scalefont setfont\n")

		// Handle subscripts (e.g., "log₂(f)")
		if strings.Contains(text, "₂") {
			// Split into parts around the subscript
			parts := strings.Split(text, "₂")
			if len(parts) == 2 {
				// Calculate total width for centering
				fmt.Fprintf(buf, "(%s) stringwidth pop ", parts[0])
				fmt.Fprintf(buf, "/MinionPro-Bold-ISOLatin1 findfont 11 scalefont setfont\n")
				fmt.Fprintf(buf, "(2) stringwidth pop add ")
				fmt.Fprintf(buf, "/MinionPro-Bold-ISOLatin1 findfont 16 scalefont setfont\n")
				fmt.Fprintf(buf, "(%s) stringwidth pop add ", parts[1])
				fmt.Fprintf(buf, "2 div neg %.3f add %.3f M\n", x, y)

				// Render the parts
				fmt.Fprintf(buf, "(%s) show\n", parts[0])
				fmt.Fprintf(buf, "0 -4 rmoveto\n") // Move down for subscript
				fmt.Fprintf(buf, "/MinionPro-Bold-ISOLatin1 findfont 11 scalefont setfont\n")
				fmt.Fprintf(buf, "(2) show\n")
				fmt.Fprintf(buf, "/MinionPro-Bold-ISOLatin1 findfont 16 scalefont setfont\n")
				fmt.Fprintf(buf, "0 4 rmoveto\n") // Move back up
				fmt.Fprintf(buf, "(%s) show\n", parts[1])
			} else {
				// Fallback to simple rendering
				fmt.Fprintf(buf, "%.3f %.3f M\n", x, y)
				text = strings.ReplaceAll(text, "₂", "2")
				fmt.Fprintf(buf, "(%s) dup stringwidth pop 2 div neg 0 rmoveto show\n", text)
			}
		} else {
			fmt.Fprintf(buf, "%.3f %.3f M\n", x, y)
			fmt.Fprintf(buf, "(%s) dup stringwidth pop 2 div neg 0 rmoveto show\n", text)
		}
		fmt.Fprintf(buf, "/MinionPro-Regular-ISOLatin1 findfont 16 scalefont setfont\n")
	} else if strings.Contains(class, "title") {
		// Center aligned, title - hanging baseline means move down, use bold font
		fmt.Fprintf(buf, "/MinionPro-Bold-ISOLatin1 findfont 16 scalefont setfont\n")
		fmt.Fprintf(buf, "%.3f %.3f M\n", x, y-16)
		fmt.Fprintf(buf, "(%s) dup stringwidth pop 2 div neg 0 rmoveto show\n", text)
		fmt.Fprintf(buf, "/MinionPro-Regular-ISOLatin1 findfont 16 scalefont setfont\n")
	} else if strings.Contains(class, "yaxis") {
		// Rotated text for y-axis - need to position it so it doesn't get cropped, use bold font
		fmt.Fprintf(buf, "/MinionPro-Bold-ISOLatin1 findfont 16 scalefont setfont\n")
		fmt.Fprintf(buf, "gsave\n")
		fmt.Fprintf(buf, "%.3f %.3f translate\n", x, y)
		fmt.Fprintf(buf, "90 rotate\n")
		// After rotation, negative Y direction moves right in original coords
		fmt.Fprintf(buf, "0 -16 translate\n")
		fmt.Fprintf(buf, "0 0 M\n")
		fmt.Fprintf(buf, "(%s) dup stringwidth pop 2 div neg 0 rmoveto show\n", text)
		fmt.Fprintf(buf, "grestore\n")
		fmt.Fprintf(buf, "/MinionPro-Regular-ISOLatin1 findfont 16 scalefont setfont\n")
	} else if strings.Contains(class, "legend") {
		// Legend text - vertically centered, adjust for lowercase text
		fmt.Fprintf(buf, "%.3f %.3f M (%s) show\n", x, y-3, text)
	} else {
		// Default
		fmt.Fprintf(buf, "%.3f %.3f M (%s) show\n", x, y, text)
	}
}

const Style = `
text {
	font-family: 'Minion 3', serif;
	font-size: 16px;
}
line {
	stroke-width: 1px;
	stroke: black;
}
path.bbox {
	fill: none;
}
path {
	stroke-width: 1px;
	stroke: black;
}
.scat {
	stroke: none;
}
.xtick {
	text-anchor: middle;
	dominant-baseline: hanging;
}
.ytick {
	dominant-baseline: middle;
	text-anchor: end;
}

.line { fill: none; stroke-width: 2px; }
.line.uscale, path.legend.uscale { stroke: #888; }
.scat.uscale { fill: #888; }
.line.uscalec, path.legend.uscalec { stroke: #000; }
.scat.uscalec { fill: #000; }
.line.uscalet, path.legend.uscalet { stroke: yellow; }
.scat.uscalet { fill: yellow; }
.line.dragonbox, path.legend.dragonbox { stroke: #88f; }
.scat.dragonbox { fill: #88f; }
.line.abseil, path.legend.abseil { stroke: #88f; }
.scat.abseil { fill: #88f; }
.line.ryu, path.legend.ryu { stroke: #00f; }
.scat.ryu { fill: #00f; }
.line.fast_float, path.legend.fast_float { stroke: #00f; }
.scat.fast_float { fill: #00f; }
.line.dblconv, path.legend.dblconv { stroke: orange; }
.scat.dblconv { fill: orange; }
.line.dmg1997, path.legend.dmg1997 { stroke: #f88; }
.scat.dmg1997 { fill: #f88; }
.line.dmg2017, path.legend.dmg2017 { stroke: #f00; }
.scat.dmg2017 { fill: #f00; }
.line.libc, path.legend.libc { stroke: #fd0; }
.scat.libc { fill: #fd0; }
.line.go125, path.legend.go125 { stroke: violet; }
.scat.go125 { fill: violet; }
.line.schubfach, path.legend.schubfach { stroke: #fd0; }
.scat.schubfach { fill: #fd0; }

text.xaxis {
	dominant-baseline: auto;
	text-anchor: middle;
	font-weight: bold;
}
text.title {
	dominant-baseline: hanging;
	text-anchor: middle;
	font-weight: bold;
	/* text-decoration: underline; */
	font-size: 16px;
}
text.yaxis {
	transform: rotate(-90deg);
	transform-origin: 50% 0%;
	text-anchor: middle;
	dominant-baseline: hanging;
	transform-box: fill-box;
	font-weight: bold;
}

text.legend { dominant-baseline: middle; }
path.legend { stroke-width: 2px; }
`

// EPSToPNGs converts EPS to multiple PNG resolutions.
// Returns a map with keys: "file.png", "file@1.5x.png", "file@2x.png", "file@3x.png", "file@4x.png"
func EPSToPNGs(basename string, eps []byte, width, height int) (map[string][]byte, error) {
	// Write EPS to temporary file
	tmpEPS, err := os.CreateTemp("", "*.eps")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpEPS.Name())

	if _, err := tmpEPS.Write(eps); err != nil {
		return nil, err
	}
	if err := tmpEPS.Close(); err != nil {
		return nil, err
	}

	// Get font path from environment or use default
	fontPath := os.Getenv("GS_FONTPATH")
	if fontPath == "" {
		fontPath = "/Users/rsc/sys/font/Adobe/TypeClassics/MinionProOpticals:/Users/rsc/sys/font/Adobe/TypeClassics/MyriadPro:/Users/rsc/sys/font/Adobe/TypeClassics/NewsGothicStd:/Users/rsc/sys/font/Misc:/Users/rsc/sys/font/Go:/usr/local/plan9/postscript/font"
	}

	result := make(map[string][]byte)
	scales := []struct {
		suffix string
		scale  float64
	}{
		{"", 1.0},
		{"@1.5x", 1.5},
		{"@2x", 2.0},
		{"@3x", 3.0},
		{"@4x", 4.0},
	}

	for _, s := range scales {
		dpi := int(108 * s.scale)

		// Create temporary output file
		tmpPNG, err := os.CreateTemp("", "*.png")
		if err != nil {
			return nil, err
		}
		tmpPNGName := tmpPNG.Name()
		tmpPNG.Close()
		defer os.Remove(tmpPNGName)

		// Run gs command
		cmd := exec.Command("gs",
			"-q",
			"-dBATCH",
			"-dNOPAUSE",
			"-sDEVICE=png16m",
			fmt.Sprintf("-r%d", dpi),
			fmt.Sprintf("-dDEVICEWIDTHPOINTS=%d", width),
			fmt.Sprintf("-dDEVICEHEIGHTPOINTS=%d", height),
			"-dFIXEDMEDIA",
			"-dTextAlphaBits=4",
			"-dGraphicsAlphaBits=4",
			fmt.Sprintf("-sOutputFile=%s", tmpPNGName),
			tmpEPS.Name(),
		)
		cmd.Env = append(os.Environ(), "GS_FONTPATH="+fontPath)

		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("gs failed for %s: %w", s.suffix, err)
		}

		// Read the generated PNG
		pngData, err := os.ReadFile(tmpPNGName)
		if err != nil {
			return nil, err
		}

		key := basename + s.suffix + ".png"
		result[key] = pngData
	}

	return result, nil
}
