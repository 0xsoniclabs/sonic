package gossip

import (
	"fmt"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"
	"time"

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
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
)

var (
	// Ethereum compatible metrics set (see go-ethereum/core)

	headBlockGauge     = metrics.GetOrRegisterGauge("chain/head/block", nil)
	headHeaderGauge    = metrics.GetOrRegisterGauge("chain/head/header", nil)
	headFastBlockGauge = metrics.GetOrRegisterGauge("chain/head/receipt", nil)

	blockExecutionTimer             = metrics.GetOrRegisterResettingTimer("chain/execution", nil)
	blockExecutionNonResettingTimer = metrics.GetOrRegisterTimer("chain/execution/nonresetting", nil)
	blockAgeGauge                   = metrics.GetOrRegisterGauge("chain/block/age", nil)

	processedTxsMeter    = metrics.GetOrRegisterMeter("chain/txs/processed", nil)
	skippedTxsMeter      = metrics.GetOrRegisterMeter("chain/txs/skipped", nil)
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
	verWatcher *verwatcher.VerWarcher,
	bootstrapping *bool,
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
				if e.AnyTxs() {
					confirmedEvents = append(confirmedEvents, e.ID())
				}
				eventProcessor.ProcessConfirmedEvent(e)
				for _, em := range *emitters {
					em.OnEventConfirmed(e)
				}
				confirmedEventsMeter.Mark(1)
			},
			EndBlock: func() (newValidators *pos.Validators) {
				if atroposTime <= bs.LastBlock.Time {
					atroposTime = bs.LastBlock.Time + 1
				}
				blockCtx := iblockproc.BlockCtx{
					Idx:     bs.LastBlock.Idx + 1,
					Time:    atroposTime,
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

				prevRandao := computePrevRandao(confirmedEvents)
				chainCfg := es.Rules.EvmChainConfig(store.GetUpgradeHeights())
				evmProcessor := blockProc.EVMModule.Start(
					blockCtx,
					statedb,
					evmStateReader,
					onNewLogAll,
					es.Rules,
					chainCfg,
					prevRandao,
				)
				executionStart := time.Now()

				// Execute pre-internal transactions
				preInternalTxs := blockProc.PreTxTransactor.PopInternalTxs(blockCtx, bs, es, sealing, statedb)
				preInternalReceipts := evmProcessor.Execute(preInternalTxs)
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

					// Start assembling the resulting block.
					number := uint64(blockCtx.Idx)
					lastBlockHeader := evmStateReader.GetHeaderByNumber(number - 1)
					maxBlockGas := es.Rules.Blocks.MaxBlockGas
					blockDuration := time.Duration(blockCtx.Time - bs.LastBlock.Time)
					blockBuilder := inter.NewBlockBuilder().
						WithEpoch(blockCtx.Atropos.Epoch()).
						WithNumber(number).
						WithParentHash(lastBlockHeader.Hash).
						WithTime(blockCtx.Time).
						WithPrevRandao(prevRandao).
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
					internalReceipts := evmProcessor.Execute(internalTxs)
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

					// sort events by Lamport time
					sort.Sort(confirmedEvents)

					blockEvents := spillBlockEvents(store, confirmedEvents, maxBlockGas)
					unorderedTxs := make(types.Transactions, 0, blockEvents.Len()*10)
					for _, e := range blockEvents {
						unorderedTxs = append(unorderedTxs, e.Txs()...)
					}

					signer := gsignercache.Wrap(types.MakeSigner(chainCfg, new(big.Int).SetUint64(number), uint64(blockCtx.Time)))
					orderedTxs := getExecutionOrder(unorderedTxs, signer, es.Rules.Upgrades.Sonic)

					for i, receipt := range evmProcessor.Execute(orderedTxs) {
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

					// Update block-hash references in receipts and logs.
					for i := range allReceipts {
						allReceipts[i].BlockHash = block.Hash()
						for j := range allReceipts[i].Logs {
							allReceipts[i].Logs[j].BlockHash = block.Hash()
						}
					}

					// memorize event position of each tx
					txPositions := make(map[common.Hash]ExtendedTxPosition)
					for _, e := range blockEvents {
						for i, tx := range e.Txs() {
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
						txListener.OnNewReceipt(evmBlock.Transactions[i], r, creator)
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

// spillBlockEvents excludes first events which exceed MaxBlockGas
func spillBlockEvents(store *Store, events hash.OrderedEvents, maxBlockGas uint64) inter.EventPayloads {
	fullEvents := make(inter.EventPayloads, len(events))
	if len(events) == 0 {
		return fullEvents
	}
	gasPowerUsedSum := uint64(0)
	// iterate in reversed order
	for i := len(events) - 1; ; i-- {
		id := events[i]
		e := store.GetEventPayload(id)
		if e == nil {
			log.Crit("Block event not found", "event", id.String())
			break
		}
		fullEvents[i] = e
		gasPowerUsedSum += e.GasPowerUsed()
		// stop if limit is exceeded, erase [:i] events
		if gasPowerUsedSum > maxBlockGas {
			// spill
			spilledEventsMeter.Mark(int64(len(fullEvents) - (i + 1)))
			fullEvents = fullEvents[i+1:]
			break
		}
		if i == 0 {
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
