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
	_ "embed"
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/tabwriter"
	"text/template"
)

// A work directory written by -prepare holds brief.md and, per group, the
// findings and the rows that the reviewer of the group fills in. preparedFile
// keeps the rows as prepared, so that -apply records only what reviewers
// changed, and only while the review file still holds what was prepared.
const (
	briefFile      = "brief.md"
	preparedFile   = ".prepared.md"
	findingsSuffix = ".findings.txt"
	rowsSuffix     = ".rows.md"
)

// maxGroupBytes limits the findings a single reviewer reads; larger groups are
// split into parts.
const maxGroupBytes = 128 << 10

//go:embed brief.md
var briefTemplate string

var brief = template.Must(template.New(briefFile).Parse(briefTemplate))

// briefData fills in the brief. Modules and BaseModules map the dependency
// modules of the head and the base to their source directories.
type briefData struct {
	Verify      bool
	Dir         string
	Head        string
	Base        string
	HeadGeth    string
	BaseGeth    string
	HeadTree    string
	Modules     map[string]string
	BaseModules map[string]string
}

// newWorkDir refuses a directory with files, so that -apply never reads rows
// of an earlier round.
func newWorkDir(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	if len(files) > 0 {
		return "", fmt.Errorf("%s is not empty, prepare into a new directory", dir)
	}
	return dir, nil
}

func clearDir(dir string) {
	files, _ := os.ReadDir(dir)
	for _, f := range files {
		os.RemoveAll(filepath.Join(dir, f.Name()))
	}
}

// selectFindings returns the findings matching only that need a reviewer, or
// with verify those that a reviewer classified as consensus critical or left
// as an open question. Module rows are always open, so they are not verified.
func selectFindings(findings []finding, entries map[string]entry, verify bool, only *regexp.Regexp) []finding {
	var out []finding
	for _, f := range findings {
		if only != nil && !only.MatchString(f.key) {
			continue
		}
		e, ok := entries[f.key]
		if verify {
			if !needsReview(f, e, ok) && !isModule(f.key) && (isCritical(e.consensus) || e.verdict == verdictTodo) {
				out = append(out, f)
			}
		} else if needsReview(f, e, ok) || (e.verdict == verdictAdopted && f.consensus && e.consensus == "") {
			out = append(out, f)
		}
	}
	return out
}

type group struct {
	name     string
	findings []finding
	details  []string
}

// groupFindings sorts fs into groups of related findings by category, ordered
// by source file within a group. It splits a group whose details exceed
// maxGroupBytes into parts of similar size, named <category>-1, -2, and so on.
func groupFindings(fs []finding, details func(finding) string) []group {
	byCategory := map[string][]finding{}
	for _, f := range fs {
		c := category(f)
		byCategory[c] = append(byCategory[c], f)
	}
	var groups []group
	for _, c := range slices.Sorted(maps.Keys(byCategory)) {
		members := byCategory[c]
		slices.SortStableFunc(members, func(a, b finding) int { return strings.Compare(sourceFile(a), sourceFile(b)) })
		texts := make([]string, len(members))
		size := 0
		for i, f := range members {
			texts[i] = details(f)
			size += len(texts[i])
		}
		parts := (size + maxGroupBytes - 1) / maxGroupBytes
		if parts <= 1 {
			groups = append(groups, group{c, members, texts})
			continue
		}
		target := (size + parts - 1) / parts
		start, sum, part := 0, 0, 1
		for i := range members {
			sum += len(texts[i])
			if sum >= target && i+1 < len(members) {
				groups = append(groups, group{fmt.Sprintf("%s-%d", c, part), members[start : i+1], texts[start : i+1]})
				start, sum, part = i+1, 0, part+1
			}
		}
		groups = append(groups, group{fmt.Sprintf("%s-%d", c, part), members[start:], texts[start:]})
	}
	return groups
}

// category assigns a finding to the reviewers of its area, by package and, for
// core/vm, by file. brief.md has a hint for each category.
func category(f finding) string {
	pkg, _, _ := strings.Cut(f.key, " ")
	in := func(roots ...string) bool {
		return slices.ContainsFunc(roots, func(r string) bool { return pkg == r || strings.HasPrefix(pkg, r+"/") })
	}
	switch {
	case isModule(f.key):
		return "modules"
	case pkg == "core/vm":
		switch file := path.Base(sourceFile(f)); {
		case slices.Contains([]string{"instructions.go", "eips.go", "jump_table.go", "opcodes.go"}, file):
			return "vm-instructions"
		case slices.Contains([]string{"stack.go", "memory.go", "memory_table.go"}, file):
			return "vm-stack-memory"
		case slices.Contains([]string{"gas.go", "gas_table.go", "operations_acl.go", "operations_verkle.go"}, file):
			return "vm-gas"
		case strings.HasPrefix(file, "contracts") || file == "precompile_cache.go":
			return "vm-precompiles"
		}
		return "vm-evm"
	case in("crypto", "consensus"):
		return "vm-precompiles"
	case pkg == "core" || in("core/types", "core/stateless"):
		return "core"
	case in("params", "core/tracing"):
		return "params-tracing"
	case in("core/state", "core/rawdb", "core/filtermaps", "trie", "triedb", "ethdb"):
		return "state-storage"
	case in("rpc", "node", "ethclient", "accounts", "signer", "common/hexutil", "internal/web3ext", "internal/ethapi"):
		return "rpc-api"
	case in("eth/tracers", "log", "metrics", "internal/telemetry"):
		return "tracers-log"
	}
	return "misc"
}

