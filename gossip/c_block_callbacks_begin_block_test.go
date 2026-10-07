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
	"bytes"
	"context"
	"crypto/sha256"
	"math"
	"math/big"
	"os"
	"os/exec"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/flushable"
	"github.com/Fantom-foundation/lachesis-base/kvdb/memorydb"
	"github.com/Fantom-foundation/lachesis-base/lachesis"
	"github.com/Fantom-foundation/lachesis-base/utils/workers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/evmcore/core_types"
	"github.com/0xsoniclabs/sonic/gossip/blockproc"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/verwatcher"
	"github.com/0xsoniclabs/sonic/gossip/emitter"
	emitter_config "github.com/0xsoniclabs/sonic/gossip/emitter/config"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/gossip/randao"
	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/drivertype"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/contracts/driver"
	"github.com/0xsoniclabs/sonic/opera/contracts/driver/driverpos"
	"github.com/0xsoniclabs/sonic/opera/contracts/evmwriter"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/0xsoniclabs/sonic/valkeystore"
)

// The tests in this file pin the behavior of consensusCallbackBeginBlockFn.
// The block processing modules are replaced by the recording fake beginBlockProc and
// the store is a fast in-memory store holding a controlled chain state (beginBlockChain).

const (
	beginBlockEpoch       = idx.Epoch(5)
	beginBlockLastBlock   = idx.Block(10)
	beginBlockMaxBlockGas = uint64(1_000_000)
	beginBlockSkipPeriod  = inter.Timestamp(10 * time.Second)
	beginBlockAllocPerSec = uint64(100_000)
)

// Markers stored in the EpochGas field, which the callbacks never modify, to
// trace which block state is handed to which module.
const (
	beginBlockTagStore     = 100 // block state in the store
	beginBlockTagEvents    = 101 // result of ConfirmedEventsProcessor.Finalize
	beginBlockTagListener1 = 102 // result of the first TxListener.Finalize
	beginBlockTagSealed    = 103 // block state result of SealerProcessor.SealEpoch
	beginBlockTagListener2 = 104 // result of the second TxListener.Finalize
)

var (
	beginBlockLastBlockTime = inter.FromUnix(1_700_000_000)
	beginBlockParentRandao  = common.Hash{0xaa}
	beginBlockStoreEsTag    = hash.Hash{0xe0} // EpochStateRoot of the epoch state in the store
	beginBlockSealedEsTag   = hash.Hash{0xe1} // EpochStateRoot of the sealed epoch state
	beginBlockSigner        = types.LatestSignerForChainID(new(big.Int).SetUint64(opera.FakeNetworkID))

	beginBlockPreSonicUpgrades = opera.Upgrades{Berlin: true, London: true}
)

// ---------------------------------------------------------------------------
// Chain state and store
// ---------------------------------------------------------------------------

// beginBlockChain is the chain state a block is processed on top of.
type beginBlockChain struct {
	bs           iblockproc.BlockState
	es           iblockproc.EpochState
	parentNumber uint64          // Number field of the stored parent block; 0 means bs.LastBlock.Idx
	parentTime   inter.Timestamp // Time field of the stored parent block; 0 means bs.LastBlock.Time
	parentRandao common.Hash
}

func defaultBeginBlockChain(stateRoot hash.Hash) beginBlockChain {
	validators, profiles := beginBlockValidators(1, 2, 3)
	rules := opera.FakeNetRules(opera.GetBrioUpgrades())
	rules.Blocks.MaxBlockGas = beginBlockMaxBlockGas
	rules.Blocks.MaxEmptyBlockSkipPeriod = beginBlockSkipPeriod
	rules.Economy.ShortGasPower.AllocPerSec = beginBlockAllocPerSec
	return beginBlockChain{
		bs: iblockproc.BlockState{
			LastBlock: iblockproc.BlockCtx{
				Idx:     beginBlockLastBlock,
				Time:    beginBlockLastBlockTime,
				Atropos: hash.Event{0x01},
			},
			FinalizedStateRoot: stateRoot,
			EpochGas:           beginBlockTagStore,
		},
		es: iblockproc.EpochState{
			Epoch:             beginBlockEpoch,
			EpochStateRoot:    beginBlockStoreEsTag,
			Validators:        validators,
			ValidatorProfiles: profiles,
			Rules:             rules,
		},
		parentRandao: beginBlockParentRandao,
	}
}

func (c *beginBlockChain) setUpgrades(upgrades opera.Upgrades) {
	c.es.Rules.Upgrades = upgrades
}

func (c *beginBlockChain) singleProposer() {
	c.es.Rules.Upgrades.SingleProposerBlockFormation = true
}

func (c beginBlockChain) writeTo(store *Store) {
	store.SetBlockEpochState(c.bs, c.es)
	number := c.parentNumber
	if number == 0 {
		number = uint64(c.bs.LastBlock.Idx)
	}
	blockTime := c.parentTime
	if blockTime == 0 {
		blockTime = c.bs.LastBlock.Time
	}
	// The base fee is above the minimum base fee, so that the base fee
	// derivation for the next block depends on all of these values.
	store.SetBlock(c.bs.LastBlock.Idx, inter.NewBlockBuilder().
		WithNumber(number).
		WithTime(blockTime).
		WithPrevRandao(c.parentRandao).
		WithEpoch(c.es.Epoch).
		WithGasLimit(c.es.Rules.Blocks.MaxBlockGas).
		WithGasUsed(c.es.Rules.Blocks.MaxBlockGas/2).
		WithDuration(time.Second).
		WithBaseFee(big.NewInt(10e9)).
		Build(),
	)
}

// beginBlockValidators returns validators with weights equal to their IDs and the
// public keys of makefakegenesis.FakeKey.
func beginBlockValidators(ids ...idx.ValidatorID) (*pos.Validators, iblockproc.ValidatorProfiles) {
	builder := pos.NewBuilder()
	profiles := iblockproc.ValidatorProfiles{}
	for _, id := range ids {
		builder.Set(id, pos.Weight(id))
		profiles[id] = drivertype.Validator{
			Weight: big.NewInt(int64(id)),
			PubKey: beginBlockPubKey(id),
		}
	}
	return builder.Build(), profiles
}

func beginBlockPubKey(key idx.ValidatorID) validatorpk.PubKey {
	private := makefakegenesis.FakeKey(key)
	return validatorpk.PubKey{
		Raw:  crypto.FromECDSAPub(&private.PublicKey),
		Type: validatorpk.Types.Secp256k1,
	}
}

// beginBlockRandaoReveal returns the randao reveal of makefakegenesis.FakeKey(key) for
// the given previous randao and the resulting randao mix.
func beginBlockRandaoReveal(t testing.TB, key idx.ValidatorID, prevRandao common.Hash) (randao.RandaoReveal, common.Hash) {
	t.Helper()
	pubKey := beginBlockPubKey(key)
	keystore := valkeystore.NewDefaultMemKeystore()
	require.NoError(t, keystore.Add(pubKey, crypto.FromECDSA(makefakegenesis.FakeKey(key)), validatorpk.FakePassword))
	require.NoError(t, keystore.Unlock(pubKey, validatorpk.FakePassword))
	reveal, mix, err := randao.NewRandaoMixerAdapter(valkeystore.NewSignerAuthority(keystore, pubKey)).MixRandao(prevRandao)
	require.NoError(t, err)
	return reveal, mix
}

// newBeginBlockStore creates a store backed by an in-memory Carmen state without
// archive and with a small StateDB cache, which is orders of magnitude faster to
// set up and smaller than a genesis store. If onPut is not nil, it is called
// before and after every non-batched write to the databases of the store.
func newBeginBlockStore(t testing.TB, onPut func()) *Store {
	t.Helper()
	cfg := MemTestStoreConfig(t.TempDir())
	cfg.EVM.StateDb.Variant = "go-memory"
	cfg.EVM.StateDb.Archive = carmen.NoArchive
	cfg.EVM.Cache.StateDbCapacity = 1000
	var dbs kvdb.FlushableDBProducer = flushable.NewSyncedPool(memorydb.NewProducer(""), []byte{0})
	if onPut != nil {
		dbs = beginBlockObservedProducer{FlushableDBProducer: dbs, onPut: onPut}
	}
	store, err := NewStore(dbs, cfg)
	require.NoError(t, err)
	require.NoError(t, store.evm.Open())
	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})
	return store
}

// beginBlockObservedProducer opens databases calling onPut before and after every Put.
type beginBlockObservedProducer struct {
	kvdb.FlushableDBProducer
	onPut func()
}

func (p beginBlockObservedProducer) OpenDB(name string) (kvdb.Store, error) {
	db, err := p.FlushableDBProducer.OpenDB(name)
	return beginBlockObservedDB{Store: db, onPut: p.onPut}, err
}

type beginBlockObservedDB struct {
	kvdb.Store
	onPut func()
}

func (db beginBlockObservedDB) Put(key, value []byte) error {
	db.onPut()
	err := db.Store.Put(key, value)
	db.onPut()
	return err
}

// beginBlockEpochBlocks returns the first blocks of epochs recorded in the store.
func beginBlockEpochBlocks(store *Store) map[idx.Block]idx.Epoch {
	res := map[idx.Block]idx.Epoch{}
	it := store.table.EpochBlocks.NewIterator(nil, nil)
	defer it.Release()
	for it.Next() {
		res[idx.Block(math.MaxUint64)-idx.BytesToBlock(it.Key())] = idx.BytesToEpoch(it.Value())
	}
	return res
}

func beginBlockLiveStateRoot(t testing.TB, store *Store) hash.Hash {
	t.Helper()
	statedb, err := store.evm.GetCurrentStateDb()
	require.NoError(t, err)
	defer statedb.Release()
	return hash.Hash(statedb.GetStateHash())
}

// beginBlockCommitState applies the given modification to the live state and returns
// the resulting state root.
func beginBlockCommitState(t testing.TB, store *Store, root hash.Hash, modify func(state.StateDB)) hash.Hash {
	t.Helper()
	statedb, err := store.evm.GetLiveStateDb(root)
	require.NoError(t, err)
	statedb.BeginBlock(1)
	modify(statedb)
	statedb.EndTransaction()
	staged, err := statedb.EndBlock(1)
	require.NoError(t, err)
	done, err := staged.Commit()
	require.NoError(t, err)
	if done != nil {
		require.NoError(t, done.Wait())
	}
	return hash.Hash(statedb.GetStateHash())
}

// ---------------------------------------------------------------------------
// Recording fakes of the block processing modules
// ---------------------------------------------------------------------------

// Records of the calls made to the block processing modules.
type (
	beginBlockEventsStart struct {
		BS iblockproc.BlockState
		ES iblockproc.EpochState
	}
	beginBlockProcessEvent struct {
		Event hash.Event
	}
	beginBlockEventsFinalize struct {
		Ctx     iblockproc.BlockCtx
		Skipped bool
	}
	beginBlockSealerStart struct {
		Ctx iblockproc.BlockCtx
		BS  iblockproc.BlockState
		ES  iblockproc.EpochState
	}
	beginBlockEpochSealing struct{}
	beginBlockSealerUpdate struct {
		BS iblockproc.BlockState
		ES iblockproc.EpochState
	}
	beginBlockSealEpoch struct {
		LastBlockHash hash.Hash
		ExecPlanHash  hash.Hash
		Txs           types.Transactions
	}
	beginBlockListenerStart struct {
		Ctx iblockproc.BlockCtx
		BS  iblockproc.BlockState
		ES  iblockproc.EpochState
	}
	beginBlockListenerLog struct {
		Log *core_types.Log
	}
	beginBlockListenerReceipt struct {
		Tx               *types.Transaction
		Receipt          *types.Receipt
		ReceiptBlockHash common.Hash // block hash of the receipt at the time of the call
		Creator          idx.ValidatorID
		BaseFee          *big.Int
		BlobBaseFee      *big.Int
	}
	beginBlockListenerFinalize struct{}
	beginBlockListenerUpdate   struct {
		BS iblockproc.BlockState
		ES iblockproc.EpochState
	}
	beginBlockPopInternalTxs struct {
		Phase   string
		Ctx     iblockproc.BlockCtx
		BS      iblockproc.BlockState
		ES      iblockproc.EpochState
		Sealing bool
		Nonce   uint64 // zero-address nonce provided by the nonce source at the time of the call
	}
	beginBlockEvmStart struct {
		Number   idx.Block
		Time     inter.Timestamp
		Epoch    idx.Epoch
		StateDB  state.StateDB
		Reader   evmcore.DummyChain
		OnNewLog func(*core_types.Log)
		Rules    opera.Rules
		ChainCfg *params.ChainConfig
		Randao   common.Hash
		Metrics  evmcore.BlockExecutionMetrics
	}
	beginBlockExecute struct {
		Txs       types.Transactions
		GasLimit  uint64
		SizeLimit uint64
		Busy      uint32 // block busy flag during the call
	}
	beginBlockEvmFinalize struct{}
)

func beginBlockCallName(call any) string {
	switch call := call.(type) {
	case beginBlockEventsStart:
		return "Events.Start"
	case beginBlockProcessEvent:
		return "Events.ProcessConfirmedEvent"
	case beginBlockEventsFinalize:
		return "Events.Finalize"
	case beginBlockSealerStart:
		return "Sealer.Start"
	case beginBlockEpochSealing:
		return "Sealer.EpochSealing"
	case beginBlockSealerUpdate:
		return "Sealer.Update"
	case beginBlockSealEpoch:
		return "Sealer.SealEpoch"
	case beginBlockListenerStart:
		return "TxListener.Start"
	case beginBlockListenerLog:
		return "TxListener.OnNewLog"
	case beginBlockListenerReceipt:
		return "TxListener.OnNewReceipt"
	case beginBlockListenerFinalize:
		return "TxListener.Finalize"
	case beginBlockListenerUpdate:
		return "TxListener.Update"
	case beginBlockPopInternalTxs:
		return call.Phase + ".PopInternalTxs"
	case beginBlockEvmStart:
		return "EVM.Start"
	case beginBlockExecute:
		return "EVM.Execute"
	case beginBlockEvmFinalize:
		return "EVM.Finalize"
	}
	panic("unknown call")
}

// beginBlockProc is a recording fake of all block processing modules. Its default
// behavior resembles the production modules: block states are passed through
// (marked with the beginBlockTag* constants), the EVM accepts every transaction and
// increments the nonce of the zero address for each, like internal transactions
// do, and sealing an epoch advances the epoch by one.
type beginBlockProc struct {
	mu    sync.Mutex
	calls []any
	hook  func(call any) // invoked after recording a call, outside of the lock

	busy    *uint32   // observed during EVM executions
	evmRoot hash.Hash // state root reported by the EVM

	sealing bool
	// modifies the results of the corresponding module calls
	eventsResult func(iblockproc.BlockState) iblockproc.BlockState
	sealResult   func(iblockproc.BlockState, iblockproc.EpochState) (iblockproc.BlockState, iblockproc.EpochState)

	preTxs  types.Transactions
	postTxs types.Transactions
	skipped map[common.Hash]bool               // txs skipped by the EVM (no receipt)
	failed  map[common.Hash]bool               // txs with a failed receipt
	logs    map[common.Hash][]*types.Log       // logs emitted by txs unless they fail
	derived map[common.Hash]types.Transactions // txs the EVM appends after executing a tx
	// replaces the default EVM finalization if set
	finalize func(*beginBlockEvmProcessor) (*evmcore.EvmBlock, int, types.Receipts)
}

func (p *beginBlockProc) blockProc() BlockProc {
	return BlockProc{
		SealerModule:     beginBlockSealerModule{p},
		TxListenerModule: beginBlockTxListenerModule{p},
		PreTxTransactor:  beginBlockTransactor{p, "PreTx"},
		PostTxTransactor: beginBlockTransactor{p, "PostTx"},
		EventsModule:     beginBlockEventsModule{p},
		EVMModule:        beginBlockEVMModule{p},
	}
}

func (p *beginBlockProc) record(call any) {
	p.mu.Lock()
	p.calls = append(p.calls, call)
	hook := p.hook
	p.mu.Unlock()
	if hook != nil {
		hook(call)
	}
}

func (p *beginBlockProc) trace() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	res := []string{}
	for _, call := range p.calls {
		res = append(res, beginBlockCallName(call))
	}
	return res
}

func (p *beginBlockProc) reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls = nil
}

// beginBlockCalls returns the recorded calls of type T in call order.
func beginBlockCalls[T any](p *beginBlockProc) []T {
	p.mu.Lock()
	defer p.mu.Unlock()
	res := []T{}
	for _, call := range p.calls {
		if call, ok := call.(T); ok {
			res = append(res, call)
		}
	}
	return res
}

type beginBlockEventsModule struct{ p *beginBlockProc }

func (m beginBlockEventsModule) Start(bs iblockproc.BlockState, es iblockproc.EpochState) blockproc.ConfirmedEventsProcessor {
	m.p.record(beginBlockEventsStart{BS: bs.Copy(), ES: es.Copy()})
	return &beginBlockEventsProcessor{p: m.p, bs: bs.Copy()}
}

type beginBlockEventsProcessor struct {
	p  *beginBlockProc
	bs iblockproc.BlockState
}

func (e *beginBlockEventsProcessor) ProcessConfirmedEvent(event inter.EventI) {
	e.p.record(beginBlockProcessEvent{Event: event.ID()})
}

func (e *beginBlockEventsProcessor) Finalize(ctx iblockproc.BlockCtx, skipped bool) iblockproc.BlockState {
	e.p.record(beginBlockEventsFinalize{Ctx: ctx, Skipped: skipped})
	res := e.bs.Copy()
	res.EpochGas = beginBlockTagEvents
	if e.p.eventsResult != nil {
		res = e.p.eventsResult(res)
	}
	return res
}

type beginBlockSealerModule struct{ p *beginBlockProc }

func (m beginBlockSealerModule) Start(ctx iblockproc.BlockCtx, bs iblockproc.BlockState, es iblockproc.EpochState) blockproc.SealerProcessor {
	m.p.record(beginBlockSealerStart{Ctx: ctx, BS: bs.Copy(), ES: es.Copy()})
	return &beginBlockSealer{p: m.p, bs: bs.Copy(), es: es.Copy()}
}

type beginBlockSealer struct {
	p  *beginBlockProc
	bs iblockproc.BlockState
	es iblockproc.EpochState
}

func (s *beginBlockSealer) EpochSealing() bool {
	s.p.record(beginBlockEpochSealing{})
	return s.p.sealing
}

func (s *beginBlockSealer) Update(bs iblockproc.BlockState, es iblockproc.EpochState) {
	s.p.record(beginBlockSealerUpdate{BS: bs.Copy(), ES: es.Copy()})
	s.bs, s.es = bs.Copy(), es.Copy()
}

func (s *beginBlockSealer) SealEpoch(lastBlockHash, execPlanHash hash.Hash, txs []*types.Transaction) (iblockproc.BlockState, iblockproc.EpochState) {
	s.p.record(beginBlockSealEpoch{LastBlockHash: lastBlockHash, ExecPlanHash: execPlanHash, Txs: slices.Clone(txs)})
	bs, es := s.bs.Copy(), s.es.Copy()
	bs.EpochGas = beginBlockTagSealed
	es.Epoch++
	es.EpochStateRoot = beginBlockSealedEsTag
	if s.p.sealResult != nil {
		bs, es = s.p.sealResult(bs, es)
	}
	return bs, es
}

type beginBlockTxListenerModule struct{ p *beginBlockProc }

func (m beginBlockTxListenerModule) Start(ctx iblockproc.BlockCtx, bs iblockproc.BlockState, es iblockproc.EpochState) blockproc.TxListener {
	m.p.record(beginBlockListenerStart{Ctx: ctx, BS: bs.Copy(), ES: es.Copy()})
	return &beginBlockTxListener{p: m.p, bs: bs.Copy()}
}

type beginBlockTxListener struct {
	p         *beginBlockProc
	bs        iblockproc.BlockState
	finalized int
}

func (l *beginBlockTxListener) OnNewLog(entry *core_types.Log) {
	l.p.record(beginBlockListenerLog{Log: entry})
}

func (l *beginBlockTxListener) OnNewReceipt(tx *types.Transaction, r *types.Receipt, originator idx.ValidatorID, baseFee *big.Int, blobBaseFee *big.Int) {
	l.p.record(beginBlockListenerReceipt{
		Tx:               tx,
		Receipt:          r,
		ReceiptBlockHash: r.BlockHash,
		Creator:          originator,
		BaseFee:          baseFee,
		BlobBaseFee:      blobBaseFee,
	})
}

func (l *beginBlockTxListener) Finalize() iblockproc.BlockState {
	l.p.record(beginBlockListenerFinalize{})
	l.finalized++
	res := l.bs.Copy()
	res.EpochGas = beginBlockTagListener1
	if l.finalized > 1 {
		res.EpochGas = beginBlockTagListener2
	}
	return res
}

func (l *beginBlockTxListener) Update(bs iblockproc.BlockState, es iblockproc.EpochState) {
	l.p.record(beginBlockListenerUpdate{BS: bs.Copy(), ES: es.Copy()})
	l.bs = bs.Copy()
}

type beginBlockTransactor struct {
	p     *beginBlockProc
	phase string
}

func (t beginBlockTransactor) PopInternalTxs(ctx iblockproc.BlockCtx, bs iblockproc.BlockState, es iblockproc.EpochState, sealing bool, nonces blockproc.NonceSource) types.Transactions {
	t.p.record(beginBlockPopInternalTxs{Phase: t.phase, Ctx: ctx, BS: bs.Copy(), ES: es.Copy(), Sealing: sealing, Nonce: nonces.ZeroAddressNonce()})
	if t.phase == "PreTx" {
		return slices.Clone(t.p.preTxs)
	}
	return slices.Clone(t.p.postTxs)
}

type beginBlockEVMModule struct{ p *beginBlockProc }

