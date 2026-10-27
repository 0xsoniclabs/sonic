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

package review

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/scripts/geth_compatibility/internal/analysis"
)

const tableHeader = "| Verdict | Consensus | Symbol | Base | Head | Note |\n|---|---|---|---|---|---|\n"

func TestParse_RejectsMalformedRows(t *testing.T) {
	tests := map[string]string{
		"unknown verdict":          "| fine | - | `core func A` | aaa | bbb | looks fine |",
		"too few cells":            "| todo | - | `core func A` | aaa |",
		"too many cells":           "| todo | - | `core func A` | aaa | bbb | note | extra |",
		"duplicate symbol":         "| todo | - | `core func A` | aaa | bbb | |\n| todo | live | `core func A` | aaa | bbb | |",
		"unknown classification":   "| todo | yes | `core func A` | aaa | bbb | |",
		"unknown upgrade":          "| todo | amsterdam | `core func A` | aaa | bbb | |",
		"uppercase classification": "| todo | Live | `core func A` | aaa | bbb | |",
		"blank line in a table":    "| todo | - | `core func A` | aaa | bbb | |\n\n| todo | - | `core func B` | aaa | bbb | |",
		"renamed column":           "\n| Verdict | Consensus | Symbol | Base | Head | Notes |\n|---|---|---|---|---|---|\n| todo | - | `core func B` | aaa | bbb | |",
		"other table":              "\n| Verdict | Meaning |\n|---|---|\n| `todo` | not reviewed |",
	}
	for name, rows := range tests {
		if _, err := parse(tableHeader+rows, "review.md", []string{"canto"}); err == nil {
			t.Errorf("%s: accepted %q", name, rows)
		}
	}
}

func TestParse_ReadsTheRowsOfEveryTable(t *testing.T) {
	text := "# Review\n\nSome prose.\n\n## todo (1)\n\n" + tableHeader +
		"| todo     |       | `core func A` | aaa | bbb |  |\n" +
		"\n## other (2)\n\n" + tableHeader +
		"| inactive | -     | `core func B` | -   | ccc | Amsterdam only, see a \\| b |\n" +
		"| adopted  | canto | `core func C` | ccc | ddd | Canto only |\n"
	rows, err := parse(text, "review.md", []string{"canto"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]row{
		"core func A": {verdict: verdictTodo, base: "aaa", head: "bbb"},
		"core func B": {verdict: verdictInactive, consensus: consensusOff, base: "-", head: "ccc", note: "Amsterdam only, see a | b"},
		"core func C": {verdict: verdictAdopted, consensus: "canto", base: "ccc", head: "ddd", note: "Canto only"},
	}
	if !maps.Equal(rows, want) {
		t.Errorf("got rows %+v, want %+v", rows, want)
	}
}

func TestUpdate_KeepsMatchingRowsAndReopensChangedOnes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review.md")
	findings := []analysis.Finding{
		{Key: "core func Kept", Base: "a", Head: "b"},
		{Key: "core func Reopened", Base: "a", Head: "c", Consensus: true},
		{Key: "core func New", Base: "-", Head: "b"},
		{Key: "core func Pending", Base: "a", Head: "c"},
		{Key: "core func Question", Base: "a", Head: "c"},
		{Key: "core func Harmless", Base: "a", Head: "c", Consensus: true},
		{Key: "core func OnPath", Base: "a", Head: "b", Consensus: true},
		{Key: "core func OffPath", Base: "a", Head: "b"},
		{Key: "core func LeftPath", Base: "a", Head: "b"},
	}
	prev := "## old\n\n" + tableHeader +
		"| refactor | -     | `core func Kept`     | a | b | renamed locals |\n" +
		"| adopted  | live  | `core func Reopened` | a | b | covered by TestX |\n" +
		"| inactive | -     | `core func Gone`     | a | b | Amsterdam only |\n" +
		"| todo     | -     | `core func Pending`  | a | b | was adopted before the last change: covered by TestY |\n" +
		"| todo     | canto | `core func Question` | a | b | OPEN: accept the new limit? |\n" +
		"| adopted  | no    | `core func Harmless` | a | b | error string only |\n" +
		"| inactive | -     | `core func OnPath`   | a | b | never executed |\n" +
		"| adopted  |       | `core func OffPath`  | a | b | fixes a panic |\n" +
		"| refactor | no    | `core func LeftPath` | a | b | same results |\n"
	if err := os.WriteFile(path, []byte(prev), 0o644); err != nil {
		t.Fatal(err)
	}
	upgrades := []string{"canto"}
	todo, err := Update(path, &analysis.Side{Geth: "geth old"}, &analysis.Side{Geth: "geth new", Modules: []string{"vm v2"}, Upgrades: upgrades}, findings)
	if err != nil {
		t.Fatal(err)
	}
	if todo != 6 {
		t.Errorf("got %d todo rows, want 6", todo)
	}
	text, _ := os.ReadFile(path)
	rows, err := parse(string(text), path, upgrades)
	if err != nil {
		t.Fatalf("written review does not parse: %v\n%s", err, text)
	}
	want := map[string]row{
		"core func Kept":     {verdict: verdictRefactor, consensus: consensusOff, base: "a", head: "b", note: "renamed locals"},
		"core func Reopened": {verdict: verdictTodo, consensus: consensusLive, base: "a", head: "c", note: "was adopted before the last change: covered by TestX"},
		"core func New":      {verdict: verdictTodo, consensus: consensusOff, base: "-", head: "b"},
		"core func Pending":  {verdict: verdictTodo, consensus: consensusOff, base: "a", head: "c", note: "was adopted before the last change: covered by TestY"},
		"core func Question": {verdict: verdictTodo, consensus: "canto", base: "a", head: "c", note: "was todo before the last change: OPEN: accept the new limit?"},
		"core func Harmless": {verdict: verdictTodo, base: "a", head: "c", note: "was adopted before the last change: error string only"},
		"core func OnPath":   {verdict: verdictTodo, base: "a", head: "b", note: "was inactive before the last change: never executed"},
		"core func OffPath":  {verdict: verdictAdopted, consensus: consensusOff, base: "a", head: "b", note: "fixes a panic"},
		"core func LeftPath": {verdict: verdictRefactor, consensus: consensusOff, base: "a", head: "b", note: "same results"},
	}
	if !maps.Equal(rows, want) {
		t.Errorf("got rows %+v, want %+v", rows, want)
	}
	if !strings.Contains(string(text), "- Base: `geth old`\n- Head: `geth new`, `vm v2`") {
		t.Errorf("header does not name the versions:\n%s", text)
	}
}

