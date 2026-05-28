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

package subsidies

import (
	byte_rand "crypto/rand"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

//go:generate mockgen -source=subsidies_test.go -destination=subsidies_test_mock.go -package=subsidies

func TestIsSponsorshipRequest_DetectsSponsorshipRequest(t *testing.T) {
	require := require.New(t)

	key, err := crypto.GenerateKey()
	require.NoError(err)

	signer := types.LatestSignerForChainID(nil)
	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		To:       &common.Address{},
		Value:    big.NewInt(0),
		Gas:      21000,
		GasPrice: big.NewInt(0),
	})
	require.True(IsSponsorshipRequest(tx))

	tx = types.NewTransaction(0, common.Address{}, nil, 21000, common.Big1, nil)
	require.False(IsSponsorshipRequest(tx))
}

func TestIsSponsorshipRequest_AcceptsNonZeroValue(t *testing.T) {
	require := require.New(t)

	key, err := crypto.GenerateKey()
	require.NoError(err)

	signer := types.LatestSignerForChainID(nil)
	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		To:       &common.Address{},
		Value:    big.NewInt(1), // < non-zero value
		Gas:      21000,
		GasPrice: big.NewInt(0),
	})
	require.True(IsSponsorshipRequest(tx))
}

func TestIsSponsorshipRequest_NilTransaction_IsRejected(t *testing.T) {
	require.False(t, IsSponsorshipRequest(nil))
}

func TestIsSponsorshipRequest_InternalTransaction_IsRejected(t *testing.T) {
	require := require.New(t)
	tx := types.NewTx(&types.LegacyTx{})
	require.True(internaltx.IsInternal(tx))
	require.False(IsSponsorshipRequest(tx))
}

func TestIsSponsorshipRequest_LegacyTransaction_IsRejectedIf(t *testing.T) {
	tests := map[string]func(tx *types.LegacyTx){
		"no recipient": func(tx *types.LegacyTx) {
			tx.To = nil
		},
		"non-zero gas price": func(tx *types.LegacyTx) {
			tx.GasPrice = big.NewInt(1)
		},
	}

	for name, modify := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			key, err := crypto.GenerateKey()
			require.NoError(err)
			signer := types.LatestSignerForChainID(nil)

			data := &types.LegacyTx{
				To: &common.Address{},
			}

			tx := types.MustSignNewTx(key, signer, data)
			require.False(internaltx.IsInternal(tx))
			require.True(IsSponsorshipRequest(tx))

			modify(data)

			tx = types.MustSignNewTx(key, signer, data)
			require.False(internaltx.IsInternal(tx))
			require.False(IsSponsorshipRequest(tx))
		})
	}
}

func TestIsSponsorshipRequest_DynamicFeeTransaction_IsRejectedIf(t *testing.T) {
	tests := map[string]func(tx *types.DynamicFeeTx){
		"no recipient": func(tx *types.DynamicFeeTx) {
			tx.To = nil
		},
		"non-zero fee cap": func(tx *types.DynamicFeeTx) {
			tx.GasFeeCap = big.NewInt(1)
		},
	}

	for name, modify := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			key, err := crypto.GenerateKey()
			require.NoError(err)
			signer := types.NewLondonSigner(big.NewInt(1))

			data := &types.DynamicFeeTx{
				To: &common.Address{},
			}

			tx := types.MustSignNewTx(key, signer, data)
			require.False(internaltx.IsInternal(tx))
			require.True(IsSponsorshipRequest(tx))

			modify(data)

			tx = types.MustSignNewTx(key, signer, data)
			require.False(internaltx.IsInternal(tx))
			require.False(IsSponsorshipRequest(tx))
		})
	}
}

func TestIsCovered_ConsultsSubsidiesRegistry(t *testing.T) {
	// This is an integration test that checks the interaction with the fake
	// subsidies registry contract. It uses a real EVM processor instance on
	// top of a mocked state database with the registry contract code.
	//
	// The test checks various scenarios with different available funds in
	// the registry contract and verifies that IsCovered returns the expected
	// result.

	tests := map[string]struct {
		availableFunds uint64
		expectCovered  bool
	}{
		"no funds available": {
			availableFunds: 0,
			expectCovered:  false,
		},
		"some funds available": {
			availableFunds: 1_000_000_000_000_000,
			expectCovered:  true,
		},
		"too little funds available": {
			availableFunds: 10, // < not enough to cover any fees
			expectCovered:  false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			require := require.New(t)
			ctrl := gomock.NewController(t)
			state := state.NewMockStateDB(ctrl)

			registryAddress := registry.GetAddress()
			code := registry.GetCode()
			hash := crypto.Keccak256Hash(code)

			// Set up a mock state that contains the subsidies registry contract
			// with funds available as specified in the test case.
			any := gomock.Any()
			state.EXPECT().Snapshot().Return(1).AnyTimes()
			state.EXPECT().Exist(registryAddress).Return(true).AnyTimes()
			state.EXPECT().GetCode(registryAddress).Return(code).AnyTimes()
			state.EXPECT().GetCodeHash(registryAddress).Return(hash).AnyTimes()
			state.EXPECT().AddRefund(any).AnyTimes()
			state.EXPECT().SubRefund(any).AnyTimes()
			state.EXPECT().GetRefund().Return(uint64(0)).AnyTimes()
			state.EXPECT().SlotInAccessList(any, any).AnyTimes()
			state.EXPECT().AddSlotToAccessList(any, any).AnyTimes()

			funds := common.Hash{}
			big.NewInt(int64(test.availableFunds)).FillBytes(funds[:])
			state.EXPECT().GetState(any, any).Return(funds).AnyTimes()

			upgrades := opera.GetBrioUpgrades()
			upgrades.GasSubsidies = true
			rules := opera.FakeNetRules(upgrades)

			var updateHeights []opera.UpgradeHeight
			chainConfig := opera.CreateTransientEvmChainConfig(
				rules.NetworkID,
				updateHeights,
				1,
			)

			// Create a transaction that is a valid sponsorship request.
			key, err := crypto.GenerateKey()
			require.NoError(err)
			signer := types.LatestSigner(chainConfig)
			tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
				To:  &common.Address{},
				Gas: 21000,
			})
			require.True(IsSponsorshipRequest(tx))

			// Create an EVM instance with the mocked state and the
			// chain configuration that enables gas subsidies.
			baseFee := big.NewInt(1)
			blockContext := vm.BlockContext{
				BlockNumber: big.NewInt(123),
				BaseFee:     baseFee,
				Transfer: func(_ vm.StateDB, _ common.Address, _ common.Address, amount *uint256.Int, _ *params.Rules) {
					require.Equal(0, amount.Sign())
				},
				Random: &common.Hash{}, // < signals Revision >= Merge
			}

			vmConfig := opera.GetVmConfig(rules)
			vm := vm.NewEVM(blockContext, state, chainConfig, vmConfig)

			result, err := IsCovered(upgrades, vm, signer, tx, baseFee)
			require.NoError(err)
			require.Equal(test.expectCovered, result.IsSponsored())
			if test.expectCovered {
				require.NotEmpty(result.id)
				// These values are hard-coded in the dev-version of the registry.
				require.Equal(uint64(60_000), result.config.gasLimitForDeductFees)
				require.Equal(uint64(80_000), result.config.gasLimitForTrack)
				require.Equal(uint64(210_000), result.config.overheadFundBackedSponsoring)
				require.Equal(uint64(230_000), result.config.overheadNetworkTrackedSponsoring)
			} else {
				require.Empty(result.id)
			}
		})
	}
}