func (m beginBlockEVMModule) Start(
	blockNumber idx.Block,
	blockTime inter.Timestamp,
	epoch idx.Epoch,
	statedb state.StateDB,
	reader evmcore.DummyChain,
	onNewLog func(*core_types.Log),
	net opera.Rules,
	evmCfg *params.ChainConfig,
	prevrandao common.Hash,
	metrics evmcore.BlockExecutionMetrics,
) blockproc.EVMProcessor {
	m.p.record(beginBlockEvmStart{
		Number:   blockNumber,
		Time:     blockTime,
		Epoch:    epoch,
		StateDB:  statedb,
		Reader:   reader,
		OnNewLog: onNewLog,
		Rules:    net,
		ChainCfg: evmCfg,
		Randao:   prevrandao,
		Metrics:  metrics,
	})
	return &beginBlockEvmProcessor{p: m.p, number: blockNumber, time: blockTime, statedb: statedb, onNewLog: onNewLog}
}

type beginBlockEvmProcessor struct {
	p         *beginBlockProc
	number    idx.Block
	time      inter.Timestamp
	statedb   state.StateDB
	onNewLog  func(*core_types.Log)
	processed []evmcore.ProcessedTransaction
}

func (e *beginBlockEvmProcessor) Execute(txs types.Transactions, gasLimit uint64, sizeLimit uint64) evmcore.ProcessSummary {
	e.p.record(beginBlockExecute{Txs: slices.Clone(txs), GasLimit: gasLimit, SizeLimit: sizeLimit, Busy: atomic.LoadUint32(e.p.busy)})
	summary := evmcore.ProcessSummary{CausedBy: map[common.Hash]common.Hash{}}
	for _, tx := range txs {
		e.process(tx, tx, &summary)
		for _, derived := range e.p.derived[tx.Hash()] {
			e.process(derived, tx, &summary)
		}
	}
	return summary
}

func (e *beginBlockEvmProcessor) process(tx, origin *types.Transaction, summary *evmcore.ProcessSummary) {
	processed := evmcore.ProcessedTransaction{Transaction: tx}
	if !e.p.skipped[tx.Hash()] {
		receipt := &types.Receipt{
			Status:  types.ReceiptStatusSuccessful,
			TxHash:  tx.Hash(),
			GasUsed: 21_000,
		}
		if e.p.failed[tx.Hash()] {
			receipt.Status = types.ReceiptStatusFailed
		} else {
			receipt.Logs = e.p.logs[tx.Hash()]
		}
		for _, entry := range receipt.Logs {
			e.onNewLog(core_types.CoreLogFromGethLog(entry))
		}
		zero := common.Address{}
		e.statedb.SetNonce(zero, e.statedb.GetNonce(zero)+1, tracing.NonceChangeUnspecified)
		processed.Receipt = receipt
		summary.CausedBy[tx.Hash()] = origin.Hash()
	}
	summary.ProcessedTransactions = append(summary.ProcessedTransactions, processed)
	e.processed = append(e.processed, processed)
}

func (e *beginBlockEvmProcessor) Finalize() (*evmcore.EvmBlock, int, types.Receipts) {
	e.p.record(beginBlockEvmFinalize{})
	if e.p.finalize != nil {
		return e.p.finalize(e)
	}
	return e.defaultFinalize()
}

// defaultFinalize reports all transactions with a receipt as part of the block.
func (e *beginBlockEvmProcessor) defaultFinalize() (*evmcore.EvmBlock, int, types.Receipts) {
	block := &evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{
			Number:      big.NewInt(int64(e.number)),
			Time:        e.time,
			Root:        common.Hash(e.p.evmRoot),
			TxHash:      common.Hash{0x7f},
			BaseFee:     big.NewInt(1_000),
			BlobBaseFee: big.NewInt(1),
		},
	}
	skipped := 0
	receipts := types.Receipts{}
	for _, processed := range e.processed {
		if processed.Receipt == nil {
			skipped++
			continue
		}
		block.Transactions = append(block.Transactions, processed.Transaction)
		block.GasUsed += processed.Receipt.GasUsed
		receipts = append(receipts, processed.Receipt)
	}
	return block, skipped, receipts
}

// ---------------------------------------------------------------------------
// Callback environment
// ---------------------------------------------------------------------------

// beginBlockEnv holds the arguments of consensusCallbackBeginBlockFn. The store,
// workers, txIndex, feed and verWatcher are captured on the first call of fn.
type beginBlockEnv struct {
	t     *testing.T
	store *Store
	chain beginBlockChain
	proc  *beginBlockProc

	workers       *workers.Workers
	wg            sync.WaitGroup
	busy          uint32
	txIndex       bool
	feed          *ServiceFeed
	emitters      []*emitter.Emitter
	verWatcher    *verwatcher.VersionWatcher
	bootstrapping bool

	beginBlock lachesis.BeginBlockFn
}

func newBeginBlockEnv(t *testing.T, modify ...func(*beginBlockChain)) *beginBlockEnv {
	t.Helper()
	store := newBeginBlockStore(t, nil)
	chain := defaultBeginBlockChain(beginBlockLiveStateRoot(t, store))
	for _, m := range modify {
		if m != nil {
			m(&chain)
		}
	}
	chain.writeTo(store)

	quit := make(chan struct{})
	var workersDone sync.WaitGroup
	tasks := workers.New(&workersDone, quit, 1)
	// A single worker processes the blocks in order, which waitIdle relies on.
	tasks.Start(1)
	t.Cleanup(func() {
		close(quit)
		workersDone.Wait()
	})

	env := &beginBlockEnv{t: t, store: store, chain: chain, workers: tasks}
	env.proc = &beginBlockProc{busy: &env.busy, evmRoot: chain.bs.FinalizedStateRoot}
	return env
}

func (env *beginBlockEnv) fn() lachesis.BeginBlockFn {
	if env.beginBlock == nil {
		env.beginBlock = consensusCallbackBeginBlockFn(
			env.workers,
			&env.wg,
			&env.busy,
			env.store,
			env.proc.blockProc(),
			env.txIndex,
			env.feed,
			&env.emitters,
			env.verWatcher,
			&env.bootstrapping,
		)
	}
	return env.beginBlock
}

// runBlock stores the events, runs a full callback cycle applying the events
// in the given order and waits until the block is processed.
func (env *beginBlockEnv) runBlock(block *lachesis.Block, events ...*inter.EventPayload) *pos.Validators {
	env.t.Helper()
	callbacks := env.startBlock(block, events...)
	validators := callbacks.EndBlock()
	env.waitIdle()
	return validators
}

// startBlock stores the events, begins the block and applies the events.
func (env *beginBlockEnv) startBlock(block *lachesis.Block, events ...*inter.EventPayload) lachesis.BlockCallbacks {
	env.t.Helper()
	for _, event := range events {
		env.store.SetEvent(event)
	}
	callbacks := env.fn()(block)
	for _, event := range events {
		callbacks.ApplyEvent(event)
	}
	return callbacks
}

// waitIdle waits until asynchronous block processing, including its deferred
// clean-up, has finished. It fails the test if that takes longer than 10 seconds.
func (env *beginBlockEnv) waitIdle() {
	env.t.Helper()
	idle := make(chan error, 1)
	go func() {
		env.wg.Wait()
		done := make(chan struct{})
		if err := env.workers.Enqueue(func() { close(done) }); err != nil {
			idle <- err
			return
		}
		<-done
		idle <- nil
	}()
	select {
	case err := <-idle:
		require.NoError(env.t, err)
	case <-time.After(10 * time.Second):
		env.t.Fatal("block processing has not finished")
	}
}

// blockOn makes calls of type T block until the returned function is called,
// which also happens on clean-up so that failing tests do not hang.
func blockOn[T any](env *beginBlockEnv) (release func()) {
	released := make(chan struct{})
	release = sync.OnceFunc(func() { close(released) })
	env.t.Cleanup(release)
	env.proc.hook = func(call any) {
		if _, ok := call.(T); ok {
			<-released
		}
	}
	return release
}

func (env *beginBlockEnv) withFeed() chan feedUpdate {
	updates := make(chan feedUpdate, 16)
	env.feed = &ServiceFeed{incomingUpdates: updates}
	return updates
}

func (env *beginBlockEnv) withVersionWatcher() *verwatcher.Store {
	store := verwatcher.NewStore(memorydb.New())
	env.verWatcher = verwatcher.New(store)
	return store
}

func (env *beginBlockEnv) parentHash() common.Hash {
	return env.store.GetBlock(env.chain.bs.LastBlock.Idx).Hash()
}

// nextBlock is the successor of the last block index, which is the index of the
// produced block unless an accepted proposal follows a parent with another number.
func (env *beginBlockEnv) nextBlock() idx.Block {
	return env.chain.bs.LastBlock.Idx + 1
}

// proposal returns a valid proposal for the block following the stored parent block.
func (env *beginBlockEnv) proposal(txs ...*types.Transaction) *inter.Proposal {
	return &inter.Proposal{
		Number:       idx.Block(env.store.GetBlock(env.chain.bs.LastBlock.Idx).Number + 1),
		ParentHash:   env.parentHash(),
		Transactions: txs,
	}
}

// userTxs returns the transactions passed to the EVM by the user-transaction
// execution, which is the third execution of a block.
func (env *beginBlockEnv) userTxs() types.Transactions {
	executions := beginBlockCalls[beginBlockExecute](env.proc)
	require.Len(env.t, executions, 3)
	return executions[2].Txs
}

func (env *beginBlockEnv) evmStart() beginBlockEvmStart {
	starts := beginBlockCalls[beginBlockEvmStart](env.proc)
	require.Len(env.t, starts, 1)
	return starts[0]
}

// processedAsynchronously reports whether the busy flag was set while executing
// the user transactions, which is the case for asynchronously processed blocks.
func (env *beginBlockEnv) processedAsynchronously() bool {
	executions := beginBlockCalls[beginBlockExecute](env.proc)
	require.Len(env.t, executions, 3)
	return executions[2].Busy == 1
}

func (env *beginBlockEnv) blockSkipped() bool {
	finalizes := beginBlockCalls[beginBlockEventsFinalize](env.proc)
	require.Len(env.t, finalizes, 1)
	return finalizes[0].Skipped
}

// ---------------------------------------------------------------------------
// Events and transactions
// ---------------------------------------------------------------------------

type beginBlockEventSpec struct {
	epoch    idx.Epoch // 0 means beginBlockEpoch
	creator  idx.ValidatorID
	lamport  idx.Lamport
	seq      idx.Event
	time     inter.Timestamp
	gas      uint64
	txs      types.Transactions // transactions included directly in the event
	proposal *inter.Proposal
	turn     inter.Turn

	misbehaviourProofs bool
	blockVotes         bool
	epochVote          bool
}

// beginBlockEvent builds a version 1 event, or a version 3 event if it carries a
// proposal or a proposal turn.
func beginBlockEvent(spec beginBlockEventSpec) *inter.EventPayload {
	builder := inter.MutableEventPayload{}
	epoch := spec.epoch
	if epoch == 0 {
		epoch = beginBlockEpoch
	}
	builder.SetEpoch(epoch)
	builder.SetCreator(spec.creator)
	builder.SetLamport(spec.lamport)
	builder.SetSeq(spec.seq)
	builder.SetMedianTime(spec.time)
	builder.SetGasPowerUsed(spec.gas)
	builder.SetTxs(spec.txs)
	if spec.misbehaviourProofs {
		builder.SetMisbehaviourProofs([]inter.MisbehaviourProof{{}})
	}
	if spec.blockVotes {
		builder.SetBlockVotes(inter.LlrBlockVotes{Start: 1, Epoch: 1, Votes: []hash.Hash{{1}}})
	}
	if spec.epochVote {
		builder.SetEpochVote(inter.LlrEpochVote{Epoch: 1, Vote: hash.Hash{1}})
	}
	if spec.proposal != nil || spec.turn != 0 {
		builder.SetVersion(3)
		builder.SetPayload(inter.Payload{
			ProposalSyncState: inter.ProposalSyncState{LastSeenProposalTurn: spec.turn},
			Proposal:          spec.proposal,
		})
	} else {
		builder.SetVersion(1)
		builder.SetPayloadHash(inter.CalcPayloadHash(&builder))
	}
	return builder.Build()
}

// beginBlockAtropos returns an event without payload, as typically used as Atropos.
func beginBlockAtropos(time inter.Timestamp) *inter.EventPayload {
	return beginBlockEvent(beginBlockEventSpec{creator: 3, lamport: 100, time: time})
}

func beginBlockOf(atropos *inter.EventPayload, cheaters ...idx.ValidatorID) *lachesis.Block {
	return &lachesis.Block{Atropos: atropos.ID(), Cheaters: cheaters}
}

// beginBlockTx returns a transfer signed by makefakegenesis.FakeKey(sender).
func beginBlockTx(sender idx.ValidatorID, nonce uint64) *types.Transaction {
	return types.MustSignNewTx(makefakegenesis.FakeKey(sender), beginBlockSigner, &types.LegacyTx{
		Nonce:    nonce,
		To:       &common.Address{0x42},
		Gas:      21_000,
		GasPrice: big.NewInt(1),
	})
}

// beginBlockNonPermissibleTx returns a signed set-code transaction without
// authorizations, which is non-permissible starting with Allegro.
func beginBlockNonPermissibleTx(sender idx.ValidatorID, nonce uint64) *types.Transaction {
	return types.MustSignNewTx(makefakegenesis.FakeKey(sender), beginBlockSigner, &types.SetCodeTx{
		ChainID: uint256.NewInt(opera.FakeNetworkID),
		Nonce:   nonce,
		Gas:     21_000,
	})
}

// beginBlockPrevRandao reimplements the randao derivation from the given confirmed
// events: the sha256 of the XOR of the 24 pseudo-random bytes of the IDs.
func beginBlockPrevRandao(events ...hash.Event) common.Hash {
	mix := [24]byte{}
	for _, event := range events {
		for i := range mix {
			mix[i] ^= event[i+8]
		}
	}
	return sha256.Sum256(mix[:])
}

const beginBlockSubprocessEnv = "SONIC_TEST_BEGIN_BLOCK_SUBPROCESS"

// beginBlockInSubprocess reports whether the test runs as child process started by
// beginBlockRunInSubprocess.
func beginBlockInSubprocess(t *testing.T) bool {
	return os.Getenv(beginBlockSubprocessEnv) == t.Name()
}

// beginBlockRunInSubprocess runs the test in a child process and returns its exit
// code and output. It is used for behavior terminating the process.
func beginBlockRunInSubprocess(t *testing.T) (int, string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$", "-test.v")
	cmd.Env = append(os.Environ(), beginBlockSubprocessEnv+"="+t.Name())
	output, err := cmd.CombinedOutput()
	exitErr := &exec.ExitError{}
	require.ErrorAs(t, err, &exitErr)
	return exitErr.ExitCode(), string(output)
}

// beginBlockForeignEvent is a dag.Event that is not an inter.EventI.
type beginBlockForeignEvent struct {
	dag.Event
}

// beginBlockTime returns the time the given duration after the last block.
func beginBlockTime(d time.Duration) inter.Timestamp {
	return beginBlockLastBlockTime + inter.Timestamp(d)
}

func beginBlockHashes(txs types.Transactions) []common.Hash {
	res := []common.Hash{}
	for _, tx := range txs {
		res = append(res, tx.Hash())
	}
	return res
}

// setStateRoot makes the given root the finalized state root of the chain.
func (env *beginBlockEnv) setStateRoot(root hash.Hash) {
	env.chain.bs.FinalizedStateRoot = root
	env.store.SetBlockEpochState(env.chain.bs, env.chain.es)
	env.proc.evmRoot = root
}

// newBeginBlockObservedEmitter returns a validator emitter for the epoch of the chain.
// Emitters query the payload of confirmed events carrying transactions, which
// makes their notification observable as GetEventPayload calls on the mock.
func newBeginBlockObservedEmitter(ctrl *gomock.Controller, env *beginBlockEnv, id idx.ValidatorID) (*emitter.Emitter, *emitter.MockExternal) {
	external := emitter.NewMockExternal(ctrl)
	external.EXPECT().GetRules().Return(env.chain.es.Rules).AnyTimes()
	external.EXPECT().GetLastEvent(gomock.Any(), gomock.Any()).AnyTimes()
	external.EXPECT().StateDB().AnyTimes()
	external.EXPECT().DagIndex().AnyTimes()
	cfg := emitter_config.DefaultConfig()
	cfg.Validator.ID = id
	em := emitter.NewEmitter(cfg, emitter.World{External: external, TransactionSigner: beginBlockSigner}, nil, nil, nil, nil)
	em.OnNewEpoch(env.chain.es.Validators, env.chain.es.Epoch)
	return em, external
}

// ---------------------------------------------------------------------------
// BeginBlock
// ---------------------------------------------------------------------------

func TestGetConsensusCallbacks_PassesServiceComponentsToBeginBlockFn(t *testing.T) {
	t.Parallel()
	tests := map[string]bool{
		"with transaction index":    true,
		"without transaction index": false,
	}
	for name, txIndex := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				env := newBeginBlockEnv(t)
				versions := verwatcher.NewStore(memorydb.New())
				observed, world := newBeginBlockObservedEmitter(gomock.NewController(t), env, 1)
				updates := make(chan feedUpdate, 1)
				svc := &Service{
					config:           Config{TxIndex: txIndex},
					store:            env.store,
					verWatcher:       verwatcher.New(versions),
					blockProcTasks:   env.workers,
					blockProcModules: env.proc.blockProc(),
					feed:             ServiceFeed{incomingUpdates: updates},
				}
				env.proc.busy = &svc.blockBusyFlag
				beginBlock := svc.GetConsensusCallbacks().BeginBlock
				// Emitters are registered after the callbacks are created.
				svc.emitters = append(svc.emitters, observed)
				user := beginBlockTx(1, 0)
				env.proc.logs = map[common.Hash][]*types.Log{user.Hash(): {{
					Address: driver.ContractAddress,
					Topics:  []common.Hash{driverpos.Topics.UpdateNetworkVersion},
					Data:    common.LeftPadBytes([]byte{77}, 32),
				}}}
				event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{user}})
				env.store.SetEvent(event)
				world.EXPECT().GetEventPayload(event.ID()).Return(event)
				release := blockOn[beginBlockEvmFinalize](env)

				callbacks := beginBlock(beginBlockOf(event))
				callbacks.ApplyEvent(event)
				callbacks.EndBlock()
				var processed atomic.Bool
				go func() {
					svc.blockProcWg.Wait()
					processed.Store(true)
				}()
				synctest.Wait()
				require.False(t, processed.Load())
				require.Equal(t, uint32(1), atomic.LoadUint32(&svc.blockBusyFlag))

				release()
				synctest.Wait()
				require.True(t, processed.Load())
				require.NotNil(t, env.store.GetBlock(env.nextBlock()))
				require.Same(t, &svc.feed, env.evmStart().Reader.(*EvmStateReader).ServiceFeed)
				require.Len(t, updates, 1)
				require.Equal(t, uint64(77), versions.GetNetworkVersion())
				require.Equal(t, txIndex, env.store.evm.GetTxPosition(user.Hash()) != nil)

				// The bootstrapping flag of the service is read when a block begins.
				svc.bootstrapping = true
				env.proc.reset()
				beginBlock(&lachesis.Block{})
				require.Empty(t, env.proc.trace())
			})
		})
	}
}

func TestBeginBlockFn_Bootstrapping_IgnoresBlock(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		env := newBeginBlockEnv(t)
		env.bootstrapping = true
		updates := env.withFeed()
		observed, _ := newBeginBlockObservedEmitter(gomock.NewController(t), env, 1)
		env.emitters = []*emitter.Emitter{observed}
		event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(1, 0)}})
		env.store.SetEvent(event)
		beginBlock := env.fn()

		// A pending block would block BeginBlock if it was not bootstrapping.
		env.wg.Add(1)
		t.Cleanup(env.wg.Done)
		result := make(chan lachesis.BlockCallbacks, 1)
		go func() { result <- beginBlock(beginBlockOf(event, 1, 2)) }()
		synctest.Wait()
		require.Len(t, result, 1)

		callbacks := <-result
		callbacks.ApplyEvent(event)
		callbacks.ApplyEvent(beginBlockForeignEvent{})
		callbacks.ApplyEvent(nil)
		require.Nil(t, callbacks.EndBlock())

		require.Empty(t, env.proc.trace())
		bs, es := env.store.GetBlockEpochState()
		require.Equal(t, env.chain.bs.Hash(), bs.Hash())
		require.Equal(t, env.chain.es.Hash(), es.Hash())
		require.Nil(t, env.store.GetBlock(env.nextBlock()))
		require.Empty(t, updates)
		require.Zero(t, env.busy)
	})
}

func TestBeginBlockFn_Bootstrapping_FlagIsOnlyReadByBeginBlock(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		atBegin    bool
		afterBegin bool
		processed  bool
	}{
		"bootstrapping ends during the block": {
			atBegin:    true,
			afterBegin: false,
			processed:  false,
		},
		"bootstrapping starts during the block": {
			atBegin:    false,
			afterBegin: true,
			processed:  true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t)
			event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(1, 0)}})
			env.store.SetEvent(event)

			env.bootstrapping = test.atBegin
			callbacks := env.fn()(beginBlockOf(event))
			env.bootstrapping = test.afterBegin
			callbacks.ApplyEvent(event)
			callbacks.EndBlock()
			env.waitIdle()

			require.Equal(t, test.processed, len(env.proc.trace()) > 0)
			require.Equal(t, test.processed, env.store.GetBlock(env.nextBlock()) != nil)
		})
	}
}

