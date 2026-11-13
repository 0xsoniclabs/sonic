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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/contracts/evmwriter"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	baseBlockNumber = 10
)

func TestSimulateV1_EmptyBlock_ReturnsOneBlockWithNoCalls(t *testing.T) {
	f := newSimulateV1Helper(t)
	f.mockState.EXPECT().GetStateHash().Return(common.Hash{})
	f.mockState.EXPECT().Release()

	results, err := f.api.SimulateV1(context.Background(), simOpts{BlockStateCalls: []simBlock{{}}}, &f.blkNr)

	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Empty(t, results[0].calls)
}

func TestSimulateV1_SingleCall_ReturnsSuccessResult(t *testing.T) {
	f := newSimulateV1Helper(t)
	setExpectedStateCalls(f.mockState)
	f.mockState.EXPECT().GetStateHash().Return(common.Hash{})

	from := common.Address{1}
	to := common.Address{2}
	gas := hexutil.Uint64(21_000)
	opts := simOpts{
		BlockStateCalls: []simBlock{
			{Calls: []TransactionArgs{{From: &from, To: &to, Gas: &gas}}},
		},
	}

	results, err := f.api.SimulateV1(context.Background(), opts, &f.blkNr)

	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].calls, 1)
	require.Equal(t, hexutil.Uint64(types.ReceiptStatusSuccessful), results[0].calls[0].Status)
	require.Equal(t, hexutil.Uint64(21_000), results[0].calls[0].GasUsed)
}

func TestSimulateV1_SingleCall_NonceValidationFailure(t *testing.T) {
	f := newSimulateV1Helper(t)
	setExpectedStateCalls(f.mockState)

	from := common.Address{1}
	to := common.Address{2}
	nonce := uint64(10)
	opts := simOpts{
		BlockStateCalls: []simBlock{
			{Calls: []TransactionArgs{
				{
					From:  &from,
					To:    &to,
					Nonce: (*hexutil.Uint64)(&nonce),
				},
			}},
		},
		Validation: true,
	}

	_, err := f.api.SimulateV1(context.Background(), opts, &f.blkNr)
	require.Error(t, err)
	simTxError, ok := err.(*simInvalidTxError)
	require.True(t, ok)
	require.Equal(t, errCodeNonceTooHigh, simTxError.ErrorCode())
}

func TestSimulateV1_Validation_SkipsGasLimitCapAndEOAChecks(t *testing.T) {
	from := common.Address{1}
	to := common.Address{2}
	setup := newSimulateV1Setup()
	txGasCap := opera.FakeNetRules(setup.upgrades).Economy.Gas.MaxEventGas
	gas := hexutil.Uint64(21_000)
	gasAboveCap := hexutil.Uint64(txGasCap + 1)

	tests := map[string]struct {
		call       TransactionArgs
		senderCode []byte
	}{
		"gas defaults to more than the transaction gas cap": {
			call: TransactionArgs{From: &from, To: &to},
		},
		"gas is set above the transaction gas cap": {
			call: TransactionArgs{From: &from, To: &to, Gas: &gasAboveCap},
		},
		"sender has code": {
			call:       TransactionArgs{From: &from, To: &to, Gas: &gas},
			senderCode: []byte{byte(vm.STOP)},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			osakaTime := uint64(0)
			setup := newSimulateV1Setup()
			setup.chainConfig.LondonBlock = big.NewInt(0)
			setup.chainConfig.OsakaTime = &osakaTime
			setup.base.GasLimit = 2 * txGasCap
			setup.gasCap = 2 * txGasCap
			f := newSimulateV1HelperWith(t, setup)
			f.mockState.EXPECT().GetCode(from).Return(test.senderCode).AnyTimes()
			setExpectedStateCalls(f.mockState)
			f.mockState.EXPECT().GetStateHash().Return(common.Hash{})

			opts := simOpts{
				BlockStateCalls: []simBlock{{Calls: []TransactionArgs{test.call}}},
				Validation:      true,
			}

			results, err := f.api.SimulateV1(t.Context(), opts, &f.blkNr)

			require.NoError(t, err)
			require.Len(t, results, 1)
			require.Len(t, results[0].calls, 1)
			require.Equal(t, hexutil.Uint64(types.ReceiptStatusSuccessful), results[0].calls[0].Status)
		})
	}
}

