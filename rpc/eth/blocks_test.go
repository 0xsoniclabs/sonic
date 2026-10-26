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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/format"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

// txChain is a chain whose block 1 holds two transfers, the second emitting a
// log, with a third transfer in the pool.
type txChain struct {
	fc     *tests.FakeCore
	h      *tests.Harness
	block  *evmcore.EvmBlock
	txs    types.Transactions
	pooled *types.Transaction
	log    *types.Log
}

func newTxChain(t *testing.T) *txChain {
	t.Helper()
	w := tests.NewWallet(t)
	to := common.HexToAddress("0x42")
	c := &txChain{fc: tests.NewFakeCore()}
	c.txs = types.Transactions{w.Transfer(t, 0, to, big.NewInt(1)), w.Transfer(t, 1, to, big.NewInt(2))}
	c.log = &types.Log{Address: to, Data: []byte{7}}
	c.block = c.fc.AddAndPublish(c.txs, nil, []*types.Log{c.log})
	c.pooled = w.Transfer(t, 2, to, big.NewInt(3))
	c.fc.SetPool(map[common.Address]types.Transactions{w.Address: {c.pooled}}, nil)
	c.h = newEth(t, c.fc)
	return c
}

func TestBlocks_ByNumberAndHash(t *testing.T) {
	c := newTxChain(t)
	var hashes struct {
		Number       hexutil.Uint64 `json:"number"`
		Hash         common.Hash    `json:"hash"`
		Transactions []common.Hash  `json:"transactions"`
		Uncles       []common.Hash  `json:"uncles"`
		TxRoot       common.Hash    `json:"transactionsRoot"`
		Size         hexutil.Uint64 `json:"size"`
	}
	c.h.MustCall(&hashes, "eth_getBlockByNumber", "latest", false)
	require.Equal(t, hexutil.Uint64(1), hashes.Number)
	require.Equal(t, c.block.Hash, hashes.Hash)
	require.Equal(t, []common.Hash{c.txs[0].Hash(), c.txs[1].Hash()}, hashes.Transactions)
	require.Empty(t, hashes.Uncles)
	require.NotNil(t, hashes.Uncles)
	require.Equal(t, c.block.TxHash, hashes.TxRoot)
	require.Positive(t, hashes.Size)

	var full struct {
		Transactions []format.Transaction `json:"transactions"`
	}
	c.h.MustCall(&full, "eth_getBlockByHash", c.block.Hash, true)
	require.Len(t, full.Transactions, 2)
	require.Equal(t, c.txs[1].Hash(), full.Transactions[1].Hash)
	require.Equal(t, hexutil.Uint64(1), *full.Transactions[1].TransactionIndex)

	require.Equal(t, "null", string(c.h.MustCall(nil, "eth_getBlockByNumber", "0x2", false)))
	require.Equal(t, "null", string(c.h.MustCall(nil, "eth_getBlockByHash", common.Hash{1}, false)))
}

func TestBlocks_HeadersCarryTheTransactionsRoot(t *testing.T) {
	c := newTxChain(t)
	var header struct {
		Hash   common.Hash `json:"hash"`
		TxRoot common.Hash `json:"transactionsRoot"`
		Bloom  string      `json:"logsBloom"`
	}
	c.h.MustCall(&header, "eth_getHeaderByNumber", "0x1")
	require.Equal(t, c.block.Hash, header.Hash)
	require.Equal(t, c.block.TxHash, header.TxRoot)
	require.NotEqual(t, common.Hash{}, header.TxRoot)
	c.h.MustCall(&header, "eth_getHeaderByHash", c.block.Hash)
	require.Equal(t, c.block.TxHash, header.TxRoot)

	require.Equal(t, "null", string(c.h.MustCall(nil, "eth_getHeaderByNumber", "0x5")))
	e := c.h.Fails("eth_getHeaderByHash", common.Hash{1})
	require.Equal(t, "block with hash 0x0100000000000000000000000000000000000000000000000000000000000000 not found", e.Message)
}

