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

package transport

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	prometheusModel "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// registerTicker adds test_subscribe: a "ticks" subscription notifies the
// numbers up to its count, a "failing" one ends its connection.
func registerTicker(dispatcher *dispatch.Dispatcher) {
	dispatcher.Add("test_subscribe", func(ctx context.Context, arguments dispatch.Arguments) (any, error) {
		call, _ := dispatch.CallFrom(ctx)
		kind := dispatch.Argument[string](arguments, 0)
		subscription, err := call.Subscribe("test", kind)
		if err != nil {
			return nil, err
		}
		count := dispatch.ArgumentOr(arguments, 1, 0)
		subscription.Go(func(ctx context.Context) {
			if kind == "failing" {
				subscription.Fail(errors.New("lost events"))
				return
			}
			for tick := range count {
				if subscription.Notify(ctx, tick) != nil {
					return
				}
			}
			<-ctx.Done()
		})
		return subscription.ID, nil
	}, dispatch.Required[string]("kind"), dispatch.Optional[int]("count"))
}

// streamServers are a WebSocket and a unix socket endpoint over the test
// service.
type streamServers struct {
	dispatcher *dispatch.Dispatcher
	server     *Server
	ipc        *IPCServer
	websocket  string
}

func newStreamServers(t *testing.T, config WebSocketConfig) streamServers {
	t.Helper()
	registry := prometheus.NewRegistry()
	dispatcher := dispatch.New(dispatch.Config{RequestTimeout: time.Minute}, dispatch.NewMetrics(registry))
	registerModuleService(dispatcher, 0)
	registerTicker(dispatcher)
	front := NewFront(dispatcher, nil)
	server := NewServer("127.0.0.1:0", Timeouts{}, dispatcher)
	server.EnableWebSocket(front, config)
	require.NoError(t, server.Start())
	ipc := NewIPCServer(filepath.Join(t.TempDir(), "sonic.ipc"), front, BatchLimits{}, dispatcher)
	require.NoError(t, ipc.Start())
	t.Cleanup(func() {
		server.Stop()
		ipc.Stop()
		dispatcher.Stop()
	})
	return streamServers{dispatcher: dispatcher, server: server, ipc: ipc, websocket: "ws://" + server.Address()}
}

func (servers streamServers) clients(t *testing.T) map[string]*gethrpc.Client {
	t.Helper()
	websocketClient, err := gethrpc.DialWebsocket(context.Background(), servers.websocket, "")
	require.NoError(t, err)
	ipcClient, err := gethrpc.DialIPC(context.Background(), servers.ipc.Path())
	require.NoError(t, err)
	t.Cleanup(websocketClient.Close)
	t.Cleanup(ipcClient.Close)
	return map[string]*gethrpc.Client{WebSocket: websocketClient, IPC: ipcClient}
}

// TestStream_ServesGethsClient holds D4 and D5 for calls and batches.
func TestStream_ServesGethsClient(t *testing.T) {
	servers := newStreamServers(t, WebSocketConfig{})
	for name, client := range servers.clients(t) {
		var echoed []any
		require.NoError(t, client.Call(&echoed, "test_echo", 7, "x"), name)
		require.Equal(t, []any{float64(7), "x"}, echoed, name)

		var rpcError gethrpc.Error
		require.ErrorAs(t, client.Call(nil, "test_fail"), &rpcError, name)
		require.Equal(t, 3, rpcError.ErrorCode(), name)
		require.ErrorAs(t, client.Call(nil, "test_missing"), &rpcError, name)
		require.Equal(t, -32601, rpcError.ErrorCode(), name)

		batch := []gethrpc.BatchElem{{Method: "test_echo", Args: []any{1}, Result: new([]any)}, {Method: "test_plain", Result: new(any)}}
		require.NoError(t, client.BatchCall(batch), name)
		require.NoError(t, batch[0].Error, name)
		require.EqualError(t, batch[1].Error, "plain failure", name)
	}
}