func sourceFile(f finding) string {
	file, _, _ := strings.Cut(f.location(), ":")
	return file
}

func prepare(b briefData, selected []finding, entries map[string]entry, details func(finding) string) (bool, error) {
	groups := groupFindings(selected, details)
	if len(groups) == 0 {
		fmt.Println("\nnothing to prepare: no finding needs a reviewer")
		return false, nil
	}
	var text strings.Builder
	if err := brief.Execute(&text, b); err != nil {
		return false, err
	}
	if err := os.WriteFile(filepath.Join(b.Dir, briefFile), []byte(text.String()), 0o644); err != nil {
		return false, err
	}
	for _, g := range groups {
		if err := os.WriteFile(filepath.Join(b.Dir, g.name+findingsSuffix), []byte(strings.Join(g.details, "")), 0o644); err != nil {
			return false, err
		}
		if err := os.WriteFile(filepath.Join(b.Dir, g.name+rowsSuffix), []byte(formatRows("the group "+g.name, g.findings, entries)), 0o644); err != nil {
			return false, err
		}
	}
	if err := os.WriteFile(filepath.Join(b.Dir, preparedFile), []byte(formatRows("all groups, as prepared", selected, entries)), 0o644); err != nil {
		return false, err
	}

	task := "Review"
	if b.Verify {
		task = "Verify"
	}
	fmt.Printf("\nprepared %d findings in %d groups in %s:\n\n", len(selected), len(groups), b.Dir)
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, g := range groups {
		size := 0
		for _, d := range g.details {
			size += len(d)
		}
		fmt.Fprintf(tw, "  %s\t%4d findings\t%4d KB\n", g.name, len(g.findings), size>>10)
	}
	tw.Flush()
	fmt.Printf("\nGive each group to its own reviewer, all at once, with this prompt:\n\n")
	fmt.Printf("  %s the group <group> of a go-ethereum upgrade of Sonic. Read %s and follow it.\n\n",
		task, filepath.Join(b.Dir, briefFile))
	fmt.Printf("When all have finished, run geth_compatibility with -apply %s.\n", b.Dir)
	return true, nil
}

const rowsHeader = "# Rows of %s\n\n" +
	"Fill in the Verdict, Consensus and Note cells of every row as `brief.md`\n" +
	"describes. Do not change the other cells, and do not add or remove rows.\n\n"

func formatRows(name string, fs []finding, entries map[string]entry) string {
	var b strings.Builder
	fmt.Fprintf(&b, rowsHeader, name)
	rows := make([][]string, len(fs))
	for i, f := range fs {
		rows[i] = tableRow(f, currentEntry(f, entries))
	}
	writeTable(&b, rows)
	return b.String()
}

// applyRows records in entries the rows that reviewers changed under dir. It
// skips a file with an invalid row as a whole, but a row whose entry changed
// since -prepare on its own, as the other rows of its file still apply.
func applyRows(dir string, findings []finding, entries map[string]entry) (int, []string, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*"+rowsSuffix))
	if err != nil {
		return 0, nil, err
	}
	if len(paths) == 0 {
		return 0, nil, fmt.Errorf("%s has no %s files written by -prepare", dir, rowsSuffix)
	}
	text, err := os.ReadFile(filepath.Join(dir, preparedFile))
	if err != nil {
		return 0, nil, err
	}
	prepared, err := parseReview(string(text), preparedFile)
	if err != nil {
		return 0, nil, err
	}
	byKey := map[string]finding{}
	for _, f := range findings {
		byKey[f.key] = f
	}
	from := map[string]string{}
	applied := 0
	var problems []string
	for _, p := range paths {
		name := filepath.Base(p)
		text, err := os.ReadFile(p)
		if err != nil {
			return 0, nil, err
		}
		rows, err := parseReview(string(text), name)
		if err != nil {
			problems = append(problems, err.Error())
			continue
		}
		keys := slices.Sorted(maps.Keys(rows))
		var bad []string
		for _, key := range keys {
			f, ok := byKey[key]
			_, wasPrepared := prepared[key]
			switch {
			case !ok:
				bad = append(bad, fmt.Sprintf("%s: %s is not a finding", name, key))
			case !wasPrepared:
				bad = append(bad, fmt.Sprintf("%s: %s was not prepared in this directory", name, key))
			case !rows[key].matches(f):
				bad = append(bad, fmt.Sprintf("%s: %s changed since -prepare", name, key))
			case from[key] != "":
				bad = append(bad, fmt.Sprintf("%s: %s is in %s too", name, key, from[key]))
			}
		}
		if len(bad) > 0 {
			problems = append(problems, bad...)
			continue
		}
		for _, key := range keys {
			from[key] = name
			e, current := rows[key], currentEntry(byKey[key], entries)
			switch {
			case e == prepared[key] || e == current:
			case current != prepared[key]:
				problems = append(problems, fmt.Sprintf("%s: %s changed in the review file since -prepare, make both rows equal to settle it", name, key))
			default:
				entries[key] = e
				applied++
			}
		}
	}
	return applied, problems, nil
}
