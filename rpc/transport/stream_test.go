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
	"encoding/json"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// streamFixture is a dispatcher with the test methods and a subscription kind
// "ticks" that delivers what is published on the hub.
type streamFixture struct {
	d     *dispatch.Dispatcher
	front *Front
	hub   *dispatch.Hub[int]
}

func newStreamFixture(t *testing.T) *streamFixture {
	t.Helper()
	d := dispatch.New(dispatch.DefaultConfig(), dispatch.NewMetrics(nil))
	t.Cleanup(func() { d.Stop(time.Second) })
	registerTestMethods(d)
	hub := dispatch.NewHub[int]()
	dispatch.AddSub0(d, "t", "ticks", func(ctx context.Context) (string, error) {
		notifier, ok := dispatch.NotifierFrom(ctx)
		if !ok {
			return "", dispatch.ErrNotificationsUnsupported
		}
		sub, err := notifier.NewSubscription(ctx, "ticks")
		if err != nil {
			return "", err
		}
		consumer := hub.Subscribe(4)
		notifier.Go(func() {
			defer consumer.Close()
			defer sub.Close()
			for {
				select {
				case <-sub.Done():
					return
				case v, ok := <-consumer.Events():
					if !ok {
						notifier.Abort(context.Canceled)
						return
					}
					if sub.Notify(v) != nil {
						return
					}
				}
			}
		})
		return sub.ID(), nil
	})
	return &streamFixture{d: d, front: NewFront(d), hub: hub}
}

