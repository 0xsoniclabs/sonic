// Copyright 2025 Sonic Operations Ltd
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
	"bytes"
	"cmp"
	"fmt"
	"math/big"
	"slices"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/scc/cert"
	scc_node "github.com/0xsoniclabs/sonic/scc/node"
	"github.com/0xsoniclabs/sonic/utils/signers/gsignercache"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/Fantom-foundation/lachesis-base/lachesis"
	"github.com/Fantom-foundation/lachesis-base/utils/workers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/verwatcher"
	"github.com/0xsoniclabs/sonic/gossip/emitter"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/gossip/randao"
	"github.com/0xsoniclabs/sonic/gossip/scrambler"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
)

//go:generate mockgen -source=c_block_callbacks.go -package=gossip -destination=c_block_callbacks_mock.go

var (
	// Ethereum compatible metrics set (see go-ethereum/core)

	headBlockGauge     = metrics.GetOrRegisterGauge("chain/head/block", nil)
	headHeaderGauge    = metrics.GetOrRegisterGauge("chain/head/header", nil)
	headFastBlockGauge = metrics.GetOrRegisterGauge("chain/head/receipt", nil)

	blockExecutionTimer             = metrics.GetOrRegisterResettingTimer("chain/execution", nil)
	blockExecutionNonResettingTimer = metrics.GetOrRegisterTimer("chain/execution/nonresetting", nil)
	blockAgeGauge                   = metrics.GetOrRegisterGauge("chain/block/age", nil)

	processedTxsMeter = metrics.GetOrRegisterMeter("chain/txs/processed", nil)
	skippedTxsMeter   = metrics.GetOrRegisterMeter("chain/txs/skipped", nil)
	invalidTxsMeter   = metrics.GetOrRegisterMeter("chain/txs/invalid", nil)

	confirmedEventsMeter = metrics.GetOrRegisterMeter("chain/events/confirmed", nil) // events received from lachesis
	spilledEventsMeter   = metrics.GetOrRegisterMeter("chain/events/spilled", nil)   // tx excluded because of MaxBlockGas
)

type ExtendedTxPosition struct {
	evmstore.TxPosition
	EventCreator idx.ValidatorID
}

// GetConsensusCallbacks returns single (for Service) callback instance.
func (s *Service) GetConsensusCallbacks() lachesis.ConsensusCallbacks {
	return lachesis.ConsensusCallbacks{
		BeginBlock: consensusCallbackBeginBlockFn(
			s.blockProcTasks,
			&s.blockProcWg,
			&s.blockBusyFlag,
			s.store,
			s.blockProcModules,
			s.config.TxIndex,
			&s.feed,
			&s.emitters,
			s.verWatcher,
			&s.bootstrapping,
			s.sccNode,
		),
	}
}