func TestSimulateV1_ChargesUnusedGasAsConfiguredByNetworkRules(t *testing.T) {
	singleProposer := opera.GetBrioUpgrades()
	singleProposer.SingleProposerBlockFormation = true

	tests := map[string]struct {
		upgrades    opera.Upgrades
		wantGasUsed hexutil.Uint64
	}{
		"multiple proposers charge 10% of unused gas": {
			upgrades:    opera.GetBrioUpgrades(),
			wantGasUsed: 21_000 + (100_000-21_000)/10,
		},
		"single proposer does not charge unused gas": {
			upgrades:    singleProposer,
			wantGasUsed: 21_000,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			setup := newSimulateV1Setup()
			setup.upgrades = test.upgrades
			f := newSimulateV1HelperWith(t, setup)
			setExpectedStateCalls(f.mockState)
			f.mockState.EXPECT().GetStateHash().Return(common.Hash{})

			from := common.Address{1}
			to := common.Address{2}
			gas := hexutil.Uint64(100_000)
			opts := simOpts{
				BlockStateCalls: []simBlock{
					{Calls: []TransactionArgs{{From: &from, To: &to, Gas: &gas}}},
				},
			}

			results, err := f.api.SimulateV1(t.Context(), opts, &f.blkNr)

			require.NoError(t, err)
			require.Len(t, results, 1)
			require.Len(t, results[0].calls, 1)
			require.Equal(t, test.wantGasUsed, results[0].calls[0].GasUsed)
		})
	}
}

func TestSimulateV1_AppliesSonicInitCodeSizeLimit(t *testing.T) {
	shanghaiTime := uint64(0)
	setup := newSimulateV1Setup()
	setup.chainConfig.LondonBlock = big.NewInt(0)
	setup.chainConfig.ShanghaiTime = &shanghaiTime
	setup.base.PrevRandao = common.Hash{1}
	f := newSimulateV1HelperWith(t, setup)
	setExpectedStateCalls(f.mockState)

	from := common.Address{1}
	gas := hexutil.Uint64(500_000)
	initCode := hexutil.Bytes(make([]byte, opera.SonicBrioMaxInitCodeSize+1))
	opts := simOpts{
		BlockStateCalls: []simBlock{
			{Calls: []TransactionArgs{{From: &from, Gas: &gas, Data: &initCode}}},
		},
	}

	_, err := f.api.SimulateV1(t.Context(), opts, &f.blkNr)

	simTxError, ok := err.(*simInvalidTxError)
	require.True(t, ok)
	require.Equal(t, errCodeMaxInitCodeSizeExceeded, simTxError.ErrorCode())
	require.ErrorContains(t, err, fmt.Sprintf("limit %d", opera.SonicBrioMaxInitCodeSize))
}

func TestSimulateV1_CodeOverrideTakesPrecedenceOverStatePrecompile(t *testing.T) {
	f := newSimulateV1Helper(t)
	// PUSH1 0x42 PUSH1 0x00 MSTORE PUSH1 0x20 PUSH1 0x00 RETURN
	code := hexutil.Bytes{0x60, 0x42, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3}
	precompile := evmwriter.ContractAddress
	f.mockState.EXPECT().SetCode(precompile, []byte(code), gomock.Any())
	f.mockState.EXPECT().GetCode(precompile).Return([]byte(code)).AnyTimes()
	f.mockState.EXPECT().GetCodeHash(precompile).Return(crypto.Keccak256Hash(code)).AnyTimes()
	f.mockState.EXPECT().RevertToSnapshot(gomock.Any()).AnyTimes()
	setExpectedStateCalls(f.mockState)
	f.mockState.EXPECT().GetStateHash().Return(common.Hash{})

	from := common.Address{1}
	gas := hexutil.Uint64(100_000)
	opts := simOpts{
		BlockStateCalls: []simBlock{{
			StateOverrides: &StateOverride{precompile: {Code: &code}},
			Calls:          []TransactionArgs{{From: &from, To: &precompile, Gas: &gas}},
		}},
	}

	results, err := f.api.SimulateV1(t.Context(), opts, &f.blkNr)

	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].calls, 1)
	require.Equal(t, hexutil.Uint64(types.ReceiptStatusSuccessful), results[0].calls[0].Status)
	require.Equal(t, hexutil.Bytes(common.LeftPadBytes([]byte{0x42}, 32)), results[0].calls[0].ReturnValue)

	vmConfig := opera.GetVmConfig(opera.FakeNetRules(opera.GetBrioUpgrades()))
	require.Contains(t, vmConfig.StatePrecompiles, precompile, "shared network configuration must not be modified")
}

