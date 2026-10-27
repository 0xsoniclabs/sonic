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
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/debug"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

var (
	counter = common.HexToAddress("0xc1")
	// counterCode increments slot 0 and returns its new value.
	counterCode = []byte{0x60, 0x00, 0x54, 0x60, 0x01, 0x01, 0x80, 0x60, 0x00, 0x55, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3}
)

// traceChain is a chain whose first block holds two increments of the
// counter by one sender; every block's state is that of genesis, so the
// replay of the first increment before the second shows in the result.
type traceChain struct {
	core    *tests.CarmenCore
	harness *tests.Harness
	sent    []*types.Transaction
	hashes  []common.Hash
}

func newTraceChain(t *testing.T, mutations ...tests.Mutate) traceChain {
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
	first := carmenCore.AppendBlock(tests.BlockSpec{Transactions: types.Transactions(sent)})
	second := carmenCore.AppendBlock(tests.BlockSpec{})
	carmenCore.PublishHead(2)
	harness := tests.NewHarness(t, carmenCore.FakeCore, func(dispatcher *dispatch.Dispatcher, environment *exec.Environment) {
		debug.Register(dispatcher, environment)
	}, mutations...)
	harness.WaitForHead(2)
	return traceChain{core: carmenCore, harness: harness, sent: sent, hashes: []common.Hash{first.Hash, second.Hash}}
}

func value(number int64) string {
	return common.BigToHash(big.NewInt(number)).Hex()[2:]
}

// TestTraceTransaction_ReplaysTheBlockUpToTheTransaction holds I8 and I7.
func TestTraceTransaction_ReplaysTheBlockUpToTheTransaction(t *testing.T) {
	chain := newTraceChain(t)
	before := chain.core.States()
	trace := tests.Decode[map[string]any](t, chain.harness.Result("debug_traceTransaction", chain.sent[1].Hash()))
	require.Equal(t, "0x"+value(2), trace["returnValue"], "the first increment ran before")
	require.Equal(t, false, trace["failed"])
	require.NotEmpty(t, trace["structLogs"])
	after := chain.core.States()
	require.Equal(t, before.Opened+1, after.Opened)
	require.Equal(t, after.Opened, after.Released)

	calls := tests.Decode[map[string]any](t, chain.harness.Result("debug_traceTransaction", chain.sent[0].Hash(), map[string]any{"tracer": "callTracer"}))
	require.Equal(t, "CALL", calls["type"])
	require.Equal(t, strings.ToLower(counter.Hex()), calls["to"])
	require.Equal(t, "0x"+value(1), calls["output"])

	require.Equal(t, "transaction not found", chain.harness.Failure("debug_traceTransaction", common.Hash{1}).Message)
}

// TestTraceTransaction_ChecksTheTracerAsItIsBuilt holds V6 and F2.
func TestTraceTransaction_ChecksTheTracerAsItIsBuilt(t *testing.T) {
	chain := newTraceChain(t)
	hash := chain.sent[0].Hash()
	require.Equal(t, "custom tracer is not permitted", chain.harness.Failure("debug_traceTransaction", hash, map[string]any{"tracer": "{result: function() {}}"}).Message)
	failure := chain.harness.Failure("debug_traceTransaction", hash, map[string]any{"timeout": "1h"})
	require.Equal(t, dispatch.CodeInvalidParameters, failure.Code)
	require.Equal(t, "tracer timeout 1h0m0s above the limit of 5s", failure.Message)
	failure = chain.harness.Failure("debug_traceTransaction", hash, map[string]any{"timeout": "soon"})
	require.Equal(t, dispatch.CodeInvalidParameters, failure.Code)
	require.Contains(t, failure.Message, `invalid tracer timeout "soon"`)
	require.Equal(t, "transaction not found", chain.harness.Failure("debug_traceTransaction", common.Hash{1}, map[string]any{"timeout": "soon"}).Message, "the subject is found first")
	chain.harness.Result("debug_traceTransaction", hash, map[string]any{"timeout": "2s"})
}

