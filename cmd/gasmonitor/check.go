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
	"slices"
	"strings"
	"time"
)

// snapshot is what the RPC reported for one block at one point in time.
type snapshot struct {
	Number    uint64
	Hash      string
	GasUsed   uint64
	TxHashes  []string
	Receipts  []rpcReceipt
	FetchedAt time.Time
	Head      uint64 // head known when the snapshot was taken
}

func newSnapshot(block *rpcBlock, receipts []rpcReceipt, head uint64) *snapshot {
	return &snapshot{
		Number:    uint64(block.Number),
		Hash:      block.Hash,
		GasUsed:   uint64(block.GasUsed),
		TxHashes:  block.Transactions,
		Receipts:  receipts,
		FetchedAt: time.Now(),
		Head:      head,
	}
}

// analysis summarizes the gas accounting consistency of a snapshot.
type analysis struct {
	Txs            int
	SumGasUsed     uint64 // sum of receipt gasUsed
	LastCumulative uint64 // cumulativeGasUsed of the last receipt
	// Gap is block.gasUsed - sum(receipt.gasUsed). A positive gap matches the
	// pattern of the known reverted-bundle issue, but is not proof of it.
	Gap int64
	// CumulativeIssues lists receipts whose cumulativeGasUsed is not the
	// previous cumulative value plus their own gasUsed, i.e. where some of the
	// gap is included in the cumulative value.
	CumulativeIssues []string
	// OtherIssues lists structural inconsistencies between block and receipts.
	OtherIssues []string
}

// cumulativeGap is the last receipt's cumulativeGasUsed minus the sum of all
// receipts' gasUsed; non-zero when cumulativeGasUsed includes gas that no
// receipt accounts for.
func (a *analysis) cumulativeGap() int64 { return int64(a.LastCumulative) - int64(a.SumGasUsed) }

func (a *analysis) hasGap() bool { return a.Gap != 0 }
func (a *analysis) hasCumulativeIssue() bool {
	return len(a.CumulativeIssues) > 0 || a.LastCumulative != a.SumGasUsed
}
func (a *analysis) hasOtherIssue() bool { return len(a.OtherIssues) > 0 }
func (a *analysis) clean() bool         { return !a.hasGap() && !a.hasCumulativeIssue() && !a.hasOtherIssue() }

func analyze(s *snapshot) analysis {
	a := analysis{Txs: len(s.TxHashes)}
	var prev uint64
	for i, r := range s.Receipts {
		gas := uint64(r.GasUsed)
		cum := uint64(r.CumulativeGasUsed)
		a.SumGasUsed += gas
		if cum != prev+gas {
			a.CumulativeIssues = append(a.CumulativeIssues, fmt.Sprintf(
				"receipt #%d (tx %s): cumulativeGasUsed %d = previous cumulative %d + own gasUsed %d %+d not covered by any receipt",
				i, shortHash(r.TransactionHash), cum, prev, gas, int64(cum)-int64(prev+gas)))
		}
		prev = cum
		if uint64(r.TransactionIndex) != uint64(i) {
			a.OtherIssues = append(a.OtherIssues, fmt.Sprintf("receipt #%d has transactionIndex %d", i, r.TransactionIndex))
		}
		if i < len(s.TxHashes) && !strings.EqualFold(r.TransactionHash, s.TxHashes[i]) {
			a.OtherIssues = append(a.OtherIssues, fmt.Sprintf("receipt #%d is for tx %s, block lists %s",
				i, shortHash(r.TransactionHash), shortHash(s.TxHashes[i])))
		}
		if r.BlockHash != "" && !strings.EqualFold(r.BlockHash, s.Hash) {
			a.OtherIssues = append(a.OtherIssues, fmt.Sprintf("receipt #%d has blockHash %s, block is %s",
				i, shortHash(r.BlockHash), shortHash(s.Hash)))
		}
	}
	a.LastCumulative = prev
	a.Gap = int64(s.GasUsed) - int64(a.SumGasUsed)
	if len(s.Receipts) != len(s.TxHashes) {
		a.OtherIssues = append(a.OtherIssues, fmt.Sprintf("block has %d txs but %d receipts", len(s.TxHashes), len(s.Receipts)))
	}
	return a
}

// changeKind tells which reported value a single difference belongs to.
type changeKind int

const (
	kindBlockGas   changeKind = iota // the block's gasUsed
	kindReceiptGas                   // a receipt's gasUsed
	kindCumulative                   // a receipt's cumulativeGasUsed
	kindOther                        // block hash, transaction list, status
)

type change struct {
	kind changeKind
	text string
}