func TestIsCovered_RegistryNotAvailable_ReturnsError(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	state := state.NewMockStateDB(ctrl)

	registryAddress := registry.GetAddress()

	// Set up a mock state not containing the subsidies registry contract.
	state.EXPECT().Snapshot().Return(1).AnyTimes()
	state.EXPECT().Exist(registryAddress).Return(false).AnyTimes()

	upgrades := opera.GetBrioUpgrades()
	upgrades.GasSubsidies = true
	rules := opera.FakeNetRules(upgrades)

	var updateHeights []opera.UpgradeHeight
	chainConfig := opera.CreateTransientEvmChainConfig(
		rules.NetworkID,
		updateHeights,
		1,
	)

	// Create a transaction that is a valid sponsorship request.
	key, err := crypto.GenerateKey()
	require.NoError(err)
	signer := types.LatestSigner(chainConfig)
	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		To:  &common.Address{},
		Gas: 21000,
	})
	require.True(IsSponsorshipRequest(tx))

	// Create an EVM instance with the mocked state and the
	// chain configuration that enables gas subsidies.
	baseFee := big.NewInt(1)
	blockContext := vm.BlockContext{
		BlockNumber: big.NewInt(123),
		BaseFee:     baseFee,
		Transfer: func(_ vm.StateDB, _ common.Address, _ common.Address, amount *uint256.Int, _ *params.Rules) {
			require.Equal(0, amount.Sign())
		},
		Random: &common.Hash{}, // < signals Revision >= Merge
	}

	vmConfig := opera.GetVmConfig(rules)
	vm := vm.NewEVM(blockContext, state, chainConfig, vmConfig)

	_, err = IsCovered(upgrades, vm, signer, tx, baseFee)
	require.ErrorContains(err, "subsidies registry contract not found")
}

func TestIsCovered_GasSubsidiesDisabled_ReturnsNotSponsored(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	vm := NewMockVirtualMachine(ctrl)

	selectedFundId := Identifier{1, 2, 3}
	any := gomock.Any()

	// GetGasConfig is always called first to get the gas config.
	vm.EXPECT().Call(any, any, any, any, any).
		Return(make([]byte, 3*32), uint64(0), nil)

	// ChooseFund is called next to select a fund.
	vm.EXPECT().Call(any, any, any, any, any).
		Return(selectedFundId[:], uint64(0), nil)

	upgrades := opera.Upgrades{}

	key, err := crypto.GenerateKey()
	require.NoError(err)
	signer := types.LatestSignerForChainID(nil)
	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		To: &common.Address{},
	})
	require.True(IsSponsorshipRequest(tx))

	result, err := IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.NoError(err)
	require.False(result.IsSponsored())
	require.Zero(result.id)

	upgrades.GasSubsidies = true
	result, err = IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.NoError(err)
	require.True(result.IsSponsored())
	require.Equal(selectedFundId, result.id)
}

func TestIsCovered_NotASponsorshipRequest_ReturnsNotSponsored(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	vm := NewMockVirtualMachine(ctrl)

	any := gomock.Any()
	selectedFundId := Identifier{1, 2, 3}
	vm.EXPECT().Call(any, any, any, any, any).
		Return(make([]byte, 3*32), uint64(0), nil)
	vm.EXPECT().Call(any, any, any, any, any).
		Return(selectedFundId[:], uint64(0), nil)

	upgrades := opera.Upgrades{
		GasSubsidies: true,
	}

	key, err := crypto.GenerateKey()
	require.NoError(err)
	signer := types.LatestSignerForChainID(nil)

	// Non-Sponsorship request (no recipient) is rejected.
	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{})
	require.False(IsSponsorshipRequest(tx))
	result, err := IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.NoError(err)
	require.False(result.IsSponsored())

	// Sponsorship request is accepted.
	tx = types.MustSignNewTx(key, signer, &types.LegacyTx{
		To: &common.Address{},
	})
	require.True(IsSponsorshipRequest(tx))
	result, err = IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.NoError(err)
	require.True(result.IsSponsored())
	require.Equal(selectedFundId, result.id)
}

func TestIsCovered_NotCoveredByFunds_ReturnsNotSponsored(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	vm := NewMockVirtualMachine(ctrl)

	upgrades := opera.Upgrades{
		GasSubsidies: true,
	}

	key, err := crypto.GenerateKey()
	require.NoError(err)
	signer := types.LatestSignerForChainID(nil)

	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		To: &common.Address{},
	})

	// If the query returns the 0-fund ID, IsCovered returns mode 0 (not covered).
	any := gomock.Any()
	selectedFundId := Identifier{}
	vm.EXPECT().Call(any, any, any, any, any).Return(make([]byte, 3*32), uint64(0), nil)
	vm.EXPECT().Call(any, any, any, any, any).Return(selectedFundId[:], uint64(0), nil)
	result, err := IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.NoError(err)
	require.False(result.IsSponsored())
	require.Zero(result.id)

	// If the query returns a non-zero fund ID, IsCovered returns mode 1 (fund-backed).
	selectedFundId = Identifier{1, 2, 3}
	vm.EXPECT().Call(any, any, any, any, any).Return(make([]byte, 3*32), uint64(0), nil)
	vm.EXPECT().Call(any, any, any, any, any).Return(selectedFundId[:], uint64(0), nil)
	result, err = IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.NoError(err)
	require.True(result.IsSponsored())
	require.Equal(selectedFundId, result.id)
}

func TestIsCovered_SenderReaderFails_ReturnsError(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	signer := NewMocksigner(ctrl)

	upgrades := opera.Upgrades{
		GasSubsidies: true,
	}

	tx := types.NewTx(&types.LegacyTx{
		To: &common.Address{},
		V:  big.NewInt(1), // < non-zero signature: transaction is not internal
	})

	issue := fmt.Errorf("injected issue")
	signer.EXPECT().Sender(tx).Return(common.Address{}, issue)

	_, err := IsCovered(upgrades, nil, signer, tx, big.NewInt(1))
	require.ErrorContains(err, "failed to derive sender")
	require.ErrorIs(err, issue)
}

func TestIsCovered_createChooseFundInputFails_ReturnsError(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	vm := NewMockVirtualMachine(ctrl)
	signer := NewMocksigner(ctrl)

	upgrades := opera.Upgrades{
		GasSubsidies: true,
	}

	tx := types.NewTx(&types.LegacyTx{
		To:  &common.Address{},
		Gas: 21000,
		V:   big.NewInt(1), // < non-zero signature: transaction is not internal
	})

	signer.EXPECT().Sender(tx).Return(common.Address{}, nil)

	// Allow the getGasConfig EVM call to succeed.
	any := gomock.Any()
	vm.EXPECT().Call(any, any, any, any, any).
		Return(make([]byte, 3*32), uint64(0), nil)

	// A huge base fee causes createChooseFundInput to fail.
	baseFee := new(big.Int).Lsh(big.NewInt(1), 256) // 2^256
	_, err := IsCovered(upgrades, vm, signer, tx, baseFee)
	require.ErrorContains(err, "fee does not fit into 32 bytes")
}

