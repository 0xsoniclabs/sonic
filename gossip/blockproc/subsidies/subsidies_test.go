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

package subsidies

import (
	byte_rand "crypto/rand"
	"encoding/binary"
	"fmt"
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
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

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

			upgrades := opera.GetSonicUpgrades()
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
				Transfer: func(_ vm.StateDB, _ common.Address, _ common.Address, amount *uint256.Int) {
					require.Equal(0, amount.Sign())
				},
				Random: &common.Hash{}, // < signals Revision >= Merge
			}

			vmConfig := opera.GetVmConfig(rules)
			vm := vm.NewEVM(blockContext, state, chainConfig, vmConfig)

			covered, fundId, err := IsCovered(upgrades, vm, signer, tx, baseFee)
			require.NoError(err)
			require.Equal(test.expectCovered, covered)
			if test.expectCovered {
				require.NotEmpty(fundId)
			} else {
				require.Empty(fundId)
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

	upgrades := opera.GetSonicUpgrades()
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
		Transfer: func(_ vm.StateDB, _ common.Address, _ common.Address, amount *uint256.Int) {
			require.Equal(0, amount.Sign())
		},
		Random: &common.Hash{}, // < signals Revision >= Merge
	}

	vmConfig := opera.GetVmConfig(rules)
	vm := vm.NewEVM(blockContext, state, chainConfig, vmConfig)

	_, _, err = IsCovered(upgrades, vm, signer, tx, baseFee)
	require.ErrorContains(err, "subsidies registry contract not found")
}

func TestIsCovered_GasSubsidiesDisabled_ReturnsFalse(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	vm := NewMockVirtualMachine(ctrl)

	selectedFundId := FundId{1, 2, 3}
	any := gomock.Any()
	vm.EXPECT().
		Call(any, any, any, any, any).
		Return(selectedFundId[:], uint64(0), nil).
		AnyTimes()

	upgrades := opera.Upgrades{}

	key, err := crypto.GenerateKey()
	require.NoError(err)
	signer := types.LatestSignerForChainID(nil)
	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		To: &common.Address{},
	})
	require.True(IsSponsorshipRequest(tx))

	covered, fundId, err := IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.NoError(err)
	require.False(covered)
	require.Empty(fundId)

	upgrades.GasSubsidies = true
	covered, fundId, err = IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.NoError(err)
	require.True(covered)
	require.Equal(fundId, selectedFundId)
}

func TestIsCovered_NotASponsorshipRequest_ReturnsFalse(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	vm := NewMockVirtualMachine(ctrl)

	any := gomock.Any()
	selectedFundId := FundId{1, 2, 3}
	vm.EXPECT().
		Call(any, any, any, any, any).
		Return(selectedFundId[:], uint64(0), nil).
		AnyTimes()

	upgrades := opera.Upgrades{
		GasSubsidies: true,
	}

	key, err := crypto.GenerateKey()
	require.NoError(err)
	signer := types.LatestSignerForChainID(nil)

	// Non-Sponsorship request (no recipient) is rejected.
	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{})
	require.False(IsSponsorshipRequest(tx))
	covered, fundId, err := IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.NoError(err)
	require.False(covered)
	require.Empty(fundId)

	// Sponsorship request is accepted.
	tx = types.MustSignNewTx(key, signer, &types.LegacyTx{
		To: &common.Address{},
	})
	require.True(IsSponsorshipRequest(tx))
	covered, fundId, err = IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.NoError(err)
	require.True(covered)
	require.Equal(fundId, selectedFundId)
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

	// If the query returns the 0-fund ID, IsCovered returns false.
	any := gomock.Any()
	selectedFundId := FundId{}
	vm.EXPECT().Call(any, any, any, any, any).Return(selectedFundId[:], uint64(0), nil)
	covered, fundId, err := IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.NoError(err)
	require.False(covered)
	require.Empty(fundId)

	// If the query returns a non-zero fund ID, IsCovered returns true.
	selectedFundId = FundId{1, 2, 3}
	vm.EXPECT().Call(any, any, any, any, any).Return(selectedFundId[:], uint64(0), nil)
	covered, fundId, err = IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.NoError(err)
	require.True(covered)
	require.Equal(fundId, selectedFundId)
}

