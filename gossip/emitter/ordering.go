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
	"cmp"
	"math/big"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/utils/frontierheap"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// stage is the consumption stage of a transaction. Stages are consumed in
// ascending order.
type stage uint8

const (
	stagePrioritizedMyTurn    stage = iota // prioritized, this validator's turn
	stagePrioritizedNotMyTurn              // prioritized, another validator's turn
	stageOrdinary                          // not prioritized
)

// txWithMetadata wraps a transaction with its effective miner tip, the address
// that submitted it, its priority and whether it is this validator's turn to
// originate it. Non-prioritized entries carry a zero-valued priority.
type txWithMetadata struct {
	tx       *txpool.LazyTransaction
	from     common.Address
	tip      *uint256.Int
	priority priorities.Priority
	myTurn   bool
}

// stage returns the consumption stage of the transaction.
func (t *txWithMetadata) stage() stage {
	switch {
	case !t.priority.IsPrioritized():
		return stageOrdinary
	case t.myTurn:
		return stagePrioritizedMyTurn
	default:
		return stagePrioritizedNotMyTurn
	}
}

// newTxWithMetadata creates a wrapped transaction, calculating the effective
// miner gasTipCap if a base fee is provided.
// Returns error in case of a negative effective miner gasTipCap.
func newTxWithMetadata(
	tx *txpool.LazyTransaction,
	from common.Address,
	baseFee *uint256.Int,
	priority priorities.Priority,
	myTurn bool,
) (*txWithMetadata, error) {
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
	return &txWithMetadata{
		tx:       tx,
		from:     from,
		tip:      tip,
		priority: priority,
		myTurn:   myTurn,
	}, nil
}

// compareTxByStagePriorityPriceTime orders transactions by (stage asc,
// priority level desc, priority weight desc, effective miner tip desc,
// first-seen time asc), returning >0 when a precedes b.
func compareTxByStagePriorityPriceTime(a, b *txWithMetadata) int {
	if c := cmp.Compare(b.stage(), a.stage()); c != 0 {
		return c
	}
	if c := a.priority.Cmp(b.priority); c != 0 {
		return c
	}
	if c := a.tip.Cmp(b.tip); c != 0 {
		return c
	}
	return b.tx.Time.Compare(a.tx.Time)
}

// priorityLookup resolves the priority of a pending transaction without
// modifying the state, so that priorities can be attached to every candidate
// while an event is being built. It is implemented by priorityCache.
type priorityLookup interface {
	// Priority returns the priority of the transaction with the given hash, or
	// a zero-valued (non-prioritized) priority if it is unknown.
	Priority(hash common.Hash) priorities.Priority
}

// lookupPriority resolves the priority of the given transaction. A nil lookup
// yields a zero-valued (non-prioritized) priority.
func lookupPriority(lookup priorityLookup, tx *txpool.LazyTransaction) priorities.Priority {
	if lookup == nil {
		return priorities.Priority{}
	}
	return lookup.Priority(tx.Hash)
}

// transactionsByPriorityAndPriceAndNonce represents a set of transactions that
// can return transactions in a profit-maximizing sorted order while respecting
// priorities and per-sender nonce sequencing.
//
// Internally it maintains a single heap of all senders' heads ordered by
// stage first, so a head of a later stage only surfaces while no earlier-stage
// head is present. Stages are not monotonic per sender: a promoted head is
// staged on its own priority and turn and may re-enter an earlier stage.
type transactionsByPriorityAndPriceAndNonce struct {
	heap *frontierheap.FrontierHeap[*txWithMetadata]
}

// newTransactionsByPriorityAndPriceAndNonce creates a transaction set that
// returns transactions in (stage asc, priority desc, effective tip desc, time
// asc) order while honouring per-sender nonce sequencing.
//
// Every transaction is wrapped up front: its priority is taken from the lookup
// and its turn from the turn policy. A transaction whose effective tip cannot
// be computed is dropped together with the sender's later nonces, which depend
// on it.
//
// Note, the input map is reowned so the caller should not interact any more
// with it after providing it to the constructor.
//
// Pass a nil lookup to disable priority ordering — every transaction is treated
// as non-prioritized.
func newTransactionsByPriorityAndPriceAndNonce(
	txs map[common.Address][]*txpool.LazyTransaction,
	baseFee *big.Int,
	lookup priorityLookup,
	turnPolicy func(tx *txpool.LazyTransaction) bool,
) *transactionsByPriorityAndPriceAndNonce {
	// Convert the basefee from header format to uint256 format
	var baseFeeUint *uint256.Int
	if baseFee != nil {
		baseFeeUint = uint256.MustFromBig(baseFee)
	}

	t := &transactionsByPriorityAndPriceAndNonce{
		heap: frontierheap.NewFrontierHeap(compareTxByStagePriorityPriceTime),
	}

	for sender, senderTxs := range txs {
		sequence := make([]*txWithMetadata, 0, len(senderTxs))
		for _, tx := range senderTxs {
			priority := lookupPriority(lookup, tx)
			wrapped, err := newTxWithMetadata(tx, sender, baseFeeUint, priority, turnPolicy(tx))
			if err != nil {
				break // the sender's later nonces depend on the dropped transaction
			}
			sequence = append(sequence, wrapped)
		}
		t.heap.AddSequence(sequence)
	}
	return t
}

// Peek returns the best head: the one of the lowest non-empty stage, and
// within that stage the one of highest priority, tip and age. If the set is
// empty, nil is returned along with a false flag.
func (t *transactionsByPriorityAndPriceAndNonce) Peek() (*txWithMetadata, bool) {
	return t.heap.Peek()
}

// Shift drops the best head (see Peek) and promotes the same sender's next
// queued transaction. The promoted transaction is staged on its own priority
// and turn, so it may well precede the head just dropped.
func (t *transactionsByPriorityAndPriceAndNonce) Shift() {
	t.heap.Shift()
}

// Discard drops the best head (see Peek) together with the sender's remaining
// queued transactions.
func (t *transactionsByPriorityAndPriceAndNonce) Discard() {
	t.heap.PopSequence()
}

func (t *transactionsByPriorityAndPriceAndNonce) Copy() *transactionsByPriorityAndPriceAndNonce {
	return &transactionsByPriorityAndPriceAndNonce{heap: t.heap.Clone()}
}
