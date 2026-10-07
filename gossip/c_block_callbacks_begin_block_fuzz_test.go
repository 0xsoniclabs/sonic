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
	"sync"
	"testing"
	"time"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/Fantom-foundation/lachesis-base/kvdb/flushable"
	"github.com/Fantom-foundation/lachesis-base/lachesis"
	"github.com/Fantom-foundation/lachesis-base/utils/workers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/verwatcher"
	"github.com/0xsoniclabs/sonic/gossip/emitter"
	"github.com/0xsoniclabs/sonic/gossip/randao"
	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/drivertype"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
)

// FuzzBeginBlockFn runs consensusCallbackBeginBlockFn on generated blocks and
// compares its observable effects with those of refConsensusCallbackBeginBlockFn,
// a copy of the implementation at a fixed commit. Panics are failures. The
// observed global meters are not touched by other tests, as fuzz targets run
// after all tests, one input at a time.
// Input minimization is too slow for this target and should be disabled:
//
//	go test ./gossip -run '^$' -fuzz '^FuzzBeginBlockFn$' -fuzzminimizetime 0 -fuzztime 10m
func FuzzBeginBlockFn(f *testing.F) {
	// Tests run before may leave the global logger bound to a completed test.
	log.SetDefault(log.NewLogger(log.DiscardHandler()))
	for _, seed := range beginBlockFuzzSeeds() {
		f.Add(seed.encode())
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		beginBlockRunFuzzScenario(t, beginBlockDecodeFuzzScenario(data))
	})
}

// ---------------------------------------------------------------------------
// Scenario
// ---------------------------------------------------------------------------

const beginBlockFuzzMaxEvents = 6

// beginBlockFuzzScenario describes a block. All values are valid, their meaning is
// defined by the accessor methods, the field comments and beginBlockObserveFuzzScenario.
type beginBlockFuzzScenario struct {
	flags            uint8
	upgrades         uint8
	maxBlockGas      uint8 // in units of 64
	skipPeriod       uint8 // in units of 100ms
	allocPerSec      uint8 // in units of 10_000 gas per second
	parentTime       int8  // time of the stored parent block relative to the last block, in units of 100ms
	parentNumber     uint8 // number of the stored parent block minus its index, modulo 4
	storedCheaters   []uint8
	blockCheaters    []uint8
	events           []beginBlockFuzzEvent
	order            uint8  // rotation of the events in the order they are applied, reversed if the high bit is set
	duplicate        uint8  // 1-based index of an event applied once more after the others, none if 0 or too large
	atropos          uint8  // index of the Atropos among the events, a separate Atropos, or an Atropos that is not applied
	atroposTime      int8   // time of an Atropos that is none of the events relative to the last block, in units of 100ms
	skippedUserTxs   uint16 // bit mask of beginBlockFuzzPool transactions skipped by the EVM
	failedUserTxs    uint16 // bit mask of beginBlockFuzzPool transactions with a failed receipt
	loggingUserTxs   uint16 // bit mask of beginBlockFuzzPool transactions emitting two logs
	derivingUserTxs  uint16 // bit mask of beginBlockFuzzPool transactions the EVM derives a transaction from
	preTxs           uint8
	postTxs          uint8
	sealedValidators uint8 // bit mask of the validators 1-3
}

type beginBlockFuzzEvent struct {
	creator      uint8
	lamport      uint8
	oldEpoch     bool
	timeOffset   int8 // relative to the last block, in units of 100ms
	gas          uint8
	kind         uint8
	txs          uint16 // bit mask of beginBlockFuzzPool transactions
	numberOffset int8
	wrongParent  bool
	turn         uint8
	reveal       uint8
}

// bootstrapping requires two bits, so that only a quarter of the scenarios ignore the block.
func (s beginBlockFuzzScenario) bootstrapping() bool  { return s.flags&3 == 3 }
func (s beginBlockFuzzScenario) singleProposer() bool { return s.flags&4 != 0 }
func (s beginBlockFuzzScenario) sealing() bool        { return s.flags&8 != 0 }
func (s beginBlockFuzzScenario) txIndex() bool        { return s.flags&16 != 0 }
func (s beginBlockFuzzScenario) feed() bool           { return s.flags&32 != 0 }
func (s beginBlockFuzzScenario) priorities() bool     { return s.flags&64 != 0 }
func (s beginBlockFuzzScenario) upgradesChange() bool { return s.flags&128 != 0 }

func (s beginBlockFuzzScenario) upgradeSet() opera.Upgrades {
	bundles := opera.GetBrioUpgrades()
	bundles.TransactionBundles = true
	return []opera.Upgrades{
		beginBlockPreSonicUpgrades,
		opera.GetSonicUpgrades(),
		opera.GetAllegroUpgrades(),
		opera.GetBrioUpgrades(),
		bundles,
	}[s.upgrades%5]
}

func (s beginBlockFuzzScenario) rules(rules opera.Rules) opera.Rules {
	rules.Upgrades = s.upgradeSet()
	rules.Upgrades.SingleProposerBlockFormation = s.singleProposer()
	rules.Upgrades.TransactionPriorities = s.priorities()
	rules.Blocks.MaxBlockGas = uint64(s.maxBlockGas) * 64
	rules.Blocks.MaxEmptyBlockSkipPeriod = inter.Timestamp(s.skipPeriod) * inter.Timestamp(100*time.Millisecond)
	rules.Economy.ShortGasPower.AllocPerSec = uint64(s.allocPerSec) * 10_000
	return rules
}

