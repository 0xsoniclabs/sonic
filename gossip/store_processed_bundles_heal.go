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
	"maps"
	"slices"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// RestoreProcessedBundles reverts the processed bundles to the state they had
// after the given epoch-sealing block, without executing any block. The
// replay protection window is rebuilt from the block proposals in the stored
// events, cross-checked against the stored blocks, and the history hash is
// derived from anchor, the history hash recorded by the epoch sealed in that
// block. Nothing is written if the window cannot be rebuilt reliably.
func (s *Store) RestoreProcessedBundles(sealingBlock idx.Block, anchor common.Hash) error {
	s.processedBundleMutex.Lock()
	defer s.processedBundleMutex.Unlock()

	last := uint64(sealingBlock)
	if last > 0 {
		if retained, found := s.GetProcessedBundleHistoryHash(last - 1); found && retained != anchor {
			return fmt.Errorf("bundle history hash of block %d is %x, epoch records %x", last-1, retained, anchor)
		}
	}

	marks, err := s.collectProcessedBundles(last)
	if err != nil {
		return err
	}

	// The table is replaced as a whole: the window from the scan, the history
	// hash from the anchor and the marks of the sealing block.
	batch := s.table.ProcessedBundles.NewBatch()
	if err := deleteAll(s.table.ProcessedBundles, batch); err != nil {
		return err
	}

	var addedHash common.Hash
	for _, block := range slices.Sorted(maps.Keys(marks)) {
		added := s.addNewBundles(block, marks[block], batch)
		if block == last {
			addedHash = added
		}
	}

	if anchor != (common.Hash{}) || len(marks[last]) > 0 {
		newHash := computeNewBundleStateHash(anchor, addedHash, last)
		if retained, found := s.GetProcessedBundleHistoryHash(last); found && retained != newHash {
			return fmt.Errorf("rebuilt bundle history hash of block %d is %x, store holds %x", last, newHash, retained)
		}
		if err := putBundleHistoryHash(batch, last, newHash); err != nil {
			return err
		}
	} else if len(marks) > 0 {
		return fmt.Errorf("bundles processed before block %d but its epoch records no bundle history", last)
	}

	return batch.Write()
}

// collectProcessedBundles returns, per block, the execution plans marked as
// processed in the replay protection window ending at the given block. Plans
// whose range starts before the window are skipped: they are out of range for
// every block after it.
func (s *Store) collectProcessedBundles(last uint64) (map[uint64]map[common.Hash]bundle.PositionInBlock, error) {
	first := uint64(1) // < the genesis block holds no bundles
	if last >= bundle.MaxBlockRangeLength {
		first = last - (bundle.MaxBlockRangeLength - 1)
	}

	marked := map[common.Hash]uint64{}
	result := map[uint64]map[common.Hash]bundle.PositionInBlock{}
	proposals := map[idx.Epoch]map[idx.Block][]*inter.Proposal{}

	for number := first; number <= last; number++ {
		block := s.GetBlock(idx.Block(number))
		if block == nil {
			return nil, fmt.Errorf("block %d is not available", number)
		}
		es := s.GetHistoryEpochState(block.Epoch)
		if es == nil {
			return nil, fmt.Errorf("epoch %d of block %d is not available", block.Epoch, number)
		}
		upgrades := es.Rules.Upgrades
		if !upgrades.Brio || !upgrades.TransactionBundles {
			continue
		}
		if !upgrades.SingleProposerBlockFormation {
			return nil, fmt.Errorf("block %d predates single-proposer block formation", number)
		}
		if _, found := proposals[block.Epoch]; !found {
			proposals[block.Epoch] = s.getEpochProposals(block.Epoch)
		}
		parent := s.GetBlock(idx.Block(number - 1))
		if parent == nil {
			return nil, fmt.Errorf("block %d is not available", number-1)
		}

		scan := blockScan{
			number:  number,
			first:   first,
			last:    last,
			time:    block.Time,
			rules:   &es.Rules,
			signer:  types.LatestSignerForChainID(s.GetEvmChainConfig(idx.Block(number)).ChainID),
			marked:  marked,
			blockTx: s.GetBlockTxs(idx.Block(number), block),
		}
		marks, err := scan.resolve(proposals[block.Epoch][idx.Block(number)], parent.Hash())
		if err != nil {
			return nil, err
		}
		if len(marks) > 0 {
			result[number] = marks
			for plan := range marks {
				marked[plan] = number
			}
		}
	}
	return result, nil
}

// getEpochProposals returns the block proposals in the events of the given
// epoch, indexed by the proposed block number.
func (s *Store) getEpochProposals(epoch idx.Epoch) map[idx.Block][]*inter.Proposal {
	res := map[idx.Block][]*inter.Proposal{}
	s.ForEachEpochEvent(epoch, func(e *inter.EventPayload) bool {
		if p := e.Payload().Proposal; p != nil {
			res[p.Number] = append(res[p.Number], p)
		}
		return true
	})
	return res
}

