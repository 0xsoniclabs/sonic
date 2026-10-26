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
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/format"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

var recipient = common.HexToAddress("0x42")

func TestSend_HandsTheTransactionToThePool(t *testing.T) {
	fc := chainOf(1)
	h := newEth(t, fc)
	w := tests.NewWallet(t)
	tx := w.Transfer(t, 0, recipient, big.NewInt(1))
	var hash common.Hash
	h.MustCall(&hash, "eth_sendRawTransaction", hexutil.Bytes(tests.Encode(t, tx)))
	require.Equal(t, tx.Hash(), hash)
	require.Len(t, fc.Sent(), 1)
	require.Equal(t, tx.Hash(), fc.Sent()[0].Hash())

	fc.SetSendError(errors.New("nonce too low"))
	require.Equal(t, "nonce too low", h.Fails("eth_sendRawTransaction", hexutil.Bytes(tests.Encode(t, tx))).Message, "the pool's verdict, verbatim")
	require.Equal(t, "typed transaction too short", h.Fails("eth_sendRawTransaction", hexutil.Bytes{0x02}).Message)
}

// TestSend_TheChecksOfTheNode are V7: replay protection and the fee cap.
func TestSend_TheChecksOfTheNode(t *testing.T) {
	fc := chainOf(1)
	h := newEth(t, fc)
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	unprotected, err := types.SignNewTx(key, types.HomesteadSigner{}, &types.LegacyTx{To: &recipient, Gas: 21000, GasPrice: big.NewInt(1e9)})
	require.NoError(t, err)
	require.Equal(t, "only replay-protected (EIP-155) transactions allowed over RPC",
		h.Fails("eth_sendRawTransaction", hexutil.Bytes(tests.Encode(t, unprotected))).Message)

	w := tests.NewWallet(t)
	expensive := w.Sign(t, &types.LegacyTx{To: &recipient, Gas: 1_000_000, GasPrice: big.NewInt(1e15)})
	require.Equal(t, "tx fee (1000.00 FTM) exceeds the configured cap (100.00 FTM)",
		h.Fails("eth_sendRawTransaction", hexutil.Bytes(tests.Encode(t, expensive))).Message)
	require.Empty(t, fc.Sent())

	allowing := newEth(t, fc, func(c *tests.Configs) { c.Exec.AllowUnprotectedTxs = true; c.Exec.TxFeeCap = 0 })
	allowing.MustCall(nil, "eth_sendRawTransaction", hexutil.Bytes(tests.Encode(t, unprotected)))
	allowing.MustCall(nil, "eth_sendRawTransaction", hexutil.Bytes(tests.Encode(t, expensive)))
	require.Len(t, fc.Sent(), 2)
}

func TestSendSync_AnswersTheReceiptOnceTheBlockArrives(t *testing.T) {
	fc := chainOf(1)
	h := newEth(t, fc)
	w := tests.NewWallet(t)
	tx := w.Transfer(t, 0, recipient, big.NewInt(1))
	done := make(chan map[string]any, 1)
	go func() {
		var receipt map[string]any
		h.MustCall(&receipt, "eth_sendRawTransactionSync", hexutil.Bytes(tests.Encode(t, tx)), "0x2710")
		done <- receipt
	}()
	tests.WaitFor(t, func() bool { return len(fc.Sent()) == 1 })
	block := fc.AddAndPublish(types.Transactions{tx})
	receipt := <-done
	require.Equal(t, tx.Hash().Hex(), receipt["transactionHash"])
	require.Equal(t, block.Hash.Hex(), receipt["blockHash"])
}

func TestSendSync_TheCodesOfEIP7966(t *testing.T) {
	fc := chainOf(1)
	h := newEth(t, fc, func(c *tests.Configs) { c.Exec.TxSyncMaxTimeout = 0 })
	w := tests.NewWallet(t)
	tx := w.Transfer(t, 0, recipient, big.NewInt(1))
	raw := hexutil.Bytes(tests.Encode(t, tx))

	e := h.Fails("eth_sendRawTransactionSync", raw, "0x1")
	require.Equal(t, 4, e.Code)
	require.Equal(t, "transaction not confirmed within timeout", e.Message)
	require.Equal(t, tx.Hash().Hex(), e.Data.(common.Hash).Hex())

	fc.SetSendError(errors.New("underpriced"))
	e = h.Fails("eth_sendRawTransactionSync", raw, "0x1")
	require.Equal(t, 5, e.Code)
	require.Equal(t, "transaction was rejected: underpriced", e.Message)

	gap := w.Transfer(t, 3, recipient, big.NewInt(1))
	e = h.Fails("eth_sendRawTransactionSync", hexutil.Bytes(tests.Encode(t, gap)), "0x1")
	require.Equal(t, 6, e.Code)
	require.Equal(t, "nonce gap: expected nonce 0", e.Message)

	require.Equal(t, "timeout must be greater than zero", h.Fails("eth_sendRawTransactionSync", raw, "0x0").Message)
}

func TestSendSync_ATransactionAlreadyMinedIsAnsweredAtOnce(t *testing.T) {
	fc := tests.NewFakeCore()
	w := tests.NewWallet(t)
	tx := w.Transfer(t, 0, recipient, big.NewInt(1))
	fc.AddAndPublish(types.Transactions{tx})
	fc.SetNonce(w.Address, 1)
	h := newEth(t, fc)
	var receipt map[string]any
	h.MustCall(&receipt, "eth_sendRawTransactionSync", hexutil.Bytes(tests.Encode(t, tx)))
	require.Equal(t, "0x1", receipt["blockNumber"])
}

func TestPendingTransactions_AreThoseOfThePool(t *testing.T) {
	fc := chainOf(1)
	w := tests.NewWallet(t)
	tx := w.Transfer(t, 0, recipient, big.NewInt(1))
	fc.SetPool(map[common.Address]types.Transactions{w.Address: {tx}}, nil)
	h := newEth(t, fc)
	var pending []format.Transaction
	h.MustCall(&pending, "eth_pendingTransactions")
	require.Len(t, pending, 1)
	require.Equal(t, tx.Hash(), pending[0].Hash)
	require.Nil(t, pending[0].BlockHash)
}
