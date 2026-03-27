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
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// traceTestBlock returns a minimal EvmBlock at height 1 (0 is genesis, has special semantics).
func traceTestBlock() *evmcore.EvmBlock {
	return &evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{
			Number:  big.NewInt(1),
			BaseFee: big.NewInt(1_000_000_000),
		},
	}
}

// traceTestMessage returns a minimal native token transfer message with zero gas price.
// Using GasLimit=21000 (exact intrinsic gas) and all fees=0 keeps the test state simple.
func traceTestMessage(from, to common.Address) *core.Message {
	return &core.Message{
		From:      from,
		To:        &to,
		GasLimit:  21000,
		GasPrice:  new(big.Int),
		GasFeeCap: new(big.Int),
		GasTipCap: new(big.Int),
		Value:     new(big.Int),
	}
}

// traceTestTx returns a minimal legacy transaction for use as the tx parameter in traceCallExec.
func traceTestTx() *types.Transaction {
	return types.NewTx(&types.LegacyTx{Gas: 21000})
}

// setupBackendForTracing returns a MockBackend pre-configured with the expectations
// needed for setupTracedEVM: GetNetworkRules, RPCEVMTimeout, and GetEVM.
// GetEVM uses getEvmFunc so the EVM is real but keeps a minimal block context.
func setupBackendForTracing(ctrl *gomock.Controller, mockState *state.MockStateDB) *MockBackend {
	backend := NewMockBackend(ctrl)
	backend.EXPECT().GetNetworkRules(gomock.Any(), gomock.Any()).Return(&opera.Rules{}, nil).AnyTimes()
	backend.EXPECT().RPCEVMTimeout().Return(time.Duration(0)).AnyTimes()
	backend.EXPECT().GetEVM(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(getEvmFunc(mockState)).AnyTimes()
	return backend
}

func TestSetupTracedEVM_GetVmConfigError(t *testing.T) {
	ctrl := gomock.NewController(t)
	backend := NewMockBackend(ctrl)
	mockState := state.NewMockStateDB(ctrl)

	injected := fmt.Errorf("db unavailable")
	backend.EXPECT().GetNetworkRules(gomock.Any(), gomock.Any()).Return(nil, injected)

	setup, cancel, err := setupTracedEVM(t.Context(), backend, traceTestBlock(), mockState, 0, false, false, false)
	defer cancel()

	require.ErrorIs(t, err, injected, "error from GetVmConfig must be propagated")
	require.Nil(t, setup)
}

func TestSetupTracedEVM_GetEVMError(t *testing.T) {
	ctrl := gomock.NewController(t)
	backend := NewMockBackend(ctrl)
	mockState := state.NewMockStateDB(ctrl)

	injected := fmt.Errorf("evm factory failed")
	backend.EXPECT().GetNetworkRules(gomock.Any(), gomock.Any()).Return(&opera.Rules{}, nil)
	backend.EXPECT().RPCEVMTimeout().Return(time.Duration(0))
	backend.EXPECT().GetEVM(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, nil, injected)

	setup, cancel, err := setupTracedEVM(t.Context(), backend, traceTestBlock(), mockState, 0, false, false, false)
	defer cancel()

	require.ErrorContains(t, err, injected.Error(), "error from GetEVM must be propagated")
	require.Nil(t, setup)
}

func TestSetupTracedEVM_LoggersRespectFlags(t *testing.T) {
	tests := []struct {
		name              string
		wantTrace         bool
		wantStateDiff     bool
		expectTxTracer    bool
		expectStateDiff   bool
		expectTracerHooks bool
	}{
		{
			name:              "no tracing",
			wantTrace:         false,
			wantStateDiff:     false,
			expectTxTracer:    false,
			expectStateDiff:   false,
			expectTracerHooks: false,
		},
		{
			name:              "trace only",
			wantTrace:         true,
			wantStateDiff:     false,
			expectTxTracer:    true,
			expectStateDiff:   false,
			expectTracerHooks: true,
		},
		{
			name:              "stateDiff only",
			wantTrace:         false,
			wantStateDiff:     true,
			expectTxTracer:    false,
			expectStateDiff:   true,
			expectTracerHooks: true, // OnEnter hook is added for SELFDESTRUCT detection
		},
		{
			name:              "trace and stateDiff",
			wantTrace:         true,
			wantStateDiff:     true,
			expectTxTracer:    true,
			expectStateDiff:   true,
			expectTracerHooks: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockState := state.NewMockStateDB(ctrl)
			backend := setupBackendForTracing(ctrl, mockState)

			setup, cancel, err := setupTracedEVM(t.Context(), backend, traceTestBlock(), mockState, 0, tt.wantTrace, tt.wantStateDiff, false)
			defer cancel()

			require.NoError(t, err)
			require.NotNil(t, setup)
			require.Equal(t, tt.expectTxTracer, setup.txTracer != nil, "txTracer presence")
			require.Equal(t, tt.expectStateDiff, setup.stateDiffLogger != nil, "stateDiffLogger presence")
			require.Equal(t, tt.expectTracerHooks, setup.tracer != nil, "tracer hooks presence")
		})
	}
}

