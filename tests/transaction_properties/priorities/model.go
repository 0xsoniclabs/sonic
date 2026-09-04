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
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/common"
)

// Priority is what the registry answers for one transaction: a level, zero meaning not prioritized
// and higher scheduled earlier, a weight breaking ties within a level, and the entity the
// transaction belongs to, whose gas budget it spends.
type Priority struct {
	Level  uint64
	Weight uint64
	Entity uint64
}

func (p Priority) IsPrioritized() bool { return p.Level != 0 }

func (p Priority) String() string {
	if !p.IsPrioritized() {
		return "-"
	}
	return fmt.Sprintf("L%d/W%d/E%d", p.Level, p.Weight, p.Entity)
}

// BlockTx is one transaction of a block as the ordering model sees it. The nonce and the gas limit
// are read out of the block rather than out of the drawn spec, because those are what the ordering
// step was given.
type BlockTx struct {
	Hash     common.Hash
	Sender   common.Address
	Nonce    uint64
	Gas      uint64
	Priority Priority
}

func (tx BlockTx) String() string {
	return fmt.Sprintf("%v nonce=%d gas=%d %s",
		tx.Hash.TerminalString(), tx.Nonce, tx.Gas, tx.Priority)
}

// PrioritizedPrefix is the prefix a block's transactions must begin with, in the order they must
// appear in: the prioritized ones, most important first, subject to the two rules that couple them
// to the rest of the block. It mirrors priorities.Prioritize, written from the specification rather
// than by calling it, which would assert nothing.
//
// The rules, in the order they are applied:
//
//   - A transaction keeps its priority only while it extends its sender's contiguous run of
//     prioritized nonces from the nonce the sender held when the block started. A later nonce
//     hoisted past a predecessor left behind would be nonce-too-high and skipped, so the run stops
//     at the first hole.
//   - An entity may spend at most budget gas, counted as gas limits, on prioritized transactions of
//     one block. The transaction that would exceed it is demoted along with the later nonces of its
//     sender, which depend on it, while other senders of the same entity keep packing.
//
// Everything not selected keeps its place behind the prefix.
func PrioritizedPrefix(
	txs []BlockTx,
	startNonce map[common.Address]uint64,
	budget uint64,
) []BlockTx {

	sequences := senderSequences(txs, startNonce)

	// The senders are taken in a fixed order, so that the model cannot depend on map iteration order
	// any more than the ordering step it mirrors does.
	senders := make([]common.Address, 0, len(sequences))
	for sender := range sequences {
		senders = append(senders, sender)
	}
	slices.SortFunc(senders, func(a, b common.Address) int {
		return bytes.Compare(a.Bytes(), b.Bytes())
	})

	remaining := map[uint64]uint64{}
	selected := make([]BlockTx, 0, len(txs))
	for {
		// The frontier is every sender's lowest transaction not yet selected; the most important of
		// those goes next.
		best := -1
		for i, sender := range senders {
			head := sequences[sender]
			if len(head) == 0 {
				continue
			}
			if best < 0 || earlier(head[0], sequences[senders[best]][0]) {
				best = i
			}
		}
		if best < 0 {
			return selected
		}

		sender := senders[best]
		head := sequences[sender][0]
		left, seen := remaining[head.Priority.Entity]
		if !seen {
			left = budget
		}
		if head.Gas > left {
			sequences[sender] = nil // over budget: this sender's remaining nonces go too
			continue
		}
		remaining[head.Priority.Entity] = left - head.Gas
		selected = append(selected, head)
		sequences[sender] = sequences[sender][1:]
	}
}

// senderSequences reduces each sender to the transactions it may have prioritized: its prioritized
// ones in nonce order, forming a contiguous run from the nonce it held when the block started.
func senderSequences(
	txs []BlockTx,
	startNonce map[common.Address]uint64,
) map[common.Address][]BlockTx {

	bySender := map[common.Address][]BlockTx{}
	for _, tx := range txs {
		if tx.Priority.IsPrioritized() {
			bySender[tx.Sender] = append(bySender[tx.Sender], tx)
		}
	}

	for sender, candidates := range bySender {
		slices.SortFunc(candidates, func(a, b BlockTx) int {
			if c := cmp.Compare(a.Nonce, b.Nonce); c != 0 {
				return c
			}
			return bytes.Compare(a.Hash.Bytes(), b.Hash.Bytes())
		})

		expected := startNonce[sender]
		run := make([]BlockTx, 0, len(candidates))
		for _, tx := range candidates {
			if tx.Nonce < expected {
				continue // a nonce already spent, which nothing can bring back
			}
			if tx.Nonce > expected {
				break // a hole, and everything above it is out of reach
			}
			run = append(run, tx)
			expected++
		}
		if len(run) == 0 {
			delete(bySender, sender)
		} else {
			bySender[sender] = run
		}
	}
	return bySender
}

// earlier reports whether a goes before b in the prioritized prefix: the higher level first, the
// higher weight within a level, and the smaller hash to break what is left.
func earlier(a, b BlockTx) bool {
	if a.Priority.Level != b.Priority.Level {
		return a.Priority.Level > b.Priority.Level
	}
	if a.Priority.Weight != b.Priority.Weight {
		return a.Priority.Weight > b.Priority.Weight
	}
	return bytes.Compare(a.Hash.Bytes(), b.Hash.Bytes()) < 0
}
