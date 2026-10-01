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
	"container/heap"
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type config struct {
	rpcURL      string
	reportPath  string
	depths      []uint64 // re-check depths in blocks behind head, ascending
	workers     int
	poll        time.Duration
	reportEvery time.Duration
	timeout     time.Duration
	retries     int
	from        uint64 // 0 = start at the current head
	maxBlocks   uint64 // 0 = unlimited
	duration    time.Duration
}

type recheckResult struct {
	Depth    uint64
	At       time.Time
	Head     uint64
	GasUsed  uint64
	Analysis analysis
	// VsHead compares with the head query, the first query of the block.
	VsHead comparison
	// VsPrevious compares with the latest earlier query that succeeded (the
	// head query or an earlier re-check).
	VsPrevious comparison
	Err        string
}

type blockRecord struct {
	Number     uint64
	ObservedAt time.Time
	HeadLag    uint64 // head - number at the time of the head query
	Hash       string
	GasUsed    uint64
	Analysis   analysis
	Rechecks   []recheckResult
	Changed    bool
	// Which values changed in any comparison of this block.
	BlockGasChanged, ReceiptSumChanged, CumulativeChanged, OtherChanged bool
	Err                                                                 string

	first     *snapshot // kept only while re-checks are pending
	last      *snapshot // latest successful query, kept while re-checks are pending
	remaining int       // re-checks still to be done
}

func (r *blockRecord) notable() bool {
	if r.Err != "" || r.Changed || !r.Analysis.clean() {
		return true
	}
	for _, rc := range r.Rechecks {
		if rc.Err != "" || !rc.Analysis.clean() {
			return true
		}
	}
	return false
}

type stats struct {
	firstBlock, lastBlock uint64
	observed              uint64
	observeErrors         uint64
	gapBlocks             uint64 // header gasUsed > sum of receipts
	negativeGapBlocks     uint64 // header gasUsed < sum of receipts
	gapTotal              int64  // sum of all block gasUsed gaps
	cumulativeGapTotal    int64  // sum of all last cumulativeGasUsed gaps
	maxGap                int64
	maxGapBlock           uint64
	cumulativeGapBlocks   uint64 // last cumulativeGasUsed != sum of receipts
	otherIssueBlocks      uint64
	rechecks              uint64
	recheckErrors         uint64
	rechecksChanged       uint64
	fullyRechecked        uint64
	changedBlocks         uint64
	// Blocks in which the given value changed in any comparison.
	blockGasChangedBlocks   uint64
	receiptSumChangedBlocks uint64
	cumulativeChangedBlocks uint64
	otherChangedBlocks      uint64
	byDepth                 map[uint64]*depthStats
}

// depthStats summarizes the re-checks done at one depth.
type depthStats struct {
	rechecks   uint64
	errors     uint64
	vsPrevious changeCounts
	vsHead     changeCounts
}

// changeCounts counts comparisons by the value that changed; one comparison
// can count for several values.
type changeCounts struct {
	same, blockGas, receiptSum, cumulative, other uint64
}

func (c *changeCounts) add(cmp *comparison) {
	if !cmp.changed() {
		c.same++
	}
	if cmp.blockGasChanged() {
		c.blockGas++
	}
	if cmp.receiptSumChanged() {
		c.receiptSum++
	}
	if cmp.cumulativeChanged() {
		c.cumulative++
	}
	if cmp.otherChanged() {
		c.other++
	}
}

type errorEntry struct {
	At  time.Time
	Msg string
}

type recheckItem struct{ target, block, depth uint64 }

type recheckHeap []recheckItem

func (h recheckHeap) Len() int           { return len(h) }
func (h recheckHeap) Less(i, j int) bool { return h[i].target < h[j].target }
func (h recheckHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *recheckHeap) Push(x any)        { *h = append(*h, x.(recheckItem)) }
func (h *recheckHeap) Pop() any {
	old := *h
	x := old[len(old)-1]
	*h = old[:len(old)-1]
	return x
}

