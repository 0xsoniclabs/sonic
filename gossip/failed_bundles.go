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
	"math/big"
	"slices"
	"sort"

	"github.com/Fantom-foundation/lachesis-base/abft"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/Fantom-foundation/lachesis-base/kvdb/memorydb"
	"github.com/Fantom-foundation/lachesis-base/lachesis"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/evmcore/core_types"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/gossip/scrambler"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils/adapters/vecmt2dagidx"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/0xsoniclabs/sonic/vecmt"
)

// FailedBundle is a bundle envelope, possibly nested in another one, whose
// failure may have changed the state through its own transactions that are not
// in the block. Before Canto, a failed bundle adds no transactions to the
// block, but only its groups revert the state changes of their transactions.
type FailedBundle struct {
	Block idx.Block
	// Pos is the number of block transactions executed before the envelope.
	Pos      int
	Envelope *types.Transaction
}

// FindFailedBundles returns the failed bundles of the blocks from the given
// start, but not before the activation of transaction bundles, until Canto, in
// execution order. The blocks of epochs with envelopes are rebuilt by replaying
// the consensus on the stored events, and the processed bundles are checked
// against the bundle history hashes. A later start also replays the epochs of
// the bundle.MaxBlockRangeLength-1 blocks before it to recognize rescheduled
// bundles; the history hash is then taken from the end of the first epoch.
func (s *Store) FindFailedBundles(start idx.Block) ([]FailedBundle, error) {
	first, last, found := bundleBlockRange(s.GetUpgradeHeights(), s.GetLatestBlockIndex())
	if !found || start > last {
		return nil, nil
	}
	start = max(start, first)

	firstEpoch, startEpoch := s.FindBlockEpoch(first), s.FindBlockEpoch(first)
	if start > first+idx.Block(bundle.MaxBlockRangeLength-1) {
		startEpoch = s.FindBlockEpoch(start - idx.Block(bundle.MaxBlockRangeLength-1))
		// The blocks of the start epoch are not checked against history hashes,
		// so it must not contain exported ones.
		if startEpoch == s.FindBlockEpoch(start) {
			startEpoch--
		}
	}

	var (
		failed      []FailedBundle
		processed   = map[common.Hash]bool{}
		historyHash common.Hash
		hashKnown   = startEpoch <= firstEpoch
	)
	for epoch := max(startEpoch, firstEpoch); ; epoch++ {
		bs, es := s.GetHistoryBlockEpochState(epoch)
		if bs == nil {
			return nil, fmt.Errorf("missing state of epoch %d", epoch)
		}
		from, to := bs.LastBlock.Idx+1, s.GetLatestBlockIndex()
		nextBs, nextEs := s.GetHistoryBlockEpochState(epoch + 1)
		if nextBs != nil {
			to = nextBs.LastBlock.Idx
		}

		var plans map[idx.Block][]common.Hash
		if from <= to && s.epochHasEnvelopes(epoch) {
			if es.Rules.Upgrades.SingleProposerBlockFormation || es.Rules.Upgrades.TransactionPriorities {
				return nil, fmt.Errorf("block formation of epoch %d is not supported", epoch)
			}
			epochFailed, epochPlans, err := s.findFailedBundlesOfEpoch(epoch, bs, es, to, processed)
			if err != nil {
				return nil, err
			}
			epochFailed = slices.DeleteFunc(epochFailed, func(f FailedBundle) bool { return f.Block < start })
			failed, plans = append(failed, epochFailed...), epochPlans
			log.Info("Searched epoch for failed bundles", "epoch", epoch, "blocks", fmt.Sprintf("%d-%d", from, to), "failed", len(epochFailed), "total", len(failed))
		}

		for n := max(from, first); n <= min(to, last); n++ {
			if !hashKnown && nextEs != nil && n == to {
				historyHash, hashKnown = common.Hash(nextEs.EpochEndExecutionPlanChainHash), true
			}
			if !hashKnown {
				continue
			}
			if historyHash != (common.Hash{}) || len(plans[n]) > 0 {
				added := common.Hash{}
				for _, plan := range plans[n] {
					added = xorHash(added, plan)
				}
				historyHash = computeNewBundleStateHash(historyHash, added, uint64(n))
			}
			if stored, ok := s.GetProcessedBundleHistoryHash(uint64(n)); ok && stored != historyHash {
				return nil, fmt.Errorf("bundle history hash of block %d is %v, but %v is stored", n, historyHash, stored)
			}
			// Epochs are sealed with the history hash of the block before the sealing block.
			if nextEs != nil && n+1 == to && historyHash != common.Hash(nextEs.EpochEndExecutionPlanChainHash) {
				return nil, fmt.Errorf("bundle history hash of block %d is %v, but epoch %d was sealed with %v", n, historyHash, epoch, common.Hash(nextEs.EpochEndExecutionPlanChainHash))
			}
		}
		if to >= last {
			return failed, nil
		}
	}
}

