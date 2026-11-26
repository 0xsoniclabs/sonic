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

package ethapi

import (
	"encoding/binary"
	"errors"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	subsidyTestSender = common.Address{0x51}
	subsidyTestTarget = common.Address{0x42}
	subsidyTestFund   = subsidies.Identifier{0xf0}
)

// newSubsidyTestSimulator returns a simulator on which gas subsidies are enabled.
func newSubsidyTestSimulator() *simulator {
	sim := newSimulator()
	sim.upgrades.GasSubsidies = true
	return sim
}

// newZeroFeeCall returns an unsigned call with no fee, as a simulated call is.
func newZeroFeeCall(chainID *big.Int) *types.Transaction {
	return types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     7,
		Gas:       21000,
		GasFeeCap: big.NewInt(0),
		GasTipCap: big.NewInt(0),
		To:        &subsidyTestTarget,
	})
}

// gasConfigResult is a legacy three-word getGasConfig answer.
func gasConfigResult() []byte {
	result := make([]byte, 3*32)
	binary.BigEndian.PutUint64(result[24:32], 100_000) // chooseFund gas limit
	binary.BigEndian.PutUint64(result[56:64], 100_000) // deductFees gas limit
	binary.BigEndian.PutUint64(result[88:96], 50_000)  // overhead charge
	return result
}

func selectorOf(input []byte) uint32 {
	return binary.BigEndian.Uint32(input[:4])
}

// expectRegistry makes the registry answer getGasConfig and then chooseFund with
// the given fund (zero meaning not covered), and returns the chooseFund input.
func expectRegistry(vmMock *subsidies.MockVirtualMachine, fund subsidies.Identifier) *[]byte {
	var chooseFundInput []byte
	gomock.InOrder(
		vmMock.EXPECT().Call(common.Address{}, registry.GetAddress(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_, _ common.Address, input []byte, _ vm.GasBudget, _ *uint256.Int) ([]byte, vm.GasBudget, error) {
				if selectorOf(input) != registry.GetGasConfigFunctionSelector {
					return nil, vm.GasBudget{}, errors.New("expected getGasConfig")
				}
				return gasConfigResult(), vm.GasBudget{}, nil
			}),
		vmMock.EXPECT().Call(common.Address{}, registry.GetAddress(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_, _ common.Address, input []byte, _ vm.GasBudget, _ *uint256.Int) ([]byte, vm.GasBudget, error) {
				if selectorOf(input) != registry.ChooseFundFunctionSelector {
					return nil, vm.GasBudget{}, errors.New("expected chooseFund")
				}
				chooseFundInput = input
				return fund[:], vm.GasBudget{}, nil
			}),
	)
	return &chooseFundInput
}

func TestIsCoveredBySubsidy_ReportsWhetherTheRegistryCoversTheCall(t *testing.T) {
	tests := map[string]struct {
		fund subsidies.Identifier
		want bool
	}{
		"covered":     {fund: subsidyTestFund, want: true},
		"not covered": {fund: subsidies.Identifier{}, want: false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			vmMock := subsidies.NewMockVirtualMachine(ctrl)
			stateMock := state.NewMockStateDB(ctrl)
			sim := newSubsidyTestSimulator()

			chooseFundInput := expectRegistry(vmMock, test.fund)
			gomock.InOrder(
				stateMock.EXPECT().Snapshot().Return(3),
				stateMock.EXPECT().RevertToSnapshot(3),
			)

			covered, err := sim.isCoveredBySubsidy(vmMock, stateMock, newZeroFeeCall(sim.chainConfig.ChainID), subsidyTestSender, big.NewInt(1e9))
			require.NoError(t, err)
			require.Equal(t, test.want, covered)

			// The registry is asked about the call's sender, not about a sender
			// derived from the placeholder signature.
			require.Equal(t, subsidyTestSender, common.BytesToAddress((*chooseFundInput)[4:36]))
			require.Equal(t, subsidyTestTarget, common.BytesToAddress((*chooseFundInput)[36:68]))
		})
	}
}

