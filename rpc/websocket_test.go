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
	"net/http"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/rpc"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/0xsoniclabs/sonic/rpc/transport"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestModule_ServesWebSocketOnTheHTTPEndpointOrItsOwn(t *testing.T) {
	ws := &rpc.WSConfig{WSConfig: transport.WSConfig{Endpoint: "127.0.0.1:0"}, Modules: []string{"eth"}}
	cases := map[string]struct {
		http   *rpc.HTTPConfig
		shared bool
	}{
		"shared": {&rpc.HTTPConfig{HTTPConfig: transport.HTTPConfig{Endpoint: "127.0.0.1:0"}, Modules: []string{"web3"}}, true},
		"own":    {nil, false},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			config := rpc.DefaultConfig()
			config.HTTP, config.WS = test.http, ws
			fake := tests.NewFakeCore()
			module := rpc.New(config, fake, gethrpc.NewServer(), prometheus.NewRegistry())
			require.NoError(t, module.Start())
			defer module.Stop()
			endpoint, err := module.WSEndpoint()
			require.NoError(t, err)
			if test.shared {
				httpEndpoint, err := module.HTTPEndpoint()
				require.NoError(t, err)
				require.Equal(t, "ws"+httpEndpoint[len("http"):], endpoint)
			}
			client, err := gethrpc.Dial(endpoint)
			require.NoError(t, err)
			defer client.Close()
			var modules map[string]string
			require.NoError(t, client.Call(&modules, "rpc_modules"))
			require.Equal(t, map[string]string{"eth": "1.0", "rpc": "1.0"}, modules, "the namespaces of the WebSocket's list")

			heads := make(chan json.RawMessage, 1)
			subscription, err := client.EthSubscribe(context.Background(), heads, "newHeads")
			require.NoError(t, err)
			defer subscription.Unsubscribe()
			fake.AddBlock()
			fake.SetHead(1)
			receive(t, heads, subscription)
		})
	}
}

func TestModule_WithoutWebSocketHasNoEndpoint(t *testing.T) {
	module := startModule(t, rpc.DefaultConfig())
	defer module.Stop()
	_, err := module.WSEndpoint()
	require.EqualError(t, err, "the RPC module serves no WebSocket endpoint")
}

func TestModule_RefusesAWebSocketFromAnOriginNotAllowed(t *testing.T) {
	config := rpc.DefaultConfig()
	config.WS = &rpc.WSConfig{WSConfig: transport.WSConfig{Endpoint: "127.0.0.1:0", Origins: []string{"https://allowed.example"}}}
	module := startModule(t, config)
	defer module.Stop()
	endpoint, err := module.WSEndpoint()
	require.NoError(t, err)
	for origin, status := range map[string]int{"https://allowed.example": http.StatusSwitchingProtocols, "https://other.example": http.StatusForbidden} {
		connection, response, err := websocket.DefaultDialer.Dial(endpoint, http.Header{"Origin": {origin}})
		require.Equal(t, status, response.StatusCode, origin)
		if err == nil {
			_ = connection.Close()
		}
	}
}

// A client that does not read its notifications is disconnected once its
// subscription falls behind, and told why, rather than holding the node's
// events.
func TestModule_DisconnectsAClientThatDoesNotRead(t *testing.T) {
	config := rpc.DefaultConfig()
	config.WS = &rpc.WSConfig{WSConfig: transport.WSConfig{Endpoint: "127.0.0.1:0"}}
	fake := tests.NewFakeCore()
	module := rpc.New(config, fake, gethrpc.NewServer(), prometheus.NewRegistry())
	require.NoError(t, module.Start())
	defer module.Stop()
	require.Eventually(t, fake.FeedsSubscribed, time.Second, time.Millisecond)
	endpoint, err := module.WSEndpoint()
	require.NoError(t, err)
	connection, _, err := websocket.DefaultDialer.Dial(endpoint, nil)
	require.NoError(t, err)
	defer func() { _ = connection.Close() }()
	require.NoError(t, connection.WriteJSON(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "eth_subscribe", "params": []any{"newPendingTransactions", true},
	}))

	// Enough notifications to fill the sockets' buffers and the
	// subscription's queue while the client reads nothing.
	account := tests.NewAccount()
	recipient := common.Address{0x42}
	for nonce := range uint64(5000) {
		fake.AddPoolTransaction(account.Sign(&types.LegacyTx{Nonce: nonce, To: &recipient, Gas: 21000, GasPrice: big.NewInt(1), Data: make([]byte, 1024)}))
	}
	for {
		require.NoError(t, connection.SetReadDeadline(time.Now().Add(30*time.Second)))
		if _, _, err = connection.ReadMessage(); err != nil {
			break
		}
	}
	closeErr, closed := err.(*websocket.CloseError)
	require.True(t, closed, "the connection was closed: %v", err)
	require.Contains(t, closeErr.Text, "the consumer fell behind the events")
}