func beginBlockFuzzCheaters(values []uint8) lachesis.Cheaters {
	res := lachesis.Cheaters{}
	for _, value := range values {
		res = append(res, idx.ValidatorID(value%4+1))
	}
	return res
}

func beginBlockFuzzTime(offset int8) inter.Timestamp {
	return beginBlockTime(time.Duration(offset) * 100 * time.Millisecond)
}

// beginBlockFuzzPool returns the user transactions available to scenarios. Besides
// transfers, it covers the kinds of transactions distinguished by isPermissible.
var beginBlockFuzzPool = sync.OnceValue(func() types.Transactions {
	to := common.Address{0x42}
	chainID := new(big.Int).SetUint64(opera.FakeNetworkID)
	return types.Transactions{
		beginBlockTx(1, 0), beginBlockTx(1, 1), beginBlockTx(2, 0), beginBlockTx(2, 1), beginBlockTx(3, 0), beginBlockTx(3, 1),
		// transaction without signature
		types.NewTx(&types.LegacyTx{Nonce: 7, To: &common.Address{0x42}, Gas: 21_000, GasPrice: big.NewInt(1)}),
		beginBlockNonPermissibleTx(2, 5),
		// blob transaction without blob hashes
		types.MustSignNewTx(makefakegenesis.FakeKey(2), beginBlockSigner, &types.BlobTx{
			ChainID: uint256.MustFromBig(chainID), Nonce: 3, To: to, Gas: 21_000,
		}),
		// blob transaction with a blob hash
		types.MustSignNewTx(makefakegenesis.FakeKey(1), beginBlockSigner, &types.BlobTx{
			ChainID: uint256.MustFromBig(chainID), Nonce: 2, To: to, Gas: 21_000, BlobHashes: []common.Hash{{0x01}},
		}),
		// set code transaction with an authorization
		types.MustSignNewTx(makefakegenesis.FakeKey(2), beginBlockSigner, &types.SetCodeTx{
			ChainID: uint256.MustFromBig(chainID), Nonce: 6, To: to, Gas: 100_000, AuthList: []types.SetCodeAuthorization{{}},
		}),
		// transaction failing the static checks, as its tip exceeds its fee cap
		types.MustSignNewTx(makefakegenesis.FakeKey(3), beginBlockSigner, &types.DynamicFeeTx{
			ChainID: chainID, Nonce: 2, To: &to, Gas: 21_000, GasTipCap: big.NewInt(2), GasFeeCap: big.NewInt(1),
		}),
		// bundle-only transaction
		types.MustSignNewTx(makefakegenesis.FakeKey(1), beginBlockSigner, &types.AccessListTx{
			ChainID: chainID, Nonce: 3, To: &to, Gas: 21_000, GasPrice: big.NewInt(1), AccessList: types.AccessList{{Address: bundle.BundleOnly}},
		}),
		// envelope of a bundle of a permissible transaction
		bundle.NewBuilder().WithSigner(beginBlockSigner).SetEnvelopeSenderKey(makefakegenesis.FakeKey(3)).SetEnvelopeNonce(3).
			AllOf(bundle.Step(makefakegenesis.FakeKey(1), &types.AccessListTx{Nonce: 4, To: &to, Gas: 21_000, GasPrice: big.NewInt(1)})).
			Build(),
		// envelope of a bundle of a non-permissible transaction
		bundle.NewBuilder().WithSigner(beginBlockSigner).SetEnvelopeSenderKey(makefakegenesis.FakeKey(3)).SetEnvelopeNonce(4).
			AllOf(bundle.Step(makefakegenesis.FakeKey(2), &types.SetCodeTx{Nonce: 7, To: to, Gas: 21_000})).
			Build(),
		// envelope that cannot be opened
		types.MustSignNewTx(makefakegenesis.FakeKey(3), beginBlockSigner, &types.LegacyTx{
			Nonce: 5, To: &bundle.BundleProcessor, Gas: 21_000, GasPrice: big.NewInt(1), Data: []byte{1, 2, 3},
		}),
	}
})

func beginBlockFuzzPoolTxs(mask uint16) types.Transactions {
	res := types.Transactions{}
	for i, tx := range beginBlockFuzzPool() {
		if mask&(1<<i) != 0 {
			res = append(res, tx)
		}
	}
	return res
}

// beginBlockFuzzLogAddress is the address of all logs emitted by beginBlockFuzzPool transactions.
var beginBlockFuzzLogAddress = common.Address{0x77}

// beginBlockFuzzInternalTxs returns 0-3 internal transactions and those of them
// skipped by the EVM, as encoded in the given byte.
func beginBlockFuzzInternalTxs(value uint8, nonceBase uint64) (types.Transactions, map[common.Hash]bool) {
	txs := types.Transactions{}
	skipped := map[common.Hash]bool{}
	for i := range int(value % 4) {
		tx := beginBlockTx(4, nonceBase+uint64(i))
		txs = append(txs, tx)
		if value&(4<<i) != 0 {
			skipped[tx.Hash()] = true
		}
	}
	return txs, skipped
}

