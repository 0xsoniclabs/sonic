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
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/0xsoniclabs/sonic/rpc/trace"
)

var (
	counter = common.HexToAddress("0xc1")
	// counterCode increments slot 0 and returns its new value.
	counterCode = []byte{0x60, 0x00, 0x54, 0x60, 0x01, 0x01, 0x80, 0x60, 0x00, 0x55, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3}
)

// traceChain is a chain whose first block holds two increments of the
// counter by one sender, and whose second block is empty; every block's
// state is that of genesis.
type traceChain struct {
	core    *tests.CarmenCore
	harness *tests.Harness
	sender  common.Address
	sent    []*types.Transaction
	hashes  []common.Hash
}

func newTraceChain(t *testing.T, receipts types.Receipts, mutations ...tests.Mutate) traceChain {
	t.Helper()
	signer := tests.NewSigner(t)
	carmenCore := tests.NewCarmenCore(t, map[common.Address]tests.Account{
		signer.Address: {Balance: new(big.Int).Lsh(big.NewInt(1), 100)},
		counter:        {Code: counterCode},
	})
	sent := []*types.Transaction{
		signer.Sign(t, &types.LegacyTx{Nonce: 0, To: &counter, Gas: 100_000, GasPrice: big.NewInt(2_000_000_000)}),
		signer.Sign(t, &types.LegacyTx{Nonce: 1, To: &counter, Gas: 100_000, GasPrice: big.NewInt(2_000_000_000)}),
	}
	first := carmenCore.AppendBlock(tests.BlockSpec{Transactions: types.Transactions(sent), Receipts: receipts})
	second := carmenCore.AppendBlock(tests.BlockSpec{})
	carmenCore.PublishHead(2)
	harness := tests.NewHarness(t, carmenCore.FakeCore, func(dispatcher *dispatch.Dispatcher, environment *exec.Environment) {
		trace.Register(dispatcher, environment)
	}, mutations...)
	harness.WaitForHead(2)
	return traceChain{core: carmenCore, harness: harness, sender: signer.Address, sent: sent, hashes: []common.Hash{first.Hash, second.Hash}}
}

func output(number int64) string {
	return "0x" + common.BigToHash(big.NewInt(number)).Hex()[2:]
}

func traces(t *testing.T, chain traceChain, method string, parameters ...any) []map[string]any {
	t.Helper()
	return tests.Decode[[]map[string]any](t, chain.harness.Result(method, parameters...))
}

// TestTransaction_ReplaysTheBlockUpToTheTransaction holds I8.
func TestTransaction_ReplaysTheBlockUpToTheTransaction(t *testing.T) {
	chain := newTraceChain(t, nil)
	found := traces(t, chain, "trace_transaction", chain.sent[1].Hash())
	require.Len(t, found, 1)
	require.Equal(t, output(2), found[0]["result"].(map[string]any)["output"], "the first increment ran untraced before")
	require.Equal(t, float64(1), found[0]["transactionPosition"])
	require.Equal(t, strings.ToLower(counter.Hex()), found[0]["action"].(map[string]any)["to"])

	require.Equal(t, found, traces(t, chain, "trace_get", chain.sent[1].Hash(), []any{}))
	require.JSONEq(t, `null`, string(chain.harness.Result("trace_get", chain.sent[1].Hash(), []any{"0x5"})), "no trace at the position")
	require.Equal(t, "transaction "+common.Hash{1}.String()+" not found", chain.harness.Failure("trace_transaction", common.Hash{1}).Message)
}

func TestBlock_TracesEveryTransaction(t *testing.T) {
	chain := newTraceChain(t, nil)
	found := traces(t, chain, "trace_block", "0x1")
	require.Len(t, found, 2)
	require.Equal(t, output(1), found[0]["result"].(map[string]any)["output"])
	require.Equal(t, found, traces(t, chain, "trace_block", map[string]any{"blockHash": chain.hashes[0]}))

	empty := traces(t, chain, "trace_block", "latest")
	require.Len(t, empty, 1)
	require.Equal(t, "Empty block", empty[0]["error"])

	cases := map[string]struct {
		block   any
		message string
	}{
		"a number above the head": {"0x9", "cannot trace block 9 got invalid block for tracing"},
		"genesis":                 {"0x0", "cannot trace block 0 got genesis block is not traceable"},
		"an unknown hash":         {map[string]any{"blockHash": common.Hash{1}}, "block not found"},
		"no selector":             {map[string]any{}, "unknown header selector"},
	}
	for name, test := range cases {
		require.Equal(t, test.message, chain.harness.Failure("trace_block", test.block).Message, name)
	}
}

// TestReplay_ChecksEveryTransactionAgainstItsReceipt holds F8 for the
// untraced branch, which indexed the receipts unguarded and panicked on a
// list too short.
func TestReplay_ChecksEveryTransactionAgainstItsReceipt(t *testing.T) {
	failed := newTraceChain(t, types.Receipts{{Status: types.ReceiptStatusFailed}})
	require.Equal(t, "invalid transaction replay state at "+failed.sent[0].Hash().String(), failed.harness.Failure("trace_transaction", failed.sent[1].Hash()).Message)
	require.Contains(t, failed.harness.Failure("trace_transaction", failed.sent[0].Hash()).Message, "invalid transaction replay state at "+failed.sent[0].Hash().String())

	missing := newTraceChain(t, nil)
	missing.core.SetMode("Receipts", tests.NotFound)
	require.Equal(t, "no receipt found for transaction "+missing.sent[0].Hash().String(), missing.harness.Failure("trace_transaction", missing.sent[1].Hash()).Message)
	require.Zero(t, missing.harness.Panics())
}

// TestFilter_ReplaysTheRangeWithinItsLimits holds V2, O1 and P2.
func TestFilter_ReplaysTheRangeWithinItsLimits(t *testing.T) {
	chain := newTraceChain(t, nil)
	all := traces(t, chain, "trace_filter", map[string]any{"fromBlock": "0x1", "toBlock": "latest"})
	require.Len(t, all, 2)
	require.Empty(t, traces(t, chain, "trace_filter", map[string]any{"fromBlock": "0x1", "toAddress": []any{common.HexToAddress("0x99").Hex()}}))
	require.Len(t, traces(t, chain, "trace_filter", map[string]any{"fromBlock": "0x1", "fromAddress": []any{chain.sender.Hex()}}), 2)
	window := traces(t, chain, "trace_filter", map[string]any{"fromBlock": "0x1", "after": 1, "count": 1})
	require.Len(t, window, 1)
	require.Equal(t, output(2), window[0]["result"].(map[string]any)["output"], "the window keeps the order of the blocks")
	require.Equal(t, "cannot get block from db 9", chain.harness.Failure("trace_filter", map[string]any{"fromBlock": "0x9", "toBlock": "0x9"}).Message)

	narrow := newTraceChain(t, nil, func(configs *tests.Configs) {
		configs.Exec.MaxTraceFilterRange = 1
		configs.Exec.MaxResponseSize = 300
	})
	require.Equal(t, "too wide blocks range, the limit is 1", narrow.harness.Failure("trace_filter", map[string]any{"fromBlock": "0x0", "toBlock": "0x2"}).Message)
	require.Equal(t, "response too large", narrow.harness.Failure("trace_filter", map[string]any{"fromBlock": "0x1", "toBlock": "0x2"}).Message)
}
