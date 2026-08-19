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

package emitter

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities"
	"github.com/0xsoniclabs/sonic/gossip/emitter/config"
	"github.com/0xsoniclabs/sonic/gossip/emitter/originatedtxs"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_DefaultMaxTxsPerAddress_Equals_txTurnNonces(t *testing.T) {

	// Although MaxTxsPerAddress can be configured, having a value less than txTurnNonces
	// could yield performance issues when dispatching batches of transactions.
	// MaxTxsPerAddress should be greater or equal to txTurnNonces to ensure timely
	// emission of transactions. Default value for this parameter should be exactly txTurnNonces.

	defaultConfig := config.DefaultConfig()
	require.EqualValues(t, txTurnNonces, defaultConfig.MaxTxsPerAddress, "Default MaxTxsPerAddress should equal txTurnNonces")
}

func TestEmitter_addTxs_PerEntityCapEnforcedForPrioritizedNotMyTurnTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)
	f := newAddTxsFixture(t, ctrl)

	// 3 prioritized txs & 1 ordinary tx whose turn it is, so eager admissions are not rolled back.
	prio1 := f.makeTx(t)
	prio2 := f.makeTx(t)
	prio3 := f.makeTx(t)
	ordinary := f.makeTx(t)

	tests := map[string]struct {
		entityCap  uint64
		entities   [3]byte
		turn       func(tx *txpool.LazyTransaction) bool
		wantTxs    []*types.Transaction
		wantCapped int64
	}{
		"my-turn prio bypasses cap":       {0, [3]byte{7, 7, 7}, alwaysMyTurn, []*types.Transaction{prio1, prio2, prio3, ordinary}, 0},
		"not-my-turn prio subject to cap": {2, [3]byte{7, 7, 7}, myTurnFor(ordinary), []*types.Transaction{prio1, prio2, ordinary}, 1},
		"cap applies per entity":          {1, [3]byte{7, 7, 8}, myTurnFor(ordinary), []*types.Transaction{prio1, prio3, ordinary}, 1},
		"zero cap admits nothing eagerly": {0, [3]byte{7, 7, 7}, myTurnFor(ordinary), []*types.Transaction{ordinary}, 3},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			f.em.cache.priorityConfig.MaxPiggybackTxsPerEntityPerEvent = test.entityCap
			lookup := map[common.Hash]priorities.Priority{
				prio1.Hash(): prio(1, 1, test.entities[0]),
				prio2.Hash(): prio(1, 1, test.entities[1]),
				prio3.Hash(): prio(1, 1, test.entities[2]),
			}

			capped := counterDelta(txsSkippedPiggybackLimit)

			event := f.makeEvent()
			f.em.addTxs(event, f.makeSorted(t, lookup, test.turn, prio1, prio2, prio3, ordinary))

			require.Equal(t, hashes(test.wantTxs), hashes(event.Transactions()))
			require.Equal(t, test.wantCapped, capped())
		})
	}
}

// TestEmitter_addTxs_EagerAdmissionsAreBoundedByGasShare verifies that the eager
// path cannot consume more than half of the event's gas budget, so large
// prioritized transactions of other validators' turns leave room for this
// validator's own ones.
func TestEmitter_addTxs_EagerAdmissionsAreBoundedByGasShare(t *testing.T) {
	ctrl := gomock.NewController(t)
	f := newAddTxsFixture(t, ctrl) // MaxEventGas is 100_000_000
	f.em.cache.priorityConfig.MaxPiggybackTxsPerEntityPerEvent = 5

	// Two prioritized txs of another validator's turn, each fitting into the
	// gas share on its own but not together, and one ordinary tx of this
	// validator's turn that only fits if the second one is dropped.
	foreign1 := f.makeTxs(t, 40_000_000, 0)[0]
	foreign2 := f.makeTxs(t, 40_000_000, 0)[0]
	mine := f.makeTxs(t, 30_000_000, 0)[0]

	lookup := map[common.Hash]priorities.Priority{
		foreign1.Hash(): prio(1, 1, 7),
		foreign2.Hash(): prio(1, 1, 7),
	}

	event := f.makeEvent()
	f.em.addTxs(event, f.makeSorted(t, lookup, myTurnFor(mine), foreign1, foreign2, mine))

	require.Equal(t,
		[]common.Hash{foreign1.Hash(), mine.Hash()},
		hashes(event.Transactions()),
	)
}