func TestFormat_ListsCriticalRowsFirstThenGroupsByVerdict(t *testing.T) {
	var findings []analysis.Finding
	var rows []row
	for _, r := range []struct{ key, verdict, consensus string }{
		{"core func A", verdictAdopted, "canto"},
		{"core func B", verdictTodo, consensusLive},
		{"core func C", verdictAdopted, consensusNo},
		{"core func D", verdictAdopted, ""},
		{"core func E", verdictInactive, consensusOff},
		{"core func F", verdictTodo, "canto"},
		{"core func G", verdictAdopted, consensusLive},
		{"core func H", verdictRefactor, consensusOff},
		{"core func I", verdictTodo, consensusOff},
	} {
		findings = append(findings, analysis.Finding{Key: r.key, Base: "a", Head: "b"})
		rows = append(rows, row{verdict: r.verdict, consensus: r.consensus, base: "a", head: "b", note: "checked | twice"})
	}
	text := format([]string{"old"}, []string{"new"}, findings, rows)

	var got []string
	for _, line := range strings.Split(text, "\n") {
		cells, ok := tableCells(line)
		switch {
		case strings.HasPrefix(line, "## "):
			got = append(got, line)
		case ok && !slices.Equal(cells, columns) && !isSeparator(cells):
			got = append(got, strings.Join(cells[:3], " "))
		}
	}
	want := []string{
		"## consensus critical (4)",
		"todo live `core func B`", "adopted live `core func G`",
		"todo canto `core func F`", "adopted canto `core func A`",
		"## todo (1)", "todo - `core func I`",
		"## adopted (2)", "adopted no `core func C`", "adopted  `core func D`",
		"## refactor (1)", "refactor - `core func H`",
		"## inactive (1)", "inactive - `core func E`",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got rows\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	parsed, err := parse(text, "review.md", []string{"canto"})
	if err != nil {
		t.Fatalf("formatted review does not parse: %v\n%s", err, text)
	}
	for i, f := range findings {
		if parsed[f.Key] != rows[i] {
			t.Errorf("%s: read back %+v, want %+v", f.Key, parsed[f.Key], rows[i])
		}
	}
}
