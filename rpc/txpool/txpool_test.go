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

package txpool_test

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/0xsoniclabs/sonic/rpc/txpool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// newHarness serves a pool with two pending transactions of one sender, of
// which one creates a contract, and one queued transaction of another.
func newHarness(t *testing.T) (*tests.Harness, *tests.Account, *tests.Account) {
	t.Helper()
	pending, queued := tests.NewAccount(), tests.NewAccount()
	recipient := common.Address{0x42}
	fake := tests.NewFakeCore()
	fake.AddPoolTransaction(pending.Sign(&types.LegacyTx{Nonce: 0, To: &recipient, Gas: 21000, GasPrice: big.NewInt(2), Value: big.NewInt(1)}))
	fake.AddPoolTransaction(pending.Sign(&types.DynamicFeeTx{ChainID: big.NewInt(tests.ChainID), Nonce: 1, Gas: 50000,
		GasFeeCap: big.NewInt(5), GasTipCap: big.NewInt(1)}))
	fake.AddQueuedTransaction(queued.Sign(&types.LegacyTx{Nonce: 7, To: &recipient, Gas: 21000, GasPrice: big.NewInt(3)}))
	h := tests.NewHarness(t, fake, func(h *tests.Harness) {
		txpool.Register(h.Dispatcher, h.Environment, h.Core)
	})
	return h, pending, queued
}

type transactionForm struct {
	Hash     common.Hash     `json:"hash"`
	From     common.Address  `json:"from"`
	GasPrice string          `json:"gasPrice"`
	Nonce    string          `json:"nonce"`
	To       *common.Address `json:"to"`
}

func TestContent_ListsTransactionsBySenderAndNonce(t *testing.T) {
	h, pending, queued := newHarness(t)
	var content map[string]map[common.Address]map[string]transactionForm
	h.Call(t, &content, "txpool_content")
	require.Len(t, content["pending"][pending.Address()], 2)
	require.Equal(t, "0x2", content["pending"][pending.Address()]["0"].GasPrice)
	require.Equal(t, "0x5", content["pending"][pending.Address()]["1"].GasPrice, "a pooled transaction is priced by its fee cap")
	require.Equal(t, pending.Address(), content["pending"][pending.Address()]["1"].From)
	require.Equal(t, "0x7", content["queued"][queued.Address()]["7"].Nonce)

	var from map[string]map[string]transactionForm
	h.Call(t, &from, "txpool_contentFrom", pending.Address())
	require.Equal(t, content["pending"][pending.Address()], from["pending"])
	require.Empty(t, from["queued"])
	require.Equal(t, `{"pending":{},"queued":{}}`, h.Result(t, "txpool_contentFrom", common.Address{0x01}))
}

func TestContent_KeysSendersByTheirChecksummedAddress(t *testing.T) {
	h, pending, _ := newHarness(t)
	var content map[string]map[string]json.RawMessage
	h.Call(t, &content, "txpool_content")
	require.Contains(t, content["pending"], pending.Address().Hex())
}

func TestStatus_CountsPendingAndQueued(t *testing.T) {
	h, _, _ := newHarness(t)
	require.Equal(t, `{"pending":"0x2","queued":"0x1"}`, h.Result(t, "txpool_status"))
}

func TestInspect_SummarisesEachTransaction(t *testing.T) {
	h, pending, queued := newHarness(t)
	var inspected map[string]map[common.Address]map[string]string
	h.Call(t, &inspected, "txpool_inspect")
	require.Equal(t, map[string]string{
		"0": "0x4200000000000000000000000000000000000000: 1 wei + 21000 gas × 2 wei",
		"1": "contract creation: 0 wei + 50000 gas × 5 wei",
	}, inspected["pending"][pending.Address()])
	require.Equal(t, map[string]string{
		"7": "0x4200000000000000000000000000000000000000: 0 wei + 21000 gas × 3 wei",
	}, inspected["queued"][queued.Address()])
}
