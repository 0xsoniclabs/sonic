// Copyright 2026 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"strings"
)

const (
	diffContext  = 3
	maxDiffCells = 1 << 22
)

type edit struct {
	op   byte
	line string
}

func lines(s *symbol) []string {
	if s == nil {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s.text, "\n"), "\n")
}

// diffLines returns a shortest edit script turning a into b, or false if the
// inputs are too large for the quadratic algorithm.
func diffLines(a, b []string) ([]edit, bool) {
	pre := 0
	for pre < len(a) && pre < len(b) && a[pre] == b[pre] {
		pre++
	}
	suf := 0
	for suf < len(a)-pre && suf < len(b)-pre && a[len(a)-1-suf] == b[len(b)-1-suf] {
		suf++
	}
	ma, mb := a[pre:len(a)-suf], b[pre:len(b)-suf]
	if (len(ma)+1)*(len(mb)+1) > maxDiffCells {
		return nil, false
	}
	w := len(mb) + 1
	lcs := make([]int32, (len(ma)+1)*w)
	for i := len(ma) - 1; i >= 0; i-- {
		for j := len(mb) - 1; j >= 0; j-- {
			if ma[i] == mb[j] {
				lcs[i*w+j] = lcs[(i+1)*w+j+1] + 1
			} else {
				lcs[i*w+j] = max(lcs[(i+1)*w+j], lcs[i*w+j+1])
			}
		}
	}
	out := make([]edit, 0, len(a)+len(b))
	for _, l := range a[:pre] {
		out = append(out, edit{' ', l})
	}
	i, j := 0, 0
	for i < len(ma) || j < len(mb) {
		switch {
		case i < len(ma) && j < len(mb) && ma[i] == mb[j]:
			out = append(out, edit{' ', ma[i]})
			i++
			j++
		case j == len(mb) || (i < len(ma) && lcs[(i+1)*w+j] >= lcs[i*w+j+1]):
			out = append(out, edit{'-', ma[i]})
			i++
		default:
			out = append(out, edit{'+', mb[j]})
			j++
		}
	}
	for _, l := range a[len(a)-suf:] {
		out = append(out, edit{' ', l})
	}
	return out, true
}

type textDiff struct {
	from, to string
	a, b     []string
	edits    []edit
	ok       bool
}

func diffSymbols(base, head *symbol) textDiff {
	d := textDiff{from: locOf(base), to: locOf(head), a: lines(base), b: lines(head)}
	d.edits, d.ok = diffLines(d.a, d.b)
	return d
}

func (d textDiff) stat() (add, del int) {
	if d.ok {
		for _, e := range d.edits {
			switch e.op {
			case '+':
				add++
			case '-':
				del++
			}
		}
		return add, del
	}
	count := map[string]int{}
	for _, l := range d.a {
		count[l]++
	}
	for _, l := range d.b {
		if count[l] > 0 {
			count[l]--
		} else {
			add++
		}
	}
	for _, c := range count {
		del += c
	}
	return add, del
}

func (d textDiff) unified() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "--- base %s\n+++ head %s\n", d.from, d.to)
	if !d.ok {
		fmt.Fprintf(&buf, "(too large to show here: %d -> %d lines)\n", len(d.a), len(d.b))
		return buf.String()
	}
	edits := d.edits
	aLine := make([]int, len(edits)+1)
	bLine := make([]int, len(edits)+1)
	for k, e := range edits {
		aLine[k+1], bLine[k+1] = aLine[k], bLine[k]
		if e.op != '+' {
			aLine[k+1]++
		}
		if e.op != '-' {
			bLine[k+1]++
		}
	}
	for i := 0; i < len(edits); {
		if edits[i].op == ' ' {
			i++
			continue
		}
		start, end := max(0, i-diffContext), i
		for end < len(edits) {
			if edits[end].op != ' ' {
				end++
				continue
			}
			k := end
			for k < len(edits) && edits[k].op == ' ' {
				k++
			}
			if k == len(edits) || k-end > 2*diffContext {
				break
			}
			end = k
		}
		stop := min(len(edits), end+diffContext)
		fmt.Fprintf(&buf, "@@ -%d,%d +%d,%d @@\n",
			aLine[start]+1, aLine[stop]-aLine[start], bLine[start]+1, bLine[stop]-bLine[start])
		for _, e := range edits[start:stop] {
			buf.WriteByte(e.op)
			buf.WriteString(e.line)
			buf.WriteByte('\n')
		}
		i = stop
	}
	return buf.String()
}

func locOf(s *symbol) string {
	if s == nil {
		return "/dev/null"
	}
	return s.loc
}