// diff lists every value that differs between two snapshots of the same block.
func diff(before, after *snapshot) []change {
	var changes []change
	add := func(kind changeKind, format string, args ...any) {
		changes = append(changes, change{kind: kind, text: fmt.Sprintf(format, args...)})
	}

	if !strings.EqualFold(before.Hash, after.Hash) {
		add(kindOther, "block hash %s -> %s", shortHash(before.Hash), shortHash(after.Hash))
	}
	if before.GasUsed != after.GasUsed {
		add(kindBlockGas, "block gasUsed %d -> %d", before.GasUsed, after.GasUsed)
	}
	if len(before.TxHashes) != len(after.TxHashes) {
		add(kindOther, "tx count %d -> %d", len(before.TxHashes), len(after.TxHashes))
	}
	if len(before.Receipts) != len(after.Receipts) {
		add(kindOther, "receipt count %d -> %d", len(before.Receipts), len(after.Receipts))
	}
	n := min(len(before.Receipts), len(after.Receipts))
	for i := range n {
		b, a := before.Receipts[i], after.Receipts[i]
		if !strings.EqualFold(b.TransactionHash, a.TransactionHash) {
			add(kindOther, "receipt #%d tx %s -> %s", i, shortHash(b.TransactionHash), shortHash(a.TransactionHash))
			continue
		}
		if b.GasUsed != a.GasUsed {
			add(kindReceiptGas, "receipt #%d gasUsed %d -> %d", i, b.GasUsed, a.GasUsed)
		}
		if b.CumulativeGasUsed != a.CumulativeGasUsed {
			add(kindCumulative, "receipt #%d cumulativeGasUsed %d -> %d", i, b.CumulativeGasUsed, a.CumulativeGasUsed)
		}
		if b.Status != a.Status {
			add(kindOther, "receipt #%d status %d -> %d", i, b.Status, a.Status)
		}
	}
	return changes
}

// comparison describes how a later query of a block differs from an earlier
// one, split by the reported value that changed. Each pair is {before, after}.
type comparison struct {
	BlockGas       [2]uint64 // the block's gasUsed
	ReceiptSum     [2]uint64 // sum of the receipts' gasUsed
	LastCumulative [2]uint64 // the last receipt's cumulativeGasUsed
	// Other lists differences not reflected in the three values above.
	Other []string
	// ReceiptDetails lists the individual receipt differences.
	ReceiptDetails []string
}

func compare(before, after *snapshot) comparison {
	b, a := analyze(before), analyze(after)
	c := comparison{
		BlockGas:       [2]uint64{before.GasUsed, after.GasUsed},
		ReceiptSum:     [2]uint64{b.SumGasUsed, a.SumGasUsed},
		LastCumulative: [2]uint64{b.LastCumulative, a.LastCumulative},
	}
	var receiptGas, cumulative bool
	for _, ch := range diff(before, after) {
		switch ch.kind {
		case kindReceiptGas:
			receiptGas = true
			c.ReceiptDetails = append(c.ReceiptDetails, ch.text)
		case kindCumulative:
			cumulative = true
			c.ReceiptDetails = append(c.ReceiptDetails, ch.text)
		case kindOther:
			c.Other = append(c.Other, ch.text)
		}
	}
	if receiptGas && !c.receiptSumChanged() {
		c.Other = append(c.Other, "individual receipt gasUsed changed, their sum did not")
	}
	if cumulative && !c.cumulativeChanged() {
		c.Other = append(c.Other, "intermediate cumulativeGasUsed changed, the last one did not")
	}
	return c
}

func (c *comparison) blockGasChanged() bool   { return c.BlockGas[0] != c.BlockGas[1] }
func (c *comparison) receiptSumChanged() bool { return c.ReceiptSum[0] != c.ReceiptSum[1] }
func (c *comparison) cumulativeChanged() bool { return c.LastCumulative[0] != c.LastCumulative[1] }
func (c *comparison) otherChanged() bool      { return len(c.Other) > 0 }
func (c *comparison) changed() bool {
	return c.blockGasChanged() || c.receiptSumChanged() || c.cumulativeChanged() || c.otherChanged()
}

// valueChange is one reported value of a block in two queries.
type valueChange struct {
	Name          string
	Before, After uint64
}

func (v valueChange) changed() bool { return v.Before != v.After }

// values lists the compared reported values, changed or not.
func (c *comparison) values() []valueChange {
	return []valueChange{
		{"block gasUsed", c.BlockGas[0], c.BlockGas[1]},
		{"receipts Σ", c.ReceiptSum[0], c.ReceiptSum[1]},
		{"last cumulativeGasUsed", c.LastCumulative[0], c.LastCumulative[1]},
	}
}

// equal reports whether two comparisons describe the same differences.
func (c *comparison) equal(o *comparison) bool {
	return c.BlockGas == o.BlockGas && c.ReceiptSum == o.ReceiptSum && c.LastCumulative == o.LastCumulative &&
		slices.Equal(c.Other, o.Other) && slices.Equal(c.ReceiptDetails, o.ReceiptDetails)
}

func shortHash(h string) string {
	if len(h) <= 14 {
		return h
	}
	return h[:8] + ".." + h[len(h)-4:]
}