func TestBeginBlockFn_WaitsForPendingBlockProcessing(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		env := newBeginBlockEnv(t)
		beginBlock := env.fn()
		env.wg.Add(1)
		done := sync.OnceFunc(env.wg.Done)
		t.Cleanup(done)

		var begun atomic.Bool
		go func() {
			beginBlock(&lachesis.Block{})
			begun.Store(true)
		}()
		synctest.Wait()
		require.False(t, begun.Load())
		require.Empty(t, env.proc.trace())

		done()
		synctest.Wait()
		require.True(t, begun.Load())
		require.Equal(t, []string{"Events.Start"}, env.proc.trace())
	})
}

func TestBeginBlockFn_StartsEventsModuleWithCopiesOfStoredStates(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t, func(c *beginBlockChain) {
		c.bs.ValidatorStates = []iblockproc.ValidatorBlockState{{DirtyGasRefund: 1, Originated: big.NewInt(0)}}
		c.es.ValidatorStates = []iblockproc.ValidatorEpochState{{GasRefund: 1}}
	})
	events := blockproc.NewMockConfirmedEventsModule(gomock.NewController(t))
	events.EXPECT().Start(gomock.Any(), gomock.Any()).DoAndReturn(func(bs iblockproc.BlockState, es iblockproc.EpochState) blockproc.ConfirmedEventsProcessor {
		// Modules may modify the validator states of the given states in place.
		bs.ValidatorStates[0].DirtyGasRefund = 7
		es.ValidatorStates[0].GasRefund = 7
		return nil
	})
	proc := env.proc.blockProc()
	proc.EventsModule = events
	env.beginBlock = consensusCallbackBeginBlockFn(env.workers, &env.wg, &env.busy, env.store, proc, false, nil, &env.emitters, nil, &env.bootstrapping)

	env.fn()(&lachesis.Block{})

	bs, es := env.store.GetBlockEpochState()
	require.Equal(t, uint64(1), bs.ValidatorStates[0].DirtyGasRefund)
	require.Equal(t, uint64(1), es.ValidatorStates[0].GasRefund)
}

func TestBeginBlockFn_ProcessesBlockWithStateReadByBeginBlock(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		txs       types.Transactions
		timeShift time.Duration // shift of the last block time stored after BeginBlock
		skipped   bool
		wantTag   uint64
	}{
		"produced block": {
			txs:       types.Transactions{beginBlockTx(1, 0)},
			timeShift: 5 * time.Second,
			skipped:   false,
			wantTag:   beginBlockTagListener2,
		},
		"skipped block": {
			timeShift: -20 * time.Second,
			skipped:   true,
			wantTag:   beginBlockTagEvents,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t)
			event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), gas: 100, txs: test.txs})
			callbacks := env.startBlock(beginBlockOf(event), event)

			modifiedBs, modifiedEs := env.chain.bs.Copy(), env.chain.es.Copy()
			modifiedBs.EpochGas = 7
			modifiedBs.LastBlock.Idx += 5
			modifiedBs.LastBlock.Time += inter.Timestamp(test.timeShift)
			modifiedEs.EpochStateRoot = hash.Hash{7}
			modifiedEs.Rules.Blocks.MaxBlockGas = 7
			modifiedEs.Rules.Blocks.MaxEmptyBlockSkipPeriod = 0
			modifiedEs.Rules.Upgrades.SingleProposerBlockFormation = true
			env.store.SetBlockEpochState(modifiedBs, modifiedEs)

			callbacks.EndBlock()
			env.waitIdle()

			require.Equal(t, test.skipped, env.blockSkipped())
			bs, es := env.store.GetBlockEpochState()
			require.Equal(t, test.wantTag, bs.EpochGas)
			require.Equal(t, beginBlockStoreEsTag, es.EpochStateRoot)
			require.Equal(t, beginBlockMaxBlockGas, es.Rules.Blocks.MaxBlockGas)
			if !test.skipped {
				require.Equal(t, env.chain.es.Rules, env.evmStart().Rules)
				require.Equal(t, env.nextBlock(), env.evmStart().Number)
				require.Equal(t, beginBlockTime(time.Second), env.evmStart().Time)
				require.Equal(t, beginBlockHashes(test.txs), beginBlockHashes(env.userTxs()))
				require.Equal(t, beginBlockMaxBlockGas, beginBlockCalls[beginBlockExecute](env.proc)[0].GasLimit)
			}
		})
	}
}

func TestBeginBlockFn_TerminatesProcessIfStateDbCannotBeOpened(t *testing.T) {
	if beginBlockInSubprocess(t) {
		log.SetDefault(log.NewLogger(log.NewTerminalHandler(os.Stderr, false)))
		env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.bs.FinalizedStateRoot = hash.Hash{0x12} })
		env.fn()(&lachesis.Block{})
		return
	}
	t.Parallel()

	code, output := beginBlockRunInSubprocess(t)
	require.Equal(t, 1, code)
	require.Contains(t, output, "Failed to open StateDB")
}

func TestBeginBlockFn_StartsEventsModuleWithStoredStateAndMergedCheaters(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		stored lachesis.Cheaters
		block  lachesis.Cheaters
		want   lachesis.Cheaters
	}{
		"no cheaters": {
			want: lachesis.Cheaters{},
		},
		"stored cheaters only": {
			stored: lachesis.Cheaters{1, 2},
			want:   lachesis.Cheaters{1, 2},
		},
		"cheaters of the block only": {
			block: lachesis.Cheaters{3},
			want:  lachesis.Cheaters{3},
		},
		"new cheaters are appended in block order": {
			stored: lachesis.Cheaters{2},
			block:  lachesis.Cheaters{3, 1},
			want:   lachesis.Cheaters{2, 3, 1},
		},
		"known cheaters are not appended": {
			stored: lachesis.Cheaters{1, 2},
			block:  lachesis.Cheaters{2, 3},
			want:   lachesis.Cheaters{1, 2, 3},
		},
		"duplicates of stored cheaters are kept": {
			stored: lachesis.Cheaters{2, 2},
			block:  lachesis.Cheaters{1},
			want:   lachesis.Cheaters{2, 2, 1},
		},
		"duplicates of block cheaters are kept": {
			stored: lachesis.Cheaters{1},
			block:  lachesis.Cheaters{3, 3},
			want:   lachesis.Cheaters{1, 3, 3},
		},
		"duplicates of block cheaters are kept without stored cheaters": {
			block: lachesis.Cheaters{3, 3},
			want:  lachesis.Cheaters{3, 3},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.bs.EpochCheaters = test.stored })
			env.fn()(&lachesis.Block{Cheaters: test.block})

			starts := beginBlockCalls[beginBlockEventsStart](env.proc)
			require.Len(t, starts, 1)
			require.Equal(t, test.want, starts[0].BS.EpochCheaters)
			want := env.chain.bs.Copy()
			want.EpochCheaters = test.want
			require.Equal(t, want.Hash(), starts[0].BS.Hash())
			require.Equal(t, env.chain.es.Hash(), starts[0].ES.Hash())

			// The merged cheaters are not written to the store.
			require.Equal(t, env.chain.bs.Hash(), env.store.GetBlockState().Hash())
		})
	}
}

// ---------------------------------------------------------------------------
// ApplyEvent
// ---------------------------------------------------------------------------

func TestBeginBlockFn_ApplyEvent_PanicsOnEventsNotImplementingEventI(t *testing.T) {
	t.Parallel()
	tests := map[string]dag.Event{
		"nil event":     nil,
		"foreign event": beginBlockForeignEvent{},
	}
	for name, event := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t)
			callbacks := env.fn()(&lachesis.Block{})
			require.Panics(t, func() { callbacks.ApplyEvent(event) })
		})
	}
}

func TestBeginBlockFn_ApplyEvent_RecognizesTheAtroposByItsFullId(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t)
	atropos := beginBlockAtropos(beginBlockTime(2 * time.Second))
	// Same epoch and Lamport time as the Atropos, which share the first bytes of the ID.
	sibling := beginBlockEvent(beginBlockEventSpec{creator: 2, lamport: atropos.Lamport(), time: beginBlockTime(3 * time.Second)})

	// The cheater makes sure the block is produced.
	env.runBlock(beginBlockOf(atropos, 2), sibling, atropos, sibling)

	require.Equal(t, beginBlockTime(2*time.Second), env.evmStart().Time)
}

func TestBeginBlockFn_ApplyEvent_CollectsEventsWithTransactionsOrProposals(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		spec      beginBlockEventSpec
		collected bool
	}{
		"no payload": {
			collected: false,
		},
		"transactions": {
			spec:      beginBlockEventSpec{txs: types.Transactions{beginBlockTx(1, 0)}},
			collected: true,
		},
		"empty proposal": {
			spec:      beginBlockEventSpec{proposal: &inter.Proposal{}},
			collected: true,
		},
		"proposal with transactions": {
			spec:      beginBlockEventSpec{proposal: &inter.Proposal{Transactions: types.Transactions{beginBlockTx(1, 0)}}},
			collected: true,
		},
		"proposal turn without proposal": {
			spec:      beginBlockEventSpec{turn: 1},
			collected: false,
		},
		"misbehaviour proofs": {
			spec:      beginBlockEventSpec{misbehaviourProofs: true},
			collected: false,
		},
		"block votes": {
			spec:      beginBlockEventSpec{blockVotes: true},
			collected: false,
		},
		"epoch vote": {
			spec:      beginBlockEventSpec{epochVote: true},
			collected: false,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t)
			spec := test.spec
			spec.creator, spec.lamport, spec.time = 1, 1, beginBlockTime(time.Second)
			event := beginBlockEvent(spec)
			atropos := beginBlockAtropos(beginBlockTime(time.Second))

			// The cheater makes the block non-empty independently of the event.
			env.runBlock(beginBlockOf(atropos, 3), event, atropos)

			collected := []hash.Event{}
			if test.collected {
				collected = append(collected, event.ID())
			}
			require.Equal(t, beginBlockPrevRandao(collected...), env.evmStart().Randao)
		})
	}
}

func TestBeginBlockFn_ApplyEvent_CollectsRepeatedlyAppliedEventsRepeatedly(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		upgrades   opera.Upgrades
		executions int
	}{
		"without Sonic, the transactions are executed twice": {
			upgrades:   beginBlockPreSonicUpgrades,
			executions: 2,
		},
		"with Sonic, the scrambler removes the duplicates": {
			upgrades:   opera.GetSonicUpgrades(),
			executions: 1,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.setUpgrades(test.upgrades) })
			tx := beginBlockTx(1, 0)
			event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{tx}})
			atropos := beginBlockAtropos(beginBlockTime(time.Second))

			env.runBlock(beginBlockOf(atropos), event, event, atropos)

			want := []common.Hash{}
			for range test.executions {
				want = append(want, tx.Hash())
			}
			require.Equal(t, want, beginBlockHashes(env.userTxs()))
			// The IDs of the duplicates cancel each other out in the randao.
			require.Equal(t, beginBlockPrevRandao(), env.evmStart().Randao)
		})
	}
}

func TestBeginBlockFn_ApplyEvent_ForwardsEveryEventToEventsProcessorInApplicationOrder(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t)
	withTxs := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 2, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(1, 0)}})
	withoutTxs := beginBlockEvent(beginBlockEventSpec{creator: 2, lamport: 1, time: beginBlockTime(time.Second)})
	atropos := beginBlockAtropos(beginBlockTime(2 * time.Second))
	// Events without transactions and proposals need no stored payload.
	unstored := beginBlockEvent(beginBlockEventSpec{creator: 3, lamport: 3, time: beginBlockTime(time.Second)})

	callbacks := env.startBlock(beginBlockOf(atropos), withTxs, atropos, withoutTxs, withTxs)
	callbacks.ApplyEvent(unstored)
	callbacks.EndBlock()
	env.waitIdle()

	ids := []hash.Event{}
	for _, call := range beginBlockCalls[beginBlockProcessEvent](env.proc) {
		ids = append(ids, call.Event)
	}
	require.Equal(t, []hash.Event{withTxs.ID(), atropos.ID(), withoutTxs.ID(), withTxs.ID(), unstored.ID()}, ids)
	require.Equal(t, []string{
		"Events.Start",
		"Events.ProcessConfirmedEvent",
		"Events.ProcessConfirmedEvent",
		"Events.ProcessConfirmedEvent",
		"Events.ProcessConfirmedEvent",
		"Events.ProcessConfirmedEvent",
		"Events.Finalize",
	}, env.proc.trace()[:7])
}

func TestBeginBlockFn_ApplyEvent_NotifiesEmittersInRegistrationOrder(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t)
	ctrl := gomock.NewController(t)
	first, firstWorld := newBeginBlockObservedEmitter(ctrl, env, 1)
	second, secondWorld := newBeginBlockObservedEmitter(ctrl, env, 2)
	env.emitters = []*emitter.Emitter{first, second}

	event1 := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(1, 0)}})
	event2 := beginBlockEvent(beginBlockEventSpec{creator: 2, lamport: 2, time: beginBlockTime(time.Second)})
	event3 := beginBlockEvent(beginBlockEventSpec{creator: 3, lamport: 3, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(3, 0)}})

	// Emitters only query the payload of events with transactions, so the
	// notifications about other events are not observable.
	gomock.InOrder(
		firstWorld.EXPECT().GetEventPayload(event1.ID()).Return(event1),
		secondWorld.EXPECT().GetEventPayload(event1.ID()).Return(event1),
		firstWorld.EXPECT().GetEventPayload(event3.ID()).Return(event3),
		secondWorld.EXPECT().GetEventPayload(event3.ID()).Return(event3),
		firstWorld.EXPECT().GetEventPayload(event1.ID()).Return(event1),
		secondWorld.EXPECT().GetEventPayload(event1.ID()).Return(event1),
	)
	env.runBlock(beginBlockOf(event2), event1, event2, event3, event1)
}

func TestBeginBlockFn_ApplyEvent_NotifiesEmittersAfterEventsProcessor(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t)
	observed, world := newBeginBlockObservedEmitter(gomock.NewController(t), env, 1)
	env.emitters = []*emitter.Emitter{observed}
	event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(1, 0)}})

	var processed []beginBlockProcessEvent
	world.EXPECT().GetEventPayload(event.ID()).DoAndReturn(func(hash.Event) *inter.EventPayload {
		processed = beginBlockCalls[beginBlockProcessEvent](env.proc)
		return event
	})
	env.runBlock(beginBlockOf(event), event)

	require.Equal(t, []beginBlockProcessEvent{{Event: event.ID()}}, processed)
}

func TestBeginBlockFn_ApplyEvent_NotifiesEmittersRegisteredDuringTheBlock(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t)
	late, lateWorld := newBeginBlockObservedEmitter(gomock.NewController(t), env, 1)

	event1 := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(1, 0)}})
	event2 := beginBlockEvent(beginBlockEventSpec{creator: 2, lamport: 2, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(2, 0)}})
	env.store.SetEvent(event1)
	env.store.SetEvent(event2)

	callbacks := env.fn()(beginBlockOf(event2))
	callbacks.ApplyEvent(event1)
	env.emitters = append(env.emitters, late)
	lateWorld.EXPECT().GetEventPayload(event2.ID()).Return(event2)
	callbacks.ApplyEvent(event2)
	callbacks.EndBlock()
	env.waitIdle()
}

func TestBeginBlockFn_ApplyEvent_MarksConfirmedEventsMeter(t *testing.T) {
	// Not parallel, as the meter is global.
	tests := map[string]struct {
		bootstrapping bool
		want          int64
	}{
		"processing": {
			bootstrapping: false,
			want:          3,
		},
		"bootstrapping": {
			bootstrapping: true,
			want:          0,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			env := newBeginBlockEnv(t)
			env.bootstrapping = test.bootstrapping
			event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(1, 0)}})
			atropos := beginBlockAtropos(beginBlockTime(time.Second))

			before := confirmedEventsMeter.Snapshot().Count()
			env.runBlock(beginBlockOf(atropos), event, atropos, atropos)
			require.Equal(t, test.want, confirmedEventsMeter.Snapshot().Count()-before)
		})
	}
}

// ---------------------------------------------------------------------------
// EndBlock: assembling the block
// ---------------------------------------------------------------------------

func TestBeginBlockFn_OrdersConfirmedEventsByEpochLamportAndHash(t *testing.T) {
	t.Parallel()
	sameLamportA := beginBlockEvent(beginBlockEventSpec{creator: 2, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(2, 0)}})
	sameLamportB := beginBlockEvent(beginBlockEventSpec{creator: 3, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(3, 0)}})
	if bytes.Compare(sameLamportA.ID().Bytes(), sameLamportB.ID().Bytes()) > 0 {
		sameLamportA, sameLamportB = sameLamportB, sameLamportA
	}
	previousEpoch := beginBlockEvent(beginBlockEventSpec{epoch: beginBlockEpoch - 1, creator: 1, lamport: 9, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(1, 0)}})
	higherLamport := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 2, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(1, 1)}})

	want := []common.Hash{}
	for _, event := range []*inter.EventPayload{previousEpoch, sameLamportA, sameLamportB, higherLamport} {
		want = append(want, event.Transactions()[0].Hash())
	}

	events := []*inter.EventPayload{higherLamport, sameLamportB, previousEpoch, sameLamportA}
	for permutation := range utils.Permute(events) {
		// Without Sonic, the transactions are executed in the order of the events.
		env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.setUpgrades(beginBlockPreSonicUpgrades) })
		atropos := beginBlockAtropos(beginBlockTime(time.Second))
		env.runBlock(beginBlockOf(atropos), append(permutation, atropos)...)
		require.Equal(t, want, beginBlockHashes(env.userTxs()))
	}
}

func TestBeginBlockFn_SpillsEarliestEventsExceedingMaxBlockGas(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		gas  []uint64 // gas power used by the events, in Lamport order
		kept int      // number of latest events kept in the block
	}{
		"events using exactly the max block gas are kept": {
			gas:  []uint64{300, 300, 400},
			kept: 3,
		},
		"earliest event exceeding the max block gas is spilled": {
			gas:  []uint64{301, 300, 400},
			kept: 2,
		},
		"events before an exceeding event are spilled even if they would fit": {
			gas:  []uint64{1, 700, 400},
			kept: 1,
		},
		"latest event exceeding the max block gas spills all events": {
			gas:  []uint64{1, 1, 1001},
			kept: 0,
		},
		"single event using the max block gas is kept": {
			gas:  []uint64{1000},
			kept: 1,
		},
		"events without gas before a full block are kept": {
			gas:  []uint64{0, 0, 1000},
			kept: 3,
		},
		"sum of the gas wrapping around is not exceeding": {
			gas:  []uint64{1, math.MaxUint64 - 400, 500},
			kept: 3,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t, func(c *beginBlockChain) {
				c.setUpgrades(beginBlockPreSonicUpgrades)
				c.es.Rules.Blocks.MaxBlockGas = 1000
			})
			events := []*inter.EventPayload{}
			txs := types.Transactions{}
			for i, gas := range test.gas {
				tx := beginBlockTx(1, uint64(i))
				txs = append(txs, tx)
				events = append(events, beginBlockEvent(beginBlockEventSpec{creator: 1, seq: idx.Event(i), lamport: idx.Lamport(i + 1), time: beginBlockTime(time.Second), gas: gas, txs: types.Transactions{tx}}))
			}
			atropos := beginBlockAtropos(beginBlockTime(time.Second))
			env.runBlock(beginBlockOf(atropos), append(events, atropos)...)

			require.Equal(t, beginBlockHashes(txs[len(txs)-test.kept:]), beginBlockHashes(env.userTxs()))
		})
	}
}

func TestBeginBlockFn_SpilledEventsAreStillConfirmedEvents(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.es.Rules.Blocks.MaxBlockGas = 1000 })
	event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), gas: 1001, txs: types.Transactions{beginBlockTx(1, 0)}})
	atropos := beginBlockAtropos(beginBlockTime(time.Second))

	env.runBlock(beginBlockOf(atropos), event, atropos)

	require.Empty(t, env.userTxs())
	// The block is not empty, its randao covers the event and it is processed
	// asynchronously, as for any other confirmed event.
	require.False(t, env.blockSkipped())
	require.Equal(t, beginBlockPrevRandao(event.ID()), env.evmStart().Randao)
	require.True(t, env.processedAsynchronously())
}

func TestBeginBlockFn_SpilledEventsMeterCountsKeptEvents(t *testing.T) {
	// Not parallel, as the meter is global.
	tests := map[string]struct {
		gas  []uint64
		want int64
	}{
		"no event spilled": {
			gas:  []uint64{100, 100, 100},
			want: 0,
		},
		"earliest event spilled": {
			gas:  []uint64{900, 100, 100},
			want: 2,
		},
		"all events spilled": {
			gas:  []uint64{100, 100, 1001},
			want: 0,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.es.Rules.Blocks.MaxBlockGas = 1000 })
			events := []*inter.EventPayload{}
			for i, gas := range test.gas {
				events = append(events, beginBlockEvent(beginBlockEventSpec{creator: 1, seq: idx.Event(i), lamport: idx.Lamport(i + 1), time: beginBlockTime(time.Second), gas: gas, txs: types.Transactions{beginBlockTx(1, uint64(i))}}))
			}
			atropos := beginBlockAtropos(beginBlockTime(time.Second))

			before := spilledEventsMeter.Snapshot().Count()
			env.runBlock(beginBlockOf(atropos), append(events, atropos)...)
			require.Equal(t, test.want, spilledEventsMeter.Snapshot().Count()-before)
		})
	}
}

func TestBeginBlockFn_PanicsIfPayloadOfConfirmedEventIsMissing(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t)
	event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(1, 0)}})

	callbacks := env.fn()(beginBlockOf(event))
	callbacks.ApplyEvent(event)
	require.Panics(t, func() { callbacks.EndBlock() })
}

func TestBeginBlockFn_PanicsIfParentBlockIsMissing(t *testing.T) {
	t.Parallel()
	tests := map[string]bool{
		"distributed":     false,
		"single proposer": true,
	}
	for name, singleProposer := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.es.Rules.Upgrades.SingleProposerBlockFormation = singleProposer })
			bs := env.chain.bs.Copy()
			bs.LastBlock.Idx++
			env.store.SetBlockEpochState(bs, env.chain.es)
			atropos := beginBlockAtropos(beginBlockTime(time.Second))

			callbacks := env.startBlock(beginBlockOf(atropos), atropos)
			require.Panics(t, func() { callbacks.EndBlock() })
		})
	}
}

