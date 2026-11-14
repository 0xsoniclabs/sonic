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

package debug_test

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/rpc/debug"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
	"github.com/stretchr/testify/require"
)

var contract = common.Address{0xc0}

// traceChain is a Carmen core whose block 1, the head, holds a call of a
// contract that writes storage, signed by a funded account.
type traceChain struct {
	harness *tests.Harness
	account *tests.Account
	call    *types.Transaction
}

func newTraceChain(t *testing.T, mutate ...func(*tests.Settings)) *traceChain {
	t.Helper()
	account := tests.NewAccount()
	fake := tests.NewCarmenCore(t, map[common.Address]tests.AccountState{
		account.Address(): {Balance: big.NewInt(1e18)},
		contract:          {Code: tests.WritesStorage},
	})
	call := account.Sign(&types.DynamicFeeTx{ChainID: big.NewInt(tests.ChainID), To: &contract, Gas: 100000,
		GasFeeCap: big.NewInt(2e9), GasTipCap: big.NewInt(1)})
	fake.AddBlock(call)
	fake.SetHead(1)
	h := tests.NewHarness(t, fake, func(h *tests.Harness) { debug.Register(h.Dispatcher, h.Environment) }, mutate...)
	return &traceChain{harness: h, account: account, call: call}
}

type structLogs struct {
	Gas        uint64           `json:"gas"`
	Failed     bool             `json:"failed"`
	StructLogs []map[string]any `json:"structLogs"`
}

func TestTraceTransaction_ReplaysTheBlockUpToTheTransaction(t *testing.T) {
	chain := newTraceChain(t)
	h := chain.harness
	var logged structLogs
	h.Call(t, &logged, "debug_traceTransaction", chain.call.Hash())
	require.False(t, logged.Failed)
	require.Len(t, logged.StructLogs, 4, "PUSH1, PUSH1, SSTORE, STOP")
	require.Equal(t, "SSTORE", logged.StructLogs[2]["op"])

	var called struct {
		From common.Address `json:"from"`
		To   common.Address `json:"to"`
		Type string         `json:"type"`
	}
	h.Call(t, &called, "debug_traceTransaction", chain.call.Hash(), map[string]any{"tracer": "callTracer"})
	require.Equal(t, chain.account.Address(), called.From)
	require.Equal(t, contract, called.To)
	require.Equal(t, "CALL", called.Type)
	require.Zero(t, h.Core.OpenStates())
}

func TestTraceTransaction_RefusesWhatItCannotTrace(t *testing.T) {
	chain := newTraceChain(t)
	h := chain.harness
	cases := map[string]struct {
		params []any
		err    *dispatch.Error
	}{
		"unknown transaction": {[]any{common.Hash{0x01}}, &dispatch.Error{Code: dispatch.CodeDefault, Message: "transaction not found"}},
		"custom tracer": {[]any{chain.call.Hash(), map[string]any{"tracer": "{result: function() {}}"}},
			&dispatch.Error{Code: dispatch.CodeDefault, Message: "custom tracer is not permitted"}},
		"malformed timeout": {[]any{chain.call.Hash(), map[string]any{"timeout": "x"}},
			&dispatch.Error{Code: dispatch.CodeInvalidParams, Message: `time: invalid duration "x"`}},
		"timeout above the limit": {[]any{chain.call.Hash(), map[string]any{"timeout": "30s"}},
			&dispatch.Error{Code: dispatch.CodeInvalidParams, Message: "tracer timeout 30s is above the limit of 5s"}},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.err, h.Fail(t, "debug_traceTransaction", test.params...))
			require.Zero(t, h.Core.OpenStates())
		})
	}
	var logged structLogs
	h.Call(t, &logged, "debug_traceTransaction", chain.call.Hash(), map[string]any{"timeout": "5s"})
}

func TestTraceBlock_TracesEveryTransaction(t *testing.T) {
	chain := newTraceChain(t)
	h := chain.harness
	block, err := h.Core.Block(1)
	require.NoError(t, err)
	for _, call := range [][]any{{"debug_traceBlockByNumber", "0x1"}, {"debug_traceBlockByNumber", "latest"}, {"debug_traceBlockByHash", block.Hash}} {
		var traces []struct {
			TxHash common.Hash     `json:"txHash"`
			Result json.RawMessage `json:"result"`
		}
		h.Call(t, &traces, call[0].(string), call[1:]...)
		require.Len(t, traces, 1)
		require.Equal(t, chain.call.Hash(), traces[0].TxHash)
	}
	unknown := common.Hash{0x01}
	require.Equal(t, "genesis is not traceable", h.Fail(t, "debug_traceBlockByNumber", "0x0").Message)
	require.Equal(t, "block #5 not found", h.Fail(t, "debug_traceBlockByNumber", "0x5").Message)
	require.Equal(t, "block "+unknown.Hex()+" not found", h.Fail(t, "debug_traceBlockByHash", unknown).Message)
	require.Zero(t, h.Core.OpenStates())
}

func TestTraceCall_TracesACallWithOverrides(t *testing.T) {
	chain := newTraceChain(t)
	h := chain.harness
	call := map[string]any{"from": chain.account.Address(), "to": contract}
	var logged structLogs
	h.Call(t, &logged, "debug_traceCall", call, "latest")
	require.Len(t, logged.StructLogs, 4)
	h.Call(t, &logged, "debug_traceCall", call, "latest", map[string]any{
		"stateOverrides": map[common.Address]any{contract: map[string]any{"code": hexutil.Encode(tests.ReturnsFortyTwo)}},
		"txIndex":        "0x0",
	})
	require.Len(t, logged.StructLogs, 6, "the overridden code")
	require.Contains(t, h.Fail(t, "debug_traceCall", call, "latest", map[string]any{"txIndex": "0x1"}).Message,
		"transaction index 1 out of range for block", "as the legacy replay, the end of a block with transactions")

	cases := map[string]struct {
		block any
		err   string
	}{
		"pending":     {"pending", "tracing on top of pending is not supported"},
		"no selector": {map[string]any{}, "invalid arguments; neither block number nor hash specified"},
		"missing":     {"0x5", "header not found"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.err, h.Fail(t, "debug_traceCall", call, test.block).Message)
		})
	}
	require.Zero(t, h.Core.OpenStates())
}

func TestTrace_BoundsTheResponseAndTheStructLogs(t *testing.T) {
	limited := newTraceChain(t, func(settings *tests.Settings) { settings.Exec.StructLogLimit = 2 })
	var logged structLogs
	limited.harness.Call(t, &logged, "debug_traceTransaction", limited.call.Hash())
	require.Len(t, logged.StructLogs, 2)

	small := newTraceChain(t, func(settings *tests.Settings) { settings.Exec.MaxResponseSize = 10 })
	for _, call := range [][]any{{"debug_traceTransaction", small.call.Hash()}, {"debug_traceBlockByNumber", "0x1"}} {
		require.Equal(t, "response size exceeded", small.harness.Fail(t, call[0].(string), call[1:]...).Message)
	}
}