func TestSimulateV1_EndsTransactionAfterEachCall(t *testing.T) {
	f := newSimulateV1Helper(t)
	gomock.InOrder(
		f.mockState.EXPECT().SetTxContext(gomock.Any(), 0),
		f.mockState.EXPECT().EndTransaction(),
		f.mockState.EXPECT().SetTxContext(gomock.Any(), 1),
		f.mockState.EXPECT().EndTransaction(),
	)
	setExpectedStateCalls(f.mockState)
	f.mockState.EXPECT().GetStateHash().Return(common.Hash{})

	from := common.Address{1}
	to := common.Address{2}
	gas := hexutil.Uint64(100_000)
	call := TransactionArgs{From: &from, To: &to, Gas: &gas}
	opts := simOpts{
		BlockStateCalls: []simBlock{{Calls: []TransactionArgs{call, call}}},
	}

	results, err := f.api.SimulateV1(t.Context(), opts, &f.blkNr)

	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].calls, 2)
}

func TestSimulateV1_BlobCallWithoutBlobFeeCapSucceeds(t *testing.T) {
	zero := uint64(0)
	setup := newSimulateV1Setup()
	setup.chainConfig = &params.ChainConfig{
		ChainID:      big.NewInt(1234),
		LondonBlock:  big.NewInt(0),
		ShanghaiTime: &zero,
		CancunTime:   &zero,
	}
	f := newSimulateV1HelperWith(t, setup)
	setExpectedStateCalls(f.mockState)
	f.mockState.EXPECT().GetStateHash().Return(common.Hash{})

	from := common.Address{1}
	to := common.Address{2}
	gas := hexutil.Uint64(100_000)
	call := TransactionArgs{
		From:       &from,
		To:         &to,
		Gas:        &gas,
		BlobHashes: []common.Hash{{0x01}},
	}
	opts := simOpts{BlockStateCalls: []simBlock{{Calls: []TransactionArgs{call}}}}

	results, err := f.api.SimulateV1(t.Context(), opts, &f.blkNr)

	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].calls, 1)
	require.Equal(t, hexutil.Uint64(types.ReceiptStatusSuccessful), results[0].calls[0].Status)
}