// consensusCallbackBeginBlockFn takes only necessaries for block processing and
// makes lachesis.BeginBlockFn.
func consensusCallbackBeginBlockFn(
	parallelTasks *workers.Workers,
	wg *sync.WaitGroup,
	blockBusyFlag *uint32,
	store *Store,
	blockProc BlockProc,
	txIndex bool,
	feed *ServiceFeed,
	emitters *[]*emitter.Emitter,
	verWatcher *verwatcher.VersionWatcher,
	bootstrapping *bool,
	sccNode *scc_node.Node,
) lachesis.BeginBlockFn {
	return func(cBlock *lachesis.Block) lachesis.BlockCallbacks {
		if *bootstrapping {
			// ignore block processing during bootstrapping
			return lachesis.BlockCallbacks{
				ApplyEvent: func(dag.Event) {},
				EndBlock: func() *pos.Validators {
					return nil
				},
			}
		}
		wg.Wait()
		start := time.Now()

		// Note: take copies to avoid race conditions with API calls
		bs := store.GetBlockState().Copy()
		es := store.GetEpochState().Copy()

		// merge cheaters to ensure that every cheater will get punished even if only previous (not current) Atropos observed a doublesign
		// this feature is needed because blocks may be skipped even if cheaters list isn't empty
		// otherwise cheaters would get punished after a first block where cheaters were observed
		bs.EpochCheaters = mergeCheaters(bs.EpochCheaters, cBlock.Cheaters)

		// Get stateDB
		statedb, err := store.evm.GetLiveStateDb(bs.FinalizedStateRoot)
		if err != nil {
			log.Crit("Failed to open StateDB", "err", err)
		}
		evmStateReader := &EvmStateReader{
			ServiceFeed: feed,
			store:       store,
		}

		eventProcessor := blockProc.EventsModule.Start(bs, es)

		atroposTime := bs.LastBlock.Time + 1
		atroposDegenerate := true
		// events with txs
		confirmedEvents := make(hash.OrderedEvents, 0, 3*es.Validators.Len())

		return lachesis.BlockCallbacks{
			ApplyEvent: func(_e dag.Event) {
				e := _e.(inter.EventI)
				if cBlock.Atropos == e.ID() {
					atroposTime = e.MedianTime()
					atroposDegenerate = false
				}
				if e.AnyTxs() || e.HasProposal() {
					confirmedEvents = append(confirmedEvents, e.ID())
				}
				eventProcessor.ProcessConfirmedEvent(e)
				for _, em := range *emitters {
					em.OnEventConfirmed(e)
				}
				confirmedEventsMeter.Mark(1)
			},
			EndBlock: func() (newValidators *pos.Validators) {

				// sort events by Lamport time
				sort.Sort(confirmedEvents)
				maxBlockGas := es.Rules.Blocks.MaxBlockGas
				blockEvents := spillBlockEvents(confirmedEvents, maxBlockGas,
					func(id hash.Event) inter.EventPayloadI {
						// Note: currently, GetEventPayload returns a pointer to struct,
						// conversion to interface may yield a broken interface if
						// the value is nil.
						// Adding a nil check to return a nil interface would
						// solve that, but at this point in the code, every event
						// must have a known payload, and getting a nil would be a
						// critical error. We will let it panic if that happens,
						// as there is no recovery from it.
						return store.GetEventPayload(id)
					},
				)

				// Start assembling the resulting block.
				number := uint64(bs.LastBlock.Idx + 1)
				lastBlockHeader := evmStateReader.GetHeaderByNumber(number - 1)

				randao := computePrevRandao(confirmedEvents)
				chainCfg := opera.CreateTransientEvmChainConfig(
					es.Rules.NetworkID,
					store.GetUpgradeHeights(),
					idx.Block(number),
				)

				// The maximum amount of gas to be used for non-internal
				// transactions in the resulting block. Note that this gas limit
				// is different than the official BlockGasLimit, which is
				// announced as part of the block, constant over the duration of
				// a block, and must be large enough to include internal
				// transactions. In Sonic, the Block's GasLimit is a network
				// rule parameter.
				// The limit defined here is the dynamically adjusted gas limit
				// used to regulate the traffic on the network. Block proposals
				// made in the single-proposer mode are expected to honor this
				// gas limit. With this parameter, this limit is enforced.
				userTransactionGasLimit := maxBlockGas

				// Get a proposal for the block to be created.
				proposal := inter.Proposal{
					Number:     idx.Block(number),
					ParentHash: lastBlockHeader.Hash,
				}
				var blockTime inter.Timestamp
				if es.Rules.Upgrades.SingleProposerBlockFormation {
					if proposed, proposer, time := extractProposalForNextBlock(lastBlockHeader, blockEvents, log.Root()); proposed != nil {
						proposal = *proposed
						blockTime = time
						validatorKeys := readEpochPubKeys(store, cBlock.Atropos.Epoch())
						randao = resolveRandaoMix(
							proposal.RandaoReveal, proposer,
							validatorKeys.PubKeys,
							lastBlockHeader.PrevRandao, randao,
							log.Root(),
						)

						userTransactionGasLimit = inter.GetEffectiveGasLimit(
							blockTime.Time().Sub(lastBlockHeader.Time.Time()),
							es.Rules.Economy.ShortGasPower.AllocPerSec,
							maxBlockGas,
						)

					} else {
						// If no proposal is found but a block needs to be
						// created (as this function has been called), we
						// use a minimum time span to avoid removing gas
						// allocation time from the next block.
						blockTime = lastBlockHeader.Time + 1
						// in this case, the original event-based randao is used.
					}
				} else {
					// Collect transactions from events and schedule them.
					unorderedTxs := make(types.Transactions, 0, len(blockEvents)*10)
					for _, e := range blockEvents {
						unorderedTxs = append(unorderedTxs, e.Transactions()...)
					}

					signer := gsignercache.Wrap(types.MakeSigner(chainCfg, new(big.Int).SetUint64(number), uint64(atroposTime)))
					proposal.Transactions = scrambler.GetExecutionOrder(unorderedTxs, signer, es.Rules.Upgrades.Sonic)

					blockTime = atroposTime
				}

				// Filter invalid transactions from the proposal.
				proposal.Transactions = filterNonPermissibleTransactions(
					proposal.Transactions, &es.Rules, log.Root(), invalidTxsMeter,
				)

				// Make sure the new block time is after the last block time.
				if blockTime <= bs.LastBlock.Time {
					blockTime = bs.LastBlock.Time + 1
				}

				blockCtx := iblockproc.BlockCtx{
					Idx:     proposal.Number,
					Time:    blockTime,
					Atropos: cBlock.Atropos,
				}

				// Note:
				// it's possible that a previous Atropos observes current Atropos (1)
				// (even stronger statement is true - it's possible that current Atropos is equal to a previous Atropos).
				// (1) is true when and only when ApplyEvent wasn't called.
				// In other words, we should assume that every non-cheater root may be elected as an Atropos in any order,
				// even if typically every previous Atropos happened-before current Atropos
				// We have to skip block in case (1) to ensure that every block ID is unique.
				// If Atropos ID wasn't used as a block ID, it wouldn't be required.
				skipBlock := atroposDegenerate
				// Check if empty block should be pruned
				emptyBlock := confirmedEvents.Len() == 0 && cBlock.Cheaters.Len() == 0
				skipBlock = skipBlock || (emptyBlock && blockCtx.Time < bs.LastBlock.Time+es.Rules.Blocks.MaxEmptyBlockSkipPeriod)
				// Finalize the progress of eventProcessor
				bs = eventProcessor.Finalize(blockCtx, skipBlock) // TODO: refactor to not mutate the bs, it is unclear
				if skipBlock {
					// save the latest block state even if block is skipped
					store.SetBlockEpochState(bs, es)
					log.Debug("Frame is skipped", "atropos", cBlock.Atropos.String())
					return nil
				}

				sealer := blockProc.SealerModule.Start(blockCtx, bs, es)
				sealing := sealer.EpochSealing()
				txListener := blockProc.TxListenerModule.Start(blockCtx, bs, es, statedb)
				onNewLogAll := func(l *types.Log) {
					txListener.OnNewLog(l)
					// Note: it's possible for logs to get indexed twice by BR and block processing
					if verWatcher != nil {
						verWatcher.OnNewLog(l)
					}
				}

				// prepare block processing
				evmProcessor := blockProc.EVMModule.Start(
					blockCtx,
					statedb,
					evmStateReader,
					onNewLogAll,
					es.Rules,
					chainCfg,
					randao,
				)
				executionStart := time.Now()

				// Execute pre-internal transactions
				preInternalTxs := blockProc.PreTxTransactor.PopInternalTxs(blockCtx, bs, es, sealing, statedb)
				preInternalReceipts := evmProcessor.Execute(preInternalTxs, maxBlockGas)
				bs = txListener.Finalize()
				for _, r := range preInternalReceipts {
					if r.Status == 0 {
						log.Warn("Pre-internal transaction reverted", "txid", r.TxHash.String())
					}
				}

				// Seal epoch if requested
				if sealing {
					sealer.Update(bs, es)
					prevUpg := es.Rules.Upgrades
					bs, es = sealer.SealEpoch() // TODO: refactor to not mutate the bs, it is unclear
					if es.Rules.Upgrades != prevUpg {
						store.AddUpgradeHeight(opera.UpgradeHeight{
							Upgrades: es.Rules.Upgrades,
							Height:   blockCtx.Idx + 1,
							Time:     blockCtx.Time + 1,
						})
					}
					store.SetBlockEpochState(bs, es)
					newValidators = es.Validators
					txListener.Update(bs, es)
				}

				// At this point, newValidators may be returned and the rest of the code may be executed in a parallel thread
				blockFn := func() {

					blockDuration := time.Duration(blockCtx.Time - bs.LastBlock.Time)
					blockBuilder := inter.NewBlockBuilder().
						WithEpoch(blockCtx.Atropos.Epoch()).
						WithNumber(number).
						WithParentHash(proposal.ParentHash).
						WithTime(blockCtx.Time).
						WithPrevRandao(randao).
						WithGasLimit(maxBlockGas).
						WithDuration(blockDuration)

					for i := range preInternalTxs {
						blockBuilder.AddTransaction(
							preInternalTxs[i],
							preInternalReceipts[i],
						)
					}

					// Execute post-internal transactions
					internalTxs := blockProc.PostTxTransactor.PopInternalTxs(blockCtx, bs, es, sealing, statedb)
					internalReceipts := evmProcessor.Execute(internalTxs, maxBlockGas)
					for _, r := range internalReceipts {
						if r.Status == 0 {
							log.Warn("Internal transaction reverted", "txid", r.TxHash.String())
						}
					}

					for i := range internalTxs {
						blockBuilder.AddTransaction(
							internalTxs[i],
							internalReceipts[i],
						)
					}

					orderedTxs := proposal.Transactions
					for i, receipt := range evmProcessor.Execute(orderedTxs, userTransactionGasLimit) {
						if receipt != nil { // < nil if skipped
							blockBuilder.AddTransaction(orderedTxs[i], receipt)
						}
					}

					evmBlock, skippedTxs, allReceipts := evmProcessor.Finalize()

					// Add results of the transaction processing to the block.
					blockBuilder.
						WithStateRoot(common.Hash(evmBlock.Root)).
						WithGasUsed(evmBlock.GasUsed).
						WithBaseFee(evmBlock.BaseFee)

					// Complete the block.
					block := blockBuilder.Build()
					evmBlock.Hash = block.Hash()
					evmBlock.Duration = blockDuration

					// Update block-hash and -time values in receipts and logs.
					for i := range allReceipts {
						allReceipts[i].BlockHash = block.Hash()
						for j := range allReceipts[i].Logs {
							allReceipts[i].Logs[j].BlockHash = block.Hash()
							allReceipts[i].Logs[j].BlockTimestamp = uint64(block.Time.Unix())
						}
					}

					// memorize event position of each tx
					txPositions := make(map[common.Hash]ExtendedTxPosition)
					for _, e := range blockEvents {
						for i, tx := range e.Transactions() {
							// If tx was met in multiple events, then assign to first ordered event
							if _, ok := txPositions[tx.Hash()]; ok {
								continue
							}
							txPositions[tx.Hash()] = ExtendedTxPosition{
								TxPosition: evmstore.TxPosition{
									Event:       e.ID(),
									EventOffset: uint32(i),
								},
								EventCreator: e.Creator(),
							}
						}
					}
					// memorize block position of each tx
					for i, tx := range evmBlock.Transactions {
						// not skipped txs only
						position := txPositions[tx.Hash()]
						position.Block = blockCtx.Idx
						position.BlockOffset = uint32(i)
						txPositions[tx.Hash()] = position
					}

					// call OnNewReceipt
					for i, r := range allReceipts {
						creator := txPositions[r.TxHash].EventCreator
						if creator != 0 && es.Validators.Get(creator) == 0 {
							creator = 0
						}
						txListener.OnNewReceipt(evmBlock.Transactions[i], r, creator, evmBlock.BaseFee, evmBlock.BlobBaseFee)
					}
					bs = txListener.Finalize() // TODO: refactor to not mutate the bs
					bs.FinalizedStateRoot = hash.Hash(evmBlock.Root)
					// At this point, block state is finalized

					// Build index for not skipped txs
					if txIndex {
						for _, tx := range evmBlock.Transactions {
							// not skipped txs only
							store.evm.SetTxPosition(tx.Hash(), txPositions[tx.Hash()].TxPosition)
						}

						// Index receipts
						// Note: it's possible for receipts to get indexed twice by BR and block processing
						if allReceipts.Len() != 0 {
							store.evm.SetReceipts(blockCtx.Idx, allReceipts)
							for _, r := range allReceipts {
								store.evm.IndexLogs(r.Logs...)
							}
						}
					}

					bs.LastBlock = blockCtx
					bs.CheatersWritten = uint32(bs.EpochCheaters.Len())
					if sealing {
						store.SetHistoryBlockEpochState(es.Epoch, bs, es)
						store.SetEpochBlock(blockCtx.Idx+1, es.Epoch)
					}

					for _, tx := range blockBuilder.GetTransactions() {
						store.evm.SetTx(tx.Hash(), tx)
					}

					store.SetBlock(blockCtx.Idx, block)
					store.SetBlockIndex(block.Hash(), blockCtx.Idx)
					store.SetBlockEpochState(bs, es)
					store.EvmStore().SetCachedEvmBlock(blockCtx.Idx, evmBlock)

					// Inform the SCC about the new block
					if sccNode != nil {
						err := sccNode.NewBlock(cert.NewBlockStatement(
							chainCfg.ChainID.Uint64(),
							blockCtx.Idx,
							block.Hash(),
							block.StateRoot,
						))
						if err != nil {
							log.Warn("Failed to inform SCC about new block", "err", err)
						}
					}

					// Update the metrics touched during block processing
					executionTime := time.Since(executionStart)
					blockExecutionTimer.Update(executionTime)
					blockExecutionNonResettingTimer.Update(executionTime)

					// Update the metrics touched by new block
					headBlockGauge.Update(int64(blockCtx.Idx))
					headHeaderGauge.Update(int64(blockCtx.Idx))
					headFastBlockGauge.Update(int64(blockCtx.Idx))

					// Notify about new block
					if feed != nil {
						var logs []*types.Log
						for _, r := range allReceipts {
							logs = append(logs, r.Logs...)
						}
						feed.notifyAboutNewBlock(evmBlock, logs)
					}

					now := time.Now()
					blockAge := now.Sub(block.Time.Time())
					log.Info("New block",
						"index", blockCtx.Idx,
						"id", block.Hash(),
						"gas_used", evmBlock.GasUsed,
						"gas_rate", float64(evmBlock.GasUsed)/blockDuration.Seconds(),
						"base_fee", evmBlock.BaseFee.String(),
						"txs", fmt.Sprintf("%d/%d", len(evmBlock.Transactions), len(skippedTxs)),
						"age", utils.PrettyDuration(blockAge),
						"t", utils.PrettyDuration(now.Sub(start)),
						"epoch", evmBlock.Epoch,
					)
					blockAgeGauge.Update(int64(blockAge.Nanoseconds()))

					processedTxsMeter.Mark(int64(len(evmBlock.Transactions)))
					skippedTxsMeter.Mark(int64(len(skippedTxs)))
				}
				if confirmedEvents.Len() != 0 {
					atomic.StoreUint32(blockBusyFlag, 1)
					wg.Add(1)
					err := parallelTasks.Enqueue(func() {
						defer atomic.StoreUint32(blockBusyFlag, 0)
						defer wg.Done()
						blockFn()
					})
					if err != nil {
						panic(err)
					}
				} else {
					blockFn()
				}

				return newValidators
			},
		}
	}
}