func (e beginBlockFuzzEvent) build(env *beginBlockEnv, seq int) *inter.EventPayload {
	creator := idx.ValidatorID(e.creator%4 + 1)
	spec := beginBlockEventSpec{
		creator: creator,
		lamport: idx.Lamport(e.lamport%4 + 1),
		seq:     idx.Event(seq),
		time:    beginBlockFuzzTime(e.timeOffset),
		gas:     uint64(e.gas),
	}
	if e.oldEpoch {
		spec.epoch = beginBlockEpoch - 1
	}
	switch e.kind % 3 {
	case 0:
		// A turn other than 0 makes a version 3 event with a proposal sync state only.
		spec.turn = inter.Turn(e.turn % 4)
	case 1:
		spec.txs = beginBlockFuzzPoolTxs(e.txs)
	case 2:
		proposal := env.proposal(beginBlockFuzzPoolTxs(e.txs)...)
		proposal.Number = idx.Block(int(proposal.Number) + int(e.numberOffset%2))
		if e.wrongParent {
			proposal.ParentHash = common.Hash{0x12}
		}
		switch e.reveal % 3 {
		case 0:
			proposal.RandaoReveal = randao.RandaoReveal{1, 2, 3}
		case 1:
			proposal.RandaoReveal, _ = beginBlockRandaoReveal(env.t, creator, beginBlockParentRandao)
		case 2:
			proposal.RandaoReveal, _ = beginBlockRandaoReveal(env.t, creator%4+1, beginBlockParentRandao)
		}
		spec.proposal = proposal
		spec.turn = inter.Turn(e.turn % 4)
	}
	return beginBlockEvent(spec)
}

// ---------------------------------------------------------------------------
// Encoding
// ---------------------------------------------------------------------------

// beginBlockFuzzCodec visits the fields of a scenario for decoding or encoding it.
type beginBlockFuzzCodec interface {
	uint8(*uint8)
	uint16(*uint16)
	int8(*int8)
	bool(*bool)
}

func (s *beginBlockFuzzScenario) visit(c beginBlockFuzzCodec) {
	c.uint8(&s.flags)
	c.uint8(&s.upgrades)
	c.uint8(&s.maxBlockGas)
	c.uint8(&s.skipPeriod)
	c.uint8(&s.allocPerSec)
	c.int8(&s.parentTime)
	c.uint8(&s.parentNumber)
	beginBlockVisitList(c, &s.storedCheaters, 3)
	beginBlockVisitList(c, &s.blockCheaters, 3)
	count := uint8(len(s.events))
	c.uint8(&count)
	count %= beginBlockFuzzMaxEvents + 1
	if len(s.events) != int(count) {
		s.events = make([]beginBlockFuzzEvent, count)
	}
	for i := range s.events {
		s.events[i].visit(c)
	}
	c.uint8(&s.order)
	c.uint8(&s.duplicate)
	c.uint8(&s.atropos)
	c.int8(&s.atroposTime)
	c.uint16(&s.skippedUserTxs)
	c.uint16(&s.failedUserTxs)
	c.uint16(&s.loggingUserTxs)
	c.uint16(&s.derivingUserTxs)
	c.uint8(&s.preTxs)
	c.uint8(&s.postTxs)
	c.uint8(&s.sealedValidators)
}

func (e *beginBlockFuzzEvent) visit(c beginBlockFuzzCodec) {
	c.uint8(&e.creator)
	c.uint8(&e.lamport)
	c.bool(&e.oldEpoch)
	c.int8(&e.timeOffset)
	c.uint8(&e.gas)
	c.uint8(&e.kind)
	c.uint16(&e.txs)
	c.int8(&e.numberOffset)
	c.bool(&e.wrongParent)
	c.uint8(&e.turn)
	c.uint8(&e.reveal)
}

func beginBlockVisitList(c beginBlockFuzzCodec, list *[]uint8, maxLen uint8) {
	count := uint8(len(*list))
	c.uint8(&count)
	count %= maxLen + 1
	if len(*list) != int(count) {
		*list = make([]uint8, count)
	}
	for i := range *list {
		c.uint8(&(*list)[i])
	}
}

// beginBlockFuzzDecoder reads fields from the input, using zero once it is exhausted.
type beginBlockFuzzDecoder struct {
	data []byte
}

func (d *beginBlockFuzzDecoder) uint8(v *uint8) {
	*v = 0
	if len(d.data) > 0 {
		*v, d.data = d.data[0], d.data[1:]
	}
}

func (d *beginBlockFuzzDecoder) uint16(v *uint16) {
	var low, high uint8
	d.uint8(&low)
	d.uint8(&high)
	*v = uint16(high)<<8 | uint16(low)
}

func (d *beginBlockFuzzDecoder) int8(v *int8) {
	var b uint8
	d.uint8(&b)
	*v = int8(b)
}

func (d *beginBlockFuzzDecoder) bool(v *bool) {
	var b uint8
	d.uint8(&b)
	*v = b&1 == 1
}

type beginBlockFuzzEncoder struct {
	data []byte
}

func (e *beginBlockFuzzEncoder) uint8(v *uint8)   { e.data = append(e.data, *v) }
func (e *beginBlockFuzzEncoder) uint16(v *uint16) { e.data = append(e.data, uint8(*v), uint8(*v>>8)) }
func (e *beginBlockFuzzEncoder) int8(v *int8)     { e.data = append(e.data, uint8(*v)) }
func (e *beginBlockFuzzEncoder) bool(v *bool) {
	b := uint8(0)
	if *v {
		b = 1
	}
	e.data = append(e.data, b)
}

