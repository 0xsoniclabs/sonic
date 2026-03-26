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
	"github.com/ethereum/go-ethereum/core/tracing"
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

	setup, cancel, err := setupTracedEVM(t.Context(), backend, traceTestBlock(), mockState, 0, TraceOptions{}, false)
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

	setup, cancel, err := setupTracedEVM(t.Context(), backend, traceTestBlock(), mockState, 0, TraceOptions{}, false)
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

			setup, cancel, err := setupTracedEVM(t.Context(), backend, traceTestBlock(), mockState, 0, TraceOptions{Trace: tt.wantTrace, StateDiff: tt.wantStateDiff}, false)
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

			setup, cancel, err := setupTracedEVM(t.Context(), backend, traceTestBlock(), mockState, 0, TraceOptions{}, tt.noBaseFee)
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

	setup, cancel, err := setupTracedEVM(t.Context(), backend, traceTestBlock(), mockState, 0, TraceOptions{}, false)
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

	setup, cancel, err := setupTracedEVM(t.Context(), backend, traceTestBlock(), mockState, 0, TraceOptions{StateDiff: true}, false)
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
		TraceOptions{Trace: true},
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
		TraceOptions{Trace: true},
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Trace, "trace must contain at least one action")
	require.Nil(t, result.StateDiff, "stateDiff must be nil when not requested")
	require.Nil(t, result.VmTrace, "vmTrace must be nil when not requested")
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
		TraceOptions{StateDiff: true},
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Nil(t, result.Trace, "trace must be nil when not requested")
	require.NotNil(t, result.StateDiff, "stateDiff must be non-nil when requested")
	require.Nil(t, result.VmTrace, "vmTrace must be nil when not requested")
}

func TestTraceCallExec_VMTraceOnly(t *testing.T) {
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
		TraceOptions{VmTrace: true},
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Nil(t, result.Trace, "trace must be nil when not requested")
	require.Nil(t, result.StateDiff, "stateDiff must be nil when not requested")
	require.NotNil(t, result.VmTrace, "vmTrace must be non-nil when requested")
}

func TestTraceCallExec_AllTypes(t *testing.T) {
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
		TraceOptions{Trace: true, StateDiff: true, VmTrace: true},
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Trace, "trace must contain at least one action")
	require.NotNil(t, result.StateDiff, "stateDiff must be non-nil when requested")
	require.NotNil(t, result.VmTrace, "vmTrace must be non-nil when requested")
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

// recorder accumulates string tokens so tests can assert call order.
type recorder struct{ calls []string }

func (r *recorder) add(s string) { r.calls = append(r.calls, s) }

func TestMergeVMHooks_BothNil(t *testing.T) {
	require.Nil(t, mergeVMHooks(nil, nil))
}

func TestMergeVMHooks_OrigNil_ReturnsNewHooks(t *testing.T) {
	newHooks := &tracing.Hooks{}
	require.Same(t, newHooks, mergeVMHooks(nil, newHooks))
}

func TestMergeVMHooks_NewHooksNil_ReturnsOrig(t *testing.T) {
	orig := &tracing.Hooks{}
	require.Same(t, orig, mergeVMHooks(orig, nil))
}

func TestMergeVMHooks_BothNonNil_ReturnsNewPointer(t *testing.T) {
	orig := &tracing.Hooks{}
	newHooks := &tracing.Hooks{}
	result := mergeVMHooks(orig, newHooks)
	require.NotNil(t, result)
	require.NotSame(t, orig, result)
	require.NotSame(t, newHooks, result)
}

func TestMergeVMHooks_OnTxStart(t *testing.T) {
	invoke := func(h *tracing.Hooks) {
		h.OnTxStart(nil, nil, common.Address{})
	}

	t.Run("orig only is preserved", func(t *testing.T) {
		rec := &recorder{}
		orig := &tracing.Hooks{OnTxStart: func(_ *tracing.VMContext, _ *types.Transaction, _ common.Address) { rec.add("orig") }}
		result := mergeVMHooks(orig, &tracing.Hooks{})
		require.NotNil(t, result.OnTxStart)
		invoke(result)
		require.Equal(t, []string{"orig"}, rec.calls)
	})

	t.Run("new only is installed", func(t *testing.T) {
		rec := &recorder{}
		newHooks := &tracing.Hooks{OnTxStart: func(_ *tracing.VMContext, _ *types.Transaction, _ common.Address) { rec.add("new") }}
		result := mergeVMHooks(&tracing.Hooks{}, newHooks)
		require.NotNil(t, result.OnTxStart)
		invoke(result)
		require.Equal(t, []string{"new"}, rec.calls)
	})

	t.Run("both: orig called first then new", func(t *testing.T) {
		rec := &recorder{}
		orig := &tracing.Hooks{OnTxStart: func(_ *tracing.VMContext, _ *types.Transaction, _ common.Address) { rec.add("orig") }}
		newHooks := &tracing.Hooks{OnTxStart: func(_ *tracing.VMContext, _ *types.Transaction, _ common.Address) { rec.add("new") }}
		invoke(mergeVMHooks(orig, newHooks))
		require.Equal(t, []string{"orig", "new"}, rec.calls)
	})
}

