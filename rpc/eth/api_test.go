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

package eth

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

// newHarness serves the eth and ftm namespaces over the core.
func newHarness(t *testing.T, fc *tests.FakeCore, mutate ...func(*dispatch.Config)) *tests.Harness {
	t.Helper()
	return tests.NewEnvHarness(t, fc, func(d *dispatch.Dispatcher, env *exec.Env) {
		Register(d, env, "eth")
		Register(d, env, "ftm")
	}, mutate...)
}

func TestEth_ChainId_AndBlockNumber(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(3))
	h := newHarness(t, fc)

	var id hexutil.Big
	h.Into(&id, "eth_chainId")
	require.Equal(t, int64(opera.FakeNetworkID), id.ToInt().Int64())

	var n hexutil.Uint64
	h.Into(&n, "eth_blockNumber")
	require.EqualValues(t, 3, n)

	// The ftm namespace mirrors eth.
	h.Into(&n, "ftm_blockNumber")
	require.EqualValues(t, 3, n)
}

func TestEth_BlockNumber_FollowsTheHeadFeed(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	h := newHarness(t, fc)

	fc.AddBlock()
	h.WaitHead(fc.AddBlock().NumberU64())

	var n hexutil.Uint64
	h.Into(&n, "eth_blockNumber")
	require.EqualValues(t, 3, n)
}

func TestEth_GetBlockByNumber(t *testing.T) {
	tx := tests.Tx(t, 0)
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	fc.AddBlock(tx)
	h := newHarness(t, fc)

	for _, tag := range []any{"latest", "pending", "safe", "finalized", "0x2", hexutil.Uint64(2)} {
		var block map[string]any
		h.Into(&block, "eth_getBlockByNumber", tag, false)
		require.Equal(t, "0x2", block["number"], "%v", tag)
		require.Equal(t, []any{tx.Hash().Hex()}, block["transactions"])
	}

	var full map[string]any
	h.Into(&full, "eth_getBlockByNumber", "0x2", true)
	txs := full["transactions"].([]any)
	require.Len(t, txs, 1)
	require.Equal(t, tx.Hash().Hex(), txs[0].(map[string]any)["hash"])

	var earliest map[string]any
	h.Into(&earliest, "eth_getBlockByNumber", "earliest", false)
	require.Equal(t, "0x0", earliest["number"])

	// A block above the head is null, not an error.
	var missing *json.RawMessage
	h.Into(&missing, "eth_getBlockByNumber", "0x9", false)
	require.Nil(t, missing)
}

func TestEth_GetBlockByNumber_RequiresBothParameters(t *testing.T) {
	h := newHarness(t, tests.NewFakeCore(t))
	err := h.Fails("eth_getBlockByNumber", "latest")
	require.Equal(t, dispatch.CodeInvalidParams, err.Code)
	require.Equal(t, "missing value for required argument 1", err.Message)
}

func TestEth_GetBlockByHash(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(2))
	h := newHarness(t, fc)

	hash := fc.BlockAt(1).Hash
	var block map[string]any
	h.Into(&block, "eth_getBlockByHash", hash, false)
	require.Equal(t, "0x1", block["number"])

	// An unknown hash is null, as is a block above the head.
	var missing *json.RawMessage
	h.Into(&missing, "eth_getBlockByHash", common.Hash{1}, false)
	require.Nil(t, missing)
}

func TestEth_GetHeaderByHash_UnknownHashIsAnError(t *testing.T) {
	h := newHarness(t, tests.NewFakeCore(t))
	err := h.Fails("eth_getHeaderByHash", common.Hash{1})
	require.Equal(t, dispatch.CodeDefault, err.Code)
	require.Contains(t, err.Message, "block with hash "+common.Hash{1}.String()+" not found")
}

func TestEth_GetHeaderByNumber(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(2))
	h := newHarness(t, fc)
	var header map[string]any
	h.Into(&header, "eth_getHeaderByNumber", "latest")
	require.Equal(t, "0x2", header["number"])
	require.NotContains(t, header, "transactions")
}