func TestBeginBlockFn_CreatesChainConfigFromUpgradeHeightsStoredAtEndBlock(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t)
	env.store.AddUpgradeHeight(opera.UpgradeHeight{Upgrades: opera.GetSonicUpgrades(), Height: 1})
	event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(1, 0)}})
	atropos := beginBlockAtropos(beginBlockTime(time.Second))

	callbacks := env.startBlock(beginBlockOf(atropos), event, atropos)
	env.store.AddUpgradeHeight(opera.UpgradeHeight{Upgrades: opera.GetAllegroUpgrades(), Height: env.nextBlock()})
	env.store.AddUpgradeHeight(opera.UpgradeHeight{Upgrades: opera.GetBrioUpgrades(), Height: env.nextBlock() + 1})
	callbacks.EndBlock()
	env.waitIdle()

	// Allegro enables Prague, Brio enables Osaka.
	chainCfg := env.evmStart().ChainCfg
	require.NotNil(t, chainCfg.PragueTime)
	require.Nil(t, chainCfg.OsakaTime)
}

func TestBeginBlockFn_UsesChainIdOfTheNetworkRules(t *testing.T) {
	t.Parallel()
	const networkID = 1234
	env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.es.Rules.NetworkID = networkID })
	signer := types.LatestSignerForChainID(big.NewInt(networkID))
	valid := types.MustSignNewTx(makefakegenesis.FakeKey(1), signer, &types.LegacyTx{To: &common.Address{0x42}, Gas: 21_000, GasPrice: big.NewInt(1)})
	event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{valid, beginBlockTx(2, 0)}})

	env.runBlock(beginBlockOf(event), event)

	// The scrambler drops transactions signed for other chains.
	require.Equal(t, beginBlockHashes(types.Transactions{valid}), beginBlockHashes(env.userTxs()))
	require.Equal(t, big.NewInt(networkID), env.evmStart().ChainCfg.ChainID)
}

// beginBlockProposalSpec describes a proposal for the next block in a single-proposer
// test.
type beginBlockProposalSpec struct {
	numberOffset int // offset of the proposed block number from the next block
	wrongParent  bool
	turn         inter.Turn
	gas          uint64 // gas power used by the proposing event
}

func TestBeginBlockFn_SingleProposer_ExecutesBestValidProposal(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		proposals []beginBlockProposalSpec
		executed  int // index of the executed proposal, or -1 if none
	}{
		"single valid proposal": {
			proposals: []beginBlockProposalSpec{{}},
			executed:  0,
		},
		"proposal for the last block": {
			proposals: []beginBlockProposalSpec{{numberOffset: -1}},
			executed:  -1,
		},
		"proposal for a future block": {
			proposals: []beginBlockProposalSpec{{numberOffset: 1}},
			executed:  -1,
		},
		"proposal with wrong parent": {
			proposals: []beginBlockProposalSpec{{wrongParent: true}},
			executed:  -1,
		},
		"invalid proposals are ignored": {
			proposals: []beginBlockProposalSpec{{numberOffset: 1}, {turn: 5}, {wrongParent: true}},
			executed:  1,
		},
		"proposal of lowest turn": {
			proposals: []beginBlockProposalSpec{{turn: 3}, {turn: 1}, {turn: 2}},
			executed:  1,
		},
		"proposal of spilled event": {
			proposals: []beginBlockProposalSpec{{gas: beginBlockMaxBlockGas + 1}},
			executed:  -1,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t, (*beginBlockChain).singleProposer)
			events := []*inter.EventPayload{}
			mixes := []common.Hash{}
			for i, spec := range test.proposals {
				// Each proposal is made by another validator with a valid reveal.
				creator := idx.ValidatorID(i%3 + 1)
				proposal := env.proposal(beginBlockTx(1, uint64(i)))
				proposal.Number = idx.Block(int(proposal.Number) + spec.numberOffset)
				if spec.wrongParent {
					proposal.ParentHash = common.Hash{0x12}
				}
				reveal, mix := beginBlockRandaoReveal(t, creator, beginBlockParentRandao)
				proposal.RandaoReveal = reveal
				mixes = append(mixes, mix)
				events = append(events, beginBlockEvent(beginBlockEventSpec{
					creator:  creator,
					lamport:  idx.Lamport(i + 1),
					time:     beginBlockTime(time.Duration(i+1) * time.Second),
					gas:      spec.gas,
					proposal: proposal,
					turn:     spec.turn,
				}))
			}
			atropos := beginBlockAtropos(beginBlockTime(time.Second))
			env.runBlock(beginBlockOf(atropos), append(events, atropos)...)

			if test.executed < 0 {
				require.True(t, env.blockSkipped())
				return
			}
			executed := events[test.executed]
			require.Equal(t, beginBlockHashes(executed.Payload().Proposal.Transactions), beginBlockHashes(env.userTxs()))
			require.Equal(t, executed.MedianTime(), env.evmStart().Time)
			require.Equal(t, mixes[test.executed], env.evmStart().Randao)
		})
	}
}

func TestBeginBlockFn_SingleProposer_ExecutesProposalWithLowestHashAmongEqualTurns(t *testing.T) {
	t.Parallel()
	tests := map[string]bool{
		"lowest hash in the earlier event": false,
		"lowest hash in the later event":   true,
	}
	for name, lowestLater := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t, (*beginBlockChain).singleProposer)
			low, high := env.proposal(beginBlockTx(1, 0)), env.proposal(beginBlockTx(1, 1))
			if a, b := low.Hash(), high.Hash(); bytes.Compare(a[:], b[:]) > 0 {
				low, high = high, low
			}
			earlier, later := low, high
			if lowestLater {
				earlier, later = high, low
			}
			atropos := beginBlockAtropos(beginBlockTime(time.Second))

			env.runBlock(beginBlockOf(atropos),
				beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), proposal: earlier, turn: 2}),
				beginBlockEvent(beginBlockEventSpec{creator: 2, lamport: 2, time: beginBlockTime(time.Second), proposal: later, turn: 2}),
				atropos,
			)

			require.Equal(t, beginBlockHashes(low.Transactions), beginBlockHashes(env.userTxs()))
		})
	}
}

func TestBeginBlockFn_SingleProposer_ExecutesProposalOfEarliestEventAmongEqualProposals(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t, (*beginBlockChain).singleProposer)
	// The proposal of an even earlier event is replaced by the better ones.
	first := beginBlockEvent(beginBlockEventSpec{creator: 3, lamport: 1, time: beginBlockTime(time.Second / 2), proposal: env.proposal(beginBlockTx(3, 0)), turn: 3})
	proposal := env.proposal(beginBlockTx(1, 0))
	earlier := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 2, time: beginBlockTime(time.Second), proposal: proposal, turn: 2})
	later := beginBlockEvent(beginBlockEventSpec{creator: 2, lamport: 3, time: beginBlockTime(2 * time.Second), proposal: proposal, turn: 2})
	atropos := beginBlockAtropos(beginBlockTime(3 * time.Second))

	env.runBlock(beginBlockOf(atropos), later, earlier, first, atropos)

	require.Equal(t, earlier.MedianTime(), env.evmStart().Time)
}

func TestBeginBlockFn_SingleProposer_ExecutesOnlyTransactionsOfTheBestProposal(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t, (*beginBlockChain).singleProposer)
	direct := beginBlockEvent(beginBlockEventSpec{creator: 2, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(2, 0)}})
	rejectedProposal := env.proposal(beginBlockTx(3, 0))
	rejectedProposal.Number++
	rejected := beginBlockEvent(beginBlockEventSpec{creator: 3, lamport: 2, time: beginBlockTime(time.Second), proposal: rejectedProposal})
	accepted := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 3, time: beginBlockTime(time.Second), proposal: env.proposal(beginBlockTx(1, 0))})

	env.runBlock(beginBlockOf(accepted), direct, rejected, accepted)

	require.Equal(t, beginBlockHashes(accepted.Payload().Proposal.Transactions), beginBlockHashes(env.userTxs()))
}

func TestBeginBlockFn_SingleProposer_ExpectsProposalForSuccessorOfStoredParentBlockNumber(t *testing.T) {
	t.Parallel()
	// The stored block at the index of the last block reports block number 20.
	const parentNumber = 20
	tests := map[string]struct {
		number   idx.Block
		accepted bool
	}{
		"proposal for the successor of the last block index": {
			number:   beginBlockLastBlock + 1,
			accepted: false,
		},
		"proposal for the successor of the parent block number": {
			number:   parentNumber + 1,
			accepted: true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t, func(c *beginBlockChain) {
				c.singleProposer()
				c.parentNumber = parentNumber
			})
			proposal := env.proposal(beginBlockTx(1, 0))
			proposal.Number = test.number
			event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), proposal: proposal})

			env.runBlock(beginBlockOf(event), event)

			require.Equal(t, !test.accepted, env.blockSkipped())
		})
	}
}

func TestBeginBlockFn_SingleProposer_IndexesBlockByProposalNumberButNumbersItBySuccessorOfLastBlockIndex(t *testing.T) {
	t.Parallel()
	// The stored block at the index of the last block reports block number 20.
	env := newBeginBlockEnv(t, func(c *beginBlockChain) {
		c.singleProposer()
		c.parentNumber = 20
	})
	env.txIndex = true
	env.proc.sealing = true
	env.proc.sealResult = func(bs iblockproc.BlockState, es iblockproc.EpochState) (iblockproc.BlockState, iblockproc.EpochState) {
		es.Rules.Upgrades.TransactionBundles = true
		return bs, es
	}
	// Osaka is enabled by Brio for the block number only.
	env.store.AddUpgradeHeight(opera.UpgradeHeight{Upgrades: opera.GetBrioUpgrades(), Height: env.nextBlock()})
	env.store.AddUpgradeHeight(opera.UpgradeHeight{Upgrades: opera.GetAllegroUpgrades(), Height: env.nextBlock() + 1})
	tx := beginBlockTx(1, 0)
	event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), proposal: env.proposal(tx)})

	env.runBlock(beginBlockOf(event), event)

	const index = idx.Block(21)
	require.Equal(t, index, env.evmStart().Number)
	require.NotNil(t, env.evmStart().ChainCfg.OsakaTime)
	require.Equal(t, index, env.store.GetBlockState().LastBlock.Idx)
	require.Nil(t, env.store.GetBlock(beginBlockLastBlock+1))
	block := env.store.GetBlock(index)
	require.Equal(t, uint64(beginBlockLastBlock+1), block.Number)
	require.Equal(t, index, *env.store.GetBlockIndex(hash.Event(block.Hash())))
	require.Equal(t, index, env.store.evm.GetTxPosition(tx.Hash()).Block)
	require.NotNil(t, env.store.evm.GetRawReceiptsRLP(index))
	require.NotNil(t, env.store.EvmStore().GetCachedEvmBlock(index))
	require.Equal(t, map[idx.Block]idx.Epoch{index + 1: beginBlockEpoch + 1}, beginBlockEpochBlocks(env.store))
	heights := env.store.GetUpgradeHeights()
	require.Equal(t, index+1, heights[len(heights)-1].Height)
}

func TestBeginBlockFn_IndexesBlocksWithoutProposalBySuccessorOfLastBlockIndex(t *testing.T) {
	t.Parallel()
	tests := map[string]bool{
		"distributed":                      false,
		"single proposer without proposal": true,
	}
	for name, singleProposer := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// The stored block at the index of the last block reports block number 20.
			env := newBeginBlockEnv(t, func(c *beginBlockChain) {
				c.es.Rules.Upgrades.SingleProposerBlockFormation = singleProposer
				c.parentNumber = 20
			})
			atropos := beginBlockAtropos(beginBlockTime(time.Second))

			// The cheater makes sure the block is produced.
			env.runBlock(beginBlockOf(atropos, 2), atropos)

			require.Equal(t, env.nextBlock(), env.evmStart().Number)
			require.Equal(t, env.nextBlock(), env.store.GetBlockState().LastBlock.Idx)
			require.Equal(t, uint64(env.nextBlock()), env.store.GetBlock(env.nextBlock()).Number)
		})
	}
}

func TestBeginBlockFn_SingleProposer_UsesRandaoOfProposerRevealOrFallsBackToEventRandao(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		proposer   idx.ValidatorID
		key        idx.ValidatorID // key signing the reveal
		prevRandao common.Hash     // randao the reveal is created for
		garbage    bool            // whether the reveal is malformed
		rejected   bool            // whether the proposal is for a wrong block
		noProposal bool            // whether no event besides the Atropos is confirmed
		useReveal  bool
	}{
		"valid reveal of proposer": {
			proposer:   1,
			key:        1,
			prevRandao: beginBlockParentRandao,
			useReveal:  true,
		},
		"reveal signed by another validator": {
			proposer:   1,
			key:        2,
			prevRandao: beginBlockParentRandao,
		},
		"reveal for another previous randao": {
			proposer:   1,
			key:        1,
			prevRandao: common.Hash{0xbb},
		},
		"proposer without validator key": {
			proposer:   9,
			key:        9,
			prevRandao: beginBlockParentRandao,
		},
		"malformed reveal": {
			proposer: 1,
			garbage:  true,
		},
		"valid reveal of rejected proposal": {
			proposer:   1,
			key:        1,
			prevRandao: beginBlockParentRandao,
			rejected:   true,
		},
		"no proposal": {
			noProposal: true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t, (*beginBlockChain).singleProposer)
			atropos := beginBlockAtropos(beginBlockTime(time.Second))
			events := []*inter.EventPayload{atropos}
			want := beginBlockPrevRandao()
			if !test.noProposal {
				proposal := env.proposal(beginBlockTx(1, 0))
				var mix common.Hash
				if test.garbage {
					proposal.RandaoReveal = randao.RandaoReveal{1, 2, 3}
				} else {
					proposal.RandaoReveal, mix = beginBlockRandaoReveal(t, test.key, test.prevRandao)
				}
				if test.rejected {
					proposal.Number++
				}
				event := beginBlockEvent(beginBlockEventSpec{creator: test.proposer, lamport: 1, time: beginBlockTime(time.Second), proposal: proposal})
				events = append(events, event)
				want = beginBlockPrevRandao(event.ID())
				if test.useReveal {
					want = mix
				}
			}

			// The cheater makes sure the block is produced.
			env.runBlock(beginBlockOf(atropos, 2), events...)

			require.Equal(t, want, env.evmStart().Randao)
			require.Equal(t, want, env.store.GetBlock(env.nextBlock()).PrevRandao)
		})
	}
}

func TestBeginBlockFn_SingleProposer_VerifiesRevealWithKeysOfAtroposEpoch(t *testing.T) {
	t.Parallel()
	const atroposEpoch = idx.Epoch(9)
	env := newBeginBlockEnv(t, (*beginBlockChain).singleProposer)
	// In the epoch of the Atropos, validator 1 uses key 7.
	history := env.chain.es.Copy()
	history.Epoch = atroposEpoch
	history.ValidatorProfiles[1] = drivertype.Validator{Weight: big.NewInt(1), PubKey: beginBlockPubKey(7)}
	env.store.SetHistoryBlockEpochState(atroposEpoch, env.chain.bs, history)

	proposal := env.proposal(beginBlockTx(1, 0))
	reveal, mix := beginBlockRandaoReveal(t, 7, beginBlockParentRandao)
	proposal.RandaoReveal = reveal
	event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), proposal: proposal})
	atropos := beginBlockEvent(beginBlockEventSpec{epoch: atroposEpoch, creator: 3, lamport: 1, time: beginBlockTime(time.Second)})

	env.runBlock(beginBlockOf(atropos), event, atropos)

	require.Equal(t, mix, env.evmStart().Randao)
}

func TestBeginBlockFn_SingleProposer_PanicsWithoutEpochStateOfAtroposEpoch(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t, (*beginBlockChain).singleProposer)
	event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), proposal: env.proposal(beginBlockTx(1, 0))})
	atropos := beginBlockEvent(beginBlockEventSpec{epoch: beginBlockEpoch + 4, creator: 3, lamport: 1, time: beginBlockTime(time.Second)})

	callbacks := env.startBlock(beginBlockOf(atropos), event, atropos)
	require.Panics(t, func() { callbacks.EndBlock() })
}

func TestBeginBlockFn_SingleProposer_LimitsUserGasByTimeSinceParentBlock(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		parentTime   inter.Timestamp // 0 means the last block time
		proposalTime inter.Timestamp
		allocPerSec  uint64
		wantGas      uint64
		wantTime     inter.Timestamp
	}{
		"half a second": {
			proposalTime: beginBlockTime(time.Second / 2),
			wantGas:      beginBlockAllocPerSec / 2,
			wantTime:     beginBlockTime(time.Second / 2),
		},
		"two seconds": {
			proposalTime: beginBlockTime(2 * time.Second),
			wantGas:      2 * beginBlockAllocPerSec,
			wantTime:     beginBlockTime(2 * time.Second),
		},
		"accumulation is capped at two seconds": {
			proposalTime: beginBlockTime(5 * time.Second),
			wantGas:      2 * beginBlockAllocPerSec,
			wantTime:     beginBlockTime(5 * time.Second),
		},
		"capped by the max block gas": {
			proposalTime: beginBlockTime(time.Second),
			allocPerSec:  2 * beginBlockMaxBlockGas,
			wantGas:      beginBlockMaxBlockGas,
			wantTime:     beginBlockTime(time.Second),
		},
		"proposal at the time of the parent block": {
			proposalTime: beginBlockLastBlockTime,
			wantGas:      0,
			wantTime:     beginBlockLastBlockTime + 1,
		},
		"proposal before the time of the parent block": {
			proposalTime: beginBlockTime(-time.Second),
			wantGas:      0,
			wantTime:     beginBlockLastBlockTime + 1,
		},
		"measured from the time of the stored parent block": {
			parentTime:   beginBlockTime(-time.Second),
			proposalTime: beginBlockTime(time.Second / 2),
			wantGas:      beginBlockAllocPerSec * 3 / 2,
			wantTime:     beginBlockTime(time.Second / 2),
		},
		"measured with the proposal time before clamping the block time": {
			parentTime:   beginBlockTime(-time.Second),
			proposalTime: beginBlockTime(-time.Second / 2),
			wantGas:      beginBlockAllocPerSec / 2,
			wantTime:     beginBlockLastBlockTime + 1,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t, func(c *beginBlockChain) {
				c.singleProposer()
				c.parentTime = test.parentTime
				if test.allocPerSec != 0 {
					c.es.Rules.Economy.ShortGasPower.AllocPerSec = test.allocPerSec
				}
			})
			event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: test.proposalTime, proposal: env.proposal(beginBlockTx(1, 0))})
			// The time of the Atropos does not affect the gas limit.
			atropos := beginBlockAtropos(beginBlockTime(time.Second / 4))

			env.runBlock(beginBlockOf(atropos), event, atropos)

			// Internal transactions are not limited by the user gas limit.
			executions := beginBlockCalls[beginBlockExecute](env.proc)
			require.Len(t, executions, 3)
			require.Equal(t, beginBlockMaxBlockGas, executions[0].GasLimit)
			require.Equal(t, beginBlockMaxBlockGas, executions[1].GasLimit)
			require.Equal(t, test.wantGas, executions[2].GasLimit)
			require.Equal(t, test.wantTime, env.evmStart().Time)
		})
	}
}

func TestBeginBlockFn_LimitsUserGasByMaxBlockGasWithoutProposal(t *testing.T) {
	t.Parallel()
	tests := map[string]bool{
		"distributed":                      false,
		"single proposer without proposal": true,
	}
	for name, singleProposer := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.es.Rules.Upgrades.SingleProposerBlockFormation = singleProposer })
			event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second / 10), txs: types.Transactions{beginBlockTx(1, 0)}})
			atropos := beginBlockAtropos(beginBlockTime(time.Second / 10))

			// The cheater makes sure the block is produced in both modes.
			env.runBlock(beginBlockOf(atropos, 2), event, atropos)

			require.Equal(t, beginBlockMaxBlockGas, beginBlockCalls[beginBlockExecute](env.proc)[2].GasLimit)
		})
	}
}

func TestBeginBlockFn_Distributed_ExecutesTransactionsOfAllEventsInEventOrder(t *testing.T) {
	t.Parallel()
	// Without Sonic, the transactions are not scrambled.
	env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.setUpgrades(beginBlockPreSonicUpgrades) })
	a, b, c, d := beginBlockTx(1, 0), beginBlockTx(1, 1), beginBlockTx(1, 2), beginBlockTx(1, 3)
	direct := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{a, b}})
	// Proposals are not validated in the distributed mode.
	invalidProposal := beginBlockEvent(beginBlockEventSpec{creator: 2, lamport: 2, time: beginBlockTime(time.Second), proposal: &inter.Proposal{Number: 1234, Transactions: types.Transactions{c}}})
	emptyProposal := beginBlockEvent(beginBlockEventSpec{creator: 3, lamport: 3, time: beginBlockTime(time.Second), proposal: &inter.Proposal{}})
	validProposal := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 4, time: beginBlockTime(time.Second), proposal: env.proposal(d)})
	atropos := beginBlockAtropos(beginBlockTime(time.Second))

	env.runBlock(beginBlockOf(atropos), validProposal, emptyProposal, invalidProposal, direct, atropos)

	require.Equal(t, beginBlockHashes(types.Transactions{a, b, c, d}), beginBlockHashes(env.userTxs()))
}

