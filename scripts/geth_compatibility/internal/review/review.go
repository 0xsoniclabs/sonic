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

// Package review maintains the review file of a go-ethereum upgrade: a
// Markdown table row per finding, with the verdict, the consensus
// classification and the note of its reviewer.
package review

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/0xsoniclabs/sonic/scripts/geth_compatibility/internal/analysis"
)

const (
	verdictTodo     = "todo"
	verdictAdopted  = "adopted"
	verdictRefactor = "refactor"
	verdictInactive = "inactive"
)

// verdicts lists the verdicts in the order the review file groups its rows.
var verdicts = []string{verdictTodo, verdictAdopted, verdictRefactor, verdictInactive}

const (
	consensusLive = "live"
	consensusNo   = "no"
	consensusOff  = "-"
)

// classifications lists the consensus classifications other than the upgrades.
var classifications = []string{"", consensusOff, consensusNo, consensusLive}

// openPrefix starts the note of a todo row that asks a human a question.
const openPrefix = "OPEN:"

var columns = []string{"Verdict", "Consensus", "Symbol", "Base", "Head", "Note"}

type row struct {
	verdict   string
	consensus string
	base      string
	head      string
	note      string
}

// Update rewrites the review file at path with a row per finding and returns
// the number of todo rows. A row of the file applies to a finding while the
// hashes of both sides still match and the finding has not moved onto the
// consensus path. Otherwise the finding is reopened as todo, and keeps the note
// of a review and a classification as consensus critical for the next
// reviewer. The upgrades of the head are the hardfork names a classification
// may use.
func Update(path string, base, head *analysis.Side, findings []analysis.Finding) (int, error) {
	prev, err := read(path, head.Upgrades)
	if err != nil {
		return 0, err
	}
	rows := make([]row, len(findings))
	todo := 0
	for i, f := range findings {
		rows[i] = current(f, prev)
		if rows[i].verdict == verdictTodo {
			todo++
		}
	}
	tmp := path + ".tmp"
	versions := func(s *analysis.Side) []string { return append([]string{s.Geth}, s.Modules...) }
	if err := os.WriteFile(tmp, []byte(format(versions(base), versions(head), findings, rows)), 0o644); err != nil {
		return 0, err
	}
	return todo, os.Rename(tmp, path)
}

func current(f analysis.Finding, prev map[string]row) row {
	r := row{verdict: verdictTodo, base: f.Base, head: f.Head}
	p, ok := prev[f.Key]
	switch {
	case !ok:
	case p.base == f.Base && p.head == f.Head && (p.consensus != consensusOff || !f.Consensus):
		r.verdict, r.consensus, r.note = p.verdict, p.consensus, p.note
	default:
		if isCritical(p.consensus) {
			r.consensus = p.consensus
		}
		r.note = p.note
		if p.verdict != verdictTodo || strings.HasPrefix(p.note, openPrefix) {
			r.note = fmt.Sprintf("was %s before the last change: %s", p.verdict, p.note)
		}
	}
	if !f.Consensus && !isCritical(r.consensus) {
		r.consensus = consensusOff
	}
	return r
}

func read(path string, upgrades []string) (map[string]row, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]row{}, nil
	}
	if err != nil {
		return nil, err
	}
	return parse(string(b), path, upgrades)
}

// parse reads the rows of every table in text. Each table must have the review
// columns, so that no edited header or blank line inside a table drops rows
// unnoticed. Lines outside tables are ignored.
func parse(text, name string, upgrades []string) (map[string]row, error) {
	rows := map[string]row{}
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
			if !slices.Equal(cells, columns) || next == nil || !isSeparator(next) {
				return nil, fmt.Errorf("%s:%d: want a table with the columns %s", name, i+1, strings.Join(columns, ", "))
			}
			inTable = true
			i++
			continue
		case len(cells) != len(columns):
			return nil, fmt.Errorf("%s:%d: want the cells %s", name, i+1, strings.Join(columns, ", "))
		}
		key := strings.Trim(cells[2], "`")
		r := row{verdict: cells[0], consensus: cells[1], base: cells[3], head: cells[4], note: cells[5]}
		if !slices.Contains(verdicts, r.verdict) {
			return nil, fmt.Errorf("%s:%d: unknown verdict %q", name, i+1, r.verdict)
		}
		if !slices.Contains(classifications, r.consensus) && !slices.Contains(upgrades, r.consensus) {
			return nil, fmt.Errorf("%s:%d: unknown consensus classification %q", name, i+1, r.consensus)
		}
		if _, dup := rows[key]; dup {
			return nil, fmt.Errorf("%s:%d: duplicate row for %s", name, i+1, key)
		}
		rows[key] = r
	}
	return rows, nil
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

const header = "# go-ethereum compatibility review\n\n" +
	"- Base: %s\n- Head: %s\n\n" +
	"Each row is a difference reachable from Sonic, in go-ethereum or in a module\n" +
	"the tool compares like it, such as Tosca and Carmen. See\n" +
	"`scripts/geth_compatibility/README.md` for the tool and `REVIEWING.md` next to\n" +
	"it for how to review. Record a verdict and a note in each row; the tool\n" +
	"regroups the rows when it runs. Verdicts:\n\n" +
	"- `todo`: not reviewed yet, or a question for a human if its note starts with `OPEN:`\n" +
	"- `adopted`: behavior changes and Sonic accepts it; the note says why and what covers it\n" +
	"- `refactor`: no change in behavior\n" +
	"- `inactive`: only affects forks or features Sonic does not enable\n\n" +
	"The Consensus column says whether the change can alter results all nodes\n" +
	"have to agree on. It is `-` off the consensus path. On the path it is empty\n" +
	"until a reviewer records one of these, which `adopted` and `todo` rows need:\n\n" +
	"- `live`: on hardforks that are already active\n" +
	"- the name of a hardfork, such as `canto`: only once that hardfork is active\n" +
	"- `no`: on the consensus path, but it cannot alter results\n\n" +
	"Rows classified `live` or with a hardfork name are listed first, whatever\n" +
	"their verdict: `live`, then the hardforks by name.\n"

// format renders the rows of the findings. Consensus critical rows come first,
// grouped by classification and verdict; the others are grouped by verdict.
func format(base, head []string, findings []analysis.Finding, rows []row) string {
	byVerdict := map[string][][]string{}
	critical := map[[2]string][][]string{}
	for i, f := range findings {
		r := rows[i]
		cells := []string{r.verdict, r.consensus, "`" + f.Key + "`", r.base, r.head, strings.ReplaceAll(r.note, "|", `\|`)}
		if isCritical(r.consensus) {
			group := [2]string{r.consensus, r.verdict}
			critical[group] = append(critical[group], cells)
		} else {
			byVerdict[r.verdict] = append(byVerdict[r.verdict], cells)
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, header, "`"+strings.Join(base, "`, `")+"`", "`"+strings.Join(head, "`, `")+"`")
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
		if len(byVerdict[v]) > 0 {
			fmt.Fprintf(&b, "\n## %s (%d)\n\n", v, len(byVerdict[v]))
			writeTable(&b, byVerdict[v])
		}
	}
	return b.String()
}

func isCritical(tag string) bool {
	return tag != "" && tag != consensusOff && tag != consensusNo
}

// compareCritical orders critical classifications: live first, then the
// upgrades by name.
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

// writeTable pads all columns but the last, so that the table stays aligned as
// text.
func writeTable(b *strings.Builder, rows [][]string) {
	widths := make([]int, len(columns)-1)
	for _, r := range append([][]string{columns}, rows...) {
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
	line(columns)
	separator := make([]string, len(columns))
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
