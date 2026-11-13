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

	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type receiptForm struct {
	TransactionHash   common.Hash     `json:"transactionHash"`
	TransactionIndex  hexutil.Uint64  `json:"transactionIndex"`
	BlockHash         common.Hash     `json:"blockHash"`
	From              common.Address  `json:"from"`
	To                *common.Address `json:"to"`
	Status            *hexutil.Uint   `json:"status"`
	GasUsed           hexutil.Uint64  `json:"gasUsed"`
	EffectiveGasPrice hexutil.Uint64  `json:"effectiveGasPrice"`
	ContractAddress   *common.Address `json:"contractAddress"`
	Logs              []*types.Log    `json:"logs"`
	Type              hexutil.Uint    `json:"type"`
}

func TestGetTransactionReceipt_FormsTheReceipt(t *testing.T) {
	chain := newTestChain(t)
	h := newHarness(t, chain.fake)

	var call receiptForm
	h.Call(t, &call, "eth_getTransactionReceipt", chain.transactions[0].Hash())
	require.Equal(t, chain.transactions[0].Hash(), call.TransactionHash)
	require.Equal(t, chain.hashOf(t, 1), call.BlockHash)
	require.Equal(t, chain.account.Address(), call.From)
	require.Equal(t, hexutil.Uint(types.ReceiptStatusSuccessful), *call.Status)
	require.Equal(t, hexutil.Uint64(21000), call.GasUsed)
	require.Equal(t, hexutil.Uint64(2e9), call.EffectiveGasPrice)
	require.Nil(t, call.ContractAddress)
	require.Equal(t, []*types.Log{}, call.Logs)

	var creation receiptForm
	h.Call(t, &creation, "eth_getTransactionReceipt", chain.transactions[2].Hash())
	require.Equal(t, hexutil.Uint64(2), creation.TransactionIndex)
	require.Nil(t, creation.To)
	require.NotNil(t, creation.ContractAddress, "a creation names its contract")
	require.Equal(t, hexutil.Uint(types.DynamicFeeTxType), creation.Type)
}

func TestGetTransactionReceipt_AnswersNullForAMissingOne(t *testing.T) {
	chain := newTestChain(t)
	h := newHarness(t, chain.fake)
	require.Equal(t, "null", h.Result(t, "eth_getTransactionReceipt", common.Hash{0x01}))

	chain.fake.SetMode("Receipts", tests.NotFound)
	require.Equal(t, "null", h.Result(t, "eth_getTransactionReceipt", chain.transactions[0].Hash()))
	chain.fake.SetMode("Receipts", tests.Fail)
	require.Equal(t, tests.ErrInjected.Error(), h.Fail(t, "eth_getTransactionReceipt", chain.transactions[0].Hash()).Message)
}

func TestGetTransactionReceipt_ReadsNoBlock(t *testing.T) {
	chain := newTestChain(t)
	h := newHarness(t, chain.fake)
	blocks := chain.fake.Calls("Block")
	h.Result(t, "eth_getTransactionReceipt", chain.transactions[0].Hash())
	require.Equal(t, blocks, chain.fake.Calls("Block"))
}

func TestGetBlockReceipts_AnswersEachReference(t *testing.T) {
	chain := newTestChain(t)
	h := newHarness(t, chain.fake)
	cases := map[string]struct {
		block any
		want  int // -1 for null
	}{
		"number":                                 {"0x1", 3},
		"latest":                                 {"latest", 0},
		"hash":                                   {chain.hashOf(t, 1), 3},
		"object with a hash":                     {map[string]any{"blockHash": chain.hashOf(t, 1)}, 3},
		"above the head":                         {"0x3", -1},
		"neither number nor hash is the genesis": {map[string]any{}, 0},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			var receipts []receiptForm
			h.Call(t, &receipts, "eth_getBlockReceipts", test.block)
			if test.want < 0 {
				require.Nil(t, receipts)
				return
			}
			require.Len(t, receipts, test.want)
			for i, receipt := range receipts {
				require.Equal(t, chain.transactions[i].Hash(), receipt.TransactionHash)
				require.Equal(t, hexutil.Uint64(i), receipt.TransactionIndex)
			}
		})
	}
}

func TestGetBlockReceipts_WordsAnUnknownHash(t *testing.T) {
	chain := newTestChain(t)
	h := newHarness(t, chain.fake)
	unknown := common.Hash{0x01}
	require.Equal(t, "block with hash "+unknown.Hex()+" not found", h.Fail(t, "eth_getBlockReceipts", unknown).Message)
}