func TestEmitter_addTxs_EagerAdmissionsNeedAnOwnContributionOtherwiseRollback(t *testing.T) {
	type candidate struct {
		prio   bool
		myTurn bool
	}

	tests := map[string]struct {
		txs          []candidate
		expectIdx    []uint
		wantRollback int64
	}{
		"no own contribution rolls the event back": {
			[]candidate{{true, false}, {true, false}},
			[]uint{},
			2,
		},
		"promoted my-turn nonce keeps the event": {
			[]candidate{{true, false}, {true, true}},
			[]uint{0, 1},
			0,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			f := newAddTxsFixture(t, ctrl)
			f.em.cache.priorityConfig.MaxPiggybackTxsPerEntityPerEvent = 5

			// Consecutive nonces of one sender, so that a candidate only surfaces
			// once its predecessor has been admitted.
			nonces := make([]uint64, len(test.txs))
			for i := range nonces {
				nonces[i] = uint64(i)
			}
			txs := f.makeTxs(t, params.TxGas, nonces...)

			lookup := map[common.Hash]priorities.Priority{}
			var myTurn []*types.Transaction
			for i, candidate := range test.txs {
				if candidate.prio {
					lookup[txs[i].Hash()] = prio(1, 1, 3)
				}
				if candidate.myTurn {
					myTurn = append(myTurn, txs[i])
				}
			}

			expected := make(types.Transactions, len(test.expectIdx))
			for i, index := range test.expectIdx {
				expected[i] = txs[index]
			}

			event := f.makeEvent()
			gasPowerLeft := event.GasPowerLeft()
			rolledBack := counterDelta(txsSkippedPiggybackRollback)
			f.em.addTxs(event, f.makeSorted(t, lookup, myTurnFor(myTurn...), txs...))

			// only the expected transactions are included, and the gasUsed and gasPowerLeft only reflects the included ones
			gasUsed := uint64(len(expected)) * params.TxGas
			require.Equal(t, hashes(expected), hashes(event.Transactions()))
			require.Equal(t, gasUsed, event.GasPowerUsed())
			require.Equal(t, gasPowerLeft.Sub(gasUsed), event.GasPowerLeft())
			require.Equal(t, test.wantRollback, rolledBack())
		})
	}
}

func Test_Emitter_isValidBundleTx_AcceptsValidBundleIfBundlesAreEnabled(t *testing.T) {
	for _, bundlesEnabled := range []bool{true, false} {
		t.Run(fmt.Sprintf("enabled=%t", bundlesEnabled), func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)

			rules := opera.Rules{
				NetworkID: 12,
				Upgrades: opera.Upgrades{
					TransactionBundles: bundlesEnabled,
				},
			}

			db := state.NewMockStateDB(ctrl)
			db.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).AnyTimes()
			db.EXPECT().Release().AnyTimes()

			external := NewMockExternal(ctrl)
			external.EXPECT().GetRules().Return(rules).AnyTimes()
			external.EXPECT().GetLatestBlockIndex().Return(idx.Block(100)).AnyTimes()
			external.EXPECT().StateDB().Return(db).AnyTimes()

			signer := types.LatestSignerForChainID(big.NewInt(int64(rules.NetworkID)))
			emitter := &Emitter{
				world: World{
					External:          external,
					TransactionSigner: signer,
				},
			}

			tx := bundle.NewBuilder().SetEarliest(50).SetRangeLength(100).WithSigner(signer).Build()

			_, _, err := bundle.ValidateEnvelope(signer, tx, opera.Upgrades{})
			require.NoError(err)

			bundleEvaluator := evmcore.NewMockBundleEvaluator(ctrl)
			if bundlesEnabled {
				// if bundles are enabled, it will be evaluated
				bundleEvaluator.EXPECT().GetBundleState(gomock.Any(), gomock.Any(), tx).
					Return(evmcore.BundleState{Executable: true})
			}

			runnable := emitter.isRunnableBundleTxInternal(tx, bundleEvaluator, effectiveBundleGasHistogram)
			require.Equal(bundlesEnabled, runnable)
		})
	}
}