func TestIsCovered_EvmCallFails_ReturnsError(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	vm := NewMockVirtualMachine(ctrl)

	upgrades := opera.Upgrades{
		GasSubsidies: true,
	}

	key, err := crypto.GenerateKey()
	require.NoError(err)
	signer := types.LatestSignerForChainID(nil)

	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		To: &common.Address{},
	})

	// If the getGasConfig EVM returns an issue, IsCovered returns that issue.
	any := gomock.Any()
	issue := fmt.Errorf("injected getGasConfig issue")
	vm.EXPECT().Call(any, any, any, any, any).Return(nil, uint64(0), issue)
	_, err = IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.ErrorContains(err, "EVM call failed")
	require.ErrorIs(err, issue)

	// If the chooseFund EVM call fails, IsCovered returns an error.
	issue = fmt.Errorf("injected chooseFund issue")
	vm.EXPECT().Call(any, any, any, any, any).Return(make([]byte, 3*32), uint64(0), nil)
	vm.EXPECT().Call(any, any, any, any, any).Return(nil, uint64(0), issue)
	_, err = IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.ErrorContains(err, "EVM call failed")
	require.ErrorIs(err, issue)
}

func TestIsCovered_EmptyResultFromChooseFund_ReportsMissingContract(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	vm := NewMockVirtualMachine(ctrl)

	upgrades := opera.Upgrades{
		GasSubsidies: true,
	}

	key, err := crypto.GenerateKey()
	require.NoError(err)
	signer := types.LatestSignerForChainID(nil)

	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		To: &common.Address{},
	})

	// If the EVM returns no data, IsCovered returns an error.
	any := gomock.Any()
	vm.EXPECT().Call(any, any, any, any, any).Return(make([]byte, 3*32), uint64(0), nil)
	vm.EXPECT().Call(any, any, any, any, any).Return(nil, uint64(0), nil)
	_, err = IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.ErrorContains(err, "subsidies registry contract not found")
}

func TestIsCovered_InvalidReturnFromEvm_ReturnsError(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	vm := NewMockVirtualMachine(ctrl)

	upgrades := opera.Upgrades{
		GasSubsidies: true,
	}

	key, err := crypto.GenerateKey()
	require.NoError(err)
	signer := types.LatestSignerForChainID(nil)

	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		To: &common.Address{},
	})

	// If the EVM returns invalid data, IsCovered returns an error.
	any := gomock.Any()
	vm.EXPECT().Call(any, any, any, any, any).Return(make([]byte, 3*32), uint64(0), nil)
	vm.EXPECT().Call(any, any, any, any, any).Return([]byte{0x01}, uint64(0), nil)
	_, err = IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.ErrorContains(err, "failed to parse result of subsidies registry call")
}

func TestIsCovered_CallsChooseFundWithFeeConsideringMaxOverheadCharge(t *testing.T) {
	tests := map[string]struct {
		fundBackedOverhead     uint64
		networkTrackedOverhead uint64
	}{
		"overhead for fund-backed sponsoring is higher": {
			fundBackedOverhead:     210_000,
			networkTrackedOverhead: 80_000,
		},
		"overhead for network-tracked sponsoring is higher": {
			fundBackedOverhead:     80_000,
			networkTrackedOverhead: 210_000,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)
			vm := NewMockVirtualMachine(ctrl)

			upgrades := opera.Upgrades{
				GasSubsidies: true,
			}

			key, err := crypto.GenerateKey()
			require.NoError(err)
			signer := types.LatestSignerForChainID(nil)

			tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
				To: &common.Address{},
			})

			// The gas config returned by the EVM has the overhead values specified in the test case.
			gasConfig := make([]byte, 5*32)
			binary.BigEndian.PutUint64(gasConfig[3*32+24:], test.fundBackedOverhead)
			binary.BigEndian.PutUint64(gasConfig[4*32+24:], test.networkTrackedOverhead)
			any := gomock.Any()
			vm.EXPECT().Call(any, any, any, any, any).Return(gasConfig, uint64(0), nil)

			// The chooseFund call should be made with a fee that considers the maximum of the two overheads.
			expectedFee := uint256.NewInt(test.fundBackedOverhead)
			if test.networkTrackedOverhead > test.fundBackedOverhead {
				expectedFee = uint256.NewInt(test.networkTrackedOverhead)
			}
			vm.EXPECT().Call(any, any, any, any, any).Return(make([]byte, 2*32), uint64(0), nil).
				Do(func(_ common.Address, _ common.Address, input []byte, _ uint64, _ *uint256.Int) {
					// The fee is the 6th argument of the call.
					feeBytes := input[4+5*32 : 4+6*32]
					fee := new(big.Int).SetBytes(feeBytes)
					require.Equal(expectedFee.ToBig(), fee)
				})

			_, err = IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
			require.NoError(err)
		})
	}
}

func TestIsCovered_ReportsTheCorrectMode(t *testing.T) {
	for _, mode := range allModes() {
		t.Run(fmt.Sprintf("mode=%d", mode), func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)
			vm := NewMockVirtualMachine(ctrl)

			upgrades := opera.Upgrades{
				GasSubsidies: true,
			}

			key, err := crypto.GenerateKey()
			require.NoError(err)
			signer := types.LatestSignerForChainID(nil)

			tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
				To: &common.Address{},
			})

			// The gas config call returns just zeros for this test.
			gasConfigResult := make([]byte, 5*32)
			any := gomock.Any()
			vm.EXPECT().Call(any, any, any, any, any).Return(gasConfigResult, uint64(0), nil)

			// The chooseFund call returns the mode specified.
			chooseFundResult := make([]byte, 2*32)
			binary.BigEndian.PutUint64(chooseFundResult[24:], uint64(mode))
			vm.EXPECT().Call(any, any, any, any, any).Return(chooseFundResult, uint64(0), nil)

			result, err := IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
			require.NoError(err)
			require.Equal(mode, result.mode)
		})
	}
}

func TestIsCovered_ReportsThePayloadId(t *testing.T) {
	for _, id := range []Identifier{{}, {1, 2, 3}, {0x12, 31: 0xff}} {
		t.Run(fmt.Sprintf("id=%v", id), func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)
			vm := NewMockVirtualMachine(ctrl)

			upgrades := opera.Upgrades{
				GasSubsidies: true,
			}

			key, err := crypto.GenerateKey()
			require.NoError(err)
			signer := types.LatestSignerForChainID(nil)

			tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
				To: &common.Address{},
			})

			// The gas config call returns just zeros for this test.
			gasConfigResult := make([]byte, 5*32)
			any := gomock.Any()
			vm.EXPECT().Call(any, any, any, any, any).Return(gasConfigResult, uint64(0), nil)

			// The chooseFund call returns the mode specified.
			chooseFundResult := make([]byte, 2*32)
			binary.BigEndian.PutUint64(chooseFundResult[24:], uint64(sponsorshipModeFundBacked))
			copy(chooseFundResult[32:], id[:])
			vm.EXPECT().Call(any, any, any, any, any).Return(chooseFundResult, uint64(0), nil)

			result, err := IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
			require.NoError(err)
			require.Equal(id, result.id)
		})
	}
}

