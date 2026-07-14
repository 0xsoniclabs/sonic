// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package emitter

import (
	"container/heap"
	"maps"
	"math/big"
	"slices"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// txWithMinerFee wraps a transaction with its effective miner tip, the
// address that submitted it, and its priority. Non-prioritized entries carry
// a zero-valued priority.
type txWithMinerFee struct {
	tx       *txpool.LazyTransaction
	from     common.Address
	fees     *uint256.Int
	priority priorities.Priority
}

// newTxWithMinerFee creates a wrapped transaction, calculating the effective
// miner gasTipCap if a base fee is provided.
// Returns error in case of a negative effective miner gasTipCap.
func newTxWithMinerFee(tx *txpool.LazyTransaction, from common.Address, baseFee *uint256.Int, priority priorities.Priority) (*txWithMinerFee, error) {
	tip := new(uint256.Int).Set(tx.GasTipCap)
	if baseFee != nil {
		if tx.GasFeeCap.Cmp(baseFee) < 0 {
			if !subsidies.IsSponsorshipRequest(tx.Tx) {
				return nil, types.ErrGasFeeCapTooLow
			}
		}
		tip = new(uint256.Int).Sub(tx.GasFeeCap, baseFee)
		if tip.Gt(tx.GasTipCap) {
			tip = tx.GasTipCap
		}
	}
	return &txWithMinerFee{
		tx:       tx,
		from:     from,
		fees:     tip,
		priority: priority,
	}, nil
}

// txByPriorityAndPrice orders transactions by (priority level desc, priority
// weight desc, effective miner tip desc, first-seen time asc). Non-prioritized
// entries carry a zero-valued priority and thus collapse to (fees desc, time
// asc) among themselves.
type txByPriorityAndPrice []*txWithMinerFee

func (s txByPriorityAndPrice) Len() int { return len(s) }
func (s txByPriorityAndPrice) Less(i, j int) bool {
	// Prioritized transactions always come before non-prioritized ones.
	// Within the same priority, sort by effective miner tip (desc).
	// Transactions with equal tip are ordered by first-seen time (asc).
	if cmp := s[i].priority.Cmp(s[j].priority); cmp != 0 {
		return cmp > 0
	}
	if cmp := s[i].fees.Cmp(s[j].fees); cmp != 0 {
		return cmp > 0
	}
	return s[i].tx.Time.Before(s[j].tx.Time)
}
func (s txByPriorityAndPrice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s *txByPriorityAndPrice) Push(x interface{}) {
	*s = append(*s, x.(*txWithMinerFee))
}

func (s *txByPriorityAndPrice) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	old[n-1] = nil
	*s = old[0 : n-1]
	return x
}

// transactionsByPriorityAndPriceAndNonce represents a set of transactions that
// can return transactions in a profit-maximizing sorted order while respecting
// priorities and per-sender nonce sequencing.
//
// Internally it maintains three heaps for the three consumption stages:
//   - prioHeads:          prioritized heads awaiting the my-turn check
//     (stage 1).
//   - prioNotMyTurnHeads: prioritized heads that failed the my-turn check and
//     are kept around for eager inclusion via the hinter (stage 2).
//   - nonPrioHeads:       non-prioritized heads (stage 3).
//
// The set does not retain any reference to a priorities.Classifier: priorities
// are resolved via a classifier passed into the constructor (for initial heap
// heads) and into each Shift call (for the head being promoted). This keeps
// classifier lifetime entirely under the caller's control.
type transactionsByPriorityAndPriceAndNonce struct {
	txs                map[common.Address][]*txpool.LazyTransaction // Per account nonce-sorted list of transactions
	prioHeads          txByPriorityAndPrice                         // Prioritized heads pending the my-turn check.
	prioNotMyTurnHeads txByPriorityAndPrice                         // Prioritized heads demoted by the my-turn check.
	nonPrioHeads       txByPriorityAndPrice                         // Non-prioritized heads.
	signer             types.Signer                                 // Signer for the set of transactions
	baseFee            *uint256.Int                                 // Current base fee
}

