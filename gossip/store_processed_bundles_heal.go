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
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/gossip/scrambler"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils/adapters/vecmt2dagidx"
	"github.com/0xsoniclabs/sonic/utils/caution"
	"github.com/0xsoniclabs/sonic/vecmt"
	"github.com/Fantom-foundation/lachesis-base/abft"
	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/memorydb"
	"github.com/Fantom-foundation/lachesis-base/lachesis"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// RestoreProcessedBundles reverts the processed bundles to the state they had
// at the start of the given epoch, without executing any block. The replay
// protection window is rebuilt by replaying the consensus over the stored
// events, cross-checked against the stored blocks, and the history hash is
// derived from the one recorded by the epoch, after reproducing it from the
// previous epoch's. Histories reaching back to the genesis are refused, as a
// re-sync is cheaper there. Nothing is written if the window cannot be rebuilt
// reliably.
func (s *Store) RestoreProcessedBundles(epoch idx.Epoch) error {
	s.processedBundleMutex.Lock()
	defer s.processedBundleMutex.Unlock()

	bs, es := s.GetHistoryBlockEpochState(epoch)
	if bs == nil || es == nil {
		return fmt.Errorf("epoch %d is not available", epoch)
	}
	last := uint64(bs.LastBlock.Idx)
	anchor := common.Hash(es.EpochEndExecutionPlanChainHash)

	// The table is replaced as a whole.
	batch := s.table.ProcessedBundles.NewBatch()
	if err := deleteAll(s.table.ProcessedBundles, batch); err != nil {
		return err
	}
	if anchor == (common.Hash{}) && !es.Rules.Upgrades.TransactionBundles {
		return batch.Write() // < no bundle history to restore
	}

	// The window of the previous epoch's seal is included to verify the
	// anchor; it must lie past the genesis, which has no events.
	var genesis uint64
	if g := s.GetGenesisBlockIndex(); g != nil {
		genesis = uint64(*g)
	}
	prevBs, prevEs := s.GetHistoryBlockEpochState(epoch - 1)
	if prevBs == nil || prevEs == nil || uint64(prevBs.LastBlock.Idx) < genesis+bundle.MaxBlockRangeLength {
		return fmt.Errorf("the bundle history of epoch %d reaches back to the genesis block %d, re-sync the node instead of healing", epoch, genesis)
	}
	prevLast := uint64(prevBs.LastBlock.Idx)

	marks, err := s.collectProcessedBundles(prevLast-(bundle.MaxBlockRangeLength-1), last)
	if err != nil {
		return err
	}

	h := common.Hash(prevEs.EpochEndExecutionPlanChainHash)
	for block := prevLast; block < last; block++ {
		h = nextBundleHistoryHash(h, marks[block], block)
	}
	if h != anchor {
		return fmt.Errorf("epoch %d records bundle history hash %x, the blocks since epoch %d lead to %x; "+
			"the node diverged before epoch %d, heal to an earlier epoch", epoch, anchor, epoch-1, h, epoch)
	}

	// The window from the scan, the history hash from the anchor and the
	// marks of the sealing block.
	for _, block := range slices.Sorted(maps.Keys(marks)) {
		if block+bundle.MaxBlockRangeLength > last {
			s.addNewBundles(block, marks[block], batch)
		}
	}

	if newHash := nextBundleHistoryHash(anchor, marks[last], last); newHash != (common.Hash{}) {
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

// nextBundleHistoryHash returns the history hash after a block marking the
// given plans, following AddProcessedBundles.
func nextBundleHistoryHash(prev common.Hash, marks map[common.Hash]bundle.PositionInBlock, block uint64) common.Hash {
	if prev == (common.Hash{}) && len(marks) == 0 {
		return prev
	}
	var added common.Hash
	for hash := range marks {
		added = xorHash(added, hash)
	}
	return computeNewBundleStateHash(prev, added, block)
}

// collectProcessedBundles returns, per block, the execution plans marked as
// processed in the blocks [first, last]. Plans whose range starts before first
// are skipped: they are out of range for every block after the window.
func (s *Store) collectProcessedBundles(first, last uint64) (map[uint64]map[common.Hash]bundle.PositionInBlock, error) {
	start, end := s.GetBlock(idx.Block(first)), s.GetBlock(idx.Block(last))
	if start == nil || end == nil {
		return nil, fmt.Errorf("blocks %d to %d are not available", first, last)
	}

	marked := map[common.Hash]uint64{}
	result := map[uint64]map[common.Hash]bundle.PositionInBlock{}
	visit := func(number uint64, time inter.Timestamp, rules *opera.Rules, signer types.Signer, txs types.Transactions) error {
		block := s.GetBlock(idx.Block(number))
		if block == nil {
			return fmt.Errorf("block %d is not available", number)
		}
		if block.Time != time {
			return fmt.Errorf("consensus replay yields block %d at %v, the store holds it at %v", number, time, block.Time)
		}
		scan := blockScan{
			epoch:   block.Epoch,
			number:  number,
			first:   first,
			time:    time,
			rules:   rules,
			signer:  signer,
			marked:  marked,
			blockTx: s.GetBlockTxs(idx.Block(number), block),
		}
		marks, err := scan.scan(txs)
		if err != nil {
			return err
		}
		if len(marks) > 0 {
			result[number] = marks
			for plan := range marks {
				marked[plan] = number
			}
		}
		return nil
	}

	for epoch := start.Epoch; epoch <= end.Epoch; epoch++ {
		if err := s.replayEpochBlocks(epoch, first, last, visit); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// replayEpochBlocks replays the consensus over the stored events of the given
// epoch and passes the user transactions of each of its blocks within
// [first, last] to visit, in block order. Block formation follows
// consensusCallbackBeginBlockFn up to the execution of the transactions.
// Epochs without transaction bundles are skipped.
func (s *Store) replayEpochBlocks(
	epoch idx.Epoch,
	first, last uint64,
	visit func(number uint64, time inter.Timestamp, rules *opera.Rules, signer types.Signer, txs types.Transactions) error,
) (err error) {
	bs, es := s.GetHistoryBlockEpochState(epoch)
	next, _ := s.GetHistoryBlockEpochState(epoch + 1)
	if bs == nil || es == nil || next == nil {
		return fmt.Errorf("epoch %d is not available", epoch)
	}
	rules := es.Rules
	if !rules.Upgrades.Brio || !rules.Upgrades.TransactionBundles {
		return nil
	}
	if rules.Upgrades.SingleProposerBlockFormation {
		return fmt.Errorf("epoch %d uses single-proposer block formation, which heal does not support", epoch)
	}

	signer := types.LatestSignerForChainID(s.GetEvmChainConfig(next.LastBlock.Idx).ChainID)
	number, lastTime := uint64(bs.LastBlock.Idx), bs.LastBlock.Time
	end := min(uint64(next.LastBlock.Idx), last)
	var visitErr error
	beginBlock := func(cBlock *lachesis.Block) lachesis.BlockCallbacks {
		atroposTime, degenerate := lastTime+1, true
		var confirmed hash.OrderedEvents
		return lachesis.BlockCallbacks{
			ApplyEvent: func(e dag.Event) {
				event := e.(inter.EventI)
				if cBlock.Atropos == event.ID() {
					atroposTime, degenerate = event.MedianTime(), false
				}
				if event.AnyTxs() || event.HasProposal() {
					confirmed = append(confirmed, event.ID())
				}
			},
			EndBlock: func() *pos.Validators {
				if number >= end || visitErr != nil {
					return nil
				}
				blockTime := max(atroposTime, lastTime+1)
				empty := confirmed.Len() == 0 && cBlock.Cheaters.Len() == 0
				if degenerate || (empty && blockTime < lastTime+rules.Blocks.MaxEmptyBlockSkipPeriod) {
					return nil
				}
				number, lastTime = number+1, blockTime
				if number < first {
					return nil
				}
				sort.Sort(confirmed)
				events := spillBlockEvents(confirmed, rules.Blocks.MaxBlockGas, func(id hash.Event) inter.EventPayloadI {
					return s.GetEventPayload(id)
				})
				var txs types.Transactions
				for _, e := range events {
					txs = append(txs, e.Transactions()...)
				}
				ordered := scrambler.GetExecutionOrder(txs, signer, rules.Upgrades.Sonic)
				visitErr = visit(number, blockTime, &rules, signer, ordered)
				return nil
			},
		}
	}

	crit := func(err error) { panic(fmt.Errorf("consensus replay of epoch %d: %w", epoch, err)) }
	consensusStore := abft.NewMemStore()
	defer caution.CloseAndReportError(&err, consensusStore, "failed to close consensus replay store")
	if err := consensusStore.ApplyGenesis(&abft.Genesis{Epoch: epoch, Validators: es.Validators}); err != nil {
		return err
	}
	dagIndex := vecmt.NewIndex(crit, vecmt.LiteConfig())
	defer caution.CloseAndReportError(&err, dagIndex, "failed to close consensus replay DAG index")
	dagIndex.Reset(es.Validators, memorydb.New(), func(id hash.Event) dag.Event {
		return s.GetEvent(id)
	})
	engine := abft.NewLachesis(consensusStore, eventSource{s}, vecmt2dagidx.Wrap(dagIndex), crit, abft.LiteConfig())
	if err := engine.Bootstrap(lachesis.ConsensusCallbacks{BeginBlock: beginBlock}); err != nil {
		return err
	}

	s.ForEachEpochEvent(epoch, func(e *inter.EventPayload) bool {
		if err = dagIndex.Add(e); err != nil {
			return false
		}
		dagIndex.Flush()
		err = engine.Process(e)
		return err == nil && visitErr == nil && number < end
	})
	if err = errors.Join(err, visitErr); err != nil {
		return err
	}
	if number < end {
		return fmt.Errorf("consensus replay of epoch %d ends at block %d, expected block %d", epoch, number, end)
	}
	return nil
}

// eventSource exposes the stored events to the consensus engine.
type eventSource struct {
	*Store
}

func (s eventSource) GetEvent(id hash.Event) dag.Event {
	if e := s.Store.GetEvent(id); e != nil {
		return e
	}
	return nil
}

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

func putBundleHistoryHash(batch kvdb.Batch, block uint64, hash common.Hash) error {
	if err := batch.Put(nil, append(bigendian.Uint64ToBytes(block), hash.Bytes()...)); err != nil {
		return err
	}
	return batch.Put(getBundleHistoryHashKey(block), hash.Bytes())
}