func TestIsCovered_ReportsTheGasPriceConfig(t *testing.T) {
	configs := []gasConfig{
		{
			gasLimitForChooseFund:            40_000,
			gasLimitForDeductFees:            60_000,
			gasLimitForTrack:                 80_000,
			overheadFundBackedSponsoring:     210_000,
			overheadNetworkTrackedSponsoring: 230_000,
		},
		{
			gasLimitForChooseFund:            1,
			gasLimitForDeductFees:            2,
			gasLimitForTrack:                 3,
			overheadFundBackedSponsoring:     4,
			overheadNetworkTrackedSponsoring: 5,
		},
	}

	for _, config := range configs {
		t.Run(fmt.Sprintf("config=%v", config), func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)
			vm := NewMockVirtualMachine(ctrl)

			upgrades := opera.Upgrades{
				GasSubsidies: true,
			}

			key, err := crypto.GenerateKey()
			require.NoError(err)
			signer := types.LatestSignerForChainID(nil)

			tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
				To: &common.Address{},
			})

			// The gas config call returns just zeros for this test.
			gasConfigResult := make([]byte, 5*32)
			binary.BigEndian.PutUint64(gasConfigResult[0*32+24:], config.gasLimitForChooseFund)
			binary.BigEndian.PutUint64(gasConfigResult[1*32+24:], config.gasLimitForDeductFees)
			binary.BigEndian.PutUint64(gasConfigResult[2*32+24:], config.gasLimitForTrack)
			binary.BigEndian.PutUint64(gasConfigResult[3*32+24:], config.overheadFundBackedSponsoring)
			binary.BigEndian.PutUint64(gasConfigResult[4*32+24:], config.overheadNetworkTrackedSponsoring)
			any := gomock.Any()
			vm.EXPECT().Call(any, any, any, any, any).Return(gasConfigResult, uint64(0), nil)

			// The chooseFund call returns the mode specified.
			chooseFundResult := make([]byte, 2*32)
			vm.EXPECT().Call(any, any, any, any, any).Return(chooseFundResult, uint64(0), nil)

			result, err := IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
			require.NoError(err)
			require.Equal(config, result.config)
		})
	}
}

func TestSponsorship_IsSponsored_ReturnsTrueIfModeIsNotZero(t *testing.T) {
	test := map[string]struct {
		mode     sponsorshipMode
		expected bool
	}{
		"not covered": {
			mode:     sponsorshipModeNotCovered,
			expected: false,
		},
		"fund-backed sponsoring": {
			mode:     sponsorshipModeFundBacked,
			expected: true,
		},
		"network sponsoring": {
			mode:     sponsorshipModeNetwork,
			expected: true,
		},
		"network-tracked sponsoring": {
			mode:     sponsorshipModeNetworkWithTracking,
			expected: true,
		},
	}

	for name, test := range test {
		t.Run(name, func(t *testing.T) {
			s := Sponsorship{mode: test.mode}
			require.Equal(t, test.expected, s.IsSponsored())
		})
	}
}

func TestSponsorship_Overhead_ReturnsConfiguredOverhead(t *testing.T) {
	config := gasConfig{
		overheadFundBackedSponsoring:     210_000,
		overheadNetworkTrackedSponsoring: 230_000,
	}

	test := map[string]struct {
		mode     sponsorshipMode
		expected Overhead
	}{
		"no-sponsoring": {
			mode:     sponsorshipModeNotCovered,
			expected: Overhead{Gas: 0, Size: 0},
		},
		"fund-backed sponsoring": {
			mode: sponsorshipModeFundBacked,
			expected: Overhead{
				Gas:  config.overheadFundBackedSponsoring,
				Size: rlpEncodedRegistryCallTxSizeInBytes,
			},
		},
		"network sponsoring": {
			mode:     sponsorshipModeNetwork,
			expected: Overhead{Gas: 0, Size: 0},
		},
		"network-tracked sponsoring": {
			mode: sponsorshipModeNetworkWithTracking,
			expected: Overhead{
				Gas:  config.overheadNetworkTrackedSponsoring,
				Size: rlpEncodedRegistryCallTxSizeInBytes,
			},
		},
	}

	for name, test := range test {
		t.Run(name, func(t *testing.T) {
			s := Sponsorship{mode: test.mode, config: config}
			require.Equal(t, test.expected, s.Overhead())
		})
	}
}

func TestOverhead_SizeIsUpperBoundForRlpEncodedTx(t *testing.T) {
	for _, mode := range allModes() {
		t.Run(fmt.Sprintf("mode=%d", mode), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			nonceSource := NewMockNonceSource(ctrl)
			nonceSource.EXPECT().GetNonce(common.Address{}).Return(uint64(math.MaxUint64)).AnyTimes()
			id := Identifier{}
			fillRandom(t, id[:])
			gasConfig := gasConfig{
				gasLimitForDeductFees:            math.MaxUint64,
				gasLimitForTrack:                 math.MaxUint64,
				overheadFundBackedSponsoring:     math.MaxUint64,
				overheadNetworkTrackedSponsoring: math.MaxUint64,
			}
			gasPrice := big.NewInt(math.MaxInt64)

			sponsorship := Sponsorship{
				mode:   mode,
				id:     id,
				config: gasConfig,
			}

			postTxs, err := sponsorship.GetPostTransactions(nonceSource, math.MaxUint64, gasPrice)
			require.NoError(t, err)

			sumTxSize := uint64(0)
			for _, tx := range postTxs {
				sumTxSize += tx.Size()
			}
			require.LessOrEqual(t, sumTxSize, sponsorship.Overhead().Size)
		})
	}
}