func TestBeginBlockFn_Distributed_ScramblesTransactionsWithSonic(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.setUpgrades(opera.GetSonicUpgrades()) })
	unsigned := types.NewTx(&types.LegacyTx{Nonce: 7, To: &common.Address{0x42}, Gas: 21_000})
	txs := types.Transactions{beginBlockTx(1, 0), beginBlockTx(2, 0), beginBlockTx(3, 0), beginBlockTx(1, 1), unsigned}
	event1 := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: txs[:3]})
	event2 := beginBlockEvent(beginBlockEventSpec{creator: 2, lamport: 2, time: beginBlockTime(time.Second), txs: append(types.Transactions{txs[0]}, txs[3:]...)})
	atropos := beginBlockAtropos(beginBlockTime(time.Second))

	env.runBlock(beginBlockOf(atropos), event1, event2, atropos)

	// Duplicates are removed and transactions without a valid signature are dropped.
	require.Equal(t, beginBlockHashes(types.Transactions{txs[2], txs[0], txs[3], txs[1]}), beginBlockHashes(env.userTxs()))
}

func TestBeginBlockFn_FiltersNonPermissibleTransactionsStartingWithAllegro(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		upgrades       opera.Upgrades
		singleProposer bool
		filtered       bool
	}{
		"distributed with Sonic": {
			upgrades: opera.GetSonicUpgrades(),
			filtered: false,
		},
		"distributed with Allegro": {
			upgrades: opera.GetAllegroUpgrades(),
			filtered: true,
		},
		"single proposer with Sonic": {
			upgrades:       opera.GetSonicUpgrades(),
			singleProposer: true,
			filtered:       false,
		},
		"single proposer with Allegro": {
			upgrades:       opera.GetAllegroUpgrades(),
			singleProposer: true,
			filtered:       true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t, func(c *beginBlockChain) {
				c.setUpgrades(test.upgrades)
				c.es.Rules.Upgrades.SingleProposerBlockFormation = test.singleProposer
			})
			valid, invalid := beginBlockTx(1, 0), beginBlockNonPermissibleTx(2, 0)
			spec := beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{invalid, valid}}
			if test.singleProposer {
				spec.txs, spec.proposal = nil, env.proposal(invalid, valid)
			}
			event := beginBlockEvent(spec)

			env.runBlock(beginBlockOf(event), event)

			want := types.Transactions{valid}
			if !test.filtered {
				want = append(want, invalid)
			}
			require.ElementsMatch(t, beginBlockHashes(want), beginBlockHashes(env.userTxs()))
		})
	}
}

func TestBeginBlockFn_Distributed_FiltersNonPermissibleTransactionsAfterScrambling(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.setUpgrades(opera.GetAllegroUpgrades()) })
	invalid := beginBlockNonPermissibleTx(1, 0)
	txs := types.Transactions{invalid, beginBlockTx(1, 1), beginBlockTx(2, 0), beginBlockTx(3, 0)}
	event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: txs})

	env.runBlock(beginBlockOf(event), event)

	// The order of the scrambler depends on all transactions, including the
	// non-permissible one.
	require.Equal(t, beginBlockHashes(types.Transactions{txs[2], txs[1], txs[3]}), beginBlockHashes(env.userTxs()))
}

func TestBeginBlockFn_InvalidTxsMeterCountsFilteredTransactions(t *testing.T) {
	// Not parallel, as the meter is global.
	env := newBeginBlockEnv(t)
	event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockNonPermissibleTx(1, 0), beginBlockTx(2, 0), beginBlockNonPermissibleTx(3, 0)}})

	before := invalidTxsMeter.Snapshot().Count()
	env.runBlock(beginBlockOf(event), event)
	require.Equal(t, int64(2), invalidTxsMeter.Snapshot().Count()-before)
}

func TestBeginBlockFn_SingleProposer_FilteringLeavesTransactionsOfStoredProposalUnchanged(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t, (*beginBlockChain).singleProposer)
	valid, invalid := beginBlockTx(1, 0), beginBlockNonPermissibleTx(2, 0)
	event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), proposal: env.proposal(invalid, valid)})

	env.runBlock(beginBlockOf(event), event)

	// The filter works on a copy of the transactions of the proposal, so the
	// cached event keeps all of them.
	require.False(t, env.blockSkipped())
	require.Equal(t, beginBlockHashes(types.Transactions{valid}), beginBlockHashes(env.userTxs()))
	require.Equal(t, beginBlockHashes(types.Transactions{invalid, valid}), beginBlockHashes(env.store.GetEventPayload(event.ID()).Payload().Proposal.Transactions))
}

func TestBeginBlockFn_TakesBlockTimeFromAtroposOrProposalButAfterLastBlock(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		singleProposer bool
		atroposTime    inter.Timestamp
		proposalTime   inter.Timestamp // 0 means no valid proposal
		want           inter.Timestamp
	}{
		"distributed: atropos time": {
			atroposTime: beginBlockTime(3 * time.Second),
			want:        beginBlockTime(3 * time.Second),
		},
		"distributed: atropos time equal to last block time": {
			atroposTime: beginBlockLastBlockTime,
			want:        beginBlockLastBlockTime + 1,
		},
		"distributed: atropos time before last block time": {
			atroposTime: beginBlockTime(-time.Second),
			want:        beginBlockLastBlockTime + 1,
		},
		"distributed: zero atropos time": {
			atroposTime: 0,
			want:        beginBlockLastBlockTime + 1,
		},
		"distributed: time of proposals is ignored": {
			atroposTime:  beginBlockTime(3 * time.Second),
			proposalTime: beginBlockTime(2 * time.Second),
			want:         beginBlockTime(3 * time.Second),
		},
		"single proposer: proposal time": {
			singleProposer: true,
			atroposTime:    beginBlockTime(3 * time.Second),
			proposalTime:   beginBlockTime(2 * time.Second),
			want:           beginBlockTime(2 * time.Second),
		},
		"single proposer: proposal time later than atropos time": {
			singleProposer: true,
			atroposTime:    beginBlockTime(2 * time.Second),
			proposalTime:   beginBlockTime(3 * time.Second),
			want:           beginBlockTime(3 * time.Second),
		},
		"single proposer: proposal time equal to last block time": {
			singleProposer: true,
			atroposTime:    beginBlockTime(3 * time.Second),
			proposalTime:   beginBlockLastBlockTime,
			want:           beginBlockLastBlockTime + 1,
		},
		"single proposer: atropos time without proposal": {
			singleProposer: true,
			atroposTime:    beginBlockTime(3 * time.Second),
			want:           beginBlockTime(3 * time.Second),
		},
		"single proposer: atropos time before last block time without proposal": {
			singleProposer: true,
			atroposTime:    beginBlockTime(-time.Second),
			want:           beginBlockLastBlockTime + 1,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.es.Rules.Upgrades.SingleProposerBlockFormation = test.singleProposer })
			events := []*inter.EventPayload{}
			if test.proposalTime != 0 {
				events = append(events, beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: test.proposalTime, proposal: env.proposal(beginBlockTx(1, 0))}))
			}
			atropos := beginBlockAtropos(test.atroposTime)
			events = append(events, atropos)

			// The cheater makes sure the block is produced.
			env.runBlock(beginBlockOf(atropos, 2), events...)

			require.Equal(t, test.want, env.evmStart().Time)
			require.Equal(t, test.want, env.store.GetBlock(env.nextBlock()).Time)
		})
	}
}

// ---------------------------------------------------------------------------
// EndBlock: skipping blocks
// ---------------------------------------------------------------------------

func TestBeginBlockFn_SkipsDegenerateAndEmptyBlocksWithinSkipPeriod(t *testing.T) {
	t.Parallel()
	withTx := func(env *beginBlockEnv) []*inter.EventPayload {
		return []*inter.EventPayload{beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(1, 0)}})}
	}
	withPayload := func(spec beginBlockEventSpec) func(env *beginBlockEnv) []*inter.EventPayload {
		return func(env *beginBlockEnv) []*inter.EventPayload {
			spec.creator, spec.lamport = 1, 1
			if spec.time == 0 {
				spec.time = beginBlockTime(time.Second)
			}
			return []*inter.EventPayload{beginBlockEvent(spec)}
		}
	}
	withProposal := func(time inter.Timestamp, txs ...*types.Transaction) func(env *beginBlockEnv) []*inter.EventPayload {
		return func(env *beginBlockEnv) []*inter.EventPayload {
			return withPayload(beginBlockEventSpec{time: time, proposal: env.proposal(txs...)})(env)
		}
	}
	endOfSkipPeriod := beginBlockLastBlockTime + beginBlockSkipPeriod

	tests := map[string]struct {
		singleProposer bool
		events         func(env *beginBlockEnv) []*inter.EventPayload
		atroposTime    inter.Timestamp
		parentTime     inter.Timestamp // 0 means the last block time
		skipPeriod     inter.Timestamp // 0 means beginBlockSkipPeriod
		degenerate     bool            // whether the Atropos is not applied
		blockCheaters  lachesis.Cheaters
		storedCheaters lachesis.Cheaters
		skipped        bool
	}{
		"distributed: no events": {
			atroposTime: beginBlockTime(time.Second),
			skipped:     true,
		},
		"distributed: no events just before the end of the skip period": {
			atroposTime: endOfSkipPeriod - 1,
			skipped:     true,
		},
		"distributed: no events at the end of the skip period": {
			atroposTime: endOfSkipPeriod,
			skipped:     false,
		},
		"distributed: skip period measured from the last block of the block state": {
			atroposTime: endOfSkipPeriod - 1,
			parentTime:  beginBlockTime(-time.Second),
			skipped:     true,
		},
		"distributed: skip period measured from the clamped block time": {
			atroposTime: beginBlockTime(-time.Second),
			skipPeriod:  1,
			skipped:     false,
		},
		"distributed: skip period overflowing the block time": {
			atroposTime: beginBlockTime(time.Second),
			skipPeriod:  math.MaxUint64,
			skipped:     false,
		},
		"distributed: cheaters of the block": {
			atroposTime:   beginBlockTime(time.Second),
			blockCheaters: lachesis.Cheaters{2},
			skipped:       false,
		},
		"distributed: stored cheaters only": {
			atroposTime:    beginBlockTime(time.Second),
			storedCheaters: lachesis.Cheaters{2},
			skipped:        true,
		},
		"distributed: event without transactions": {
			events:      withPayload(beginBlockEventSpec{}),
			atroposTime: beginBlockTime(time.Second),
			skipped:     true,
		},
		"distributed: event with transactions": {
			events:      withTx,
			atroposTime: beginBlockTime(time.Second),
			skipped:     false,
		},
		"distributed: event with empty proposal": {
			events:      withPayload(beginBlockEventSpec{proposal: &inter.Proposal{}}),
			atroposTime: beginBlockTime(time.Second),
			skipped:     false,
		},
		"distributed: event with non-permissible transactions only": {
			events:      withPayload(beginBlockEventSpec{txs: types.Transactions{beginBlockNonPermissibleTx(1, 0)}}),
			atroposTime: beginBlockTime(time.Second),
			skipped:     false,
		},
		"distributed: degenerate atropos with transactions": {
			events:      withTx,
			atroposTime: beginBlockTime(time.Second),
			degenerate:  true,
			skipped:     true,
		},
		"distributed: degenerate atropos with cheaters at the end of the skip period": {
			atroposTime:   endOfSkipPeriod,
			degenerate:    true,
			blockCheaters: lachesis.Cheaters{2},
			skipped:       true,
		},
		"single proposer: valid proposal": {
			singleProposer: true,
			events:         withProposal(beginBlockTime(time.Second), beginBlockTx(1, 0)),
			atroposTime:    beginBlockTime(time.Second),
			skipped:        false,
		},
		"single proposer: empty proposal": {
			singleProposer: true,
			events:         withProposal(beginBlockTime(time.Second)),
			atroposTime:    beginBlockTime(time.Second),
			skipped:        true,
		},
		"single proposer: empty proposal at the end of the skip period": {
			singleProposer: true,
			events:         withProposal(endOfSkipPeriod),
			atroposTime:    beginBlockTime(time.Second),
			skipped:        false,
		},
		"single proposer: proposal with transactions for a future block": {
			singleProposer: true,
			events: func(env *beginBlockEnv) []*inter.EventPayload {
				proposal := env.proposal(beginBlockTx(1, 0))
				proposal.Number++
				return withPayload(beginBlockEventSpec{proposal: proposal})(env)
			},
			atroposTime: beginBlockTime(time.Second),
			skipped:     true,
		},
		"single proposer: transactions outside of proposals": {
			singleProposer: true,
			events:         withTx,
			atroposTime:    beginBlockTime(time.Second),
			skipped:        true,
		},
		"single proposer: proposal with non-permissible transactions only": {
			singleProposer: true,
			events:         withProposal(beginBlockTime(time.Second), beginBlockNonPermissibleTx(1, 0)),
			atroposTime:    beginBlockTime(time.Second),
			skipped:        true,
		},
		"single proposer: cheaters of the block": {
			singleProposer: true,
			atroposTime:    beginBlockTime(time.Second),
			blockCheaters:  lachesis.Cheaters{2},
			skipped:        false,
		},
		"single proposer: stored cheaters only": {
			singleProposer: true,
			atroposTime:    beginBlockTime(time.Second),
			storedCheaters: lachesis.Cheaters{2},
			skipped:        true,
		},
		"single proposer: no proposal at the end of the skip period": {
			singleProposer: true,
			atroposTime:    endOfSkipPeriod,
			skipped:        false,
		},
		"single proposer: degenerate atropos with valid proposal": {
			singleProposer: true,
			events:         withProposal(beginBlockTime(time.Second), beginBlockTx(1, 0)),
			atroposTime:    beginBlockTime(time.Second),
			degenerate:     true,
			skipped:        true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t, func(c *beginBlockChain) {
				c.es.Rules.Upgrades.SingleProposerBlockFormation = test.singleProposer
				c.bs.EpochCheaters = test.storedCheaters
				c.parentTime = test.parentTime
				if test.skipPeriod != 0 {
					c.es.Rules.Blocks.MaxEmptyBlockSkipPeriod = test.skipPeriod
				}
			})
			events := []*inter.EventPayload{}
			if test.events != nil {
				events = test.events(env)
			}
			atropos := beginBlockAtropos(test.atroposTime)
			if !test.degenerate {
				events = append(events, atropos)
			}

			env.runBlock(&lachesis.Block{Atropos: atropos.ID(), Cheaters: test.blockCheaters}, events...)

			require.Equal(t, test.skipped, env.blockSkipped())
		})
	}
}

func TestBeginBlockFn_SkippedBlock_OnlyFinalizesEventProcessingAndStoresItsState(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		degenerate bool
		cheaters   lachesis.Cheaters
		wantTime   inter.Timestamp
	}{
		"empty block": {
			degenerate: false,
			wantTime:   beginBlockTime(time.Second),
		},
		"degenerate atropos with cheaters": {
			degenerate: true,
			cheaters:   lachesis.Cheaters{2},
			wantTime:   beginBlockLastBlockTime + 1,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t)
			updates := env.withFeed()
			env.proc.sealing = true
			atropos := beginBlockAtropos(beginBlockTime(time.Second))
			events := []*inter.EventPayload{atropos}
			if test.degenerate {
				events = []*inter.EventPayload{beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(1, 0)}})}
			}

			require.Nil(t, env.runBlock(beginBlockOf(atropos, test.cheaters...), events...))

			// Even a block sealing the epoch is not sealed when skipped.
			require.Equal(t, []string{"Events.Start", "Events.ProcessConfirmedEvent", "Events.Finalize"}, env.proc.trace())
			require.Equal(t, beginBlockEventsFinalize{
				Ctx:     iblockproc.BlockCtx{Idx: env.nextBlock(), Time: test.wantTime, Atropos: atropos.ID()},
				Skipped: true,
			}, beginBlockCalls[beginBlockEventsFinalize](env.proc)[0])

			// The cheaters of the skipped block are stored.
			bs, es := env.store.GetBlockEpochState()
			require.ElementsMatch(t, test.cheaters, bs.EpochCheaters)
			want := beginBlockCalls[beginBlockEventsStart](env.proc)[0].BS
			want.EpochGas = beginBlockTagEvents
			require.Equal(t, want.Hash(), bs.Hash())
			require.Equal(t, env.chain.es.Hash(), es.Hash())
			require.Nil(t, env.store.GetBlock(env.nextBlock()))
			require.Empty(t, updates)
			require.Zero(t, env.busy)
		})
	}
}

func TestBeginBlockFn_SkippedBlock_NextBlockStartsFromStoredState(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t)
	atropos := beginBlockAtropos(beginBlockTime(time.Second))
	env.runBlock(beginBlockOf(atropos), atropos)
	env.proc.reset()

	env.fn()(&lachesis.Block{})

	require.Equal(t, uint64(beginBlockTagEvents), beginBlockCalls[beginBlockEventsStart](env.proc)[0].BS.EpochGas)
}

// ---------------------------------------------------------------------------
// EndBlock: transaction priorities
// ---------------------------------------------------------------------------

func TestBeginBlockFn_AppliesTransactionPrioritiesToUserTransactions(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		singleProposer bool
		priorities     bool
	}{
		"distributed without priorities":     {},
		"distributed with priorities":        {priorities: true},
		"single proposer without priorities": {singleProposer: true},
		"single proposer with priorities":    {singleProposer: true, priorities: true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t, func(c *beginBlockChain) {
				c.es.Rules.Upgrades.SingleProposerBlockFormation = test.singleProposer
				c.es.Rules.Upgrades.TransactionPriorities = test.priorities
			})
			plain, prioritized := beginBlockTx(1, 0), beginBlockTx(2, 0)
			sender, err := types.Sender(beginBlockSigner, prioritized)
			require.NoError(t, err)
			env.setStateRoot(beginBlockCommitState(t, env.store, env.chain.bs.FinalizedStateRoot, func(statedb state.StateDB) {
				statedb.SetCode(registry.GetAddress(), registry.GetCode(), tracing.CodeChangeUnspecified)
				statedb.SetState(registry.GetAddress(), senderPriorityStorageSlot(sender), packSenderPriority(1, 0, 1))
			}))
			spec := beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{plain, prioritized}}
			if test.singleProposer {
				spec.txs, spec.proposal = nil, env.proposal(plain, prioritized)
			}
			event := beginBlockEvent(spec)

			env.runBlock(beginBlockOf(event), event)

			// The scrambler keeps the order of these transactions.
			want := []common.Hash{plain.Hash(), prioritized.Hash()}
			if test.priorities {
				want = []common.Hash{prioritized.Hash(), plain.Hash()}
			}
			require.Equal(t, want, beginBlockHashes(env.userTxs()))
		})
	}
}

// beginBlockContextRegistryCode returns the code of a priority registry with a
// permissive configuration, prioritizing the sender stored in slot 1 if the
// given block context opcode yields the value stored in slot 0. For BLOCKHASH,
// the hash of the given ancestor of the block is queried.
func beginBlockContextRegistryCode(op vm.OpCode, ancestor uint8) []byte {
	context := []byte{byte(op)}
	switch op {
	case vm.BLOCKHASH:
		context = []byte{byte(vm.PUSH1), ancestor, byte(vm.NUMBER), byte(vm.SUB), byte(vm.BLOCKHASH)}
	case vm.CLZ:
		// The number of leading zeros of 1, an Osaka feature.
		context = []byte{byte(vm.PUSH1), 1, byte(vm.CLZ)}
	case vm.CALL:
		// The success of calling the EVM writer with 10_000 gas, which only the driver may call.
		context = []byte{byte(vm.PUSH1), 0, byte(vm.PUSH1), 0, byte(vm.PUSH1), 0, byte(vm.PUSH1), 0, byte(vm.PUSH1), 0, byte(vm.PUSH20)}
		context = append(context, evmwriter.ContractAddress.Bytes()...)
		context = append(context, byte(vm.PUSH2), 0x27, 0x10, byte(vm.CALL))
	}
	code := []byte{
		// getPriorityConfig is the only query without arguments.
		byte(vm.CALLDATASIZE), byte(vm.PUSH1), 4, byte(vm.EQ), byte(vm.PUSH1), 0, byte(vm.JUMPI),
	}
	// getPriority: level and id are 1 if both conditions hold, 0 otherwise.
	code = append(code, context...)
	code = append(code,
		byte(vm.PUSH1), 0, byte(vm.SLOAD), byte(vm.EQ),
		byte(vm.PUSH1), 4, byte(vm.CALLDATALOAD), byte(vm.PUSH1), 1, byte(vm.SLOAD), byte(vm.EQ), byte(vm.AND),
		byte(vm.DUP1), byte(vm.PUSH1), 0, byte(vm.MSTORE), byte(vm.PUSH1), 64, byte(vm.MSTORE),
		byte(vm.PUSH1), 96, byte(vm.PUSH1), 0, byte(vm.RETURN),
	)
	code[5] = byte(len(code))
	return append(code,
		// getPriorityConfig: 10_000_000 gas and 4 transactions per entity.
		byte(vm.JUMPDEST), byte(vm.PUSH3), 0x98, 0x96, 0x80, byte(vm.PUSH1), 0, byte(vm.MSTORE),
		byte(vm.PUSH1), 4, byte(vm.PUSH1), 32, byte(vm.MSTORE),
		byte(vm.PUSH1), 64, byte(vm.PUSH1), 0, byte(vm.RETURN),
	)
}

