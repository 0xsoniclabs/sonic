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
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	verdictTodo     = "todo"
	verdictInactive = "inactive"
	verdictRefactor = "refactor"
	verdictAdopted  = "adopted"
)

// verdicts lists the verdicts in the order the review file groups its rows.
var verdicts = []string{verdictTodo, verdictAdopted, verdictRefactor, verdictInactive}

const (
	consensusLive = "live"
	consensusNo   = "no"
	consensusOff  = "-"
)

// validConsensus reports whether tag is a consensus classification: live, no,
// or the lowercase name of the hardfork from which on a change alters results.
func validConsensus(tag string) bool {
	return tag != "" && strings.Trim(tag, "abcdefghijklmnopqrstuvwxyz") == "" && !slices.Contains(verdicts, tag)
}

func isCritical(tag string) bool {
	return tag != "" && tag != consensusNo
}

// compareCritical orders critical classifications: live first, then the
// hardforks by name, which is the order in which Sonic names them.
func compareCritical(a, b string) int {
	switch {
	case a == b:
		return 0
	case a == consensusLive:
		return -1
	case b == consensusLive:
		return 1
	}
	return strings.Compare(a, b)
}

func describeCritical(counts map[string]int) string {
	tags := slices.SortedFunc(maps.Keys(counts), compareCritical)
	total, parts := 0, make([]string, len(tags))
	for i, tag := range tags {
		total += counts[tag]
		parts[i] = fmt.Sprintf("%d %s", counts[tag], tag)
	}
	if total == 0 {
		return "0 critical"
	}
	return fmt.Sprintf("%d critical (%s)", total, strings.Join(parts, ", "))
}

// entry is a row of the review file. It applies to a finding only while the
// hashes of both sides still match, so any further change reopens the review.
type entry struct {
	verdict   string
	consensus string
	base      string
	head      string
	note      string
}

func (e entry) matches(f finding) bool {
	return e.base == f.base.hash() && e.head == f.head.hash()
}

// openPrefix starts the note of a todo row that a reviewer left as a question
// for a human.
const openPrefix = "OPEN:"

// reviewed reports whether a reviewer decided on the row or left a question.
func (e entry) reviewed() bool {
	return e.verdict != verdictTodo || strings.HasPrefix(e.note, openPrefix)
}

func needsReview(f finding, e entry, ok bool) bool {
	return !ok || !e.matches(f) || !e.reviewed()
}

func readReview(path string) (map[string]entry, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]entry{}, nil
	}
	if err != nil {
		return nil, err
	}
	return parseReview(string(b), path)
}

var reviewColumns = []string{"Verdict", "Consensus", "Symbol", "Base", "Head", "Note"}

// parseReview reads the rows of every table in text. Each table must have the
// review columns, so that no edited header or blank line inside a table drops
// rows unnoticed. Lines outside tables are ignored.
func parseReview(text, name string) (map[string]entry, error) {
	entries := map[string]entry{}
	lines := strings.Split(text, "\n")
	inTable := false
	for i := 0; i < len(lines); i++ {
		cells, ok := tableCells(lines[i])
		switch {
		case !ok:
			inTable = false
			continue
		case !inTable:
			var next []string
			if i+1 < len(lines) {
				next, _ = tableCells(lines[i+1])
			}
			if !slices.Equal(cells, reviewColumns) || next == nil || !isSeparator(next) {
				return nil, fmt.Errorf("%s:%d: want a table with the columns %s", name, i+1, strings.Join(reviewColumns, ", "))
			}
			inTable = true
			i++
			continue
		case len(cells) != len(reviewColumns):
			return nil, fmt.Errorf("%s:%d: want the cells %s", name, i+1, strings.Join(reviewColumns, ", "))
		}
		key := strings.Trim(cells[2], "`")
		e := entry{verdict: cells[0], consensus: cells[1], base: cells[3], head: cells[4], note: cells[5]}
		if e.consensus == consensusOff {
			e.consensus = "" // recomputed on every write
		}
		switch {
		case !slices.Contains(verdicts, e.verdict):
			return nil, fmt.Errorf("%s:%d: unknown verdict %q", name, i+1, e.verdict)
		case e.verdict != verdictTodo && e.note == "":
			return nil, fmt.Errorf("%s:%d: verdict %q needs a note", name, i+1, e.verdict)
		case e.consensus != "" && !validConsensus(e.consensus):
			return nil, fmt.Errorf("%s:%d: unknown consensus classification %q", name, i+1, e.consensus)
		}
		if _, dup := entries[key]; dup {
			return nil, fmt.Errorf("%s:%d: duplicate row for %s", name, i+1, key)
		}
		entries[key] = e
	}
	return entries, nil
}

func tableCells(line string) ([]string, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") {
		return nil, false
	}
	line = strings.TrimSuffix(strings.TrimPrefix(line, "|"), "|")
	cells := strings.Split(strings.ReplaceAll(line, `\|`, "\x00"), "|")
	for i, c := range cells {
		cells[i] = strings.TrimSpace(strings.ReplaceAll(c, "\x00", "|"))
	}
	return cells, true
}