func beginBlockDecodeFuzzScenario(data []byte) beginBlockFuzzScenario {
	s := beginBlockFuzzScenario{}
	s.visit(&beginBlockFuzzDecoder{data: data})
	return s
}

func (s beginBlockFuzzScenario) encode() []byte {
	encoder := &beginBlockFuzzEncoder{}
	s.visit(encoder)
	return encoder.data
}

func TestBeginBlockFuzzScenario_DecodingTheEncodingYieldsTheScenario(t *testing.T) {
	t.Parallel()
	for _, seed := range beginBlockFuzzSeeds() {
		require.Equal(t, seed, beginBlockDecodeFuzzScenario(seed.encode()))
	}
}

// ---------------------------------------------------------------------------
// Execution
// ---------------------------------------------------------------------------

type beginBlockFnFactory func(
	*workers.Workers, *sync.WaitGroup, *uint32, *Store, BlockProc, bool,
	*ServiceFeed, *[]*emitter.Emitter, *verwatcher.VersionWatcher, *bool,
) lachesis.BeginBlockFn

// beginBlockRunFuzzScenario runs the scenario with the reference implementation and
// with consensusCallbackBeginBlockFn and requires the same observations.
func beginBlockRunFuzzScenario(t *testing.T, s beginBlockFuzzScenario) {
	want := beginBlockObserveFuzzScenario(t, s, refConsensusCallbackBeginBlockFn)
	got := beginBlockObserveFuzzScenario(t, s, consensusCallbackBeginBlockFn)
	require.Equal(t, want, got)
}

