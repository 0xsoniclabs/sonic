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

package evmcore

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

// static assert interface implementation
var _ subsidiesChecker = &SubsidiesIntegrationImplementation{}

func TestSubsidiesIntegration_SubsidiesCheckerCanExecuteContracts(t *testing.T) {
	ctrl := gomock.NewController(t)

	rules := opera.Rules{
		Upgrades: opera.Upgrades{
			GasSubsidies: true,
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

	checker := newSubsidiesChecker(rules, chain, state, signer)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		Nonce:    0,
		GasPrice: big.NewInt(0),
		Gas:      21000,
		To:       &common.Address{},
		Value:    big.NewInt(0),
		Data:     []byte{},
	})

	// This test does not have any expectations on the result of the contract execution,
	// just that it was executed without error.
	checker.isSponsored(tx)
}

func TestSubsidiesIntegration_SubsidiesCheckerReturnsFalseIfContractIsNotDeployed(t *testing.T) {
	ctrl := gomock.NewController(t)

	rules := opera.Rules{
		Upgrades: opera.Upgrades{
			GasSubsidies: true,
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

	checker := newSubsidiesChecker(rules, chain, state, signer)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		Nonce:    0,
		GasPrice: big.NewInt(0),
		Gas:      21000,
		To:       &common.Address{},
		Value:    big.NewInt(0),
		Data:     []byte{},
	})

	res := checker.isSponsored(tx)
	require.False(t, res)
}

// makeHappyStateDb creates a mock StateDB and StateReader that behaves "happily" for the purposes of testing
func makeHappyStateDb(
	ctrl *gomock.Controller,
	chainConfig *params.ChainConfig,
) (*MockStateReader, *state.MockStateDB) {

	any := gomock.Any()

	chain := NewMockStateReader(ctrl)
	chain.EXPECT().CurrentBlock().Return(&EvmBlock{
		EvmHeader: EvmHeader{
			Number:     big.NewInt(1),
			PrevRandao: common.Hash{1}, // revision >= merge
		},
	}).AnyTimes()
	chain.EXPECT().GetCurrentBaseFee().Return(big.NewInt(1)).AnyTimes()
	chain.EXPECT().Config().Return(chainConfig).AnyTimes()

	state := state.NewMockStateDB(ctrl)
	state.EXPECT().GetNonce(any).Return(uint64(0)).AnyTimes()
	state.EXPECT().GetBalance(any).Return(uint256.NewInt(1e18)).AnyTimes()
	state.EXPECT().GetCodeHash(any).Return(types.EmptyCodeHash).AnyTimes()

	state.EXPECT().Snapshot().MinTimes(1)
	state.EXPECT().Exist(any).Return(true).AnyTimes()
	state.EXPECT().SubBalance(any, any, any).AnyTimes()
	state.EXPECT().AddBalance(any, any, any).AnyTimes()
	state.EXPECT().AddRefund(any).AnyTimes().AnyTimes()
	state.EXPECT().GetState(any, any).Return(common.Hash{}).AnyTimes()
	state.EXPECT().GetRefund().Return(uint64(0)).AnyTimes()
	state.EXPECT().SubRefund(any).Return().AnyTimes()
	state.EXPECT().RevertToSnapshot(any).AnyTimes()
	return chain, state
}
