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
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	baseBlockNumber = 10
)

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

// newSimulateV1Helper sets up the common mocks and returns a fixture for SimulateV1 tests.
func newSimulateV1Helper(t *testing.T) *simulateV1Helper {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockBackend := NewMockBackend(ctrl)
	mockState := state.NewMockStateDB(ctrl)

	baseBlock := newSimTestBase()
	blkNr := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)

	mockBackend.EXPECT().StateAndBlockByNumberOrHash(gomock.Any(), blkNr).Return(mockState, baseBlock, nil)
	mockBackend.EXPECT().RPCGasCap().Return(uint64(10_000_000))
	mockBackend.EXPECT().ChainConfig(idx.Block(10)).Return(newSimTestChainConfig())
	mockBackend.EXPECT().RPCEVMTimeout().Return(time.Duration(0))

	return &simulateV1Helper{
		api:       NewPublicBlockChainAPI(mockBackend),
		mockState: mockState,
		blkNr:     blkNr,
	}
}

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

func TestSimTracer_CaptureTransfer_CreatesExpectedLog(t *testing.T) {
	from := common.Address{1}
	to := common.Address{2}
	value := big.NewInt(1_000_000)

	blockNum := uint64(5)
	blockTs := uint64(1000)
	blockHash := common.Hash{3}
	txHash := common.Hash{4}

	tracer := newSimTracer(true, blockNum, blockTs, blockHash, txHash, 0)
	// Simulate onEnter allocating the per-call slice.
	tracer.logs = append(tracer.logs, make([]*types.Log, 0))

	tracer.captureTransfer(from, to, value)

	require.Len(t, tracer.logs[0], 1)
	log := tracer.logs[0][0]
	require.Equal(t, simTransferAddress, log.Address)
	require.Equal(t, simTransferTopic, log.Topics[0])
	require.Equal(t, common.BytesToHash(from.Bytes()), log.Topics[1])
	require.Equal(t, common.BytesToHash(to.Bytes()), log.Topics[2])
	require.Equal(t, blockNum, log.BlockNumber)
	require.Equal(t, blockTs, log.BlockTimestamp)
	require.Equal(t, blockHash, log.BlockHash)
	require.Equal(t, txHash, log.TxHash)
}

func TestSimTracer_CaptureTransfer_NoOpWhenTracingDisabled(t *testing.T) {
	tracer := newSimTracer(false, 1, 0, common.Hash{}, common.Hash{}, 0)
	tracer.logs = append(tracer.logs, make([]*types.Log, 0))

	tracer.captureTransfer(common.Address{1}, common.Address{2}, big.NewInt(1))

	// No log should be appended because traceTransfers is false.
	require.Empty(t, tracer.logs[0])
}

func TestSimTracer_OnExit_ClearsLogsOnTopLevelRevert(t *testing.T) {
	tracer := newSimTracer(true, 1, 0, common.Hash{}, common.Hash{}, 0)
	tracer.logs = [][]*types.Log{
		{{Address: common.Address{1}}},
	}
	tracer.count = 1

	tracer.onExit(0, nil, 0, nil, true) // depth=0, reverted=true

	require.Nil(t, tracer.logs[0])
	require.Equal(t, 0, tracer.count)
}

func TestSimTracer_OnExit_KeepsLogsOnSuccessfulTopLevelCall(t *testing.T) {
	tracer := newSimTracer(true, 1, 0, common.Hash{}, common.Hash{}, 0)
	log := &types.Log{Address: common.Address{1}}
	tracer.logs = [][]*types.Log{{log}}
	tracer.count = 1

	tracer.onExit(0, nil, 0, nil, false) // depth=0, reverted=false

	require.NotNil(t, tracer.logs[0])
	require.Equal(t, 1, tracer.count)
}

func TestSimTracer_OnExit_KeepsLogsOnNestedRevert(t *testing.T) {
	tracer := newSimTracer(true, 1, 0, common.Hash{}, common.Hash{}, 0)
	log := &types.Log{Address: common.Address{1}}
	tracer.logs = [][]*types.Log{{log}}
	tracer.count = 1

	// Depth > 0 revert should not clear logs.
	tracer.onExit(1, nil, 0, nil, true)

	require.NotNil(t, tracer.logs[0])
	require.Equal(t, 1, tracer.count)
}

func TestSimTracer_Reset_ClearsLogsAndUpdatesContext(t *testing.T) {
	tracer := newSimTracer(true, 1, 0, common.Hash{1}, common.Hash{2}, 0)
	tracer.logs = [][]*types.Log{{{Address: common.Address{1}}}}

	newHash := common.Hash{42}
	newIdx := uint(3)
	tracer.reset(newHash, newIdx)

	require.Nil(t, tracer.logs)
	require.Equal(t, newHash, tracer.txHash)
	require.Equal(t, newIdx, tracer.txIdx)
}

