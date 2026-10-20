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
	"maps"
	"slices"
	"strings"
	"testing"
)

const tableHeader = "| Verdict | Consensus | Symbol | Base | Head | Note |\n|---|---|---|---|---|---|\n"

func TestParseReview_RejectsMalformedRows(t *testing.T) {
	tests := map[string]string{
		"unknown verdict":       "| fine | - | `core func A` | aaa | bbb | looks fine |",
		"missing note":          "| refactor | - | `core func A` | aaa | bbb | |",
		"blank note":            "| adopted | - | `core func A` | aaa | bbb |    |",
		"uppercase consensus":   "| adopted | Canto | `core func A` | aaa | bbb | looks fine |",
		"verdict as consensus":  "| adopted | todo | `core func A` | aaa | bbb | looks fine |",
		"too few cells":         "| todo | - | `core func A` | aaa |",
		"too many cells":        "| todo | - | `core func A` | aaa | bbb | note | extra |",
		"row without consensus": "| todo | `core func A` | aaa | bbb | |",
		"duplicate symbol":      "| todo | - | `core func A` | aaa | bbb | |\n| todo | live | `core func A` | aaa | bbb | |",
		"empty verdict cell":    "| | - | `core func A` | aaa | bbb | |",
		"blank line in a table": "| todo | - | `core func A` | aaa | bbb | |\n\n| todo | - | `core func B` | aaa | bbb | |",
		"renamed column":        "\n| Verdict | Consensus | Symbol | Base | Head | Notes |\n|---|---|---|---|---|---|\n| todo | - | `core func B` | aaa | bbb | |",
		"other table":           "\n| Verdict | Meaning |\n|---|---|\n| `todo` | not reviewed |",
	}
	for name, rows := range tests {
		if _, err := parseReview(tableHeader+rows, "review.md"); err == nil {
			t.Errorf("%s: accepted %q", name, rows)
		}
	}
}

func TestParseReview_ReadsTheRowsOfEveryTable(t *testing.T) {
	text := "# Review\n\nSome prose.\n\n- a list\n\n## todo (1)\n\n" +
		tableHeader +
		"| todo     |       | `core func A` | aaa | bbb |  |\n" +
		"\n## other (2)\n\n" +
		tableHeader +
		"| inactive | -     | `core func B` | -   | ccc | Amsterdam only, see a \\| b |\n" +
		"| adopted  | canto | `core func C` | ccc | ddd | Canto only |\n"
	entries, err := parseReview(text, "review.md")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]entry{
		"core func A": {verdict: verdictTodo, base: "aaa", head: "bbb"},
		"core func B": {verdict: verdictInactive, base: "-", head: "ccc", note: "Amsterdam only, see a | b"},
		"core func C": {verdict: verdictAdopted, consensus: "canto", base: "ccc", head: "ddd", note: "Canto only"},
	}
	if !maps.Equal(entries, want) {
		t.Errorf("got entries %+v, want %+v", entries, want)
	}
}

