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

package trace

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

// fixture serves the trace namespace over the replay chain.
type fixture struct {
	*tests.ReplayChain
	h *tests.Harness
}

func newFixture(t *testing.T, mutate ...func(*dispatch.Config)) *fixture {
	t.Helper()
	r := tests.NewReplayChain(t)
	return &fixture{ReplayChain: r, h: newHarness(t, r.Core, mutate...)}
}

func newHarness(t *testing.T, fc *tests.FakeCore, mutate ...func(*dispatch.Config)) *tests.Harness {
	t.Helper()
	return tests.NewEnvHarness(t, fc, func(d *dispatch.Dispatcher, env *exec.Env) { Register(d, env) }, mutate...)
}

type actionTrace struct {
	Type            string `json:"type"`
	TransactionHash string `json:"transactionHash"`
	Error           string `json:"error"`
	Action          struct {
		From  common.Address `json:"from"`
		To    common.Address `json:"to"`
		Value hexutil.Big    `json:"value"`
	} `json:"action"`
}

func TestTransaction(t *testing.T) {
	f := newFixture(t)
	var traces []actionTrace
	f.h.Into(&traces, "trace_transaction", f.Transfer.Hash())
	require.Len(t, traces, 1)
	require.Equal(t, "call", traces[0].Type)
	require.Equal(t, f.Alice.Addr, traces[0].Action.From)
	require.Equal(t, f.Bob, traces[0].Action.To)
	require.Equal(t, int64(5), traces[0].Action.Value.ToInt().Int64())
	require.Equal(t, f.Transfer.Hash().Hex(), traces[0].TransactionHash)

	// The second transaction needs the first replayed: the state it runs on is
	// the state after.
	f.h.Into(&traces, "trace_transaction", f.Store.Hash())
	require.Len(t, traces, 1)
	require.Equal(t, f.Contract, traces[0].Action.To)
	require.Empty(t, traces[0].Error)
}

func TestTransaction_Unknown(t *testing.T) {
	f := newFixture(t)
	err := f.h.Fails("trace_transaction", common.Hash{9})
	require.Equal(t, "transaction "+common.Hash{9}.String()+" not found", err.Message)
}

func TestBlock(t *testing.T) {
	f := newFixture(t)
	var traces []actionTrace
	f.h.Into(&traces, "trace_block", "latest")
	require.Len(t, traces, 2)
	f.h.Into(&traces, "trace_block", f.Core.BlockAt(f.Block).Hash)
	require.Len(t, traces, 2)

	// An empty block has one pseudo trace; the genesis block cannot be traced.
	f.h.WaitHead(f.Core.AddBlock().NumberU64())
	f.h.Into(&traces, "trace_block", "latest")
	require.Len(t, traces, 1)
	require.Equal(t, "Empty block", traces[0].Error)

	err := f.h.Fails("trace_block", "0x0")
	require.Equal(t, "cannot trace block 0 got genesis block is not traceable", err.Message)
	err = f.h.Fails("trace_block", "0x99")
	require.Equal(t, "cannot trace block 153 got invalid block for tracing", err.Message)
	err = f.h.Fails("trace_block", common.Hash{9})
	require.Equal(t, "block not found", err.Message)
}

func TestGet(t *testing.T) {
	f := newFixture(t)
	var traces []actionTrace
	f.h.Into(&traces, "trace_get", f.Transfer.Hash(), []string{})
	require.Len(t, traces, 1)
	require.Equal(t, f.Transfer.Hash().Hex(), traces[0].TransactionHash)
	// A trace index that is not there answers null.
	require.JSONEq(t, `null`, string(f.h.Result("trace_get", f.Transfer.Hash(), []string{"0x7"})))
}

func TestCall(t *testing.T) {
	f := newFixture(t)
	var res struct {
		Output    hexutil.Bytes  `json:"output"`
		Trace     []actionTrace  `json:"trace"`
		StateDiff map[string]any `json:"stateDiff"`
		VmTrace   map[string]any `json:"vmTrace"`
	}
	f.h.Into(&res, "trace_call", map[string]any{"from": f.Alice.Addr, "to": f.Contract, "gas": "0x30000"},
		[]string{"trace", "stateDiff", "vmTrace"}, "latest")
	require.Len(t, res.Trace, 1)
	require.Equal(t, f.Contract, res.Trace[0].Action.To)
	require.NotEmpty(t, res.StateDiff)
	require.NotEmpty(t, res.VmTrace)

	err := f.h.Fails("trace_call", map[string]any{"to": f.Contract}, []string{"bogus"}, "latest")
	require.Equal(t, "unrecognized trace type: bogus", err.Message)

	// The trace types are checked before the block, the block before the call.
	err = f.h.Fails("trace_call", map[string]any{"gasPrice": "0x1", "maxFeePerGas": "0x1"}, []string{"trace"}, "0x99")
	require.Equal(t, "header not found", err.Message)
	err = f.h.Fails("trace_call", map[string]any{"gasPrice": "0x1", "maxFeePerGas": "0x1"}, []string{"trace"}, "latest")
	require.Equal(t, "both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified", err.Message)
	err = f.h.Fails("trace_call", map[string]any{"to": f.Contract}, []string{"trace"}, common.Hash{9})
	require.Equal(t, "header not found", err.Message)
}

