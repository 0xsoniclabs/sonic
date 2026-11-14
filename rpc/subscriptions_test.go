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

package rpc_test

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/rpc"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

// startSubscribable starts a module over the fake with an IPC endpoint, and
// answers a client of it.
func startSubscribable(t *testing.T, fake *tests.FakeCore) *gethrpc.Client {
	t.Helper()
	config := moduleConfig(t, nil)
	module := rpc.New(config, fake, gethrpc.NewServer(), prometheus.NewRegistry())
	require.NoError(t, module.Start())
	t.Cleanup(module.Stop)
	require.Eventually(t, fake.FeedsSubscribed, time.Second, time.Millisecond)
	client, err := gethrpc.Dial(config.IPCPath)
	require.NoError(t, err)
	t.Cleanup(client.Close)
	return client
}

func receive(t *testing.T, notifications <-chan json.RawMessage, subscription *gethrpc.ClientSubscription) json.RawMessage {
	t.Helper()
	select {
	case notification := <-notifications:
		return notification
	case err := <-subscription.Err():
		t.Fatalf("the subscription ended: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("no notification")
	}
	return nil
}

func TestSubscriptions_NotifyHeadsLogsAndTransactions(t *testing.T) {
	fake := tests.NewFakeCore()
	client := startSubscribable(t, fake)
	ctx := context.Background()
	emitter := common.Address{0xe0}

	heads, logs, hashes, full := make(chan json.RawMessage, 4), make(chan json.RawMessage, 4), make(chan json.RawMessage, 4), make(chan json.RawMessage, 4)
	headSubscription, err := client.EthSubscribe(ctx, heads, "newHeads")
	require.NoError(t, err)
	logSubscription, err := client.EthSubscribe(ctx, logs, "logs", map[string]any{"address": emitter})
	require.NoError(t, err)
	hashSubscription, err := client.EthSubscribe(ctx, hashes, "newPendingTransactions")
	require.NoError(t, err)
	fullSubscription, err := client.EthSubscribe(ctx, full, "newPendingTransactions", true)
	require.NoError(t, err)

	recipient := common.Address{0x42}
	transaction := tests.NewAccount().Sign(&types.LegacyTx{To: &recipient, Gas: 21000, GasPrice: big.NewInt(1e9)})
	fake.AddPoolTransaction(transaction)
	require.JSONEq(t, `"`+transaction.Hash().Hex()+`"`, string(receive(t, hashes, hashSubscription)))
	var whole struct {
		Hash common.Hash `json:"hash"`
	}
	require.NoError(t, json.Unmarshal(receive(t, full, fullSubscription), &whole))
	require.Equal(t, transaction.Hash(), whole.Hash)

	number := fake.AddBlockWithReceipts(types.Transactions{transaction}, types.Receipts{{
		Status: types.ReceiptStatusSuccessful, CumulativeGasUsed: 21000,
		Logs: []*types.Log{{Address: common.Address{0x01}, Topics: []common.Hash{}}, {Address: emitter, Topics: []common.Hash{}}},
	}})
	fake.SetHead(number)
	var head struct {
		Number string `json:"number"`
	}
	require.NoError(t, json.Unmarshal(receive(t, heads, headSubscription), &head))
	require.Equal(t, "0x1", head.Number)
	var log types.Log
	require.NoError(t, json.Unmarshal(receive(t, logs, logSubscription), &log))
	require.Equal(t, emitter, log.Address)

	headSubscription.Unsubscribe()
}

func TestSubscriptions_EndByUnsubscribing(t *testing.T) {
	client := startSubscribable(t, tests.NewFakeCore())
	var id string
	require.NoError(t, client.Call(&id, "eth_subscribe", "newHeads"))
	var ended bool
	require.NoError(t, client.Call(&ended, "eth_unsubscribe", id))
	require.True(t, ended)
	require.EqualError(t, client.Call(&ended, "eth_unsubscribe", id), "subscription not found")
	require.EqualError(t, client.Call(&ended, "other_unsubscribe", 1), "invalid argument 0: json: cannot unmarshal number into Go value of type rpc.ID")
}

func TestSubscriptions_RefuseWhatTheyCannotServe(t *testing.T) {
	client := startSubscribable(t, tests.NewFakeCore())
	ctx := context.Background()
	_, err := client.EthSubscribe(ctx, make(chan any), "syncing")
	require.EqualError(t, err, `no "syncing" subscription in eth namespace`)
	_, err = client.EthSubscribe(ctx, make(chan any), "logs", map[string]any{"fromBlock": "0x2", "toBlock": "0x1"})
	require.EqualError(t, err, "invalid from and to block combination: from > to")
	_, err = client.EthSubscribe(ctx, make(chan any), "newHeads", 1)
	require.EqualError(t, err, "too many arguments, want at most 1")
}

func TestSubscriptions_AreNotServedOverHTTP(t *testing.T) {
	config := moduleConfig(t, nil)
	module := rpc.New(config, tests.NewFakeCore(), gethrpc.NewServer(), prometheus.NewRegistry())
	require.NoError(t, module.Start())
	defer module.Stop()
	endpoint, err := module.HTTPEndpoint()
	require.NoError(t, err)
	client, err := gethrpc.Dial(endpoint)
	require.NoError(t, err)
	defer client.Close()
	var id string
	require.EqualError(t, client.Call(&id, "eth_subscribe", "newHeads"), "notifications not supported")
	require.EqualError(t, client.Call(&id, "eth_unsubscribe", "0x1"), "subscription not found")
}

// A head is formed once for all newHeads subscriptions: one read of its
// receipts.
func TestSubscriptions_FormEachHeadOnce(t *testing.T) {
	fake := tests.NewFakeCore()
	client := startSubscribable(t, fake)
	notifications := make([]chan json.RawMessage, 5)
	subscriptions := make([]*gethrpc.ClientSubscription, 5)
	for i := range notifications {
		notifications[i] = make(chan json.RawMessage, 1)
		var err error
		subscriptions[i], err = client.EthSubscribe(context.Background(), notifications[i], "newHeads")
		require.NoError(t, err)
	}
	receipts := fake.Calls("Receipts")
	fake.AddBlock()
	fake.SetHead(1)
	for i := range notifications {
		receive(t, notifications[i], subscriptions[i])
	}
	require.Equal(t, receipts+1, fake.Calls("Receipts"))
}

// A subscription that loses events ends its connection, so that its client
// learns it.
func TestSubscriptions_EndTheirConnectionWhenTheyLoseEvents(t *testing.T) {
	fake := tests.NewFakeCore()
	client := startSubscribable(t, fake)
	subscription, err := client.EthSubscribe(context.Background(), make(chan json.RawMessage), "newPendingTransactions")
	require.NoError(t, err)
	fake.StopFeeds()
	select {
	case err := <-subscription.Err():
		require.Error(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("the subscription goes on")
	}
}