func Test_Emitter_isValidBundleTx_RejectsInvalidBundle(t *testing.T) {
	tests := map[string]*types.Transaction{
		"not a bundle": types.NewTx(&types.LegacyTx{}),
		"invalid bundle data": types.NewTx(&types.LegacyTx{
			To:   &bundle.BundleProcessor,
			Data: []byte{0x01, 0x02, 0x03},
		}),
		"bundle with out-of-range block numbers": bundle.NewBuilder().
			SetEarliest(150).
			SetRangeLength(100).
			Build(),
	}

	for name, tx := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)

			rules := opera.Rules{
				Upgrades: opera.Upgrades{
					TransactionBundles: true,
				},
			}

			state := state.NewMockStateDB(ctrl)
			state.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).AnyTimes()

			external := NewMockExternal(ctrl)
			external.EXPECT().GetRules().Return(rules).AnyTimes()
			external.EXPECT().GetLatestBlockIndex().Return(idx.Block(100)).AnyTimes()
			external.EXPECT().StateDB().Return(state).AnyTimes()

			emitter := &Emitter{
				world: World{External: external},
			}

			valid := emitter.isValidBundleTx(tx)
			require.False(valid)
		})
	}
}

func Test_Emitter_isValidBundleTx_RejectsAlreadyProcessedBundle(t *testing.T) {
	for _, processed := range []bool{true, false} {
		t.Run(fmt.Sprintf("processed=%t", processed), func(t *testing.T) {
			ctrl := gomock.NewController(t)

			rules := opera.Rules{
				Upgrades: opera.Upgrades{
					TransactionBundles: true,
				},
			}

			db := state.NewMockStateDB(ctrl)
			db.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).Return(processed).AnyTimes()
			db.EXPECT().Release().AnyTimes()

			external := NewMockExternal(ctrl)
			external.EXPECT().GetRules().Return(rules).AnyTimes()
			external.EXPECT().GetLatestBlockIndex().Return(idx.Block(100)).AnyTimes()
			external.EXPECT().StateDB().Return(db).AnyTimes()

			signer := types.LatestSignerForChainID(big.NewInt(1))
			emitter := &Emitter{
				world: World{
					External:          external,
					TransactionSigner: signer,
				},
			}

			tx := bundle.NewBuilder().SetEarliest(50).SetRangeLength(100).Build()

			_, _, err := bundle.ValidateEnvelope(signer, tx, opera.Upgrades{})
			require.NoError(t, err)

			bundleEvaluator := evmcore.NewMockBundleEvaluator(ctrl)
			if !processed {
				// if not processed already, it will be evaluated
				bundleEvaluator.EXPECT().GetBundleState(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(evmcore.BundleState{Executable: true})
			}

			valid := emitter.isRunnableBundleTxInternal(tx, bundleEvaluator, effectiveBundleGasHistogram)
			require.Equal(t, !processed, valid)
		})
	}
}

func Test_preCheckStateAdapter_ForwardsNetworkRuleRequest(t *testing.T) {
	rules := opera.Rules{
		NetworkID: 42,
	}

	ctrl := gomock.NewController(t)
	external := NewMockExternal(ctrl)
	external.EXPECT().GetRules().Return(rules)

	adapter := &preCheckChainStateAdapter{external: external}
	returnedRules := adapter.GetCurrentNetworkRules()

	require.Equal(t, rules, returnedRules)
}

func Test_preCheckStateAdapter_ForwardsHeaderRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	header := &evmcore.EvmHeader{}

	hash := common.Hash{1, 2, 3}
	number := uint64(42)

	external := NewMockExternal(ctrl)
	external.EXPECT().Header(hash, number).Return(header)

	adapter := &preCheckChainStateAdapter{external: external}
	returnedHeader := adapter.Header(hash, number)

	require.Same(t, header, returnedHeader)
}

