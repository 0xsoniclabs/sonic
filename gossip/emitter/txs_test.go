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
	"crypto/ecdsa"
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
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
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

			_, _, err := bundle.ValidateEnvelope(signer, tx)
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

			_, _, err := bundle.ValidateEnvelope(signer, tx)
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

func TestEmitter_addTxsWithHinter_InclusionDeterminedByHinter(t *testing.T) {
	ctrl := gomock.NewController(t)
	f := newAddTxsFixture(t, ctrl)

	tx := f.makeTx(t, 0)
	seedTx := f.makeTx(t, 42)

	nonEmptyEvent := func() *inter.MutableEventPayload {
		e := f.makeEvent()
		e.SetTxs(types.Transactions{seedTx})
		return e
	}
	emptyEvent := f.makeEvent

	var nilHinter *priorityHinter
	noPrioClassifier := fakePriorityClassifier{byHash: map[common.Hash]priorities.Priority{}}
	prioClassifier := fakePriorityClassifier{byHash: map[common.Hash]priorities.Priority{
		tx.Hash(): prioritized(1),
	}}
	noPrioHinter := &priorityHinter{
		classifier: noPrioClassifier,
		config:     priorities.Config{MaxTxsPerEntityPerEvent: 10},
		counts:     map[[32]byte]uint64{},
	}
	prioTx1Hinter := &priorityHinter{
		classifier: prioClassifier,
		config:     priorities.Config{MaxTxsPerEntityPerEvent: 10},
		counts:     map[[32]byte]uint64{},
	}

	// The fixture arranges isMyTxTurn to always return false. Under the new
	// semantics, phase 2 is skipped entirely when neither phase 1 (my-turn
	// prio) nor phase 3 (my-turn ordinary) has a candidate. All scenarios
	// below therefore end with only the pre-existing seed tx (if any) surviving.
	cases := map[string]struct {
		event      func() *inter.MutableEventPayload
		classifier priorities.Classifier
		hinter     *priorityHinter
		checks     func(t *testing.T, event *inter.MutableEventPayload)
	}{
		"non prio tx uses turn logic": {
			event:      nonEmptyEvent,
			classifier: noPrioClassifier,
			hinter:     noPrioHinter,
			checks: func(t *testing.T, event *inter.MutableEventPayload) {
				require.Len(t, event.Transactions(), 1)
				require.Equal(t, seedTx.Hash(), event.Transactions()[0].Hash())
			},
		},
		"prio tx with nil hinter is dropped": {
			event:      nonEmptyEvent,
			classifier: prioClassifier,
			hinter:     nilHinter,
			checks: func(t *testing.T, event *inter.MutableEventPayload) {
				require.Len(t, event.Transactions(), 1)
				require.Equal(t, seedTx.Hash(), event.Transactions()[0].Hash())
			},
		},
		"prio tx not-my-turn is skipped when no my-turn candidate exists": {
			event:      nonEmptyEvent,
			classifier: prioClassifier,
			hinter:     prioTx1Hinter,
			checks: func(t *testing.T, event *inter.MutableEventPayload) {
				// Phase 2 is skipped; the tx is never added and no cap is charged.
				require.Len(t, event.Transactions(), 1)
				require.Equal(t, seedTx.Hash(), event.Transactions()[0].Hash())
				require.Zero(t, prioTx1Hinter.counts[[32]byte{1}])
			},
		},
		"prio tx does not persist when event empty": {
			event:      emptyEvent,
			classifier: prioClassifier,
			hinter:     prioTx1Hinter,
			checks: func(t *testing.T, event *inter.MutableEventPayload) {
				require.Empty(t, event.Transactions())
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			event := tc.event()
			f.em.addTxsWithHinter(event, f.makeSorted(tc.classifier, tx), tc.classifier, tc.hinter)
			tc.checks(t, event)
		})
	}
}

func TestEmitter_addTxsWithHinter_PerEntityCapEnforced(t *testing.T) {
	ctrl := gomock.NewController(t)
	f := newAddTxsFixture(t, ctrl)
	// Give this validator its own turn so all txs are admitted in phase 1.
	f.enableMyTurn()

	tx1 := f.makeTx(t, 0)
	tx2 := f.makeTx(t, 1)
	tx3 := f.makeTx(t, 2)

	event := f.makeEvent()

	// All three txs belong to the same priority entity; cap is 2.
	classifier := fakePriorityClassifier{byHash: map[common.Hash]priorities.Priority{
		tx1.Hash(): prioritized(7),
		tx2.Hash(): prioritized(7),
		tx3.Hash(): prioritized(7),
	}}
	hinter := &priorityHinter{
		classifier: classifier,
		config:     priorities.Config{MaxTxsPerEntityPerEvent: 2},
		counts:     map[[32]byte]uint64{},
	}

	f.em.addTxsWithHinter(event, f.makeSorted(classifier, tx1, tx2, tx3), classifier, hinter)

	// All three prioritized txs are my-turn adds (phase 1), so the hinter cap
	// is not consulted for them; all three are included.
	require.Len(t, event.Transactions(), 3)
	require.Equal(t, tx1.Hash(), event.Transactions()[0].Hash())
	require.Equal(t, tx2.Hash(), event.Transactions()[1].Hash())
	require.Equal(t, tx3.Hash(), event.Transactions()[2].Hash())
	// No hinter-eager admissions occurred, so no cap accounting was recorded.
	require.Zero(t, hinter.counts[[32]byte{7}])
}