type monitor struct {
	cfg     config
	rpc     *rpcClient
	console *console

	head     atomic.Uint64
	inflight atomic.Int64

	mu        sync.Mutex
	startedAt time.Time
	finished  bool
	records   map[uint64]*blockRecord // pending or notable blocks
	due       recheckHeap
	pending   int // re-checks scheduled but not completed
	stats     stats
	errors    []errorEntry
}

func newMonitor(cfg config, con *console) *monitor {
	return &monitor{
		cfg:       cfg,
		rpc:       newRPCClient(cfg.rpcURL, cfg.timeout),
		console:   con,
		startedAt: time.Now(),
		records:   map[uint64]*blockRecord{},
		stats:     stats{byDepth: map[uint64]*depthStats{}},
	}
}

// run follows the chain until ctx is cancelled, or until following stops
// (stopFollowing closed, block/time limit reached) and all scheduled
// re-checks are done.
func (m *monitor) run(ctx context.Context, stopFollowing <-chan struct{}) error {
	head, err := m.rpc.blockNumber(ctx)
	if err != nil {
		return fmt.Errorf("cannot reach %s: %w", redactURL(m.cfg.rpcURL), err)
	}
	m.head.Store(head)
	next := head
	if m.cfg.from != 0 {
		next = m.cfg.from
	}
	m.console.event("connected to %s, head is %d, starting at block %d", redactURL(m.cfg.rpcURL), head, next)

	jobs := make(chan func(), m.cfg.workers*4)
	var wg sync.WaitGroup
	for range m.cfg.workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				job()
				m.inflight.Add(-1)
			}
		}()
	}
	defer func() {
		close(jobs)
		wg.Wait()
	}()
	dispatch := func(job func()) bool {
		m.inflight.Add(1)
		select {
		case jobs <- job:
			return true
		case <-ctx.Done():
			m.inflight.Add(-1)
			return false
		}
	}

	var deadline <-chan time.Time
	if m.cfg.duration > 0 {
		deadline = time.After(m.cfg.duration)
	}
	following := true
	stopFollow := func(reason string) {
		if following {
			following = false
			m.console.event("%s; no new blocks will be followed, waiting for pending re-checks (Ctrl-C again to quit now)", reason)
		}
	}
	ticker := time.NewTicker(m.cfg.poll)
	defer ticker.Stop()

	var observedCount uint64
	for {
		if head, err := m.rpc.blockNumber(ctx); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			m.recordError("eth_blockNumber: %v", err)
		} else if head > m.head.Load() {
			m.head.Store(head)
		}
		head := m.head.Load()

		for following && next <= head {
			if m.cfg.maxBlocks > 0 && observedCount >= m.cfg.maxBlocks {
				stopFollow(fmt.Sprintf("observed %d blocks", observedCount))
				break
			}
			n := next
			if !dispatch(func() { m.observe(ctx, n) }) {
				return nil
			}
			next++
			observedCount++
		}
		for _, item := range m.popDue(head) {
			if !dispatch(func() { m.recheck(ctx, item.block, item.depth) }) {
				return nil
			}
		}
		if !following && m.idle() {
			return nil
		}

		select {
		case <-ctx.Done():
			return nil
		case <-stopFollowing:
			stopFollow("interrupted")
			stopFollowing = nil
		case <-deadline:
			stopFollow(fmt.Sprintf("ran for %s", m.cfg.duration))
		case <-ticker.C:
		}
	}
}

func (m *monitor) idle() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pending == 0 && m.inflight.Load() == 0
}

func (m *monitor) popDue(head uint64) []recheckItem {
	m.mu.Lock()
	defer m.mu.Unlock()
	var items []recheckItem
	for m.due.Len() > 0 && m.due[0].target <= head {
		items = append(items, heap.Pop(&m.due).(recheckItem))
	}
	return items
}