// newTransactionsByPriorityAndPriceAndNonce creates a transaction set that
// returns transactions in (priority desc, effective tip desc, time asc) order
// while honouring per-sender nonce sequencing.
//
// Note, the input map is reowned so the caller should not interact any more
// with it after providing it to the constructor.
//
// The optional classifier is used to resolve priorities for the initial heap
// heads only. Subsequent promotions receive their classifier via Shift. Pass
// nil to disable priority ordering — every head is treated as non-prioritized.
func newTransactionsByPriorityAndPriceAndNonce(signer types.Signer, txs map[common.Address][]*txpool.LazyTransaction, baseFee *big.Int, classifier priorities.Classifier) *transactionsByPriorityAndPriceAndNonce {
	// Convert the basefee from header format to uint256 format
	var baseFeeUint *uint256.Int
	if baseFee != nil {
		baseFeeUint = uint256.MustFromBig(baseFee)
	}

	// Initialize the prioHeads and nonPrioHeads heaps with the first
	// transaction from each sender. The prioNotMyTurnHeads heap is empty at
	// construction and will be populated later as prioritized heads fail the
	// my-turn check.
	prioHeads := make(txByPriorityAndPrice, 0)
	nonPrioHeads := make(txByPriorityAndPrice, 0, len(txs))
	for from, accTxs := range txs {
		priority := classifyPriority(classifier, accTxs[0])
		wrapped, err := newTxWithMinerFee(accTxs[0], from, baseFeeUint, priority)
		if err != nil {
			delete(txs, from)
			continue
		}
		if priority.IsPrioritized() {
			prioHeads = append(prioHeads, wrapped)
		} else {
			nonPrioHeads = append(nonPrioHeads, wrapped)
		}
		txs[from] = accTxs[1:]
	}
	heap.Init(&prioHeads)
	heap.Init(&nonPrioHeads)

	// Assemble and return the transaction set
	return &transactionsByPriorityAndPriceAndNonce{
		txs:          txs,
		prioHeads:    prioHeads,
		nonPrioHeads: nonPrioHeads,
		signer:       signer,
		baseFee:      baseFeeUint,
	}
}

// classifyPriority resolves the priority of the given transaction. Nil
// classifier or an error yields a zero-valued (non-prioritized) priority.
func classifyPriority(classifier priorities.Classifier, tx *txpool.LazyTransaction) priorities.Priority {
	if classifier == nil {
		return priorities.Priority{}
	}
	p, err := classifier.Priority(tx.Tx)
	if err != nil {
		return priorities.Priority{}
	}
	return p
}

// PeekBest returns the current best head using the combined priority-first
// view across all three heaps (prioHeads, then prioNotMyTurnHeads, then
// nonPrioHeads) or nil if all heaps are empty. This is used for the single
// propose mode where there is no my-turn check.
func (t *transactionsByPriorityAndPriceAndNonce) PeekBest() *txWithMinerFee {
	if len(t.prioHeads) > 0 {
		return t.prioHeads[0]
	}
	if len(t.prioNotMyTurnHeads) > 0 {
		return t.prioNotMyTurnHeads[0]
	}
	if len(t.nonPrioHeads) > 0 {
		return t.nonPrioHeads[0]
	}
	return nil
}

// ShiftBest pops the current best head (see PeekBest) and promotes the same
// sender's next queued transaction. The optional classifier resolves the
// promoted transaction's priority; pass nil to treat it as non-prioritized.
func (t *transactionsByPriorityAndPriceAndNonce) ShiftBest(classifier priorities.Classifier) {
	var acc common.Address
	switch {
	case len(t.prioHeads) > 0:
		acc = t.prioHeads[0].from
		heap.Pop(&t.prioHeads)
	case len(t.prioNotMyTurnHeads) > 0:
		acc = t.prioNotMyTurnHeads[0].from
		heap.Pop(&t.prioNotMyTurnHeads)
	case len(t.nonPrioHeads) > 0:
		acc = t.nonPrioHeads[0].from
		heap.Pop(&t.nonPrioHeads)
	default:
		return
	}
	t.advanceSenderInto(acc, classifier, &t.prioHeads)
}

// DiscardBest drops the current best head (see PeekBest) without promoting
// the sender's next queued transaction.
func (t *transactionsByPriorityAndPriceAndNonce) DiscardBest() {
	switch {
	case len(t.prioHeads) > 0:
		heap.Pop(&t.prioHeads)
	case len(t.prioNotMyTurnHeads) > 0:
		heap.Pop(&t.prioNotMyTurnHeads)
	case len(t.nonPrioHeads) > 0:
		heap.Pop(&t.nonPrioHeads)
	}
}

// PeekPrioHead returns the current head of the prioHeads heap or nil if it
// is empty.
func (t *transactionsByPriorityAndPriceAndNonce) PeekPrioHead() *txWithMinerFee {
	if len(t.prioHeads) == 0 {
		return nil
	}
	return t.prioHeads[0]
}

// ShiftPrioHead pops the current head from the prioHeads heap and promotes
// the same sender's next queued transaction, routing prioritized results
// back into prioHeads.
func (t *transactionsByPriorityAndPriceAndNonce) ShiftPrioHead(classifier priorities.Classifier) {
	if len(t.prioHeads) == 0 {
		return
	}
	acc := t.prioHeads[0].from
	heap.Pop(&t.prioHeads)
	t.advanceSenderInto(acc, classifier, &t.prioHeads)
}

// DiscardPrioHead pops the current head from the prioHeads heap and drops
// the sender's remaining queued transactions.
func (t *transactionsByPriorityAndPriceAndNonce) DiscardPrioHead() {
	if len(t.prioHeads) == 0 {
		return
	}
	acc := t.prioHeads[0].from
	heap.Pop(&t.prioHeads)
	t.discardSender(acc)
}