// resolveRandaoMix computes the randao mix to be used by the block processor
// when using single block proposal.
//
// If randao reveal cannot be verified, this block will be computed using the
// event derived randao value. This can happen if the randao reveal value
// was not created according to specification. This fallback mechanism will
// increase the entropy of the system by introducing an un-biased random value
// reproducible by all nodes.
func resolveRandaoMix(
	reveal randao.RandaoReveal,
	proposer idx.ValidatorID,
	validatorKeys map[idx.ValidatorID]validatorpk.PubKey,
	lastBlockRandao common.Hash,
	fallbackRandao common.Hash,
	logger log.Logger,
) common.Hash {
	blockProposalRandao, ok := reveal.VerifyAndGetRandao(lastBlockRandao, validatorKeys[proposer])
	if ok {
		return blockProposalRandao
	} else {
		logger.Warn("Failed to verify randao reveal, using DAG randomization", "proposer validator", proposer)
		//  TODO: instrument a prometheus metric for this case (#209)
		return fallbackRandao
	}
}

// spillBlockEvents excludes first events which exceed MaxBlockGas
func spillBlockEvents(
	events hash.OrderedEvents,
	maxBlockGas uint64,
	getEventPayload func(id hash.Event) inter.EventPayloadI,
) []inter.EventPayloadI {
	fullEvents := make([]inter.EventPayloadI, len(events))
	if len(events) == 0 {
		return fullEvents
	}
	gasPowerUsedSum := uint64(0)
	// iterate in reversed order
	for i := len(events) - 1; i >= 0; i-- {
		id := events[i]
		e := getEventPayload(id)
		fullEvents[i] = e
		gasPowerUsedSum += e.GasPowerUsed()
		// stop if limit is exceeded, erase [:i] events
		if gasPowerUsedSum > maxBlockGas {
			// spill
			spilledEventsMeter.Mark(int64(len(fullEvents) - (i + 1)))
			fullEvents = fullEvents[i+1:]
			break
		}
	}
	return fullEvents
}