func beginBlockObserveFuzzScenario(t *testing.T, s beginBlockFuzzScenario, newBeginBlockFn beginBlockFnFactory) beginBlockFuzzObservation {
	env := newBeginBlockEnv(t, func(c *beginBlockChain) {
		c.es.Rules = s.rules(c.es.Rules)
		c.bs.EpochCheaters = beginBlockFuzzCheaters(s.storedCheaters)
		c.parentTime = beginBlockFuzzTime(s.parentTime)
		if s.parentNumber%4 != 0 {
			c.parentNumber = uint64(beginBlockLastBlock) + uint64(s.parentNumber%4)
		}
	})
	// The Atropos may be an event of the previous epoch, in which validator i
	// uses key i%3+1.
	previous := env.chain.es.Copy()
	previous.Epoch = beginBlockEpoch - 1
	for id := idx.ValidatorID(1); id <= 3; id++ {
		previous.ValidatorProfiles[id] = drivertype.Validator{Weight: big.NewInt(int64(id)), PubKey: beginBlockPubKey(id%3 + 1)}
	}
	env.store.SetHistoryBlockEpochState(beginBlockEpoch-1, env.chain.bs, previous)
	env.store.AddProcessedBundles(uint64(beginBlockLastBlock), map[common.Hash]bundle.PositionInBlock{{0x01}: {Offset: 0, Count: 1}})
	// The chain config of the block differs from that of its successor.
	env.store.AddUpgradeHeight(opera.UpgradeHeight{Upgrades: opera.GetAllegroUpgrades(), Height: env.nextBlock() + 1})
	if s.priorities() {
		sender, err := types.Sender(beginBlockSigner, beginBlockTx(1, 0))
		require.NoError(t, err)
		env.setStateRoot(beginBlockCommitState(t, env.store, env.chain.bs.FinalizedStateRoot, func(statedb state.StateDB) {
			statedb.SetCode(registry.GetAddress(), registry.GetCode(), tracing.CodeChangeUnspecified)
			statedb.SetState(registry.GetAddress(), senderPriorityStorageSlot(sender), packSenderPriority(1, 0, 1))
		}))
	}
	env.proc.evmRoot = hash.Hash{0x55}

	env.txIndex = s.txIndex()
	var updates chan feedUpdate
	if s.feed() {
		updates = env.withFeed()
	}
	env.bootstrapping = s.bootstrapping()
	env.proc.sealing = s.sealing()
	sealedIDs := []idx.ValidatorID{}
	for id := idx.ValidatorID(1); id <= 3; id++ {
		if s.sealedValidators&(1<<(id-1)) != 0 {
			sealedIDs = append(sealedIDs, id)
		}
	}
	sealedValidators, sealedProfiles := beginBlockValidators(sealedIDs...)
	env.proc.sealResult = func(bs iblockproc.BlockState, es iblockproc.EpochState) (iblockproc.BlockState, iblockproc.EpochState) {
		es.Validators, es.ValidatorProfiles = sealedValidators, sealedProfiles
		es.Rules.Blocks.MaxBlockGas += 64
		if s.upgradesChange() {
			es.Rules.Upgrades.TransactionBundles = !es.Rules.Upgrades.TransactionBundles
			es.Rules.Upgrades.Brio = !es.Rules.Upgrades.Brio
		}
		return bs, es
	}
	var preSkipped, postSkipped map[common.Hash]bool
	env.proc.preTxs, preSkipped = beginBlockFuzzInternalTxs(s.preTxs, 0)
	env.proc.postTxs, postSkipped = beginBlockFuzzInternalTxs(s.postTxs, 10)
	env.proc.skipped = map[common.Hash]bool{}
	for _, tx := range beginBlockFuzzPoolTxs(s.skippedUserTxs) {
		env.proc.skipped[tx.Hash()] = true
	}
	for hash := range preSkipped {
		env.proc.skipped[hash] = true
	}
	for hash := range postSkipped {
		env.proc.skipped[hash] = true
	}
	env.proc.failed = map[common.Hash]bool{}
	for _, tx := range beginBlockFuzzPoolTxs(s.failedUserTxs) {
		env.proc.failed[tx.Hash()] = true
	}
	// The logs are created per environment, as the callbacks modify them.
	env.proc.logs = map[common.Hash][]*types.Log{}
	for _, tx := range beginBlockFuzzPoolTxs(s.loggingUserTxs) {
		for i := range 2 {
			env.proc.logs[tx.Hash()] = append(env.proc.logs[tx.Hash()], &types.Log{
				Address:     beginBlockFuzzLogAddress,
				Topics:      []common.Hash{tx.Hash()},
				BlockNumber: uint64(env.nextBlock()),
				TxHash:      tx.Hash(),
				Index:       uint(i),
			})
		}
	}
	env.proc.derived = map[common.Hash]types.Transactions{}
	for i, tx := range beginBlockFuzzPool() {
		if s.derivingUserTxs&(1<<i) != 0 {
			env.proc.derived[tx.Hash()] = types.Transactions{beginBlockTx(4, 20+uint64(i))}
		}
	}

	events := []*inter.EventPayload{}
	for i, event := range s.events {
		events = append(events, event.build(env, i))
	}
	applied := slices.Clone(events)
	if len(applied) > 0 {
		rotation := int(s.order) % len(applied)
		applied = append(applied[rotation:], applied[:rotation]...)
		if s.order&0x80 != 0 {
			slices.Reverse(applied)
		}
	}
	if d := int(s.duplicate); d > 0 && d <= len(events) {
		applied = append(applied, events[d-1])
	}
	block := &lachesis.Block{Cheaters: beginBlockFuzzCheaters(s.blockCheaters)}
	switch choice := int(s.atropos) % (len(events) + 2); {
	case choice < len(events):
		block.Atropos = events[choice].ID()
	case choice == len(events):
		atropos := beginBlockEvent(beginBlockEventSpec{creator: 3, lamport: 5, time: beginBlockFuzzTime(s.atroposTime)})
		applied = append(applied, atropos)
		block.Atropos = atropos.ID()
	default:
		block.Atropos = beginBlockEvent(beginBlockEventSpec{creator: 3, lamport: 6, time: beginBlockFuzzTime(s.atroposTime)}).ID()
	}

	env.beginBlock = newBeginBlockFn(
		env.workers, &env.wg, &env.busy, env.store, env.proc.blockProc(), env.txIndex,
		env.feed, &env.emitters, env.verWatcher, &env.bootstrapping,
	)
	// The states in the store are observed at every call of a module.
	states := []hash.Hash{}
	env.proc.hook = func(any) {
		bs, es := env.store.GetBlockEpochState()
		states = append(states, bs.Hash(), es.Hash())
	}
	meters, gauges := beginBlockFuzzMetrics()
	counts := []int64{}
	for _, meter := range meters {
		counts = append(counts, meter.Snapshot().Count())
	}
	for _, gauge := range gauges {
		gauge.Update(-1)
	}
	validators := env.runBlock(block, applied...)
	o := beginBlockObserve(env, events, validators, updates)
	o.StatesAtCalls = states
	for i, meter := range meters {
		o.Metrics = append(o.Metrics, meter.Snapshot().Count()-counts[i])
	}
	for _, gauge := range gauges {
		o.Metrics = append(o.Metrics, gauge.Snapshot().Value())
	}
	return o
}

// beginBlockFuzzMetrics returns the deterministically updated meters and gauges,
// which both implementations share.
func beginBlockFuzzMetrics() ([]*metrics.Meter, []*metrics.Gauge) {
	return []*metrics.Meter{
			confirmedEventsMeter, spilledEventsMeter, invalidTxsMeter, processedTxsMeter, skippedTxsMeter,
			priorityFailures.config.(*metrics.Meter), priorityFailures.txs.(*metrics.Meter),
		},
		[]*metrics.Gauge{headBlockGauge, headHeaderGauge, headFastBlockGauge}
}

// ---------------------------------------------------------------------------
// Observation
// ---------------------------------------------------------------------------

// beginBlockFuzzObservation holds the observable effects of processing a block in a
// form that is comparable across environments.
type beginBlockFuzzObservation struct {
	Calls                   [][]any
	StatesAtCalls           []hash.Hash // block and epoch states in the store at each call
	Validators              *pos.Validators
	Busy                    uint32
	BlockState, EpochState  hash.Hash
	CachedEvmBlock          []any
	Feed                    [][]any
	ProposalsAndTxsOfEvents [][]common.Hash
	Metrics                 []int64
	Databases               map[string][][2][]byte // all entries of all databases of the store
}