func Test_preCheckStateAdapter_UsesNetworkRulesAndUpgradeHeights(t *testing.T) {
	ctrl := gomock.NewController(t)
	rules := opera.Rules{NetworkID: 42}

	heights := []opera.UpgradeHeight{
		{Height: 100, Upgrades: opera.Upgrades{Sonic: true}},
		{Height: 200, Upgrades: opera.Upgrades{Allegro: true}},
	}

	blockHeight := idx.Block(150)

	external := NewMockExternal(ctrl)
	external.EXPECT().GetRules().Return(rules)
	external.EXPECT().GetUpgradeHeights().Return(heights)
	external.EXPECT().GetLatestBlockIndex().Return(blockHeight)

	adapter := &preCheckChainStateAdapter{external: external}
	got := adapter.GetCurrentChainConfig()

	expected := opera.CreateTransientEvmChainConfig(rules.NetworkID, heights, blockHeight)
	require.Equal(t, expected, got)
}

func Test_preCheckStateAdapter_ForwardsGetLatestHeader(t *testing.T) {
	ctrl := gomock.NewController(t)
	header := &evmcore.EvmHeader{}

	block := inter.Block{}
	block.Number = 42

	external := NewMockExternal(ctrl)
	external.EXPECT().GetLatestBlock().Return(&block)
	external.EXPECT().Header(block.Hash(), block.Number).Return(header)

	adapter := &preCheckChainStateAdapter{external: external}
	returnedHeader := adapter.GetLatestHeader()

	require.Same(t, header, returnedHeader)
}

func Test_Emitter_evaluateBundleTx_ReturnsGasEfficiencyFromEvaluator(t *testing.T) {
	asPointer := func(f float64) *float64 {
		return &f
	}
	tests := map[string]struct {
		gasEfficiency *float64
		executable    bool
	}{
		"low efficiency rejected": {
			gasEfficiency: asPointer(0.1),
			executable:    false,
		},
		"medium efficiency accepted": {
			gasEfficiency: asPointer(0.5),
			executable:    true,
		},
		"full efficiency accepted": {
			gasEfficiency: asPointer(1.0),
			executable:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			rules := opera.Rules{
				NetworkID: 12,
				Upgrades: opera.Upgrades{
					TransactionBundles: true,
				},
			}

			db := state.NewMockStateDB(ctrl)
			db.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).Return(false).AnyTimes()
			db.EXPECT().Release().AnyTimes()

			external := NewMockExternal(ctrl)
			external.EXPECT().GetRules().Return(rules).AnyTimes()
			external.EXPECT().GetLatestBlockIndex().Return(idx.Block(100)).AnyTimes()
			external.EXPECT().StateDB().Return(db).AnyTimes()

			signer := types.LatestSignerForChainID(big.NewInt(int64(rules.NetworkID)))
			emitter := &Emitter{
				world: World{
					External:          external,
					TransactionSigner: signer,
				},
			}

			tx := bundle.NewBuilder().SetEarliest(50).SetRangeLength(100).WithSigner(signer).Build()

			bundleEvaluator := evmcore.NewMockBundleEvaluator(ctrl)
			bundleEvaluator.EXPECT().GetBundleState(gomock.Any(), gomock.Any(), tx).
				Return(evmcore.BundleState{
					Executable:    tc.executable,
					GasEfficiency: tc.gasEfficiency,
				})

			gasEfficiencyMock := utils.NewMockMetricsHistogram(ctrl)
			// ensure the metric is updated with the correct gas efficiency value
			gasEfficiencyMock.EXPECT().Observe(*tc.gasEfficiency)

			valid := emitter.isRunnableBundleTxInternal(tx, bundleEvaluator, gasEfficiencyMock)
			require.Equal(t, tc.executable, valid)
		})
	}
}

// counterDelta captures a counter and returns how much it advanced since the
// capture.
func counterDelta(counter *metrics.Counter) func() int64 {
	before := counter.Snapshot().Count()
	return func() int64 { return counter.Snapshot().Count() - before }
}

func hashes(txs types.Transactions) []common.Hash {
	hashes := make([]common.Hash, len(txs))
	for i, tx := range txs {
		hashes[i] = tx.Hash()
	}
	return hashes
}