func mergeCheaters(a, b lachesis.Cheaters) lachesis.Cheaters {
	if len(b) == 0 {
		return a
	}
	if len(a) == 0 {
		return b
	}
	aSet := a.Set()
	merged := make(lachesis.Cheaters, 0, len(b)+len(a))
	merged = append(merged, a...)
	for _, v := range b {
		if _, ok := aSet[v]; !ok {
			merged = append(merged, v)
		}
	}
	return merged
}

// extractProposalForNextBlock attempts to obtain the canonical block proposal for
// the next block in the given events. A proposal is considered valid, if
//   - it has the correct block number (last block number + 1), and
//   - it has the correct parent hash (last block hash)
//
// If multiple valid proposals are found, the one proposed in the lowest turn
// is returned. If there are multiple proposals with the same turn, the one with
// the lowest hash is returned.
//
// If no valid proposals are found, nil is returned. In such a case, no or an
// empty block should be produced.
func extractProposalForNextBlock(
	lastBlock *evmcore.EvmHeader,
	events []inter.EventPayloadI,
	logger log.Logger,
) (*inter.Proposal, idx.ValidatorID, inter.Timestamp) {

	desiredBlockNumber := idx.Block(lastBlock.Number.Uint64() + 1)
	parentHash := lastBlock.Hash

	type PayloadInfo struct {
		Payload  *inter.Payload
		Proposer idx.ValidatorID
		Time     inter.Timestamp
	}

	// Collect all payloads from events proposing the desired block.
	payloads := []PayloadInfo{}
	for _, e := range events {
		payload := e.Payload()
		if proposal := payload.Proposal; proposal != nil {
			if proposal.Number != desiredBlockNumber {
				logger.Warn(
					"Confirmed events contains proposal with wrong block number",
					"wanted", desiredBlockNumber,
					"got", proposal.Number,
					"creator", e.Creator(),
				)
				continue
			}
			if proposal.ParentHash != parentHash {
				logger.Warn(
					"Confirmed events contains proposal with wrong parent hash",
					"wanted", parentHash,
					"got", proposal.ParentHash,
					"creator", e.Creator(),
				)
				continue
			}

			payloads = append(payloads, PayloadInfo{
				Payload:  payload,
				Proposer: e.Creator(),
				Time:     e.MedianTime(),
			})
		}
	}
	if len(payloads) > 1 {
		logger.Warn("Found multiple proposals for the same block",
			"block", desiredBlockNumber,
			"proposals", len(payloads),
		)
	}

	if len(payloads) == 0 {
		return nil, 0, 0
	}

	if len(payloads) == 1 {
		return payloads[0].Payload.Proposal, payloads[0].Proposer, payloads[0].Time
	}

	best := payloads[0]
	for _, p := range payloads {
		switch cmp.Compare(p.Payload.LastSeenProposalTurn, best.Payload.LastSeenProposalTurn) {
		case -1:
			best = p
		case 0:
			// The validation of events should not allow multiple proposals
			// with the same turn number in a forkless DAG, and forks should
			// be ignored by the consensus when producing confirmed events.
			// However, to be conservative, we consider the possibility of
			// two proposals with the same turn number and use the proposal
			// hash as a tie breaker.
			a := p.Payload.Proposal.Hash()
			b := best.Payload.Proposal.Hash()
			if bytes.Compare(a[:], b[:]) < 0 {
				best = p
			}
		case 1:
		}
	}
	return best.Payload.Proposal, best.Proposer, best.Time
}

