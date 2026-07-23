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

// TestEmitter_addTxsWithHinter_FollowsThreePhases verifies that a single call
// consumes transactions in the documented phase order: prioritized my-turn
// (phase 1), prioritized not-my-turn admitted eagerly via the hinter (phase 2),
// then ordinary my-turn (phase 3). The resulting event lists them in that
// order, and only the phase-2 admission consumes the per-entity hinter cap.
func TestEmitter_addTxsWithHinter_FollowsThreePhases(t *testing.T) {
	ctrl := gomock.NewController(t)
	f := newAddTxsFixture(t, ctrl)

	p1PrioMyTurn := f.makeTx(t, 0)
	p2s := f.makeTxsWithSingleSender(t, 0, 1)
	p2PrioNotMyTurn, p2PrioMyTurn := p2s[0], p2s[1]
	p3s := f.makeTxsWithSingleSender(t, 0, 1)
	p3Ordinary, p3PrioMyTurn := p3s[0], p3s[1]

	classifier := fakePriorityClassifier{byHash: map[common.Hash]priorities.Priority{
		p1PrioMyTurn.Hash():    prioritized(1),
		p2PrioNotMyTurn.Hash(): prioritized(2),
		p2PrioMyTurn.Hash():    prioritized(3),
		p3PrioMyTurn.Hash():    prioritized(4),
	}}
	hinter := &priorityHinter{
		config: priorities.Config{MaxPiggybackTxsPerEntityPerEvent: 10},
		counts: map[[16]byte]uint64{},
	}
	txSet := f.makeSorted(classifier, p1PrioMyTurn, p2PrioNotMyTurn, p2PrioMyTurn, p3Ordinary, p3PrioMyTurn)
	myTurn := myTurnFor(p1PrioMyTurn, p2PrioMyTurn, p3Ordinary, p3PrioMyTurn)

	event := f.makeEvent()
	f.em.addTxsWithHinter(event, txSet, classifier, hinter, myTurn)

	require.Equal(t,
		[]common.Hash{
			p1PrioMyTurn.Hash(),    // phase 1
			p2PrioNotMyTurn.Hash(), // phase 2, not my turn
			p2PrioMyTurn.Hash(),    // phase 2, my turn following its not-my-turn predecessor
			p3Ordinary.Hash(),      // phase 3
			p3PrioMyTurn.Hash(),    // phase 3, my turn following its ordinary predecessor
		},
		txHashes(event.Transactions()),
	)
	// Only the not-my-turn admission consumes the hinter cap.
	require.Len(t, hinter.counts, 1)
	require.Equal(t, uint64(1), hinter.counts[[16]byte{2}])
}

// TestEmitter_addTxsWithHinter_PerEntityCapEnforced verifies that the per-entity
// hinter cap bounds only prioritized transactions admitted while it is not this
// validator's turn (phase 2); prioritized transactions admitted on the
// validator's own turn (phase 1) are unbounded by it.
func TestEmitter_addTxsWithHinter_PerEntityCapEnforced(t *testing.T) {
	ctrl := gomock.NewController(t)
	f := newAddTxsFixture(t, ctrl)

	// Three prioritized txs of the same entity plus one ordinary tx whose turn
	// it always is (so phase 2 is never skipped for lack of a my-turn candidate).
	prio1 := f.makeTx(t, 0)
	prio2 := f.makeTx(t, 0)
	prio3 := f.makeTx(t, 0)
	ordinary := f.makeTx(t, 0)

	classifier := fakePriorityClassifier{byHash: map[common.Hash]priorities.Priority{
		prio1.Hash(): prioritized(7),
		prio2.Hash(): prioritized(7),
		prio3.Hash(): prioritized(7),
	}}

	cases := map[string]struct {
		turn        func(*txpool.LazyTransaction) bool
		wantCounter uint64
		wantTxs     int
	}{
		"my-turn prio bypasses cap": {
			turn:        alwaysMyTurn,
			wantCounter: 0, // all admitted in phase 1, cap never consulted
			wantTxs:     4, // 3 prio + ordinary
		},
		"not-my-turn prio subject to cap": {
			turn:        myTurnFor(ordinary),
			wantCounter: 2, // cap admits 2 of 3 in phase 2, drops the third
			wantTxs:     3, // 2 prio + ordinary
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			hinter := &priorityHinter{
				config: priorities.Config{MaxPiggybackTxsPerEntityPerEvent: 2},
				counts: map[[16]byte]uint64{},
			}
			event := f.makeEvent()

			f.em.addTxsWithHinter(
				event,
				f.makeSorted(classifier, prio1, prio2, prio3, ordinary),
				classifier, hinter, tc.turn,
			)

			require.Len(t, event.Transactions(), tc.wantTxs)
			require.Equal(t, tc.wantCounter, hinter.counts[[16]byte{7}])
		})
	}
}

