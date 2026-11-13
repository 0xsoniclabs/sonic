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

	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type transactionForm struct {
	Hash             common.Hash     `json:"hash"`
	BlockHash        *common.Hash    `json:"blockHash"`
	BlockNumber      *hexutil.Big    `json:"blockNumber"`
	TransactionIndex *hexutil.Uint64 `json:"transactionIndex"`
	GasPrice         *hexutil.Big    `json:"gasPrice"`
	ChainID          *hexutil.Big    `json:"chainId"`
}

func TestGetTransactionByHash_AnswersFromBlocksThenThePool(t *testing.T) {
	chain := newTestChain(t)
	pending := chain.account.Sign(&types.DynamicFeeTx{ChainID: big.NewInt(tests.ChainID), Nonce: 3, Gas: 21000, GasFeeCap: big.NewInt(5e9), GasTipCap: big.NewInt(1e9)})
	chain.fake.AddPoolTransaction(pending)
	h := newHarness(t, chain.fake)

	var mined transactionForm
	h.Call(t, &mined, "eth_getTransactionByHash", chain.transactions[2].Hash())
	require.Equal(t, chain.hashOf(t, 1), *mined.BlockHash)
	require.Equal(t, int64(1), mined.BlockNumber.ToInt().Int64())
	require.Equal(t, hexutil.Uint64(2), *mined.TransactionIndex)
	require.Equal(t, int64(2e9), mined.GasPrice.ToInt().Int64())
	require.Equal(t, int64(tests.ChainID), mined.ChainID.ToInt().Int64())

	var pooled transactionForm
	h.Call(t, &pooled, "eth_getTransactionByHash", pending.Hash())
	require.Nil(t, pooled.BlockHash)
	require.Nil(t, pooled.TransactionIndex)
	require.Equal(t, int64(5e9), pooled.GasPrice.ToInt().Int64(), "a pending transaction is priced by its fee cap")

	require.Equal(t, "null", h.Result(t, "eth_getTransactionByHash", common.Hash{0x01}))
}

func TestGetTransactionByHash_PassesOnAFailedLookup(t *testing.T) {
	chain := newTestChain(t)
	h := newHarness(t, chain.fake)
	chain.fake.SetMode("TransactionByHash", tests.Fail)
	require.Equal(t, tests.ErrInjected.Error(), h.Fail(t, "eth_getTransactionByHash", chain.transactions[0].Hash()).Message)
	require.Equal(t, tests.ErrInjected.Error(), h.Fail(t, "eth_getRawTransactionByHash", chain.transactions[0].Hash()).Message)
	require.Equal(t, tests.ErrInjected.Error(), h.Fail(t, "eth_getTransactionReceipt", chain.transactions[0].Hash()).Message)
}

func TestGetRawTransactionByHash_AnswersTheBinaryForm(t *testing.T) {
	chain := newTestChain(t)
	pending := chain.account.Sign(&types.LegacyTx{Nonce: 3, Gas: 21000, GasPrice: big.NewInt(1e9)})
	chain.fake.AddPoolTransaction(pending)
	h := newHarness(t, chain.fake)
	cases := map[string]struct {
		hash common.Hash
		want *types.Transaction
	}{
		"mined":   {chain.transactions[1].Hash(), chain.transactions[1]},
		"pending": {pending.Hash(), pending},
		"unknown": {common.Hash{0x01}, nil},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			var raw hexutil.Bytes
			h.Call(t, &raw, "eth_getRawTransactionByHash", test.hash)
			if test.want == nil {
				require.Empty(t, raw)
				return
			}
			want, err := test.want.MarshalBinary()
			require.NoError(t, err)
			require.Equal(t, hexutil.Bytes(want), raw)
		})
	}
}

func TestTransactionByBlockAndIndex_AnswersNullForAMissingOne(t *testing.T) {
	chain := newTestChain(t)
	h := newHarness(t, chain.fake)
	cases := map[string]struct {
		method string
		block  any
		index  string
		want   *common.Hash
	}{
		"by number":          {"eth_getTransactionByBlockNumberAndIndex", "0x1", "0x1", pointer(chain.transactions[1].Hash())},
		"by hash":            {"eth_getTransactionByBlockHashAndIndex", chain.hashOf(t, 1), "0x1", pointer(chain.transactions[1].Hash())},
		"index out of range": {"eth_getTransactionByBlockNumberAndIndex", "0x1", "0x3", nil},
		"missing block":      {"eth_getTransactionByBlockNumberAndIndex", "0x3", "0x0", nil},
		"unknown hash":       {"eth_getTransactionByBlockHashAndIndex", common.Hash{0x01}, "0x0", nil},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			var transaction *transactionForm
			h.Call(t, &transaction, test.method, test.block, test.index)
			if test.want == nil {
				require.Nil(t, transaction)
				return
			}
			require.Equal(t, *test.want, transaction.Hash)
		})
	}
}

func TestRawTransactionByBlockAndIndex_AnswersEmptyBytesForAMissingOne(t *testing.T) {
	chain := newTestChain(t)
	h := newHarness(t, chain.fake)
	raw, err := chain.transactions[1].MarshalBinary()
	require.NoError(t, err)
	cases := map[string]struct {
		method string
		block  any
		index  string
		want   string
	}{
		"by number":          {"eth_getRawTransactionByBlockNumberAndIndex", "0x1", "0x1", hexutil.Encode(raw)},
		"by hash":            {"eth_getRawTransactionByBlockHashAndIndex", chain.hashOf(t, 1), "0x1", hexutil.Encode(raw)},
		"index out of range": {"eth_getRawTransactionByBlockNumberAndIndex", "0x1", "0x3", "0x"},
		"unknown block":      {"eth_getRawTransactionByBlockHashAndIndex", common.Hash{0x01}, "0x0", "0x"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, `"`+test.want+`"`, h.Result(t, test.method, test.block, test.index))
		})
	}
}

func pointer[Value any](value Value) *Value {
	return &value
}