func TestWebSocket_RequestsBatchesAndSubscriptions(t *testing.T) {
	f := newStreamFixture(t)
	srv, err := NewHTTPServer(HTTPConfig{
		Host: "127.0.0.1",
		WS:   &WSConfig{StreamConfig: StreamConfig{MaxSubscriptions: 2}, Origins: []string{"http://ok.example"}},
	}, f.front)
	require.NoError(t, err)
	require.NoError(t, srv.Start())
	defer func() { require.NoError(t, srv.Stop(context.Background())) }()

	client, err := rpc.DialWebsocket(context.Background(), "ws://"+srv.Addr(), "")
	require.NoError(t, err)
	defer client.Close()

	var res map[string]any
	require.NoError(t, client.Call(&res, "t_echo", 5, "x"))
	require.Equal(t, map[string]any{"a": float64(5), "b": "x"}, res)

	var batch = []rpc.BatchElem{
		{Method: "t_two", Args: []any{1, 2}, Result: new(int)},
		{Method: "t_fail", Result: new(int)},
		{Method: "t_nope", Result: new(int)},
	}
	require.NoError(t, client.BatchCall(batch))
	require.Equal(t, 3, *batch[0].Result.(*int))
	require.ErrorContains(t, batch[1].Error, "boom")
	require.ErrorContains(t, batch[2].Error, "does not exist")

	ticks := make(chan int, 8)
	sub, err := client.Subscribe(context.Background(), "t", ticks, "ticks")
	require.NoError(t, err)
	f.hub.Publish(7)
	f.hub.Publish(8)
	require.Equal(t, 7, <-ticks)
	require.Equal(t, 8, <-ticks)

	// A second subscription fits, a third does not.
	sub2, err := client.Subscribe(context.Background(), "t", make(chan int, 1), "ticks")
	require.NoError(t, err)
	_, err = client.Subscribe(context.Background(), "t", make(chan int, 1), "ticks")
	require.ErrorContains(t, err, "too many subscriptions")
	sub2.Unsubscribe()
	sub.Unsubscribe()

	// Invalid JSON gets a parse error, and the connection closes.
	conn, _, err := websocket.DefaultDialer.Dial("ws://"+srv.Addr(), nil)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"jsonrpc":"2.0",`)))
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)
	require.JSONEq(t, `{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"parse error"}}`, string(data))
	_, _, err = conn.ReadMessage()
	require.Error(t, err)
}

func TestWebSocket_PathPrefixAndOrigins(t *testing.T) {
	f := newStreamFixture(t)
	srv, err := NewHTTPServer(HTTPConfig{
		Host: "127.0.0.1", DisableRPC: true,
		WS:       &WSConfig{Origins: []string{"http://ok.example", "*.sub.example"}},
		WSPrefix: "/ws",
	}, f.front)
	require.NoError(t, err)
	require.NoError(t, srv.Start())
	defer func() { require.NoError(t, srv.Stop(context.Background())) }()

	dial := func(path, origin string) error {
		header := http.Header{}
		if origin != "" {
			header.Set("Origin", origin)
		}
		conn, _, err := websocket.DefaultDialer.Dial("ws://"+srv.Addr()+path, header)
		if err == nil {
			conn.Close()
		}
		return err
	}
	require.NoError(t, dial("/ws", ""))
	require.NoError(t, dial("/ws/more", "http://ok.example"))
	require.Error(t, dial("/ws", "http://evil.example"))
	require.Error(t, dial("/other", ""))

	// The listener serves no plain RPC.
	resp, err := http.Post("http://"+srv.Addr()+"/ws", "application/json", nil)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestIPC_RequestsAndSubscriptions(t *testing.T) {
	f := newStreamFixture(t)
	path := filepath.Join(t.TempDir(), "sub", "x.ipc")
	srv := NewIPCServer(path, StreamConfig{}, f.front)
	require.NoError(t, srv.Start())
	require.NoError(t, srv.Start()) // idempotent

	client, err := rpc.DialIPC(context.Background(), path)
	require.NoError(t, err)
	defer client.Close()

	var sum int
	require.NoError(t, client.Call(&sum, "t_two", 20, 22))
	require.Equal(t, 42, sum)
	require.ErrorContains(t, client.Call(&sum, "t_coded"), "revert")

	ticks := make(chan int, 4)
	sub, err := client.Subscribe(context.Background(), "t", ticks, "ticks")
	require.NoError(t, err)
	f.hub.Publish(3)
	require.Equal(t, 3, <-ticks)
	sub.Unsubscribe()

	// A raw connection: several messages in a row, answered in order, and a
	// notification without answer.
	conn, err := net.Dial("unix", path)
	require.NoError(t, err)
	defer conn.Close()
	_, err = conn.Write([]byte(`{"jsonrpc":"2.0","id":1,"method":"t_two","params":[1,1]}` + "\n" +
		`{"jsonrpc":"2.0","method":"t_nil"}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"t_two","params":[2,2]}` + "\n"))
	require.NoError(t, err)
	dec := json.NewDecoder(conn)
	answers := map[int]float64{}
	for range 2 {
		var m struct {
			ID     int     `json:"id"`
			Result float64 `json:"result"`
		}
		require.NoError(t, conn.SetReadDeadline(time.Now().Add(2*time.Second)))
		require.NoError(t, dec.Decode(&m))
		answers[m.ID] = m.Result
	}
	require.Equal(t, map[int]float64{1: 2, 2: 4}, answers)

	require.NoError(t, srv.Stop())
	_, err = rpc.DialIPC(context.Background(), path)
	require.Error(t, err)
}

func TestIPC_ClosingTheServerClosesTheConnections(t *testing.T) {
	f := newStreamFixture(t)
	path := filepath.Join(t.TempDir(), "x.ipc")
	srv := NewIPCServer(path, StreamConfig{}, f.front)
	require.NoError(t, srv.Start())
	conn, err := net.Dial("unix", path)
	require.NoError(t, err)
	defer conn.Close()

	stopped := make(chan struct{})
	go func() { _ = srv.Stop(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(3 * time.Second):
		t.Fatal("Stop did not return with a connection open")
	}
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	_, err = conn.Read(make([]byte, 1))
	require.Error(t, err)
}

func TestHTTPServer_ValidatesThePrefix(t *testing.T) {
	d := dispatch.New(dispatch.DefaultConfig(), dispatch.NewMetrics(nil))
	defer d.Stop(0)
	_, err := NewHTTPServer(HTTPConfig{Prefix: "rpc"}, NewFront(d))
	require.ErrorContains(t, err, "does not contain leading")
	_, err = NewHTTPServer(HTTPConfig{Prefix: "/rpc?x"}, NewFront(d))
	require.ErrorContains(t, err, "URL meta-characters")
}

func TestHTTPServer_PathPrefixAndCORS(t *testing.T) {
	f := newStreamFixture(t)
	srv, err := NewHTTPServer(HTTPConfig{
		Host: "127.0.0.1", Prefix: "/rpc", Cors: []string{"http://app.example"},
		VHosts: []string{"localhost"},
	}, f.front)
	require.NoError(t, err)
	require.NoError(t, srv.Start())
	defer func() { require.NoError(t, srv.Stop(context.Background())) }()
	base := "http://" + srv.Addr()

	call := `{"jsonrpc":"2.0","id":1,"method":"t_two","params":[1,2]}`
	post := func(path string, headers map[string]string) *http.Response {
		req, err := http.NewRequest(http.MethodPost, base+path, strings.NewReader(call))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		return resp
	}
	require.Equal(t, http.StatusOK, post("/rpc", nil).StatusCode)
	require.Equal(t, http.StatusOK, post("/rpc/sub", nil).StatusCode)
	require.Equal(t, http.StatusNotFound, post("/", nil).StatusCode)
	require.Equal(t, http.StatusNotFound, post("/other", nil).StatusCode)

	resp := post("/rpc", map[string]string{"Origin": "http://app.example"})
	require.Equal(t, "http://app.example", resp.Header.Get("Access-Control-Allow-Origin"))
	resp = post("/rpc", map[string]string{"Origin": "http://evil.example"})
	require.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))

	// Gzip, when the client accepts it.
	req, _ := http.NewRequest(http.MethodPost, base+"/rpc", strings.NewReader(call))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("X-Probe", "1")
	client := &http.Client{Transport: &http.Transport{DisableCompression: true}}
	gz, err := client.Do(req)
	require.NoError(t, err)
	gz.Body.Close()
	require.Equal(t, "gzip", gz.Header.Get("Content-Encoding"))
}

func TestHTTPServer_StopRejectsConnections(t *testing.T) {
	f := newStreamFixture(t)
	srv, err := NewHTTPServer(HTTPConfig{Host: "127.0.0.1"}, f.front)
	require.NoError(t, err)
	require.Empty(t, srv.Addr())
	require.NoError(t, srv.Stop(context.Background())) // not started
	require.NoError(t, srv.Start())
	addr := srv.Addr()
	require.NoError(t, srv.Stop(context.Background()))
	_, err = net.DialTimeout("tcp", addr, 500*time.Millisecond)
	require.Error(t, err)
}
