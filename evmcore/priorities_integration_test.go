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

package evmcore

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestPrioritiesIntegration_PrioritizedCheckerCanExecuteContracts(t *testing.T) {
	ctrl := gomock.NewController(t)

	rules := opera.Rules{
		Upgrades: opera.Upgrades{
			TransactionPriorities: true,
		},
	}

	chainConfig := opera.CreateTransientEvmChainConfig(1,
		[]opera.UpgradeHeight{
			{Upgrades: rules.Upgrades, Height: 0},
		}, 1)

	chain, state := makeHappyStateDb(ctrl, chainConfig)
	// Expect contract to be executed
	any := gomock.Any()
	state.EXPECT().GetCode(registry.GetAddress()).Return(registry.GetCode()).MinTimes(1)
	state.EXPECT().SlotInAccessList(registry.GetAddress(), any).MinTimes(1)
	state.EXPECT().AddSlotToAccessList(registry.GetAddress(), any).MinTimes(1)

	signer := types.LatestSignerForChainID(big.NewInt(1))

	isPrioritized := newPrioritizedChecker(rules, chain, state, signer)

	// This test does not have any expectations on the result of the contract
	// execution, just that it was executed without error.
	isPrioritized(makePriorityCheckTx(t, signer))
}

func TestPrioritiesIntegration_PrioritizedCheckerReturnsFalseIfContractIsNotDeployed(t *testing.T) {
	ctrl := gomock.NewController(t)

	rules := opera.Rules{
		Upgrades: opera.Upgrades{
			TransactionPriorities: true,
		},
	}

	chainConfig := opera.CreateTransientEvmChainConfig(1,
		[]opera.UpgradeHeight{
			{Upgrades: rules.Upgrades, Height: 0},
		}, 1)

	chain, state := makeHappyStateDb(ctrl, chainConfig)

	// Contract execution fails when the contract is not deployed
	state.EXPECT().GetCode(registry.GetAddress()).Return(nil)

	signer := types.LatestSignerForChainID(big.NewInt(1))

	isPrioritized := newPrioritizedChecker(rules, chain, state, signer)

	require.False(t, isPrioritized(makePriorityCheckTx(t, signer)))
}

func TestPrioritiesIntegration_PrioritizedCheckerReturnsFalseWhileTheFeatureIsDisabled(t *testing.T) {
	ctrl := gomock.NewController(t)

	rules := opera.Rules{} // transaction priorities disabled
	chainConfig := opera.CreateTransientEvmChainConfig(1,
		[]opera.UpgradeHeight{
			{Upgrades: rules.Upgrades, Height: 0},
		}, 1)

	chain := NewMockStateReader(ctrl)
	chain.EXPECT().CurrentBlock().Return(&EvmBlock{
		EvmHeader: EvmHeader{
			Number:     big.NewInt(1),
			PrevRandao: common.Hash{1}, // revision >= merge
		},
	})
	chain.EXPECT().CurrentConfig().Return(chainConfig)

	signer := types.LatestSignerForChainID(big.NewInt(1))

	// The registry is never queried, so no state access is expected.
	isPrioritized := newPrioritizedChecker(rules, chain, nil, signer)

	require.False(t, isPrioritized(makePriorityCheckTx(t, signer)))
}

// makePriorityCheckTx signs a minimal transaction to be classified.
func makePriorityCheckTx(t *testing.T, signer types.Signer) *types.Transaction {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	return types.MustSignNewTx(key, signer, &types.LegacyTx{
		Nonce:    0,
		GasPrice: big.NewInt(0),
		Gas:      21000,
		To:       &common.Address{},
		Value:    big.NewInt(0),
		Data:     []byte{},
	})
}