func TestIsCovered_SenderReaderFails_ReturnsError(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	reader := NewMockSenderReader(ctrl)

	upgrades := opera.Upgrades{
		GasSubsidies: true,
	}

	key, err := crypto.GenerateKey()
	require.NoError(err)
	signer := types.LatestSignerForChainID(nil)

	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		To: &common.Address{},
	})

	issue := fmt.Errorf("injected issue")
	reader.EXPECT().Sender(tx).Return(common.Address{}, issue)

	_, _, err = IsCovered(upgrades, nil, reader, tx, big.NewInt(1))
	require.ErrorContains(err, "failed to derive sender")
	require.ErrorIs(err, issue)
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

	// If the EVM returns false, IsCovered returns false.
	any := gomock.Any()
	issue := fmt.Errorf("injected issue")
	vm.EXPECT().Call(any, any, any, any, any).Return(nil, uint64(0), issue)
	_, _, err = IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.ErrorContains(err, "EVM call failed")
	require.ErrorIs(err, issue)
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
	vm.EXPECT().Call(any, any, any, any, any).Return([]byte{0x01}, uint64(0), nil)
	_, _, err = IsCovered(upgrades, vm, signer, tx, big.NewInt(1))
	require.ErrorContains(err, "failed to parse result of subsidies registry call")
}

func TestGetFeeChargeTransaction_ValidInputs_ProducesCorrectInternalTransaction(t *testing.T) {
	nonces := []uint64{
		0, 1, 42, 1000,
	}
	fundIds := []FundId{
		{}, {1, 2, 3}, {0x12, 31: 0xff},
	}
	gasUsed := []int{
		0, 21000, 100000, 1_000_000,
	}
	gasPrice := []int{
		0, 1, 1e12,
	}
	for _, nonce := range nonces {
		for _, fundId := range fundIds {
			for _, gas := range gasUsed {
				for _, price := range gasPrice {
					t.Run(fmt.Sprintf("nonce=%d/fundId=%v/gas=%d/price=%d", nonce, fundId, gas, price), func(t *testing.T) {
						require := require.New(t)
						ctrl := gomock.NewController(t)
						nonceSource := NewMockNonceSource(ctrl)
						nonceSource.EXPECT().GetNonce(common.Address{}).Return(nonce)

						gasPrice := big.NewInt(int64(price))
						tx, err := GetFeeChargeTransaction(nonceSource, fundId, uint64(gas), gasPrice)
						require.NoError(err)
						require.NotNil(tx)

						require.True(internaltx.IsInternal(tx))
						require.Equal(nonce, tx.Nonce())
						require.NotNil(tx.To)
						require.Equal(registry.GetAddress(), *tx.To())
						require.Equal(common.Big0, tx.Value())
						require.Equal(uint64(registry.GasLimitForDeductFeesCall), tx.Gas())
						require.Equal(common.Big0, tx.GasPrice())
						require.Equal(common.Big0, tx.GasFeeCap())
						require.Equal(common.Big0, tx.GasTipCap())

						got := tx.Data()
						fee := uint256.NewInt(uint64(price * (gas + SponsorshipOverheadGasCost)))
						want := createDeductFeesInput(fundId, *fee)
						require.Equal(want, got)
					})
				}
			}
		}
	}
}

func TestGetFeeChargeTransaction_FeeOverflows_ReturnsError(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	nonceSource := NewMockNonceSource(ctrl)
	nonceSource.EXPECT().GetNonce(common.Address{}).Return(uint64(0))

	fundId := FundId{}
	gasUsed := uint64(0)
	gasPrice := new(big.Int).Lsh(big.NewInt(1), 256) // 2^256
	_, err := GetFeeChargeTransaction(nonceSource, fundId, gasUsed, gasPrice)
	require.ErrorContains(err, "fee calculation overflow")
}