func TestBlocks_Receipts(t *testing.T) {
	c := newTxChain(t)
	var receipts []map[string]any
	c.h.MustCall(&receipts, "eth_getBlockReceipts", "0x1")
	require.Len(t, receipts, 2)
	require.Equal(t, c.txs[1].Hash().Hex(), receipts[1]["transactionHash"])
	require.Len(t, receipts[1]["logs"], 1)

	c.h.MustCall(&receipts, "eth_getBlockReceipts", c.block.Hash)
	require.Len(t, receipts, 2)
	require.Equal(t, "null", string(c.h.MustCall(nil, "eth_getBlockReceipts", "0x7")))
	require.Equal(t, "[]", string(c.h.MustCall(nil, "eth_getBlockReceipts", map[string]any{})), "no selector reads the genesis")
	e := c.h.Fails("eth_getBlockReceipts", common.Hash{1})
	require.Equal(t, "block with hash 0x0100000000000000000000000000000000000000000000000000000000000000 not found", e.Message)
}

func TestBlocks_CountsAndUncles(t *testing.T) {
	c := newTxChain(t)
	require.Equal(t, `"0x2"`, string(c.h.MustCall(nil, "eth_getBlockTransactionCountByNumber", "0x1")))
	require.Equal(t, `"0x2"`, string(c.h.MustCall(nil, "eth_getBlockTransactionCountByHash", c.block.Hash)))
	require.Equal(t, "null", string(c.h.MustCall(nil, "eth_getBlockTransactionCountByNumber", "0x9")))
	require.Equal(t, "null", string(c.h.MustCall(nil, "eth_getBlockTransactionCountByHash", common.Hash{1})))

	require.Equal(t, `"0x0"`, string(c.h.MustCall(nil, "eth_getUncleCountByBlockNumber", "0x1")))
	require.Equal(t, `"0x0"`, string(c.h.MustCall(nil, "eth_getUncleCountByBlockHash", c.block.Hash)))
	require.Equal(t, "null", string(c.h.MustCall(nil, "eth_getUncleCountByBlockNumber", "0x9")))
	require.Equal(t, "null", string(c.h.MustCall(nil, "eth_getUncleCountByBlockHash", common.Hash{1})))
	require.Equal(t, "null", string(c.h.MustCall(nil, "eth_getUncleByBlockNumberAndIndex", "0x1", "0x0")))
	require.Equal(t, "null", string(c.h.MustCall(nil, "eth_getUncleByBlockHashAndIndex", c.block.Hash, "0x0")))
}

// TestBlocks_AFailedLookupIsAMissingBlock keeps the reference's answer: these
// methods ignored the error of the block lookup.
func TestBlocks_AFailedLookupIsAMissingBlock(t *testing.T) {
	c := newTxChain(t)
	c.fc.SetMode("Block", tests.Fail)
	c.fc.SetMode("Header", tests.Fail)
	require.Equal(t, "null", string(c.h.MustCall(nil, "eth_getBlockTransactionCountByNumber", "0x1")))
	require.Equal(t, "null", string(c.h.MustCall(nil, "eth_getUncleCountByBlockNumber", "0x1")))
	require.Equal(t, "null", string(c.h.MustCall(nil, "eth_getTransactionByBlockNumberAndIndex", "0x1", "0x0")))
	require.Equal(t, `"0x"`, string(c.h.MustCall(nil, "eth_getRawTransactionByBlockNumberAndIndex", "0x1", "0x0")))
	e := c.h.Fails("eth_getBlockByNumber", "0x1", false)
	require.Equal(t, tests.ErrFake.Error(), e.Message, "a failure that is not absence is reported")
}