// bundleBlockRange returns the blocks from the activation of transaction
// bundles until Canto or the given latest block.
func bundleBlockRange(heights []opera.UpgradeHeight, latest idx.Block) (first, last idx.Block, found bool) {
	last = latest
	for _, height := range heights {
		if height.Upgrades.Canto {
			last = min(last, height.Height-1)
			break
		}
		if !found && height.Upgrades.TransactionBundles {
			first, found = max(height.Height, 1), true
		}
	}
	return first, last, found && first <= last
}

// epochHasEnvelopes reports whether one of the events of the given epoch
// carries a bundle envelope.
func (s *Store) epochHasEnvelopes(epoch idx.Epoch) (found bool) {
	s.ForEachEpochEvent(epoch, func(e *inter.EventPayload) bool {
		found = slices.ContainsFunc(e.Transactions(), bundle.IsEnvelope)
		return !found
	})
	return found
}

// findFailedBundlesOfEpoch rebuilds the blocks of the given epoch up to the
// last one and returns their failed bundles and the execution plans processed
// in each block. The processed plans are added to processed.
func (s *Store) findFailedBundlesOfEpoch(
	epoch idx.Epoch,
	bs *iblockproc.BlockState,
	es *iblockproc.EpochState,
	last idx.Block,
	processed map[common.Hash]bool,
) ([]FailedBundle, map[idx.Block][]common.Hash, error) {
	blocks, err := s.replayConsensus(epoch, es.Validators)
	if err != nil {
		return nil, nil, err
	}

	rules := es.Rules
	signer := types.LatestSignerForChainID(new(big.Int).SetUint64(rules.NetworkID))
	var failed []FailedBundle
	plans := map[idx.Block][]common.Hash{}
	n, lastTime := bs.LastBlock.Idx+1, bs.LastBlock.Time
	for _, b := range blocks {
		if n > last {
			break
		}
		blockTime := max(b.time, lastTime+1)
		if b.degenerate || (len(b.events) == 0 && blockTime < lastTime+rules.Blocks.MaxEmptyBlockSkipPeriod) {
			continue
		}
		block := s.GetBlock(n)
		if block == nil {
			return nil, nil, fmt.Errorf("missing block %d", n)
		}
		if block.Time != blockTime {
			return nil, nil, fmt.Errorf("consensus replay of epoch %d yields time %d for block %d, but it has time %d", epoch, blockTime, n, block.Time)
		}
		included := s.GetBlockTxs(n, block)
		receipts, _ := s.EvmStore().GetRawReceipts(n)
		if len(receipts) != len(included) {
			return nil, nil, fmt.Errorf("block %d has %d transactions, but %d receipts", n, len(included), len(receipts))
		}
		replay := &bundleReplay{
			included:  included,
			receipts:  receipts,
			signer:    signer,
			upgrades:  rules.Upgrades,
			number:    n,
			time:      block.Time,
			processed: processed,
		}
		blockFailed, err := replay.findFailedBundles(s.orderBlockTransactions(b.events, rules, signer))
		if err != nil {
			return nil, nil, err
		}
		failed = append(failed, blockFailed...)
		plans[n] = replay.plans
		lastTime = block.Time
		n++
	}
	if n <= last {
		return nil, nil, fmt.Errorf("consensus replay of epoch %d yields blocks up to %d, but the epoch ends at block %d", epoch, n-1, last)
	}
	return failed, plans, nil
}

// consensusBlock is a block decided by the consensus.
type consensusBlock struct {
	time       inter.Timestamp
	degenerate bool
	events     hash.OrderedEvents // confirmed events with transactions or proposals
}

// replayConsensus decides the blocks of the given epoch from its stored events.
func (s *Store) replayConsensus(epoch idx.Epoch, validators *pos.Validators) (blocks []consensusBlock, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("consensus replay of epoch %d failed: %v", epoch, r)
		}
	}()
	crit := func(err error) { panic(err) }
	dagIndex := vecmt.NewIndex(crit, vecmt.LiteConfig())
	dagIndex.Reset(validators, memorydb.New(), func(id hash.Event) dag.Event { return s.GetEvent(id) })
	engine := abft.NewLachesis(abft.NewMemStore(), eventSource{s}, vecmt2dagidx.Wrap(dagIndex), crit, abft.DefaultConfig())
	err = engine.StartFrom(lachesis.ConsensusCallbacks{
		BeginBlock: func(cBlock *lachesis.Block) lachesis.BlockCallbacks {
			b := consensusBlock{degenerate: true}
			return lachesis.BlockCallbacks{
				ApplyEvent: func(_e dag.Event) {
					e := _e.(inter.EventI)
					if e.ID() == cBlock.Atropos {
						b.time = e.MedianTime()
						b.degenerate = false
					}
					if e.AnyTxs() || e.HasProposal() {
						b.events = append(b.events, e.ID())
					}
				},
				EndBlock: func() *pos.Validators {
					blocks = append(blocks, b)
					return nil
				},
			}
		},
	}, epoch, validators)
	if err != nil {
		return nil, err
	}
	s.ForEachEpochEvent(epoch, func(e *inter.EventPayload) bool {
		if err = dagIndex.Add(e); err != nil {
			return false
		}
		dagIndex.Flush()
		err = engine.Process(e)
		return err == nil
	})
	return blocks, err
}