func TestMergeVMHooks_OnTxEnd(t *testing.T) {
	invoke := func(h *tracing.Hooks) { h.OnTxEnd(nil, nil) }

	t.Run("orig only is preserved", func(t *testing.T) {
		rec := &recorder{}
		orig := &tracing.Hooks{OnTxEnd: func(_ *types.Receipt, _ error) { rec.add("orig") }}
		result := mergeVMHooks(orig, &tracing.Hooks{})
		require.NotNil(t, result.OnTxEnd)
		invoke(result)
		require.Equal(t, []string{"orig"}, rec.calls)
	})

	t.Run("new only is installed", func(t *testing.T) {
		rec := &recorder{}
		newHooks := &tracing.Hooks{OnTxEnd: func(_ *types.Receipt, _ error) { rec.add("new") }}
		result := mergeVMHooks(&tracing.Hooks{}, newHooks)
		require.NotNil(t, result.OnTxEnd)
		invoke(result)
		require.Equal(t, []string{"new"}, rec.calls)
	})

	t.Run("both: orig called first then new", func(t *testing.T) {
		rec := &recorder{}
		orig := &tracing.Hooks{OnTxEnd: func(_ *types.Receipt, _ error) { rec.add("orig") }}
		newHooks := &tracing.Hooks{OnTxEnd: func(_ *types.Receipt, _ error) { rec.add("new") }}
		invoke(mergeVMHooks(orig, newHooks))
		require.Equal(t, []string{"orig", "new"}, rec.calls)
	})
}

func TestMergeVMHooks_OnEnter(t *testing.T) {
	invoke := func(h *tracing.Hooks) {
		h.OnEnter(0, 0, common.Address{}, common.Address{}, nil, 0, big.NewInt(0))
	}

	t.Run("orig only is preserved", func(t *testing.T) {
		rec := &recorder{}
		orig := &tracing.Hooks{OnEnter: func(_ int, _ byte, _, _ common.Address, _ []byte, _ uint64, _ *big.Int) { rec.add("orig") }}
		result := mergeVMHooks(orig, &tracing.Hooks{})
		require.NotNil(t, result.OnEnter)
		invoke(result)
		require.Equal(t, []string{"orig"}, rec.calls)
	})

	t.Run("new only is installed", func(t *testing.T) {
		rec := &recorder{}
		newHooks := &tracing.Hooks{OnEnter: func(_ int, _ byte, _, _ common.Address, _ []byte, _ uint64, _ *big.Int) { rec.add("new") }}
		result := mergeVMHooks(&tracing.Hooks{}, newHooks)
		require.NotNil(t, result.OnEnter)
		invoke(result)
		require.Equal(t, []string{"new"}, rec.calls)
	})

	t.Run("both: orig called first then new", func(t *testing.T) {
		rec := &recorder{}
		orig := &tracing.Hooks{OnEnter: func(_ int, _ byte, _, _ common.Address, _ []byte, _ uint64, _ *big.Int) { rec.add("orig") }}
		newHooks := &tracing.Hooks{OnEnter: func(_ int, _ byte, _, _ common.Address, _ []byte, _ uint64, _ *big.Int) { rec.add("new") }}
		invoke(mergeVMHooks(orig, newHooks))
		require.Equal(t, []string{"orig", "new"}, rec.calls)
	})
}

func TestMergeVMHooks_OnExit(t *testing.T) {
	invoke := func(h *tracing.Hooks) { h.OnExit(0, nil, 0, nil, false) }

	t.Run("orig only is preserved", func(t *testing.T) {
		rec := &recorder{}
		orig := &tracing.Hooks{OnExit: func(_ int, _ []byte, _ uint64, _ error, _ bool) { rec.add("orig") }}
		result := mergeVMHooks(orig, &tracing.Hooks{})
		require.NotNil(t, result.OnExit)
		invoke(result)
		require.Equal(t, []string{"orig"}, rec.calls)
	})

	t.Run("new only is installed", func(t *testing.T) {
		rec := &recorder{}
		newHooks := &tracing.Hooks{OnExit: func(_ int, _ []byte, _ uint64, _ error, _ bool) { rec.add("new") }}
		result := mergeVMHooks(&tracing.Hooks{}, newHooks)
		require.NotNil(t, result.OnExit)
		invoke(result)
		require.Equal(t, []string{"new"}, rec.calls)
	})

	t.Run("both: orig called first then new", func(t *testing.T) {
		rec := &recorder{}
		orig := &tracing.Hooks{OnExit: func(_ int, _ []byte, _ uint64, _ error, _ bool) { rec.add("orig") }}
		newHooks := &tracing.Hooks{OnExit: func(_ int, _ []byte, _ uint64, _ error, _ bool) { rec.add("new") }}
		invoke(mergeVMHooks(orig, newHooks))
		require.Equal(t, []string{"orig", "new"}, rec.calls)
	})
}

