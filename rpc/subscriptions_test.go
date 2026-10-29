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

package rpc

import (
	"context"
	"encoding/json"
	"math/big"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/0xsoniclabs/sonic/rpc/transport"
)

// streamModule is a started module whose HTTP and WebSocket endpoints share
// a port, with a unix socket, over a fake core.
type streamModule struct {
	core    *tests.FakeCore
	module  *Module
	clients map[string]*gethrpc.Client
}

func newStreamModule(t *testing.T) streamModule {
	t.Helper()
	fake := tests.NewFakeCore(t)
	config := DefaultConfig()
	config.HTTP = &HTTPEndpoint{Address: "127.0.0.1:0", HTTPConfig: transport.HTTPConfig{VirtualHosts: []string{"localhost"}}}
	config.WebSocket = &WebSocketEndpoint{Address: "127.0.0.1:0"}
	config.IPC = &IPCEndpoint{Path: filepath.Join(t.TempDir(), "sonic.ipc")}
	module := New(config, fake, prometheus.NewRegistry())
	require.NoError(t, module.Start())
	t.Cleanup(func() { require.NoError(t, module.Stop()) })
	tests.Eventually(t, func() bool { return fake.HeadSubscribers() == 1 && fake.PoolSubscribers() == 1 }, "the feeds to be taken")

	websocketClient, err := gethrpc.DialWebsocket(context.Background(), module.WebSocketEndpoint(), "")
	require.NoError(t, err)
	t.Cleanup(websocketClient.Close)
	ipcClient, err := gethrpc.DialIPC(context.Background(), config.IPC.Path)
	require.NoError(t, err)
	t.Cleanup(ipcClient.Close)
	return streamModule{core: fake, module: module, clients: map[string]*gethrpc.Client{transport.WebSocket: websocketClient, transport.IPC: ipcClient}}
}

func receive[Value any](t *testing.T, events chan Value) Value {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(10 * time.Second):
		t.Fatal("no notification")
	}
	var zero Value
	return zero
}

// TestSubscriptions_NewHeadsIsBuiltOncePerBlock holds I16 and D4, D5: every
// subscriber, two on every stream transport, gets the header, built with one
// read of the receipts.
func TestSubscriptions_NewHeadsIsBuiltOncePerBlock(t *testing.T) {
	stream := newStreamModule(t)
	channels := []chan map[string]any{}
	for name, client := range stream.clients {
		for range 2 {
			headers := make(chan map[string]any, 4)
			_, err := client.Subscribe(context.Background(), "eth", headers, "newHeads")
			require.NoError(t, err, name)
			channels = append(channels, headers)
		}
	}
	receipts := stream.core.Calls("Receipts")
	block := stream.core.AppendAndPublish(tests.BlockSpec{})
	for _, headers := range channels {
		header := receive(t, headers)
		require.Equal(t, block.Hash.Hex(), header["hash"])
		require.Equal(t, "0x1", header["number"])
	}
	require.Equal(t, receipts+1, stream.core.Calls("Receipts"), "one read for four subscribers")
}

func TestSubscriptions_LogsMatchTheirCriteria(t *testing.T) {
	stream := newStreamModule(t)
	watched := common.HexToAddress("0xa1")
	logs := make(chan types.Log, 4)
	_, err := stream.clients[transport.WebSocket].EthSubscribe(context.Background(), logs, "logs", map[string]any{"address": watched})
	require.NoError(t, err)
	signer := tests.NewSigner(t)
	sent := signer.Sign(t, &types.LegacyTx{To: &watched, Gas: 21000, GasPrice: big.NewInt(1)})
	stream.core.AppendAndPublish(tests.BlockSpec{Transactions: types.Transactions{sent}, Receipts: types.Receipts{{Status: 1, Logs: []*types.Log{
		{Address: common.HexToAddress("0xa2")}, {Address: watched, Topics: []common.Hash{{1}}},
	}}}})
	log := receive(t, logs)
	require.Equal(t, watched, log.Address)
	require.Equal(t, sent.Hash(), log.TxHash)
	require.Equal(t, uint64(1), log.BlockNumber)

	var failure gethrpc.Error
	_, err = stream.clients[transport.IPC].EthSubscribe(context.Background(), logs, "logs", map[string]any{"fromBlock": "0x5", "toBlock": "0x1"})
	require.ErrorAs(t, err, &failure)
	require.Equal(t, "invalid from and to block combination: from > to", failure.Error())
}