// TestStream_DeliversTheNotificationsOfASubscription holds D4 and D5 for
// subscriptions.
func TestStream_DeliversTheNotificationsOfASubscription(t *testing.T) {
	servers := newStreamServers(t, WebSocketConfig{})
	for name, client := range servers.clients(t) {
		ticks := make(chan int, 10)
		subscription, err := client.Subscribe(context.Background(), "test", ticks, "ticks", 3)
		require.NoError(t, err, name)
		for expected := range 3 {
			select {
			case tick := <-ticks:
				require.Equal(t, expected, tick, name)
			case <-time.After(5 * time.Second):
				t.Fatalf("%s: no notification %d", name, expected)
			}
		}
		subscription.Unsubscribe()

		_, err = client.Subscribe(context.Background(), "test", make(chan int), "unknown")
		require.NoError(t, err, "the test service accepts any kind")
	}
}

// TestStream_AnswersTheSubscriptionBeforeItsNotifications: a notification
// never overtakes the response that carries its id.
func TestStream_AnswersTheSubscriptionBeforeItsNotifications(t *testing.T) {
	servers := newStreamServers(t, WebSocketConfig{})
	connection, _, err := websocket.DefaultDialer.Dial(servers.websocket, nil)
	require.NoError(t, err)
	defer func() { _ = connection.Close() }()
	for range 20 {
		require.NoError(t, connection.WriteMessage(websocket.TextMessage, []byte(`{"jsonrpc":"2.0","id":1,"method":"test_subscribe","params":["ticks",5]}`)))
		var first map[string]any
		require.NoError(t, connection.ReadJSON(&first))
		require.Contains(t, first, "result", "the response comes first")
		for range 5 {
			var notification map[string]any
			require.NoError(t, connection.ReadJSON(&notification))
			require.Equal(t, "test_subscription", notification["method"])
			require.Equal(t, first["result"], notification["params"].(map[string]any)["subscription"])
		}
	}
}

// TestStream_EndsAConnectionThatSendsNoJSON answers a syntax error first, as
// go-ethereum's stream reader does; a message cut short just ends it.
func TestStream_EndsAConnectionThatSendsNoJSON(t *testing.T) {
	servers := newStreamServers(t, WebSocketConfig{})
	connection, _, err := websocket.DefaultDialer.Dial(servers.websocket, nil)
	require.NoError(t, err)
	defer func() { _ = connection.Close() }()
	require.NoError(t, connection.WriteMessage(websocket.TextMessage, []byte(`{]`)))
	var answer map[string]any
	require.NoError(t, connection.ReadJSON(&answer))
	require.Equal(t, float64(-32700), answer["error"].(map[string]any)["code"])
	_, _, err = connection.ReadMessage()
	require.Error(t, err, "the connection is closed")

	unix, err := net.Dial("unix", servers.ipc.Path())
	require.NoError(t, err)
	defer func() { _ = unix.Close() }()
	_, err = unix.Write([]byte("{]\n"))
	require.NoError(t, err)
	decoder := json.NewDecoder(unix)
	require.NoError(t, decoder.Decode(&answer))
	require.Equal(t, float64(-32700), answer["error"].(map[string]any)["code"])
	require.Error(t, decoder.Decode(&answer), "the connection is closed")
}