func TestGetPostTransactions_ProducesCorrectInternalTransaction(t *testing.T) {
	nonces := []uint64{
		0, 1, 42, 1000,
	}
	fundIds := []Identifier{
		{}, {1, 2, 3}, {0x12, 31: 0xff},
	}
	gasLimits := []int{
		0, 15000, 1_000_000,
	}
	overheadGasLimit := []int{
		0, 125000, 1_000_000,
	}
	gasUsed := []int{
		0, 21000, 100000, 1_000_000,
	}
	gasPrice := []int{
		0, 1, 1e12,
	}
	for _, mode := range allModes() {
		for _, nonce := range nonces {
			for _, fundId := range fundIds {
				for _, gasLimitDeduct := range gasLimits {
					for _, gasLimitTrack := range gasLimits {
						for _, overheadFundBacked := range overheadGasLimit {
							for _, overheadNetworkTracked := range overheadGasLimit {
								for _, gasUsed := range gasUsed {
									for _, price := range gasPrice {
										t.Run(fmt.Sprintf("mode=%d/nonce=%d/fundId=%v/gasLimitDeduct=%d/gasLimitTracked=%d/overheadFundBacked=%d/overheadNetworkTracked=%d/gasUsed=%d/price=%d", mode, nonce, fundId, gasLimitDeduct, gasLimitTrack, overheadFundBacked, overheadNetworkTracked, gasUsed, price), func(t *testing.T) {
											require := require.New(t)
											ctrl := gomock.NewController(t)
											nonceSource := NewMockNonceSource(ctrl)
											nonceSource.EXPECT().GetNonce(common.Address{}).Return(nonce).AnyTimes()

											config := gasConfig{
												gasLimitForDeductFees:            uint64(gasLimitDeduct),
												gasLimitForTrack:                 uint64(gasLimitTrack),
												overheadFundBackedSponsoring:     uint64(overheadFundBacked),
												overheadNetworkTrackedSponsoring: uint64(overheadNetworkTracked),
											}

											gasPrice := big.NewInt(int64(price))
											postTxs, err := (Sponsorship{mode: mode, id: [32]byte(fundId), config: config}).
												GetPostTransactions(nonceSource, uint64(gasUsed), gasPrice)
											require.NoError(err)

											if mode == sponsorshipModeNotCovered || mode == sponsorshipModeNetwork {
												require.Empty(postTxs)
												return
											}

											gasLimit := gasLimitDeduct
											if mode == sponsorshipModeNetworkWithTracking {
												gasLimit = gasLimitTrack
											}

											require.Len(postTxs, 1)
											tx := postTxs[0]

											require.True(internaltx.IsInternal(tx))
											require.Equal(nonce, tx.Nonce())
											require.NotNil(tx.To())
											require.Equal(registry.GetAddress(), *tx.To())
											require.Equal(common.Big0, tx.Value())
											require.Equal(uint64(gasLimit), tx.Gas())
											require.Equal(common.Big0, tx.GasPrice())
											require.Equal(common.Big0, tx.GasFeeCap())
											require.Equal(common.Big0, tx.GasTipCap())

											selector := registry.DeductFeesFunctionSelector
											if mode == sponsorshipModeNetworkWithTracking {
												selector = registry.TrackFunctionSelector
											}

											overhead := config.overheadFundBackedSponsoring
											if mode == sponsorshipModeNetworkWithTracking {
												overhead = config.overheadNetworkTrackedSponsoring
											}

											got := tx.Data()
											fee := uint256.NewInt(uint64(price * (gasUsed + int(overhead))))
											want := createRegistryCallInput(uint32(selector), [32]byte(fundId), *fee)
											require.Equal(want, got)
										})
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

func TestGetPostTransactions_FeeOverflows_ReturnsError(t *testing.T) {
	modesWithPostTxs := []sponsorshipMode{
		sponsorshipModeFundBacked,
		sponsorshipModeNetworkWithTracking,
	}
	for _, mode := range modesWithPostTxs {
		t.Run(fmt.Sprintf("mode=%d", mode), func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)
			nonceSource := NewMockNonceSource(ctrl)
			nonceSource.EXPECT().GetNonce(common.Address{}).Return(uint64(0))

			fundId := Identifier{}
			config := gasConfig{
				overheadFundBackedSponsoring:     1,
				overheadNetworkTrackedSponsoring: 1,
			}
			gasUsed := uint64(0)
			gasPrice := new(big.Int).Lsh(big.NewInt(1), 256) // 2^256
			_, err := (Sponsorship{mode: mode, id: [32]byte(fundId), config: config}).
				GetPostTransactions(nonceSource, gasUsed, gasPrice)
			require.ErrorContains(err, "fee calculation overflow")
		})
	}
}

func TestIsFeeChargeTransaction_ValidFeeChargeTransaction_ReturnsTrue(t *testing.T) {
	addr := registry.GetAddress()
	tx := types.NewTx(&types.LegacyTx{To: &addr, Data: createRegistryCallInput(registry.DeductFeesFunctionSelector, [32]byte{}, uint256.Int{})})
	require.True(t, internaltx.IsInternal(tx))
	require.True(t, IsFeeChargeTransaction(tx))
}

func TestIsFeeChargeTransaction_NilTransaction_ReturnsFalse(t *testing.T) {
	require.False(t, IsFeeChargeTransaction(nil))
}

func TestIsFeeChargeTransaction_NonInternalTransaction_ReturnsFalse(t *testing.T) {
	addr := registry.GetAddress()
	tx := types.NewTx(&types.LegacyTx{
		To:   &addr,
		Data: createRegistryCallInput(registry.DeductFeesFunctionSelector, [32]byte{}, uint256.Int{}),
		V:    big.NewInt(1),
	})
	require.False(t, internaltx.IsInternal(tx))
	require.False(t, IsFeeChargeTransaction(tx))
}

func TestIsFeeChargeTransaction_NilRecipient_ReturnsFalse(t *testing.T) {
	tx := types.NewTx(&types.LegacyTx{Data: createRegistryCallInput(registry.DeductFeesFunctionSelector, [32]byte{}, uint256.Int{})})
	require.True(t, internaltx.IsInternal(tx))
	require.False(t, IsFeeChargeTransaction(tx))
}

func TestIsFeeChargeTransaction_WrongRecipient_ReturnsFalse(t *testing.T) {
	addr := common.Address{0x42}
	tx := types.NewTx(&types.LegacyTx{To: &addr, Data: createRegistryCallInput(registry.DeductFeesFunctionSelector, [32]byte{}, uint256.Int{})})
	require.True(t, internaltx.IsInternal(tx))
	require.False(t, IsFeeChargeTransaction(tx))
}

func TestIsFeeChargeTransaction_WrongDataLength_ReturnsFalse(t *testing.T) {
	addr := registry.GetAddress()
	tests := map[string]int{
		"too short": 4 + 2*32 - 1,
		"too long":  4 + 2*32 + 1,
	}
	for name, length := range tests {
		t.Run(name, func(t *testing.T) {
			tx := types.NewTx(&types.LegacyTx{To: &addr, Data: make([]byte, length)})
			require.True(t, internaltx.IsInternal(tx))
			require.False(t, IsFeeChargeTransaction(tx))
		})
	}
}

func TestIsFeeChargeTransaction_WrongSelector_ReturnsFalse(t *testing.T) {
	data := createRegistryCallInput(registry.DeductFeesFunctionSelector, [32]byte{}, uint256.Int{})
	binary.BigEndian.PutUint32(data, registry.DeductFeesFunctionSelector+1)
	addr := registry.GetAddress()
	tx := types.NewTx(&types.LegacyTx{To: &addr, Data: data})
	require.True(t, internaltx.IsInternal(tx))
	require.False(t, IsFeeChargeTransaction(tx))
}

func TestIsTrackTransaction_ValidTrackTransaction_ReturnsTrue(t *testing.T) {
	addr := registry.GetAddress()
	data := createRegistryCallInput(registry.TrackFunctionSelector, [32]byte{}, uint256.Int{})
	tx := types.NewTx(&types.LegacyTx{To: &addr, Data: data})
	require.True(t, internaltx.IsInternal(tx))
	require.True(t, IsTrackTransaction(tx))
}

func TestIsTrackTransaction_FeeChargeTransaction_ReturnsFalse(t *testing.T) {
	addr := registry.GetAddress()
	data := createRegistryCallInput(registry.DeductFeesFunctionSelector, [32]byte{}, uint256.Int{})
	tx := types.NewTx(&types.LegacyTx{To: &addr, Data: data})
	require.True(t, internaltx.IsInternal(tx))
	require.False(t, IsTrackTransaction(tx))
}

func TestParseFeeChargeAmount_ValidInput_ReturnsFee(t *testing.T) {
	tests := map[string]*uint256.Int{
		"zero fee":  new(uint256.Int),
		"small fee": uint256.NewInt(1_000),
		"large fee": uint256.NewInt(0).Lsh(uint256.NewInt(1), 200),
	}
	for name, fee := range tests {
		t.Run(name, func(t *testing.T) {
			addr := registry.GetAddress()
			tx := types.NewTx(&types.LegacyTx{
				To:   &addr,
				Data: createRegistryCallInput(registry.DeductFeesFunctionSelector, [32]byte{}, *fee),
			})
			require.True(t, IsFeeChargeTransaction(tx))
			got, err := ParseFeeChargeAmount(tx)
			require.NoError(t, err)
			require.Equal(t, fee, got)
		})
	}
}

func TestParseFeeChargeAmount_NotAFeeChargeTransaction_ReturnsError(t *testing.T) {
	tests := map[string]*types.Transaction{
		"nil transaction": nil,
		"non-internal transaction": types.NewTx(&types.LegacyTx{
			To:   &common.Address{},
			Data: createRegistryCallInput(registry.DeductFeesFunctionSelector, [32]byte{}, uint256.Int{}),
			V:    big.NewInt(1),
		}),
	}
	for name, tx := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := ParseFeeChargeAmount(tx)
			require.ErrorContains(t, err, "transaction is not a fee charge transaction")
		})
	}
}

func TestParseTrackAmount_ValidInput_ReturnsFee(t *testing.T) {
	tests := map[string]*uint256.Int{
		"zero fee":  new(uint256.Int),
		"small fee": uint256.NewInt(1_000),
		"large fee": uint256.NewInt(0).Lsh(uint256.NewInt(1), 200),
	}
	for name, fee := range tests {
		t.Run(name, func(t *testing.T) {
			addr := registry.GetAddress()
			tx := types.NewTx(&types.LegacyTx{
				To:   &addr,
				Data: createRegistryCallInput(registry.TrackFunctionSelector, [32]byte{}, *fee),
			})
			require.True(t, IsTrackTransaction(tx))
			got, err := ParseTrackAmount(tx)
			require.NoError(t, err)
			require.Equal(t, fee, got)
		})
	}
}

func TestParseTrackAmount_NotATrackTransaction_ReturnsError(t *testing.T) {
	tests := map[string]*types.Transaction{
		"nil transaction": nil,
		"non-internal transaction": types.NewTx(&types.LegacyTx{
			To:   &common.Address{},
			Data: createRegistryCallInput(registry.TrackFunctionSelector, [32]byte{}, uint256.Int{}),
			V:    big.NewInt(1),
		}),
	}
	for name, tx := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := ParseTrackAmount(tx)
			require.ErrorContains(t, err, "transaction is not a track transaction")
		})
	}
}

func TestAllModes_ReturnsAllDefinedModes(t *testing.T) {
	modes := allModes()
	require.Len(t, modes, 4)
	require.Contains(t, modes, sponsorshipModeNotCovered)
	require.Contains(t, modes, sponsorshipModeFundBacked)
	require.Contains(t, modes, sponsorshipModeNetwork)
	require.Contains(t, modes, sponsorshipModeNetworkWithTracking)
}

func TestGetGasConfig_ValidLegacyFormat_ReturnsExpectedConfig(t *testing.T) {
	// The legacy format covers three values in the config:
	// - gasLimitForChooseFund
	// - gasLimitForDeductFees
	// - overhead
	// This test checks whether those legacy values are parsed correctly.
	values := []uint64{0, 1, 42, 1000, 1_000_000, math.MaxUint64}

	cases := []gasConfig{}
	for _, choose := range values {
		for _, deduct := range values {
			for _, overhead := range values {
				cases = append(cases, gasConfig{
					gasLimitForChooseFund:            choose,
					gasLimitForDeductFees:            deduct,
					gasLimitForTrack:                 0, // < ignored in 3-field format
					overheadFundBackedSponsoring:     overhead,
					overheadNetworkTrackedSponsoring: overhead,
				})
			}
		}
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("choose=%d/deduct=%d/overhead=%d", c.gasLimitForChooseFund, c.gasLimitForDeductFees, c.overheadFundBackedSponsoring), func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)
			vm := NewMockVirtualMachine(ctrl)

			any := gomock.Any()
			caller := common.Address{}
			target := registry.GetAddress()
			input := make([]byte, 4) // < function selector only
			binary.BigEndian.PutUint32(input, registry.GetGasConfigFunctionSelector)
			gas := uint64(registry.GasLimitForGetGasConfig)

			result := make([]byte, 3*32)
			binary.BigEndian.PutUint64(result[32*0+24:32*0+32], c.gasLimitForChooseFund)
			binary.BigEndian.PutUint64(result[32*1+24:32*1+32], c.gasLimitForDeductFees)
			binary.BigEndian.PutUint64(result[32*2+24:32*2+32], c.overheadFundBackedSponsoring)

			vm.EXPECT().Call(caller, target, input, gas, any).
				Return(result, uint64(0), nil)

			config, err := getGasConfig(vm)
			require.NoError(err)

			require.Equal(c.gasLimitForChooseFund, config.gasLimitForChooseFund)
			require.Equal(c.gasLimitForDeductFees, config.gasLimitForDeductFees)
			require.Equal(c.overheadFundBackedSponsoring, config.overheadFundBackedSponsoring)
			require.Equal(c.overheadNetworkTrackedSponsoring, config.overheadNetworkTrackedSponsoring)
			require.Zero(config.gasLimitForTrack)
		})
	}
}

