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

package eth_test

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/eth"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

// simulatedBlock is the part of a simulated block the tests read.
type simulatedBlock struct {
	Number       hexutil.Uint64    `json:"number"`
	Hash         common.Hash       `json:"hash"`
	ParentHash   common.Hash       `json:"parentHash"`
	Timestamp    hexutil.Uint64    `json:"timestamp"`
	GasUsed      hexutil.Uint64    `json:"gasUsed"`
	Transactions []json.RawMessage `json:"transactions"`
	Calls        []struct {
		ReturnData hexutil.Bytes  `json:"returnData"`
		GasUsed    hexutil.Uint64 `json:"gasUsed"`
		Status     hexutil.Uint64 `json:"status"`
		Logs       []struct {
			Address     common.Address `json:"address"`
			Topics      []common.Hash  `json:"topics"`
			BlockHash   common.Hash    `json:"blockHash"`
			BlockNumber hexutil.Uint64 `json:"blockNumber"`
			LogIndex    hexutil.Uint64 `json:"logIndex"`
		} `json:"logs"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    string `json:"data"`
		} `json:"error"`
	} `json:"calls"`
}

func simulate(t *testing.T, h *tests.Harness, options map[string]any) []simulatedBlock {
	t.Helper()
	var blocks []simulatedBlock
	h.Call(t, &blocks, "eth_simulateV1", options)
	require.Zero(t, h.Core.OpenStates())
	return blocks
}

func blocksOfCalls(calls ...[]any) []any {
	blocks := make([]any, len(calls))
	for i, block := range calls {
		blocks[i] = map[string]any{"calls": block}
	}
	return blocks
}

func TestSimulateV1_AnswersTheOutcomeOfEachCall(t *testing.T) {
	h := newStateHarness(t)
	emitter := common.Address{0xc4}
	blocks := simulate(t, h, map[string]any{"blockStateCalls": []any{map[string]any{
		"stateOverrides": map[common.Address]any{emitter: map[string]any{"code": hexutil.Encode(tests.EmitsLog)}},
		"calls": []any{
			map[string]any{"to": contract},
			map[string]any{"to": reverter},
			map[string]any{"to": emitter},
			map[string]any{"to": emitter},
		},
	}}})
	require.Len(t, blocks, 1)
	block := blocks[0]
	require.Equal(t, hexutil.Uint64(2), block.Number)
	base, err := h.Core.Header(1)
	require.NoError(t, err)
	require.Equal(t, base.Hash, block.ParentHash)
	require.Equal(t, hexutil.Uint64(2+12), block.Timestamp)
	require.Len(t, block.Transactions, 4)
	calls := block.Calls
	require.Equal(t, hexutil.Uint64(1), calls[0].Status)
	require.Equal(t, common.BigToHash(big.NewInt(42)).Bytes(), []byte(calls[0].ReturnData))
	require.Equal(t, hexutil.Uint64(0), calls[1].Status)
	require.Equal(t, -32000, calls[1].Error.Code)
	require.Equal(t, "execution reverted", calls[1].Error.Message)
	require.Equal(t, "0xdeadbeef", calls[1].Error.Data)
	require.Equal(t, []byte{0xde, 0xad, 0xbe, 0xef}, []byte(calls[1].ReturnData))
	for i, call := range calls[2:] {
		require.Len(t, call.Logs, 1)
		log := call.Logs[0]
		require.Equal(t, emitter, log.Address)
		require.Equal(t, []common.Hash{{31: 0x01}}, log.Topics)
		require.Equal(t, block.Hash, log.BlockHash)
		require.Equal(t, block.Number, log.BlockNumber)
		require.Equal(t, hexutil.Uint64(i), log.LogIndex)
	}
	require.Equal(t, calls[0].GasUsed+calls[1].GasUsed+calls[2].GasUsed+calls[3].GasUsed, block.GasUsed)
}

// The blocks of a simulation build on each other: one sees the state the
// one before left and the hash of every block before it.
func TestSimulateV1_ChainsItsBlocks(t *testing.T) {
	h := newStateHarness(t)
	blocks := simulate(t, h, map[string]any{"blockStateCalls": []any{
		map[string]any{"stateOverrides": map[common.Address]any{contract: map[string]any{"code": hexutil.Encode(tests.WritesStorage)}},
			"calls": []any{map[string]any{"to": contract}}},
		map[string]any{"blockOverrides": map[string]any{"blockNumber": "0x4"},
			"calls": []any{map[string]any{"to": hasher, "input": hexutil.Encode(common.BigToHash(common.Big2).Bytes())}}},
	}})
	require.Len(t, blocks, 3, "the gap at 3 is filled")
	for i, block := range blocks {
		require.Equal(t, hexutil.Uint64(2+i), block.Number)
		require.Equal(t, hexutil.Uint64(2+12*(i+1)), block.Timestamp)
		if i > 0 {
			require.Equal(t, blocks[i-1].Hash, block.ParentHash)
		}
	}
	require.Empty(t, blocks[1].Calls)
	require.Equal(t, blocks[0].Hash.Bytes(), []byte(blocks[2].Calls[0].ReturnData))
}

func TestSimulateV1_AnswersBlockHashesOfTheChain(t *testing.T) {
	h := newStateHarness(t)
	h.Core.AddBlock()
	h.SetHead(t, 3)
	blocks := simulate(t, h, map[string]any{"blockStateCalls": blocksOfCalls(
		[]any{map[string]any{"to": hasher, "input": hexutil.Encode(common.BigToHash(common.Big1).Bytes())}})})
	first, err := h.Core.Header(1)
	require.NoError(t, err)
	require.Equal(t, first.Hash.Bytes(), []byte(blocks[0].Calls[0].ReturnData))
}

func TestSimulateV1_AbortsAtTheTimeout(t *testing.T) {
	fake := newStateHarness(t).Core
	h := tests.NewHarness(t, fake, func(h *tests.Harness) {
		eth.Register(h.Dispatcher, h.Environment, h.Core)
	}, func(settings *tests.Settings) {
		settings.Exec.RPCGasCap = 1 << 40
		settings.Exec.RPCEVMTimeout = 100 * time.Millisecond
	})
	err := h.Fail(t, "eth_simulateV1", map[string]any{"blockStateCalls": blocksOfCalls([]any{map[string]any{"to": spinner}})})
	require.Equal(t, &dispatch.Error{Code: dispatch.CodeInternal, Message: "execution aborted (timeout = 100ms)"}, err)
	require.Zero(t, fake.OpenStates())
}

func TestSimulateV1_TracesTransfersIfAsked(t *testing.T) {
	h := newStateHarness(t)
	recipient := common.Address{0x42}
	for _, trace := range []bool{false, true} {
		blocks := simulate(t, h, map[string]any{"traceTransfers": trace, "blockStateCalls": blocksOfCalls(
			[]any{map[string]any{"from": funded, "to": recipient, "value": "0x5"}})})
		logs := blocks[0].Calls[0].Logs
		if !trace {
			require.Empty(t, logs)
			continue
		}
		require.Len(t, logs, 1)
		require.Equal(t, common.HexToAddress("0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"), logs[0].Address)
		require.Equal(t, []common.Hash{
			common.HexToHash("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
			common.BytesToHash(funded.Bytes()), common.BytesToHash(recipient.Bytes()),
		}, logs[0].Topics)
	}
}

func TestSimulateV1_AnswersFullTransactionsWithTheirSenders(t *testing.T) {
	h := newStateHarness(t)
	blocks := simulate(t, h, map[string]any{"returnFullTransactions": true, "blockStateCalls": blocksOfCalls(
		[]any{map[string]any{"from": funded, "to": contract}})})
	var transaction struct {
		From  common.Address `json:"from"`
		Nonce hexutil.Uint64 `json:"nonce"`
	}
	require.NoError(t, json.Unmarshal(blocks[0].Transactions[0], &transaction))
	require.Equal(t, funded, transaction.From)
	require.Equal(t, hexutil.Uint64(7), transaction.Nonce, "the nonce is the sender's")
}

func TestSimulateV1_RefusesWhatItCannotSimulate(t *testing.T) {
	h := newStateHarness(t)
	call := map[string]any{"to": contract}
	manyCalls := func(count int) []any {
		calls := make([]any, count)
		for i := range calls {
			calls[i] = call
		}
		return calls
	}
	cases := map[string]struct {
		options map[string]any
		block   any
		err     *dispatch.Error
	}{
		"no blocks": {options: map[string]any{"blockStateCalls": []any{}},
			err: &dispatch.Error{Code: dispatch.CodeInvalidParams, Message: "empty input"}},
		"too many blocks": {options: map[string]any{"blockStateCalls": make([]any, 257)},
			err: &dispatch.Error{Code: -38026, Message: "too many blocks"}},
		"too many calls in a block": {options: map[string]any{"blockStateCalls": blocksOfCalls(manyCalls(5001))},
			err: &dispatch.Error{Code: -38026, Message: "too many calls in block: 5001 > 5000"}},
		"too many calls": {options: map[string]any{"blockStateCalls": blocksOfCalls(manyCalls(5000), manyCalls(5000), manyCalls(1))},
			err: &dispatch.Error{Code: -38026, Message: "too many calls: 10001 > 10000"}},
		"a number too far": {options: map[string]any{"blockStateCalls": []any{map[string]any{"blockOverrides": map[string]any{"blockNumber": "0x102"}}}},
			err: &dispatch.Error{Code: -38026, Message: "too many blocks"}},
		"numbers out of order": {options: map[string]any{"blockStateCalls": []any{map[string]any{"blockOverrides": map[string]any{"blockNumber": "0x1"}}}},
			err: &dispatch.Error{Code: -38020, Message: "block numbers must be in order: 1 <= 1"}},
		"times out of order": {options: map[string]any{"blockStateCalls": []any{map[string]any{"blockOverrides": map[string]any{"timestamp": "0x2"}}}},
			err: &dispatch.Error{Code: -38021, Message: "block timestamps must be in order: 2 <= 2"}},
		"block gas limit": {options: map[string]any{"blockStateCalls": []any{map[string]any{"blockOverrides": map[string]any{"gasLimit": "0x5208"},
			"calls": []any{map[string]any{"to": contract, "gas": "0x5209"}}}}},
			err: &dispatch.Error{Code: -38015, Message: "block gas limit reached: 21001 >= 21000"}},
		"nonce too low": {options: map[string]any{"validation": true, "blockStateCalls": blocksOfCalls(
			[]any{map[string]any{"from": funded, "to": contract, "nonce": "0x1"}})},
			err: &dispatch.Error{Code: -38010, Message: "err: nonce too low: address " + funded.Hex() + ", tx: 1 state: 7 (supplied gas 50000000)"}},
		"nonce too high": {options: map[string]any{"validation": true, "blockStateCalls": blocksOfCalls(
			[]any{map[string]any{"from": funded, "to": contract, "nonce": "0x9"}})},
			err: &dispatch.Error{Code: -38011, Message: "err: nonce too high: address " + funded.Hex() + ", tx: 9 state: 7 (supplied gas 50000000)"}},
		"insufficient funds": {options: map[string]any{"blockStateCalls": blocksOfCalls(
			[]any{map[string]any{"from": common.Address{0x99}, "to": contract, "value": "0x1"}})},
			err: &dispatch.Error{Code: -38014, Message: "err: insufficient funds for gas * price + value: address 0x9900000000000000000000000000000000000000 have 0 want 1 (supplied gas 50000000)"}},
		"fee cap below the base fee": {options: map[string]any{"validation": true, "blockStateCalls": blocksOfCalls(
			[]any{map[string]any{"from": funded, "to": contract, "maxFeePerGas": "0x1"}})},
			err: &dispatch.Error{Code: dispatch.CodeInvalidParams, Message: "err: max fee per gas less than block base fee: address " + funded.Hex() + ", maxFeePerGas: 1, baseFee: 1000000000 (supplied gas 50000000)"}},
		"intrinsic gas": {options: map[string]any{"blockStateCalls": blocksOfCalls([]any{map[string]any{"to": contract, "gas": "0x1"}})},
			err: &dispatch.Error{Code: -38013, Message: "err: intrinsic gas too low: have 1, want 21000 (supplied gas 1)"}},
		"both pricings": {options: map[string]any{"blockStateCalls": blocksOfCalls([]any{map[string]any{"to": contract, "gasPrice": "0x1", "maxFeePerGas": "0x1"}})},
			err: &dispatch.Error{Code: dispatch.CodeDefault, Message: "both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified"}},
		"blob creation": {options: map[string]any{"blockStateCalls": blocksOfCalls([]any{map[string]any{"blobVersionedHashes": []string{}}})},
			err: &dispatch.Error{Code: dispatch.CodeDefault, Message: "blob transaction of type create"}},
		"missing block": {options: map[string]any{"blockStateCalls": blocksOfCalls([]any{call})}, block: "0x2",
			err: &dispatch.Error{Code: dispatch.CodeDefault, Message: "header not found"}},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			params := []any{test.options}
			if test.block != nil {
				params = append(params, test.block)
			}
			require.Equal(t, test.err, h.Fail(t, "eth_simulateV1", params...))
			require.Zero(t, h.Core.OpenStates())
		})
	}
}