// TestTraceTransaction_KeepsTheLimitsOfTheReference holds O1.
func TestTraceTransaction_KeepsTheLimitsOfTheReference(t *testing.T) {
	chain := newTraceChain(t, func(configs *tests.Configs) { configs.Exec.StructLogLimit = 3 })
	trace := tests.Decode[map[string]any](t, chain.harness.Result("debug_traceTransaction", chain.sent[0].Hash()))
	require.Len(t, trace["structLogs"], 3, "StructLogLimit counts entries")

	small := newTraceChain(t, func(configs *tests.Configs) { configs.Exec.MaxResponseSize = 100 })
	require.Equal(t, "response size exceeded", small.harness.Failure("debug_traceTransaction", small.sent[0].Hash()).Message)
	require.Equal(t, "response size exceeded", small.harness.Failure("debug_traceBlockByNumber", "0x1", map[string]any{"tracer": "callTracer"}).Message)
}

func TestTraceBlock_TracesEveryTransaction(t *testing.T) {
	chain := newTraceChain(t)
	results := tests.Decode[[]map[string]any](t, chain.harness.Result("debug_traceBlockByNumber", "0x1", map[string]any{"tracer": "callTracer"}))
	require.Len(t, results, 2)
	require.Equal(t, chain.sent[1].Hash().Hex(), results[1]["txHash"])
	require.Equal(t, "0x"+value(2), results[1]["result"].(map[string]any)["output"])
	byHash := tests.Decode[[]map[string]any](t, chain.harness.Result("debug_traceBlockByHash", chain.hashes[0], map[string]any{"tracer": "callTracer"}))
	require.Equal(t, results, byHash)
	require.Equal(t, []map[string]any{}, tests.Decode[[]map[string]any](t, chain.harness.Result("debug_traceBlockByNumber", "latest")))

	require.Equal(t, "block #9 not found", chain.harness.Failure("debug_traceBlockByNumber", "0x9").Message)
	require.Equal(t, "block "+common.Hash{1}.Hex()+" not found", chain.harness.Failure("debug_traceBlockByHash", common.Hash{1}).Message)
	require.Equal(t, "genesis is not traceable", chain.harness.Failure("debug_traceBlockByNumber", "0x0").Message)
}

func TestTraceCall_RunsAtAPositionWithTheOverrides(t *testing.T) {
	chain := newTraceChain(t)
	call := map[string]any{"to": counter}
	result := tests.Decode[map[string]any](t, chain.harness.Result("debug_traceCall", call, "0x1", map[string]any{"txIndex": "0x1", "tracer": "callTracer"}))
	require.Equal(t, "0x"+value(2), result["output"], "after the first transaction of the block")
	result = tests.Decode[map[string]any](t, chain.harness.Result("debug_traceCall", call, "0x1", map[string]any{
		"tracer":         "callTracer",
		"stateOverrides": map[string]any{counter.Hex(): map[string]any{"stateDiff": map[string]any{common.Hash{}.Hex(): common.BigToHash(big.NewInt(10)).Hex()}}},
		"blockOverrides": map[string]any{"number": "0x99"},
	}))
	require.Equal(t, "0x"+value(11), result["output"])

	require.Equal(t, "tracing on top of pending is not supported", chain.harness.Failure("debug_traceCall", call, "pending").Message)
	require.Equal(t, "header not found", chain.harness.Failure("debug_traceCall", call, common.Hash{1}).Message)
	require.Equal(t, "invalid arguments; neither block number nor hash specified", chain.harness.Failure("debug_traceCall", call, map[string]any{}).Message)
	require.Equal(t, "no transaction in genesis", chain.harness.Failure("debug_traceCall", call, "0x0").Message)
}

// TestBlocks_AreAnsweredAsStored: debug_getBlockRlp encodes the block as the
// reference did, whose block has a field RLP cannot encode, so it answers
// that error, as the reference does.
func TestBlocks_AreAnsweredAsStored(t *testing.T) {
	chain := newTraceChain(t)
	require.Contains(t, chain.harness.Failure("debug_getBlockRlp", 1).Message, "is not RLP-serializable")
	require.Equal(t, "block #9 not found", chain.harness.Failure("debug_getBlockRlp", 9).Message)
	printed := tests.Decode[string](t, chain.harness.Result("debug_printBlock", 1))
	require.Contains(t, printed, "EvmHeader")
	require.Equal(t, "block #9 not found", chain.harness.Failure("debug_printBlock", 9).Message)
	latest := tests.Decode[string](t, chain.harness.Result("debug_printBlock", uint64(1<<64-2)))
	require.Contains(t, latest, "Number: (*big.Int)", "a number cast to a tag names the head, as in the reference")
}
