// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fpfmt

import (
	"bytes"
	"fmt"
	"os"
	"text/tabwriter"
)

var rows bytes.Buffer

func row(cols ...any) {
	for i, c := range cols {
		if i > 0 {
			rows.WriteString("\t")
		}
		fmt.Fprint(&rows, c)
	}
	rows.WriteString("\n")
}

func table() {
	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	tw.Write(rows.Bytes())
	tw.Flush()
	rows.Reset()
}

func Example_unround() {
	// Output:
	// x      raw  str
	// 6      24   ⟨6.0⟩
	// 6.001  25   ⟨6.0+⟩
	// 6.499  25   ⟨6.0+⟩
	// 6.5    26   ⟨6.5⟩
	// 6.501  27   ⟨6.5+⟩
	// 6.999  27   ⟨6.5+⟩
	// 7      28   ⟨7.0⟩

	row("x", "raw", "str")
	for _, x := range []float64{6, 6.001, 6.499, 6.5, 6.501, 6.999, 7} {
		u := unround(x)
		row(x, uint64(u), u)
	}
	table()
}

func Example_unround_floor() {
	// Output:
	// x       floor  round½↓  round  round½↑  ceil
	// ⟨6.0⟩   6      6        6      6        6
	// ⟨6.0+⟩  6      6        6      6        7
	// ⟨6.0+⟩  6      6        6      6        7
	// ⟨6.5⟩   6      6        6      7        7
	// ⟨6.5+⟩  6      7        7      7        7
	// ⟨6.5+⟩  6      7        7      7        7
	// ⟨7.0⟩   7      7        7      7        7
	// ⟨7.5⟩   7      7        8      8        8
	// ⟨8.5⟩   8      8        8      9        9

	row("x", "floor", "round½↓", "round", "round½↑", "ceil")
	for _, x := range []float64{6, 6.001, 6.499, 6.5, 6.501, 6.999, 7, 7.5, 8.5} {
		u := unround(x)
		row(u, u.floor(), u.roundHalfDown(), u.round(), u.roundHalfUp(), u.ceil())
	}
	table()
}

func Example_div() {
	// Output:
	// ⟨2.5+⟩ 3

	u := unround(15.1).div(6)
	fmt.Println(u, u.round())
}

func Example_nudge() {
	// Output:
	// x        nudge(-1).floor  floor  ceil  nudge(+1).ceil
	// ⟨15.0⟩   14               15     15    16
	// ⟨15.0+⟩  15               15     16    16
	// ⟨15.5+⟩  15               15     16    16
	// ⟨16.0⟩   15               16     16    17

	row("x", "nudge(-1).floor", "floor", "ceil", "nudge(+1).ceil")
	for _, x := range []float64{15, 15.1, 15.9, 16} {
		u := unround(x)
		row(u, u.nudge(-1).floor(), u.floor(), u.ceil(), u.nudge(+1).ceil())
	}
	table()
}