func TestSetupTracedEVM_NoBaseFeeFlag(t *testing.T) {
	tests := []struct {
		name          string
		noBaseFee     bool
		wantNoBaseFee bool
	}{
		{"noBaseFee=true skips base fee validation", true, true},
		{"noBaseFee=false preserves base fee validation", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockState := state.NewMockStateDB(ctrl)
			backend := NewMockBackend(ctrl)

			backend.EXPECT().GetNetworkRules(gomock.Any(), gomock.Any()).Return(&opera.Rules{}, nil)
			backend.EXPECT().RPCEVMTimeout().Return(time.Duration(0))

			var capturedNoBaseFee bool
			backend.EXPECT().GetEVM(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, statedb vm.StateDB, header *evmcore.EvmHeader, vmConfig *vm.Config, blockContext *vm.BlockContext) (*vm.EVM, func() error, error) {
					capturedNoBaseFee = vmConfig.NoBaseFee
					return makeTestEVM(opera.Upgrades{})(t.Context(), statedb, header, vmConfig, blockContext)
				})

			setup, cancel, err := setupTracedEVM(t.Context(), backend, traceTestBlock(), mockState, 0, false, false, tt.noBaseFee)
			defer cancel()

			require.NoError(t, err)
			require.NotNil(t, setup)
			require.Equal(t, tt.wantNoBaseFee, capturedNoBaseFee)
		})
	}
}

func TestSetupTracedEVM_ContextHasDeadlineFromRPCTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockState := state.NewMockStateDB(ctrl)
	backend := NewMockBackend(ctrl)

	customTimeout := 500 * time.Millisecond
	backend.EXPECT().GetNetworkRules(gomock.Any(), gomock.Any()).Return(&opera.Rules{}, nil)
	// RPCEVMTimeout is called twice in setupTracedEVM: once for the >0 check, once to read the value.
	backend.EXPECT().RPCEVMTimeout().Return(customTimeout).AnyTimes()

	before := time.Now()
	var capturedDeadline time.Time
	backend.EXPECT().GetEVM(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, statedb vm.StateDB, header *evmcore.EvmHeader, vmConfig *vm.Config, blockContext *vm.BlockContext) (*vm.EVM, func() error, error) {
			deadline, ok := ctx.Deadline()
			require.True(t, ok, "context passed to GetEVM must have a deadline")
			capturedDeadline = deadline
			return makeTestEVM(opera.Upgrades{})(ctx, statedb, header, vmConfig, blockContext)
		})

	setup, cancel, err := setupTracedEVM(t.Context(), backend, traceTestBlock(), mockState, 0, false, false, false)
	defer cancel()

	require.NoError(t, err)
	require.NotNil(t, setup)

	// Deadline must be in [before+timeout, before+2*timeout] — generous window to avoid flakiness.
	require.True(t, capturedDeadline.After(before), "deadline must be in the future")
	require.True(t, capturedDeadline.Before(before.Add(2*customTimeout)), "deadline must not be too far in the future")
}

