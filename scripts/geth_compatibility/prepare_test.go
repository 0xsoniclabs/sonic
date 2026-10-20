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
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func keysOf(fs []finding) []string {
	keys := make([]string, len(fs))
	for i, f := range fs {
		keys[i] = f.key
	}
	return keys
}

func TestSelectFindings_TakesUnreviewedRowsOrThoseToVerify(t *testing.T) {
	a, b, c := &symbol{text: "a"}, &symbol{text: "b"}, &symbol{text: "c"}
	var findings []finding
	for _, key := range []string{"core func New", "core func Changed", "core func Unreviewed", "core func Open", "core func Canto", "core func No", "core func Refactor", "module x"} {
		findings = append(findings, finding{key: key, status: statusChanged, base: a, head: b})
	}
	findings = append(findings,
		finding{key: "core func Unclassified", status: statusChanged, base: a, head: b, consensus: true},
		finding{key: "core func OffPath", status: statusChanged, base: a, head: b},
	)
	entries := map[string]entry{
		"module x":               {verdict: verdictTodo, base: a.hash(), head: b.hash(), note: "OPEN: accept x v2? Found: y"},
		"core func Changed":      {verdict: verdictAdopted, consensus: "canto", base: a.hash(), head: c.hash(), note: "Canto only"},
		"core func Unreviewed":   {verdict: verdictTodo, base: a.hash(), head: b.hash(), note: "was adopted before the last change: x"},
		"core func Open":         {verdict: verdictTodo, base: a.hash(), head: b.hash(), note: "OPEN: accept? Found: x"},
		"core func Canto":        {verdict: verdictAdopted, consensus: "canto", base: a.hash(), head: b.hash(), note: "Canto only"},
		"core func No":           {verdict: verdictAdopted, consensus: consensusNo, base: a.hash(), head: b.hash(), note: "tracing only"},
		"core func Refactor":     {verdict: verdictRefactor, base: a.hash(), head: b.hash(), note: "renamed"},
		"core func Unclassified": {verdict: verdictAdopted, base: a.hash(), head: b.hash(), note: "bug fix"},
		"core func OffPath":      {verdict: verdictAdopted, base: a.hash(), head: b.hash(), note: "bug fix"},
	}
	tests := []struct {
		verify bool
		only   *regexp.Regexp
		want   []string
	}{
		{false, nil, []string{"core func New", "core func Changed", "core func Unreviewed", "core func Unclassified"}},
		{false, regexp.MustCompile("New$"), []string{"core func New"}},
		{true, nil, []string{"core func Open", "core func Canto"}},
	}
	for _, test := range tests {
		if got := keysOf(selectFindings(findings, entries, test.verify, test.only)); !slices.Equal(got, test.want) {
			t.Errorf("verify %v, only %v: got %v, want %v", test.verify, test.only, got, test.want)
		}
	}
}

func TestCategory_AssignsByPackageAndVmFile(t *testing.T) {
	tests := map[string]string{
		"core/vm func opAdd|core/vm/instructions.go:10":                 "vm-instructions",
		"core/vm func gasSStore|core/vm/gas_table.go:10":                "vm-gas",
		"core/vm type Stack|core/vm/stack.go:10":                        "vm-stack-memory",
		"core/vm func RunPrecompiledContract|core/vm/contracts.go:10":   "vm-precompiles",
		"core/vm func EVM.Call|core/vm/evm.go:10":                       "vm-evm",
		"crypto func Keccak256|crypto/crypto.go:10":                     "vm-precompiles",
		"core func IntrinsicGas|core/state_transition.go:10":            "core",
		"core/types/bal type BlockAccessList|core/types/bal/bal.go:10":  "core",
		"core/state func New|core/state/statedb.go:10":                  "state-storage",
		"core/stateless func Witness|core/stateless/witness.go:10":      "core",
		"trie/bintrie func New|trie/bintrie/trie.go:10":                 "state-storage",
		"params func ChainConfig.Rules|params/config.go:10":             "params-tracing",
		"rpc func Server.ServeHTTP|rpc/http.go:10":                      "rpc-api",
		"eth/tracers/native func newCallTracer|eth/tracers/native/c:10": "tracers-log",
		"logger func New|logger/x.go:10":                                "misc",
		"p2p/discover func UDPv5.Lookup|p2p/discover/v5.go:10":          "misc",
		"module github.com/holiman/uint256|":                            "modules",
	}
	for spec, want := range tests {
		key, loc, _ := strings.Cut(spec, "|")
		f := finding{key: key, status: statusChanged, base: &symbol{text: "a", loc: loc}, head: &symbol{text: "b", loc: loc}}
		if got := category(f); got != want {
			t.Errorf("%s: got %s, want %s", key, got, want)
		}
	}
}

