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
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func rawTx(t *testing.T, tx *types.Transaction) hexutil.Bytes {
	t.Helper()
	raw, err := tx.MarshalBinary()
	require.NoError(t, err)
	return raw
}

func TestSendRawTransaction(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	h := newHarness(t, fc)
	alice := tests.NewAccount(t, nil, nil)
	tx := alice.Transfer(t, 0, common.Address{1}, 1)

	var hash common.Hash
	h.Into(&hash, "eth_sendRawTransaction", rawTx(t, tx))
	require.Equal(t, tx.Hash(), hash)
	require.Equal(t, tx.Hash(), fc.PoolTransaction(tx.Hash()).Hash())

	// Not an RLP transaction.
	err := h.Fails("eth_sendRawTransaction", "0x1234")
	require.Equal(t, dispatch.CodeDefault, err.Code)
}

func TestSendRawTransaction_PoolVerdictIsReturnedVerbatim(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	fc.SetSendTx(func(*types.Transaction) error { return errors.New("nonce too low") })
	h := newHarness(t, fc)
	alice := tests.NewAccount(t, nil, nil)
	err := h.Fails("eth_sendRawTransaction", rawTx(t, alice.Transfer(t, 0, common.Address{1}, 1)))
	require.Equal(t, &dispatch.Error{Code: dispatch.CodeDefault, Message: "nonce too low"}, err)
}

func TestSendRawTransaction_RefusesUnprotectedAndExpensiveTransactions(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	h := newHarness(t, fc)
	alice := tests.NewAccount(t, nil, nil)

	unprotected, err := types.SignNewTx(alice.Key, types.HomesteadSigner{}, &types.LegacyTx{
		Nonce: 0, GasPrice: big.NewInt(1000), Gas: 21000, To: &common.Address{1},
	})
	require.NoError(t, err)
	e := h.Fails("eth_sendRawTransaction", rawTx(t, unprotected))
	require.Equal(t, "only replay-protected (EIP-155) transactions allowed over RPC", e.Message)

	// A fee above the cap: 1000 gas * 1e18 wei per gas = 1e21 wei = 1000 FTM.
	expensive, err := types.SignNewTx(alice.Key, tests.Signer(), &types.LegacyTx{
		Nonce: 0, GasPrice: new(big.Int).SetUint64(1e18), Gas: 1000, To: &common.Address{1},
	})
	require.NoError(t, err)
	e = h.Fails("eth_sendRawTransaction", rawTx(t, expensive))
	require.Contains(t, e.Message, "exceeds the configured cap")
}

func TestSendRawTransactionSync_ReturnsTheReceiptOnceTheBlockIsServed(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	h := newHarness(t, fc)
	alice := tests.NewAccount(t, nil, nil)
	tx := alice.Transfer(t, 0, common.Address{1}, 1)

	go func() {
		time.Sleep(100 * time.Millisecond)
		fc.AddBlock(tx)
	}()
	var receipt map[string]any
	h.Into(&receipt, "eth_sendRawTransactionSync", rawTx(t, tx), "0x2710")
	require.Equal(t, tx.Hash().Hex(), receipt["transactionHash"])
	require.Equal(t, "0x1", receipt["status"])
	require.Equal(t, "0x2", receipt["blockNumber"])
}

func TestSendRawTransactionSync_Errors(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	h := newHarness(t, fc)
	alice := tests.NewAccount(t, nil, nil)

	// Code 4: added to the pool, not confirmed within the timeout.
	tx := alice.Transfer(t, 0, common.Address{1}, 1)
	err := h.Fails("eth_sendRawTransactionSync", rawTx(t, tx), "0x32")
	require.Equal(t, 4, err.Code)
	require.Equal(t, "transaction not confirmed within timeout", err.Message)
	require.Equal(t, tx.Hash(), err.Data)
	require.NotNil(t, fc.PoolTransaction(tx.Hash()))

	// Code 6: nonce gap.
	fc.SetPoolNonce(alice.Addr, 1)
	err = h.Fails("eth_sendRawTransactionSync", rawTx(t, alice.Transfer(t, 5, common.Address{1}, 1)))
	require.Equal(t, 6, err.Code)
	require.Equal(t, "nonce gap: expected nonce 1", err.Message)
	require.Equal(t, hexutil.Uint64(1), err.Data)

	// Code 5: rejected by the pool.
	fc.SetSendTx(func(*types.Transaction) error { return errors.New("underpriced") })
	rejected := alice.Transfer(t, 1, common.Address{1}, 1)
	err = h.Fails("eth_sendRawTransactionSync", rawTx(t, rejected))
	require.Equal(t, 5, err.Code)
	require.Contains(t, err.Message, "underpriced")
	require.Equal(t, rejected.Hash(), err.Data)

	// A timeout of zero is refused.
	err = h.Fails("eth_sendRawTransactionSync", rawTx(t, tx), "0x0")
	require.Equal(t, "timeout must be greater than zero", err.Message)
}

func TestSendRawTransactionSync_AlreadyConfirmedTransactionAnswersAtOnce(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	alice := tests.NewAccount(t, nil, nil)
	tx := alice.Transfer(t, 0, common.Address{1}, 1)
	fc.AddBlock(tx)
	fc.SetPoolNonce(alice.Addr, 0)
	h := newHarness(t, fc)
	var receipt map[string]any
	h.Into(&receipt, "eth_sendRawTransactionSync", rawTx(t, tx))
	require.Equal(t, tx.Hash().Hex(), receipt["transactionHash"])
}

func TestPendingTransactions(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	h := newHarness(t, fc)
	alice := tests.NewAccount(t, nil, nil)
	tx := alice.Transfer(t, 0, common.Address{1}, 1)
	fc.AddPoolTransaction(tx)

	var txs []map[string]any
	h.Into(&txs, "eth_pendingTransactions")
	require.Len(t, txs, 1)
	require.Equal(t, tx.Hash().Hex(), txs[0]["hash"])
	require.Equal(t, alice.Addr.Hex(), common.HexToAddress(txs[0]["from"].(string)).Hex())
	require.Nil(t, txs[0]["blockNumber"])
}