func TestBeginBlockFn_QueriesTransactionPrioritiesInContextOfTheBlock(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		op           vm.OpCode
		ancestor     uint8     // ancestor of the block whose hash is queried by BLOCKHASH
		parentNumber idx.Block // number reported by the stored parent block; 0 means its index
		distributed  bool
		wrong        bool // whether the registry expects a different value
	}{
		"block number":                           {op: vm.NUMBER},
		"block number of the proposal":           {op: vm.NUMBER, parentNumber: 20},
		"block number without proposal":          {op: vm.NUMBER, parentNumber: 20, distributed: true},
		"block time":                             {op: vm.TIMESTAMP},
		"coinbase":                               {op: vm.COINBASE},
		"randao of the block":                    {op: vm.PREVRANDAO},
		"base fee":                               {op: vm.BASEFEE},
		"blob base fee":                          {op: vm.BLOBBASEFEE},
		"max block gas of the rules":             {op: vm.GASLIMIT},
		"chain ID":                               {op: vm.CHAINID},
		"chain config of the block":              {op: vm.CLZ},
		"parent block hash":                      {op: vm.BLOCKHASH, ancestor: 1},
		"hash of the block before the parent":    {op: vm.BLOCKHASH, ancestor: 2},
		"precompiled contracts of the Sonic EVM": {op: vm.CALL},
		"different block time":                   {op: vm.TIMESTAMP, wrong: true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// The order of the proposal, and in the distributed mode the order of
			// the scrambler, is kept unless a transaction is prioritized.
			env := newBeginBlockEnv(t, func(c *beginBlockChain) {
				c.es.Rules.Upgrades.SingleProposerBlockFormation = !test.distributed
				c.es.Rules.Upgrades.TransactionPriorities = true
				c.es.Rules.Economy.ShortGasPower.AllocPerSec = 4 * beginBlockMaxBlockGas
				c.parentNumber = uint64(test.parentNumber)
			})
			// The rules differ from those the stored parent block was created with.
			env.chain.es.Rules.Blocks.MaxBlockGas = 2 * beginBlockMaxBlockGas
			env.store.SetBlock(beginBlockLastBlock-1, inter.NewBlockBuilder().WithNumber(uint64(beginBlockLastBlock-1)).Build())
			// Osaka is enabled by Brio for this block only.
			env.store.AddUpgradeHeight(opera.UpgradeHeight{Upgrades: opera.GetSonicUpgrades(), Height: 1})
			env.store.AddUpgradeHeight(opera.UpgradeHeight{Upgrades: opera.GetBrioUpgrades(), Height: env.nextBlock()})
			env.store.AddUpgradeHeight(opera.UpgradeHeight{Upgrades: opera.GetAllegroUpgrades(), Height: env.nextBlock() + 1})
			plain, prioritized := beginBlockTx(1, 0), beginBlockTx(2, 0)
			sender, err := types.Sender(beginBlockSigner, prioritized)
			require.NoError(t, err)
			proposal := env.proposal(plain, prioritized)
			reveal, mix := beginBlockRandaoReveal(t, 1, beginBlockParentRandao)
			proposal.RandaoReveal = reveal
			blockTime := beginBlockTime(2*time.Second - 1)
			spec := beginBlockEventSpec{creator: 1, lamport: 1, time: blockTime, proposal: proposal}
			index := proposal.Number
			if test.distributed {
				spec.txs, spec.proposal = proposal.Transactions, nil
				index = env.nextBlock()
			}
			event := beginBlockEvent(spec)
			// The Atropos time and the randao of the events differ from those of the block.
			atropos := beginBlockAtropos(beginBlockTime(3 * time.Second))

			value := map[vm.OpCode]*big.Int{
				vm.NUMBER:     big.NewInt(int64(index)),
				vm.COINBASE:   new(big.Int).SetBytes(evmcore.GetCoinbase().Bytes()),
				vm.CALL:       big.NewInt(0),
				vm.TIMESTAMP:  big.NewInt(blockTime.Unix()),
				vm.PREVRANDAO: mix.Big(),
				// Derived from the base fee, gas used and duration of the parent block.
				vm.BASEFEE:     big.NewInt(9_941_577_576),
				vm.BLOBBASEFEE: evmcore.GetBlobBaseFee().ToBig(),
				vm.GASLIMIT:    new(big.Int).SetUint64(2 * beginBlockMaxBlockGas),
				vm.CHAINID:     new(big.Int).SetUint64(opera.FakeNetworkID),
				vm.CLZ:         big.NewInt(255),
			}[test.op]
			if test.op == vm.BLOCKHASH {
				value = env.store.GetBlock(env.nextBlock() - idx.Block(test.ancestor)).Hash().Big()
			}
			if test.wrong {
				value.Add(value, big.NewInt(1))
			}
			env.setStateRoot(beginBlockCommitState(t, env.store, env.chain.bs.FinalizedStateRoot, func(statedb state.StateDB) {
				statedb.SetCode(registry.GetAddress(), beginBlockContextRegistryCode(test.op, test.ancestor), tracing.CodeChangeUnspecified)
				statedb.SetState(registry.GetAddress(), common.Hash{0}, common.BigToHash(value))
				statedb.SetState(registry.GetAddress(), common.Hash{31: 1}, common.BytesToHash(sender.Bytes()))
			}))

			env.runBlock(beginBlockOf(atropos), event, atropos)

			want := []common.Hash{prioritized.Hash(), plain.Hash()}
			if test.wrong {
				want = []common.Hash{plain.Hash(), prioritized.Hash()}
			}
			require.Equal(t, want, beginBlockHashes(env.userTxs()))
		})
	}
}

func TestBeginBlockFn_QueriesTransactionPrioritiesBeforeStartingTheEvm(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.es.Rules.Upgrades.TransactionPriorities = true })
	plain, prioritized := beginBlockTx(1, 0), beginBlockTx(2, 0)
	sender, err := types.Sender(beginBlockSigner, prioritized)
	require.NoError(t, err)
	slot := senderPriorityStorageSlot(sender)
	env.setStateRoot(beginBlockCommitState(t, env.store, env.chain.bs.FinalizedStateRoot, func(statedb state.StateDB) {
		statedb.SetCode(registry.GetAddress(), registry.GetCode(), tracing.CodeChangeUnspecified)
		statedb.SetState(registry.GetAddress(), slot, packSenderPriority(1, 0, 1))
	}))
	// Starting the EVM, which begins the block in the state, removes the priority.
	env.proc.hook = func(call any) {
		if start, ok := call.(beginBlockEvmStart); ok {
			start.StateDB.SetState(registry.GetAddress(), slot, common.Hash{})
		}
	}
	event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{plain, prioritized}})

	env.runBlock(beginBlockOf(event), event)

	require.Equal(t, beginBlockHashes(types.Transactions{prioritized, plain}), beginBlockHashes(env.userTxs()))
}

func TestBeginBlockFn_RevertsStateChangesOfThePriorityConfigQuery(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.es.Rules.Upgrades.TransactionPriorities = true })
	written := common.Hash{31: 2}
	env.setStateRoot(beginBlockCommitState(t, env.store, env.chain.bs.FinalizedStateRoot, func(statedb state.StateDB) {
		// A registry writing 1 to slot 2 if getPriorityConfig, the only query
		// without arguments, is called.
		code := []byte{
			byte(vm.CALLDATASIZE), byte(vm.PUSH1), 4, byte(vm.EQ), byte(vm.ISZERO), byte(vm.PUSH1), 13, byte(vm.JUMPI),
			byte(vm.PUSH1), 1, byte(vm.PUSH1), 2, byte(vm.SSTORE),
			byte(vm.JUMPDEST),
		}
		statedb.SetCode(registry.GetAddress(), code, tracing.CodeChangeUnspecified)
	}))
	event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(1, 0)}})

	env.runBlock(beginBlockOf(event), event)

	require.Equal(t, common.Hash{}, env.evmStart().StateDB.GetState(registry.GetAddress(), written))
}

func TestBeginBlockFn_QueriesTransactionPrioritiesOnlyForProducedBlocksWithUserTransactions(t *testing.T) {
	// Not parallel, as the metrics are global.
	tests := map[string]struct {
		degenerate     bool
		txs            types.Transactions
		wantConfigs    int64
		wantPriorities int64
	}{
		"produced block": {
			degenerate:     false,
			txs:            types.Transactions{beginBlockTx(1, 0), beginBlockTx(2, 0)},
			wantConfigs:    1,
			wantPriorities: 2,
		},
		"produced block without user transactions": {
			degenerate: false,
		},
		"skipped block": {
			degenerate: true,
			txs:        types.Transactions{beginBlockTx(1, 0), beginBlockTx(2, 0)},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Without a deployed registry, every query fails.
			env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.es.Rules.Upgrades.TransactionPriorities = true })
			event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: test.txs})
			atropos := beginBlockAtropos(beginBlockTime(time.Second))
			events := []*inter.EventPayload{event, atropos}
			if test.degenerate {
				events = events[:1]
			}

			configFailures := priorityFailures.config.(*metrics.Meter)
			txFailures := priorityFailures.txs.(*metrics.Meter)
			configBefore, txsBefore := configFailures.Snapshot().Count(), txFailures.Snapshot().Count()
			// The cheater makes sure the block is produced unless the Atropos is degenerate.
			env.runBlock(beginBlockOf(atropos, 2), events...)

			require.Equal(t, test.wantConfigs, configFailures.Snapshot().Count()-configBefore)
			require.Equal(t, test.wantPriorities, txFailures.Snapshot().Count()-txsBefore)
		})
	}
}

// ---------------------------------------------------------------------------
// EndBlock: producing blocks
// ---------------------------------------------------------------------------

// runProducedBlock runs a block with a single event carrying (or proposing)
// one user transaction, using an Atropos one second after the last block.
func (env *beginBlockEnv) runProducedBlock() (event, atropos *inter.EventPayload) {
	user := beginBlockTx(1, 0)
	spec := beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{user}}
	if env.chain.es.Rules.Upgrades.SingleProposerBlockFormation {
		spec.txs, spec.proposal = nil, env.proposal(user)
	}
	event = beginBlockEvent(spec)
	atropos = beginBlockAtropos(beginBlockTime(time.Second))
	env.runBlock(beginBlockOf(atropos), event, atropos)
	return event, atropos
}

func TestBeginBlockFn_ProducedBlock_CallsModulesInOrder(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		singleProposer bool
		sealing        bool
	}{
		"distributed":              {},
		"distributed, sealing":     {sealing: true},
		"single proposer":          {singleProposer: true},
		"single proposer, sealing": {singleProposer: true, sealing: true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.es.Rules.Upgrades.SingleProposerBlockFormation = test.singleProposer })
			env.proc.sealing = test.sealing
			env.proc.preTxs = types.Transactions{beginBlockTx(3, 0)}
			env.proc.postTxs = types.Transactions{beginBlockTx(3, 1)}

			env.runProducedBlock()

			want := []string{
				"Events.Start",
				"Events.ProcessConfirmedEvent",
				"Events.ProcessConfirmedEvent",
				"Events.Finalize",
				"Sealer.Start",
				"Sealer.EpochSealing",
				"TxListener.Start",
				"EVM.Start",
				"PreTx.PopInternalTxs",
				"EVM.Execute",
				"TxListener.Finalize",
			}
			if test.sealing {
				want = append(want,
					"Sealer.Update",
					"Sealer.SealEpoch",
					"TxListener.Update",
				)
			}
			want = append(want,
				"PostTx.PopInternalTxs",
				"EVM.Execute",
				"EVM.Execute",
				"EVM.Finalize",
				"TxListener.OnNewReceipt",
				"TxListener.OnNewReceipt",
				"TxListener.OnNewReceipt",
				"TxListener.Finalize",
			)
			require.Equal(t, want, env.proc.trace())
		})
	}
}

func TestBeginBlockFn_ProducedBlock_PassesBlockContextAndStatesBetweenModules(t *testing.T) {
	t.Parallel()
	tests := map[string]bool{
		"without sealing": false,
		"with sealing":    true,
	}
	for name, sealing := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t)
			env.proc.sealing = sealing

			_, atropos := env.runProducedBlock()

			ctx := iblockproc.BlockCtx{Idx: env.nextBlock(), Time: beginBlockTime(time.Second), Atropos: atropos.ID()}
			require.Equal(t, beginBlockEventsFinalize{Ctx: ctx, Skipped: false}, beginBlockCalls[beginBlockEventsFinalize](env.proc)[0])

			sealerStart := beginBlockCalls[beginBlockSealerStart](env.proc)[0]
			require.Equal(t, ctx, sealerStart.Ctx)
			require.Equal(t, uint64(beginBlockTagEvents), sealerStart.BS.EpochGas)
			require.Equal(t, beginBlockStoreEsTag, sealerStart.ES.EpochStateRoot)

			listenerStart := beginBlockCalls[beginBlockListenerStart](env.proc)[0]
			require.Equal(t, ctx, listenerStart.Ctx)
			require.Equal(t, uint64(beginBlockTagEvents), listenerStart.BS.EpochGas)
			require.Equal(t, beginBlockStoreEsTag, listenerStart.ES.EpochStateRoot)

			pops := beginBlockCalls[beginBlockPopInternalTxs](env.proc)
			require.Len(t, pops, 2)
			require.Equal(t, "PreTx", pops[0].Phase)
			require.Equal(t, ctx, pops[0].Ctx)
			require.Equal(t, uint64(beginBlockTagEvents), pops[0].BS.EpochGas)
			require.Equal(t, beginBlockStoreEsTag, pops[0].ES.EpochStateRoot)
			require.Equal(t, sealing, pops[0].Sealing)

			postBsTag, postEsTag := uint64(beginBlockTagListener1), beginBlockStoreEsTag
			if sealing {
				update := beginBlockCalls[beginBlockSealerUpdate](env.proc)[0]
				require.Equal(t, uint64(beginBlockTagListener1), update.BS.EpochGas)
				require.Equal(t, beginBlockStoreEsTag, update.ES.EpochStateRoot)
				listenerUpdate := beginBlockCalls[beginBlockListenerUpdate](env.proc)[0]
				require.Equal(t, uint64(beginBlockTagSealed), listenerUpdate.BS.EpochGas)
				require.Equal(t, beginBlockSealedEsTag, listenerUpdate.ES.EpochStateRoot)
				postBsTag, postEsTag = beginBlockTagSealed, beginBlockSealedEsTag
			}
			require.Equal(t, "PostTx", pops[1].Phase)
			require.Equal(t, ctx, pops[1].Ctx)
			require.Equal(t, postBsTag, pops[1].BS.EpochGas)
			require.Equal(t, postEsTag, pops[1].ES.EpochStateRoot)
			require.Equal(t, sealing, pops[1].Sealing)
		})
	}
}

func TestBeginBlockFn_ProducedBlock_StartsEvmWithBlockParameters(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t)
	env.withFeed()
	event, _ := env.runProducedBlock()

	start := env.evmStart()
	require.Equal(t, env.nextBlock(), start.Number)
	require.Equal(t, beginBlockTime(time.Second), start.Time)
	require.Equal(t, beginBlockEpoch, start.Epoch)
	require.Equal(t, common.Hash(env.chain.bs.FinalizedStateRoot), start.StateDB.GetStateHash())
	reader, ok := start.Reader.(*EvmStateReader)
	require.True(t, ok)
	require.Same(t, env.store, reader.store)
	require.Same(t, env.feed, reader.ServiceFeed)
	require.Nil(t, reader.gpo)
	require.Equal(t, env.chain.es.Rules, start.Rules)
	past := uint64(0)
	require.Equal(t, &params.ChainConfig{
		ChainID:             new(big.Int).SetUint64(opera.FakeNetworkID),
		HomesteadBlock:      big.NewInt(0),
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
		ShanghaiTime:        &past,
		CancunTime:          &past,
	}, start.ChainCfg)
	require.Equal(t, beginBlockPrevRandao(event.ID()), start.Randao)
	require.Same(t, sonicFeaturesMetrics, start.Metrics)
}

func TestBeginBlockFn_ProducedBlock_TakesEpochFromAtropos(t *testing.T) {
	t.Parallel()
	const atroposEpoch = idx.Epoch(9)
	env := newBeginBlockEnv(t)
	event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(1, 0)}})
	atropos := beginBlockEvent(beginBlockEventSpec{epoch: atroposEpoch, creator: 3, lamport: 1, time: beginBlockTime(time.Second)})

	env.runBlock(beginBlockOf(atropos), event, atropos)

	require.Equal(t, atroposEpoch, env.evmStart().Epoch)
	require.Equal(t, atroposEpoch, env.store.GetBlock(env.nextBlock()).Epoch)
}

func TestBeginBlockFn_ProducedBlock_ExecutesInternalAndUserTransactionsWithLimits(t *testing.T) {
	t.Parallel()
	pre := types.Transactions{beginBlockTx(3, 0), beginBlockTx(3, 1)}
	post := types.Transactions{beginBlockTx(3, 2), beginBlockTx(3, 3)}
	oversized := types.Transactions{types.MustSignNewTx(makefakegenesis.FakeKey(3), beginBlockSigner, &types.LegacyTx{
		Gas:      21_000,
		GasPrice: big.NewInt(1),
		Data:     make([]byte, params.MaxBlockSize),
	})}
	// The size of the block header is bounded by 1024 bytes.
	tests := map[string]struct {
		upgrades opera.Upgrades
		pre      types.Transactions
		wantSize uint64
	}{
		"before Brio, the size is not limited": {
			upgrades: opera.GetAllegroUpgrades(),
			pre:      pre,
			wantSize: math.MaxUint64,
		},
		"with Brio, the size of included internal transactions is deducted": {
			upgrades: opera.GetBrioUpgrades(),
			pre:      pre,
			wantSize: params.MaxBlockSize - 1024 - pre[0].Size() - post[0].Size(),
		},
		"with Brio, internal transactions exceeding the block size leave no size": {
			upgrades: opera.GetBrioUpgrades(),
			pre:      oversized,
			wantSize: 0,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.setUpgrades(test.upgrades) })
			env.proc.preTxs, env.proc.postTxs = test.pre, post
			env.proc.skipped = map[common.Hash]bool{pre[1].Hash(): true, post[1].Hash(): true}
			user := beginBlockTx(1, 0)
			event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{user}})

			env.runBlock(beginBlockOf(event), event)

			executions := beginBlockCalls[beginBlockExecute](env.proc)
			require.Len(t, executions, 3)
			require.Equal(t, beginBlockHashes(test.pre), beginBlockHashes(executions[0].Txs))
			require.Equal(t, beginBlockMaxBlockGas, executions[0].GasLimit)
			require.Equal(t, uint64(math.MaxUint64), executions[0].SizeLimit)
			require.Equal(t, beginBlockHashes(post), beginBlockHashes(executions[1].Txs))
			require.Equal(t, beginBlockMaxBlockGas, executions[1].GasLimit)
			require.Equal(t, uint64(math.MaxUint64), executions[1].SizeLimit)
			require.Equal(t, beginBlockHashes(types.Transactions{user}), beginBlockHashes(executions[2].Txs))
			require.Equal(t, beginBlockMaxBlockGas, executions[2].GasLimit)
			require.Equal(t, test.wantSize, executions[2].SizeLimit)
		})
	}
}

func TestBeginBlockFn_ProducedBlock_ProvidesNoncesOfTheExecutedStateToInternalTransactions(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t)
	env.setStateRoot(beginBlockCommitState(t, env.store, env.chain.bs.FinalizedStateRoot, func(statedb state.StateDB) {
		statedb.SetNonce(common.Address{}, 42, tracing.NonceChangeUnspecified)
	}))
	executed, skipped := beginBlockTx(3, 0), beginBlockTx(3, 1)
	env.proc.preTxs = types.Transactions{executed, skipped}
	env.proc.skipped = map[common.Hash]bool{skipped.Hash(): true}

	env.runProducedBlock()

	// The internal transactions executed before increment the nonce.
	pops := beginBlockCalls[beginBlockPopInternalTxs](env.proc)
	require.Len(t, pops, 2)
	require.Equal(t, uint64(42), pops[0].Nonce)
	require.Equal(t, uint64(43), pops[1].Nonce)
}

func TestBeginBlockFn_ProducedBlock_ForwardsLogsToTxListenerAndVersionWatcher(t *testing.T) {
	t.Parallel()
	tests := map[string]bool{
		"with version watcher":    true,
		"without version watcher": false,
	}
	for name, withWatcher := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t)
			var versions *verwatcher.Store
			if withWatcher {
				versions = env.withVersionWatcher()
			}
			user := beginBlockTx(1, 0)
			entry := &types.Log{
				Address: driver.ContractAddress,
				Topics:  []common.Hash{driverpos.Topics.UpdateNetworkVersion},
				Data:    common.LeftPadBytes([]byte{77}, 32),
			}
			env.proc.logs = map[common.Hash][]*types.Log{user.Hash(): {entry}}
			event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{user}})

			env.runBlock(beginBlockOf(event), event)

			logs := beginBlockCalls[beginBlockListenerLog](env.proc)
			require.Len(t, logs, 1)
			require.Equal(t, core_types.CoreLogFromGethLog(entry), logs[0].Log)
			if withWatcher {
				require.Equal(t, uint64(77), versions.GetNetworkVersion())
			}
		})
	}
}

func TestBeginBlockFn_Sealing_RecordsUpgradeHeightIfUpgradesChange(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		modify   func(*opera.Rules)
		recorded bool
	}{
		"unchanged rules": {
			modify:   func(*opera.Rules) {},
			recorded: false,
		},
		"changed rules without upgrades": {
			modify:   func(rules *opera.Rules) { rules.Blocks.MaxBlockGas++ },
			recorded: false,
		},
		"changed upgrades": {
			modify:   func(rules *opera.Rules) { rules.Upgrades.TransactionBundles = true },
			recorded: true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t)
			existing := opera.UpgradeHeight{Upgrades: opera.GetSonicUpgrades(), Height: 1, Time: 2}
			env.store.AddUpgradeHeight(existing)
			env.proc.sealing = true
			env.proc.sealResult = func(bs iblockproc.BlockState, es iblockproc.EpochState) (iblockproc.BlockState, iblockproc.EpochState) {
				test.modify(&es.Rules)
				return bs, es
			}
			// The time of the event is clamped to the block time after the last block.
			event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(-time.Second), txs: types.Transactions{beginBlockTx(1, 0)}})

			env.runBlock(beginBlockOf(event), event)

			want := []opera.UpgradeHeight{existing}
			if test.recorded {
				upgrades := env.chain.es.Rules.Upgrades
				upgrades.TransactionBundles = true
				want = append(want, opera.UpgradeHeight{Upgrades: upgrades, Height: env.nextBlock() + 1, Time: beginBlockLastBlockTime + 2})
			}
			require.Equal(t, want, env.store.GetUpgradeHeights())
		})
	}
}

