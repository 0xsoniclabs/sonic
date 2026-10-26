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

package trace_test

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/0xsoniclabs/sonic/rpc/trace"
)

func newTrace(t *testing.T, mutate ...tests.Mutate) (*tests.ReplayChain, *tests.Harness) {
	t.Helper()
	r := tests.NewReplayChain(t)
	return r, tests.NewHarness(t, r.Core, func(h *tests.Harness) { trace.Register(h.D, h.Env) }, mutate...)
}

type actionTrace struct {
	Action struct {
		From common.Address `json:"from"`
		To   common.Address `json:"to"`
	} `json:"action"`
	BlockHash           common.Hash `json:"blockHash"`
	TransactionHash     common.Hash `json:"transactionHash"`
	TransactionPosition uint64      `json:"transactionPosition"`
	Type                string      `json:"type"`
	Error               string      `json:"error"`
}

func TestTrace_BlockTransactionAndGet(t *testing.T) {
	r, h := newTrace(t)
	var traces []actionTrace
	h.MustCall(&traces, "trace_block", "0x1")
	require.Len(t, traces, 2)
	require.Equal(t, r.Transfer.Hash(), traces[0].TransactionHash)
	require.Equal(t, r.Call.Hash(), traces[1].TransactionHash)
	require.Equal(t, r.Writer, traces[1].Action.To)
	require.Equal(t, "call", traces[1].Type)

	h.MustCall(&traces, "trace_transaction", r.Call.Hash())
	require.Len(t, traces, 1)
	require.Equal(t, r.Block.Hash, traces[0].BlockHash)

	// The index is the address of a trace in the call tree: the call itself is
	// at [], its first inner call at [0].
	h.MustCall(&traces, "trace_get", r.Call.Hash(), []string{})
	require.Len(t, traces, 1)
	require.Equal(t, r.Call.Hash(), traces[0].TransactionHash)
	require.Equal(t, "null", string(h.MustCall(nil, "trace_get", r.Call.Hash(), []string{"0x0"})))
}

func TestTrace_TheWordingsOfTheReference(t *testing.T) {
	r, h := newTrace(t)
	require.Equal(t, "transaction 0x0100000000000000000000000000000000000000000000000000000000000000 not found",
		h.Fails("trace_transaction", common.Hash{1}).Message)
	require.Equal(t, "cannot trace block 9 got invalid block for tracing", h.Fails("trace_block", "0x9").Message)
	require.Equal(t, "cannot trace block 0 got genesis block is not traceable", h.Fails("trace_block", "0x0").Message)
	require.Equal(t, "unrecognized trace type: everything", h.Fails("trace_call", map[string]any{"to": r.Returner}, []string{"everything"}, "latest").Message)
}

func TestTrace_AnEmptyBlockHasItsEmptyTrace(t *testing.T) {
	r, h := newTrace(t)
	r.Core.AddAndPublish(nil)
	h.WaitHead(2)
	var traces []actionTrace
	h.MustCall(&traces, "trace_block", "0x2")
	require.Len(t, traces, 1)
	require.Equal(t, "empty", traces[0].Type)
	require.Equal(t, "Empty block", traces[0].Error)
}

func TestTrace_CallAndCallMany(t *testing.T) {
	r, h := newTrace(t)
	var res struct {
		Output    string            `json:"output"`
		Trace     []json.RawMessage `json:"trace"`
		StateDiff json.RawMessage   `json:"stateDiff"`
		VmTrace   json.RawMessage   `json:"vmTrace"`
	}
	call := map[string]any{"from": r.Wallet.Address, "to": r.Returner}
	h.MustCall(&res, "trace_call", call, []string{"trace", "stateDiff", "vmTrace"}, "latest")
	require.Equal(t, "0x000000000000000000000000000000000000000000000000000000000000002a", res.Output)
	require.Len(t, res.Trace, 1)
	require.NotEqual(t, "null", string(res.VmTrace))

	var many []json.RawMessage
	h.MustCall(&many, "trace_callMany", []any{
		[]any{call, []string{"trace"}},
		[]any{map[string]any{"from": r.Wallet.Address, "to": r.Writer}, []string{"stateDiff"}},
	}, "latest")
	require.Len(t, many, 2)

	e := h.Fails("trace_callMany", []any{[]any{call, []string{"trace"}}, []any{call}}, "latest")
	require.Equal(t, -32602, e.Code)
	require.Contains(t, e.Message, "parameter 0 (calls): ")
}

