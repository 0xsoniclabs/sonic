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

package tests

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/contracts/read_history_storage"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	req "github.com/stretchr/testify/require"
)

// TestTraceConsistency_TracesReproduceBlockProcessing checks that every tracing
// RPC replays a transaction against the same state the block processor used,
// and that all of them agree with each other.
//
// The probe is a transaction reading the hash of the parent of its own block
// from the EIP-2935 history storage contract. That slot is written by the
// pre-block system call of the very block the transaction ends up in, so the
// value the transaction observes tells whether the tracing entry point prepared
// its base state the way the block processor did.
//
// The ground truth is taken from the real execution of the transaction: the
// event emitted during block processing and recorded in the receipt.
func TestTraceConsistency_TracesReproduceBlockProcessing(t *testing.T) {
	require := req.New(t)

	// Allegro enables Prague, and with it the EIP-2935 pre-block system call.
	session := getIntegrationTestNetSession(t, opera.GetAllegroUpgrades())
	t.Parallel()

	client, err := session.GetClient()
	require.NoError(err)
	defer client.Close()
	rpcClient := client.Client()

	contract, receipt, err := DeployContract(session, read_history_storage.DeployReadHistoryStorage)
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	receipt = runTransactionReadingParentBlockHash(t, session, client, contract)
	txHash := receipt.TxHash

	block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
	require.NoError(err)

	// Ground truth: what the transaction actually read while the block was being
	// processed, as recorded in the receipt.
	require.Len(receipt.Logs, 1)
	event, err := contract.ParseBlockHash(*receipt.Logs[0])
	require.NoError(err)
	require.Equal(receipt.BlockNumber.Uint64()-1, event.QueriedBlock.Uint64())

	observedDuringBlockProcessing := common.Hash(event.BlockHash)
	require.Equal(block.ParentHash(), observedDuringBlockProcessing,
		"history storage should hold the parent hash while the block is being processed")

	// Every tracing RPC replaying this transaction has to reproduce that value.
	t.Run("debug_traceTransaction", func(t *testing.T) {
		trace := debugTraceTransaction(t, rpcClient, txHash)
		req.New(t).Equal(observedDuringBlockProcessing, historyStorageOutput(t, trace))
	})

	t.Run("debug_traceBlockByNumber", func(t *testing.T) {
		require := req.New(t)
		entry := debugTraceBlockByNumber(t, rpcClient, receipt.BlockNumber)[receipt.TransactionIndex]
		require.Equal(txHash, entry.TxHash)
		require.Equal(observedDuringBlockProcessing, historyStorageOutput(t, entry.Result))
	})

	t.Run("trace_transaction", func(t *testing.T) {
		traces := traceTransaction(t, rpcClient, txHash)
		req.New(t).Equal(observedDuringBlockProcessing, historyStorageOutput(t, traces...))
	})

	t.Run("trace_block", func(t *testing.T) {
		traces := tracesOfTransaction(t, traceBlock(t, rpcClient, receipt.BlockNumber), txHash)
		req.New(t).Equal(observedDuringBlockProcessing, historyStorageOutput(t, traces...))
	})

	// Tracing a block and tracing its transactions one by one must produce the
	// same result, in both namespaces.
	t.Run("block traces equal transaction traces", func(t *testing.T) {
		require := req.New(t)
		txs := block.Transactions()

		debugTraces := debugTraceBlockByNumber(t, rpcClient, receipt.BlockNumber)
		require.Len(debugTraces, len(txs))
		parityTraces := traceBlock(t, rpcClient, receipt.BlockNumber)

		for i, tx := range txs {
			require.Empty(debugTraces[i].Error)
			require.Equal(tx.Hash(), debugTraces[i].TxHash)
			require.JSONEq(
				string(debugTraceTransaction(t, rpcClient, tx.Hash())),
				string(debugTraces[i].Result),
				"debug_traceBlockByNumber and debug_traceTransaction disagree on transaction %d (%s)",
				i, tx.Hash())

			require.Equal(
				traceTransaction(t, rpcClient, tx.Hash()),
				tracesOfTransaction(t, parityTraces, tx.Hash()),
				"trace_block and trace_transaction disagree on transaction %s", tx.Hash())
		}
	})
}

// runTransactionReadingParentBlockHash issues a transaction reading the hash of
// the parent of the block the transaction itself ends up in. The block a
// transaction lands in cannot be chosen in advance, so the current head is
// queried and the attempt is repeated until the transaction is included in the
// block directly following the queried one.
func runTransactionReadingParentBlockHash(
	t *testing.T,
	session IntegrationTestNetSession,
	client *PooledEhtClient,
	contract *read_history_storage.ReadHistoryStorage,
) *types.Receipt {
	t.Helper()
	require := req.New(t)

	const attempts = 10
	for range attempts {
		head, err := client.BlockNumber(t.Context())
		require.NoError(err)

		receipt, err := session.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
			return contract.ReadHistoryStorage(opts, new(big.Int).SetUint64(head))
		})
		require.NoError(err)
		require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

		if receipt.BlockNumber.Uint64() == head+1 {
			return receipt
		}
	}

	t.Fatalf("failed to place a transaction in the block directly following the queried one in %d attempts", attempts)
	return nil
}