// filterNonPermissibleTransactions filters out transactions that are not allowed
// to be included in a block according to the network rules. It returns a slice
// of permissible transactions. For encountered non-permissible transactions
// log messages are emitted and the number of such transactions is reported to
// the provided metric counter.
func filterNonPermissibleTransactions(
	transactions []*types.Transaction,
	rules *opera.Rules,
	log log.Logger,
	counter metricCounter,
) []*types.Transaction {
	// This filter is only enabled with the Allegro upgrade.
	if !rules.Upgrades.Allegro {
		return transactions
	}
	return slices.DeleteFunc(transactions, func(tx *types.Transaction) bool {
		if err := isPermissible(tx, rules); err != nil {
			if log != nil {
				log.Warn("Non-permissible transaction in the proposal", "tx", tx.Hash(), "issue", err)
			}
			if counter != nil {
				counter.Mark(1)
			}
			return true
		}
		return false
	})
}

// isPermissible checks whether a transaction is allowed to be included in a
// block according to the network rules. It is used to control the set of
// supported transaction types and their properties on the block chain.
//
// Rejected transactions are considered non-permissible transactions.
// Honest validators should not suggest non-permissible transactions.
//
// Permissible transactions may still be rejected by the block processor due to
// nonce or balance issues. In such cases, the transaction is considered a
// skipped transaction. Skips should be minimized, but can not be completely
// avoided.
func isPermissible(
	tx *types.Transaction,
	rules *opera.Rules,
) error {

	if tx == nil {
		return fmt.Errorf("nil transaction")
	}

	// -- Check transaction type --

	maxTxType := uint8(types.BlobTxType)
	if rules.Upgrades.Allegro {
		maxTxType = types.SetCodeTxType
	}
	if tx.Type() > maxTxType {
		return fmt.Errorf("unsupported transaction type %d, max supported is %d", tx.Type(), maxTxType)
	}

	// -- Check Type specific properties --

	if tx.Type() == types.BlobTxType {
		if have := len(tx.BlobHashes()); have > 0 {
			return fmt.Errorf(
				"blob transaction with blob hashes is not supported, got %d",
				have,
			)
		}
	}

	if tx.Type() == types.SetCodeTxType {
		if have := len(tx.SetCodeAuthorizations()); have == 0 {
			return fmt.Errorf(
				"set code transaction without authorizations is not supported",
			)
		}
	}

	return nil
}

// metricCounter is an abstraction of the *metrics.Meter type to facilitate
// mocking in tests.
type metricCounter interface {
	Mark(int64)
}