func TestSimulateV1_FailsWithoutNetworkRules(t *testing.T) {
	injectedErr := errors.New("injected error")
	tests := map[string]struct {
		rules   *opera.Rules
		err     error
		wantErr string
	}{
		"lookup fails": {
			err:     injectedErr,
			wantErr: injectedErr.Error(),
		},
		"no rules": {
			wantErr: "no network rules found for block height 10",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockBackend := NewMockBackend(ctrl)
			mockState := state.NewMockStateDB(ctrl)
			blkNr := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)

			mockBackend.EXPECT().StateAndBlockByNumberOrHash(gomock.Any(), blkNr).Return(mockState, newSimTestBase(), nil)
			mockBackend.EXPECT().RPCGasCap().Return(uint64(10_000_000))
			mockBackend.EXPECT().ChainConfig(idx.Block(baseBlockNumber)).Return(newSimTestChainConfig())
			mockBackend.EXPECT().GetNetworkRules(gomock.Any(), idx.Block(baseBlockNumber)).Return(test.rules, test.err)
			mockState.EXPECT().Release()

			_, err := NewPublicBlockChainAPI(mockBackend).SimulateV1(t.Context(), simOpts{BlockStateCalls: []simBlock{{}}}, &blkNr)

			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestSimulateV1_MultipleBlocks_ReturnsResultsPerBlock(t *testing.T) {
	f := newSimulateV1Helper(t)
	// Two empty blocks: GetStateHash is called once per block.
	f.mockState.EXPECT().GetStateHash().Return(common.Hash{}).Times(2)
	f.mockState.EXPECT().Release()

	results, err := f.api.SimulateV1(context.Background(), simOpts{BlockStateCalls: []simBlock{{}, {}}}, &f.blkNr)

	require.NoError(t, err)
	require.Len(t, results, 2)
	// Block numbers should be base+1 and base+2.
	require.Equal(t, big.NewInt(baseBlockNumber+1), results[0].block.Number)
	require.Equal(t, big.NewInt(baseBlockNumber+2), results[1].block.Number)
}

func TestSimulateV1_SingleCall_ReturnsFullTxResult(t *testing.T) {
	f := newSimulateV1Helper(t)
	setExpectedStateCalls(f.mockState)
	f.mockState.EXPECT().GetStateHash().Return(common.Hash{})

	from := common.Address{1}
	to := common.Address{2}
	gas := hexutil.Uint64(21_000)
	opts := simOpts{
		BlockStateCalls: []simBlock{
			{Calls: []TransactionArgs{{From: &from, To: &to, Gas: &gas}}},
		},
		ReturnFullTransactions: true,
	}

	results, err := f.api.SimulateV1(context.Background(), opts, &f.blkNr)

	require.NoError(t, err)
	require.Len(t, results, 1)
	require.True(t, results[0].fullTx)

	// This is happening automatically when marshaling the result to JSON
	// with RPC api.
	blockBytes, err := results[0].MarshalJSON()
	require.NoError(t, err)

	var blockJson map[string]interface{}
	require.NoError(t, json.Unmarshal(blockBytes, &blockJson))
	require.Equal(t, from, common.HexToAddress(blockJson["transactions"].([]interface{})[0].(map[string]interface{})["from"].(string)))
}

// newSimTestBase constructs a minimal EvmBlock suitable as a simulation base block.
func newSimTestBase() *evmcore.EvmBlock {
	return &evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{
			Number:   big.NewInt(baseBlockNumber),
			GasLimit: 1_000_000,
		},
	}
}

// newSimTestChainConfig returns a minimal chain config.
func newSimTestChainConfig() *params.ChainConfig {
	return &params.ChainConfig{ChainID: big.NewInt(1234)}
}

// newSimulator constructs a simulator with a pre-configured test base block
// and chain config, for testing internal methods directly.
func newSimulator() *simulator {
	return &simulator{
		base:        newSimTestBase().Header(),
		chainConfig: newSimTestChainConfig(),
	}
}

// simulateV1Helper holds the common setup shared across SimulateV1 tests.
type simulateV1Helper struct {
	api       *PublicBlockChainAPI
	mockState *state.MockStateDB
	blkNr     rpc.BlockNumberOrHash
}

type simulateV1Setup struct {
	upgrades    opera.Upgrades
	base        *evmcore.EvmBlock
	chainConfig *params.ChainConfig
	gasCap      uint64
}

func newSimulateV1Setup() simulateV1Setup {
	return simulateV1Setup{
		upgrades:    opera.GetBrioUpgrades(),
		base:        newSimTestBase(),
		chainConfig: newSimTestChainConfig(),
		gasCap:      10_000_000,
	}
}

// newSimulateV1Helper sets up the common mocks and returns a fixture for SimulateV1 tests.
func newSimulateV1Helper(t *testing.T) *simulateV1Helper {
	t.Helper()
	return newSimulateV1HelperWith(t, newSimulateV1Setup())
}

func newSimulateV1HelperWith(t *testing.T, setup simulateV1Setup) *simulateV1Helper {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockBackend := NewMockBackend(ctrl)
	mockState := state.NewMockStateDB(ctrl)

	blkNr := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
	rules := opera.FakeNetRules(setup.upgrades)

	mockBackend.EXPECT().StateAndBlockByNumberOrHash(gomock.Any(), blkNr).Return(mockState, setup.base, nil)
	mockBackend.EXPECT().RPCGasCap().Return(setup.gasCap)
	mockBackend.EXPECT().ChainConfig(idx.Block(baseBlockNumber)).Return(setup.chainConfig)
	mockBackend.EXPECT().GetNetworkRules(gomock.Any(), idx.Block(baseBlockNumber)).Return(&rules, nil)
	mockBackend.EXPECT().RPCEVMTimeout().Return(time.Duration(0))

	return &simulateV1Helper{
		api:       NewPublicBlockChainAPI(mockBackend),
		mockState: mockState,
		blkNr:     blkNr,
	}
}

func TestMakeEvmHeader_NilOverride_ReturnsCopyOfTemplate(t *testing.T) {
	var override *simBlockOverrides
	template := &evmcore.EvmHeader{
		Number:   big.NewInt(5),
		GasLimit: 500_000,
	}

	result := override.applyTo(template)

	require.Equal(t, template.Number, result.Number)
	require.Equal(t, template.GasLimit, result.GasLimit)
	// Must be a distinct copy, not the same pointer.
	require.NotSame(t, template, result)
}

func TestMakeEvmHeader_AppliesAllOverrides(t *testing.T) {
	newNumber := big.NewInt(99)
	newTime := hexutil.Uint64(5000)
	newGasLimit := hexutil.Uint64(200_000)
	newCoinbase := common.Address{7}
	newPrevRandao := common.Hash{8}
	newBaseFee := big.NewInt(1234)

	override := &simBlockOverrides{
		Number:        (*hexutil.Big)(newNumber),
		Time:          &newTime,
		GasLimit:      &newGasLimit,
		FeeRecipient:  &newCoinbase,
		PrevRandao:    &newPrevRandao,
		BaseFeePerGas: (*hexutil.Big)(newBaseFee),
	}
	template := &evmcore.EvmHeader{Number: big.NewInt(1)}

	result := override.applyTo(template)

	require.Equal(t, newNumber, result.Number)
	require.Equal(t, inter.FromUnix(int64(newTime)), result.Time)
	require.Equal(t, uint64(newGasLimit), result.GasLimit)
	require.Equal(t, newCoinbase, result.Coinbase)
	require.Equal(t, newPrevRandao, result.PrevRandao)
	require.Equal(t, newBaseFee, result.BaseFee)
}

func TestMakeEvmHeader_UnsetFieldsRetainTemplateValues(t *testing.T) {
	override := &simBlockOverrides{} // no fields set
	template := &evmcore.EvmHeader{
		Number:     big.NewInt(42),
		GasLimit:   999_999,
		Coinbase:   common.Address{5},
		Time:       inter.FromUnix(1000),
		PrevRandao: common.Hash{6},
		BaseFee:    big.NewInt(123),
	}

	result := override.applyTo(template)

	// Unset override fields should leave template values intact.
	require.Equal(t, template.Number, result.Number)
	require.Equal(t, template.GasLimit, result.GasLimit)
	require.Equal(t, template.Coinbase, result.Coinbase)
	require.Equal(t, template.Time, result.Time)
	require.Equal(t, template.PrevRandao, result.PrevRandao)
	require.Equal(t, template.BaseFee, result.BaseFee)
}

func TestSanitizeChain_AutoAssignsSequentialBlockNumbers(t *testing.T) {
	sim := newSimulator()
	blocks := []simBlock{{}, {}}

	result, err := sim.sanitizeChain(blocks)

	require.NoError(t, err)
	require.Len(t, result, 2)
	// Base is 10, so blocks should be at 11 and 12.
	require.Equal(t, big.NewInt(baseBlockNumber+1), result[0].BlockOverrides.Number.ToInt())
	require.Equal(t, big.NewInt(baseBlockNumber+2), result[1].BlockOverrides.Number.ToInt())
}

func TestSanitizeChain_AutoAssignsIncreasingTimestamps(t *testing.T) {
	sim := newSimulator()
	sim.base.Time = inter.FromUnix(1000)
	blocks := []simBlock{{}, {}}

	result, err := sim.sanitizeChain(blocks)

	require.NoError(t, err)
	require.Len(t, result, 2)
	require.Equal(t, hexutil.Uint64(1000+timestampIncrement), *result[0].BlockOverrides.Time)
	require.Equal(t, hexutil.Uint64(1000+2*timestampIncrement), *result[1].BlockOverrides.Time)
}

func TestSanitizeChain_FillsGapsWithEmptyBlocks(t *testing.T) {
	sim := newSimulator()
	// Base is 10; request block 13 directly, which creates a gap at 11 and 12.
	n := hexutil.Big(*big.NewInt(baseBlockNumber + 3))
	blocks := []simBlock{{BlockOverrides: &simBlockOverrides{Number: &n}}}

	result, err := sim.sanitizeChain(blocks)

	require.NoError(t, err)
	// Gap blocks 11, 12 plus the requested block 13.
	require.Len(t, result, 3)
	require.Equal(t, big.NewInt(baseBlockNumber+1), result[0].BlockOverrides.Number.ToInt())
	require.Equal(t, big.NewInt(baseBlockNumber+2), result[1].BlockOverrides.Number.ToInt())
	require.Equal(t, big.NewInt(baseBlockNumber+3), result[2].BlockOverrides.Number.ToInt())
}

func TestSanitizeChain_RejectsNonIncreasingBlockNumbers(t *testing.T) {
	sim := newSimulator()
	n1 := hexutil.Big(*big.NewInt(baseBlockNumber + 1))
	n2 := hexutil.Big(*big.NewInt(baseBlockNumber + 1)) // same as previous (non-increasing)
	blocks := []simBlock{
		{BlockOverrides: &simBlockOverrides{Number: &n1}},
		{BlockOverrides: &simBlockOverrides{Number: &n2}},
	}

	_, err := sim.sanitizeChain(blocks)
	simTxError, ok := err.(*simInvalidTxError)
	require.True(t, ok)
	require.Equal(t, errCodeBlockNumberInvalid, simTxError.ErrorCode())
}

func TestSanitizeChain_RejectsNonIncreasingTimestamps(t *testing.T) {
	sim := newSimulator()
	sim.base.Time = inter.FromUnix(1000)
	// Timestamp 999 is earlier than the base timestamp of 1000.
	ts := hexutil.Uint64(999)
	blocks := []simBlock{
		{BlockOverrides: &simBlockOverrides{Time: &ts}},
	}

	_, err := sim.sanitizeChain(blocks)

	simTxError, ok := err.(*simInvalidTxError)
	require.True(t, ok)
	require.Equal(t, errCodeBlockTimestampInvalid, simTxError.ErrorCode())
}

func TestSanitizeChain_RejectsBlockRangeExceedingLimit(t *testing.T) {
	sim := newSimulator()
	// Request a block more than maxSimulateBlocks beyond the base.
	tooFar := new(big.Int).Add(sim.base.Number, big.NewInt(maxSimulateBlocks+1))
	n := hexutil.Big(*tooFar)
	blocks := []simBlock{{BlockOverrides: &simBlockOverrides{Number: &n}}}

	_, err := sim.sanitizeChain(blocks)

	simTxError, ok := err.(*simInvalidTxError)
	require.True(t, ok)
	require.Equal(t, errCodeClientLimitExceeded, simTxError.ErrorCode())
}

func TestSimDummyChain_Header(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockBackend := NewMockBackend(ctrl)
	ctx := context.Background()

	baseHeader := &evmcore.EvmHeader{
		Number: big.NewInt(100),
		Hash:   common.Hash{0x01},
	}

	processedHeader1 := &evmcore.EvmHeader{
		Number: big.NewInt(101),
		Hash:   common.Hash{0x02},
	}
	processedHeader2 := &evmcore.EvmHeader{
		Number: big.NewInt(102),
		Hash:   common.Hash{0x03},
	}

	chain := &simDummyChain{
		ctx:              ctx,
		backend:          mockBackend,
		base:             baseHeader,
		processedHeaders: []*evmcore.EvmHeader{processedHeader1, processedHeader2},
	}

	t.Run("ReturnsBaseHeader", func(t *testing.T) {
		h := chain.Header(baseHeader.Hash, baseHeader.Number.Uint64())
		require.Equal(t, baseHeader, h)
	})

	t.Run("ReturnsProcessedHeader", func(t *testing.T) {
		h := chain.Header(processedHeader1.Hash, processedHeader1.Number.Uint64())
		require.Equal(t, processedHeader1, h)

		h = chain.Header(processedHeader2.Hash, processedHeader2.Number.Uint64())
		require.Equal(t, processedHeader2, h)
	})

	t.Run("ReturnsHeaderFromBackend", func(t *testing.T) {
		backendHeader := &evmcore.EvmHeader{
			Number: big.NewInt(99),
			Hash:   common.Hash{0x99},
		}
		mockBackend.EXPECT().HeaderByNumber(ctx, rpc.BlockNumber(99)).Return(backendHeader, nil)

		h := chain.Header(backendHeader.Hash, backendHeader.Number.Uint64())
		require.Equal(t, backendHeader, h)
	})

	t.Run("ReturnsNilIfBackendFails", func(t *testing.T) {
		mockBackend.EXPECT().HeaderByNumber(ctx, rpc.BlockNumber(98)).Return(nil, errors.New("backend error"))

		h := chain.Header(common.Hash{0x98}, 98)
		require.Nil(t, h)
	})

	t.Run("ReturnsNilIfBackendReturnsNil", func(t *testing.T) {
		mockBackend.EXPECT().HeaderByNumber(ctx, rpc.BlockNumber(97)).Return(nil, nil)

		h := chain.Header(common.Hash{0x97}, 97)
		require.Nil(t, h)
	})

	t.Run("ReturnsNilIfHashMismatch", func(t *testing.T) {
		backendHeader := &evmcore.EvmHeader{
			Number: big.NewInt(96),
			Hash:   common.Hash{0x96},
		}
		mockBackend.EXPECT().HeaderByNumber(ctx, rpc.BlockNumber(96)).Return(backendHeader, nil)

		h := chain.Header(common.Hash{0xAA}, 96) // Mismatched hash
		require.Nil(t, h)
	})

	t.Run("ReturnsNilIfHashMismatchWithProcessed", func(t *testing.T) {
		// Even if number matches processed header, if hash doesn't match, it should fall through to backend
		mockBackend.EXPECT().HeaderByNumber(ctx, rpc.BlockNumber(processedHeader1.Number.Uint64())).Return(nil, nil)

		h := chain.Header(common.Hash{0xBB}, processedHeader1.Number.Uint64())
		require.Nil(t, h)
	})
}

func TestSanitizeChain_SetsEmptyWithdrawalsOnEachBlock(t *testing.T) {
	sim := newSimulator()
	blocks := []simBlock{{}}

	result, err := sim.sanitizeChain(blocks)

	require.NoError(t, err)
	require.NotNil(t, result[0].BlockOverrides.Withdrawals)
}

func TestSanitizeCall_SetsDefaultsForMissingFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockState := state.NewMockStateDB(ctrl)

	gasUsed := uint64(0)
	nonce := uint64(1)
	gasLimit := uint64(10_000)
	expectedGas := gasLimit - gasUsed
	mockState.EXPECT().GetNonce(gomock.Any()).Return(nonce).AnyTimes()

	header := &evmcore.EvmHeader{
		Number:   big.NewInt(10),
		GasLimit: gasLimit,
	}
	sim := newSimulator()
	call := TransactionArgs{}

	err := sim.sanitizeCall(&call, mockState, header, &gasUsed)

	require.NoError(t, err)
	require.NotNil(t, call.Nonce)
	require.Equal(t, (*hexutil.Uint64)(&nonce), call.Nonce)
	require.NotNil(t, call.Gas)
	require.Equal(t, (*hexutil.Uint64)(&expectedGas), call.Gas)
}