func (m *monitor) fetch(ctx context.Context, number uint64) (*snapshot, error) {
	var lastErr error
	for attempt := 0; attempt <= m.cfg.retries; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(time.Duration(attempt) * 300 * time.Millisecond):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		head := m.head.Load()
		block, receipts, err := m.rpc.fetchBlock(ctx, number)
		if err == nil {
			return newSnapshot(block, receipts, head), nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (m *monitor) observe(ctx context.Context, number uint64) {
	snap, err := m.fetch(ctx, number)
	if ctx.Err() != nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	st := &m.stats
	if st.observed+st.observeErrors == 0 || number < st.firstBlock {
		st.firstBlock = number
	}
	st.lastBlock = max(st.lastBlock, number)

	rec := &blockRecord{Number: number, ObservedAt: time.Now()}
	if err != nil {
		st.observeErrors++
		rec.Err = err.Error()
		m.records[number] = rec
		m.addError("block %d: RPC error on head query: %v", number, err)
		return
	}
	st.observed++
	rec.Hash, rec.GasUsed, rec.first, rec.last = snap.Hash, snap.GasUsed, snap, snap
	if snap.Head > number {
		rec.HeadLag = snap.Head - number
	}
	rec.Analysis = analyze(snap)
	a := &rec.Analysis
	if a.Gap > 0 {
		st.gapBlocks++
	} else if a.Gap < 0 {
		st.negativeGapBlocks++
	}
	if a.hasGap() && abs(a.Gap) > abs(st.maxGap) {
		st.maxGap, st.maxGapBlock = a.Gap, number
	}
	st.gapTotal += a.Gap
	if a.cumulativeGap() != 0 {
		st.cumulativeGapBlocks++
		st.cumulativeGapTotal += a.cumulativeGap()
	}
	if a.hasOtherIssue() {
		st.otherIssueBlocks++
	}
	if !a.clean() {
		m.console.event("block %d: %s", number, describe(rec.GasUsed, a))
	}

	if len(m.cfg.depths) == 0 {
		rec.first, rec.last = nil, nil
		if rec.notable() {
			m.records[number] = rec
		}
		return
	}
	m.records[number] = rec
	rec.remaining = len(m.cfg.depths)
	m.pending += len(m.cfg.depths)
	for _, d := range m.cfg.depths {
		heap.Push(&m.due, recheckItem{target: number + d, block: number, depth: d})
	}
}

func (m *monitor) recheck(ctx context.Context, number, depth uint64) {
	snap, err := m.fetch(ctx, number)
	if ctx.Err() != nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	rec := m.records[number]
	if rec == nil || rec.first == nil {
		return
	}
	m.stats.rechecks++
	ds := m.stats.byDepth[depth]
	if ds == nil {
		ds = &depthStats{}
		m.stats.byDepth[depth] = ds
	}
	ds.rechecks++
	res := recheckResult{Depth: depth, At: time.Now(), Head: m.head.Load()}
	if err != nil {
		m.stats.recheckErrors++
		ds.errors++
		res.Err = err.Error()
		m.addError("block %d: RPC error on re-check at depth %d: %v", number, depth, err)
	} else {
		res.Head = snap.Head
		res.GasUsed = snap.GasUsed
		res.Analysis = analyze(snap)
		res.VsHead = compare(rec.first, snap)
		res.VsPrevious = compare(rec.last, snap)
		rec.last = snap
		ds.vsHead.add(&res.VsHead)
		ds.vsPrevious.add(&res.VsPrevious)
	}
	rec.Rechecks = append(rec.Rechecks, res)
	// A value that changed and later changed back differs from the previous
	// query only, so either comparison counts as a change.
	if res.VsHead.changed() || res.VsPrevious.changed() {
		m.stats.rechecksChanged++
		if !rec.Changed {
			m.stats.changedBlocks++
		}
		rec.Changed = true
		st := &m.stats
		mark := func(flag *bool, changed bool, count *uint64) {
			if changed && !*flag {
				*flag = true
				*count++
			}
		}
		for _, c := range []*comparison{&res.VsHead, &res.VsPrevious} {
			mark(&rec.BlockGasChanged, c.blockGasChanged(), &st.blockGasChangedBlocks)
			mark(&rec.ReceiptSumChanged, c.receiptSumChanged(), &st.receiptSumChangedBlocks)
			mark(&rec.CumulativeChanged, c.cumulativeChanged(), &st.cumulativeChangedBlocks)
			mark(&rec.OtherChanged, c.otherChanged(), &st.otherChangedBlocks)
		}
		m.console.event("block %d: CHANGED at re-check depth %d\n%s", number, depth, queryHistory(rec))
	}
	rec.remaining--
	m.pending--
	if rec.remaining == 0 {
		m.stats.fullyRechecked++
		rec.first, rec.last = nil, nil
		if !rec.notable() {
			delete(m.records, number)
		}
	}
}

func (m *monitor) recordError(format string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.addError(format, args...)
}

// addError must be called with m.mu held.
func (m *monitor) addError(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	m.errors = append(m.errors, errorEntry{At: time.Now(), Msg: msg})
	if len(m.errors) > 1000 {
		m.errors = m.errors[len(m.errors)-1000:]
	}
	m.console.event("error: %s", msg)
}

func (m *monitor) statusLine() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := &m.stats
	return fmt.Sprintf("head %d | observed %d (last %d) | gap: block gasUsed %d, cumulativeGasUsed %d | CHANGED on re-check %d blocks (%d re-checks, %d pending) | RPC errors %d",
		m.head.Load(), st.observed, st.lastBlock, st.gapBlocks+st.negativeGapBlocks, st.cumulativeGapBlocks,
		st.changedBlocks, st.rechecks, m.pending, st.observeErrors+st.recheckErrors)
}

func describe(gasUsed uint64, a *analysis) string {
	desc := fmt.Sprintf("receipts Σ %d (%d txs) | block gasUsed %d (gap %+d) | last cumulativeGasUsed %d (gap %+d)",
		a.SumGasUsed, a.Txs, gasUsed, a.Gap, a.LastCumulative, a.cumulativeGap())
	if len(a.OtherIssues) > 0 {
		desc += " | " + strings.Join(a.OtherIssues, "; ")
	}
	return desc
}

// maxReceiptDetails limits the per-receipt differences shown per comparison.
const maxReceiptDetails = 5

// queryHistory lists every query of a block so far with its values, and below
// each re-check what differs from the previous query (and from the head query,
// when that says something else).
func queryHistory(rec *blockRecord) string {
	var b strings.Builder
	values := func(label string, gasUsed uint64, a *analysis) {
		fmt.Fprintf(&b, "    %-19s receipts Σ %d | block gasUsed %d (gap %+d) | last cumulativeGasUsed %d (gap %+d)\n",
			label, a.SumGasUsed, gasUsed, a.Gap, a.LastCumulative, a.cumulativeGap())
	}
	compared := func(title string, c *comparison) {
		if !c.changed() {
			fmt.Fprintf(&b, "        %s: same values\n", title)
			return
		}
		fmt.Fprintf(&b, "        %s:\n", title)
		for _, v := range c.values() {
			if v.changed() {
				fmt.Fprintf(&b, "            %-24s CHANGED    %d -> %d (%+d)\n", v.Name, v.Before, v.After, int64(v.After)-int64(v.Before))
			} else {
				fmt.Fprintf(&b, "            %-24s unchanged  %d\n", v.Name, v.Before)
			}
		}
		for _, o := range c.Other {
			fmt.Fprintf(&b, "            other: %s\n", o)
		}
		for i, d := range c.ReceiptDetails {
			if i == maxReceiptDetails {
				fmt.Fprintf(&b, "            … %d more receipt differences\n", len(c.ReceiptDetails)-maxReceiptDetails)
				break
			}
			fmt.Fprintf(&b, "            %s\n", d)
		}
	}

	values(fmt.Sprintf("head (lag %d)", rec.HeadLag), rec.GasUsed, &rec.Analysis)
	for _, rc := range rec.Rechecks {
		label := fmt.Sprintf("re-check depth %d", rc.Depth)
		if rc.Err != "" {
			fmt.Fprintf(&b, "    %-19s RPC error: %s\n", label, rc.Err)
			continue
		}
		values(label, rc.GasUsed, &rc.Analysis)
		compared("vs previous query", &rc.VsPrevious)
		if !rc.VsPrevious.equal(&rc.VsHead) {
			compared("vs head query", &rc.VsHead)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func percent(part, total uint64) string {
	if total == 0 {
		return "0%"
	}
	return fmt.Sprintf("%.2f%%", 100*float64(part)/float64(total))
}