func TestGetGasConfig_ValidConfiguration_ReturnsExpectedConfig(t *testing.T) {
	values := []uint64{0, 1, 42, 1000, 1_000_000, math.MaxUint64}

	cases := []gasConfig{}
	for _, choose := range values {
		for _, deduct := range values {
			for _, track := range values {
				for _, fundBackedOverhead := range values {
					for _, networkTrackedOverhead := range values {
						cases = append(cases, gasConfig{
							gasLimitForChooseFund:            choose,
							gasLimitForDeductFees:            deduct,
							gasLimitForTrack:                 track,
							overheadFundBackedSponsoring:     fundBackedOverhead,
							overheadNetworkTrackedSponsoring: networkTrackedOverhead,
						})
					}
				}
			}
		}
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("choose=%d/deduct=%d/track=%d/fundBacked=%d/networkTracked=%d", c.gasLimitForChooseFund, c.gasLimitForDeductFees, c.gasLimitForTrack, c.overheadFundBackedSponsoring, c.overheadNetworkTrackedSponsoring), func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)
			vm := NewMockVirtualMachine(ctrl)

			any := gomock.Any()
			caller := common.Address{}
			target := registry.GetAddress()
			input := make([]byte, 4) // < function selector only
			binary.BigEndian.PutUint32(input, registry.GetGasConfigFunctionSelector)
			gas := uint64(registry.GasLimitForGetGasConfig)

			result := make([]byte, 5*32)
			binary.BigEndian.PutUint64(result[32*0+24:32*0+32], c.gasLimitForChooseFund)
			binary.BigEndian.PutUint64(result[32*1+24:32*1+32], c.gasLimitForDeductFees)
			binary.BigEndian.PutUint64(result[32*2+24:32*2+32], c.gasLimitForTrack)
			binary.BigEndian.PutUint64(result[32*3+24:32*3+32], c.overheadFundBackedSponsoring)
			binary.BigEndian.PutUint64(result[32*4+24:32*4+32], c.overheadNetworkTrackedSponsoring)

			vm.EXPECT().Call(caller, target, input, gas, any).
				Return(result, uint64(0), nil)

			config, err := getGasConfig(vm)
			require.NoError(err)

			require.Equal(c.gasLimitForChooseFund, config.gasLimitForChooseFund)
			require.Equal(c.gasLimitForDeductFees, config.gasLimitForDeductFees)
			require.Equal(c.gasLimitForTrack, config.gasLimitForTrack)
			require.Equal(c.overheadFundBackedSponsoring, config.overheadFundBackedSponsoring)
			require.Equal(c.overheadNetworkTrackedSponsoring, config.overheadNetworkTrackedSponsoring)
		})
	}
}