func TestSanitizeCall_BlockGasLimitReachedError(t *testing.T) {
	gasUsed := uint64(20_000)
	nonce := uint64(1)
	gasLimit := uint64(10_000)

	header := &evmcore.EvmHeader{
		Number:   big.NewInt(10),
		GasLimit: gasLimit,
	}
	sim := newSimulator()
	call := TransactionArgs{
		Nonce: (*hexutil.Uint64)(&nonce),
		Gas:   (*hexutil.Uint64)(&gasLimit),
	}

	err := sim.sanitizeCall(&call, nil, header, &gasUsed)
	require.Error(t, err)
	simTxError, ok := err.(*simInvalidTxError)
	require.True(t, ok)
	require.Equal(t, errCodeBlockGasLimitReached, simTxError.ErrorCode())
}

func TestSanitizeCall_RejectsBlobTxWithNilTo(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockState := state.NewMockStateDB(ctrl)

	gasUsed := uint64(0)
	nonce := uint64(1)
	gasLimit := uint64(10_000)
	mockState.EXPECT().GetNonce(gomock.Any()).Return(nonce).AnyTimes()

	header := &evmcore.EvmHeader{
		Number:   big.NewInt(10),
		GasLimit: gasLimit,
	}
	sim := newSimulator()
	call := TransactionArgs{
		BlobHashes: []common.Hash{{0x01}},
	}

	err := sim.sanitizeCall(&call, mockState, header, &gasUsed)

	require.ErrorIs(t, err, core.ErrBlobTxCreate)
}

