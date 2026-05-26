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
	"github.com/ethereum/go-ethereum/rlp"
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

func TestFundId_String_PrintsAsHexValue(t *testing.T) {
	require := require.New(t)
	id := FundId{}
	require.Equal("0x0000000000000000000000000000000000000000000000000000000000000000", id.String())
	id = FundId{0x01, 0x02, 0x03, 0xef}
	require.Equal("0x010203ef00000000000000000000000000000000000000000000000000000000", id.String())
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
			require.Equal(test.expectCovered, result.mode != sponsorshipModeNotCovered)
			if test.expectCovered {
				require.NotEmpty(result.fundId)
				// These values are hard-coded in the dev-version of the registry.
				require.Equal(uint64(60_000), result.config.DeductFeesGasCost)
				require.Equal(uint64(210_000), result.config.SponsorshipOverheadGasCost)
			} else {
				require.Empty(result.fundId)
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

func TestIsCovered_GasSubsidiesDisabled_ReturnsFalse(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	vm := NewMockVirtualMachine(ctrl)

	selectedFundId := FundId{1, 2, 3}
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
	require.Equal(sponsorshipModeNotCovered, result.mode)
	require.Empty(result.fundId)

	upgrades.GasSubsidies = true
	result, err = IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.NoError(err)
	require.Equal(sponsorshipModeFundBacked, result.mode)
	require.Equal(selectedFundId, result.fundId)
}

func TestIsCovered_NotASponsorshipRequest_ReturnsFalse(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	vm := NewMockVirtualMachine(ctrl)

	any := gomock.Any()
	selectedFundId := FundId{1, 2, 3}
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
	require.Equal(sponsorshipModeNotCovered, result.mode)
	require.Empty(result.fundId)

	// Sponsorship request is accepted.
	tx = types.MustSignNewTx(key, signer, &types.LegacyTx{
		To: &common.Address{},
	})
	require.True(IsSponsorshipRequest(tx))
	result, err = IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.NoError(err)
	require.Equal(sponsorshipModeFundBacked, result.mode)
	require.Equal(selectedFundId, result.fundId)
}

func TestIsCovered_NotCoveredByFunds_ReturnsFalse(t *testing.T) {
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
	selectedFundId := FundId{}
	vm.EXPECT().Call(any, any, any, any, any).Return(make([]byte, 3*32), uint64(0), nil)
	vm.EXPECT().Call(any, any, any, any, any).Return(selectedFundId[:], uint64(0), nil)
	result, err := IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.NoError(err)
	require.Equal(sponsorshipModeNotCovered, result.mode)
	require.Empty(result.fundId)

	// If the query returns a non-zero fund ID, IsCovered returns mode 1 (fund-backed).
	selectedFundId = FundId{1, 2, 3}
	vm.EXPECT().Call(any, any, any, any, any).Return(make([]byte, 3*32), uint64(0), nil)
	vm.EXPECT().Call(any, any, any, any, any).Return(selectedFundId[:], uint64(0), nil)
	result, err = IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.NoError(err)
	require.Equal(sponsorshipModeFundBacked, result.mode)
	require.Equal(selectedFundId, result.fundId)
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

func TestOverhead_FundBacked_SizeIsUpperBoundForRlpEncodedTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	nonceSource := NewMockNonceSource(ctrl)
	nonceSource.EXPECT().GetNonce(common.Address{}).Return(uint64(math.MaxUint64))
	id := FundId(make([]byte, 32))
	gasConfig := GasConfig{SponsorshipOverheadGasCost: math.MaxUint64, DeductFeesGasCost: math.MaxUint64}
	gasPrice := big.NewInt(math.MaxInt64)

	postTxs, err := (Sponsorship{mode: sponsorshipModeFundBacked, fundId: FundId(id), config: gasConfig}).
		GetPostTransactions(nonceSource, math.MaxUint64, gasPrice)
	require.NoError(t, err)
	require.Len(t, postTxs, 1)
	data, err := rlp.EncodeToBytes(postTxs[0])
	require.NoError(t, err)
	s := Sponsorship{mode: sponsorshipModeFundBacked, config: gasConfig}
	require.Less(t, len(data), int(s.Overhead().Size))
}

func TestGetPostTransactions_FundBacked_ProducesCorrectInternalTransaction(t *testing.T) {
	nonces := []uint64{
		0, 1, 42, 1000,
	}
	fundIds := []FundId{
		{}, {1, 2, 3}, {0x12, 31: 0xff},
	}
	deductFeeGasLimit := []int{
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
	for _, nonce := range nonces {
		for _, fundId := range fundIds {
			for _, deductFees := range deductFeeGasLimit {
				for _, overhead := range overheadGasLimit {
					for _, gas := range gasUsed {
						for _, price := range gasPrice {
							t.Run(fmt.Sprintf("nonce=%d/fundId=%v/gas=%d/price=%d", nonce, fundId, gas, price), func(t *testing.T) {
								require := require.New(t)
								ctrl := gomock.NewController(t)
								nonceSource := NewMockNonceSource(ctrl)
								nonceSource.EXPECT().GetNonce(common.Address{}).Return(nonce)

								config := GasConfig{
									DeductFeesGasCost:          uint64(deductFees),
									SponsorshipOverheadGasCost: uint64(overhead),
								}

								gasPrice := big.NewInt(int64(price))
								postTxs, err := (Sponsorship{mode: sponsorshipModeFundBacked, fundId: fundId, config: config}).
									GetPostTransactions(nonceSource, uint64(gas), gasPrice)
								require.NoError(err)
								require.Len(postTxs, 1)
								tx := postTxs[0]

								require.True(internaltx.IsInternal(tx))
								require.Equal(nonce, tx.Nonce())
								require.NotNil(tx.To)
								require.Equal(registry.GetAddress(), *tx.To())
								require.Equal(common.Big0, tx.Value())
								require.Equal(uint64(deductFees), tx.Gas())
								require.Equal(common.Big0, tx.GasPrice())
								require.Equal(common.Big0, tx.GasFeeCap())
								require.Equal(common.Big0, tx.GasTipCap())

								got := tx.Data()
								fee := uint256.NewInt(uint64(price * (gas + int(config.SponsorshipOverheadGasCost))))
								want := createRegistryCallInput(registry.DeductFeesFunctionSelector, [32]byte(fundId), *fee)
								require.Equal(want, got)
							})
						}
					}
				}
			}
		}
	}
}

func TestGetPostTransactions_FundBacked_FeeOverflows_ReturnsError(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	nonceSource := NewMockNonceSource(ctrl)
	nonceSource.EXPECT().GetNonce(common.Address{}).Return(uint64(0))

	fundId := FundId{}
	config := GasConfig{DeductFeesGasCost: 1, SponsorshipOverheadGasCost: 1}
	gasUsed := uint64(0)
	gasPrice := new(big.Int).Lsh(big.NewInt(1), 256) // 2^256
	_, err := (Sponsorship{mode: sponsorshipModeFundBacked, fundId: fundId, config: config}).
		GetPostTransactions(nonceSource, gasUsed, gasPrice)
	require.ErrorContains(err, "fee calculation overflow")
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

func TestGetGasConfig_ValidSetup_ReturnsExpectedConfig(t *testing.T) {
	cases := []gasConfig{}
	values := []uint64{0, 1, 42, 1000, 15000, 125000, 1_000_000, math.MaxUint64}

	for _, choose := range values {
		for _, deduct := range values {
			for _, overhead := range values {
				cases = append(cases, gasConfig{
					chooseFundGasLimit: choose,
					deductFeesGasLimit: deduct,
					overheadToCharge:   overhead,
				})
			}
		}
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("choose=%d/deduct=%d/overhead=%d", c.chooseFundGasLimit, c.deductFeesGasLimit, c.overheadToCharge), func(t *testing.T) {
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
			binary.BigEndian.PutUint64(result[32*0+24:32*0+32], c.chooseFundGasLimit)
			binary.BigEndian.PutUint64(result[32*1+24:32*1+32], c.deductFeesGasLimit)
			binary.BigEndian.PutUint64(result[32*2+24:32*2+32], c.overheadToCharge)

			vm.EXPECT().Call(caller, target, input, gas, any).
				Return(result, uint64(0), nil)

			config, err := getGasConfig(vm)
			require.NoError(err)

			require.Equal(c.chooseFundGasLimit, config.chooseFundGasLimit)
			require.Equal(c.deductFeesGasLimit, config.deductFeesGasLimit)
			require.Equal(c.overheadToCharge, config.overheadToCharge)
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
		"too short": {
			result: make([]byte, 3*32-1),
			issue:  "invalid result length",
		},
		"between 96 and 128": {
			result: make([]byte, 3*32+1),
			issue:  "invalid result length",
		},
		"too long": {
			result: make([]byte, 4*32+1),
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

func TestParseIsCoveredResult_ValidInputs_ParsesCorrectly(t *testing.T) {
	tests := map[string]struct {
		wantMode sponsorshipMode
		fundId   FundId
	}{
		"empty fund": {
			wantMode: sponsorshipModeNotCovered,
			fundId:   FundId{},
		},
		"non-empty fund": {
			wantMode: sponsorshipModeFundBacked,
			fundId:   FundId{1, 2, 3},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			input := test.fundId[:]
			mode, fundId, _, err := parseChooseFundResult(input)
			require.NoError(err)
			require.Equal(test.wantMode, mode)
			wantedFund := test.fundId
			if test.wantMode == sponsorshipModeNotCovered {
				wantedFund = FundId{}
			}
			require.Equal(wantedFund, fundId)
		})
	}
}

func TestParseIsCoveredResult_InvalidInputs_ReturnsError(t *testing.T) {
	tests := map[string]struct {
		input []byte
		issue string
	}{
		"missing input": {
			input: nil,
			issue: "invalid result length",
		},
		"too short": {
			input: make([]byte, 31),
			issue: "invalid result length",
		},
		"between 32 and 64": {
			input: make([]byte, 48),
			issue: "invalid result length",
		},
		"too long": {
			input: make([]byte, 65),
			issue: "invalid result length",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			_, _, _, err := parseChooseFundResult(test.input)
			require.ErrorContains(err, test.issue)
		})
	}
}

func TestCreateDeductFeeInput_CombinesFundIdWithFee(t *testing.T) {
	randomId := FundId{}
	fillRandom(t, randomId[:])
	ids := []FundId{{}, {1, 2, 3}, randomId}

	randomFee := [32]byte{}
	fillRandom(t, randomFee[:])
	fees := []*uint256.Int{
		uint256.NewInt(0),
		uint256.NewInt(1),
		uint256.NewInt(0).SetBytes(randomFee[:]),
	}

	for _, id := range ids {
		for _, fee := range fees {
			t.Run(fmt.Sprintf("id=%v/fee=%s", id, fee.String()), func(t *testing.T) {
				require := require.New(t)
				input := createRegistryCallInput(registry.DeductFeesFunctionSelector, [32]byte(id), *fee)
				require.Equal(4+2*32, len(input))

				// Function Selector
				require.Equal(
					binary.BigEndian.Uint32(input[0:4]),
					uint32(registry.DeductFeesFunctionSelector),
				)
				input = input[4:]

				// Fund ID
				require.Equal(id[:], input[:32])
				input = input[32:]

				// Fee
				got := new(uint256.Int).SetBytes(input[:32])
				require.Equal(got, fee)
			})
		}
	}
}

// --- New tests for modes 2, 3, and backward compatibility ---

func makeTestTx(t *testing.T) (*types.Transaction, types.Signer) {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	signer := types.LatestSignerForChainID(nil)
	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{To: &common.Address{}, Gas: 21000})
	return tx, signer
}

func makeGasConfigResponse(chooseFund, deductFees, overhead uint64) []byte {
	result := make([]byte, 3*32)
	binary.BigEndian.PutUint64(result[32*0+24:32*0+32], chooseFund)
	binary.BigEndian.PutUint64(result[32*1+24:32*1+32], deductFees)
	binary.BigEndian.PutUint64(result[32*2+24:32*2+32], overhead)
	return result
}

func makeGasConfigResponseWithTrack(chooseFund, deductFees, overhead, trackGasCost uint64) []byte {
	result := make([]byte, 4*32)
	binary.BigEndian.PutUint64(result[32*0+24:32*0+32], chooseFund)
	binary.BigEndian.PutUint64(result[32*1+24:32*1+32], deductFees)
	binary.BigEndian.PutUint64(result[32*2+24:32*2+32], overhead)
	binary.BigEndian.PutUint64(result[32*3+24:32*3+32], trackGasCost)
	return result
}

func makeChooseFundResponse(mode uint8, payload [32]byte) []byte {
	result := make([]byte, 2*32)
	result[31] = mode
	copy(result[32:64], payload[:])
	return result
}

// TestIsCovered_Mode2_ReturnsNetworkSponsored verifies that a 64-byte chooseFund
// response with mode=2 is correctly interpreted as network-sponsored.
func TestIsCovered_Mode2_ReturnsNetworkSponsored(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	vm := NewMockVirtualMachine(ctrl)

	upgrades := opera.Upgrades{GasSubsidies: true}
	tx, signer := makeTestTx(t)
	any := gomock.Any()

	vm.EXPECT().Call(any, any, any, any, any).Return(makeGasConfigResponse(100_000, 60_000, 210_000), uint64(0), nil)
	vm.EXPECT().Call(any, any, any, any, any).Return(makeChooseFundResponse(2, [32]byte{}), uint64(0), nil)

	result, err := IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.NoError(err)
	require.Equal(sponsorshipModeNetwork, result.mode)
	require.Equal(FundId{}, result.fundId)
	require.Equal(TrackingId{}, result.trackingId)
}

// TestIsCovered_Mode3_ReturnsNetworkSponsoredWithTracking verifies that a 64-byte
// chooseFund response with mode=3 is correctly interpreted, and that the tracking
// ID payload is stored in TrackingId (not FundId).
func TestIsCovered_Mode3_ReturnsNetworkSponsoredWithTracking(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	vm := NewMockVirtualMachine(ctrl)

	upgrades := opera.Upgrades{GasSubsidies: true}
	tx, signer := makeTestTx(t)
	any := gomock.Any()

	trackingId := TrackingId{0xde, 0xad, 0xbe, 0xef}
	vm.EXPECT().Call(any, any, any, any, any).Return(makeGasConfigResponse(100_000, 60_000, 210_000), uint64(0), nil)
	vm.EXPECT().Call(any, any, any, any, any).Return(makeChooseFundResponse(3, [32]byte(trackingId)), uint64(0), nil)

	result, err := IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.NoError(err)
	require.Equal(sponsorshipModeNetworkWithTracking, result.mode)
	require.Equal(FundId{}, result.fundId)
	require.Equal(trackingId, result.trackingId)
}

// TestIsCovered_LegacyRegistry_32ByteChooseFund_MapsToMode1 verifies that a
// 32-byte chooseFund response with a non-zero fundId is mapped to mode 1.
func TestIsCovered_LegacyRegistry_32ByteChooseFund_MapsToMode1(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	vm := NewMockVirtualMachine(ctrl)

	upgrades := opera.Upgrades{GasSubsidies: true}
	tx, signer := makeTestTx(t)
	any := gomock.Any()

	fundId := FundId{0x01, 0x02, 0x03}
	vm.EXPECT().Call(any, any, any, any, any).Return(makeGasConfigResponse(100_000, 60_000, 210_000), uint64(0), nil)
	vm.EXPECT().Call(any, any, any, any, any).Return(fundId[:], uint64(0), nil)

	result, err := IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.NoError(err)
	require.Equal(sponsorshipModeFundBacked, result.mode)
	require.Equal(fundId, result.fundId)
	require.Equal(TrackingId{}, result.trackingId)
}

// TestIsCovered_LegacyRegistry_32ByteChooseFund_ZeroFundId_MapsToMode0 verifies
// that a 32-byte chooseFund response with a zero fundId maps to mode 0 (not covered).
func TestIsCovered_LegacyRegistry_32ByteChooseFund_ZeroFundId_MapsToMode0(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	vm := NewMockVirtualMachine(ctrl)

	upgrades := opera.Upgrades{GasSubsidies: true}
	tx, signer := makeTestTx(t)
	any := gomock.Any()

	vm.EXPECT().Call(any, any, any, any, any).Return(makeGasConfigResponse(100_000, 60_000, 210_000), uint64(0), nil)
	vm.EXPECT().Call(any, any, any, any, any).Return(make([]byte, 32), uint64(0), nil) // zero fundId

	result, err := IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.NoError(err)
	require.Equal(sponsorshipModeNotCovered, result.mode)
}

// TestIsCovered_LegacyGetGasConfig_96Bytes_DefaultsTrackGasCostToZero verifies
// that a legacy 96-byte getGasConfig response sets TrackGasCost to zero.
func TestIsCovered_LegacyGetGasConfig_96Bytes_DefaultsTrackGasCostToZero(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	vm := NewMockVirtualMachine(ctrl)

	upgrades := opera.Upgrades{GasSubsidies: true}
	tx, signer := makeTestTx(t)
	any := gomock.Any()

	fundId := FundId{0x01}
	vm.EXPECT().Call(any, any, any, any, any).Return(makeGasConfigResponse(100_000, 60_000, 210_000), uint64(0), nil)
	vm.EXPECT().Call(any, any, any, any, any).Return(fundId[:], uint64(0), nil)

	result, err := IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.NoError(err)
	require.Equal(uint64(0), result.config.TrackGasCost)
}

// TestIsCovered_ExtendedGetGasConfig_128Bytes_ParsesTrackGasCost verifies
// that a 128-byte getGasConfig response correctly provides the TrackGasCost.
func TestIsCovered_ExtendedGetGasConfig_128Bytes_ParsesTrackGasCost(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	vm := NewMockVirtualMachine(ctrl)

	upgrades := opera.Upgrades{GasSubsidies: true}
	tx, signer := makeTestTx(t)
	any := gomock.Any()

	const wantTrackGasCost = uint64(55_000)
	fundId := FundId{0x01}
	vm.EXPECT().Call(any, any, any, any, any).Return(makeGasConfigResponseWithTrack(100_000, 60_000, 210_000, wantTrackGasCost), uint64(0), nil)
	vm.EXPECT().Call(any, any, any, any, any).Return(fundId[:], uint64(0), nil)

	result, err := IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.NoError(err)
	require.Equal(wantTrackGasCost, result.config.TrackGasCost)
}

// TestIsCovered_UnknownMode_IsRejected verifies that a 64-byte chooseFund response
// with an out-of-range mode value returns an error and does not panic.
func TestIsCovered_UnknownMode_IsRejected(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	vm := NewMockVirtualMachine(ctrl)

	upgrades := opera.Upgrades{GasSubsidies: true}
	tx, signer := makeTestTx(t)
	any := gomock.Any()

	vm.EXPECT().Call(any, any, any, any, any).Return(makeGasConfigResponse(100_000, 60_000, 210_000), uint64(0), nil)
	vm.EXPECT().Call(any, any, any, any, any).Return(makeChooseFundResponse(4, [32]byte{}), uint64(0), nil)

	_, err := IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.ErrorContains(err, "unknown sponsorship mode")
}

// TestGetPostTransactions_NetworkWithTracking_ProducesCorrectTransaction verifies
// that GetPostTransactions builds an internal transaction targeting the registry
// with the correct selector, tracking ID, and fee encoding.
func TestGetPostTransactions_NetworkWithTracking_ProducesCorrectTransaction(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	nonceSource := NewMockNonceSource(ctrl)
	nonceSource.EXPECT().GetNonce(common.Address{}).Return(uint64(7))

	trackingId := TrackingId{0xca, 0xfe, 0xba, 0xbe}
	config := GasConfig{
		SponsorshipOverheadGasCost: 10_000,
		TrackGasCost:               55_000,
	}
	gasUsed := uint64(21000)
	gasPrice := big.NewInt(1_000_000_000)

	postTxs, err := (Sponsorship{mode: sponsorshipModeNetworkWithTracking, trackingId: trackingId, config: config}).
		GetPostTransactions(nonceSource, gasUsed, gasPrice)
	require.NoError(err)
	require.Len(postTxs, 1)
	tx := postTxs[0]

	require.True(internaltx.IsInternal(tx))
	require.Equal(uint64(7), tx.Nonce())
	require.Equal(registry.GetAddress(), *tx.To())
	require.Equal(common.Big0, tx.Value())
	require.Equal(uint64(55_000), tx.Gas())
	require.Equal(common.Big0, tx.GasPrice())

	data := tx.Data()
	require.Equal(4+2*32, len(data))
	selector := binary.BigEndian.Uint32(data[:4])
	require.Equal(uint32(registry.TrackFunctionSelector), selector)
	require.Equal(trackingId[:], data[4:36])

	wantFee := new(big.Int).Mul(
		new(big.Int).SetUint64(gasUsed+config.SponsorshipOverheadGasCost),
		gasPrice,
	)
	gotFee := new(uint256.Int).SetBytes32(data[36:68]).ToBig()
	require.Equal(wantFee, gotFee)
}

// TestGetPostTransactions_NetworkWithTracking_FeeOverflow_ReturnsError verifies
// that an overflowing fee calculation returns an error instead of panicking.
func TestGetPostTransactions_NetworkWithTracking_FeeOverflow_ReturnsError(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	nonceSource := NewMockNonceSource(ctrl)
	nonceSource.EXPECT().GetNonce(common.Address{}).Return(uint64(0))

	config := GasConfig{SponsorshipOverheadGasCost: 1, TrackGasCost: 1}
	gasPrice := new(big.Int).Lsh(big.NewInt(1), 256) // 2^256
	_, err := (Sponsorship{mode: sponsorshipModeNetworkWithTracking, config: config}).
		GetPostTransactions(nonceSource, 0, gasPrice)
	require.ErrorContains(err, "fee calculation overflow")
}

// TestIsTrackTransaction_ValidTrackTransaction_ReturnsTrue verifies detection of
// track transactions.
func TestIsTrackTransaction_ValidTrackTransaction_ReturnsTrue(t *testing.T) {
	addr := registry.GetAddress()
	tx := types.NewTx(&types.LegacyTx{To: &addr, Data: createRegistryCallInput(registry.TrackFunctionSelector, [32]byte{}, uint256.Int{})})
	require.True(t, internaltx.IsInternal(tx))
	require.True(t, IsTrackTransaction(tx))
}

// TestIsTrackTransaction_FeeChargeTransaction_ReturnsFalse verifies that a
// deductFees transaction is not identified as a track transaction.
func TestIsTrackTransaction_FeeChargeTransaction_ReturnsFalse(t *testing.T) {
	addr := registry.GetAddress()
	tx := types.NewTx(&types.LegacyTx{To: &addr, Data: createRegistryCallInput(registry.DeductFeesFunctionSelector, [32]byte{}, uint256.Int{})})
	require.True(t, internaltx.IsInternal(tx))
	require.False(t, IsTrackTransaction(tx))
}

// TestParseTrackAmount_ValidInput_ReturnsFee verifies extraction of the fee
// from a track transaction's calldata.
func TestParseTrackAmount_ValidInput_ReturnsFee(t *testing.T) {
	fee := uint256.NewInt(42_000_000_000)
	addr := registry.GetAddress()
	tx := types.NewTx(&types.LegacyTx{To: &addr, Data: createRegistryCallInput(registry.TrackFunctionSelector, [32]byte{0x01}, *fee)})
	require.True(t, IsTrackTransaction(tx))
	got, err := ParseTrackAmount(tx)
	require.NoError(t, err)
	require.Equal(t, fee, got)
}

// TestOverhead_NetworkWithTracking_SizeIsUpperBoundForRlpEncodedTx verifies
// that Overhead().Size is also an upper bound for the track transaction size.
func TestOverhead_NetworkWithTracking_SizeIsUpperBoundForRlpEncodedTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	nonceSource := NewMockNonceSource(ctrl)
	nonceSource.EXPECT().GetNonce(common.Address{}).Return(uint64(math.MaxUint64))
	config := GasConfig{SponsorshipOverheadGasCost: math.MaxUint64, TrackGasCost: math.MaxUint64}
	gasPrice := big.NewInt(math.MaxInt64)

	s := Sponsorship{mode: sponsorshipModeNetworkWithTracking, config: config}
	postTxs, err := s.GetPostTransactions(nonceSource, math.MaxUint64, gasPrice)
	require.NoError(t, err)
	require.Len(t, postTxs, 1)
	data, err := rlp.EncodeToBytes(postTxs[0])
	require.NoError(t, err)
	require.Less(t, len(data), int(s.Overhead().Size))
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
