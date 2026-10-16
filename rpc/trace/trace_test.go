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
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

var (
	bob      = common.HexToAddress("0xb0b")
	contract = common.HexToAddress("0xc0de")
	// PUSH1 1 PUSH1 0 SSTORE STOP
	stores = []byte{0x60, 0x01, 0x60, 0x00, 0x55, 0x00}
)

type fixture struct {
	h     *tests.Harness
	fc    *tests.FakeCore
	alice tests.Account
	tx0   *types.Transaction // a transfer to bob
	tx1   *types.Transaction // a call that writes storage
	block uint64
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	fc, cs := tests.NewCarmenCore(t, tests.WithBlocks(1))
	alice := tests.NewAccount(t, cs, big.NewInt(1e18))
	cs.SetCode(contract, stores)
	tx0 := alice.Transfer(t, 0, bob, 5)
	tx1 := alice.Call(t, 1, contract, 0, nil, 100_000)
	block := fc.AddBlock(tx0, tx1)
	cfg := tests.Default()
	h := tests.NewHarness(t, fc, cfg, func(d *dispatch.Dispatcher, chain *dispatch.Chain) {
		Register(d, exec.NewEnv(chain, oracle.New(oracle.Config{}, fc)))
	})
	return &fixture{h: h, fc: fc, alice: alice, tx0: tx0, tx1: tx1, block: block.NumberU64()}
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
	f.h.Into(&traces, "trace_transaction", f.tx0.Hash())
	require.Len(t, traces, 1)
	require.Equal(t, "call", traces[0].Type)
	require.Equal(t, f.alice.Addr, traces[0].Action.From)
	require.Equal(t, bob, traces[0].Action.To)
	require.Equal(t, int64(5), traces[0].Action.Value.ToInt().Int64())
	require.Equal(t, f.tx0.Hash().Hex(), traces[0].TransactionHash)

	// The second transaction needs the first replayed: the state it runs on is
	// the state after.
	f.h.Into(&traces, "trace_transaction", f.tx1.Hash())
	require.Len(t, traces, 1)
	require.Equal(t, contract, traces[0].Action.To)
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
	f.h.Into(&traces, "trace_block", f.fc.BlockAt(f.block).Hash)
	require.Len(t, traces, 2)

	// An empty block has one pseudo trace; the genesis block cannot be traced.
	f.fc.AddBlock()
	require.Eventually(t, func() bool { return f.h.Chain.Latest() == f.block+1 }, time.Second, time.Millisecond)
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
	f.h.Into(&traces, "trace_get", f.tx0.Hash(), []string{})
	require.Len(t, traces, 1)
	require.Equal(t, f.tx0.Hash().Hex(), traces[0].TransactionHash)
	// A trace index that is not there answers null.
	require.JSONEq(t, `null`, string(f.h.Result("trace_get", f.tx0.Hash(), []string{"0x7"})))
}

func TestCall(t *testing.T) {
	f := newFixture(t)
	var res struct {
		Output    hexutil.Bytes  `json:"output"`
		Trace     []actionTrace  `json:"trace"`
		StateDiff map[string]any `json:"stateDiff"`
		VmTrace   map[string]any `json:"vmTrace"`
	}
	f.h.Into(&res, "trace_call", map[string]any{"from": f.alice.Addr, "to": contract, "gas": "0x30000"},
		[]string{"trace", "stateDiff", "vmTrace"}, "latest")
	require.Len(t, res.Trace, 1)
	require.Equal(t, contract, res.Trace[0].Action.To)
	require.NotEmpty(t, res.StateDiff)
	require.NotEmpty(t, res.VmTrace)

	err := f.h.Fails("trace_call", map[string]any{"to": contract}, []string{"bogus"}, "latest")
	require.Equal(t, "unrecognized trace type: bogus", err.Message)

	// The trace types are checked before the block, the block before the call.
	err = f.h.Fails("trace_call", map[string]any{"gasPrice": "0x1", "maxFeePerGas": "0x1"}, []string{"trace"}, "0x99")
	require.Equal(t, "header not found", err.Message)
	err = f.h.Fails("trace_call", map[string]any{"gasPrice": "0x1", "maxFeePerGas": "0x1"}, []string{"trace"}, "latest")
	require.Equal(t, "both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified", err.Message)
	err = f.h.Fails("trace_call", map[string]any{"to": contract}, []string{"trace"}, common.Hash{9})
	require.Equal(t, "header not found", err.Message)
}

