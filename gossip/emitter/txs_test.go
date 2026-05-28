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
	"github.com/0xsoniclabs/sonic/gossip/emitter/config"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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