func TestBeginBlockFn_Sealing_SealsEpochWithParentHashExecutionPlanHashAndInternalTransactions(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t)
	env.store.AddProcessedBundles(5, map[common.Hash]bundle.PositionInBlock{{0x01}: {Offset: 0, Count: 1}})
	_, execPlanHash := env.store.GetLatestProcessedBundleHistoryHash()
	require.NotEqual(t, common.Hash{}, execPlanHash)
	env.proc.sealing = true
	pre := types.Transactions{beginBlockTx(3, 0), beginBlockTx(3, 1)}
	env.proc.preTxs = pre
	// Internal transactions skipped by the EVM are still handed to the sealer.
	env.proc.skipped = map[common.Hash]bool{pre[1].Hash(): true}

	env.runProducedBlock()

	seals := beginBlockCalls[beginBlockSealEpoch](env.proc)
	require.Len(t, seals, 1)
	require.Equal(t, hash.Hash(env.parentHash()), seals[0].LastBlockHash)
	require.Equal(t, hash.Hash(execPlanHash), seals[0].ExecPlanHash)
	require.Equal(t, beginBlockHashes(pre), beginBlockHashes(seals[0].Txs))
}

func TestBeginBlockFn_Sealing_StoresSealedStatesBeforeUpdatingTheTxListener(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t)
	env.proc.sealing = true
	var bs iblockproc.BlockState
	var es iblockproc.EpochState
	env.proc.hook = func(call any) {
		if _, ok := call.(beginBlockListenerUpdate); ok {
			bs, es = env.store.GetBlockEpochState()
		}
	}

	env.runProducedBlock()

	require.Equal(t, uint64(beginBlockTagSealed), bs.EpochGas)
	require.Equal(t, env.chain.bs.LastBlock, bs.LastBlock)
	require.Equal(t, beginBlockSealedEsTag, es.EpochStateRoot)
}

func TestBeginBlockFn_Sealing_ReturnsValidatorsOfSealedEpoch(t *testing.T) {
	t.Parallel()
	newValidators, _ := beginBlockValidators(2, 3)
	tests := map[string]struct {
		sealing       bool
		newValidators *pos.Validators // nil means the validators are not changed
	}{
		"without sealing": {
			sealing: false,
		},
		"with sealing and unchanged validators": {
			sealing: true,
		},
		"with sealing and new validators": {
			sealing:       true,
			newValidators: newValidators,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t)
			env.proc.sealing = test.sealing
			if test.newValidators != nil {
				env.proc.sealResult = func(bs iblockproc.BlockState, es iblockproc.EpochState) (iblockproc.BlockState, iblockproc.EpochState) {
					es.Validators = test.newValidators
					return bs, es
				}
			}
			event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(1, 0)}})

			validators := env.runBlock(beginBlockOf(event), event)

			switch {
			case !test.sealing:
				require.Nil(t, validators)
			case test.newValidators == nil:
				require.Same(t, env.chain.es.Validators, validators)
			default:
				require.Same(t, test.newValidators, validators)
			}
		})
	}
}

func TestBeginBlockFn_Sealing_ExecutesWithMaxBlockGasOfBlockStartRules(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t)
	env.proc.sealing = true
	env.proc.sealResult = func(bs iblockproc.BlockState, es iblockproc.EpochState) (iblockproc.BlockState, iblockproc.EpochState) {
		es.Rules.Blocks.MaxBlockGas = 7
		return bs, es
	}

	env.runProducedBlock()

	for _, execution := range beginBlockCalls[beginBlockExecute](env.proc) {
		require.Equal(t, beginBlockMaxBlockGas, execution.GasLimit)
	}
	require.Equal(t, beginBlockMaxBlockGas, env.store.GetBlock(env.nextBlock()).GasLimit)
	require.Equal(t, uint64(7), env.store.GetEpochState().Rules.Blocks.MaxBlockGas)
}

func TestBeginBlockFn_Sealing_ProcessesUserTransactionsWithUpgradesOfBlockStartRules(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t)
	env.proc.sealing = true
	env.proc.sealResult = func(bs iblockproc.BlockState, es iblockproc.EpochState) (iblockproc.BlockState, iblockproc.EpochState) {
		es.Rules.Upgrades = opera.GetAllegroUpgrades()
		return bs, es
	}
	origin, derived := beginBlockTx(1, 0), beginBlockTx(3, 7)
	env.proc.derived = map[common.Hash]types.Transactions{origin.Hash(): {derived}}
	event := beginBlockEvent(beginBlockEventSpec{creator: 2, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{origin}})

	env.runBlock(beginBlockOf(event), event)

	// Brio of the ending epoch limits the block size and attributes derived transactions.
	require.Equal(t, uint64(params.MaxBlockSize-1024), beginBlockCalls[beginBlockExecute](env.proc)[2].SizeLimit)
	receipts := beginBlockCalls[beginBlockListenerReceipt](env.proc)
	require.Len(t, receipts, 2)
	require.Equal(t, derived.Hash(), receipts[1].Receipt.TxHash)
	require.Equal(t, idx.ValidatorID(2), receipts[1].Creator)
}

// ---------------------------------------------------------------------------
// EndBlock: processing the block
// ---------------------------------------------------------------------------

func TestBeginBlockFn_ProducedBlock_StoresBlockOfIncludedTransactions(t *testing.T) {
	t.Parallel()
	tests := map[string]bool{
		"distributed":     false,
		"single proposer": true,
	}
	for name, singleProposer := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.es.Rules.Upgrades.SingleProposerBlockFormation = singleProposer })
			pre, skippedPre, post := beginBlockTx(3, 0), beginBlockTx(3, 1), beginBlockTx(3, 2)
			env.proc.preTxs = types.Transactions{pre, skippedPre}
			env.proc.postTxs = types.Transactions{post}
			env.proc.skipped = map[common.Hash]bool{skippedPre.Hash(): true}
			// Failed internal transactions are included like successful ones.
			env.proc.failed = map[common.Hash]bool{pre.Hash(): true, post.Hash(): true}
			env.proc.evmRoot = hash.Hash{0x55}

			event, _ := env.runProducedBlock()

			receipts := beginBlockCalls[beginBlockListenerReceipt](env.proc)
			require.Len(t, receipts, 3)
			builder := inter.NewBlockBuilder().
				WithEpoch(beginBlockEpoch).
				WithNumber(uint64(env.nextBlock())).
				WithParentHash(env.parentHash()).
				WithTime(beginBlockTime(time.Second)).
				WithPrevRandao(env.evmStart().Randao).
				WithGasLimit(beginBlockMaxBlockGas).
				WithDuration(time.Second)
			for _, receipt := range receipts {
				builder.AddTransaction(receipt.Tx, receipt.Receipt)
			}
			want := builder.
				WithStateRoot(common.Hash{0x55}).
				WithGasUsed(3 * 21_000).
				WithBaseFee(big.NewInt(1_000)).
				Build()

			block := env.store.GetBlock(env.nextBlock())
			require.Equal(t, want.Hash(), block.Hash())
			require.Equal(t, beginBlockHashes(types.Transactions{pre, post, event.Transactions()[0]}), block.TransactionHashes)
			require.Equal(t, env.nextBlock(), *env.store.GetBlockIndex(hash.Event(block.Hash())))
		})
	}
}

func TestBeginBlockFn_ProducedBlock_MeasuresDurationFromLastBlockOfProcessedState(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		sealing       bool
		lastBlockTime inter.Timestamp // last block time in the state returned by the modules
		wantBlock     time.Duration
		wantEvmBlock  time.Duration
	}{
		"state returned by the events processor": {
			lastBlockTime: beginBlockTime(-4 * time.Second),
			wantBlock:     5 * time.Second,
			wantEvmBlock:  5 * time.Second,
		},
		"state returned by sealing the epoch": {
			sealing:       true,
			lastBlockTime: beginBlockTime(-6 * time.Second),
			wantBlock:     7 * time.Second,
			wantEvmBlock:  7 * time.Second,
		},
		"negative durations are clamped in the block only": {
			lastBlockTime: beginBlockTime(2 * time.Second),
			wantBlock:     0,
			wantEvmBlock:  -time.Second,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t)
			env.proc.sealing = test.sealing
			modify := func(bs iblockproc.BlockState) iblockproc.BlockState {
				bs.LastBlock.Time = test.lastBlockTime
				return bs
			}
			if test.sealing {
				env.proc.sealResult = func(bs iblockproc.BlockState, es iblockproc.EpochState) (iblockproc.BlockState, iblockproc.EpochState) {
					return modify(bs), es
				}
			} else {
				env.proc.eventsResult = modify
			}

			env.runProducedBlock()

			require.Equal(t, uint64(test.wantBlock), env.store.GetBlock(env.nextBlock()).Duration)
			require.Equal(t, test.wantEvmBlock, env.store.EvmStore().GetCachedEvmBlock(env.nextBlock()).Duration)
		})
	}
}

func TestBeginBlockFn_ProducedBlock_UsesGasLimitOfEndingEpochExceptForSonicBlock8054923(t *testing.T) {
	t.Parallel()
	const sealedMaxBlockGas = 7_000_000
	tests := map[string]struct {
		networkID    uint64
		number       idx.Block // number of the block, following the last block index
		parentNumber uint64    // number of the stored parent block in the single-proposer mode; 0 means its index
		sealing      bool
		want         uint64
	}{
		"sealing block 8054923 of other networks": {
			networkID: opera.FakeNetworkID,
			number:    8054923,
			sealing:   true,
			want:      beginBlockMaxBlockGas,
		},
		"sealing block 8054923 of Sonic": {
			networkID: 146,
			number:    8054923,
			sealing:   true,
			want:      sealedMaxBlockGas,
		},
		"sealing block 8054924 of Sonic": {
			networkID: 146,
			number:    8054924,
			sealing:   true,
			want:      beginBlockMaxBlockGas,
		},
		"non-sealing block 8054923 of Sonic": {
			networkID: 146,
			number:    8054923,
			sealing:   false,
			want:      beginBlockMaxBlockGas,
		},
		"sealing block 8054923 of Sonic stored at another index": {
			networkID:    146,
			number:       8054923,
			parentNumber: 20,
			sealing:      true,
			want:         sealedMaxBlockGas,
		},
		"sealing block stored at index 8054923 of Sonic": {
			networkID:    146,
			number:       21,
			parentNumber: 8054922,
			sealing:      true,
			want:         beginBlockMaxBlockGas,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t, func(c *beginBlockChain) {
				c.es.Rules.NetworkID = test.networkID
				c.bs.LastBlock.Idx = test.number - 1
				if test.parentNumber != 0 {
					c.singleProposer()
					c.parentNumber = test.parentNumber
				}
			})
			env.proc.sealing = test.sealing
			env.proc.sealResult = func(bs iblockproc.BlockState, es iblockproc.EpochState) (iblockproc.BlockState, iblockproc.EpochState) {
				es.Rules.Blocks.MaxBlockGas = sealedMaxBlockGas
				return bs, es
			}

			env.runProducedBlock()

			block := env.store.GetLatestBlock()
			require.Equal(t, uint64(test.number), block.Number)
			require.Equal(t, test.want, block.GasLimit)
		})
	}
}

func TestBeginBlockFn_ProducedBlock_AddsBlockHashAndTimeToReceiptsAndLogs(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t)
	user := beginBlockTx(1, 0)
	logs := []*types.Log{
		{Address: common.Address{0x77}, BlockNumber: 1234, TxHash: common.Hash{0x12}, TxIndex: 3, Index: 4},
		{Address: common.Address{0x78}, BlockNumber: 1234, TxHash: common.Hash{0x12}, TxIndex: 3, Index: 5},
	}
	env.proc.logs = map[common.Hash][]*types.Log{user.Hash(): logs}
	// The time of the event is clamped to the block time after the last block.
	event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(-time.Second), txs: types.Transactions{user}})

	env.runBlock(beginBlockOf(event), event)

	block := env.store.GetBlock(env.nextBlock())
	receipts := beginBlockCalls[beginBlockListenerReceipt](env.proc)
	require.Len(t, receipts, 1)
	// The block hash is set before the receipt is reported.
	require.Equal(t, block.Hash(), receipts[0].ReceiptBlockHash)
	require.Equal(t, &types.Receipt{
		Status:    types.ReceiptStatusSuccessful,
		TxHash:    user.Hash(),
		GasUsed:   21_000,
		Logs:      logs,
		BlockHash: block.Hash(),
	}, receipts[0].Receipt)
	for i, entry := range logs {
		require.Equal(t, &types.Log{
			Address:        common.Address{0x77 + byte(i)},
			BlockNumber:    1234,
			TxHash:         common.Hash{0x12},
			TxIndex:        3,
			BlockHash:      block.Hash(),
			BlockTimestamp: 1_700_000_000,
			Index:          uint(4 + i),
		}, entry)
	}
	evmBlock := env.store.EvmStore().GetCachedEvmBlock(env.nextBlock())
	require.Equal(t, block.Hash(), evmBlock.Hash)
}

func TestBeginBlockFn_ProducedBlock_ReportsCreatorOfFirstEventContainingTheOriginTransaction(t *testing.T) {
	t.Parallel()
	type setup func(env *beginBlockEnv) (events []*inter.EventPayload, tx *types.Transaction)
	singleEvent := func(creator idx.ValidatorID) setup {
		return func(env *beginBlockEnv) ([]*inter.EventPayload, *types.Transaction) {
			tx := beginBlockTx(1, 0)
			return []*inter.EventPayload{beginBlockEvent(beginBlockEventSpec{creator: creator, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{tx}})}, tx
		}
	}
	derived := func(env *beginBlockEnv) ([]*inter.EventPayload, *types.Transaction) {
		origin, derived := beginBlockTx(1, 0), beginBlockTx(3, 7)
		env.proc.derived = map[common.Hash]types.Transactions{origin.Hash(): {derived}}
		return []*inter.EventPayload{beginBlockEvent(beginBlockEventSpec{creator: 2, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{origin}})}, derived
	}
	tests := map[string]struct {
		modify func(*beginBlockChain)
		setup  setup
		want   idx.ValidatorID
	}{
		"creator of the event": {
			setup: singleEvent(2),
			want:  2,
		},
		"creator of the earliest event": {
			setup: func(env *beginBlockEnv) ([]*inter.EventPayload, *types.Transaction) {
				tx := beginBlockTx(1, 0)
				return []*inter.EventPayload{
					beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 2, time: beginBlockTime(time.Second), txs: types.Transactions{tx}}),
					beginBlockEvent(beginBlockEventSpec{creator: 2, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{tx}}),
				}, tx
			},
			want: 2,
		},
		"internal transaction": {
			setup: func(env *beginBlockEnv) ([]*inter.EventPayload, *types.Transaction) {
				tx := beginBlockTx(3, 0)
				env.proc.preTxs = types.Transactions{tx}
				return nil, tx
			},
			want: 0,
		},
		"spilled events are ignored": {
			modify: func(c *beginBlockChain) { c.es.Rules.Blocks.MaxBlockGas = 1000 },
			setup: func(env *beginBlockEnv) ([]*inter.EventPayload, *types.Transaction) {
				tx := beginBlockTx(1, 0)
				return []*inter.EventPayload{
					beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), gas: 1000, txs: types.Transactions{tx}}),
					beginBlockEvent(beginBlockEventSpec{creator: 2, lamport: 2, time: beginBlockTime(time.Second), gas: 1, txs: types.Transactions{tx}}),
				}, tx
			},
			want: 2,
		},
		"creator not being a validator": {
			setup: singleEvent(9),
			want:  0,
		},
		"creator removed from the validators by sealing": {
			setup: func(env *beginBlockEnv) ([]*inter.EventPayload, *types.Transaction) {
				env.proc.sealing = true
				env.proc.sealResult = func(bs iblockproc.BlockState, es iblockproc.EpochState) (iblockproc.BlockState, iblockproc.EpochState) {
					es.Validators, es.ValidatorProfiles = beginBlockValidators(2, 3)
					return bs, es
				}
				return singleEvent(1)(env)
			},
			want: 0,
		},
		"derived transaction with Brio": {
			setup: derived,
			want:  2,
		},
		"derived transaction contained in an earlier event": {
			setup: func(env *beginBlockEnv) ([]*inter.EventPayload, *types.Transaction) {
				// Being non-permissible, the derived transaction is not executed on its own.
				origin, derived := beginBlockTx(1, 0), beginBlockNonPermissibleTx(3, 7)
				env.proc.derived = map[common.Hash]types.Transactions{origin.Hash(): {derived}}
				return []*inter.EventPayload{
					beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{derived}}),
					beginBlockEvent(beginBlockEventSpec{creator: 2, lamport: 2, time: beginBlockTime(time.Second), txs: types.Transactions{origin}}),
				}, derived
			},
			want: 2,
		},
		"derived transaction without Brio": {
			modify: func(c *beginBlockChain) { c.setUpgrades(opera.GetAllegroUpgrades()) },
			setup:  derived,
			want:   0,
		},
		"single proposer: proposer": {
			modify: (*beginBlockChain).singleProposer,
			setup: func(env *beginBlockEnv) ([]*inter.EventPayload, *types.Transaction) {
				tx := beginBlockTx(1, 0)
				return []*inter.EventPayload{beginBlockEvent(beginBlockEventSpec{creator: 2, lamport: 1, time: beginBlockTime(time.Second), proposal: env.proposal(tx)})}, tx
			},
			want: 2,
		},
		"single proposer: earlier event with the same transaction": {
			modify: (*beginBlockChain).singleProposer,
			setup: func(env *beginBlockEnv) ([]*inter.EventPayload, *types.Transaction) {
				tx := beginBlockTx(1, 0)
				return []*inter.EventPayload{
					beginBlockEvent(beginBlockEventSpec{creator: 2, lamport: 2, time: beginBlockTime(time.Second), proposal: env.proposal(tx)}),
					beginBlockEvent(beginBlockEventSpec{creator: 3, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{tx}}),
				}, tx
			},
			want: 3,
		},
		"single proposer: earlier rejected proposal with the same transaction": {
			modify: (*beginBlockChain).singleProposer,
			setup: func(env *beginBlockEnv) ([]*inter.EventPayload, *types.Transaction) {
				tx := beginBlockTx(1, 0)
				rejected := env.proposal(tx)
				rejected.Number++
				return []*inter.EventPayload{
					beginBlockEvent(beginBlockEventSpec{creator: 2, lamport: 2, time: beginBlockTime(time.Second), proposal: env.proposal(tx)}),
					beginBlockEvent(beginBlockEventSpec{creator: 3, lamport: 1, time: beginBlockTime(time.Second), proposal: rejected}),
				}, tx
			},
			want: 3,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t, test.modify)
			events, tx := test.setup(env)
			atropos := beginBlockAtropos(beginBlockTime(time.Second))

			// The cheater makes sure the block is produced.
			env.runBlock(beginBlockOf(atropos, 3), append(events, atropos)...)

			creators := map[common.Hash]idx.ValidatorID{}
			for _, receipt := range beginBlockCalls[beginBlockListenerReceipt](env.proc) {
				creators[receipt.Receipt.TxHash] = receipt.Creator
			}
			require.Contains(t, creators, tx.Hash())
			require.Equal(t, test.want, creators[tx.Hash()])
		})
	}
}

func TestBeginBlockFn_ProducedBlock_ReportsReceiptsWithTransactionAtSamePositionAndFeesOfEvmBlock(t *testing.T) {
	t.Parallel()
	// Without Sonic, the transactions are executed in the order of the event.
	env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.setUpgrades(beginBlockPreSonicUpgrades) })
	first, second := beginBlockTx(1, 0), beginBlockTx(2, 0)
	var evmBlock *evmcore.EvmBlock
	env.proc.finalize = func(e *beginBlockEvmProcessor) (*evmcore.EvmBlock, int, types.Receipts) {
		block, skipped, receipts := e.defaultFinalize()
		slices.Reverse(block.Transactions)
		evmBlock = block
		return block, skipped, receipts
	}
	event := beginBlockEvent(beginBlockEventSpec{creator: 2, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{first, second}})

	env.runBlock(beginBlockOf(event), event)

	receipts := beginBlockCalls[beginBlockListenerReceipt](env.proc)
	require.Len(t, receipts, 2)
	require.Equal(t, second.Hash(), receipts[0].Tx.Hash())
	require.Equal(t, first.Hash(), receipts[0].Receipt.TxHash)
	require.Equal(t, first.Hash(), receipts[1].Tx.Hash())
	require.Equal(t, second.Hash(), receipts[1].Receipt.TxHash)
	for _, receipt := range receipts {
		require.Same(t, evmBlock.BaseFee, receipt.BaseFee)
		require.Same(t, evmBlock.BlobBaseFee, receipt.BlobBaseFee)
	}
}

func TestBeginBlockFn_ProducedBlock_PanicsOnInconsistentEvmResults(t *testing.T) {
	t.Parallel()
	tests := map[string]func(block *evmcore.EvmBlock, receipts types.Receipts) types.Receipts{
		"more receipts than transactions": func(block *evmcore.EvmBlock, receipts types.Receipts) types.Receipts {
			block.Transactions = nil
			return receipts
		},
		"empty transaction root": func(block *evmcore.EvmBlock, receipts types.Receipts) types.Receipts {
			block.TxHash = common.Hash{}
			return receipts
		},
		"missing base fee": func(block *evmcore.EvmBlock, receipts types.Receipts) types.Receipts {
			block.BaseFee = nil
			return receipts
		},
	}
	for name, corrupt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t)
			env.proc.preTxs = types.Transactions{beginBlockTx(3, 0)}
			env.proc.finalize = func(e *beginBlockEvmProcessor) (*evmcore.EvmBlock, int, types.Receipts) {
				block, skipped, receipts := e.defaultFinalize()
				return block, skipped, corrupt(block, receipts)
			}
			atropos := beginBlockAtropos(beginBlockTime(time.Second))

			// Without confirmed events, the block is processed synchronously.
			callbacks := env.startBlock(beginBlockOf(atropos, 2), atropos)
			require.Panics(t, func() { callbacks.EndBlock() })
		})
	}
}