func TestGetGasConfig_VmFailing_ReturnsVmError(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	vm := NewMockVirtualMachine(ctrl)

	any := gomock.Any()
	issue := fmt.Errorf("injected issue")
	vm.EXPECT().Call(any, any, any, any, any).
		Return(nil, uint64(0), issue)

	_, err := getGasConfig(vm)
	require.ErrorIs(err, issue)
}

func TestGetGasConfig_InvalidVmResult_ReturnsIssue(t *testing.T) {

	tests := map[string]struct {
		result []byte
		issue  string
	}{
		"no contract": {
			result: nil,
			issue:  "subsidies registry contract not found",
		},
		"too short for 3 words": {
			result: make([]byte, 3*32-1),
			issue:  "invalid result length",
		},
		"too long for 3 words": {
			result: make([]byte, 3*32+1),
			issue:  "invalid result length",
		},
		"too short for 5 words": {
			result: make([]byte, 5*32-1),
			issue:  "invalid result length",
		},
		"too long for 5 words": {
			result: make([]byte, 5*32+1),
			issue:  "invalid result length",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)
			vm := NewMockVirtualMachine(ctrl)

			any := gomock.Any()
			vm.EXPECT().Call(any, any, any, any, any).
				Return(test.result, uint64(0), nil)

			_, err := getGasConfig(vm)
			require.ErrorContains(err, test.issue)
		})
	}
}

func TestGetGasConfig_GasLimitOverflow_ReportsOverflow(t *testing.T) {

	t.Run("3-field legacy format", func(t *testing.T) {
		inRange := func(i int) bool {
			return 24 <= i && i < 32 || 56 <= i && i < 64 || 88 <= i && i < 96
		}

		for i := range 96 {
			if !inRange(i) {
				t.Run(fmt.Sprintf("index=%d", i), func(t *testing.T) {
					require := require.New(t)
					ctrl := gomock.NewController(t)
					vm := NewMockVirtualMachine(ctrl)

					result := make([]byte, 3*32)
					result[i] = 1

					any := gomock.Any()
					vm.EXPECT().Call(any, any, any, any, any).
						Return(result, uint64(0), nil)

					_, err := getGasConfig(vm)
					require.ErrorContains(err, "values do not fit into uint64")
				})
			}
		}
	})

	t.Run("5-field format", func(t *testing.T) {
		inRange := func(i int) bool {
			return 24 <= i && i < 32 ||
				56 <= i && i < 64 ||
				88 <= i && i < 96 ||
				120 <= i && i < 128 ||
				152 <= i && i < 160
		}

		for i := range 160 {
			if !inRange(i) {
				t.Run(fmt.Sprintf("index=%d", i), func(t *testing.T) {
					require := require.New(t)
					ctrl := gomock.NewController(t)
					vm := NewMockVirtualMachine(ctrl)

					result := make([]byte, 5*32)
					result[i] = 1

					any := gomock.Any()
					vm.EXPECT().Call(any, any, any, any, any).
						Return(result, uint64(0), nil)

					_, err := getGasConfig(vm)
					require.ErrorContains(err, "values do not fit into uint64")
				})
			}
		}
	})
}

func TestCreateChooseFundInput_ValidInputs_ProducesCorrectInputData(t *testing.T) {
	require := require.New(t)

	sender := common.Address{}
	receiver := common.Address{}
	data := make([]byte, 12)

	fillRandom(t, sender[:])
	fillRandom(t, receiver[:])
	fillRandom(t, data)

	valueData := [32]byte{}
	fillRandom(t, valueData[:])
	value := new(big.Int).SetBytes(valueData[:])

	nonce := rand.Uint64()

	tx := types.NewTransaction(nonce, receiver, value, 21000, common.Big0, data)

	feeData := [32]byte{}
	fillRandom(t, feeData[:])
	fee := new(big.Int).SetBytes(feeData[:])

	input, err := createChooseFundInput(sender, tx, fee)
	require.NoError(err)

	// Check the length of the input data.
	// - 4 bytes function selector
	// - 6 * 32 bytes for parameters
	// - 2 * 32 bytes for dynamic bytes parameter (length + one 32-byte chunk)
	require.Equal(4+6*32+2*32, len(input))

	// Function Selector
	require.Equal(
		binary.BigEndian.Uint32(input[0:4]),
		uint32(registry.ChooseFundFunctionSelector),
	)
	input = input[4:]

	// Sender Address
	parameter := [32]byte{}
	copy(parameter[12:32], sender[:])
	require.Equal(parameter[:], input[:32])
	input = input[32:]

	// Receiver Address
	parameter = [32]byte{}
	copy(parameter[12:32], receiver[:])
	require.Equal(parameter[:], input[:32])
	input = input[32:]

	// Value
	require.Equal(input[:32], valueData[:])
	input = input[32:]

	// Nonce
	parameter = [32]byte{}
	binary.BigEndian.PutUint64(parameter[24:32], nonce)
	require.Equal(parameter[:], input[:32])
	input = input[32:]

	// Offset for call data
	parameter = [32]byte{31: 6 * 32}
	require.Equal(parameter[:], input[:32])
	input = input[32:]

	// Fee
	parameter = [32]byte{}
	fee.FillBytes(parameter[:])
	require.Equal(parameter[:], input[:32])
	input = input[32:]

	// Call data length
	parameter = [32]byte{}
	binary.BigEndian.PutUint64(parameter[24:32], uint64(len(data)))
	require.Equal(parameter[:], input[:32])
	input = input[32:]

	// Call data (one 32-byte chunk)
	parameter = [32]byte{}
	copy(parameter[:], data)
	require.Equal(parameter[:], input[:32])
}

func TestCreateChooseFundInput_NilTransaction_ReturnsError(t *testing.T) {
	require := require.New(t)
	_, err := createChooseFundInput(common.Address{}, nil, nil)
	require.ErrorContains(err, "invalid transaction")
}

func TestCreateChooseFundInput_FeeOverflow_ReturnsError(t *testing.T) {
	require := require.New(t)
	tx := types.NewTx(&types.LegacyTx{})

	tooHighFee := new(big.Int).Lsh(big.NewInt(1), 256) // 2^256
	_, err := createChooseFundInput(common.Address{}, tx, tooHighFee)
	require.ErrorContains(err, "fee does not fit into 32 bytes")

	justAcceptableFee := new(big.Int).Sub(tooHighFee, big.NewInt(1))
	_, err = createChooseFundInput(common.Address{}, tx, justAcceptableFee)
	require.NoError(err)
}