// callRPC issues an RPC call and decodes its result, failing the test if the
// call does not succeed. Traces are kept as raw JSON wherever they are only
// passed on or compared, so that comparisons cover the whole server response
// and not just the fields declared below.
func callRPC[T any](t *testing.T, rpcClient *rpc.Client, method string, args ...any) T {
	t.Helper()
	var res T
	err := rpcClient.Call(&res, method, args...)
	req.New(t).NoErrorf(err, "failed to call %s", method)
	return res
}

// blockTraceResult mirrors the per-transaction entries returned by
// debug_traceBlockByNumber.
type blockTraceResult struct {
	TxHash common.Hash     `json:"txHash"`
	Result json.RawMessage `json:"result"`
	Error  string          `json:"error,omitempty"`
}

func debugTraceBlockByNumber(t *testing.T, rpcClient *rpc.Client, blockNumber *big.Int) []blockTraceResult {
	t.Helper()
	return callRPC[[]blockTraceResult](t, rpcClient, "debug_traceBlockByNumber",
		rpc.BlockNumber(blockNumber.Int64()), callTracerConfig())
}

func debugTraceTransaction(t *testing.T, rpcClient *rpc.Client, txHash common.Hash) json.RawMessage {
	t.Helper()
	return callRPC[json.RawMessage](t, rpcClient, "debug_traceTransaction", txHash, callTracerConfig())
}

func callTracerConfig() *ethapi.TraceCallConfig {
	tracer := "callTracer"
	return &ethapi.TraceCallConfig{TraceConfig: tracers.TraceConfig{Tracer: &tracer}}
}

func traceBlock(t *testing.T, rpcClient *rpc.Client, blockNumber *big.Int) []json.RawMessage {
	t.Helper()
	return callRPC[[]json.RawMessage](t, rpcClient, "trace_block", rpc.BlockNumber(blockNumber.Int64()))
}

func traceTransaction(t *testing.T, rpcClient *rpc.Client, txHash common.Hash) []json.RawMessage {
	t.Helper()
	return callRPC[[]json.RawMessage](t, rpcClient, "trace_transaction", txHash)
}

// traceFrame is the subset of a trace entry needed by these tests. It covers
// both flavors: the nested callTracer frames of the debug_* namespace, where
// callee and output sit in the same frame, and the flat parity style action
// traces of the trace_* namespace, where the callee is in the action and the
// output in the result.
type traceFrame struct {
	To     *common.Address `json:"to"`
	Output hexutil.Bytes   `json:"output"`
	Calls  []traceFrame    `json:"calls"`

	Action *traceFrame `json:"action"`
	Result *traceFrame `json:"result"`

	TransactionHash common.Hash `json:"transactionHash"`
}

func decodeTraceFrame(t *testing.T, trace json.RawMessage) traceFrame {
	t.Helper()
	var frame traceFrame
	req.New(t).NoError(json.Unmarshal(trace, &frame))
	return frame
}

// tracesOfTransaction filters block level action traces down to a single
// transaction.
func tracesOfTransaction(t *testing.T, traces []json.RawMessage, txHash common.Hash) []json.RawMessage {
	t.Helper()
	res := make([]json.RawMessage, 0, len(traces))
	for _, trace := range traces {
		if decodeTraceFrame(t, trace).TransactionHash == txHash {
			res = append(res, trace)
		}
	}
	return res
}

// historyStorageOutput returns the value the history storage contract returned
// to the traced transaction, as reported by the given traces.
func historyStorageOutput(t *testing.T, traces ...json.RawMessage) common.Hash {
	t.Helper()
	require := req.New(t)

	var find func(frame traceFrame) (common.Hash, bool)
	find = func(frame traceFrame) (common.Hash, bool) {
		if action := frame.Action; action != nil && action.To != nil &&
			*action.To == params.HistoryStorageAddress {
			require.NotNil(frame.Result, "history storage call has no result")
			return common.BytesToHash(frame.Result.Output), true
		}
		if frame.To != nil && *frame.To == params.HistoryStorageAddress {
			return common.BytesToHash(frame.Output), true
		}
		for _, call := range frame.Calls {
			if hash, found := find(call); found {
				return hash, true
			}
		}
		return common.Hash{}, false
	}

	for _, trace := range traces {
		if hash, found := find(decodeTraceFrame(t, trace)); found {
			return hash
		}
	}

	require.Fail("no call to the history storage contract found in the traces")
	return common.Hash{}
}