func TestFormatReview_KeepsMatchingVerdictsAndReopensChangedOnes(t *testing.T) {
	a, b, c := &symbol{text: "a"}, &symbol{text: "b"}, &symbol{text: "c"}
	findings := []finding{
		{key: "core func Kept", status: statusChanged, base: a, head: b},
		{key: "core func Reopened", status: statusChanged, base: a, head: c},
		{key: "core func New", status: statusAdded, head: b},
		{key: "core func Pending", status: statusChanged, base: a, head: c},
		{key: "core func Asked", status: statusChanged, base: a, head: c},
	}
	prev := map[string]entry{
		"core func Kept":     {verdict: verdictRefactor, base: a.hash(), head: b.hash(), note: "renamed locals"},
		"core func Reopened": {verdict: verdictAdopted, consensus: consensusLive, base: a.hash(), head: b.hash(), note: "covered by TestX"},
		"core func Gone":     {verdict: verdictInactive, base: a.hash(), head: b.hash(), note: "Amsterdam only"},
		"core func Pending":  {verdict: verdictTodo, base: a.hash(), head: b.hash(), note: "check with TestY"},
		"core func Asked":    {verdict: verdictTodo, base: a.hash(), head: b.hash(), note: "OPEN: accept X? Found: Y"},
	}
	text := formatReview("geth old", "geth new", findings, prev)
	entries, err := parseReview(text, "review.md")
	if err != nil {
		t.Fatalf("formatted review does not parse: %v\n%s", err, text)
	}
	if len(entries) != 5 {
		t.Errorf("want one row per finding, got %d:\n%s", len(entries), text)
	}
	if e := entries["core func Kept"]; e.verdict != verdictRefactor || !e.matches(findings[0]) {
		t.Errorf("matching verdict not kept: %+v", e)
	}
	if e := entries["core func Reopened"]; e.verdict != verdictTodo || e.consensus != consensusLive || !strings.Contains(e.note, "covered by TestX") || !e.matches(findings[1]) {
		t.Errorf("changed finding not reopened with its old note and consensus classification: %+v", e)
	}
	if e := entries["core func Pending"]; e.verdict != verdictTodo || e.note != "check with TestY" {
		t.Errorf("note of a pending finding not kept: %+v", e)
	}
	if e := entries["core func Asked"]; e.note != "was todo before the last change: OPEN: accept X? Found: Y" || e.reviewed() {
		t.Errorf("changed open question not reopened for review: %+v", e)
	}
	if e := entries["core func New"]; e.verdict != verdictTodo || e.base != "-" {
		t.Errorf("new finding not added as todo: %+v", e)
	}
	if !strings.Contains(text, "- Base: `geth old`") || !strings.Contains(text, "- Head: `geth new`") {
		t.Errorf("header does not name the versions:\n%s", text)
	}
}