// TestTrace_CallsShareOneBudget is V12: two calls that fit alone fail
// together, and a vmTrace over the limit fails.
func TestTrace_CallsShareOneBudget(t *testing.T) {
	r, h := newTrace(t)
	call := map[string]any{"from": r.Wallet.Address, "to": r.Returner}
	raw := h.MustCall(nil, "trace_call", call, []string{"trace", "vmTrace"}, "latest")
	limit := len(raw) + len(raw)/2

	r2, h2 := newTrace(t, func(c *tests.Configs) { c.Exec.MaxResponseSize = limit })
	call2 := map[string]any{"from": r2.Wallet.Address, "to": r2.Returner}
	h2.MustCall(nil, "trace_call", call2, []string{"trace", "vmTrace"}, "latest")
	e := h2.Fails("trace_callMany", []any{
		[]any{call2, []string{"trace", "vmTrace"}},
		[]any{call2, []string{"trace", "vmTrace"}},
	}, "latest")
	require.Equal(t, "call 1: "+trace.ErrResponseTooLarge.Error(), e.Message)

	_, h3 := newTrace(t, func(c *tests.Configs) { c.Exec.MaxResponseSize = 64 })
	e = h3.Fails("trace_call", map[string]any{"to": r.Returner}, []string{"vmTrace"}, "latest")
	require.Equal(t, trace.ErrResponseTooLarge.Error(), e.Message)
}

func TestTrace_Filter(t *testing.T) {
	r, h := newTrace(t)
	var traces []actionTrace
	h.MustCall(&traces, "trace_filter", map[string]any{"fromBlock": "0x1", "toBlock": "latest"})
	require.Len(t, traces, 2)

	h.MustCall(&traces, "trace_filter", map[string]any{"fromBlock": "0x1", "toBlock": "0x1", "toAddress": []any{r.Writer}})
	require.Len(t, traces, 1)
	require.Equal(t, r.Call.Hash(), traces[0].TransactionHash)

	h.MustCall(&traces, "trace_filter", map[string]any{"fromBlock": "0x1", "toBlock": "0x1", "after": 1, "count": 1})
	require.Len(t, traces, 1)
	require.Equal(t, r.Call.Hash(), traces[0].TransactionHash)

	h.MustCall(&traces, "trace_filter", map[string]any{"fromBlock": "0x1", "toBlock": "0x1", "fromAddress": []any{common.Address{1}}})
	require.Empty(t, traces)
}

// TestTrace_FilterChecksItsRangeFirst is V2.
func TestTrace_FilterChecksItsRangeFirst(t *testing.T) {
	r, h := newTrace(t, func(c *tests.Configs) { c.Exec.MaxTraceFilterRange = 10 })
	for range 20 {
		r.Core.AddBlock(nil)
	}
	r.Core.Publish(21)
	h.WaitHead(21)
	blocks := r.Core.Calls("Block")
	require.Equal(t, "too wide blocks range, the limit is 10", h.Fails("trace_filter", map[string]any{"fromBlock": "0x0", "toBlock": "0x14"}).Message)
	require.Equal(t, blocks, r.Core.Calls("Block"))
}

func TestTrace_AReplayThatDisagreesWithItsReceiptFails(t *testing.T) {
	r := tests.NewReplayChain(t)
	failing := r.Wallet.Transfer(t, 2, r.Recipient, new(big.Int).Lsh(big.NewInt(1), 200))
	r.Core.AddAndPublish(types.Transactions{failing})
	h := tests.NewHarness(t, r.Core, func(h *tests.Harness) { trace.Register(h.D, h.Env) })
	h.WaitHead(2)
	e := h.Fails("trace_block", "0x2")
	require.Contains(t, e.Message, "cannot trace block 2 got")
}
