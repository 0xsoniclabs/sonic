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
	"testing"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type blockForm struct {
	Number       *hexutil.Big    `json:"number"`
	Hash         common.Hash     `json:"hash"`
	ReceiptsRoot common.Hash     `json:"receiptsRoot"`
	Transactions []any           `json:"transactions"`
	Uncles       []common.Hash   `json:"uncles"`
	Size         *hexutil.Uint64 `json:"size"`
}

func TestGetBlockByNumber_AnswersEachSelector(t *testing.T) {
	chain := newTestChain(t)
	h := newHarness(t, chain.fake)
	cases := map[string]struct {
		number any
		want   int64 // -1 for null
	}{
		"latest":                     {"latest", 2},
		"pending is latest":          {"pending", 2},
		"safe is latest":             {"safe", 2},
		"finalized is latest":        {"finalized", 2},
		"earliest":                   {"earliest", 0},
		"past":                       {"0x1", 1},
		"stored above the head":      {"0x3", -1},
		"beyond the stored chain":    {"0x99", -1},
		"largest number of the type": {"0x7fffffffffffffff", -1},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			var block *blockForm
			h.Call(t, &block, "eth_getBlockByNumber", test.number, false)
			if test.want < 0 {
				require.Nil(t, block)
				return
			}
			require.Equal(t, test.want, block.Number.ToInt().Int64())
			require.Equal(t, []common.Hash{}, block.Uncles)
			require.NotNil(t, block.Size)
		})
	}
}

func TestGetBlockByHash_AnswersKnownHashesOnly(t *testing.T) {
	chain := newTestChain(t)
	h := newHarness(t, chain.fake)
	cases := map[string]struct {
		hash common.Hash
		want int64
	}{
		"known":                 {chain.hashOf(t, 1), 1},
		"unknown":               {common.Hash{0x01}, -1},
		"stored above the head": {chain.unserved, -1},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			var block *blockForm
			h.Call(t, &block, "eth_getBlockByHash", test.hash, false)
			if test.want < 0 {
				require.Nil(t, block)
				return
			}
			require.Equal(t, test.want, block.Number.ToInt().Int64())
		})
	}
}

func TestGetBlockByNumber_FormsTransactionsAsHashesOrInFull(t *testing.T) {
	chain := newTestChain(t)
	h := newHarness(t, chain.fake)

	var hashes blockForm
	h.Call(t, &hashes, "eth_getBlockByNumber", "0x1", false)
	require.Len(t, hashes.Transactions, 3)
	require.Equal(t, chain.transactions[0].Hash().Hex(), hashes.Transactions[0])

	var full struct {
		Transactions []struct {
			Hash             common.Hash    `json:"hash"`
			From             common.Address `json:"from"`
			BlockHash        common.Hash    `json:"blockHash"`
			TransactionIndex hexutil.Uint64 `json:"transactionIndex"`
			GasPrice         *hexutil.Big   `json:"gasPrice"`
		} `json:"transactions"`
	}
	h.Call(t, &full, "eth_getBlockByNumber", "0x1", true)
	require.Len(t, full.Transactions, 3)
	for i, transaction := range full.Transactions {
		require.Equal(t, chain.transactions[i].Hash(), transaction.Hash)
		require.Equal(t, chain.account.Address(), transaction.From)
		require.Equal(t, chain.hashOf(t, 1), transaction.BlockHash)
		require.Equal(t, hexutil.Uint64(i), transaction.TransactionIndex)
	}
	// The dynamic fee transaction pays the base fee of 1 gwei and its tip of
	// 1 gwei, below its fee cap of 3 gwei.
	require.Equal(t, int64(2e9), full.Transactions[2].GasPrice.ToInt().Int64())
}

func TestGetBlockByNumber_TakesTheReceiptsOfTheBlock(t *testing.T) {
	chain := newTestChain(t)
	h := newHarness(t, chain.fake)
	receipts, err := chain.fake.Receipts(1)
	require.NoError(t, err)

	var block blockForm
	h.Call(t, &block, "eth_getBlockByNumber", "0x1", false)
	require.Equal(t, types.DeriveSha(receipts, trieHasher()), block.ReceiptsRoot)

	chain.fake.SetMode("Receipts", tests.Fail)
	require.Equal(t, tests.ErrInjected.Error(), h.Fail(t, "eth_getBlockByNumber", "0x1", false).Message)
}

