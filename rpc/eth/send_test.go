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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// sendHarness serves a chain of one block whose sender's pool nonce is 0.
func sendHarness(t *testing.T, mutate ...func(*tests.Settings)) (*tests.Harness, *tests.Account) {
	t.Helper()
	fake := tests.NewFakeCore()
	h := tests.NewHarness(t, fake, func(h *tests.Harness) {
		eth.Register(h.Dispatcher, h.Environment, h.Core)
	}, mutate...)
	return h, tests.NewAccount()
}

func transfer(account *tests.Account, nonce uint64) *types.Transaction {
	recipient := common.Address{0x42}
	return account.Sign(&types.LegacyTx{Nonce: nonce, To: &recipient, Gas: 21000, GasPrice: big.NewInt(1e9)})
}

func encode(t *testing.T, transaction *types.Transaction) string {
	t.Helper()
	binary, err := transaction.MarshalBinary()
	require.NoError(t, err)
	return hexutil.Encode(binary)
}

func TestSendRawTransaction_HandsTheTransactionToThePool(t *testing.T) {
	h, account := sendHarness(t)
	transaction := transfer(account, 0)
	var hash common.Hash
	h.Call(t, &hash, "eth_sendRawTransaction", encode(t, transaction))
	require.Equal(t, transaction.Hash(), hash)
	pooled, err := h.Core.PoolTransaction(hash)
	require.NoError(t, err)
	require.Equal(t, transaction.Hash(), pooled.Hash())

	creation := account.Sign(&types.LegacyTx{Nonce: 1, Gas: 60000, GasPrice: big.NewInt(1e9)})
	h.Call(t, &hash, "eth_sendRawTransaction", encode(t, creation))
	require.Equal(t, creation.Hash(), hash)
}

func TestSendRawTransaction_RefusesWhatTheSendPathRefuses(t *testing.T) {
	h, account := sendHarness(t)
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	unprotected := types.MustSignNewTx(key, types.HomesteadSigner{}, &types.LegacyTx{Gas: 21000, GasPrice: big.NewInt(1e9)})
	expensive := account.Sign(&types.LegacyTx{Gas: 21000, GasPrice: big.NewInt(1e16)})
	cases := map[string]struct {
		encoded string
		err     string
	}{
		"undecodable":       {"0x01", "typed transaction too short"},
		"unprotected":       {encode(t, unprotected), "only replay-protected (EIP-155) transactions allowed over RPC"},
		"above the fee cap": {encode(t, expensive), "tx fee (210.00 FTM) exceeds the configured cap (100.00 FTM)"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, &dispatch.Error{Code: dispatch.CodeDefault, Message: test.err}, h.Fail(t, "eth_sendRawTransaction", test.encoded))
		})
	}
	require.Zero(t, h.Core.Calls("SendTransaction"))

	h.Core.SetMode("SendTransaction", tests.Fail)
	require.Equal(t, tests.ErrInjected.Error(), h.Fail(t, "eth_sendRawTransaction", encode(t, transfer(account, 0))).Message,
		"the pool's verdict is answered as it is")
}

func TestSendRawTransaction_TakesTheSettingsOfTheSendPath(t *testing.T) {
	h, account := sendHarness(t, func(settings *tests.Settings) {
		settings.Exec.RPCTxFeeCap = 0
		settings.Exec.AllowUnprotectedTxs = true
	})
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	unprotected := types.MustSignNewTx(key, types.HomesteadSigner{}, &types.LegacyTx{Gas: 21000, GasPrice: big.NewInt(1e9)})
	expensive := account.Sign(&types.LegacyTx{Gas: 21000, GasPrice: big.NewInt(1e16)})
	for _, transaction := range []*types.Transaction{unprotected, expensive} {
		var hash common.Hash
		h.Call(t, &hash, "eth_sendRawTransaction", encode(t, transaction))
		require.Equal(t, transaction.Hash(), hash)
	}
}

