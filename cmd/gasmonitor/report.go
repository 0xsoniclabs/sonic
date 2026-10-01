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
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	timeFormat         = "2006-01-02 15:04:05 MST"
	maxIssuesPerBlock  = 5
	maxReportedErrors  = 100
	reportedChangeRows = 1000
)

// writeReport renders the current state as Markdown and atomically replaces
// the report file.
func (m *monitor) writeReport() error {
	content := m.renderReport()
	tmp := m.cfg.reportPath + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, m.cfg.reportPath)
}

func (m *monitor) renderReport() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	var b strings.Builder
	p := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }
	st := &m.stats
	now := time.Now()

	state := "running"
	if m.finished {
		state = "finished"
	}
	p("# Block gas consistency report")
	p("")
	p("| | |")
	p("|---|---|")
	p("| RPC endpoint | `%s` |", redactURL(m.cfg.rpcURL))
	p("| Started | %s |", m.startedAt.Format(timeFormat))
	p("| Last updated | %s (%s, running for %s) |", now.Format(timeFormat), state, now.Sub(m.startedAt).Round(time.Second))
	if st.observed+st.observeErrors > 0 {
		p("| Blocks observed | %d – %d (%d fetched, %d RPC errors) |", st.firstBlock, st.lastBlock, st.observed, st.observeErrors)
	} else {
		p("| Blocks observed | none yet |")
	}
	p("| Chain head | %d |", m.head.Load())
	p("| Re-check depths | %s |", depthList(m.cfg.depths))
	p("")

	notable := make([]*blockRecord, 0, len(m.records))
	for _, r := range m.records {
		if r.notable() {
			notable = append(notable, r)
		}
	}
	slices.SortFunc(notable, func(a, b *blockRecord) int { return compareUint(a.Number, b.Number) })

	depths := depthList(m.cfg.depths)

	// Result in two lines: the gap, and whether values changed later.
	p("## Result")
	p("")
	p("- **Gap between receipts Σ and reported gas:** %d of %d blocks (%s) report a block `gasUsed` different from their receipts Σ; %d (%s) report a last `cumulativeGasUsed` different from it.",
		st.gapBlocks+st.negativeGapBlocks, st.observed, percent(st.gapBlocks+st.negativeGapBlocks, st.observed),
		st.cumulativeGapBlocks, percent(st.cumulativeGapBlocks, st.observed))
	switch {
	case st.changedBlocks > 0:
		p("- **Change deeper in history: YES.** %d block(s) returned different values on re-check (%d of %d re-checks). Which value changed:",
			st.changedBlocks, st.rechecksChanged, st.rechecks)
		p("  - reported block `gasUsed`: **%d block(s)**", st.blockGasChangedBlocks)
		p("  - receipts Σ: **%d block(s)**", st.receiptSumChangedBlocks)
		p("  - last receipt `cumulativeGasUsed`: **%d block(s)**", st.cumulativeChangedBlocks)
		p("  - other fields (block hash, transactions, statuses): **%d block(s)**", st.otherChangedBlocks)
	case st.rechecks > 0:
		p("- **Change deeper in history: none observed.** In all %d re-checks (%s), neither the reported block `gasUsed` nor the receipts Σ (nor any other compared field) differed from the head query or from the previous re-check.", st.rechecks, depths)
	default:
		p("- **Change deeper in history:** no re-checks completed yet.")
	}
	p("")

	// 1. Gap.
	p("## 1. Gap between receipts Σ and reported gas")
	p("")
	p("*Receipts Σ* is the sum of `gasUsed` over all receipts of the block (`eth_getBlockReceipts`), as returned by that query.")
	p("*Reported* is the block's `gasUsed` (`eth_getBlockByNumber`) and the last receipt's `cumulativeGasUsed`.")
	p("Both reported values should equal the receipts Σ; *gap* = reported − receipts Σ.")
	p("")
	p("| Reported value | Blocks with a gap | Share of observed | Total gap (gas) |")
	p("|---|---:|---:|---:|")
	p("| Block `gasUsed` | %d | %s | %+d |", st.gapBlocks+st.negativeGapBlocks, percent(st.gapBlocks+st.negativeGapBlocks, st.observed), st.gapTotal)
	p("| Last receipt `cumulativeGasUsed` | %d | %s | %+d |", st.cumulativeGapBlocks, percent(st.cumulativeGapBlocks, st.observed), st.cumulativeGapTotal)
	p("")
	if st.maxGap != 0 {
		p("Largest block `gasUsed` gap: %+d at block %d.", st.maxGap, st.maxGapBlock)
	}
	if st.negativeGapBlocks > 0 {
		p("**%d block(s) report less gas than their receipts (negative gap); this is not explained by the known issue.**", st.negativeGapBlocks)
	}
	p("")
	p("A positive gap matches the known issue where a reverted single-transaction bundle without an envelope leaves its gas in the block.")
	p("The tool measures the gap only; it does not verify that such a bundle is present in the block.")
	p("")
	p("Values below are from the head query (the first query of the block). Lag = blocks behind the head at that query.")
	p("")
	p("| Block | Lag | Receipts Σ | Block `gasUsed` | Gap | Last `cumulativeGasUsed` | Gap | Re-checks |")
	p("|---:|---:|---:|---:|---:|---:|---:|---|")
	rows := 0
	for _, r := range notable {
		a := &r.Analysis
		if r.Err != "" || !a.hasGap() && a.cumulativeGap() == 0 {
			continue
		}
		rows++
		p("| %d | %d | %d | %d | %+d | %d | %+d | %s |", r.Number, r.HeadLag, a.SumGasUsed,
			r.GasUsed, a.Gap, a.LastCumulative, a.cumulativeGap(), m.recheckState(r))
	}
	if rows == 0 {
		p("| _none_ | | | | | | | |")
	}
	p("")

	// 2. Changes on re-check.
	p("## 2. Changes observed deeper in history")
	p("")
	p("The *head query* is the first query of a block, made when it appeared at the head. Each block is queried again at %s.", depths)
	p("Every re-check is compared with the previous query of the same block (the head query or the re-check at the previous depth)")
	p("and with the head query. Each comparison states separately whether the reported block `gasUsed`, the receipts Σ, or the")
	p("last receipt's `cumulativeGasUsed` changed; *other fields* are the block hash, the transaction list and receipt statuses.")
	p("The endpoint may be served by several backend nodes, so a change may come from a different node answering rather than one")
	p("node changing its data.")
	p("")
	p("A node serves the receipts of recently processed blocks from a cache, where each receipt's `gasUsed` is the gas of its own")
	p("transaction. After a block leaves the cache, the receipts are read from the database, which does not store `gasUsed`; it is")
	p("re-derived from the difference of consecutive `cumulativeGasUsed` values. A change of the receipts Σ while the block `gasUsed`")
	p("and the last `cumulativeGasUsed` stay the same is what this switch looks like when `cumulativeGasUsed` contains gas that no")
	p("receipt accounts for.")
	p("")
	p("- Re-checks: %d done, **%d returned different values** (%d blocks), %d pending, %d RPC errors",
		st.rechecks, st.rechecksChanged, st.changedBlocks, m.pending, st.recheckErrors)
	p("")
	depthTable := func(title string, counts func(*depthStats) *changeCounts) {
		p("### %s", title)
		p("")
		p("| Re-check depth | Re-checks | Same values | Block `gasUsed` changed | Receipts Σ changed | Last `cumulativeGasUsed` changed | Other fields changed | RPC errors |")
		p("|---:|---:|---:|---:|---:|---:|---:|---:|")
		for _, d := range m.cfg.depths {
			ds := st.byDepth[d]
			if ds == nil {
				ds = &depthStats{}
			}
			c := counts(ds)
			p("| %d | %d | %d | %s | %s | %s | %s | %d |", d, ds.rechecks, c.same, boldIfNonZero(c.blockGas),
				boldIfNonZero(c.receiptSum), boldIfNonZero(c.cumulative), boldIfNonZero(c.other), ds.errors)
		}
		p("")
	}
	depthTable("Compared with the previous query of the same block", func(ds *depthStats) *changeCounts { return &ds.vsPrevious })
	depthTable("Compared with the head query", func(ds *depthStats) *changeCounts { return &ds.vsHead })

	p("### Blocks that returned different values")
	p("")
	changed := 0
	for _, r := range notable {
		if !r.Changed {
			continue
		}
		changed++
		if changed > reportedChangeRows {
			p("_… further blocks omitted._")
			break
		}
		p("#### Block %d: changed %s", r.Number, changedValues(r))
		p("")
		p("| Query | Time | Depth | Receipts Σ | Reported block `gasUsed` | Gap | Last `cumulativeGasUsed` | Gap |")
		p("|---|---|---:|---:|---:|---:|---:|---:|")
		a := &r.Analysis
		p("| head | %s | %d | %d | %d | %+d | %d | %+d |", r.ObservedAt.Format(time.TimeOnly), r.HeadLag,
			a.SumGasUsed, r.GasUsed, a.Gap, a.LastCumulative, a.cumulativeGap())
		for i, rc := range r.Rechecks {
			if rc.Err != "" {
				p("| re-check %d | %s | %d | RPC error: %s | | | | |", i+1, rc.At.Format(time.TimeOnly), rc.Depth, escape(rc.Err))
				continue
			}
			ra := &rc.Analysis
			p("| re-check %d | %s | %d | %d | %d | %+d | %d | %+d |", i+1, rc.At.Format(time.TimeOnly), headDepth(rc.Head, r.Number),
				ra.SumGasUsed, rc.GasUsed, ra.Gap, ra.LastCumulative, ra.cumulativeGap())
		}
		p("")
		for i, rc := range r.Rechecks {
			if rc.Err != "" {
				continue
			}
			p("- re-check %d (depth %d)", i+1, rc.Depth)
			comparisonList(p, "vs previous query", &rc.VsPrevious)
			if !rc.VsPrevious.equal(&rc.VsHead) {
				comparisonList(p, "vs head query", &rc.VsHead)
			}
		}
		p("")
	}
	if changed == 0 {
		p("None.")
		p("")
	}

	// Supporting detail.
	p("## Supporting detail")
	p("")
	p("### Receipts whose `cumulativeGasUsed` includes gas not in any receipt")
	p("")
	details := 0
	for _, r := range notable {
		issues := r.Analysis.CumulativeIssues
		if len(issues) == 0 {
			continue
		}
		details++
		p("- Block %d", r.Number)
		for i, issue := range issues {
			if i == maxIssuesPerBlock {
				p("  - … %d more", len(issues)-maxIssuesPerBlock)
				break
			}
			p("  - %s", issue)
		}
	}
	if details == 0 {
		p("None.")
	}
	p("")
	mismatches := 0
	for _, r := range notable {
		if len(r.Analysis.OtherIssues) == 0 {
			continue
		}
		if mismatches == 0 {
			p("### Receipts that do not line up with the block's transactions")
			p("")
		}
		mismatches++
		p("- Block %d: %s", r.Number, strings.Join(r.Analysis.OtherIssues, "; "))
	}
	if mismatches > 0 {
		p("")
	}
	p("### RPC errors")
	p("")
	p("%d on head queries, %d on re-checks. Blocks whose head query failed are not counted above.", st.observeErrors, st.recheckErrors)
	p("")
	if len(m.errors) > 0 {
		start := max(0, len(m.errors)-maxReportedErrors)
		if start > 0 {
			p("_Showing the last %d of %d errors._", maxReportedErrors, len(m.errors))
			p("")
		}
		for _, e := range m.errors[start:] {
			p("- %s: %s", e.At.Format(time.TimeOnly), e.Msg)
		}
	}
	return b.String()
}

