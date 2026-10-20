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
	"strings"
	"testing"
)

func TestUnifiedDiff_ShowsChangedLinesWithContext(t *testing.T) {
	base := &symbol{text: "a\nb\nc\nd\ne\nf\ng\nh\ni\n", loc: "x.go:1"}
	head := &symbol{text: "a\nb\nc\nd\nE\nf\ng\nh\ni\n", loc: "x.go:1"}
	want := "--- base x.go:1\n+++ head x.go:1\n@@ -2,7 +2,7 @@\n b\n c\n d\n-e\n+E\n f\n g\n h\n"
	if got := diffSymbols(base, head).unified(); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestUnifiedDiff_SeparatesDistantChangesIntoHunks(t *testing.T) {
	var a, b []string
	for i := range 20 {
		a = append(a, string(rune('a'+i)))
		b = append(b, string(rune('a'+i)))
	}
	b[1], b[18] = "B", "S"
	got := diffSymbols(&symbol{text: strings.Join(a, "\n")}, &symbol{text: strings.Join(b, "\n")}).unified()
	if n := strings.Count(got, "@@ -"); n != 2 {
		t.Errorf("want 2 hunks, got %d:\n%s", n, got)
	}
}

func TestDiffStat_CountsAddedAndRemovedLines(t *testing.T) {
	tests := []struct {
		base, head *symbol
		add, del   int
	}{
		{&symbol{text: "a\nb\n"}, &symbol{text: "a\nc\nd\n"}, 2, 1},
		{nil, &symbol{text: "a\nb\n"}, 2, 0},
		{&symbol{text: "a\nb\nc\n"}, nil, 0, 3},
	}
	for _, tc := range tests {
		if add, del := diffSymbols(tc.base, tc.head).stat(); add != tc.add || del != tc.del {
			t.Errorf("stat of %v -> %v = +%d -%d, want +%d -%d", tc.base, tc.head, add, del, tc.add, tc.del)
		}
	}
}