func TestMergeVMHooks_OnOpcode(t *testing.T) {
	invoke := func(h *tracing.Hooks) { h.OnOpcode(0, 0, 0, 0, nil, nil, 0, nil) }

	t.Run("orig only is preserved", func(t *testing.T) {
		rec := &recorder{}
		orig := &tracing.Hooks{OnOpcode: func(_ uint64, _ byte, _, _ uint64, _ tracing.OpContext, _ []byte, _ int, _ error) { rec.add("orig") }}
		result := mergeVMHooks(orig, &tracing.Hooks{})
		require.NotNil(t, result.OnOpcode)
		invoke(result)
		require.Equal(t, []string{"orig"}, rec.calls)
	})

	t.Run("new only is installed", func(t *testing.T) {
		rec := &recorder{}
		newHooks := &tracing.Hooks{OnOpcode: func(_ uint64, _ byte, _, _ uint64, _ tracing.OpContext, _ []byte, _ int, _ error) { rec.add("new") }}
		result := mergeVMHooks(&tracing.Hooks{}, newHooks)
		require.NotNil(t, result.OnOpcode)
		invoke(result)
		require.Equal(t, []string{"new"}, rec.calls)
	})

	t.Run("both: orig called first then new", func(t *testing.T) {
		rec := &recorder{}
		orig := &tracing.Hooks{OnOpcode: func(_ uint64, _ byte, _, _ uint64, _ tracing.OpContext, _ []byte, _ int, _ error) { rec.add("orig") }}
		newHooks := &tracing.Hooks{OnOpcode: func(_ uint64, _ byte, _, _ uint64, _ tracing.OpContext, _ []byte, _ int, _ error) { rec.add("new") }}
		invoke(mergeVMHooks(orig, newHooks))
		require.Equal(t, []string{"orig", "new"}, rec.calls)
	})
}

func TestMergeVMHooks_OnFault(t *testing.T) {
	invoke := func(h *tracing.Hooks) { h.OnFault(0, 0, 0, 0, nil, 0, nil) }

	t.Run("orig only is preserved", func(t *testing.T) {
		rec := &recorder{}
		orig := &tracing.Hooks{OnFault: func(_ uint64, _ byte, _, _ uint64, _ tracing.OpContext, _ int, _ error) { rec.add("orig") }}
		result := mergeVMHooks(orig, &tracing.Hooks{})
		require.NotNil(t, result.OnFault)
		invoke(result)
		require.Equal(t, []string{"orig"}, rec.calls)
	})

	t.Run("new only is installed", func(t *testing.T) {
		rec := &recorder{}
		newHooks := &tracing.Hooks{OnFault: func(_ uint64, _ byte, _, _ uint64, _ tracing.OpContext, _ int, _ error) { rec.add("new") }}
		result := mergeVMHooks(&tracing.Hooks{}, newHooks)
		require.NotNil(t, result.OnFault)
		invoke(result)
		require.Equal(t, []string{"new"}, rec.calls)
	})

	t.Run("both: orig called first then new", func(t *testing.T) {
		rec := &recorder{}
		orig := &tracing.Hooks{OnFault: func(_ uint64, _ byte, _, _ uint64, _ tracing.OpContext, _ int, _ error) { rec.add("orig") }}
		newHooks := &tracing.Hooks{OnFault: func(_ uint64, _ byte, _, _ uint64, _ tracing.OpContext, _ int, _ error) { rec.add("new") }}
		invoke(mergeVMHooks(orig, newHooks))
		require.Equal(t, []string{"orig", "new"}, rec.calls)
	})
}

func TestMergeVMHooks_UnrelatedHooksFromOrigArePreserved(t *testing.T) {
	// newHooks only contributes OnTxStart; all other hooks from orig must survive unchanged.
	origOnTxEndCalled := false
	origOnEnterCalled := false

	orig := &tracing.Hooks{
		OnTxEnd: func(_ *types.Receipt, _ error) { origOnTxEndCalled = true },
		OnEnter: func(_ int, _ byte, _, _ common.Address, _ []byte, _ uint64, _ *big.Int) { origOnEnterCalled = true },
	}
	newHooks := &tracing.Hooks{
		OnTxStart: func(_ *tracing.VMContext, _ *types.Transaction, _ common.Address) {},
	}

	result := mergeVMHooks(orig, newHooks)

	result.OnTxEnd(nil, nil)
	result.OnEnter(0, 0, common.Address{}, common.Address{}, nil, 0, big.NewInt(0))

	require.True(t, origOnTxEndCalled, "OnTxEnd from orig must be preserved when newHooks does not define it")
	require.True(t, origOnEnterCalled, "OnEnter from orig must be preserved when newHooks does not define it")
}