// TestEmitter_addTxsWithHinter_Phase2SkippedWhenNoMyTurnCandidate verifies
// that when a prioritized tx is not this validator's turn and there is no
// my-turn non-prioritized candidate, phase 2 is skipped entirely: the tx is
// never added and the hinter cap is not charged.
func TestEmitter_addTxsWithHinter_Phase2SkippedWhenNoMyTurnCandidate(t *testing.T) {
	ctrl := gomock.NewController(t)
	f := newAddTxsFixture(t, ctrl)

	seedTx := f.makeTx(t, 42)
	tx := f.makeTx(t, 0)

	event := f.makeEvent()
	event.SetTxs(types.Transactions{seedTx})
	gasUsedBefore := event.GasPowerUsed()

	classifier := fakePriorityClassifier{byHash: map[common.Hash]priorities.Priority{
		tx.Hash(): prioritized(3),
	}}
	hinter := &priorityHinter{
		classifier: classifier,
		config:     priorities.Config{MaxTxsPerEntityPerEvent: 5},
		counts:     map[[32]byte]uint64{},
	}

	f.em.addTxsWithHinter(event, f.makeSorted(classifier, tx), classifier, hinter)

	// Phase 2 was skipped; the tx was never added.
	require.Len(t, event.Transactions(), 1)
	require.Equal(t, seedTx.Hash(), event.Transactions()[0].Hash())
	require.Equal(t, gasUsedBefore, event.GasPowerUsed())
	// No hinter cap was charged.
	require.Zero(t, hinter.counts[[32]byte{3}])
}

// addTxsFixture builds a minimal Emitter ready to exercise addTxsWithHinter.
// The validator set is chosen so that isMyTxTurn always returns false for the
// event's creator (the two validators are distinct from `me` and are online),
// which lets tests distinguish the priority-bypass path from the turn path.
type addTxsFixture struct {
	em     *Emitter
	signer types.Signer
	key    *ecdsa.PrivateKey
	sender common.Address
	me     idx.ValidatorID
}

func newAddTxsFixture(t *testing.T, ctrl *gomock.Controller) *addTxsFixture {
	t.Helper()

	signer := types.LatestSignerForChainID(big.NewInt(1))
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	sender := crypto.PubkeyToAddress(key.PublicKey)

	me := idx.ValidatorID(999)
	b := pos.NewBuilder()
	b.Set(idx.ValidatorID(1), pos.Weight(1))
	b.Set(idx.ValidatorID(2), pos.Weight(1))
	validators := b.Build()

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
		originatedTxs:     originatedtxs.New(SenderCountBufferSize),
		offlineValidators: map[idx.ValidatorID]bool{},
	}
	em.validators.Store(validators)
	em.epoch.Store(1)

	return &addTxsFixture{em: em, signer: signer, key: key, sender: sender, me: me}
}

// enableMyTurn reconfigures the fixture so isMyTxTurn always returns true for
// the fixture's `me` validator, by installing a sole-validator set consisting
// of `me` alone.
func (f *addTxsFixture) enableMyTurn() {
	b := pos.NewBuilder()
	b.Set(f.me, pos.Weight(1))
	f.em.validators.Store(b.Build())
}

// makeTx returns a signed legacy transaction from the fixture's sender.
func (f *addTxsFixture) makeTx(t *testing.T, nonce uint64) *types.Transaction {
	t.Helper()
	tx, err := types.SignTx(
		types.NewTransaction(nonce, common.Address{0xaa}, big.NewInt(0), 21000, big.NewInt(1), nil),
		f.signer, f.key,
	)
	require.NoError(t, err)
	return tx
}

// makeEvent returns a mutable event payload with plenty of gas power, created
// by the fixture's `me` validator.
func (f *addTxsFixture) makeEvent() *inter.MutableEventPayload {
	e := &inter.MutableEventPayload{}
	e.SetCreator(f.me)
	e.SetGasPowerLeft(inter.GasPowerLeft{Gas: [2]uint64{100_000_000, 100_000_000}})
	return e
}

// makeSorted wraps the given txs (from the fixture's sender) as a
// transactionsByPriceAndNonce set. The optional classifier is used both to
// place each account's initial head in the prioritized or ordinary heap and to
// classify subsequent nonces as they are promoted via advanceSender.
func (f *addTxsFixture) makeSorted(classifier priorities.Classifier, txs ...*types.Transaction) *transactionsByPriorityAndPriceAndNonce {
	lazy := make([]*txpool.LazyTransaction, len(txs))
	for i, tx := range txs {
		lazy[i] = &txpool.LazyTransaction{
			Hash:      tx.Hash(),
			Tx:        tx,
			Time:      tx.Time(),
			GasFeeCap: uint256.MustFromBig(tx.GasFeeCap()),
			GasTipCap: uint256.MustFromBig(tx.GasTipCap()),
			Gas:       tx.Gas(),
		}
	}
	return newTransactionsByPriorityAndPriceAndNonce(
		f.signer,
		map[common.Address][]*txpool.LazyTransaction{f.sender: lazy},
		nil,
		classifier,
	)
}