func TestEth_BlockTransactionCountAndUncles(t *testing.T) {
	fc := tests.NewFakeCore(t)
	fc.AddBlock(tests.Tx(t, 0), tests.Tx(t, 1))
	h := newHarness(t, fc)

	var count *hexutil.Uint
	h.Into(&count, "eth_getBlockTransactionCountByNumber", "latest")
	require.EqualValues(t, 2, *count)
	h.Into(&count, "eth_getBlockTransactionCountByHash", fc.BlockAt(1).Hash)
	require.EqualValues(t, 2, *count)
	h.Into(&count, "eth_getBlockTransactionCountByNumber", "0x7")
	require.Nil(t, count)

	h.Into(&count, "eth_getUncleCountByBlockNumber", "latest")
	require.EqualValues(t, 0, *count)
	h.Into(&count, "eth_getUncleCountByBlockHash", common.Hash{9})
	require.Nil(t, count)

	var uncle *json.RawMessage
	h.Into(&uncle, "eth_getUncleByBlockNumberAndIndex", "latest", "0x0")
	require.Nil(t, uncle)
	h.Into(&uncle, "eth_getUncleByBlockHashAndIndex", fc.BlockAt(1).Hash, "0x0")
	require.Nil(t, uncle)
}

func TestEth_TransactionByIndexAndRaw(t *testing.T) {
	tx := tests.Tx(t, 0)
	fc := tests.NewFakeCore(t)
	block := fc.AddBlock(tx)
	h := newHarness(t, fc)

	var rpcTx map[string]any
	h.Into(&rpcTx, "eth_getTransactionByBlockNumberAndIndex", "latest", "0x0")
	require.Equal(t, tx.Hash().Hex(), rpcTx["hash"])
	require.Equal(t, "0x0", rpcTx["transactionIndex"])
	h.Into(&rpcTx, "eth_getTransactionByBlockHashAndIndex", block.Hash, "0x0")
	require.Equal(t, tx.Hash().Hex(), rpcTx["hash"])

	var none *json.RawMessage
	h.Into(&none, "eth_getTransactionByBlockNumberAndIndex", "latest", "0x5")
	require.Nil(t, none)

	want, err := tx.MarshalBinary()
	require.NoError(t, err)
	var raw hexutil.Bytes
	h.Into(&raw, "eth_getRawTransactionByBlockNumberAndIndex", "latest", "0x0")
	require.Equal(t, hexutil.Bytes(want), raw)
	h.Into(&raw, "eth_getRawTransactionByBlockHashAndIndex", block.Hash, "0x0")
	require.Equal(t, hexutil.Bytes(want), raw)

	// An index out of range or an unknown block yields the empty bytes, as the
	// node's earlier handlers did, not null.
	require.JSONEq(t, `"0x"`, string(h.Result("eth_getRawTransactionByBlockNumberAndIndex", "latest", "0x5")))
	require.JSONEq(t, `"0x"`, string(h.Result("eth_getRawTransactionByBlockNumberAndIndex", "0x99", "0x0")))
}

func TestEth_GetTransactionByHash_ChainAndPool(t *testing.T) {
	mined, pending := tests.Tx(t, 0), tests.Tx(t, 1)
	fc := tests.NewFakeCore(t)
	fc.AddBlock(mined)
	fc.AddPoolTransaction(pending)
	h := newHarness(t, fc)

	var rpcTx map[string]any
	h.Into(&rpcTx, "eth_getTransactionByHash", mined.Hash())
	require.Equal(t, "0x1", rpcTx["blockNumber"])
	require.Equal(t, mined.Hash().Hex(), rpcTx["hash"])

	h.Into(&rpcTx, "eth_getTransactionByHash", pending.Hash())
	require.Nil(t, rpcTx["blockNumber"])
	require.Nil(t, rpcTx["blockHash"])
	require.Equal(t, pending.Hash().Hex(), rpcTx["hash"])

	var none *json.RawMessage
	h.Into(&none, "eth_getTransactionByHash", common.Hash{7})
	require.Nil(t, none)
}

func TestEth_GetTransactionByHash_HidesTransactionsAboveTheHead(t *testing.T) {
	tx := tests.Tx(t, 0)
	fc := tests.NewFakeCore(t)
	h := newHarness(t, fc)
	fc.AddBlockSilently(tx) // not announced: the module does not serve it yet

	var none *json.RawMessage
	h.Into(&none, "eth_getTransactionByHash", tx.Hash())
	require.Nil(t, none)
}