// eventSource provides the stored events to the consensus.
type eventSource struct {
	*Store
}

func (s eventSource) GetEvent(id hash.Event) dag.Event {
	if e := s.Store.GetEvent(id); e != nil {
		return e
	}
	return nil
}

// orderBlockTransactions returns the user transactions of a block formed from
// the given confirmed events, in the order the block processing runs them.
func (s *Store) orderBlockTransactions(events hash.OrderedEvents, rules opera.Rules, signer types.Signer) types.Transactions {
	sort.Sort(events)
	blockEvents := spillBlockEvents(events, rules.Blocks.MaxBlockGas, func(id hash.Event) inter.EventPayloadI {
		return s.GetEventPayload(id)
	})
	var txs types.Transactions
	for _, e := range blockEvents {
		txs = append(txs, e.Transactions()...)
	}
	txs = scrambler.GetExecutionOrder(txs, signer, rules.Upgrades.Sonic)
	return filterNonPermissibleTransactions(txs, &rules, signer, nil, nil)
}

// bundleReplay runs the user transactions of a block along the transactions
// included in it, executing bundles like the block processing.
type bundleReplay struct {
	included  types.Transactions
	receipts  []*types.ReceiptForStorage
	signer    types.Signer
	upgrades  opera.Upgrades
	number    idx.Block
	time      inter.Timestamp
	processed map[common.Hash]bool // plans processed so far, including this block's
	next      int                  // index of the next included transaction
	plans     []common.Hash        // plans processed in this block
	missing   int                  // transactions of running bundles that are not included
	failed    []FailedBundle       // failed bundles of this block
	snapshots []bundleReplaySnapshot
}

type bundleReplaySnapshot struct{ next, plans, missing, failed int }

// findFailedBundles runs the given ordered user transactions of the block and
// returns its failed bundles.
func (r *bundleReplay) findFailedBundles(ordered types.Transactions) ([]FailedBundle, error) {
	for r.next < len(r.included) && internaltx.IsInternal(r.included[r.next]) {
		r.next++
	}
	for _, tx := range ordered {
		r.Run(tx)
	}
	for _, tx := range r.included[r.next:] {
		if !internaltx.IsInternal(tx) {
			return nil, fmt.Errorf("transaction %v of block %d is not reproduced", tx.Hash(), r.number)
		}
	}
	return r.failed, nil
}

// Run runs a bundle envelope or consumes the next included transaction, along
// the internal ones following it, if it is the given one. Transactions that are
// not included count as failed.
func (r *bundleReplay) Run(tx *types.Transaction) core_types.TransactionResult {
	if bundle.IsEnvelope(tx) {
		return r.runEnvelope(tx)
	}
	if r.next == len(r.included) || r.included[r.next].Hash() != tx.Hash() {
		r.missing++
		return core_types.TransactionResultFailed
	}
	r.next++
	status := r.receipts[r.next-1].Status
	// Sponsored transactions are followed by internal fee payments.
	for r.next < len(r.included) && internaltx.IsInternal(r.included[r.next]) {
		r.next++
	}
	if status == types.ReceiptStatusFailed {
		return core_types.TransactionResultFailed
	}
	return core_types.TransactionResultSuccessful
}

// runEnvelope runs the bundle of the given envelope. If it fails, its own
// missing transactions that were not reverted may have changed the state.
// Those of nested envelopes are not counted, as nested failed envelopes are
// recorded themselves.
func (r *bundleReplay) runEnvelope(tx *types.Transaction) core_types.TransactionResult {
	txBundle, plan, err := bundle.ValidateEnvelope(r.signer, tx)
	if err != nil || !plan.Range.IsInRange(uint64(r.number)) || !plan.Period.IsInPeriod(r.time) || r.processed[plan.Hash()] {
		return core_types.TransactionResultInvalid
	}
	next, missing := r.next, r.missing
	success := bundle.RunBundle(txBundle, r, r.upgrades)
	r.processed[plan.Hash()] = true
	r.plans = append(r.plans, plan.Hash())
	if success {
		r.missing = missing
		return core_types.TransactionResultSuccessful
	}
	if r.missing > missing {
		r.failed = append(r.failed, FailedBundle{Block: r.number, Pos: next, Envelope: tx})
	}
	// The transactions of a failed bundle are not added to the block.
	r.next, r.missing = next, missing
	return core_types.TransactionResultFailed
}

func (r *bundleReplay) CreateSnapshot() int {
	r.snapshots = append(r.snapshots, bundleReplaySnapshot{r.next, len(r.plans), r.missing, len(r.failed)})
	return len(r.snapshots) - 1
}

func (r *bundleReplay) RevertToSnapshot(id int) {
	s := r.snapshots[id]
	for _, plan := range r.plans[s.plans:] {
		delete(r.processed, plan)
	}
	r.next, r.plans, r.missing, r.failed, r.snapshots = s.next, r.plans[:s.plans], s.missing, r.failed[:s.failed], r.snapshots[:id]
}
