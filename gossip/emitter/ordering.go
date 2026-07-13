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

// txWithMinerFee wraps a transaction with its gas price or effective miner gasTipCap
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

// txByPriorityAndPriceAndTime implements both the sort and the heap interface,
// making it useful for all at once sorting as well as individually adding and
// removing elements.
type txByPriorityAndPriceAndTime []*txWithMinerFee

func (s txByPriorityAndPriceAndTime) Len() int { return len(s) }
func (s txByPriorityAndPriceAndTime) Less(i, j int) bool {
	// Prioritized transactions always come before non-prioritized ones.
	// Within the same priority, sort by effective miner tip (desc).
	// Transactions with equal tip are ordered by first-seen time (asc).
	if c := s[i].priority.Cmp(s[j].priority); c != 0 {
		return c > 0
	}
	cmp := s[i].fees.Cmp(s[j].fees)
	if cmp == 0 {
		return s[i].tx.Time.Before(s[j].tx.Time)
	}
	return cmp > 0
}
func (s txByPriorityAndPriceAndTime) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s *txByPriorityAndPriceAndTime) Push(x interface{}) {
	*s = append(*s, x.(*txWithMinerFee))
}

func (s *txByPriorityAndPriceAndTime) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	old[n-1] = nil
	*s = old[0 : n-1]
	return x
}

// transactionsByPriorityAndPriceAndNonce represents a set of transactions that
// can return transactions in a profit-maximizing sorted order — prioritized
// transactions first, then by effective miner tip, both in descending order —
// while respecting per-sender nonce sequencing.
//
// The set does not retain any reference to a priorities.Classifier: priorities
// are resolved via a classifier passed into the constructor (for initial heap
// heads) and into each Shift call (for the head being promoted). This keeps
// classifier lifetime entirely under the caller's control.
type transactionsByPriorityAndPriceAndNonce struct {
	txs     map[common.Address][]*txpool.LazyTransaction // Per account nonce-sorted list of transactions
	heads   txByPriorityAndPriceAndTime                  // Next transaction for each unique account (priority+price heap)
	signer  types.Signer                                 // Signer for the set of transactions
	baseFee *uint256.Int                                 // Current base fee
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
// nil to disable priority ordering and fall back to (effective tip desc,
// time asc).
func newTransactionsByPriorityAndPriceAndNonce(signer types.Signer, txs map[common.Address][]*txpool.LazyTransaction, baseFee *big.Int, classifier priorities.Classifier) *transactionsByPriorityAndPriceAndNonce {
	// Convert the basefee from header format to uint256 format
	var baseFeeUint *uint256.Int
	if baseFee != nil {
		baseFeeUint = uint256.MustFromBig(baseFee)
	}

	// Initialize a priority+price+time heap with the head transactions
	heads := make(txByPriorityAndPriceAndTime, 0, len(txs))
	for from, accTxs := range txs {
		wrapped, err := newTxWithMinerFee(accTxs[0], from, baseFeeUint, priorityOf(classifier, accTxs[0]))
		if err != nil {
			delete(txs, from)
			continue
		}
		heads = append(heads, wrapped)
		txs[from] = accTxs[1:]
	}
	heap.Init(&heads)

	// Assemble and return the transaction set
	return &transactionsByPriorityAndPriceAndNonce{
		txs:     txs,
		heads:   heads,
		signer:  signer,
		baseFee: baseFeeUint,
	}
}

// priorityOf resolves the priority of a transaction via the classifier. It
// returns a zero (non-prioritized) Priority if the classifier is nil or the
// query fails.
func priorityOf(classifier priorities.Classifier, tx *txpool.LazyTransaction) priorities.Priority {
	if classifier == nil {
		return priorities.Priority{}
	}
	p, err := classifier.Priority(tx.Tx)
	if err != nil {
		return priorities.Priority{}
	}
	return p
}

// Peek returns the next transaction by priority and price.
func (t *transactionsByPriorityAndPriceAndNonce) Peek() (*txpool.LazyTransaction, *uint256.Int) {
	if len(t.heads) == 0 {
		return nil, nil
	}
	return t.heads[0].tx, t.heads[0].fees
}

// Shift replaces the current best head with the next one from the same account.
// The optional classifier resolves the priority of the promoted transaction;
// pass nil to treat it as non-prioritized.
func (t *transactionsByPriorityAndPriceAndNonce) Shift(classifier priorities.Classifier) {
	acc := t.heads[0].from
	if txs, ok := t.txs[acc]; ok && len(txs) > 0 {
		if wrapped, err := newTxWithMinerFee(txs[0], acc, t.baseFee, priorityOf(classifier, txs[0])); err == nil {
			t.heads[0], t.txs[acc] = wrapped, txs[1:]
			heap.Fix(&t.heads, 0)
			return
		}
	}
	heap.Pop(&t.heads)
}

// Pop removes the best transaction, *not* replacing it with the next one from
// the same account. This should be used when a transaction cannot be executed
// and hence all subsequent ones should be discarded from the same account.
func (t *transactionsByPriorityAndPriceAndNonce) Pop() {
	heap.Pop(&t.heads)
}

// Empty returns if the price heap is empty. It can be used to check it simpler
// than calling peek and checking for nil return.
func (t *transactionsByPriorityAndPriceAndNonce) Empty() bool {
	return len(t.heads) == 0
}

// Clear removes the entire content of the heap.
func (t *transactionsByPriorityAndPriceAndNonce) Clear() {
	t.heads, t.txs = nil, nil
}

func (t *transactionsByPriorityAndPriceAndNonce) Copy() *transactionsByPriorityAndPriceAndNonce {
	txsCopy := maps.Clone(t.txs)
	return &transactionsByPriorityAndPriceAndNonce{
		txs:     txsCopy,
		heads:   slices.Clone(t.heads),
		signer:  t.signer,
		baseFee: t.baseFee, // not writable, no need to copy
	}
}