func TestCall_StateOverride(t *testing.T) {
	f := newFixture(t)
	var res struct {
		Trace []actionTrace `json:"trace"`
	}
	f.h.Into(&res, "trace_call", map[string]any{"from": f.alice.Addr, "to": bob, "value": "0x7"}, []string{"trace"}, "latest",
		map[string]any{"stateOverrides": map[string]any{f.alice.Addr.Hex(): map[string]any{"balance": "0x1"}}})
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
		[]any{map[string]any{"from": f.alice.Addr, "to": bob, "value": "0x1"}, []string{"trace"}},
		[]any{map[string]any{"from": f.alice.Addr, "to": contract}, []string{"trace"}},
	}, "latest")
	require.Len(t, res, 2)
	require.Len(t, res[0].Trace, 1)

	err := f.h.Fails("trace_callMany", []any{[]any{map[string]any{}, []string{"nope"}}}, "latest")
	require.Equal(t, "call 0: unrecognized trace type: nope", err.Message)
	err = f.h.Fails("trace_callMany", []any{[]any{map[string]any{}}}, "latest")
	require.Equal(t, dispatch.CodeInvalidParams, err.Code)
}

func TestFilter(t *testing.T) {
	f := newFixture(t)
	var traces []actionTrace
	f.h.Into(&traces, "trace_filter", map[string]any{"fromBlock": "0x2", "toBlock": "0x2"})
	require.Len(t, traces, 2)

	f.h.Into(&traces, "trace_filter", map[string]any{"fromBlock": "0x2", "toBlock": "0x2", "toAddress": []string{bob.Hex()}})
	require.Len(t, traces, 1)
	require.Equal(t, bob, traces[0].Action.To)

	// In series: after and count.
	f.h.Into(&traces, "trace_filter", map[string]any{"fromBlock": "0x1", "toBlock": "0x2", "after": 1, "count": 1})
	require.Len(t, traces, 1)
	require.Equal(t, contract, traces[0].Action.To)

	// By hash and by tag.
	f.h.Into(&traces, "trace_filter", map[string]any{
		"fromBlock": map[string]any{"blockHash": f.fc.BlockAt(2).Hash}, "toBlock": "latest"})
	require.Len(t, traces, 2)
	err := f.h.Fails("trace_filter", map[string]any{"fromBlock": map[string]any{"blockHash": common.Hash{9}}})
	require.Equal(t, "block not found", err.Message)
}

func TestFilter_RangeLimit(t *testing.T) {
	f := newFixture(t)
	for range 5 {
		f.fc.AddBlock()
	}
	h := tests.NewHarness(t, f.fc, func() dispatch.Config { c := tests.Default(); c.MaxTraceFilterRange = 3; return c }(),
		func(d *dispatch.Dispatcher, chain *dispatch.Chain) {
			Register(d, exec.NewEnv(chain, oracle.New(oracle.Config{}, f.fc)))
		})
	// The limit is checked on the requested bounds, before any block is read.
	calls := f.fc.Calls("Block")
	err := h.Fails("trace_filter", map[string]any{"fromBlock": "0x1", "toBlock": "0x6"})
	require.Equal(t, "too wide blocks range, the limit is 3", err.Message)
	err = h.Fails("trace_filter", map[string]any{"fromBlock": "0x1", "toBlock": "0x6", "count": 5})
	require.Equal(t, "too wide blocks range, the limit is 3", err.Message)
	require.Equal(t, calls, f.fc.Calls("Block"))

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
	f.fc.SetReceipts(f.block, types.Receipts{}) // fewer receipts than transactions
	err := f.h.Fails("trace_block", "latest")
	require.Contains(t, err.Message, "no receipt found for transaction "+f.tx0.Hash().String())
	require.Zero(t, f.h.Panics())

	// The transaction in the middle of a replay hits the untraced branch.
	f.fc.SetReceipts(f.block, types.Receipts{{Status: 1}})
	err = f.h.Fails("trace_transaction", f.tx1.Hash())
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
