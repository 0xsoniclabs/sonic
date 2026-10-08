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

package gossip

import (
	"fmt"
	"slices"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// blockScan determines the execution plans marked as processed in one block.
type blockScan struct {
	epoch         idx.Epoch
	number, first uint64
	time          inter.Timestamp
	rules         *opera.Rules
	signer        types.Signer
	marked        map[common.Hash]uint64 // < plans marked in earlier blocks
	blockTx       types.Transactions
}

// scan marks the top-level bundles of the given proposal transactions passing
// the checks performed before a bundle is executed, and cross-checks the
// result with the bundle-only markers of the stored block. Nested bundles are
// refused, as their processing depends on the execution of the enclosing
// bundle.
func (b *blockScan) scan(proposed []*types.Transaction) (map[common.Hash]bundle.PositionInBlock, error) {
	txs := filterNonPermissibleTransactions(slices.Clone(proposed), b.rules, b.signer, nil, nil)

	marks := map[common.Hash]bundle.PositionInBlock{}
	plans := map[common.Hash]*bundle.ExecutionPlan{}
	for _, tx := range txs {
		if !bundle.IsEnvelope(tx) {
			continue
		}
		txBundle, plan, err := bundle.ValidateEnvelope(b.signer, tx)
		if err != nil {
			continue
		}
		hash := plan.Hash()
		for _, inner := range txBundle.Transactions {
			if bundle.IsEnvelope(inner) {
				return nil, fmt.Errorf("epoch %d contains nested bundles (plan %x in block %d), which heal does not support; "+
					"heal to epoch %d, before them", b.epoch, hash, b.number, b.epoch)
			}
		}
		plans[hash] = plan
		if b.isExecuted(hash, plan, marks) {
			marks[hash] = bundle.PositionInBlock{}
		}
	}

	// Plans whose transactions are in the block were executed in this block.
	positions := map[common.Hash][]int{}
	for i, tx := range b.blockTx {
		for _, hash := range bundle.GetApprovedExecutionPlans(tx) {
			positions[hash] = append(positions[hash], i)
		}
	}
	for hash := range positions {
		if _, found := marks[hash]; found {
			continue
		}
		if plan, found := plans[hash]; found && plan.Range.First < b.first {
			continue
		}
		return nil, fmt.Errorf("block %d holds transactions of plan %x not marked as processed", b.number, hash)
	}

	// Positions only serve RPC queries. Reverted bundles have no transactions
	// and keep a zero position.
	for hash := range marks {
		if idxs := positions[hash]; len(idxs) > 0 {
			marks[hash] = bundle.PositionInBlock{
				Offset: uint32(slices.Min(idxs)),
				Count:  uint32(slices.Max(idxs) - slices.Min(idxs) + 1),
			}
		}
	}
	return marks, nil
}

// isExecuted reports whether a valid top-level bundle with the given plan gets
// executed, and thus marked, in the scanned block.
func (b *blockScan) isExecuted(hash common.Hash, plan *bundle.ExecutionPlan, marks map[common.Hash]bundle.PositionInBlock) bool {
	if plan.Range.First < b.first {
		return false
	}
	if !plan.Range.IsInRange(b.number) || !plan.Period.IsInPeriod(b.time) {
		return false
	}
	_, inBlock := marks[hash]
	_, before := b.marked[hash]
	return !inBlock && !before
}
