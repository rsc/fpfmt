package svg

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"strconv"
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
	for p := int(math.Round(lmin)); p <= int(math.Round(lmax)); p++ {
		for m := range 9 {
			x := math.Pow10(p) * float64(m+1)
			if math.Log10(x) < a.Min || a.Max < math.Log10(x) {
				continue
			}
			label := ""
			if m == 0 || m == 2 {
				label = fmt.Sprint(x)
			}
			a.Ticks = append(a.Ticks, Tick{At: math.Log10(x), Label: label})
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
	var path bytes.Buffer
	fmt.Fprintf(&path, "M %d %d L %d %d L %d %d L %d %d Z",
		g.BBox.Min.X, g.BBox.Min.Y,
		g.BBox.Min.X, g.BBox.Max.Y,
		g.BBox.Max.X, g.BBox.Max.Y,
		g.BBox.Max.X, g.BBox.Min.Y,
	)
	g.inner += g.elem("path", g.attrs("class", "bbox", "d", path.String()), "")
}

func (g *Graph) SVG() string {
	for _, s := range g.Scatters {
		g.drawScatter(s)
	}
	for _, l := range g.Lines {
		g.drawLine(l)
	}
	g.drawLegend()
	g.drawAxes()
	g.drawBBox()
	return svgHeader + g.elem("svg",
		g.attrs(
			"width", g.BBox.Dx(),
			"height", g.BBox.Dy(),
			"version", "1.1",
			"xmlns", "http://www.w3.org/2000/svg",
		),
		"<defs><style type='text/css'><![CDATA["+g.Style+"]]></style></defs>"+g.inner,
	)
}

const svgHeader = `<?xml version="1.0" standalone="no"?>
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
`

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