func TestSubscriptions_NewPendingTransactionsAreHashesOrWhole(t *testing.T) {
	stream := newStreamModule(t)
	hashes := make(chan common.Hash, 4)
	_, err := stream.clients[transport.IPC].EthSubscribe(context.Background(), hashes, "newPendingTransactions")
	require.NoError(t, err)
	whole := make(chan map[string]any, 4)
	_, err = stream.clients[transport.WebSocket].EthSubscribe(context.Background(), whole, "newPendingTransactions", true)
	require.NoError(t, err)

	signer := tests.NewSigner(t)
	recipient := common.HexToAddress("0xb0")
	pending := signer.Sign(t, &types.LegacyTx{To: &recipient, Gas: 21000, GasPrice: big.NewInt(9)})
	stream.core.SendPoolTransactions(pending)
	require.Equal(t, pending.Hash(), receive(t, hashes))
	transaction := receive(t, whole)
	require.Equal(t, pending.Hash().Hex(), transaction["hash"])
	require.Equal(t, "0x9", transaction["gasPrice"])
	require.Nil(t, transaction["blockHash"])
}

// TestSubscriptions_AreRefusedAsTheReference holds F4.
func TestSubscriptions_AreRefusedAsTheReference(t *testing.T) {
	stream := newStreamModule(t)
	var failure gethrpc.Error
	_, err := stream.clients[transport.WebSocket].EthSubscribe(context.Background(), make(chan any), "syncing")
	require.ErrorAs(t, err, &failure)
	require.Equal(t, -32601, failure.ErrorCode())
	require.Equal(t, `no "syncing" subscription in eth namespace`, failure.Error())

	_, err = stream.clients[transport.IPC].EthSubscribe(context.Background(), make(chan any), "logs")
	require.ErrorAs(t, err, &failure)
	require.Equal(t, "missing value for required argument 0", failure.Error())

	http, err := gethrpc.Dial(stream.module.HTTPEndpoint())
	require.NoError(t, err)
	defer http.Close()
	var answer json.RawMessage
	err = http.Call(&answer, "eth_subscribe", "newHeads")
	require.ErrorAs(t, err, &failure)
	require.Equal(t, "notifications not supported", failure.Error())
}

// TestSubscriptions_EndWithTheFeedTheyLost holds I18: a restarted head loop
// drops its consumers, and their subscriptions end with their connections.
func TestSubscriptions_EndWithTheFeedTheyLost(t *testing.T) {
	stream := newStreamModule(t)
	subscriptions := map[string]*gethrpc.ClientSubscription{}
	for name, client := range stream.clients {
		subscription, err := client.EthSubscribe(context.Background(), make(chan types.Log), "logs", map[string]any{})
		require.NoError(t, err, name)
		subscriptions[name] = subscription
	}
	stream.core.EndFeeds()
	for name, subscription := range subscriptions {
		select {
		case <-subscription.Err():
		case <-time.After(10 * time.Second):
			t.Fatalf("%s: the subscription did not end", name)
		}
	}
}

func TestSubscriptions_EndOnUnsubscribe(t *testing.T) {
	stream := newStreamModule(t)
	client := stream.clients[transport.WebSocket]
	subscription, err := client.EthSubscribe(context.Background(), make(chan map[string]any), "newHeads")
	require.NoError(t, err)
	subscription.Unsubscribe()
	var removed bool
	require.ErrorContains(t, client.Call(&removed, "eth_unsubscribe", "0x1"), "subscription not found")
}