func beginBlockObserve(env *beginBlockEnv, events []*inter.EventPayload, validators *pos.Validators, updates chan feedUpdate) beginBlockFuzzObservation {
	o := beginBlockFuzzObservation{Validators: validators, Busy: env.busy, Databases: map[string][][2][]byte{}}
	for _, call := range env.proc.calls {
		o.Calls = append(o.Calls, beginBlockNormalizeCall(call))
	}
	bs, es := env.store.GetBlockEpochState()
	o.BlockState, o.EpochState = bs.Hash(), es.Hash()
	// The cached EVM block of the produced block, or of its parent if the block is skipped.
	if block := env.store.EvmStore().GetCachedEvmBlock(bs.LastBlock.Idx); block != nil {
		o.CachedEvmBlock = []any{block.EvmHeader, beginBlockHashes(block.Transactions)}
	}
	for len(updates) > 0 {
		update := <-updates
		o.Feed = append(o.Feed, []any{update.block.EvmHeader, beginBlockHashes(update.block.Transactions), update.logs})
	}
	for _, event := range events {
		txs := beginBlockHashes(event.TransactionsToMeter())
		if proposal := event.Payload().Proposal; proposal != nil {
			txs = append(txs, beginBlockHashes(proposal.Transactions)...)
		}
		o.ProposalsAndTxsOfEvents = append(o.ProposalsAndTxsOfEvents, txs)
	}
	dbs := env.store.dbs.(*flushable.SyncedPool)
	for _, name := range dbs.Names() {
		db, err := dbs.OpenDB(name)
		require.NoError(env.t, err)
		it := db.NewIterator(nil, nil)
		for it.Next() {
			o.Databases[name] = append(o.Databases[name], [2][]byte{slices.Clone(it.Key()), slices.Clone(it.Value())})
		}
		it.Release()
	}
	return o
}

// beginBlockNormalizeCall reduces block states and transactions to their hashes and
// environment-specific instances to their presence.
func beginBlockNormalizeCall(call any) []any {
	name := beginBlockCallName(call)
	switch c := call.(type) {
	case beginBlockEventsStart:
		return []any{name, c.BS.Hash(), c.ES.Hash()}
	case beginBlockSealerStart:
		return []any{name, c.Ctx, c.BS.Hash(), c.ES.Hash()}
	case beginBlockSealerUpdate:
		return []any{name, c.BS.Hash(), c.ES.Hash()}
	case beginBlockSealEpoch:
		return []any{name, c.LastBlockHash, c.ExecPlanHash, beginBlockHashes(c.Txs)}
	case beginBlockListenerStart:
		return []any{name, c.Ctx, c.BS.Hash(), c.ES.Hash()}
	case beginBlockListenerReceipt:
		return []any{name, c.Tx.Hash(), c.Receipt, c.ReceiptBlockHash, c.Creator, c.BaseFee, c.BlobBaseFee}
	case beginBlockListenerUpdate:
		return []any{name, c.BS.Hash(), c.ES.Hash()}
	case beginBlockPopInternalTxs:
		return []any{name, c.Ctx, c.BS.Hash(), c.ES.Hash(), c.Sealing, c.Nonce}
	case beginBlockEvmStart:
		return []any{name, c.Number, c.Time, c.Epoch, c.StateDB != nil, fmt.Sprintf("%T", c.Reader), c.OnNewLog != nil, c.Rules, c.ChainCfg, c.Randao, c.Metrics != nil}
	case beginBlockExecute:
		return []any{name, beginBlockHashes(c.Txs), c.GasLimit, c.SizeLimit, c.Busy}
	}
	return []any{name, call}
}

// ---------------------------------------------------------------------------
// Seeds
// ---------------------------------------------------------------------------