func TestFormatReview_GroupsRowsByVerdictWithTodoFirst(t *testing.T) {
	a, b := &symbol{text: "a"}, &symbol{text: "b"}
	var findings []finding
	prev := map[string]entry{}
	for _, row := range []struct{ key, verdict string }{
		{"core func A", verdictInactive},
		{"core func B", verdictTodo},
		{"core func C", verdictRefactor},
		{"core func D", verdictAdopted},
		{"core func E", verdictTodo},
	} {
		findings = append(findings, finding{key: row.key, status: statusChanged, base: a, head: b})
		prev[row.key] = entry{verdict: row.verdict, base: a.hash(), head: b.hash(), note: "checked | twice"}
	}
	text := formatReview("old", "new", findings, prev)

	var got []string
	for _, line := range strings.Split(text, "\n") {
		cells, ok := tableCells(line)
		switch {
		case strings.HasPrefix(line, "## "):
			got = append(got, line)
		case ok && !slices.Equal(cells, reviewColumns) && !isSeparator(cells):
			got = append(got, cells[0]+" "+cells[2])
		}
	}
	want := []string{
		"## todo (2)", "todo `core func B`", "todo `core func E`",
		"## adopted (1)", "adopted `core func D`",
		"## refactor (1)", "refactor `core func C`",
		"## inactive (1)", "inactive `core func A`",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got rows\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	entries, err := parseReview(text, "review.md")
	if err != nil {
		t.Fatalf("formatted review does not parse: %v\n%s", err, text)
	}
	for key, e := range entries {
		if want := prev[key]; e != want {
			t.Errorf("%s: read back %+v, want %+v", key, e, want)
		}
	}
}

func TestFormatReview_ListsConsensusCriticalRowsFirst(t *testing.T) {
	a, b := &symbol{text: "a"}, &symbol{text: "b"}
	var findings []finding
	prev := map[string]entry{}
	for _, row := range []struct {
		key, verdict, consensus string
		onPath                  bool
	}{
		{"core func A", verdictAdopted, "canto", true},
		{"core func B", verdictTodo, consensusLive, true},
		{"core func C", verdictAdopted, consensusNo, true},
		{"core func D", verdictAdopted, "", true},
		{"core func E", verdictAdopted, "", false},
		{"core func F", verdictTodo, "canto", false},
		{"core func G", verdictAdopted, consensusLive, true},
		{"core func H", verdictAdopted, "dolce", true},
	} {
		findings = append(findings, finding{key: row.key, status: statusChanged, base: a, head: b, consensus: row.onPath})
		prev[row.key] = entry{verdict: row.verdict, consensus: row.consensus, base: a.hash(), head: b.hash(), note: "checked"}
	}
	text := formatReview("old", "new", findings, prev)

	var got []string
	for _, line := range strings.Split(text, "\n") {
		cells, ok := tableCells(line)
		switch {
		case strings.HasPrefix(line, "## "):
			got = append(got, line)
		case ok && !slices.Equal(cells, reviewColumns) && !isSeparator(cells):
			got = append(got, strings.Join(cells[:3], " "))
		}
	}
	want := []string{
		"## consensus critical (5)",
		"todo live `core func B`", "adopted live `core func G`",
		"todo canto `core func F`", "adopted canto `core func A`",
		"adopted dolce `core func H`",
		"## adopted (3)",
		"adopted no `core func C`", "adopted  `core func D`", "adopted - `core func E`",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got rows\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	entries, err := parseReview(text, "review.md")
	if err != nil {
		t.Fatalf("formatted review does not parse: %v\n%s", err, text)
	}
	for key, e := range entries {
		if want := prev[key]; e != want {
			t.Errorf("%s: read back %+v, want %+v", key, e, want)
		}
	}
}

func TestValidConsensus_AcceptsLiveNoAndHardforkNames(t *testing.T) {
	for tag, want := range map[string]bool{
		"live": true, "no": true, "canto": true, "dolce": true,
		"": false, "-": false, "Canto": false, "dolce2": false, "can to": false, "todo": false, "adopted": false,
	} {
		if got := validConsensus(tag); got != want {
			t.Errorf("validConsensus(%q) = %v, want %v", tag, got, want)
		}
	}
}

func TestDescribeCritical_CountsLiveFirstThenHardforksByName(t *testing.T) {
	got := describeCritical(map[string]int{"dolce": 2, "live": 7, "canto": 62})
	if want := "71 critical (7 live, 62 canto, 2 dolce)"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := describeCritical(map[string]int{}); got != "0 critical" {
		t.Errorf("got %q for no critical rows", got)
	}
}

func TestGethVersion_PrefersTheReplaceDirective(t *testing.T) {
	gomod := `module github.com/0xsoniclabs/sonic

go 1.26.0

require github.com/ethereum/go-ethereum v1.17.1

replace github.com/ethereum/go-ethereum => github.com/0xsoniclabs/go-ethereum v0.0.0-20261013140340-66cf338cad69
`
	got, err := gethVersion(gomod)
	if err != nil {
		t.Fatal(err)
	}
	if want := "github.com/0xsoniclabs/go-ethereum v0.0.0-20261013140340-66cf338cad69"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	got, err = gethVersion(strings.Split(gomod, "replace")[0])
	if err != nil || got != "github.com/ethereum/go-ethereum v1.17.1" {
		t.Errorf("without a replace directive got %q, %v", got, err)
	}

	stale := strings.Replace(strings.Replace(gomod, "v1.17.1", "v1.17.8", 1), "go-ethereum =>", "go-ethereum v1.17.1 =>", 1)
	got, err = gethVersion(stale)
	if err != nil || got != "github.com/ethereum/go-ethereum v1.17.8" {
		t.Errorf("with a replace directive for another version got %q, %v", got, err)
	}

	exact := gomod + "replace github.com/ethereum/go-ethereum v1.17.1 => github.com/0xsoniclabs/go-ethereum v1.17.1-exact\n"
	got, err = gethVersion(exact)
	if err != nil || got != "github.com/0xsoniclabs/go-ethereum v1.17.1-exact" {
		t.Errorf("with an exact and a wildcard replace directive got %q, %v", got, err)
	}
}