func (m *monitor) recheckState(r *blockRecord) string {
	total := len(m.cfg.depths)
	if total == 0 {
		return "disabled"
	}
	rpcErrors := 0
	for _, rc := range r.Rechecks {
		if rc.Err != "" {
			rpcErrors++
		}
	}
	var parts []string
	switch {
	case r.Changed:
		parts = append(parts, "**CHANGED**")
	case r.remaining > 0:
		parts = append(parts, fmt.Sprintf("pending (%d/%d done)", len(r.Rechecks), total))
	default:
		parts = append(parts, fmt.Sprintf("unchanged (%d/%d)", len(r.Rechecks)-rpcErrors, total))
	}
	if rpcErrors > 0 {
		parts = append(parts, fmt.Sprintf("%d RPC errors", rpcErrors))
	}
	return strings.Join(parts, ", ")
}

func defaultReportPath() string {
	return filepath.Join(".", fmt.Sprintf("gas-report-%s.md", time.Now().Format("20060102-150405")))
}

func depthList(depths []uint64) string {
	if len(depths) == 0 {
		return "disabled"
	}
	s := make([]string, len(depths))
	for i, d := range depths {
		s[i] = fmt.Sprint(d)
	}
	return strings.Join(s, ", ") + " blocks behind head"
}

func headDepth(head, number uint64) uint64 {
	if head < number {
		return 0
	}
	return head - number
}

