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

			state := state.NewMockStateDB(ctrl)
			state.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).AnyTimes()

			external := NewMockExternal(ctrl)
			external.EXPECT().GetRules().Return(rules).AnyTimes()
			external.EXPECT().GetLatestBlockIndex().Return(idx.Block(100)).AnyTimes()
			external.EXPECT().StateDB().Return(state).AnyTimes()

			emitter := &Emitter{
				world: World{External: external},
			}

			signer := types.LatestSignerForChainID(big.NewInt(int64(rules.NetworkID)))
			tx := bundle.NewBuilder(signer).SetEarliest(50).SetLatest(150).Build()

			_, _, err := bundle.ValidateEnvelope(signer, tx)
			require.NoError(err)

			allBundlesRunnable := func(evmcore.ChainState, *types.Transaction) evmcore.BundleState {
				return evmcore.BundleState{Executable: true}
			}

			require.Equal(bundlesEnabled, emitter.isRunnableBundleTxInternal(tx, allBundlesRunnable))
		})
	}
}

func Test_Emitter_isValidBundleTx_RejectsInvalidBundle(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))
	tests := map[string]*types.Transaction{
		"not a bundle": types.NewTx(&types.LegacyTx{}),
		"invalid bundle data": types.NewTx(&types.LegacyTx{
			To:   &bundle.BundleProcessor,
			Data: []byte{0x01, 0x02, 0x03},
		}),
		"bundle with out-of-range block numbers": bundle.NewBuilder(signer).
			SetEarliest(150).
			SetLatest(250).
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

			require.False(emitter.isValidBundleTx(tx))
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

			state := state.NewMockStateDB(ctrl)
			state.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).Return(processed).AnyTimes()

			external := NewMockExternal(ctrl)
			external.EXPECT().GetRules().Return(rules).AnyTimes()
			external.EXPECT().GetLatestBlockIndex().Return(idx.Block(100)).AnyTimes()
			external.EXPECT().StateDB().Return(state).AnyTimes()

			emitter := &Emitter{
				world: World{External: external},
			}

			signer := types.LatestSignerForChainID(big.NewInt(1))
			tx := bundle.NewBuilder(signer).SetEarliest(50).SetLatest(150).Build()

			_, _, err := bundle.ValidateEnvelope(signer, tx)
			require.NoError(t, err)

			getBundleState := func(evmcore.ChainState, *types.Transaction) evmcore.BundleState {
				return evmcore.BundleState{Executable: true}
			}

			require.Equal(t, !processed, emitter.isRunnableBundleTxInternal(tx, getBundleState))
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

func Test_preCheckStateAdapter_ForwardsStateDBRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	stateDB := state.NewMockStateDB(ctrl)
	stateDB.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).AnyTimes()

	external := NewMockExternal(ctrl)
	external.EXPECT().StateDB().Return(stateDB)

	adapter := &preCheckChainStateAdapter{external: external}
	returnedStateDB := adapter.StateDB()

	require.Same(t, stateDB, returnedStateDB)
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

	adapter := &preCheckChainStateAdapter{external: external}
	got := adapter.GetEvmChainConfig(blockHeight)

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