func TestSetupTracedEVM_StateDiffWrapsState(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockState := state.NewMockStateDB(ctrl)
	backend := NewMockBackend(ctrl)

	backend.EXPECT().GetNetworkRules(gomock.Any(), gomock.Any()).Return(&opera.Rules{}, nil)
	backend.EXPECT().RPCEVMTimeout().Return(time.Duration(0))

	var evmReceivedState vm.StateDB
	backend.EXPECT().GetEVM(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, statedb vm.StateDB, header *evmcore.EvmHeader, vmConfig *vm.Config, blockContext *vm.BlockContext) (*vm.EVM, func() error, error) {
			evmReceivedState = statedb
			return makeTestEVM(opera.Upgrades{})(t.Context(), statedb, header, vmConfig, blockContext)
		})

	setup, cancel, err := setupTracedEVM(t.Context(), backend, traceTestBlock(), mockState, 0, false, true, false)
	defer cancel()

	require.NoError(t, err)
	// The state passed to GetEVM must be the wrapped state, not the original mockState.
	require.NotNil(t, evmReceivedState)
	require.NotEqual(t, mockState, evmReceivedState, "GetEVM must receive the wrapped state, not the raw MockStateDB")
	// The setup's activeState must also be the wrapped state.
	require.Equal(t, evmReceivedState, setup.activeState)
}

func TestTraceCallExec_VmConfigError(t *testing.T) {
	ctrl := gomock.NewController(t)
	backend := NewMockBackend(ctrl)

	injected := fmt.Errorf("network rules unavailable")
	backend.EXPECT().GetNetworkRules(gomock.Any(), gomock.Any()).Return(nil, injected)

	api := &PublicTxTraceAPI{b: backend}
	from := common.Address{1}
	to := common.Address{2}
	mockState := state.NewMockStateDB(ctrl)

	result, err := api.traceCallExec(
		t.Context(),
		traceTestBlock(),
		traceTestMessage(from, to),
		mockState,
		traceTestTx(),
		0,
		true,
		false,
	)

	require.ErrorIs(t, err, injected)
	require.Nil(t, result)
}

func TestTraceCallExec_TraceOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockState := state.NewMockStateDB(ctrl)
	setExpectedStateCalls(mockState)

	backend := NewMockBackend(ctrl)
	backend.EXPECT().GetNetworkRules(gomock.Any(), gomock.Any()).Return(&opera.Rules{}, nil)
	backend.EXPECT().RPCEVMTimeout().Return(time.Duration(0))
	backend.EXPECT().GetEVM(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(makeTestEVM(opera.Upgrades{}))

	api := &PublicTxTraceAPI{b: backend}
	from, to := common.Address{1}, common.Address{2}

	result, err := api.traceCallExec(
		t.Context(),
		traceTestBlock(),
		traceTestMessage(from, to),
		mockState,
		traceTestTx(),
		0,
		true,  // wantTrace
		false, // wantStateDiff
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Trace, "trace must contain at least one action")
	require.Nil(t, result.StateDiff, "stateDiff must be nil when not requested")
}