func compareUint(a, b uint64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// changedValues names the values that changed in any comparison of a block.
func changedValues(r *blockRecord) string {
	var names []string
	if r.BlockGasChanged {
		names = append(names, "reported block `gasUsed`")
	}
	if r.ReceiptSumChanged {
		names = append(names, "receipts Σ")
	}
	if r.CumulativeChanged {
		names = append(names, "last `cumulativeGasUsed`")
	}
	if r.OtherChanged {
		names = append(names, "other fields")
	}
	return strings.Join(names, ", ")
}

// comparisonList renders a comparison as a nested Markdown list, one line
// per reported value.
func comparisonList(p func(string, ...any), title string, c *comparison) {
	if !c.changed() {
		p("  - %s: same values", title)
		return
	}
	p("  - %s:", title)
	for _, v := range c.values() {
		if v.changed() {
			p("    - %s **CHANGED** %d → %d (%+d)", v.Name, v.Before, v.After, int64(v.After)-int64(v.Before))
		} else {
			p("    - %s unchanged (%d)", v.Name, v.Before)
		}
	}
	for _, o := range c.Other {
		p("    - other: %s", o)
	}
	for i, d := range c.ReceiptDetails {
		if i == maxReceiptDetails {
			p("    - … %d more receipt differences", len(c.ReceiptDetails)-maxReceiptDetails)
			break
		}
		p("    - %s", d)
	}
}

func boldIfNonZero(v uint64) string {
	if v == 0 {
		return "0"
	}
	return fmt.Sprintf("**%d**", v)
}

func escape(s string) string { return strings.ReplaceAll(s, "|", "\\|") }