func TestGroupFindings_SplitsLargeCategoriesBySourceFile(t *testing.T) {
	var fs []finding
	for i, file := range []string{"b.go", "a.go", "b.go", "a.go", "c.go"} {
		loc := "core/" + file + ":1"
		fs = append(fs, finding{key: "core func F" + string(rune('A'+i)), status: statusAdded, head: &symbol{text: "x", loc: loc}})
	}
	fs = append(fs, finding{key: "rlp func Encode", status: statusAdded, head: &symbol{text: "x", loc: "rlp/encode.go:1"}})
	details := func(f finding) string {
		if strings.HasPrefix(f.key, "rlp ") {
			return "small"
		}
		return strings.Repeat("x", maxGroupBytes/2)
	}

	var got []string
	for _, g := range groupFindings(fs, details) {
		got = append(got, g.name+": "+strings.Join(keysOf(g.findings), ", "))
	}
	want := []string{
		"core-1: core func FB, core func FD",
		"core-2: core func FA, core func FC",
		"core-3: core func FE",
		"misc: rlp func Encode",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got groups\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestPrepare_WritesRowsAndApplyRecordsTheChangedOnes(t *testing.T) {
	a, b := &symbol{text: "a", loc: "core/x.go:1"}, &symbol{text: "b", loc: "core/x.go:1"}
	findings := []finding{
		{key: "core func A", status: statusChanged, base: a, head: b},
		{key: "rpc func B", status: statusAdded, head: b},
	}
	entries := map[string]entry{
		"core func A": {verdict: verdictAdopted, base: a.hash(), head: a.hash(), note: "covered by TestA"},
	}
	dir, err := newWorkDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	details := func(f finding) string { return "=== " + f.key + "\n" }
	if wrote, err := prepare(briefData{Dir: dir}, findings, entries, details); err != nil || !wrote {
		t.Fatalf("prepare wrote %v, error %v", wrote, err)
	}

	for _, name := range []string{briefFile, preparedFile, "core.findings.txt", "core.rows.md", "rpc-api.findings.txt", "rpc-api.rows.md"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("missing %s: %v", name, err)
		}
	}
	if text, _ := os.ReadFile(filepath.Join(dir, "core.findings.txt")); string(text) != "=== core func A\n" {
		t.Errorf("core findings: got %q", text)
	}
	rows, err := os.ReadFile(filepath.Join(dir, "core.rows.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rows), "was adopted before the last change: covered by TestA") {
		t.Errorf("rows do not carry the earlier note:\n%s", rows)
	}

	applied, problems, err := applyRows(dir, findings, entries)
	if err != nil || applied != 0 || len(problems) > 0 {
		t.Fatalf("applying the rows as prepared: %d applied, problems %v, error %v", applied, problems, err)
	}

	filled := tableHeader + "| refactor | - | `core func A` | " + a.hash() + " | " + b.hash() + " | renamed locals |\n"
	if err := os.WriteFile(filepath.Join(dir, "core.rows.md"), []byte(filled), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, want := range []int{1, 0} {
		applied, problems, err := applyRows(dir, findings, entries)
		if err != nil || applied != want || len(problems) > 0 {
			t.Fatalf("applying a filled in row: %d applied, want %d, problems %v, error %v", applied, want, problems, err)
		}
	}
	if e := entries["core func A"]; e.verdict != verdictRefactor || !e.matches(findings[0]) {
		t.Errorf("filled in row not recorded: %+v", e)
	}
	if _, ok := entries["rpc func B"]; ok {
		t.Errorf("unchanged row recorded: %+v", entries["rpc func B"])
	}
	if _, err := newWorkDir(dir); err == nil {
		t.Error("a second prepare into the same directory was accepted")
	}
}

func TestApplyRows_KeepsRowsChangedInTheReviewSincePrepare(t *testing.T) {
	a, b := &symbol{text: "a", loc: "core/x.go:1"}, &symbol{text: "b", loc: "core/x.go:1"}
	findings := []finding{
		{key: "core func A", status: statusChanged, base: a, head: b},
		{key: "core func B", status: statusChanged, base: a, head: b},
	}
	entries := map[string]entry{}
	dir, err := newWorkDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := prepare(briefData{Dir: dir}, findings, entries, func(finding) string { return "" }); err != nil {
		t.Fatal(err)
	}
	filled := tableHeader +
		"| refactor | - | `core func A` | " + a.hash() + " | " + b.hash() + " | renamed locals |\n" +
		"| inactive | - | `core func B` | " + a.hash() + " | " + b.hash() + " | Amsterdam only |\n"
	if err := os.WriteFile(filepath.Join(dir, "core.rows.md"), []byte(filled), 0o644); err != nil {
		t.Fatal(err)
	}
	verified := entry{verdict: verdictAdopted, consensus: consensusLive, base: a.hash(), head: b.hash(), note: "verified"}
	entries["core func A"] = verified

	applied, problems, err := applyRows(dir, findings, entries)
	if err != nil {
		t.Fatal(err)
	}
	if applied != 1 || entries["core func B"].verdict != verdictInactive {
		t.Errorf("unaffected row not applied: %d applied, entries %+v", applied, entries)
	}
	if entries["core func A"] != verified {
		t.Errorf("row changed in the review overwritten: %+v", entries["core func A"])
	}
	if len(problems) != 1 || !strings.HasPrefix(problems[0], "core.rows.md: core func A changed in the review file since -prepare") {
		t.Errorf("got problems %q", problems)
	}
}

func TestApplyRows_SkipsFilesWithInvalidRows(t *testing.T) {
	a, b := &symbol{text: "a"}, &symbol{text: "b"}
	findings := []finding{
		{key: "core func A", status: statusChanged, base: a, head: b},
		{key: "core func B", status: statusChanged, base: a, head: b},
		{key: "core func C", status: statusChanged, base: a, head: b},
		{key: "core func E", status: statusChanged, base: a, head: b},
	}
	row := func(key, verdict, base, head string) string {
		return "| " + verdict + " | - | `" + key + "` | " + base + " | " + head + " | checked |\n"
	}
	dir := t.TempDir()
	entries := map[string]entry{}
	if err := os.WriteFile(filepath.Join(dir, preparedFile), []byte(formatRows("x", findings[:3], entries)), 0o644); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"1.rows.md": row("core func A", verdictRefactor, a.hash(), b.hash()) + row("core func B", verdictInactive, a.hash(), b.hash()),
		"2.rows.md": row("core func C", verdictRefactor, a.hash(), a.hash()),
		"3.rows.md": row("core func D", verdictRefactor, a.hash(), b.hash()),
		"4.rows.md": row("core func A", verdictAdopted, a.hash(), b.hash()) + row("core func C", verdictAdopted, a.hash(), b.hash()),
		"5.rows.md": row("core func C", "fine", a.hash(), b.hash()),
		"6.rows.md": row("core func E", verdictRefactor, a.hash(), b.hash()),
	}
	for name, rows := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(tableHeader+rows), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	applied, problems, err := applyRows(dir, findings, entries)
	if err != nil {
		t.Fatal(err)
	}
	if applied != 2 || entries["core func A"].verdict != verdictRefactor || entries["core func B"].verdict != verdictInactive {
		t.Errorf("valid file not applied: %d rows, entries %+v", applied, entries)
	}
	if _, ok := entries["core func C"]; ok {
		t.Errorf("row of a skipped file applied: %+v", entries["core func C"])
	}
	want := []string{
		"2.rows.md: core func C changed since -prepare",
		"3.rows.md: core func D is not a finding",
		"4.rows.md: core func A is in 1.rows.md too",
		`5.rows.md:3: unknown verdict "fine"`,
		"6.rows.md: core func E was not prepared in this directory",
	}
	if !slices.Equal(problems, want) {
		t.Errorf("got problems\n%s\nwant\n%s", strings.Join(problems, "\n"), strings.Join(want, "\n"))
	}

	if _, _, err := applyRows(t.TempDir(), findings, entries); err == nil {
		t.Error("a directory without rows files was accepted")
	}
}

func TestBrief_NamesTheSourcesForBothTasks(t *testing.T) {
	data := briefData{
		Dir:         "/work",
		Head:        "simon/amsterdam",
		Base:        "3140eba96c (merge base with main)",
		HeadGeth:    "geth new",
		BaseGeth:    "geth old",
		HeadTree:    "/work/head",
		Modules:     map[string]string{gethModule: "/mod/geth@new", "github.com/0xsoniclabs/tosca": "/mod/tosca", "github.com/0xsoniclabs/carmen/go": "/mod/carmen"},
		BaseModules: map[string]string{gethModule: "/mod/geth@old"},
	}
	for _, verify := range []bool{false, true} {
		data.Verify = verify
		var b strings.Builder
		if err := brief.Execute(&b, data); err != nil {
			t.Fatal(err)
		}
		text := b.String()
		for _, want := range []string{"`/work`", "`/work/head`", "`/mod/geth@old`", "`/mod/geth@new`", "`/mod/tosca`", "`/mod/carmen`", "`geth old`", "`simon/amsterdam`", "### vm-evm"} {
			if !strings.Contains(text, want) {
				t.Errorf("verify %v: brief does not contain %s", verify, want)
			}
		}
		if strings.Contains(text, "<no value>") || strings.Contains(text, "{{") {
			t.Errorf("verify %v: brief has unfilled fields", verify)
		}
		if got := strings.Contains(text, "## Your task"); got != verify {
			t.Errorf("verify %v: brief has the verification task: %v", verify, got)
		}
		if !strings.HasPrefix(text, "# go-ethereum update") {
			t.Errorf("verify %v: brief starts with %q", verify, text[:min(len(text), 40)])
		}
	}
}

func TestBrief_HasAHintForEveryCategory(t *testing.T) {
	keys := []string{
		"core/vm func a|core/vm/instructions.go:1", "core/vm func a|core/vm/stack.go:1", "core/vm func a|core/vm/gas.go:1",
		"core/vm func a|core/vm/contracts.go:1", "core/vm func a|core/vm/evm.go:1", "core func a|", "params func a|",
		"trie func a|", "rpc func a|", "log func a|", "rlp func a|", "module x|",
	}
	for _, spec := range keys {
		key, loc, _ := strings.Cut(spec, "|")
		c := category(finding{key: key, head: &symbol{loc: loc}})
		if !strings.Contains(briefTemplate, "\n### "+c+"\n") {
			t.Errorf("brief.md has no hint for the category %s", c)
		}
	}
}