func TestCreateChooseFundInput_ValidInputs_ProducesCorrectInputData(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	reader := NewMockSenderReader(ctrl)

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
	reader.EXPECT().Sender(tx).Return(sender, nil)

	feeData := [32]byte{}
	fillRandom(t, feeData[:])
	fee := new(big.Int).SetBytes(feeData[:])

	input, err := createChooseFundInput(reader, tx, fee)
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
	_, err := createChooseFundInput(nil, nil, nil)
	require.ErrorContains(err, "invalid transaction")
}

func TestCreateChooseFundInput_FeeOverflow_ReturnsError(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	reader := NewMockSenderReader(ctrl)
	reader.EXPECT().Sender(gomock.Any()).Return(common.Address{}, nil).AnyTimes()
	tx := types.NewTx(&types.LegacyTx{})

	tooHighFee := new(big.Int).Lsh(big.NewInt(1), 256) // 2^256
	_, err := createChooseFundInput(reader, tx, tooHighFee)
	require.ErrorContains(err, "fee does not fit into 32 bytes")

	justAcceptableFee := new(big.Int).Sub(tooHighFee, big.NewInt(1))
	_, err = createChooseFundInput(reader, tx, justAcceptableFee)
	require.NoError(err)
}

func TestCreateChooseFundInput_TransactionWithoutReceiver_ProducesAZeroedReceiver(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	reader := NewMockSenderReader(ctrl)

	sender := common.Address{}
	fillRandom(t, sender[:])
	nonce := rand.Uint64()

	tx := types.NewContractCreation(nonce, common.Big0, 21000, common.Big0, nil)
	reader.EXPECT().Sender(tx).Return(sender, nil)

	input, err := createChooseFundInput(reader, tx, common.Big0)
	require.NoError(err)

	target := input[4+32 : 4+2*32] // < receiver address
	require.Equal(make([]byte, 32), target)
}

func TestCreateChooseFundInput_SenderReaderFails_ReturnsError(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	reader := NewMockSenderReader(ctrl)

	tx := &types.Transaction{}
	issue := fmt.Errorf("injected issue")
	reader.EXPECT().Sender(tx).Return(common.Address{}, issue)

	_, err := createChooseFundInput(reader, tx, big.NewInt(1))
	require.ErrorContains(err, "failed to derive sender")
	require.ErrorIs(err, issue)
}

func TestCreateChooseFundInput_LongCallData_CallDataIsEncodedCorrectly(t *testing.T) {
	for n := range 1024 {
		t.Run(fmt.Sprintf("data length %d", n), func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)
			reader := NewMockSenderReader(ctrl)

			sender := common.Address{}
			receiver := common.Address{}
			data := make([]byte, n)

			fillRandom(t, sender[:])
			fillRandom(t, receiver[:])
			fillRandom(t, data)
			nonce := rand.Uint64()

			tx := types.NewTransaction(nonce, receiver, common.Big0, 21000, common.Big0, data)
			reader.EXPECT().Sender(tx).Return(sender, nil)

			feeData := [32]byte{}
			fillRandom(t, feeData[:])
			fee := new(big.Int).SetBytes(feeData[:])

			input, err := createChooseFundInput(reader, tx, fee)
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
		covered bool
		fundId  FundId
	}{
		"empty fund": {
			covered: false,
			fundId:  FundId{},
		},
		"non-empty fund": {
			covered: true,
			fundId:  FundId{1, 2, 3},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			input := test.fundId[:]
			covered, fundId, err := parseChooseFundResult(input)
			require.NoError(err)
			require.Equal(test.covered, covered)
			wantedFund := test.fundId
			if !test.covered {
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
		"too long": {
			input: make([]byte, 32+1),
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
				input := createDeductFeesInput(id, *fee)
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

func fillRandom(t *testing.T, b []byte) {
	_, err := byte_rand.Read(b)
	require.NoError(t, err)
}