// The receipt is answered once a served block holds the transaction, also
// when one did before it was sent.
func TestSendRawTransactionSync_AnswersTheReceiptOnceTheBlockIsServed(t *testing.T) {
	h, account := sendHarness(t)
	transaction := transfer(account, 0)
	encoded := encode(t, transaction)
	answered := make(chan dispatch.Response)
	go func() { answered <- h.Serve(t, "eth_sendRawTransactionSync", encoded) }()
	require.Eventually(t, func() bool { return h.Core.Calls("SendTransaction") == 1 }, time.Second, time.Millisecond)
	h.Core.AddBlock(transaction)
	h.Core.SetHead(1)
	response := <-answered
	require.Nil(t, response.Error)
	var receipt receiptForm
	require.NoError(t, json.Unmarshal(response.Result, &receipt))
	block, err := h.Core.Block(1)
	require.NoError(t, err)
	require.Equal(t, transaction.Hash(), receipt.TransactionHash)
	require.Equal(t, block.Hash, receipt.BlockHash)
	require.Equal(t, account.Address(), receipt.From)

	var again receiptForm
	h.Call(t, &again, "eth_sendRawTransactionSync", encode(t, transaction))
	require.Equal(t, receipt, again)
}

func TestSendRawTransactionSync_RefusesWhatItCannotWaitFor(t *testing.T) {
	h, account := sendHarness(t, func(settings *tests.Settings) {
		settings.Exec.TxSyncMaxTimeout = 20 * time.Millisecond
	})
	h.Core.SetPoolNonce(account.Address(), 2)
	inTime := transfer(account, 2)
	cases := map[string]struct {
		params []any
		err    *dispatch.Error
	}{
		"undecodable": {[]any{"0x01"}, &dispatch.Error{Code: dispatch.CodeDefault, Message: "typed transaction too short"}},
		"no wait":     {[]any{encode(t, inTime), "0x0"}, &dispatch.Error{Code: dispatch.CodeDefault, Message: "timeout must be greater than zero"}},
		"nonce gap": {[]any{encode(t, transfer(account, 3))},
			&dispatch.Error{Code: 6, Message: "nonce gap: expected nonce 2", Data: hexutil.Uint64(2)}},
		"rejected": {[]any{encode(t, account.Sign(&types.LegacyTx{Nonce: 1, Gas: 21000, GasPrice: big.NewInt(1e16)}))},
			&dispatch.Error{Code: 5, Message: "transaction was rejected: tx fee (210.00 FTM) exceeds the configured cap (100.00 FTM)",
				Data: account.Sign(&types.LegacyTx{Nonce: 1, Gas: 21000, GasPrice: big.NewInt(1e16)}).Hash()}},
		"not confirmed": {[]any{encode(t, inTime), hexutil.Uint64(1 << 62)},
			&dispatch.Error{Code: 4, Message: "transaction not confirmed within timeout", Data: inTime.Hash()}},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.err, h.Fail(t, "eth_sendRawTransactionSync", test.params...))
		})
	}
}

func TestSendRawTransactionSync_WordsAMissingBlockOrReceipt(t *testing.T) {
	h, account := sendHarness(t)
	transaction := transfer(account, 0)
	h.Core.AddBlock(transaction)
	h.SetHead(t, 1)
	cases := map[string]struct {
		method string
		err    string
	}{
		"block":   {"Header", "confirmed transaction " + transaction.Hash().Hex() + " found in block 1 but the block is unavailable"},
		"receipt": {"Receipts", "confirmed transaction " + transaction.Hash().Hex() + " found in block 1 but receipt is unavailable or transaction index 0 is out of range"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			h.Core.SetMode(test.method, tests.NotFound)
			defer h.Core.SetMode(test.method, tests.Normal)
			require.Equal(t, test.err, h.Fail(t, "eth_sendRawTransactionSync", encode(t, transaction)).Message)
		})
	}
}

func TestPendingTransactions_AreNoneOfTheNodesAccounts(t *testing.T) {
	h, account := sendHarness(t)
	h.Core.AddPoolTransaction(transfer(account, 0))
	require.Equal(t, "[]", h.Result(t, "eth_pendingTransactions"))
}