func TestBeginBlockFn_ProducedBlock_StoresTransactionsAndIndexesThemIfEnabled(t *testing.T) {
	t.Parallel()
	tests := map[string]bool{
		"with transaction index":    true,
		"without transaction index": false,
	}
	for name, txIndex := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t)
			env.txIndex = txIndex
			pre, skippedPre := beginBlockTx(3, 0), beginBlockTx(3, 1)
			env.proc.preTxs = types.Transactions{pre, skippedPre}
			env.proc.skipped = map[common.Hash]bool{skippedPre.Hash(): true}
			user := beginBlockTx(1, 0)
			entry := &types.Log{Address: common.Address{0x77}, Topics: []common.Hash{{0x01}}, BlockNumber: uint64(env.nextBlock()), TxHash: user.Hash()}
			env.proc.logs = map[common.Hash][]*types.Log{user.Hash(): {entry}}
			event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{user}})

			env.runBlock(beginBlockOf(event), event)

			// Transaction bodies are stored independently of the index.
			require.Equal(t, pre.Hash(), env.store.evm.GetTx(pre.Hash()).Hash())
			require.Equal(t, user.Hash(), env.store.evm.GetTx(user.Hash()).Hash())
			require.Nil(t, env.store.evm.GetTx(skippedPre.Hash()))

			logs, err := env.store.evm.EvmLogs.FindInBlocks(context.Background(), env.nextBlock(), env.nextBlock(), [][]common.Hash{{common.BytesToHash(entry.Address[:])}}, 0)
			require.NoError(t, err)
			if !txIndex {
				require.Nil(t, env.store.evm.GetTxPosition(pre.Hash()))
				require.Nil(t, env.store.evm.GetTxPosition(user.Hash()))
				require.Nil(t, env.store.evm.GetRawReceiptsRLP(env.nextBlock()))
				require.Empty(t, logs)
				return
			}
			require.Equal(t, &evmstore.TxPosition{Block: env.nextBlock(), BlockOffset: 0}, env.store.evm.GetTxPosition(pre.Hash()))
			require.Equal(t, &evmstore.TxPosition{Block: env.nextBlock(), BlockOffset: 1}, env.store.evm.GetTxPosition(user.Hash()))
			require.Nil(t, env.store.evm.GetTxPosition(skippedPre.Hash()))
			receipts, _ := env.store.evm.GetRawReceipts(env.nextBlock())
			require.Len(t, receipts, 2)
			require.Len(t, logs, 1)
			require.Equal(t, env.store.GetBlock(env.nextBlock()).Hash(), logs[0].BlockHash)
		})
	}
}

func TestBeginBlockFn_ProducedBlock_StoresDataBeforeReferencingIt(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t)
	env.txIndex = true
	env.proc.sealing = true
	pre, user := beginBlockTx(3, 0), beginBlockTx(1, 0)
	env.proc.preTxs = types.Transactions{pre}
	logAddress := common.Address{0x77}
	env.proc.logs = map[common.Hash][]*types.Log{user.Hash(): {
		{Address: logAddress, BlockNumber: uint64(env.nextBlock()), TxHash: user.Hash(), Index: 0},
		{Address: logAddress, BlockNumber: uint64(env.nextBlock()), TxHash: user.Hash(), Index: 1},
	}}
	// missing lists the data of the produced block that cannot be read from the store.
	missing := func(store *Store) []string {
		res := []string{}
		next := env.nextBlock()
		if block := store.GetBlock(next); block == nil || store.GetBlockIndex(hash.Event(block.Hash())) == nil {
			res = append(res, "block")
		}
		for _, tx := range []*types.Transaction{pre, user} {
			if store.evm.GetTx(tx.Hash()) == nil || store.evm.GetTxPosition(tx.Hash()) == nil {
				res = append(res, "transaction "+tx.Hash().Hex())
			}
		}
		if store.evm.GetRawReceiptsRLP(next) == nil {
			res = append(res, "receipts")
		}
		if logs, err := store.evm.EvmLogs.FindInBlocks(context.Background(), next, next, [][]common.Hash{{common.BytesToHash(logAddress[:])}}, 0); err != nil || len(logs) != 2 {
			res = append(res, "logs")
		}
		if bs, _ := store.GetHistoryBlockEpochState(beginBlockEpoch + 1); bs == nil || beginBlockEpochBlocks(store)[next+1] != beginBlockEpoch+1 {
			res = append(res, "epoch history")
		}
		return res
	}
	// Around every write to the store, no position may be readable without its
	// body, no block hash may be indexed without its block, and the latest block
	// index may only refer to a completely stored block.
	var store *Store
	var unreadable []string
	env.store = newBeginBlockStore(t, func() {
		if store == nil {
			return
		}
		for _, tx := range []*types.Transaction{pre, user} {
			if store.evm.GetTxPosition(tx.Hash()) != nil && store.evm.GetTx(tx.Hash()) == nil {
				unreadable = append(unreadable, "body of "+tx.Hash().Hex())
			}
		}
		it := store.table.BlockHashes.NewIterator(nil, nil)
		for it.Next() {
			if store.GetBlock(idx.BytesToBlock(it.Value())) == nil {
				unreadable = append(unreadable, "indexed block")
			}
		}
		it.Release()
		if store.GetLatestBlockIndex() >= env.nextBlock() {
			unreadable = append(unreadable, missing(store)...)
		}
	})
	store = env.store
	env.chain.writeTo(env.store)
	event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{user}})

	env.runBlock(beginBlockOf(event), event)

	require.Equal(t, env.nextBlock(), env.store.GetLatestBlockIndex())
	require.Empty(t, missing(env.store))
	require.Empty(t, unreadable)
}

func TestBeginBlockFn_ProducedBlock_DoesNotStoreReceiptsOfBlocksWithoutTransactions(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t)
	env.txIndex = true
	atropos := beginBlockAtropos(beginBlockTime(time.Second))

	env.runBlock(beginBlockOf(atropos, 2), atropos)

	require.NotNil(t, env.store.GetBlock(env.nextBlock()))
	require.Nil(t, env.store.evm.GetRawReceiptsRLP(env.nextBlock()))
}

func TestBeginBlockFn_ProducedBlock_IndexesPositionsOfTransactionsOfEvmBlockButStoresBodiesOfIncludedTransactions(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t)
	env.txIndex = true
	user, unknown := beginBlockTx(1, 0), beginBlockTx(2, 0)
	env.proc.finalize = func(e *beginBlockEvmProcessor) (*evmcore.EvmBlock, int, types.Receipts) {
		block, skipped, receipts := e.defaultFinalize()
		block.Transactions = types.Transactions{user, unknown, user}
		return block, skipped, receipts
	}
	event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{user}})

	env.runBlock(beginBlockOf(event), event)

	// For repeated transactions, the last position is indexed.
	require.Equal(t, &evmstore.TxPosition{Block: env.nextBlock(), BlockOffset: 2}, env.store.evm.GetTxPosition(user.Hash()))
	require.Equal(t, &evmstore.TxPosition{Block: env.nextBlock(), BlockOffset: 1}, env.store.evm.GetTxPosition(unknown.Hash()))
	require.NotNil(t, env.store.evm.GetTx(user.Hash()))
	require.Nil(t, env.store.evm.GetTx(unknown.Hash()))
}

func TestBeginBlockFn_ProducedBlock_StoresFinalBlockStateAndEpochHistoryWhenSealing(t *testing.T) {
	t.Parallel()
	tests := map[string]bool{
		"without sealing": false,
		"with sealing":    true,
	}
	for name, sealing := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.bs.EpochCheaters = lachesis.Cheaters{1} })
			env.proc.sealing = sealing
			// The sealed epoch is not the successor of the epoch of the Atropos.
			const sealedEpoch = beginBlockEpoch + 2
			env.proc.sealResult = func(bs iblockproc.BlockState, es iblockproc.EpochState) (iblockproc.BlockState, iblockproc.EpochState) {
				es.Epoch = sealedEpoch
				return bs, es
			}
			env.proc.evmRoot = hash.Hash{0x55}
			event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(1, 0)}})
			atropos := beginBlockAtropos(beginBlockTime(time.Second))

			env.runBlock(beginBlockOf(atropos, 2), event, atropos)

			want := beginBlockCalls[beginBlockEventsStart](env.proc)[0].BS
			want.EpochGas = beginBlockTagListener2
			want.FinalizedStateRoot = hash.Hash{0x55}
			want.LastBlock = iblockproc.BlockCtx{Idx: env.nextBlock(), Time: beginBlockTime(time.Second), Atropos: atropos.ID()}
			want.CheatersWritten = 2
			wantEs := env.chain.es.Copy()
			if sealing {
				wantEs = beginBlockCalls[beginBlockListenerUpdate](env.proc)[0].ES
			}
			bs, es := env.store.GetBlockEpochState()
			require.Equal(t, lachesis.Cheaters{1, 2}, bs.EpochCheaters)
			require.Equal(t, want.Hash(), bs.Hash())
			require.Equal(t, wantEs.Hash(), es.Hash())

			historyBs, historyEs := env.store.GetHistoryBlockEpochState(sealedEpoch)
			if !sealing {
				require.Nil(t, historyBs)
				require.Nil(t, historyEs)
				require.Empty(t, beginBlockEpochBlocks(env.store))
				return
			}
			require.Equal(t, want.Hash(), historyBs.Hash())
			require.Equal(t, wantEs.Hash(), historyEs.Hash())
			require.Equal(t, map[idx.Block]idx.Epoch{env.nextBlock() + 1: sealedEpoch}, beginBlockEpochBlocks(env.store))
		})
	}
}

func TestBeginBlockFn_ProducedBlock_CachesEvmBlock(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t)
	var evmBlock *evmcore.EvmBlock
	env.proc.finalize = func(e *beginBlockEvmProcessor) (*evmcore.EvmBlock, int, types.Receipts) {
		block, skipped, receipts := e.defaultFinalize()
		evmBlock = block
		return block, skipped, receipts
	}

	env.runProducedBlock()

	require.Same(t, evmBlock, env.store.EvmStore().GetCachedEvmBlock(env.nextBlock()))
}

func TestBeginBlockFn_ProducedBlock_NotifiesFeedAboutBlockAndLogsOfAllReceipts(t *testing.T) {
	t.Parallel()
	tests := map[string]bool{
		"with logs":    true,
		"without logs": false,
	}
	for name, withLogs := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.setUpgrades(beginBlockPreSonicUpgrades) })
			updates := env.withFeed()
			pre, user1, user2 := beginBlockTx(3, 0), beginBlockTx(1, 0), beginBlockTx(2, 0)
			env.proc.preTxs = types.Transactions{pre}
			if withLogs {
				env.proc.logs = map[common.Hash][]*types.Log{
					pre.Hash():   {{Index: 1}},
					user2.Hash(): {{Index: 2}, {Index: 3}},
				}
			}
			var evmBlock *evmcore.EvmBlock
			env.proc.finalize = func(e *beginBlockEvmProcessor) (*evmcore.EvmBlock, int, types.Receipts) {
				block, skipped, receipts := e.defaultFinalize()
				evmBlock = block
				return block, skipped, receipts
			}
			event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{user1, user2}})

			env.runBlock(beginBlockOf(event), event)

			require.Len(t, updates, 1)
			update := <-updates
			require.Same(t, evmBlock, update.block)
			if !withLogs {
				require.Nil(t, update.logs)
				return
			}
			want := []*types.Log{env.proc.logs[pre.Hash()][0], env.proc.logs[user2.Hash()][0], env.proc.logs[user2.Hash()][1]}
			require.Equal(t, want, update.logs)
		})
	}
}

func TestBeginBlockFn_ProducedBlock_NotifiesFeedAfterStoringTheBlock(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		env := newBeginBlockEnv(t)
		updates := make(chan feedUpdate)
		env.feed = &ServiceFeed{incomingUpdates: updates}
		event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(1, 0)}})
		env.startBlock(beginBlockOf(event), event).EndBlock()

		// The block processing is blocked on notifying the feed.
		synctest.Wait()
		block := env.store.GetBlock(env.nextBlock())
		lastBlock := env.store.GetBlockState().LastBlock.Idx
		evmBlock := env.store.EvmStore().GetCachedEvmBlock(env.nextBlock())
		<-updates
		env.waitIdle()

		require.NotNil(t, block)
		require.Equal(t, env.nextBlock(), lastBlock)
		require.NotNil(t, evmBlock)
	})
}

func TestBeginBlockFn_ProducedBlock_UpdatesBlockMetrics(t *testing.T) {
	// Not parallel, as the metrics are global.
	tests := map[string]struct {
		produced  bool
		wantIndex idx.Block
		modify    func(*beginBlockChain)
	}{
		"produced block": {
			produced:  true,
			wantIndex: beginBlockLastBlock + 1,
		},
		"produced block of a proposal for the successor of the parent block number": {
			produced:  true,
			wantIndex: 21,
			modify: func(c *beginBlockChain) {
				c.singleProposer()
				c.parentNumber = 20
			},
		},
		"skipped block": {
			produced: false,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// The fake clock makes the block age exact.
			synctest.Test(t, func(t *testing.T) {
				env := newBeginBlockEnv(t, test.modify)
				executed, skipped := beginBlockTx(1, 0), beginBlockTx(2, 0)
				env.proc.skipped = map[common.Hash]bool{skipped.Hash(): true}
				txs := types.Transactions{executed, beginBlockTx(1, 1), skipped}
				if !test.produced {
					txs = nil
				}
				// The time of the event is clamped to the block time after the last block.
				spec := beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(-time.Second), txs: txs}
				if env.chain.es.Rules.Upgrades.SingleProposerBlockFormation {
					spec.txs, spec.proposal = nil, env.proposal(txs...)
				}
				event := beginBlockEvent(spec)

				for _, gauge := range []*metrics.Gauge{headBlockGauge, headHeaderGauge, headFastBlockGauge, blockAgeGauge} {
					gauge.Update(-1)
				}
				processedBefore := processedTxsMeter.Snapshot().Count()
				skippedBefore := skippedTxsMeter.Snapshot().Count()
				env.runBlock(beginBlockOf(event), event)

				wantHead, wantProcessed, wantSkipped, wantAge := int64(-1), int64(0), int64(0), int64(-1)
				if test.produced {
					wantHead, wantProcessed, wantSkipped = int64(test.wantIndex), 2, 1
					wantAge = int64(time.Since((beginBlockLastBlockTime + 1).Time()))
				}
				// The execution timers are not observable, as metrics are disabled in tests.
				for _, gauge := range []*metrics.Gauge{headBlockGauge, headHeaderGauge, headFastBlockGauge} {
					require.Equal(t, wantHead, gauge.Snapshot().Value())
				}
				require.Equal(t, wantProcessed, processedTxsMeter.Snapshot().Count()-processedBefore)
				require.Equal(t, wantSkipped, skippedTxsMeter.Snapshot().Count()-skippedBefore)
				require.Equal(t, wantAge, blockAgeGauge.Snapshot().Value())
			})
		})
	}
}

// ---------------------------------------------------------------------------
// EndBlock: scheduling the block processing
// ---------------------------------------------------------------------------

func TestBeginBlockFn_ProcessesBlocksWithConfirmedEventsAsynchronously(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		env := newBeginBlockEnv(t)
		env.proc.sealing = true
		release := blockOn[beginBlockEvmFinalize](env)
		event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(1, 0)}})
		callbacks := env.startBlock(beginBlockOf(event), event)

		// EndBlock returns the new validators while the block is processed.
		require.Same(t, env.chain.es.Validators, callbacks.EndBlock())
		var processed atomic.Bool
		go func() {
			env.wg.Wait()
			processed.Store(true)
		}()
		synctest.Wait()
		require.False(t, processed.Load())
		require.Equal(t, uint32(1), atomic.LoadUint32(&env.busy))
		require.Nil(t, env.store.GetBlock(env.nextBlock()))
		require.Equal(t, beginBlockSealedEsTag, env.store.GetEpochState().EpochStateRoot)

		release()
		synctest.Wait()
		require.True(t, processed.Load())
		require.Zero(t, atomic.LoadUint32(&env.busy))
		require.NotNil(t, env.store.GetBlock(env.nextBlock()))
	})
}

func TestBeginBlockFn_ProcessesBlocksAsynchronouslyIfAnyEventCarriesTransactionsOrProposals(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		singleProposer bool
		spec           *beginBlockEventSpec // event confirmed besides the Atropos, if any
		proposal       bool                 // whether the event proposes an empty block
		async          bool
	}{
		"block of an empty proposal": {
			singleProposer: true,
			spec:           &beginBlockEventSpec{},
			proposal:       true,
			async:          true,
		},
		"block of spilled events only": {
			spec:  &beginBlockEventSpec{gas: beginBlockMaxBlockGas + 1, txs: types.Transactions{beginBlockTx(1, 0)}},
			async: true,
		},
		"block of non-permissible transactions only": {
			spec:  &beginBlockEventSpec{txs: types.Transactions{beginBlockNonPermissibleTx(1, 0)}},
			async: true,
		},
		"block of cheaters only": {
			async: false,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := newBeginBlockEnv(t, func(c *beginBlockChain) { c.es.Rules.Upgrades.SingleProposerBlockFormation = test.singleProposer })
			events := []*inter.EventPayload{}
			if test.spec != nil {
				spec := *test.spec
				spec.creator, spec.lamport, spec.time = 1, 1, beginBlockTime(time.Second)
				if test.proposal {
					spec.proposal = env.proposal()
				}
				events = append(events, beginBlockEvent(spec))
			}
			atropos := beginBlockAtropos(beginBlockTime(time.Second))

			env.runBlock(beginBlockOf(atropos, 2), append(events, atropos)...)

			require.Equal(t, test.async, env.processedAsynchronously())
		})
	}
}

func TestBeginBlockFn_ProcessesBlocksWithoutConfirmedEventsSynchronouslyWithoutTouchingBusyFlag(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t)
	env.busy = 7
	atropos := beginBlockAtropos(beginBlockTime(time.Second))

	callbacks := env.startBlock(beginBlockOf(atropos, 2), atropos)
	callbacks.EndBlock()

	require.NotNil(t, env.store.GetBlock(env.nextBlock()))
	require.Equal(t, uint32(7), env.busy)
	for _, execution := range beginBlockCalls[beginBlockExecute](env.proc) {
		require.Equal(t, uint32(7), execution.Busy)
	}
}

func TestBeginBlockFn_PanicsIfBlockProcessingCannotBeScheduled(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t)
	quit := make(chan struct{})
	close(quit)
	env.workers = workers.New(&sync.WaitGroup{}, quit, 0)
	event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(1, 0)}})

	callbacks := env.startBlock(beginBlockOf(event), event)
	require.PanicsWithError(t, "terminated", func() { callbacks.EndBlock() })

	// The busy flag remains set and the block is not processed.
	require.Equal(t, uint32(1), env.busy)
	require.NotContains(t, env.proc.trace(), "PostTx.PopInternalTxs")
}

func TestBeginBlockFn_NextBlockStartsFromStateOfPendingBlock(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		env := newBeginBlockEnv(t)
		release := blockOn[beginBlockEvmFinalize](env)
		event := beginBlockEvent(beginBlockEventSpec{creator: 1, lamport: 1, time: beginBlockTime(time.Second), txs: types.Transactions{beginBlockTx(1, 0)}})
		callbacks := env.startBlock(beginBlockOf(event), event)
		callbacks.EndBlock()

		beginBlock := env.fn()
		go beginBlock(&lachesis.Block{})
		synctest.Wait()
		require.Len(t, beginBlockCalls[beginBlockEventsStart](env.proc), 1)

		release()
		synctest.Wait()
		starts := beginBlockCalls[beginBlockEventsStart](env.proc)
		require.Len(t, starts, 2)
		require.Equal(t, iblockproc.BlockCtx{Idx: env.nextBlock(), Time: beginBlockTime(time.Second), Atropos: event.ID()}, starts[1].BS.LastBlock)
	})
}

func TestBeginBlockFn_ConsecutiveBlocksBuildOnEachOther(t *testing.T) {
	t.Parallel()
	env := newBeginBlockEnv(t)
	env.runProducedBlock()
	first := env.store.GetBlock(env.nextBlock())
	env.proc.reset()

	user := beginBlockTx(2, 0)
	event := beginBlockEvent(beginBlockEventSpec{creator: 2, lamport: 2, time: beginBlockTime(2 * time.Second), txs: types.Transactions{user}})
	env.runBlock(beginBlockOf(event), event)

	second := env.store.GetBlock(env.nextBlock() + 1)
	require.Equal(t, first.Hash(), second.ParentHash)
	require.Equal(t, env.nextBlock()+1, env.evmStart().Number)
	require.Equal(t, uint64(time.Second), second.Duration)
	require.Equal(t, env.nextBlock(), beginBlockCalls[beginBlockEventsStart](env.proc)[0].BS.LastBlock.Idx)
	// The confirmed events and the Atropos of a block are not carried over.
	require.Equal(t, beginBlockHashes(types.Transactions{user}), beginBlockHashes(env.userTxs()))
	require.Equal(t, beginBlockPrevRandao(event.ID()), env.evmStart().Randao)
	env.proc.reset()

	// The cheater makes sure the block is only skipped for its degenerate Atropos.
	env.runBlock(beginBlockOf(beginBlockAtropos(beginBlockTime(3*time.Second)), 3))
	require.True(t, env.blockSkipped())
}