func TestEth_GetRawTransactionByHash(t *testing.T) {
	mined, pending := tests.Tx(t, 0), tests.Tx(t, 1)
	fc := tests.NewFakeCore(t)
	fc.AddBlock(mined)
	fc.AddPoolTransaction(pending)
	h := newHarness(t, fc)

	for _, tx := range []*types.Transaction{mined, pending} {
		want, err := tx.MarshalBinary()
		require.NoError(t, err)
		var raw hexutil.Bytes
		h.Into(&raw, "eth_getRawTransactionByHash", tx.Hash())
		require.Equal(t, hexutil.Bytes(want), raw)
	}
	// An unknown transaction yields the empty bytes, as the node's earlier
	// handlers did, not null.
	require.JSONEq(t, `"0x"`, string(h.Result("eth_getRawTransactionByHash", common.Hash{7})))
}

func TestEth_GetTransactionReceipt(t *testing.T) {
	tx := tests.Tx(t, 0)
	fc := tests.NewFakeCore(t)
	block := fc.AddBlock(tx)
	h := newHarness(t, fc)

	var receipt map[string]any
	h.Into(&receipt, "eth_getTransactionReceipt", tx.Hash())
	require.Equal(t, tx.Hash().Hex(), receipt["transactionHash"])
	require.Equal(t, block.Hash.Hex(), receipt["blockHash"])
	require.Equal(t, "0x1", receipt["blockNumber"])
	require.Equal(t, "0x1", receipt["status"])
	require.Equal(t, "0x5208", receipt["gasUsed"])
	require.Equal(t, []any{}, receipt["logs"])
	require.Nil(t, receipt["contractAddress"])

	var none *json.RawMessage
	h.Into(&none, "eth_getTransactionReceipt", common.Hash{7})
	require.Nil(t, none)
}

func TestEth_GetBlockReceipts(t *testing.T) {
	a, b := tests.Tx(t, 0), tests.Tx(t, 1)
	fc := tests.NewFakeCore(t)
	block := fc.AddBlock(a, b)
	h := newHarness(t, fc)

	var receipts []map[string]any
	h.Into(&receipts, "eth_getBlockReceipts", "latest")
	require.Len(t, receipts, 2)
	require.Equal(t, a.Hash().Hex(), receipts[0]["transactionHash"])
	require.Equal(t, "0x1", receipts[1]["transactionIndex"])

	h.Into(&receipts, "eth_getBlockReceipts", block.Hash)
	require.Len(t, receipts, 2)

	h.Into(&receipts, "eth_getBlockReceipts", map[string]any{"blockHash": block.Hash})
	require.Len(t, receipts, 2)

	err := h.Fails("eth_getBlockReceipts", common.Hash{4})
	require.Contains(t, err.Message, "not found")
}

func TestEth_Syncing(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	h := newHarness(t, fc)

	// The newest block is recent (the fake chain was just built): not syncing.
	fc.AddBlock()
	require.JSONEq(t, `false`, string(h.Result("eth_syncing")))

	fc.SetProgress(core.Progress{CurrentBlock: 5, HighestBlock: 100, CurrentEpoch: 2, HighestEpoch: 3, CurrentBlockTime: 1})
	var res map[string]any
	h.Into(&res, "eth_syncing")
	require.Equal(t, "0x5", res["currentBlock"])
	require.Equal(t, "0x64", res["highestBlock"])
}

func TestEth_Constants(t *testing.T) {
	h := newHarness(t, tests.NewFakeCore(t))
	require.JSONEq(t, `"0x0000000000000000000000000000000000000000"`, string(h.Result("eth_etherbase")))
	require.JSONEq(t, `"0x0000000000000000000000000000000000000000"`, string(h.Result("eth_coinbase")))
	require.JSONEq(t, `"0x0"`, string(h.Result("eth_hashrate")))
}

func TestEth_Rules_Epoch_Config(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(2))
	fc.SetProgress(core.Progress{CurrentEpoch: 7})
	h := newHarness(t, fc)

	var rules opera.Rules
	h.Into(&rules, "eth_getRules", "latest")
	require.Equal(t, opera.FakeNetworkID, rules.NetworkID)

	var none *json.RawMessage
	h.Into(&none, "eth_getRules", "0x99")
	require.Nil(t, none)

	require.JSONEq(t, `"0x7"`, string(h.Result("eth_currentEpoch")))

	var cfg map[string]any
	h.Into(&cfg, "eth_config")
	current := cfg["current"].(map[string]any)
	require.Equal(t, "0x0", current["blockHeight"])
	require.NotEmpty(t, current["forkId"])
	require.Nil(t, cfg["next"])
	require.Nil(t, cfg["last"])
}