func TestCall_StateOverride(t *testing.T) {
	f := newFixture(t)
	var res struct {
		Trace []actionTrace `json:"trace"`
	}
	f.h.Into(&res, "trace_call", map[string]any{"from": f.Alice.Addr, "to": f.Bob, "value": "0x7"}, []string{"trace"}, "latest",
		map[string]any{"stateOverrides": map[string]any{f.Alice.Addr.Hex(): map[string]any{"balance": "0x1"}}})
	// The sender cannot pay after the override: the call fails.
	require.Len(t, res.Trace, 1)
}

func TestCallMany(t *testing.T) {
	f := newFixture(t)
	var res []struct {
		Output hexutil.Bytes `json:"output"`
		Trace  []actionTrace `json:"trace"`
	}
	f.h.Into(&res, "trace_callMany", []any{
		[]any{map[string]any{"from": f.Alice.Addr, "to": f.Bob, "value": "0x1"}, []string{"trace"}},
		[]any{map[string]any{"from": f.Alice.Addr, "to": f.Contract}, []string{"trace"}},
	}, "latest")
	require.Len(t, res, 2)
	require.Len(t, res[0].Trace, 1)

	err := f.h.Fails("trace_callMany", []any{[]any{map[string]any{}, []string{"nope"}}}, "latest")
	require.Equal(t, "call 0: unrecognized trace type: nope", err.Message)
	err = f.h.Fails("trace_callMany", []any{[]any{map[string]any{}}}, "latest")
	require.Equal(t, dispatch.CodeInvalidParams, err.Code)
}

// spinner runs tests.SpinCode.
var spinner = common.HexToAddress("0x5917")

// newSizeHarness serves the trace methods with a response size limit, over a
// state that holds the spinner.
func newSizeHarness(t *testing.T, maxResponseSize int) *tests.Harness {
	t.Helper()
	fc, cs := tests.NewCarmenCore(t, tests.WithBlocks(1))
	cs.SetCode(spinner, tests.SpinCode())
	return newHarness(t, fc, func(c *dispatch.Config) { c.MaxResponseSize = maxResponseSize })
}

// spin is a call of the spinner with 100,000 gas: about 20,000 operations,
// whose vmTrace is about 2 MiB in the response.
var spin = map[string]any{"to": spinner, "gas": hexutil.Uint64(100_000)}

func TestCall_ResponseIsBoundedByMaxResponseSize(t *testing.T) {
	// Without a vmTrace the result is checked once encoded.
	err := newSizeHarness(t, 10).Fails("trace_call", spin, []string{"trace"}, "latest")
	require.Equal(t, ErrResponseTooLarge.Error(), err.Message)

	// A vmTrace stops growing at the limit, long before the execution ends.
	err = newSizeHarness(t, 1<<20).Fails("trace_call", spin, []string{"vmTrace"}, "latest")
	require.Equal(t, "vmTrace exceeds size limit", err.Message)

	res := newSizeHarness(t, 4<<20).Result("trace_call", spin, []string{"vmTrace"}, "latest")
	require.Less(t, len(res), 4<<20)
	require.Greater(t, len(res), 1<<20)
}

func TestCallMany_CallsShareMaxResponseSize(t *testing.T) {
	h := newSizeHarness(t, 4<<20)
	call := []any{spin, []string{"vmTrace"}}
	res := h.Result("trace_callMany", []any{call}, "latest")
	require.Greater(t, len(res), 1<<20)

	// The second call may only use what the first one left.
	err := h.Fails("trace_callMany", []any{call, call}, "latest")
	require.Equal(t, "call 1: vmTrace exceeds size limit", err.Message)

	// The results without a vmTrace count as well.
	err = newSizeHarness(t, 1000).Fails("trace_callMany", []any{
		[]any{spin, []string{"trace"}}, []any{spin, []string{"trace"}}, []any{spin, []string{"trace"}},
	}, "latest")
	require.Equal(t, ErrResponseTooLarge.Error(), err.Message)

	// Without a limit there is no shared budget; each vmTrace keeps its own cap.
	res = newSizeHarness(t, 0).Result("trace_callMany", []any{call, call}, "latest")
	require.Greater(t, len(res), 3<<20)
}