func TestSanitizeCall_RejectsSetCodeTxWithNilTo(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockState := state.NewMockStateDB(ctrl)

	gasUsed := uint64(0)
	nonce := uint64(1)
	gasLimit := uint64(10_000)
	mockState.EXPECT().GetNonce(gomock.Any()).Return(nonce).AnyTimes()

	header := &evmcore.EvmHeader{
		Number:   big.NewInt(10),
		GasLimit: gasLimit,
	}
	sim := newSimulator()
	call := TransactionArgs{
		AuthorizationList: []types.SetCodeAuthorization{{}},
	}

	err := sim.sanitizeCall(&call, mockState, header, &gasUsed)

	require.ErrorIs(t, err, core.ErrSetCodeTxCreate)
}

func TestSanitizeCall_AllowsPlainContractCreationWithNilTo(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockState := state.NewMockStateDB(ctrl)

	gasUsed := uint64(0)
	nonce := uint64(1)
	gasLimit := uint64(10_000)
	mockState.EXPECT().GetNonce(gomock.Any()).Return(nonce).AnyTimes()

	header := &evmcore.EvmHeader{
		Number:   big.NewInt(10),
		GasLimit: gasLimit,
	}
	sim := newSimulator()
	call := TransactionArgs{}

	err := sim.sanitizeCall(&call, mockState, header, &gasUsed)

	require.NoError(t, err)
}