// TestEmitter_addTxsWithHinter_Phase2SkippedWhenNoMyTurnCandidate verifies the
// "do not emit an event solely for foreign priorities" invariant: when this
// validator has no my-turn transaction of its own to contribute, phase 2 is
// skipped entirely, so prioritized not-my-turn txs are neither emitted nor
// charged against the hinter cap.
func TestEmitter_addTxsWithHinter_Phase2SkippedWhenNoMyTurnCandidate(t *testing.T) {
	ctrl := gomock.NewController(t)
	f := newAddTxsFixture(t, ctrl)

	prioTx := f.makeTx(t, 0)     // prioritized
	ordinaryTx := f.makeTx(t, 0) // ordinary

	classifier := fakePriorityClassifier{byHash: map[common.Hash]priorities.Priority{
		prioTx.Hash(): prioritized(3),
	}}
	hinter := &priorityHinter{
		config: priorities.Config{MaxPiggybackTxsPerEntityPerEvent: 5},
		counts: map[[16]byte]uint64{},
	}

	event := f.makeEvent()
	f.em.addTxsWithHinter(
		event, f.makeSorted(classifier, prioTx, ordinaryTx), classifier, hinter, neverMyTurn,
	)

	require.Empty(t, event.Transactions())
	require.Zero(t, hinter.counts[[16]byte{3}])
}

// TestEmitter_addTxsWithHinter_AllTransactionsTreatedAsNotPrioritized verifies
// that when the hinter is nil (feature disabled) priority classification grants
// no eager inclusion: every transaction is admitted purely by turn, exactly as
// an ordinary one would be. Prioritized txs are only included on their own turn
// (phase 1) and never piggybacked while it is another validator's turn.
func TestEmitter_addTxsWithHinter_AllTransactionsTreatedAsNotPrioritized(t *testing.T) {
	ctrl := gomock.NewController(t)
	f := newAddTxsFixture(t, ctrl)

	prioMyTurn := f.makeTx(t, 0)
	prioNotMyTurn := f.makeTx(t, 0)
	ordinaryMyTurn := f.makeTx(t, 0)
	ordinaryNotMyTurn := f.makeTx(t, 0)

	classifier := fakePriorityClassifier{byHash: map[common.Hash]priorities.Priority{
		prioMyTurn.Hash():    prioritized(1),
		prioNotMyTurn.Hash(): prioritized(2),
	}}

	event := f.makeEvent()
	f.em.addTxsWithHinter(
		event,
		f.makeSorted(classifier, prioMyTurn, prioNotMyTurn, ordinaryMyTurn, ordinaryNotMyTurn),
		classifier, nil, myTurnFor(prioMyTurn, ordinaryMyTurn),
	)

	require.Equal(t,
		[]common.Hash{prioMyTurn.Hash(), ordinaryMyTurn.Hash()},
		txHashes(event.Transactions()),
	)
}

func alwaysMyTurn(*txpool.LazyTransaction) bool { return true }
func neverMyTurn(*txpool.LazyTransaction) bool  { return false }
func myTurnFor(txs ...*types.Transaction) func(*txpool.LazyTransaction) bool {
	mine := make(map[common.Hash]bool, len(txs))
	for _, tx := range txs {
		mine[tx.Hash()] = true
	}
	return func(tx *txpool.LazyTransaction) bool { return mine[tx.Hash] }
}

func txHashes(txs types.Transactions) []common.Hash {
	if len(txs) == 0 {
		return nil
	}
	hashes := make([]common.Hash, len(txs))
	for i, tx := range txs {
		hashes[i] = tx.Hash()
	}
	return hashes
}

// addTxsFixture builds a minimal Emitter ready to exercise addTxsWithHinter.
type addTxsFixture struct {
	em     *Emitter
	signer types.Signer
	me     idx.ValidatorID
}

func newAddTxsFixture(t *testing.T, ctrl *gomock.Controller) *addTxsFixture {
	t.Helper()

	signer := types.LatestSignerForChainID(big.NewInt(1))

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

	return &addTxsFixture{em: em, signer: signer, me: me}
}

// makeTx returns a signed legacy transaction from a fresh, unique sender, so
// each tx forms its own account queue in makeSorted.
func (f *addTxsFixture) makeTx(t *testing.T, nonce uint64) *types.Transaction {
	t.Helper()
	return f.makeTxsWithSingleSender(t, nonce)[0]
}

// makeTxsWithSingleSender returns one signed legacy transaction per given nonce, all from
// a single fresh sender, so they form one nonce-ordered account queue in
// makeSorted. Pass the nonces in ascending order.
func (f *addTxsFixture) makeTxsWithSingleSender(t *testing.T, nonces ...uint64) []*types.Transaction {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	txs := make([]*types.Transaction, len(nonces))
	for i, nonce := range nonces {
		tx, err := types.SignTx(
			types.NewTransaction(nonce, common.Address{0xaa}, big.NewInt(0), 21000, big.NewInt(1), nil),
			f.signer, key,
		)
		require.NoError(t, err)
		txs[i] = tx
	}
	return txs
}

// makeEvent returns a mutable event payload with plenty of gas power, created
// by the fixture's `me` validator.
func (f *addTxsFixture) makeEvent() *inter.MutableEventPayload {
	e := &inter.MutableEventPayload{}
	e.SetCreator(f.me)
	e.SetGasPowerLeft(inter.GasPowerLeft{Gas: [2]uint64{100_000_000, 100_000_000}})
	return e
}

// makeSorted wraps the given txs as a transactionsByPriorityAndPriceAndNonce
// set, grouping them by recovered sender. The classifier is used both to place
// each account's initial head in the prioritized or ordinary heap and to
// classify subsequent nonces as they are promoted via advanceSenderInto.
func (f *addTxsFixture) makeSorted(classifier priorities.Classifier, txs ...*types.Transaction) *transactionsByPriorityAndPriceAndNonce {
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
	return newTransactionsByPriorityAndPriceAndNonce(f.signer, bySender, nil, classifier)
}