func TestSimTracer_Hooks_ReturnsNilWhenTracingDisabled(t *testing.T) {
	tracer := newSimTracer(false, 1, 0, common.Hash{}, common.Hash{}, 0)
	require.Nil(t, tracer.Hooks())
}

func TestSimTracer_Hooks_ReturnsNonNilWhenTracingEnabled(t *testing.T) {
	tracer := newSimTracer(true, 1, 0, common.Hash{}, common.Hash{}, 0)
	require.NotNil(t, tracer.Hooks())
}

func TestMakeEvmHeader_NilOverride_ReturnsCopyOfTemplate(t *testing.T) {
	var override *simBlockOverrides
	template := &evmcore.EvmHeader{
		Number:   big.NewInt(5),
		GasLimit: 500_000,
	}

	result := override.makeEvmHeader(template)

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

	result := override.makeEvmHeader(template)

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
		Number:   big.NewInt(42),
		GasLimit: 999_999,
		Coinbase: common.Address{5},
	}

	result := override.makeEvmHeader(template)

	// Unset override fields should leave template values intact.
	require.Equal(t, template.Number, result.Number)
	require.Equal(t, template.GasLimit, result.GasLimit)
	require.Equal(t, template.Coinbase, result.Coinbase)
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
	n2 := hexutil.Big(*big.NewInt(baseBlockNumber + 2)) // same as previous (non-increasing)
	blocks := []simBlock{
		{BlockOverrides: &simBlockOverrides{Number: &n1}},
		{BlockOverrides: &simBlockOverrides{Number: &n2}},
	}

	_, err := sim.sanitizeChain(blocks)

	var target *simInvalidBlockNumberError
	require.True(t, errors.As(err, &target))
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

	var target *simInvalidBlockTimestampError
	require.True(t, errors.As(err, &target))
}

func TestSanitizeChain_RejectsBlockRangeExceedingLimit(t *testing.T) {
	sim := newSimulator()
	// Request a block more than maxSimulateBlocks beyond the base.
	tooFar := new(big.Int).Add(sim.base.Number, big.NewInt(maxSimulateBlocks+1))
	n := hexutil.Big(*tooFar)
	blocks := []simBlock{{BlockOverrides: &simBlockOverrides{Number: &n}}}

	_, err := sim.sanitizeChain(blocks)

	var target *simClientLimitExceededError
	require.True(t, errors.As(err, &target))
}

func TestSanitizeChain_SetsEmptyWithdrawalsOnEachBlock(t *testing.T) {
	sim := newSimulator()
	blocks := []simBlock{{}}

	result, err := sim.sanitizeChain(blocks)

	require.NoError(t, err)
	require.NotNil(t, result[0].BlockOverrides.Withdrawals)
}

func TestRepairSimLogs_UpdatesBlockHashInContractLogs(t *testing.T) {
	blockHash := common.Hash{42}
	log1 := &types.Log{BlockHash: common.Hash{}}
	log2 := &types.Log{BlockHash: common.Hash{}}

	calls := []simCallResult{}
	allLogs := []*types.Log{log1, log2}

	repairSimLogs(calls, allLogs, blockHash)

	require.Equal(t, blockHash, log1.BlockHash)
	require.Equal(t, blockHash, log2.BlockHash)
}

func TestRepairSimLogs_UpdatesBlockHashInTransferLogs(t *testing.T) {
	blockHash := common.Hash{99}
	transferLog := &types.Log{BlockHash: common.Hash{}}

	calls := []simCallResult{
		{Logs: []*types.Log{transferLog}},
	}

	repairSimLogs(calls, nil, blockHash)

	require.Equal(t, blockHash, transferLog.BlockHash)
}

func TestRepairSimLogs_UpdatesBothContractAndTransferLogs(t *testing.T) {
	blockHash := common.Hash{77}
	contractLog := &types.Log{BlockHash: common.Hash{}}
	transferLog := &types.Log{BlockHash: common.Hash{}}

	calls := []simCallResult{
		{Logs: []*types.Log{transferLog}},
	}
	allLogs := []*types.Log{contractLog}

	repairSimLogs(calls, allLogs, blockHash)

	require.Equal(t, blockHash, contractLog.BlockHash)
	require.Equal(t, blockHash, transferLog.BlockHash)
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

	var target *simInvalidParamsError
	require.True(t, errors.As(err, &target))
}

func TestSimulateV1_RejectsTooManyBlockStateCalls(t *testing.T) {
	ctrl := gomock.NewController(t)
	api := NewPublicBlockChainAPI(NewMockBackend(ctrl))

	blocks := make([]simBlock, maxSimulateBlocks+1)
	_, err := api.SimulateV1(context.Background(), simOpts{BlockStateCalls: blocks}, nil)

	var target *simClientLimitExceededError
	require.True(t, errors.As(err, &target))
}
