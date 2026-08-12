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

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils/frontierheap"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

//go:generate mockgen -source=ordering.go -destination=ordering_mock.go -package=priorities

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

// Snapshotter is the subset of state.StateDB used to isolate each registry query
// so a read leaves no residue (warm slots, transient storage, refunds,
// self-destructs) in the state used for block execution.
type Snapshotter interface {
	InterTxSnapshot() int
	RevertToInterTxSnapshot(id int)
}

// EvmClassifier classifies transactions by issuing one getPriority call per
// transaction against the registry. Each query is wrapped in a state snapshot
// that is immediately reverted, so individual queries are isolated from one
// another and from subsequent block execution.
type EvmClassifier struct {
	upgrades opera.Upgrades
	vm       VirtualMachine
	signer   types.Signer
	state    Snapshotter
	failures Meter
}

// NewEvmClassifier creates a Classifier that queries the registry per
// transaction. The provided state must be the same one backing vm so that
// snapshots isolate the query. Failed classifications, which silently degrade
// the transaction to "not prioritized", are reported to the failures meter.
func NewEvmClassifier(
	upgrades opera.Upgrades,
	vm VirtualMachine,
	signer types.Signer,
	state Snapshotter,
	failures Meter,
) *EvmClassifier {
	return &EvmClassifier{
		upgrades: upgrades,
		vm:       vm,
		signer:   signer,
		state:    state,
		failures: failures,
	}
}

// Priority implements Classifier.
func (c *EvmClassifier) Priority(tx *types.Transaction) (Priority, error) {
	snapshot := c.state.InterTxSnapshot()
	defer c.state.RevertToInterTxSnapshot(snapshot)

	priority, err := GetPriority(c.upgrades, c.vm, c.signer, tx)
	if err != nil {
		c.failures.Mark(1)
	}
	return priority, err
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
	prioritizedPrefixIndices := computePrioritizedTxsPrefix(txsWithPrio, prioSenderSequences, cfg.MaxGasPerEntityPerBlock)

	return combinePrioritizedPrefixWithRemainder(txsWithPrio, prioritizedPrefixIndices)
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
// execute in the prioritized prefix.
func computePrioritizedTxsPrefix(
	txsWithPrio []transactionWithPriority,
	bySender map[common.Address][]int,
	perEntityBudget uint64,
) []int {
	frontier := frontierheap.NewFrontierHeap(func(a, b int) int {
		return txsWithPrio[a].cmpLevelWeightHash(txsWithPrio[b])
	})
	for _, sequence := range bySender {
		frontier.AddSequence(sequence)
	}

	selected := make([]int, 0, len(txsWithPrio))
	remaining := make(map[PriorityID]uint64)
	for {
		idx, ok := frontier.Peek()
		if !ok {
			break
		}
		gas := txsWithPrio[idx].tx.Gas()
		id := txsWithPrio[idx].priority.ID
		budget, seen := remaining[id]
		if !seen {
			budget = perEntityBudget
		}
		if gas > budget {
			frontier.PopSequence() // tx does not fit the budget: sender blocked
			continue
		}
		remaining[id] = budget - gas
		selected = append(selected, idx)
		frontier.Shift()
	}
	return selected
}

// combinePrioritizedPrefixWithRemainder builds the final transaction order:
// the prioritized entries in prefix order, followed by the remaining entries
// (demoted + non-prioritized) in their original base order.
func combinePrioritizedPrefixWithRemainder(
	txsWithPrio []transactionWithPriority,
	prioritizedPrefixIndices []int,
) types.Transactions {
	isPrioritized := make([]bool, len(txsWithPrio))
	result := make(types.Transactions, 0, len(txsWithPrio))
	for _, i := range prioritizedPrefixIndices {
		isPrioritized[i] = true
		result = append(result, txsWithPrio[i].tx)
	}
	// Append the remainder in original base order (demoted + non-prioritized).
	for i := range txsWithPrio {
		if !isPrioritized[i] {
			result = append(result, txsWithPrio[i].tx)
		}
	}
	return result
}