func TestTransactions_ByBlockAndIndex(t *testing.T) {
	c := newTxChain(t)
	var tx format.Transaction
	c.h.MustCall(&tx, "eth_getTransactionByBlockNumberAndIndex", "0x1", "0x1")
	require.Equal(t, c.txs[1].Hash(), tx.Hash)
	require.Equal(t, c.block.Hash, *tx.BlockHash)
	c.h.MustCall(&tx, "eth_getTransactionByBlockHashAndIndex", c.block.Hash, "0x0")
	require.Equal(t, c.txs[0].Hash(), tx.Hash)
	require.Equal(t, "null", string(c.h.MustCall(nil, "eth_getTransactionByBlockNumberAndIndex", "0x1", "0x2")))
	require.Equal(t, "null", string(c.h.MustCall(nil, "eth_getTransactionByBlockHashAndIndex", common.Hash{1}, "0x0")))

	enc, err := c.txs[1].MarshalBinary()
	require.NoError(t, err)
	var raw hexutil.Bytes
	c.h.MustCall(&raw, "eth_getRawTransactionByBlockNumberAndIndex", "0x1", "0x1")
	require.Equal(t, hexutil.Bytes(enc), raw)
	c.h.MustCall(&raw, "eth_getRawTransactionByBlockHashAndIndex", c.block.Hash, "0x1")
	require.Equal(t, hexutil.Bytes(enc), raw)
	require.Equal(t, `"0x"`, string(c.h.MustCall(nil, "eth_getRawTransactionByBlockNumberAndIndex", "0x5", "0x0")))
	require.Equal(t, `"0x"`, string(c.h.MustCall(nil, "eth_getRawTransactionByBlockHashAndIndex", c.block.Hash, "0x9")))
}

func TestTransactions_ByHashCommittedPooledOrMissing(t *testing.T) {
	c := newTxChain(t)
	var tx format.Transaction
	c.h.MustCall(&tx, "eth_getTransactionByHash", c.txs[0].Hash())
	require.Equal(t, c.block.Hash, *tx.BlockHash)
	c.h.MustCall(&tx, "eth_getTransactionByHash", c.pooled.Hash())
	require.Nil(t, tx.BlockHash)
	require.Equal(t, c.pooled.Hash(), tx.Hash)
	require.Equal(t, "null", string(c.h.MustCall(nil, "eth_getTransactionByHash", common.Hash{1})))

	var raw hexutil.Bytes
	c.h.MustCall(&raw, "eth_getRawTransactionByHash", c.txs[0].Hash())
	enc, err := c.txs[0].MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, hexutil.Bytes(enc), raw)
	c.h.MustCall(&raw, "eth_getRawTransactionByHash", c.pooled.Hash())
	enc, err = c.pooled.MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, hexutil.Bytes(enc), raw)
	require.Equal(t, `"0x"`, string(c.h.MustCall(nil, "eth_getRawTransactionByHash", common.Hash{1})))
}

func TestTransactions_Receipt(t *testing.T) {
	c := newTxChain(t)
	var receipt map[string]any
	c.h.MustCall(&receipt, "eth_getTransactionReceipt", c.txs[1].Hash())
	require.Equal(t, c.block.Hash.Hex(), receipt["blockHash"])
	require.Equal(t, "0x1", receipt["transactionIndex"])
	require.Equal(t, "0x1", receipt["status"])
	require.Nil(t, receipt["contractAddress"])
	logs := receipt["logs"].([]any)
	require.Len(t, logs, 1)
	require.Equal(t, []any{}, logs[0].(map[string]any)["topics"], "topics are never null")
	require.Equal(t, "null", string(c.h.MustCall(nil, "eth_getTransactionReceipt", c.pooled.Hash())))
}

// TestTransactions_TxIndexOffIsAnError is the adapter guard of section 2: a
// node without the transaction index answers the error, not an empty result.
func TestTransactions_TxIndexOffIsAnError(t *testing.T) {
	c := newTxChain(t)
	c.fc.SetTxIndex(false)
	e := c.h.Fails("eth_getTransactionReceipt", c.txs[0].Hash())
	require.Contains(t, e.Message, "transactions index is disabled")
	e = c.h.Fails("eth_getBlockReceipts", "0x1")
	require.Contains(t, e.Message, "transactions index is disabled")
	e = c.h.Fails("eth_getTransactionByHash", c.txs[0].Hash())
	require.Contains(t, e.Message, "transactions index is disabled")
}
