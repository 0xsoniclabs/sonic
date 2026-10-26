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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func transfer(t *testing.T, signer *tests.Signer, nonce uint64, gasPrice *big.Int) *types.Transaction {
	t.Helper()
	recipient := common.HexToAddress("0xb0")
	return signer.Sign(t, &types.LegacyTx{Nonce: nonce, To: &recipient, Gas: 21000, GasPrice: gasPrice})
}

// TestSendRawTransaction_HandsThePoolItsVerdict holds V7.
func TestSendRawTransaction_HandsThePoolItsVerdict(t *testing.T) {
	fake := tests.NewFakeCore(t)
	harness := newEth(t, fake)
	signer := tests.NewSigner(t)
	sent := transfer(t, signer, 0, big.NewInt(1_000_000_000))
	require.JSONEq(t, `"`+sent.Hash().Hex()+`"`, string(harness.Result("eth_sendRawTransaction", tests.Encode(t, sent))))
	require.Len(t, fake.Sent(), 1)
	require.Equal(t, sent.Hash(), fake.Sent()[0].Hash())

	fake.SetSendError(errors.New("nonce too low"))
	require.Equal(t, "nonce too low", harness.Failure("eth_sendRawTransaction", tests.Encode(t, transfer(t, signer, 1, big.NewInt(1)))).Message, "verbatim")
}

func TestSendRawTransaction_RefusesBeforeThePool(t *testing.T) {
	fake := tests.NewFakeCore(t)
	harness := newEth(t, fake)
	signer := tests.NewSigner(t)

	expensive := transfer(t, signer, 0, new(big.Int).Mul(big.NewInt(10_000), big.NewInt(1_000_000_000_000)))
	require.Equal(t, "tx fee (210.00 FTM) exceeds the configured cap (100.00 FTM)", harness.Failure("eth_sendRawTransaction", tests.Encode(t, expensive)).Message)

	recipient := common.HexToAddress("0xb0")
	unprotected := signer.SignUnprotected(t, &types.LegacyTx{To: &recipient, Gas: 21000, GasPrice: big.NewInt(1)})
	require.Equal(t, "only replay-protected (EIP-155) transactions allowed over RPC", harness.Failure("eth_sendRawTransaction", tests.Encode(t, unprotected)).Message)

	require.Equal(t, "typed transaction too short", harness.Failure("eth_sendRawTransaction", hexutil.Bytes{0x01}).Message)
	require.Empty(t, fake.Sent())

	permissive := newEth(t, fake, func(configs *tests.Configs) {
		configs.Exec.AllowUnprotectedTxs = true
		configs.Exec.RPCTxFeeCap = 0
	})
	permissive.Result("eth_sendRawTransaction", tests.Encode(t, unprotected))
	permissive.Result("eth_sendRawTransaction", tests.Encode(t, expensive))
	require.Len(t, fake.Sent(), 2)
}

// TestSendRawTransactionSync_WaitsForTheBlockOnTheHeadFeed holds I9: the
// receipt is answered when a block that holds the transaction arrives.
func TestSendRawTransactionSync_WaitsForTheBlockOnTheHeadFeed(t *testing.T) {
	fake := tests.NewFakeCore(t)
	harness := newEth(t, fake)
	signer := tests.NewSigner(t)
	sent := transfer(t, signer, 0, big.NewInt(1_000_000_000))

	answer := make(chan *dispatch.Error, 1)
	result := make(chan []byte, 1)
	go func() {
		raw, failure := harness.Call("eth_sendRawTransactionSync", tests.Encode(t, sent), "0x7530")
		result <- raw
		answer <- failure
	}()
	tests.Eventually(t, func() bool { return len(fake.Sent()) == 1 }, "the transaction to be sent")
	fake.AppendAndPublish(tests.BlockSpec{})
	fake.AppendAndPublish(tests.BlockSpec{Transactions: types.Transactions{sent}})
	require.Nil(t, <-answer)
	receipt := decodeMap(t, <-result)
	require.Equal(t, sent.Hash().Hex(), receipt["transactionHash"])
	require.Equal(t, "0x2", receipt["blockNumber"])
}

// TestSendRawTransactionSync_AnswersWithTheCodesOfEIP7966 holds V7.
func TestSendRawTransactionSync_AnswersWithTheCodesOfEIP7966(t *testing.T) {
	fake := tests.NewFakeCore(t)
	harness := newEth(t, fake, func(configs *tests.Configs) {
		configs.Exec.TxSyncDefaultTimeout = 50 * time.Millisecond
		configs.Exec.TxSyncMaxTimeout = 100 * time.Millisecond
	})
	signer := tests.NewSigner(t)
	waiting := transfer(t, signer, 0, big.NewInt(1_000_000_000))
	failure := harness.Failure("eth_sendRawTransactionSync", tests.Encode(t, waiting))
	require.Equal(t, &dispatch.Error{Code: 4, Message: "transaction not confirmed within timeout", Data: waiting.Hash()}, failure)

	start := time.Now()
	bounded := harness.Failure("eth_sendRawTransactionSync", tests.Encode(t, transfer(t, signer, 0, big.NewInt(2_000_000_000))), "0xffffffffffffffff")
	require.Equal(t, 4, bounded.Code)
	require.Less(t, time.Since(start), 3*time.Second, "the timeout is bounded by TxSyncMaxTimeout")

	gap := transfer(t, signer, 5, big.NewInt(1_000_000_000))
	require.Equal(t, &dispatch.Error{Code: 6, Message: "nonce gap: expected nonce 0", Data: hexutil.Uint64(0)}, harness.Failure("eth_sendRawTransactionSync", tests.Encode(t, gap)))

	fake.SetSendError(errors.New("underpriced"))
	rejected := transfer(t, signer, 0, big.NewInt(3_000_000_000))
	require.Equal(t, &dispatch.Error{Code: 5, Message: "transaction was rejected: underpriced", Data: rejected.Hash()}, harness.Failure("eth_sendRawTransactionSync", tests.Encode(t, rejected)))

	require.Equal(t, "timeout must be greater than zero", harness.Failure("eth_sendRawTransactionSync", tests.Encode(t, rejected), "0x0").Message)
}

// TestSendRawTransactionSync_TakesTheFeedAgainWhenItIsDropped: a head loop
// that restarts drops its consumers; the wait takes the feed again and
// searches the head it may have missed.
func TestSendRawTransactionSync_TakesTheFeedAgainWhenItIsDropped(t *testing.T) {
	fake := tests.NewFakeCore(t)
	harness := newEth(t, fake)
	signer := tests.NewSigner(t)
	sent := transfer(t, signer, 0, big.NewInt(1_000_000_000))

	answer := make(chan *dispatch.Error, 1)
	go func() {
		_, failure := harness.Call("eth_sendRawTransactionSync", tests.Encode(t, sent), "0x7530")
		answer <- failure
	}()
	tests.Eventually(t, func() bool { return len(fake.Sent()) == 1 }, "the transaction to be sent")
	subscribed := fake.Calls("SubscribeHead")
	fake.AppendBlock(tests.BlockSpec{Transactions: types.Transactions{sent}})
	fake.EndFeeds()
	tests.Eventually(t, func() bool { return fake.Calls("SubscribeHead") > subscribed }, "the head loop to subscribe again")
	fake.PublishHead(1)
	require.Nil(t, <-answer)
}
