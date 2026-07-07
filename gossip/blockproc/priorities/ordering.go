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
	"slices"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// Classifier determines the Priority of a transaction. Implementations may query
// the registry per transaction (EvmClassifier) or apply criteria fetched once per
// block in native code; both classify every transaction.
type Classifier interface {
	// Priority returns the priority of the transaction. A non-nil error must be
	// treated by the caller as "not prioritized".
	Priority(tx *types.Transaction) (Priority, error)
}

// snapshotter is the subset of state.StateDB used to isolate each registry query
// so a read leaves no residue (warm slots, transient storage, refunds,
// self-destructs) in the state used for block execution.
type snapshotter interface {
	Snapshot() int
	RevertToSnapshot(int)
}

// EvmClassifier classifies transactions by issuing one getPriority call per
// transaction against the registry. Each query is wrapped in a state snapshot
// that is immediately reverted, so individual queries are isolated from one
// another and from subsequent block execution.
type EvmClassifier struct {
	upgrades opera.Upgrades
	vm       VirtualMachine
	signer   types.Signer
	state    snapshotter
}

// NewEvmClassifier creates a Classifier that queries the registry per
// transaction. The provided state must be the same one backing vm so that
// snapshots isolate the query.
func NewEvmClassifier(
	upgrades opera.Upgrades,
	vm VirtualMachine,
	signer types.Signer,
	state snapshotter,
) *EvmClassifier {
	return &EvmClassifier{upgrades: upgrades, vm: vm, signer: signer, state: state}
}

// Priority implements Classifier.
func (c *EvmClassifier) Priority(tx *types.Transaction) (Priority, error) {
	snapshot := c.state.Snapshot()
	defer c.state.RevertToSnapshot(snapshot)
	return GetPriority(c.upgrades, c.vm, c.signer, tx)
}

// Prioritize reorders the given base-ordered transactions so that prioritized
// transactions appear first, sorted by (level desc, weight desc, hash asc), with
// at most cfg.MaxTxsPerEntityPerBlock transactions kept per entity id. Demoted
// (rate-limited) and non-prioritized transactions keep their original base-order
// position.
//
// The base order must already be the mode-specific order (scrambler output in
// legacy mode, proposal order in single-proposer mode) and must already be
// filtered to permissible transactions. The result is a permutation of base.
//
// Prioritize is a pure, deterministic, total-order function of (base, classifier
// results, cfg): any classifier error is treated as "not prioritized", no pass
// depends on Go map iteration order, and ties are broken by transaction hash.
func Prioritize(
	base types.Transactions,
	classifier Classifier,
	cfg Config,
) types.Transactions {
	if len(base) == 0 {
		return base
	}

	type entry struct {
		tx     *types.Transaction
		level  uint256.Int
		weight uint256.Int
		id     [32]byte
		hash   common.Hash
	}

	entries := make([]entry, len(base))
	for i, tx := range base {
		p, err := classifier.Priority(tx)
		if err != nil {
			p = zeroPriority() // deterministic failure rule: errors => not prioritized
		}
		entries[i] = entry{tx: tx, level: p.Level, weight: p.Weight, id: p.Id, hash: tx.Hash()}
	}

	// Group prioritized transactions by entity id (map used only for grouping;
	// it never determines output order).
	byID := make(map[[32]byte][]int)
	for i := range entries {
		if entries[i].level.Sign() > 0 {
			byID[entries[i].id] = append(byID[entries[i].id], i)
		}
	}

	// Within each entity keep at most MaxTxsPerEntityPerBlock by (weight desc,
	// hash asc); the rest are demoted.
	kept := make([]bool, len(entries))
	for _, idxs := range byID {
		slices.SortFunc(idxs, func(a, b int) int {
			if c := entries[b].weight.Cmp(&entries[a].weight); c != 0 {
				return c
			}
			return bytes.Compare(entries[a].hash[:], entries[b].hash[:])
		})
		for k, idx := range idxs {
			if uint64(k) >= cfg.MaxTxsPerEntityPerBlock {
				break
			}
			kept[idx] = true
		}
	}

	// Collect kept (prioritized) entries and sort by (level desc, weight desc,
	// hash asc).
	keptIdx := make([]int, 0, len(entries))
	for i := range entries {
		if kept[i] {
			keptIdx = append(keptIdx, i)
		}
	}
	slices.SortFunc(keptIdx, func(a, b int) int {
		if c := entries[b].level.Cmp(&entries[a].level); c != 0 {
			return c
		}
		if c := entries[b].weight.Cmp(&entries[a].weight); c != 0 {
			return c
		}
		return bytes.Compare(entries[a].hash[:], entries[b].hash[:])
	})

	result := make(types.Transactions, 0, len(entries))
	for _, i := range keptIdx {
		result = append(result, entries[i].tx)
	}
	// Append the remainder in original base order (demoted + non-prioritized).
	for i := range entries {
		if !kept[i] {
			result = append(result, entries[i].tx)
		}
	}
	return result
}