func TestSanitizeCall_RejectsBlobFeeCapWithNilTo(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockState := state.NewMockStateDB(ctrl)

	gasUsed := uint64(0)
	nonce := uint64(1)
	gasLimit := uint64(10_000)
	mockState.EXPECT().GetNonce(gomock.Any()).Return(nonce).AnyTimes()

	header := &evmcore.EvmHeader{
		Number:   big.NewInt(10),
		GasLimit: gasLimit,
	}
	sim := newSimulator()
	// BlobFeeCap alone makes ToTransaction build a BlobTx, which needs "to".
	call := TransactionArgs{
		BlobFeeCap: (*hexutil.Big)(big.NewInt(1)),
	}

	err := sim.sanitizeCall(&call, mockState, header, &gasUsed)

	require.ErrorIs(t, err, core.ErrBlobTxCreate)
}

func TestRepairSimLogs(t *testing.T) {

	evmHeader := &evmcore.EvmHeader{
		Hash:   common.Hash{42},
		Number: big.NewInt(10),
		Time:   inter.FromUnix(1000),
	}

	callLog := &types.Log{BlockHash: common.Hash{}}

	calls := []simCallResult{
		{Logs: []*types.Log{callLog}},
	}

	repairSimLogs(calls, &evmcore.EvmBlock{EvmHeader: *evmHeader})
	require.Equal(t, evmHeader.Hash, callLog.BlockHash)

}