// addTxsFixture builds a minimal Emitter ready to exercise addTxs. Its
// cache.priorityConfig carries the rate limits an ordering build would have read
// from the registry, zero by default, which admits nothing eagerly.
type addTxsFixture struct {
	em     *Emitter
	signer types.Signer
}

func newAddTxsFixture(t *testing.T, ctrl *gomock.Controller) *addTxsFixture {
	t.Helper()

	signer := types.LatestSignerForChainID(big.NewInt(1))

	rules := opera.Rules{
		NetworkID: 1,
		Economy: opera.EconomyRules{
			Gas: opera.GasRules{MaxEventGas: 100_000_000},
		},
		Blocks: opera.BlocksRules{MaxBlockGas: 100_000_000},
	}

	external := NewMockExternal(ctrl)
	external.EXPECT().GetRules().Return(rules).AnyTimes()

	txPool := NewMockTxPool(ctrl)
	txPool.EXPECT().Has(gomock.Any()).Return(true).AnyTimes()

	em := &Emitter{
		world: World{
			External:          external,
			TxPool:            txPool,
			TransactionSigner: signer,
		},
		originatedTxs: originatedtxs.New(SenderCountBufferSize),
	}

	return &addTxsFixture{em: em, signer: signer}
}

// makeTx returns a signed legacy transaction from a fresh, unique sender, so
// each tx forms its own account queue in makeSorted.
func (f *addTxsFixture) makeTx(t *testing.T) *types.Transaction {
	t.Helper()
	return f.makeTxs(t, params.TxGas, 0)[0]
}

// makeTxs returns one signed legacy transaction per given nonce, all from a
// single fresh sender and with the given gas limit, so they form one
// nonce-ordered account queue in makeSorted. Pass the nonces in ascending order.
func (f *addTxsFixture) makeTxs(t *testing.T, gas uint64, nonces ...uint64) []*types.Transaction {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	txs := make([]*types.Transaction, len(nonces))
	for i, nonce := range nonces {
		tx, err := types.SignTx(
			types.NewTransaction(nonce, common.Address{0xaa}, big.NewInt(0), gas, big.NewInt(1), nil),
			f.signer, key,
		)
		require.NoError(t, err)
		txs[i] = tx
	}
	return txs
}

// makeEvent returns a mutable event payload with plenty of gas power.
func (f *addTxsFixture) makeEvent() *inter.MutableEventPayload {
	e := &inter.MutableEventPayload{}
	e.SetGasPowerLeft(inter.GasPowerLeft{Gas: [2]uint64{100_000_000, 100_000_000}})
	return e
}

// makeSorted wraps the given txs as a transactionsByPriorityAndPriceAndNonce
// set, grouping them by recovered sender. The priorities and the turn policy
// stage every transaction of the set.
func (f *addTxsFixture) makeSorted(t *testing.T, priorityOf map[common.Hash]priorities.Priority, policy func(tx *txpool.LazyTransaction) bool, txs ...*types.Transaction) *transactionsByPriorityAndPriceAndNonce {
	bySender := map[common.Address][]*txpool.LazyTransaction{}
	for _, tx := range txs {
		sender, _ := types.Sender(f.signer, tx)
		bySender[sender] = append(bySender[sender], &txpool.LazyTransaction{
			Hash:      tx.Hash(),
			Tx:        tx,
			Time:      tx.Time(),
			GasFeeCap: uint256.MustFromBig(tx.GasFeeCap()),
			GasTipCap: uint256.MustFromBig(tx.GasTipCap()),
			Gas:       tx.Gas(),
		})
	}
	classifier := priorities.NewMockClassifier(gomock.NewController(t))
	classifier.EXPECT().Priority(gomock.Any()).DoAndReturn(
		func(tx *types.Transaction) (priorities.Priority, error) {
			return priorityOf[tx.Hash()], nil
		},
	).AnyTimes()

	context := &priorityContext{
		cache:      evmcore.NewPriorityCache(evmcore.DefaultTxPoolConfig),
		classifier: classifier,
	}
	return newTransactionsByPriorityAndPriceAndNonce(bySender, nil, context, policy)
}