func isSeparator(cells []string) bool {
	for _, c := range cells {
		if strings.Trim(c, "-:") != "" {
			return false
		}
	}
	return true
}

const reviewHeader = "# go-ethereum compatibility review\n\n" +
	"- Base: `%s`\n- Head: `%s`\n\n" +
	"Each row is a go-ethereum difference reachable from Sonic, see\n" +
	"`scripts/geth_compatibility/README.md`. Record a verdict and a note in its row;\n" +
	"`-write` and `-apply` regroup the rows. Verdicts:\n\n" +
	"- `todo`: not reviewed yet, or a question for a human if its note starts with `OPEN:`\n" +
	"- `adopted`: behavior changes and Sonic accepts it; the note says why and what covers it\n" +
	"- `refactor`: no change in behavior\n" +
	"- `inactive`: only affects forks or features Sonic does not enable\n\n" +
	"The Consensus column says whether the change can alter results all nodes\n" +
	"have to agree on. It is `-` off the consensus path, and empty on it until a\n" +
	"reviewer records one of these; `adopted` rows on the path need one:\n\n" +
	"- `live`: on hardforks that are already active\n" +
	"- the name of a hardfork, such as `canto`: only once that hardfork is active\n" +
	"- `no`: on the consensus path, but it cannot alter results\n\n" +
	"Rows classified `live` or with a hardfork name are listed first, whatever\n" +
	"their verdict: `live`, then the hardforks by name.\n"

// currentEntry is the row of f: its row in prev if that still matches,
// otherwise a todo row that keeps the note and the consensus classification of
// prev for the next reviewer.
func currentEntry(f finding, prev map[string]entry) entry {
	e := entry{verdict: verdictTodo, base: f.base.hash(), head: f.head.hash()}
	p, ok := prev[f.key]
	if !ok {
		return e
	}
	switch {
	case p.matches(f):
		e.verdict, e.note = p.verdict, p.note
	case p.reviewed():
		e.note = fmt.Sprintf("was %s before the last change: %s", p.verdict, p.note)
	default:
		e.note = p.note
	}
	e.consensus = p.consensus
	return e
}

func tableRow(f finding, e entry) []string {
	tag := e.consensus
	if tag == "" && !f.consensus {
		tag = consensusOff
	}
	return []string{e.verdict, tag, "`" + f.key + "`", e.base, e.head, strings.ReplaceAll(e.note, "|", `\|`)}
}

// formatReview renders the current row of every finding. Consensus critical
// rows come first, grouped by classification and verdict; the others are
// grouped by verdict.
func formatReview(baseGeth, headGeth string, findings []finding, prev map[string]entry) string {
	rows := map[string][][]string{}
	critical := map[[2]string][][]string{}
	for _, f := range findings {
		e := currentEntry(f, prev)
		row := tableRow(f, e)
		if isCritical(e.consensus) {
			group := [2]string{e.consensus, e.verdict}
			critical[group] = append(critical[group], row)
		} else {
			rows[e.verdict] = append(rows[e.verdict], row)
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, reviewHeader, baseGeth, headGeth)
	tags := map[string]bool{}
	for group := range critical {
		tags[group[0]] = true
	}
	var section [][]string
	for _, tag := range slices.SortedFunc(maps.Keys(tags), compareCritical) {
		for _, v := range verdicts {
			section = append(section, critical[[2]string{tag, v}]...)
		}
	}
	if len(section) > 0 {
		fmt.Fprintf(&b, "\n## consensus critical (%d)\n\n", len(section))
		writeTable(&b, section)
	}
	for _, v := range verdicts {
		if len(rows[v]) > 0 {
			fmt.Fprintf(&b, "\n## %s (%d)\n\n", v, len(rows[v]))
			writeTable(&b, rows[v])
		}
	}
	return b.String()
}

// writeTable writes rows as a Markdown table under the review header. All
// columns but the last are padded, so that the table stays aligned as text.
func writeTable(b *strings.Builder, rows [][]string) {
	widths := make([]int, len(reviewColumns)-1)
	for _, r := range append([][]string{reviewColumns}, rows...) {
		for i := range widths {
			widths[i] = max(widths[i], len(r[i]))
		}
	}
	line := func(cells []string) {
		for i, c := range cells {
			if i < len(widths) {
				fmt.Fprintf(b, "| %-*s ", widths[i], c)
			} else {
				fmt.Fprintf(b, "| %s |\n", c)
			}
		}
	}
	line(reviewColumns)
	separator := make([]string, len(reviewColumns))
	for i := range separator {
		separator[i] = "---"
		if i < len(widths) {
			separator[i] = strings.Repeat("-", widths[i])
		}
	}
	line(separator)
	for _, r := range rows {
		line(r)
	}
}

func writeReview(path, baseGeth, headGeth string, findings []finding, prev map[string]entry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(formatReview(baseGeth, headGeth, findings, prev)), 0o644)
}