func TestSetCallPriceDefaults_SetsEIP1559FieldsWhenBaseFeePresent(t *testing.T) {
	sim := &simulator{chainConfig: newSimTestChainConfig()}
	call := TransactionArgs{}
	baseFee := big.NewInt(100)

	err := sim.setCallPriceDefaults(&call, baseFee)

	require.NoError(t, err)
	require.NotNil(t, call.MaxFeePerGas)
	require.NotNil(t, call.MaxPriorityFeePerGas)
	require.Equal(t, big.NewInt(0), call.MaxFeePerGas.ToInt())
	require.Equal(t, big.NewInt(0), call.MaxPriorityFeePerGas.ToInt())
	require.Nil(t, call.GasPrice)
}

func TestSetCallPriceDefaults_SetsLegacyGasPriceWhenNoBaseFee(t *testing.T) {
	sim := &simulator{chainConfig: newSimTestChainConfig()}
	call := TransactionArgs{}

	err := sim.setCallPriceDefaults(&call, nil)

	require.NoError(t, err)
	require.NotNil(t, call.GasPrice)
	require.Equal(t, big.NewInt(0), call.GasPrice.ToInt())
	require.Nil(t, call.MaxFeePerGas)
	require.Nil(t, call.MaxPriorityFeePerGas)
}

func TestSetCallPriceDefaults_SetsChainIDWhenMissing(t *testing.T) {
	chainConfig := newSimTestChainConfig()
	sim := &simulator{chainConfig: chainConfig}
	call := TransactionArgs{}

	err := sim.setCallPriceDefaults(&call, nil)

	require.NoError(t, err)
	require.NotNil(t, call.ChainID)
	require.Equal(t, chainConfig.ChainID, call.ChainID.ToInt())
}

func TestSetCallPriceDefaults_SetsZeroValueWhenMissing(t *testing.T) {
	sim := &simulator{chainConfig: newSimTestChainConfig()}
	call := TransactionArgs{}

	err := sim.setCallPriceDefaults(&call, nil)

	require.NoError(t, err)
	require.NotNil(t, call.Value)
	require.Equal(t, big.NewInt(0), call.Value.ToInt())
}

func TestSetCallPriceDefaults_ErrorOnConflictingGasPriceFields(t *testing.T) {
	sim := &simulator{chainConfig: newSimTestChainConfig()}
	gasPrice := hexutil.Big(*big.NewInt(100))
	maxFee := hexutil.Big(*big.NewInt(200))
	call := TransactionArgs{
		GasPrice:     &gasPrice,
		MaxFeePerGas: &maxFee,
	}

	err := sim.setCallPriceDefaults(&call, nil)

	require.Error(t, err)
}

func TestSimulateV1_RejectsEmptyBlockStateCalls(t *testing.T) {
	ctrl := gomock.NewController(t)
	api := NewPublicBlockChainAPI(NewMockBackend(ctrl))

	_, err := api.SimulateV1(context.Background(), simOpts{}, nil)

	simTxError, ok := err.(*simInvalidTxError)
	require.True(t, ok)
	require.Equal(t, errCodeInvalidParams, simTxError.ErrorCode())
}

func TestSimulateV1_RejectsTooManyBlockStateCalls(t *testing.T) {
	ctrl := gomock.NewController(t)
	api := NewPublicBlockChainAPI(NewMockBackend(ctrl))

	blocks := make([]simBlock, maxSimulateBlocks+1)
	_, err := api.SimulateV1(context.Background(), simOpts{BlockStateCalls: blocks}, nil)

	simTxError, ok := err.(*simInvalidTxError)
	require.True(t, ok)
	require.Equal(t, errCodeClientLimitExceeded, simTxError.ErrorCode())
}

func TestSimulateV1_TooManyCallsInBlock_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	api := NewPublicBlockChainAPI(NewMockBackend(ctrl))

	opts := simOpts{BlockStateCalls: []simBlock{{Calls: make([]TransactionArgs, maxSimulateCallsPerBlock+1)}}}
	_, err := api.SimulateV1(context.Background(), opts, nil)
	require.ErrorContains(t, err, "too many calls in block")
}

func TestSimulateV1_TooManyCallsInTotal_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	api := NewPublicBlockChainAPI(NewMockBackend(ctrl))

	var blocks []simBlock
	for i := 0; i < 3; i++ {
		blocks = append(blocks, simBlock{Calls: make([]TransactionArgs, maxSimulateCallsPerBlock)})
	}
	_, err := api.SimulateV1(context.Background(), simOpts{BlockStateCalls: blocks}, nil)
	require.ErrorContains(t, err, "too many calls:")
}