func TestCreateChooseFundInput_TransactionWithoutReceiver_ProducesAZeroedReceiver(t *testing.T) {
	require := require.New(t)

	sender := common.Address{}
	fillRandom(t, sender[:])
	nonce := rand.Uint64()

	tx := types.NewContractCreation(nonce, common.Big0, 21000, common.Big0, nil)

	input, err := createChooseFundInput(sender, tx, common.Big0)
	require.NoError(err)

	target := input[4+32 : 4+2*32] // < receiver address
	require.Equal(make([]byte, 32), target)
}

func TestCreateChooseFundInput_LongCallData_CallDataIsEncodedCorrectly(t *testing.T) {
	for n := range 1024 {
		t.Run(fmt.Sprintf("data length %d", n), func(t *testing.T) {
			require := require.New(t)

			sender := common.Address{}
			receiver := common.Address{}
			data := make([]byte, n)

			fillRandom(t, sender[:])
			fillRandom(t, receiver[:])
			fillRandom(t, data)
			nonce := rand.Uint64()

			tx := types.NewTransaction(nonce, receiver, common.Big0, 21000, common.Big0, data)

			feeData := [32]byte{}
			fillRandom(t, feeData[:])
			fee := new(big.Int).SetBytes(feeData[:])

			input, err := createChooseFundInput(sender, tx, fee)
			require.NoError(err)

			numChunks := (len(data) + 31) / 32

			// Check the length of the input data.
			require.Equal(4+6*32+(1+numChunks)*32, len(input))

			// Offset for call data
			parameter := [32]byte{31: 6 * 32}
			input = input[4+4*32:] // skip function selector + first 4 parameters
			require.Equal(parameter[:], input[:32])
			input = input[32:]

			// Call data length
			parameter = [32]byte{}
			binary.BigEndian.PutUint64(parameter[24:32], uint64(len(data)))
			input = input[32:] // skip the fee parameter
			require.Equal(parameter[:], input[:32])
			input = input[32:]

			// Call data (N 32-byte chunks, padded with zeros)
			padded := make([]byte, numChunks*32)
			copy(padded, data)
			require.Equal(padded, input)
		})
	}
}

func TestParseChooseFundResult_ValidLegacyInputs_ParsesCorrectly(t *testing.T) {
	tests := map[string]struct {
		fundId   Identifier
		wantMode sponsorshipMode
	}{
		"empty fund": {
			fundId:   Identifier{}, // < interpreted as not covered
			wantMode: sponsorshipModeNotCovered,
		},
		"non-empty fund": {
			fundId:   Identifier{1, 2, 3},
			wantMode: sponsorshipModeFundBacked,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			input := test.fundId[:]
			mode, id, err := parseChooseFundResult(input)
			require.NoError(err)
			require.Equal(test.wantMode, mode)
			wantedId := test.fundId
			if test.wantMode == sponsorshipModeNotCovered {
				wantedId = Identifier{}
			}
			require.Equal(wantedId, id)
		})
	}
}

func TestParseChooseFundResult_ValidInputs_ParsesCorrectly(t *testing.T) {
	tests := map[string]struct {
		mode   sponsorshipMode
		fundId Identifier
	}{
		"not covered with empty fund": {
			mode:   sponsorshipModeNotCovered,
			fundId: Identifier{},
		},
		"not covered with non-empty fund": {
			mode:   sponsorshipModeNotCovered,
			fundId: Identifier{1, 2, 3}, // < ignored, but should not cause an error
		},
		"fund-backed": {
			mode:   sponsorshipModeFundBacked,
			fundId: Identifier{1, 2, 3},
		},
		"network-sponsored": {
			mode:   sponsorshipModeNetwork,
			fundId: Identifier{},
		},
		"network-sponsored with non-empty track Id": {
			mode:   sponsorshipModeNetwork,
			fundId: Identifier{1, 2, 3}, // < ignored but should not cause an error
		},
		"network-sponsored with tracking": {
			mode:   sponsorshipModeNetworkWithTracking,
			fundId: Identifier{1, 2, 3},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			input := make([]byte, 2*32)
			input[31] = byte(test.mode)
			copy(input[32:64], test.fundId[:])
			mode, id, err := parseChooseFundResult(input)
			require.NoError(err)
			require.Equal(test.mode, mode)
			wantedId := test.fundId
			if test.mode == sponsorshipModeNotCovered || test.mode == sponsorshipModeNetwork {
				wantedId = Identifier{}
			}
			require.Equal(wantedId, id)
		})
	}
}

func TestParseChooseFundResult_InvalidInputs_ReturnsError(t *testing.T) {
	tests := map[string]struct {
		input []byte
		issue string
	}{
		"missing input": {
			input: nil,
			issue: "invalid result length",
		},
		"too short for 1 word": {
			input: make([]byte, 32-1),
			issue: "invalid result length",
		},
		"too long for 1 word": {
			input: make([]byte, 32+1),
			issue: "invalid result length",
		},
		"too short for 2 words": {
			input: make([]byte, 2*32-1),
			issue: "invalid result length",
		},
		"too long for 2 words": {
			input: make([]byte, 2*32+1),
			issue: "invalid result length",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			_, _, err := parseChooseFundResult(test.input)
			require.ErrorContains(err, test.issue)
		})
	}
}

func TestParseChooseFundResult_InvalidMode_ReturnsError(t *testing.T) {
	tests := map[string]struct {
		mode  [32]byte
		issue string
	}{
		"mode 4": {
			mode:  [32]byte{31: 4},
			issue: "unknown sponsorship mode",
		},
		"mode 255": {
			mode:  [32]byte{31: 255},
			issue: "unknown sponsorship mode",
		},
		"mode out of range": {
			mode:  [32]byte{0: 1},
			issue: "mode value out of range",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			input := make([]byte, 2*32)
			copy(input[0:32], test.mode[:])
			_, _, err := parseChooseFundResult(input)
			require.ErrorContains(err, test.issue)
		})
	}
}

func TestCreateRegistryCallInput_CombinesSelectorIdentifierAndFee(t *testing.T) {
	selectors := []uint32{12, registry.DeductFeesFunctionSelector, registry.TrackFunctionSelector}

	randomId := Identifier{}
	fillRandom(t, randomId[:])
	ids := []Identifier{{}, {1, 2, 3}, randomId}

	randomFee := [32]byte{}
	fillRandom(t, randomFee[:])
	fees := []*uint256.Int{
		uint256.NewInt(0),
		uint256.NewInt(1),
		uint256.NewInt(0).SetBytes(randomFee[:]),
	}

	for _, selector := range selectors {
		for _, id := range ids {
			for _, fee := range fees {
				t.Run(fmt.Sprintf("selector=%v/id=%v/fee=%s", selector, id, fee.String()), func(t *testing.T) {
					require := require.New(t)
					input := createRegistryCallInput(selector, [32]byte(id), *fee)
					require.Equal(4+2*32, len(input))

					// Function Selector
					require.Equal(
						binary.BigEndian.Uint32(input[0:4]),
						selector,
					)
					input = input[4:]

					// ID
					require.Equal(id[:], input[:32])
					input = input[32:]

					// Fee
					got := new(uint256.Int).SetBytes(input[:32])
					require.Equal(got, fee)
				})
			}
		}
	}
}

func fillRandom(t *testing.T, b []byte) {
	_, err := byte_rand.Read(b)
	require.NoError(t, err)
}

// signer is an alias for types.Signer to allow mocking it.
type signer interface {
	types.Signer
}

// Added to avoid unused warning
var _ signer