// TestWebSocket_ChecksTheOriginAndTheMessageSize holds D11.
func TestWebSocket_ChecksTheOriginAndTheMessageSize(t *testing.T) {
	servers := newStreamServers(t, WebSocketConfig{Origins: []string{"http://good.example"}})
	_, response, err := websocket.DefaultDialer.Dial(servers.websocket, http.Header{"Origin": {"http://evil.example"}})
	require.Error(t, err)
	require.Equal(t, http.StatusForbidden, response.StatusCode)
	allowed, _, err := websocket.DefaultDialer.Dial(servers.websocket, http.Header{"Origin": {"http://good.example"}})
	require.NoError(t, err)
	defer func() { _ = allowed.Close() }()

	// The header of a masked text frame announcing one byte more than the
	// limit; the server refuses it before any payload arrives.
	header := []byte{0x81, 0x80 | 127}
	header = binary.BigEndian.AppendUint64(header, websocketMessageLimit+1)
	header = append(header, 1, 2, 3, 4)
	_, err = allowed.NetConn().Write(header)
	require.NoError(t, err)
	_, _, err = allowed.ReadMessage()
	var closed *websocket.CloseError
	require.ErrorAs(t, err, &closed, "a message above 32 MiB ends the connection")
	require.Equal(t, websocket.CloseMessageTooBig, closed.Code)

	for origin, accepted := range map[string]bool{"http://good.example": true, "https://good.example": false, "http://good.example:8080": true, "http://bad.example": false} {
		require.Equal(t, accepted, originChecker([]string{"http://good.example"})(&http.Request{Header: http.Header{"Origin": {origin}}}), origin)
	}
	require.True(t, originChecker(nil)(&http.Request{Header: http.Header{"Origin": {"http://localhost"}}}))
	require.True(t, originChecker([]string{"*"})(&http.Request{Header: http.Header{"Origin": {"http://anything"}}}))
	require.True(t, originChecker(nil)(&http.Request{Header: http.Header{}}), "no origin, no browser")
}

func TestIPC_TheSocketIsTheOwners(t *testing.T) {
	servers := newStreamServers(t, WebSocketConfig{})
	info, err := os.Stat(servers.ipc.Path())
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

// TestStream_AFailedSubscriptionEndsItsConnectionWithTheReason holds I18.
func TestStream_AFailedSubscriptionEndsItsConnectionWithTheReason(t *testing.T) {
	servers := newStreamServers(t, WebSocketConfig{})
	connection, _, err := websocket.DefaultDialer.Dial(servers.websocket, nil)
	require.NoError(t, err)
	defer func() { _ = connection.Close() }()
	require.NoError(t, connection.WriteMessage(websocket.TextMessage, []byte(`{"jsonrpc":"2.0","id":1,"method":"test_subscribe","params":["failing"]}`)))
	for {
		_, _, err = connection.ReadMessage()
		if err != nil {
			break
		}
	}
	var closed *websocket.CloseError
	require.ErrorAs(t, err, &closed)
	require.Contains(t, closed.Text, "lost events")
}

// TestStream_BoundsTheSubscriptionsOfAConnection holds D4.
func TestStream_BoundsTheSubscriptionsOfAConnection(t *testing.T) {
	servers := newStreamServers(t, WebSocketConfig{})
	client, err := gethrpc.DialIPC(context.Background(), servers.ipc.Path())
	require.NoError(t, err)
	defer client.Close()
	batch := make([]gethrpc.BatchElem, maxSubscriptionsPerConnection+1)
	for index := range batch {
		batch[index] = gethrpc.BatchElem{Method: "test_subscribe", Args: []any{"ticks"}, Result: new(string)}
	}
	require.NoError(t, client.BatchCall(batch))
	for index := range maxSubscriptionsPerConnection {
		require.NoError(t, batch[index].Error)
	}
	require.EqualError(t, batch[maxSubscriptionsPerConnection].Error, "too many subscriptions on this connection, the limit is 10000")
}

// TestStream_StopEndsTheConnections: the server waits for the connections it
// ended, their subscriptions included.
func TestStream_StopEndsTheConnections(t *testing.T) {
	servers := newStreamServers(t, WebSocketConfig{})
	clients := servers.clients(t)
	subscriptions := map[string]*gethrpc.ClientSubscription{}
	for name, client := range clients {
		subscription, err := client.Subscribe(context.Background(), "test", make(chan int), "ticks")
		require.NoError(t, err, name)
		subscriptions[name] = subscription
	}
	servers.server.Stop()
	servers.ipc.Stop()
	for name, subscription := range subscriptions {
		select {
		case <-subscription.Err():
		case <-time.After(5 * time.Second):
			t.Fatalf("%s: the subscription did not end", name)
		}
	}
	gauge := &prometheusModel.Metric{}
	require.NoError(t, servers.dispatcher.Metrics().Subscriptions.WithLabelValues("ticks").Write(gauge))
	require.Zero(t, gauge.GetGauge().GetValue(), "no subscription is left")
}