func TestFilter(t *testing.T) {
	f := newFixture(t)
	var traces []actionTrace
	f.h.Into(&traces, "trace_filter", map[string]any{"fromBlock": "0x2", "toBlock": "0x2"})
	require.Len(t, traces, 2)

	f.h.Into(&traces, "trace_filter", map[string]any{"fromBlock": "0x2", "toBlock": "0x2", "toAddress": []string{f.Bob.Hex()}})
	require.Len(t, traces, 1)
	require.Equal(t, f.Bob, traces[0].Action.To)

	// In series: after and count.
	f.h.Into(&traces, "trace_filter", map[string]any{"fromBlock": "0x1", "toBlock": "0x2", "after": 1, "count": 1})
	require.Len(t, traces, 1)
	require.Equal(t, f.Contract, traces[0].Action.To)

	// By hash and by tag.
	f.h.Into(&traces, "trace_filter", map[string]any{
		"fromBlock": map[string]any{"blockHash": f.Core.BlockAt(2).Hash}, "toBlock": "latest"})
	require.Len(t, traces, 2)
	err := f.h.Fails("trace_filter", map[string]any{"fromBlock": map[string]any{"blockHash": common.Hash{9}}})
	require.Equal(t, "block not found", err.Message)
}

func TestFilter_RangeLimit(t *testing.T) {
	f := newFixture(t)
	for range 5 {
		f.Core.AddBlock()
	}
	h := newHarness(t, f.Core, func(c *dispatch.Config) { c.MaxTraceFilterRange = 3 })
	// The limit is checked on the requested bounds, before any block is read.
	calls := f.Core.Calls("Block")
	err := h.Fails("trace_filter", map[string]any{"fromBlock": "0x1", "toBlock": "0x6"})
	require.Equal(t, "too wide blocks range, the limit is 3", err.Message)
	err = h.Fails("trace_filter", map[string]any{"fromBlock": "0x1", "toBlock": "0x6", "count": 5})
	require.Equal(t, "too wide blocks range, the limit is 3", err.Message)
	require.Equal(t, calls, f.Core.Calls("Block"))

	var traces []actionTrace
	h.Into(&traces, "trace_filter", map[string]any{"fromBlock": "0x1", "toBlock": "0x4"})
	require.Len(t, traces, 2)
}

func TestFilter_MissingBlock(t *testing.T) {
	f := newFixture(t)
	err := f.h.Fails("trace_filter", map[string]any{"fromBlock": "0x2", "toBlock": "0x9"})
	require.Contains(t, err.Message, "cannot get block from db")
}

func TestFilter_ShortReceiptListIsAnErrorNotAPanic(t *testing.T) {
	f := newFixture(t)
	f.Core.SetReceipts(f.Block, types.Receipts{}) // fewer receipts than transactions
	err := f.h.Fails("trace_block", "latest")
	require.Contains(t, err.Message, "no receipt found for transaction "+f.Transfer.Hash().String())
	require.Zero(t, f.h.Panics())

	// The transaction in the middle of a replay hits the untraced branch.
	f.Core.SetReceipts(f.Block, types.Receipts{{Status: 1}})
	err = f.h.Fails("trace_transaction", f.Store.Hash())
	require.Contains(t, err.Message, "no receipt found for transaction")
	require.Zero(t, f.h.Panics())
}

func TestParseTraceOptions(t *testing.T) {
	opts, err := parseTraceOptions([]string{"trace", "vmTrace"})
	require.NoError(t, err)
	require.Equal(t, TraceOptions{Trace: true, VmTrace: true}, opts)
	_, err = parseTraceOptions([]string{"x"})
	require.Error(t, err)
}

func TestCallRequest_Unmarshal(t *testing.T) {
	var r CallRequest
	require.NoError(t, json.Unmarshal([]byte(`[{"to":"0x0000000000000000000000000000000000000001"},["trace"]]`), &r))
	require.Equal(t, []string{"trace"}, r.TraceTypes)
	require.Error(t, json.Unmarshal([]byte(`[1]`), &r))
	require.Error(t, json.Unmarshal([]byte(`[{}, 5]`), &r))
}