func TestIsCoveredBySubsidy_RegistryFailureIsAnError(t *testing.T) {
	ctrl := gomock.NewController(t)
	vmMock := subsidies.NewMockVirtualMachine(ctrl)
	stateMock := state.NewMockStateDB(ctrl)
	sim := newSubsidyTestSimulator()

	injected := errors.New("injected")
	vmMock.EXPECT().Call(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, vm.GasBudget{}, injected)
	gomock.InOrder(
		stateMock.EXPECT().Snapshot().Return(5),
		stateMock.EXPECT().RevertToSnapshot(5),
	)

	covered, err := sim.isCoveredBySubsidy(vmMock, stateMock, newZeroFeeCall(sim.chainConfig.ChainID), subsidyTestSender, big.NewInt(1e9))
	require.ErrorIs(t, err, injected)
	require.ErrorContains(t, err, "failed to query subsidies registry")
	require.False(t, covered)
}

func TestIsCoveredBySubsidy_CallsThatAreNoSponsorshipRequestAreNotQueried(t *testing.T) {
	chainID := newSimTestChainConfig().ChainID
	withFee := types.NewTx(&types.DynamicFeeTx{
		ChainID: chainID, Gas: 21000, GasFeeCap: big.NewInt(1), GasTipCap: big.NewInt(0), To: &subsidyTestTarget,
	})
	creation := types.NewTx(&types.DynamicFeeTx{
		ChainID: chainID, Gas: 21000, GasFeeCap: big.NewInt(0), GasTipCap: big.NewInt(0),
	})

	tests := map[string]struct {
		subsidies bool
		tx        *types.Transaction
		from      common.Address
	}{
		"gas subsidies disabled": {subsidies: false, tx: newZeroFeeCall(chainID), from: subsidyTestSender},
		"sent from zero address": {subsidies: true, tx: newZeroFeeCall(chainID), from: common.Address{}},
		"call pays a fee":        {subsidies: true, tx: withFee, from: subsidyTestSender},
		"contract creation":      {subsidies: true, tx: creation, from: subsidyTestSender},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			// No expectations: neither the registry nor the state may be touched.
			vmMock := subsidies.NewMockVirtualMachine(ctrl)
			stateMock := state.NewMockStateDB(ctrl)
			sim := newSimulator()
			sim.upgrades.GasSubsidies = test.subsidies

			covered, err := sim.isCoveredBySubsidy(vmMock, stateMock, test.tx, test.from, big.NewInt(1e9))
			require.NoError(t, err)
			require.False(t, covered)
		})
	}
}

func TestIsCoveredBySubsidy_CallOfAnotherChainIsAnError(t *testing.T) {
	ctrl := gomock.NewController(t)
	vmMock := subsidies.NewMockVirtualMachine(ctrl)
	stateMock := state.NewMockStateDB(ctrl)
	sim := newSubsidyTestSimulator()

	other := new(big.Int).Add(sim.chainConfig.ChainID, big.NewInt(1))
	covered, err := sim.isCoveredBySubsidy(vmMock, stateMock, newZeroFeeCall(other), subsidyTestSender, big.NewInt(1e9))
	require.ErrorContains(t, err, "failed to judge call as a sponsorship request")
	require.False(t, covered)
}

func TestSimSenderSigner_GivesTheCallSender(t *testing.T) {
	inner := types.LatestSignerForChainID(big.NewInt(1234))
	signer := simSenderSigner{Signer: inner, from: subsidyTestSender}

	sender, err := types.Sender(signer, newZeroFeeCall(big.NewInt(1234)))
	require.NoError(t, err)
	require.Equal(t, subsidyTestSender, sender)
}

func TestSimSenderSigner_EqualsOnlyASignerForTheSameSender(t *testing.T) {
	inner := types.LatestSignerForChainID(big.NewInt(1234))
	signer := simSenderSigner{Signer: inner, from: subsidyTestSender}

	require.True(t, signer.Equal(simSenderSigner{Signer: inner, from: subsidyTestSender}))
	require.False(t, signer.Equal(simSenderSigner{Signer: inner, from: common.Address{0x52}}))
	require.False(t, signer.Equal(inner))
}
