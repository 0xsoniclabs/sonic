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

package priorities

import (
	"bytes"
	"cmp"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Classifier determines the Priority of a transaction. Implementations may
// query the registry per transaction or apply criteria fetched once per block
// in native code.
type Classifier interface {
	// Priority returns the priority of the transaction. A non-nil error must be
	// treated by the caller as "not prioritized".
	Priority(tx *types.Transaction) (Priority, error)
}

// NonceReader reads block-start account nonces; it is the subset of
// state.StateDB that Prioritize needs to enforce per-sender nonce ordering.
type NonceReader interface {
	GetNonce(common.Address) uint64
}

// transactionWithPriority pairs a transaction with its classified priority.
type transactionWithPriority struct {
	tx       *types.Transaction
	priority Priority
}

// cmpLevelWeightHash orders two transactionWithPriority for the prioritized
// prefix, returning >0 when x precedes y, i.e. by (level desc, weight desc,
// hash asc). Note this is the opposite sign convention to the ascending
// cmpNonceHash.
func (x transactionWithPriority) cmpLevelWeightHash(y transactionWithPriority) int {
	if c := cmp.Compare(x.priority.Level, y.priority.Level); c != 0 {
		return c
	}
	if c := cmp.Compare(x.priority.Weight, y.priority.Weight); c != 0 {
		return c
	}
	return bytes.Compare(y.tx.Hash().Bytes(), x.tx.Hash().Bytes())
}

// cmpNonceHash orders two transactionWithPriority by (nonce asc, hash asc).
func (x transactionWithPriority) cmpNonceHash(y transactionWithPriority) int {
	if c := cmp.Compare(x.tx.Nonce(), y.tx.Nonce()); c != 0 {
		return c
	}
	return bytes.Compare(x.tx.Hash().Bytes(), y.tx.Hash().Bytes())
}

// Prioritize reorders the given base-ordered transactions so that prioritized
// transactions appear first, in (level desc, weight desc, hash asc) order,
// subject to two coupled constraints:
//
//   - Per-sender nonce ordering: a transaction keeps its priority only while it
//     extends its sender's contiguous sequence of prioritized nonces from the
//     block-start account nonce. This keeps each sender in nonce order: a later
//     nonce hoisted ahead of a lower-nonce predecessor left behind would be
//     nonce-too-high and skipped.
//   - Per-entity gas budget: an entity may spend at most
//     cfg.MaxGasPerEntityPerBlock gas (gas limit) on prioritized transactions.
//
// Transactions that are not selected — non-prioritized, nonce-blocked, or over
// budget — are pushed back by the hoisted prioritized prefix but keep their
// original relative order among themselves.
//
// The base order must already be the mode-specific order (scrambler output in
// legacy mode, proposal order in single-proposer mode) and must already be
// filtered to permissible transactions. The result is a permutation of base.
//
// Prioritize is a pure, deterministic, total function of (base, classifier,
// signer, state, cfg): any classifier error is treated as "not prioritized",
// no pass depends on Go map iteration order.
func Prioritize(
	base types.Transactions,
	classifier Classifier,
	signer types.Signer,
	state NonceReader,
	cfg Config,
) types.Transactions {
	if len(base) == 0 {
		return base
	}

	// Determine the priority of every transaction.
	txsWithPrio := classify(base, classifier)

	// Collect the prioritized transactions per sender ordered by nonce which
	// form a continuous sequence starting at the start-of-block sender nonce.
	prioSenderSequences := prioritizedSenderSequences(txsWithPrio, signer, state)

	// Collect the transaction indices which should form the block txs prefix.
	prioPrefixIndices := computePrioritizedTxsPrefix(txsWithPrio, prioSenderSequences, cfg.MaxGasPerEntityPerBlock)

	return combinePrioPrefixWithRemainder(txsWithPrio, prioPrefixIndices)
}

// classify pairs each transaction with its priority, preserving order. A
// classifier error is treated as "not prioritized" (zero Priority), which
// keeps Prioritize a total function of its inputs.
func classify(base types.Transactions, classifier Classifier) []transactionWithPriority {
	txsWithPrio := make([]transactionWithPriority, len(base))
	for i, tx := range base {
		p, err := classifier.Priority(tx)
		if err != nil {
			p = Priority{} // deterministic failure rule: errors => not prioritized
		}
		txsWithPrio[i] = transactionWithPriority{tx: tx, priority: p}
	}
	return txsWithPrio
}

// prioritizedSenderSequences groups the prioritized entries by sender and
// reduces each sender to its sequence: its prioritized transactions in nonce
// order forming a contiguous sequence from the block-start account nonce. Stale
// nonces (below the account nonce) are skipped and the first gap ends the
// sequence, as later nonces are unreachable. A transaction whose sender cannot
// be recovered is left non-prioritized. It returns, per sender, the entry
// indices of the sequence in nonce order; senders left with an empty sequence
// are omitted.
func prioritizedSenderSequences(
	txsWithPrio []transactionWithPriority,
	signer types.Signer,
	state NonceReader,
) map[common.Address][]int {
	// Group prioritized transactions by sender.
	bySender := make(map[common.Address][]int)
	for i := range txsWithPrio {
		if !txsWithPrio[i].priority.IsPrioritized() {
			continue
		}
		sender, err := types.Sender(signer, txsWithPrio[i].tx)
		if err != nil {
			continue // sender unknown: cannot nonce-check, leave non-prioritized
		}
		bySender[sender] = append(bySender[sender], i)
	}

	// Reduce each sender's transactions to its sequence.
	for sender, idxs := range bySender {
		slices.SortFunc(idxs, func(a, b int) int {
			// Sort by nonce to ensure transactions can execute and use hash as
			// tie breaker.
			return txsWithPrio[a].cmpNonceHash(txsWithPrio[b])
		})
		expected := state.GetNonce(sender)
		var sequence []int
		for _, idx := range idxs {
			n := txsWithPrio[idx].tx.Nonce()
			if n < expected {
				continue // stale nonce: do not prioritize this transaction
			}
			if n > expected {
				// nonce gap: do not prioritize this or any later transaction
				// from this sender
				break
			}
			sequence = append(sequence, idx)
			expected++
		}
		if len(sequence) == 0 {
			delete(bySender, sender)
		} else {
			bySender[sender] = sequence
		}
	}
	return bySender
}

// computePrioritizedTxsPrefix walks the per-sender sequences greedily,
// returning the entry indices in prioritized-prefix order. Each step takes the
// highest-priority eligible frontier transaction (a sender's lowest un-selected
// nonce) and advances that sender. A frontier that does not fit its entity
// budget removes the sender's remaining transactions (its later nonces depend
// on it), so a budget is only ever spent on transactions that can actually
// execute in the prioritized prefix. It consumes bySender, mutating it as
// senders are advanced and exhausted.
func computePrioritizedTxsPrefix(
	txsWithPrio []transactionWithPriority,
	bySender map[common.Address][]int,
	perEntityBudget uint64,
) []int {
	selected := make([]int, 0, len(txsWithPrio))
	remaining := make(map[[16]byte]uint64)
	budgetOf := func(id [16]byte) uint64 {
		if r, ok := remaining[id]; ok {
			return r
		}
		return perEntityBudget
	}
	for len(bySender) > 0 {
		best := -1
		var bestSender common.Address
		for sender, sequence := range bySender {
			idx := sequence[0]
			if txsWithPrio[idx].tx.Gas() > budgetOf(txsWithPrio[idx].priority.ID) {
				delete(bySender, sender) // frontier does not fit the budget: sender blocked
				continue
			}
			if best == -1 || txsWithPrio[idx].cmpLevelWeightHash(txsWithPrio[best]) > 0 {
				best, bestSender = idx, sender
			}
		}
		if best == -1 {
			break
		}
		id := txsWithPrio[best].priority.ID
		remaining[id] = budgetOf(id) - txsWithPrio[best].tx.Gas()
		selected = append(selected, best)
		bySender[bestSender] = bySender[bestSender][1:]
		if len(bySender[bestSender]) == 0 {
			delete(bySender, bestSender)
		}
	}
	return selected
}

// combinePrioPrefixWithRemainder builds the final transaction order: the
// prioritized entries in prefix order, followed by the remaining entries
// (demoted + non-prioritized) in their original base order.
func combinePrioPrefixWithRemainder(entries []transactionWithPriority, prioPrefixIndices []int) types.Transactions {
	isPrioritized := make([]bool, len(entries))
	result := make(types.Transactions, 0, len(entries))
	for _, i := range prioPrefixIndices {
		isPrioritized[i] = true
		result = append(result, entries[i].tx)
	}
	// Append the remainder in original base order (demoted + non-prioritized).
	for i := range entries {
		if !isPrioritized[i] {
			result = append(result, entries[i].tx)
		}
	}
	return result
}