func TestGetBlockByNumber_RequiresTheTransactionFlag(t *testing.T) {
	h := newHarness(t, newTestChain(t).fake)
	require.Equal(t, &dispatch.Error{Code: dispatch.CodeInvalidParams, Message: "missing value for required argument 1"},
		h.Fail(t, "eth_getBlockByNumber", "latest"))
}

func TestGetHeaderByNumber_ReadsNoTransactions(t *testing.T) {
	chain := newTestChain(t)
	h := newHarness(t, chain.fake)
	var header struct {
		Number *hexutil.Big `json:"number"`
	}
	blocks := chain.fake.Calls("Block")
	h.Call(t, &header, "eth_getHeaderByNumber", "0x1")
	require.Equal(t, int64(1), header.Number.ToInt().Int64())
	require.Equal(t, blocks, chain.fake.Calls("Block"))
}

func TestGetHeader_AnswersMissingBlocks(t *testing.T) {
	chain := newTestChain(t)
	h := newHarness(t, chain.fake)
	require.Equal(t, "null", h.Result(t, "eth_getHeaderByNumber", "0x3"))
	require.Equal(t, "null", h.Result(t, "eth_getHeaderByNumber", "0x99"))

	unknown := common.Hash{0x01}
	require.Equal(t, "block with hash "+unknown.Hex()+" not found", h.Fail(t, "eth_getHeaderByHash", unknown).Message)
	require.Equal(t, "block with hash "+chain.unserved.Hex()+" not found", h.Fail(t, "eth_getHeaderByHash", chain.unserved).Message)

	chain.fake.SetMode("Receipts", tests.NotFound)
	require.Equal(t, "null", h.Result(t, "eth_getHeaderByHash", chain.hashOf(t, 1)))
}

func TestGetHeaderByHash_AnswersAKnownHash(t *testing.T) {
	chain := newTestChain(t)
	h := newHarness(t, chain.fake)
	var header struct {
		Hash common.Hash `json:"hash"`
	}
	h.Call(t, &header, "eth_getHeaderByHash", chain.hashOf(t, 1))
	require.Equal(t, chain.hashOf(t, 1), header.Hash)
}

func TestCounts_AnswerNullForAMissingOrFailingBlock(t *testing.T) {
	chain := newTestChain(t)
	h := newHarness(t, chain.fake)
	cases := map[string]struct {
		method string
		param  any
		want   string
	}{
		"transactions by number":     {"eth_getBlockTransactionCountByNumber", "0x1", `"0x3"`},
		"transactions by hash":       {"eth_getBlockTransactionCountByHash", chain.hashOf(t, 1), `"0x3"`},
		"transactions above head":    {"eth_getBlockTransactionCountByNumber", "0x3", "null"},
		"transactions unknown hash":  {"eth_getBlockTransactionCountByHash", common.Hash{0x01}, "null"},
		"uncles by number":           {"eth_getUncleCountByBlockNumber", "latest", `"0x0"`},
		"uncles by hash":             {"eth_getUncleCountByBlockHash", chain.hashOf(t, 1), `"0x0"`},
		"uncles of a missing block":  {"eth_getUncleCountByBlockNumber", "0x99", "null"},
		"uncles of an unknown block": {"eth_getUncleCountByBlockHash", common.Hash{0x01}, "null"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.want, h.Result(t, test.method, test.param))
		})
	}
	chain.fake.SetMode("Block", tests.Fail)
	require.Equal(t, "null", h.Result(t, "eth_getBlockTransactionCountByNumber", "0x1"))
}

func TestUncles_AreNeverThere(t *testing.T) {
	chain := newTestChain(t)
	h := newHarness(t, chain.fake)
	require.Equal(t, "null", h.Result(t, "eth_getUncleByBlockNumberAndIndex", "latest", "0x0"))
	require.Equal(t, "null", h.Result(t, "eth_getUncleByBlockHashAndIndex", chain.hashOf(t, 1), "0x0"))
}