// blockScan determines the execution plans marked as processed in one block.
type blockScan struct {
	number, first, last uint64
	time                inter.Timestamp
	rules               *opera.Rules
	signer              types.Signer
	marked              map[common.Hash]uint64 // < plans marked in earlier blocks
	blockTx             types.Transactions
}

// resolve returns the plans marked in the block, using the proposals for it
// that are consistent with the stored block. Proposals that are consistent but
// lead to different marks make the block ambiguous.
func (b *blockScan) resolve(candidates []*inter.Proposal, parent common.Hash) (map[common.Hash]bundle.PositionInBlock, error) {
	var (
		result   map[common.Hash]bundle.PositionInBlock
		found    bool
		firstErr error
	)
	for _, p := range candidates {
		if p.ParentHash != parent {
			continue
		}
		marks, err := b.scan(p.Transactions)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if found && !maps.Equal(result, marks) {
			return nil, fmt.Errorf("block %d has proposals leading to different processed bundles", b.number)
		}
		result, found = marks, true
	}
	if found {
		return result, nil
	}
	if firstErr != nil {
		return nil, firstErr
	}
	// Without a proposal, the block holds no user transactions.
	return b.scan(nil)
}

// nestedPlan is an execution plan reached through an envelope inside a bundle.
type nestedPlan struct {
	hash  common.Hash
	plan  *bundle.ExecutionPlan
	outer common.Hash // < top-level plan containing it
}

// scan marks the top-level bundles of the given proposal transactions passing
// the checks performed before a bundle is executed, and cross-checks the
// result with the bundle-only markers of the stored block.
func (b *blockScan) scan(proposed []*types.Transaction) (map[common.Hash]bundle.PositionInBlock, error) {
	txs := filterNonPermissibleTransactions(slices.Clone(proposed), b.rules, b.signer, nil, nil)

	marks := map[common.Hash]bundle.PositionInBlock{}
	plans := map[common.Hash]*bundle.ExecutionPlan{}
	var nested []nestedPlan
	for _, tx := range txs {
		if !bundle.IsEnvelope(tx) {
			continue
		}
		txBundle, plan, err := bundle.ValidateEnvelope(b.signer, tx)
		if err != nil {
			continue
		}
		hash := plan.Hash()
		plans[hash] = plan
		if !b.isExecuted(hash, plan, marks) {
			continue
		}
		marks[hash] = bundle.PositionInBlock{}
		nested = append(nested, b.nestedPlans(txBundle, hash)...)
	}

	// Plans whose transactions are in the block were executed in this block.
	positions := map[common.Hash][]int{}
	for i, tx := range b.blockTx {
		for _, hash := range bundle.GetApprovedExecutionPlans(tx) {
			positions[hash] = append(positions[hash], i)
		}
	}
	for _, n := range nested {
		plans[n.hash] = n.plan
		if !b.isExecuted(n.hash, n.plan, marks) {
			continue
		}
		if _, found := positions[n.hash]; found {
			marks[n.hash] = bundle.PositionInBlock{}
			continue
		}
		// ponytail: inner marks without transactions in the block are not
		// derived from the plan tree; heal refuses when one can matter.
		if lastBlockOf(n.plan.Range) >= b.last {
			return nil, fmt.Errorf("block %d contains nested plan %x whose processing cannot be determined", b.number, n.hash)
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

	// Positions only serve RPC queries. A bundle covers the contiguous range of
	// its own transactions and those of the bundles nested in it; reverted
	// bundles have no transactions and keep a zero position.
	members := map[common.Hash][]common.Hash{}
	for _, n := range nested {
		members[n.outer] = append(members[n.outer], n.hash)
	}
	for hash := range marks {
		var idxs []int
		for _, member := range append(members[hash], hash) {
			idxs = append(idxs, positions[member]...)
		}
		if len(idxs) > 0 {
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

// nestedPlans lists the valid plans of envelopes nested at any depth in the
// given bundle.
func (b *blockScan) nestedPlans(txBundle *bundle.TransactionBundle, outer common.Hash) []nestedPlan {
	var res []nestedPlan
	for _, tx := range txBundle.Transactions {
		if !bundle.IsEnvelope(tx) {
			continue
		}
		inner, plan, err := bundle.ValidateEnvelope(b.signer, tx)
		if err != nil {
			continue
		}
		res = append(res, nestedPlan{hash: plan.Hash(), plan: plan, outer: outer})
		res = append(res, b.nestedPlans(inner, outer)...)
	}
	return res
}

func deleteAll(table kvdb.Store, batch kvdb.Batch) error {
	it := table.NewIterator(nil, nil)
	defer it.Release()
	for it.Next() {
		if err := batch.Delete(slices.Clone(it.Key())); err != nil {
			return err
		}
	}
	return it.Error()
}

func lastBlockOf(r bundle.BlockRange) uint64 {
	return r.First + r.Length - 1
}

func putBundleHistoryHash(batch kvdb.Batch, block uint64, hash common.Hash) error {
	if err := batch.Put(nil, append(bigendian.Uint64ToBytes(block), hash.Bytes()...)); err != nil {
		return err
	}
	return batch.Put(getBundleHistoryHashKey(block), hash.Bytes())
}