func TestTraceCallExec_StateDiffOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockState := state.NewMockStateDB(ctrl)
	setExpectedStateCalls(mockState)

	backend := NewMockBackend(ctrl)
	backend.EXPECT().GetNetworkRules(gomock.Any(), gomock.Any()).Return(&opera.Rules{}, nil)
	backend.EXPECT().RPCEVMTimeout().Return(time.Duration(0))
	backend.EXPECT().GetEVM(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(makeTestEVM(opera.Upgrades{}))

	api := &PublicTxTraceAPI{b: backend}
	from, to := common.Address{1}, common.Address{2}

	result, err := api.traceCallExec(
		t.Context(),
		traceTestBlock(),
		traceTestMessage(from, to),
		mockState,
		traceTestTx(),
		0,
		false, // wantTrace
		true,  // wantStateDiff
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Nil(t, result.Trace, "trace must be nil when not requested")
	require.NotNil(t, result.StateDiff, "stateDiff must be non-nil when requested")
}

func TestTraceCallExec_BothTypes(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockState := state.NewMockStateDB(ctrl)
	setExpectedStateCalls(mockState)

	backend := NewMockBackend(ctrl)
	backend.EXPECT().GetNetworkRules(gomock.Any(), gomock.Any()).Return(&opera.Rules{}, nil)
	backend.EXPECT().RPCEVMTimeout().Return(time.Duration(0))
	backend.EXPECT().GetEVM(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(makeTestEVM(opera.Upgrades{}))

	api := &PublicTxTraceAPI{b: backend}
	from, to := common.Address{1}, common.Address{2}

	result, err := api.traceCallExec(
		t.Context(),
		traceTestBlock(),
		traceTestMessage(from, to),
		mockState,
		traceTestTx(),
		0,
		true, // wantTrace
		true, // wantStateDiff
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Trace, "trace must contain at least one action")
	require.NotNil(t, result.StateDiff, "stateDiff must be non-nil when requested")
}

// setupBackendForCallMany returns a MockBackend pre-configured with all expectations
// needed by CallMany: BlockByNumber, StateAndBlockByNumberOrHash, RPCGasCap, and
// everything required by setupTracedEVM (GetNetworkRules, RPCEVMTimeout, GetEVM).
// makeTestEVM is used instead of getEvmFunc so the block context is properly populated
// for the trace struct logger.
func setupBackendForCallMany(t *testing.T) (*MockBackend, *state.MockStateDB) {
	ctrl := gomock.NewController(t)
	mockState := state.NewMockStateDB(ctrl)
	mockState.EXPECT().Release().AnyTimes()
	block := traceTestBlock()
	backend := NewMockBackend(ctrl)
	backend.EXPECT().GetNetworkRules(gomock.Any(), gomock.Any()).Return(&opera.Rules{}, nil).AnyTimes()
	backend.EXPECT().RPCEVMTimeout().Return(time.Duration(0)).AnyTimes()
	backend.EXPECT().GetEVM(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(makeTestEVM(opera.Upgrades{})).AnyTimes()
	backend.EXPECT().BlockByNumber(gomock.Any(), gomock.Any()).Return(block, nil).AnyTimes()
	backend.EXPECT().StateAndBlockByNumberOrHash(gomock.Any(), gomock.Any()).Return(mockState, block, nil).AnyTimes()
	backend.EXPECT().RPCGasCap().Return(uint64(10_000_000)).AnyTimes()
	return backend, mockState
}

func TestCallMany_TraceTypeValidation(t *testing.T) {
	tests := []struct {
		name       string
		calls      []CallRequest
		wantErrMsg string
	}{
		{
			name:       "empty call list succeeds",
			calls:      []CallRequest{},
			wantErrMsg: "",
		},
		{
			name: "unrecognized trace type",
			calls: []CallRequest{
				{Args: TransactionArgs{}, TraceTypes: []string{"unknownType"}},
			},
			wantErrMsg: "unrecognized trace type",
		},
		{
			name: "vmTrace not supported",
			calls: []CallRequest{
				{Args: TransactionArgs{}, TraceTypes: []string{TraceTypeVmTrace}},
			},
			wantErrMsg: "vmTrace trace type is not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, _ := setupBackendForCallMany(t)
			api := &PublicTxTraceAPI{b: backend}

			results, err := api.CallMany(t.Context(), tt.calls, rpc.BlockNumberOrHashWithNumber(1), nil)

			if tt.wantErrMsg == "" {
				require.NoError(t, err)
				require.Empty(t, results)
			} else {
				require.ErrorContains(t, err, tt.wantErrMsg)
			}
		})
	}
}

func TestCallMany_SingleCall_TraceOnly(t *testing.T) {
	backend, mockState := setupBackendForCallMany(t)
	setExpectedStateCalls(mockState)

	api := &PublicTxTraceAPI{b: backend}
	from, to := common.Address{1}, common.Address{2}
	calls := []CallRequest{
		{
			Args:       TransactionArgs{From: &from, To: &to},
			TraceTypes: []string{TraceTypeTrace},
		},
	}

	results, err := api.CallMany(t.Context(), calls, rpc.BlockNumberOrHashWithNumber(1), nil)

	require.NoError(t, err)
	require.Len(t, results, 1)
	require.NotEmpty(t, results[0].Trace, "trace must contain at least one action")
	require.Nil(t, results[0].StateDiff, "stateDiff must be nil when not requested")
}

func TestCallMany_SingleCall_StateDiffOnly(t *testing.T) {
	backend, mockState := setupBackendForCallMany(t)
	setExpectedStateCalls(mockState)

	api := &PublicTxTraceAPI{b: backend}
	from, to := common.Address{1}, common.Address{2}
	calls := []CallRequest{
		{
			Args:       TransactionArgs{From: &from, To: &to},
			TraceTypes: []string{TraceTypeStateDiff},
		},
	}

	results, err := api.CallMany(t.Context(), calls, rpc.BlockNumberOrHashWithNumber(1), nil)

	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Nil(t, results[0].Trace, "trace must be nil when not requested")
	require.NotNil(t, results[0].StateDiff, "stateDiff must be non-nil when requested")
}

func TestCallMany_MultipleCalls_IndependentTraceTypes(t *testing.T) {
	backend, mockState := setupBackendForCallMany(t)
	setExpectedStateCalls(mockState)

	api := &PublicTxTraceAPI{b: backend}
	from, to := common.Address{1}, common.Address{2}
	calls := []CallRequest{
		{
			Args:       TransactionArgs{From: &from, To: &to},
			TraceTypes: []string{TraceTypeTrace},
		},
		{
			Args:       TransactionArgs{From: &from, To: &to},
			TraceTypes: []string{TraceTypeStateDiff},
		},
		{
			Args:       TransactionArgs{From: &from, To: &to},
			TraceTypes: []string{TraceTypeTrace, TraceTypeStateDiff},
		},
	}

	results, err := api.CallMany(t.Context(), calls, rpc.BlockNumberOrHashWithNumber(1), nil)

	require.NoError(t, err)
	require.Len(t, results, 3)

	// Call 0: trace only
	require.NotEmpty(t, results[0].Trace)
	require.Nil(t, results[0].StateDiff)

	// Call 1: stateDiff only
	require.Nil(t, results[1].Trace)
	require.NotNil(t, results[1].StateDiff)

	// Call 2: both
	require.NotEmpty(t, results[2].Trace)
	require.NotNil(t, results[2].StateDiff)
}

func TestCallRequest_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErrMsg string
	}{
		{
			name:       "completely invalid JSON",
			input:      `not-json`,
			wantErrMsg: "each call must be [tx, traceTypes]",
		},
		{
			name:       "JSON object instead of array",
			input:      `{"from": "0x1234"}`,
			wantErrMsg: "each call must be [tx, traceTypes]",
		},
		{
			name:       "JSON string instead of array",
			input:      `"trace"`,
			wantErrMsg: "each call must be [tx, traceTypes]",
		},
		{
			name:       "empty array leaves both slots nil",
			input:      `[]`,
			wantErrMsg: "cannot parse transaction args",
		},
		{
			name:       "array with one element leaves second slot nil",
			input:      `[{}]`,
			wantErrMsg: "cannot parse trace types",
		},
		{
			name:       "first element has invalid field type for TransactionArgs",
			input:      `[{"from": 12345}, ["trace"]]`,
			wantErrMsg: "cannot parse transaction args",
		},
		{
			name:       "second element is an object instead of string array",
			input:      `[{}, {"type": "trace"}]`,
			wantErrMsg: "cannot parse trace types",
		},
		{
			name:       "second element is a string instead of string array",
			input:      `[{}, "trace"]`,
			wantErrMsg: "cannot parse trace types",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r CallRequest
			err := r.UnmarshalJSON([]byte(tt.input))
			require.ErrorContains(t, err, tt.wantErrMsg)
		})
	}
}

func TestCallMany_BlockNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	backend := NewMockBackend(ctrl)
	injected := fmt.Errorf("block not found")
	backend.EXPECT().BlockByNumber(gomock.Any(), gomock.Any()).Return(nil, injected)

	api := &PublicTxTraceAPI{b: backend}
	_, err := api.CallMany(t.Context(), []CallRequest{}, rpc.BlockNumberOrHashWithNumber(99), nil)

	require.ErrorIs(t, err, injected)
}