func TestEth_CoreFailuresAreNotPanics(t *testing.T) {
	for _, mode := range []tests.Mode{tests.Fail, tests.NotFound, tests.Nil} {
		fc := tests.NewFakeCore(t, tests.WithBlocks(2))
		fc.AddBlock(tests.Tx(t, 0))
		h := newHarness(t, fc)
		fc.Inject("*", mode)
		for _, call := range []struct {
			method string
			params []any
		}{
			{"eth_chainId", nil},
			{"eth_blockNumber", nil},
			{"eth_syncing", nil},
			{"eth_getBlockByNumber", []any{"latest", true}},
			{"eth_getBlockByHash", []any{common.Hash{1}, true}},
			{"eth_getHeaderByNumber", []any{"latest"}},
			{"eth_getHeaderByHash", []any{common.Hash{1}}},
			{"eth_getBlockReceipts", []any{"latest"}},
			{"eth_getBlockTransactionCountByNumber", []any{"latest"}},
			{"eth_getUncleByBlockNumberAndIndex", []any{"latest", "0x0"}},
			{"eth_getTransactionByBlockNumberAndIndex", []any{"latest", "0x0"}},
			{"eth_getRawTransactionByBlockNumberAndIndex", []any{"latest", "0x0"}},
			{"eth_getTransactionByHash", []any{common.Hash{1}}},
			{"eth_getRawTransactionByHash", []any{common.Hash{1}}},
			{"eth_getTransactionReceipt", []any{common.Hash{1}}},
			{"eth_getRules", []any{"latest"}},
			{"eth_currentEpoch", nil},
			{"eth_config", nil},
		} {
			_, _ = h.Call(call.method, call.params...)
		}
		require.Zero(t, h.Panics(), "mode %v", mode)
	}
}

func TestEth_TxLookupFailuresAreReported(t *testing.T) {
	// A node that does not index transactions fails the lookups, as the
	// reference does; an unknown transaction is not an error.
	fc := tests.NewFakeCore(t, tests.WithBlocks(2))
	h := newHarness(t, fc)
	require.JSONEq(t, `null`, string(h.Result("eth_getTransactionByHash", common.Hash{1})))
	require.JSONEq(t, `"0x"`, string(h.Result("eth_getRawTransactionByHash", common.Hash{1})))
	require.JSONEq(t, `null`, string(h.Result("eth_getTransactionReceipt", common.Hash{1})))
	fc.Inject("TxByHash", tests.Fail)
	for _, method := range []string{"eth_getTransactionByHash", "eth_getRawTransactionByHash", "eth_getTransactionReceipt"} {
		err := h.Fails(method, common.Hash{1})
		require.Equal(t, tests.ErrInjected.Error(), err.Message, method)
	}
}

func TestEth_CorePanicsAreRecovered(t *testing.T) {
	for method, call := range map[string][]any{
		"Block":             {"eth_getBlockByNumber", "latest", true},
		"Header":            {"eth_getBlockReceipts", "0x1"},
		"Receipts":          {"eth_getBlockReceipts", "latest"},
		"ChainInfo":         {"eth_chainId"},
		"BlockNumberByHash": {"eth_getBlockByHash", common.Hash{1}, true},
		"Progress":          {"eth_currentEpoch"},
		"Rules":             {"eth_getRules", "0x1"},
		"TxByHash":          {"eth_getTransactionByHash", common.Hash{1}},
		"PoolTransaction":   {"eth_getTransactionByHash", common.Hash{1}},
	} {
		fc := tests.NewFakeCore(t, tests.WithBlocks(2))
		h := newHarness(t, fc)
		fc.Inject(method, tests.Panic)
		err := h.Fails(call[0].(string), call[1:]...)
		require.Equal(t, dispatch.CodeInternal, err.Code, method)
		require.Equal(t, "method handler crashed", err.Message, method)
		require.EqualValues(t, 1, h.Panics(), method)
	}
}