func beginBlockFuzzSeeds() []beginBlockFuzzScenario {
	const (
		singleProposer = 4
		sealing        = 8
		txIndex        = 16
		feed           = 32
		priorities     = 64
		upgradesChange = 128
		brio           = 3
		bundles        = 4
	)
	transfers := beginBlockFuzzEvent{creator: 0, lamport: 1, timeOffset: 10, gas: 10, kind: 1, txs: 0b0000_0111}
	allKinds := beginBlockFuzzEvent{creator: 1, lamport: 1, timeOffset: 10, kind: 1, txs: 0b1111_1111_1000_0001}
	proposal := beginBlockFuzzEvent{creator: 1, lamport: 2, timeOffset: 15, gas: 10, kind: 2, txs: 0b0011_0000, reveal: 1}
	return []beginBlockFuzzScenario{
		// Distributed block with two events, failed, logging, deriving and prioritized transactions, a separate Atropos,
		// and a stored parent block with a time before the last block.
		{
			flags: txIndex | feed | priorities, upgrades: brio, maxBlockGas: 200, skipPeriod: 100, allocPerSec: 10, parentTime: -5,
			events:  []beginBlockFuzzEvent{transfers, {creator: 2, lamport: 2, timeOffset: 20, gas: 20, kind: 1, txs: 0b1100_1000}},
			atropos: 2, atroposTime: 20, failedUserTxs: 0b10, loggingUserTxs: 0b1001, derivingUserTxs: 0b100, preTxs: 1, postTxs: 2,
		},
		// Single proposer block with a valid reveal and a deriving transaction, sealing the epoch and changing the upgrades.
		{
			flags: singleProposer | sealing | txIndex | feed | upgradesChange, upgrades: brio, maxBlockGas: 200, skipPeriod: 100, allocPerSec: 10,
			events:  []beginBlockFuzzEvent{proposal, transfers},
			atropos: 0, loggingUserTxs: 0b10_0000, derivingUserTxs: 0b1_0000, preTxs: 2, postTxs: 1, sealedValidators: 0b011,
		},
		// Single proposer block with a non-permissible transaction in the proposal.
		{
			flags: singleProposer | txIndex, upgrades: brio, maxBlockGas: 200, skipPeriod: 100, allocPerSec: 10,
			events:  []beginBlockFuzzEvent{{creator: 1, lamport: 1, timeOffset: 10, kind: 2, txs: 0b1000_0001, reveal: 1}},
			atropos: 0,
		},
		// Competing proposals of different turns with valid reveals, and proposals for a future block and with a wrong
		// parent, with a user gas limit below the max block gas measured from a stored parent block before the last block.
		{
			flags: singleProposer, upgrades: brio, maxBlockGas: 200, skipPeriod: 100, allocPerSec: 1, parentTime: -3,
			events: []beginBlockFuzzEvent{
				{creator: 0, lamport: 0, timeOffset: 5, kind: 2, txs: 0b01, turn: 2, reveal: 1},
				{creator: 1, lamport: 1, timeOffset: 7, kind: 2, txs: 0b10, turn: 1, reveal: 1},
				{creator: 2, lamport: 2, timeOffset: 8, kind: 2, txs: 0b1_0000, turn: 3, reveal: 1},
				{creator: 2, lamport: 3, timeOffset: 9, kind: 2, txs: 0b100, numberOffset: 1},
				{creator: 0, lamport: 3, timeOffset: 9, kind: 2, txs: 0b1000, wrongParent: true},
			},
			atropos: 5, atroposTime: 9,
		},
		// Single proposer block with a valid proposal for the successor of a stored parent block with a number other
		// than its index, sealing the epoch with changed upgrades.
		{
			flags: singleProposer | sealing | txIndex | feed | upgradesChange, upgrades: brio, maxBlockGas: 200, skipPeriod: 100, allocPerSec: 10,
			parentNumber: 2, events: []beginBlockFuzzEvent{proposal, transfers}, atropos: 0, loggingUserTxs: 0b10_0000,
		},
		// Distributed block on top of a stored parent block with a number other than its index.
		{
			flags: txIndex, upgrades: brio, maxBlockGas: 200, skipPeriod: 100, parentNumber: 1,
			events: []beginBlockFuzzEvent{transfers}, atropos: 1, atroposTime: 5,
		},
		// Single proposer block of confirmed events without a valid proposal and without cheaters.
		{
			flags: singleProposer, upgrades: brio, maxBlockGas: 200, skipPeriod: 100, allocPerSec: 10,
			events:  []beginBlockFuzzEvent{{creator: 0, lamport: 1, timeOffset: 5, kind: 2, txs: 0b1, numberOffset: 1}, transfers},
			atropos: 2, atroposTime: 6,
		},
		// Single proposer block without a valid proposal, produced for the cheaters of the block, with an event carrying
		// a proposal sync state only.
		{
			flags: singleProposer, upgrades: brio, maxBlockGas: 200, skipPeriod: 100, allocPerSec: 10, blockCheaters: []uint8{1},
			events: []beginBlockFuzzEvent{
				{creator: 0, lamport: 1, timeOffset: 5, kind: 2, txs: 0b1, wrongParent: true, reveal: 1},
				transfers,
				{creator: 2, lamport: 2, timeOffset: 5, turn: 1},
			},
			atropos: 3, atroposTime: 6,
		},
		// Single proposer block with an Atropos of the previous epoch, whose keys verify the reveal, carrying a proposal
		// sync state only.
		{
			flags: singleProposer, upgrades: brio, maxBlockGas: 200, skipPeriod: 100, allocPerSec: 10,
			events:  []beginBlockFuzzEvent{{creator: 1, lamport: 2, timeOffset: 10, kind: 2, txs: 0b1, reveal: 2}, {creator: 2, lamport: 3, oldEpoch: true, timeOffset: 12, turn: 1}},
			atropos: 1,
		},
		// Proposals of equal turns, once with each order of their hashes, and equal proposals of different events.
		{
			flags: singleProposer, upgrades: brio, maxBlockGas: 200, skipPeriod: 100, allocPerSec: 10,
			events: []beginBlockFuzzEvent{
				{creator: 0, lamport: 1, timeOffset: 5, kind: 2, txs: 0b01, turn: 1},
				{creator: 1, lamport: 2, timeOffset: 5, kind: 2, txs: 0b10, turn: 1},
			},
			atropos: 2,
		},
		{
			flags: singleProposer, upgrades: brio, maxBlockGas: 200, skipPeriod: 100, allocPerSec: 10,
			events: []beginBlockFuzzEvent{
				{creator: 0, lamport: 1, timeOffset: 5, kind: 2, txs: 0b10, turn: 1},
				{creator: 1, lamport: 2, timeOffset: 5, kind: 2, txs: 0b01, turn: 1},
			},
			atropos: 2,
		},
		{
			flags: singleProposer, upgrades: brio, maxBlockGas: 200, skipPeriod: 100, allocPerSec: 10,
			events: []beginBlockFuzzEvent{
				{creator: 0, lamport: 0, timeOffset: 5, kind: 2, txs: 0b01, turn: 1},
				{creator: 1, lamport: 1, timeOffset: 7, kind: 2, txs: 0b01, turn: 1},
			},
			atropos: 2,
		},
		// Pre-Sonic block with a proposal event of lower Lamport time applied after a transfer event.
		{
			upgrades: 0, maxBlockGas: 200, skipPeriod: 100,
			events:  []beginBlockFuzzEvent{transfers, {creator: 1, lamport: 0, timeOffset: 5, kind: 2, txs: 0b1000}},
			atropos: 2, atroposTime: 5,
		},
		// Pre-Sonic block with a duplicated event and spilled events.
		{
			upgrades: 0, maxBlockGas: 1, skipPeriod: 100,
			events:    []beginBlockFuzzEvent{transfers, {creator: 1, lamport: 3, timeOffset: 1, gas: 60, kind: 1, txs: 0b0100_0001}},
			duplicate: 2, atropos: 0, order: 0x81,
		},
		// Events using exactly the max block gas after a spilled event.
		{
			upgrades: brio, maxBlockGas: 1, skipPeriod: 100,
			events: []beginBlockFuzzEvent{
				{creator: 0, lamport: 1, timeOffset: 5, gas: 1, kind: 1, txs: 0b1},
				{creator: 1, lamport: 2, timeOffset: 5, gas: 32, kind: 1, txs: 0b10},
				{creator: 2, lamport: 3, timeOffset: 5, gas: 32, kind: 1, txs: 0b100},
			},
			atropos: 3, atroposTime: 5,
		},
		// Empty block within the skip period.
		{upgrades: brio, skipPeriod: 100, atropos: 0, atroposTime: 5},
		// Empty block within the skip period of the block state, but not of the earlier parent block.
		{upgrades: brio, skipPeriod: 100, parentTime: -10, atropos: 0, atroposTime: 95},
		// Empty block before the last block without a skip period.
		{upgrades: brio, atropos: 0, atroposTime: -5},
		// Empty block at the end of the skip period.
		{upgrades: brio, skipPeriod: 1, atropos: 0, atroposTime: 1},
		// Empty block within the skip period with stored cheaters only.
		{upgrades: brio, skipPeriod: 100, storedCheaters: []uint8{1}, atropos: 0, atroposTime: 5},
		// Block of spilled events only.
		{upgrades: brio, maxBlockGas: 1, skipPeriod: 100, events: []beginBlockFuzzEvent{{creator: 0, lamport: 1, timeOffset: 5, gas: 65, kind: 1, txs: 0b1}}, atropos: 1, atroposTime: 5},
		// Empty block within the skip period with cheaters, one of them stored already.
		{upgrades: brio, skipPeriod: 100, storedCheaters: []uint8{0, 0}, blockCheaters: []uint8{0, 2}, atropos: 0, atroposTime: 5},
		// Block at the time of the last block.
		{upgrades: brio, maxBlockGas: 200, skipPeriod: 100, events: []beginBlockFuzzEvent{transfers}, atropos: 1},
		// Block before the last block, sealing the epoch with changed upgrades, with logging transactions.
		{
			flags: sealing | upgradesChange | txIndex, upgrades: brio, maxBlockGas: 200, skipPeriod: 100,
			events: []beginBlockFuzzEvent{transfers}, atropos: 1, atroposTime: -5, loggingUserTxs: 0b1,
		},
		// Degenerate Atropos.
		{upgrades: brio, skipPeriod: 1, events: []beginBlockFuzzEvent{transfers}, atropos: 2, atroposTime: 5},
		// Bootstrapping.
		{flags: 3, upgrades: brio, events: []beginBlockFuzzEvent{transfers}},
		// Non-permissible transactions in the distributed mode and priorities.
		{
			flags: priorities | txIndex, upgrades: 2, maxBlockGas: 200, skipPeriod: 100,
			events:  []beginBlockFuzzEvent{{creator: 1, lamport: 1, timeOffset: -3, kind: 1, txs: 0b1000_0001}},
			atropos: 1, atroposTime: -5, skippedUserTxs: 0b1,
		},
		// Transactions of all kinds distinguished by isPermissible, not filtered by Sonic, with skipped internal transactions.
		{upgrades: 1, maxBlockGas: 200, skipPeriod: 100, events: []beginBlockFuzzEvent{allKinds}, preTxs: 6, postTxs: 5},
		// Transactions of all kinds distinguished by isPermissible, filtered by Allegro, Brio, and Brio with bundles.
		{upgrades: 2, maxBlockGas: 200, skipPeriod: 100, events: []beginBlockFuzzEvent{allKinds}},
		{upgrades: brio, maxBlockGas: 200, skipPeriod: 100, events: []beginBlockFuzzEvent{allKinds}},
		{upgrades: bundles, maxBlockGas: 200, skipPeriod: 100, events: []beginBlockFuzzEvent{allKinds}},
		// An event of the previous epoch with transactions being the Atropos, sealing the epoch with a single validator.
		{
			flags: sealing, upgrades: brio, maxBlockGas: 255, skipPeriod: 100,
			events:  []beginBlockFuzzEvent{{creator: 2, lamport: 4, oldEpoch: true, timeOffset: 30, kind: 1, txs: 0b10}, transfers},
			atropos: 0, sealedValidators: 0b100,
		},
	}
}