// DemotePrioHead moves the current head from prioHeads into
// prioNotMyTurnHeads, keeping the same wrapped entry.
func (t *transactionsByPriorityAndPriceAndNonce) DemotePrioHead() {
	if len(t.prioHeads) == 0 {
		return
	}
	entry := heap.Pop(&t.prioHeads).(*txWithMinerFee)
	heap.Push(&t.prioNotMyTurnHeads, entry)
}

// PeekPrioNotMyTurnHead returns the current head of the prioNotMyTurnHeads
// heap or nil if it is empty.
func (t *transactionsByPriorityAndPriceAndNonce) PeekPrioNotMyTurnHead() *txWithMinerFee {
	if len(t.prioNotMyTurnHeads) == 0 {
		return nil
	}
	return t.prioNotMyTurnHeads[0]
}

// ShiftPrioNotMyTurnHead pops the current head from the prioNotMyTurnHeads
// heap and promotes the same sender's next queued transaction. Because the
// sender was already determined not to be at its turn in this iteration,
// prioritized promotions are routed back into prioNotMyTurnHeads so they can
// be processed by the same phase.
func (t *transactionsByPriorityAndPriceAndNonce) ShiftPrioNotMyTurnHead(classifier priorities.Classifier) {
	if len(t.prioNotMyTurnHeads) == 0 {
		return
	}
	acc := t.prioNotMyTurnHeads[0].from
	heap.Pop(&t.prioNotMyTurnHeads)
	t.advanceSenderInto(acc, classifier, &t.prioNotMyTurnHeads)
}

// DiscardPrioNotMyTurnHead pops the current head from the prioNotMyTurnHeads
// heap and drops the sender's remaining queued transactions.
func (t *transactionsByPriorityAndPriceAndNonce) DiscardPrioNotMyTurnHead() {
	if len(t.prioNotMyTurnHeads) == 0 {
		return
	}
	acc := t.prioNotMyTurnHeads[0].from
	heap.Pop(&t.prioNotMyTurnHeads)
	t.discardSender(acc)
}

// PeekNonPrioHead returns the current head of the nonPrioHeads heap or nil
// if it is empty.
func (t *transactionsByPriorityAndPriceAndNonce) PeekNonPrioHead() *txWithMinerFee {
	if len(t.nonPrioHeads) == 0 {
		return nil
	}
	return t.nonPrioHeads[0]
}

// ShiftNonPrioHead pops the current head from the nonPrioHeads heap and
// promotes the same sender's next queued transaction.
func (t *transactionsByPriorityAndPriceAndNonce) ShiftNonPrioHead() {
	if len(t.nonPrioHeads) == 0 {
		return
	}
	acc := t.nonPrioHeads[0].from
	heap.Pop(&t.nonPrioHeads)
	t.advanceSenderInto(acc, nil, nil)
}

// DiscardNonPrioHead pops the current head from the nonPrioHeads heap and
// drops the sender's remaining queued transactions.
func (t *transactionsByPriorityAndPriceAndNonce) DiscardNonPrioHead() {
	if len(t.nonPrioHeads) == 0 {
		return
	}
	acc := t.nonPrioHeads[0].from
	heap.Pop(&t.nonPrioHeads)
	t.discardSender(acc)
}

// advanceSenderInto is like advanceSender but routes prioritized results into
// the given target heap instead of prioHeads. Non-prioritized results always
// go to the nonPrioHeads heap.
func (t *transactionsByPriorityAndPriceAndNonce) advanceSenderInto(sender common.Address, classifier priorities.Classifier, prioTarget *txByPriorityAndPrice) {
	queue, ok := t.txs[sender]
	if !ok || len(queue) == 0 {
		return
	}
	next := queue[0]
	t.txs[sender] = queue[1:]
	priority := classifyPriority(classifier, next)
	wrapped, err := newTxWithMinerFee(next, sender, t.baseFee, priority)
	if err != nil {
		return
	}
	if priority.IsPrioritized() {
		heap.Push(prioTarget, wrapped)
	} else {
		heap.Push(&t.nonPrioHeads, wrapped)
	}
}

// discardSender drops every remaining queued transaction for the given
// sender so they cannot resurface via advanceSender.
func (t *transactionsByPriorityAndPriceAndNonce) discardSender(sender common.Address) {
	delete(t.txs, sender)
}

func (t *transactionsByPriorityAndPriceAndNonce) Copy() *transactionsByPriorityAndPriceAndNonce {
	return &transactionsByPriorityAndPriceAndNonce{
		txs:                maps.Clone(t.txs),
		prioHeads:          slices.Clone(t.prioHeads),
		prioNotMyTurnHeads: slices.Clone(t.prioNotMyTurnHeads),
		nonPrioHeads:       slices.Clone(t.nonPrioHeads),
		signer:             t.signer,
		baseFee:            t.baseFee, // not writable, no need to copy
	}
}
