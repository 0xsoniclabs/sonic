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

package transport_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/transport"
)

// The test methods, served by the module and, for the differential test, by
// go-ethereum's server.

type echoArgs struct {
	S string `rpc:"s"`
}

type sizeArgs struct {
	N int `rpc:"n"`
}

type codedError struct{}

func (codedError) Error() string  { return "coded" }
func (codedError) ErrorCode() int { return 42 }
func (codedError) ErrorData() any { return "0x01" }

const slowFor = 1500 * time.Millisecond

func register(d *dispatch.Dispatcher) {
	d.Add("test_echo", dispatch.Func(func(_ context.Context, p echoArgs) (any, error) { return p.S, nil }))
	d.Add("test_null", dispatch.Func(func(context.Context, dispatch.None) (any, error) { return nil, nil }))
	d.Add("test_fail", dispatch.Func(func(context.Context, dispatch.None) (any, error) { return nil, errors.New("boom") }))
	d.Add("test_coded", dispatch.Func(func(context.Context, dispatch.None) (any, error) { return nil, codedError{} }))
	d.Add("test_slow", dispatch.Func(func(context.Context, dispatch.None) (any, error) {
		time.Sleep(slowFor)
		return "done", nil
	}))
	d.Add("test_big", dispatch.Func(func(_ context.Context, p sizeArgs) (any, error) { return strings.Repeat("x", p.N), nil }))
	d.Add("other_echo", dispatch.Func(func(_ context.Context, p echoArgs) (any, error) { return p.S, nil }))
	d.AddSubscription("test", "events", dispatch.Func(func(ctx context.Context, p sizeArgs) (any, error) {
		n, ok := dispatch.NotifierFrom(ctx)
		if !ok {
			return nil, dispatch.ErrNotificationsUnsupported
		}
		sub, err := n.Subscribe("test", "events")
		if err != nil {
			return nil, err
		}
		n.Go(func(ctx context.Context) {
			for i := range p.N {
				sub.Notify(i)
			}
		})
		return sub.ID(), nil
	}))
}

type gethService struct{}

func (gethService) Echo(s string) string { return s }
func (gethService) Null() *int           { return nil }
func (gethService) Fail() error          { return errors.New("boom") }
func (gethService) Coded() error         { return gethCoded{} }
func (gethService) Slow() string {
	time.Sleep(slowFor)
	return "done"
}
func (gethService) Big(n int) string { return strings.Repeat("x", n) }

type gethCoded struct{}

func (gethCoded) Error() string          { return "coded" }
func (gethCoded) ErrorCode() int         { return 42 }
func (gethCoded) ErrorData() interface{} { return "0x01" }

const (
	batchItems    = 4
	batchResponse = 300
)

// module starts a dispatcher with the test methods behind a server.
func module(t *testing.T, cfg transport.Config, front transport.FrontConfig) (*transport.Server, *prometheus.Registry) {
	t.Helper()
	reg := prometheus.NewRegistry()
	d := dispatch.New(dispatch.DefaultConfig(), dispatch.NewMetrics(reg))
	register(d)
	f := transport.NewFront(d, front)
	srv := transport.NewServer(d, f, cfg, func() any { return map[string]any{"head": "0x5", "lag": 0} })
	d.Start()
	require.NoError(t, srv.Start())
	t.Cleanup(func() {
		srv.Close(context.Background())
		d.Stop(time.Second)
	})
	return srv, reg
}

func httpConfig(write time.Duration) *transport.HTTPConfig {
	timeouts := transport.Timeouts{Read: 30 * time.Second, ReadHeader: 30 * time.Second, Write: write, Idle: 30 * time.Second}
	return &transport.HTTPConfig{Host: "127.0.0.1", Vhosts: []string{"localhost", "127.0.0.1"}, Cors: []string{"https://allowed.example"},
		Timeouts: timeouts, FallbackTimeouts: timeouts}
}

func limits() transport.Limits {
	return transport.Limits{BatchRequestLimit: batchItems, BatchResponseMaxSize: batchResponse}
}

// geth starts go-ethereum's server with the same methods and limits.
func geth(t *testing.T, write time.Duration) string {
	t.Helper()
	srv := gethrpc.NewServer()
	require.NoError(t, srv.RegisterName("test", gethService{}))
	srv.SetBatchLimits(batchItems, batchResponse)
	ts := httptest.NewUnstartedServer(srv)
	ts.Config.WriteTimeout = write
	ts.Start()
	t.Cleanup(func() {
		ts.Close()
		srv.Stop()
	})
	return ts.URL
}

func post(t *testing.T, url, body string) (int, string, []byte) {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, resp.Header.Get("content-type"), raw
}

func decode(t *testing.T, raw []byte) any {
	t.Helper()
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	var v any
	require.NoError(t, json.Unmarshal(raw, &v), string(raw))
	return v
}

// TestEnvelope_MatchesGoEthereum is the differential test of Q5 and D3: the
// same requests sent to go-ethereum's server and to the module, compared
// answer by answer, the configured limits included.
func TestEnvelope_MatchesGoEthereum(t *testing.T) {
	const write = 1 * time.Second
	srv, _ := module(t, transport.Config{HTTP: httpConfig(write), Limits: limits()}, transport.FrontConfig{})
	ours, theirs := srv.HTTPEndpoint(), geth(t, write)
	call := func(id, method string, params string) string {
		return fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"method":%q,"params":%s}`, id, method, params)
	}
	cases := map[string]string{
		"call":               call("1", "test_echo", `["a"]`),
		"string id":          call(`"x"`, "test_echo", `["a"]`),
		"float id":           call("1.5", "test_echo", `["a"]`),
		"null result":        call("2", "test_null", `[]`),
		"no params":          `{"jsonrpc":"2.0","id":3,"method":"test_null"}`,
		"error":              call("3", "test_fail", `[]`),
		"coded error":        call("4", "test_coded", `[]`),
		"unknown method":     call("5", "test_nope", `[]`),
		"unknown namespace":  call("5", "nope_nope", `[]`),
		"notification":       `{"jsonrpc":"2.0","method":"test_echo","params":["a"]}`,
		"null id":            call("null", "test_echo", `["a"]`),
		"wrong version":      `{"jsonrpc":"1.0","id":1,"method":"test_echo","params":["a"]}`,
		"no method":          `{"jsonrpc":"2.0","id":1}`,
		"not json":           `{"jsonrpc":`,
		"not an object":      `"text"`,
		"empty batch":        `[]`,
		"batch":              "[" + call("1", "test_echo", `["a"]`) + "," + call("2", "test_fail", `[]`) + "]",
		"batch notification": "[" + `{"jsonrpc":"2.0","method":"test_echo","params":["a"]}` + "]",
		"batch mixed":        "[" + call("1", "test_echo", `["a"]`) + `,{"jsonrpc":"2.0","method":"test_echo","params":["b"]},1,{}` + "]",
		"batch too large":    "[" + strings.Repeat(call("1", "test_null", `[]`)+",", batchItems) + call("9", "test_null", `[]`) + "]",
		"response too large": "[" + call("1", "test_big", `[200]`) + "," + call("2", "test_big", `[200]`) + "," + call("3", "test_echo", `["a"]`) + "]",
		"slow call":          call("1", "test_slow", `[]`),
		"slow batch":         "[" + call("1", "test_echo", `["a"]`) + "," + call("2", "test_slow", `[]`) + "," + call("3", "test_echo", `["b"]`) + "]",
		"subscribe":          call("1", "test_subscribe", `["events",1]`),
		"unsubscribe":        call("1", "test_unsubscribe", `["0x1"]`),
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ourStatus, ourType, ourBody := post(t, ours, body)
			theirStatus, theirType, theirBody := post(t, theirs, body)
			require.Equal(t, theirStatus, ourStatus)
			require.Equal(t, theirType, ourType)
			require.Equal(t, decode(t, theirBody), decode(t, ourBody), "geth: %s\nours: %s", theirBody, ourBody)
		})
	}
}

// TestHTTP_RequestChecksMatchGoEthereum: what the server refuses before
// reading the request.
func TestHTTP_RequestChecksMatchGoEthereum(t *testing.T) {
	srv, _ := module(t, transport.Config{HTTP: httpConfig(time.Second)}, transport.FrontConfig{})
	theirs := geth(t, time.Second)
	for _, url := range []string{srv.HTTPEndpoint(), theirs} {
		req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(`{}`))
		require.NoError(t, err)
		req.Header.Set("content-type", "text/plain")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)
		resp.Body.Close()

		req, err = http.NewRequest(http.MethodPut, url, strings.NewReader(`{}`))
		require.NoError(t, err)
		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
		resp.Body.Close()

		status, _, _ := post(t, url, strings.Repeat(" ", 5*1024*1024+1))
		require.Equal(t, http.StatusRequestEntityTooLarge, status, "an oversize body")

		resp, err = http.Get(url)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, "the health check of load balancers")
		resp.Body.Close()
	}
}

// TestHTTP_HostsOriginsAndHealth is D11: a disallowed Host (DNS rebinding)
// and CORS as configured; /health reports the head and the lag.
func TestHTTP_HostsOriginsAndHealth(t *testing.T) {
	srv, _ := module(t, transport.Config{HTTP: httpConfig(time.Second)}, transport.FrontConfig{})
	url := srv.HTTPEndpoint()
	body := `{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["a"]}`

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("content-type", "application/json")
	req.Host = "evil.example"
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()

	req, err = http.NewRequest(http.MethodOptions, url, nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "https://allowed.example")
	req.Header.Set("Access-Control-Request-Method", "POST")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, "https://allowed.example", resp.Header.Get("Access-Control-Allow-Origin"))
	resp.Body.Close()

	req.Header.Set("Origin", "https://other.example")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
	resp.Body.Close()

	resp, err = http.Get(url + "/health")
	require.NoError(t, err)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()
	require.JSONEq(t, `{"head":"0x5","lag":0}`, string(raw))
}

func TestHTTP_ResponsesAreCompressedOnRequest(t *testing.T) {
	srv, _ := module(t, transport.Config{HTTP: httpConfig(time.Second)}, transport.FrontConfig{})
	req, err := http.NewRequest(http.MethodPost, srv.HTTPEndpoint(), strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"test_big","params":[2000]}`))
	require.NoError(t, err)
	req.Header.Set("content-type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err := (&http.Transport{DisableCompression: true}).RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
}

// TestTransports_NamespacesPerTransport is D8 and S1: a namespace a transport
// does not enable answers -32601 there, and rpc_modules lists what it serves.
func TestTransports_NamespacesPerTransport(t *testing.T) {
	ipc := filepath.Join(t.TempDir(), "sonic.ipc")
	ws := &transport.WSConfig{Host: "127.0.0.1", Origins: []string{"*"}}
	srv, _ := module(t, transport.Config{HTTP: httpConfig(time.Second), WS: ws, IPC: ipc}, transport.FrontConfig{
		Modules: map[transport.Kind][]string{transport.HTTP: {"test"}, transport.WS: {"other"}},
	})
	ctx := context.Background()
	httpClient, err := gethrpc.DialContext(ctx, srv.HTTPEndpoint())
	require.NoError(t, err)
	defer httpClient.Close()
	wsClient, err := gethrpc.DialContext(ctx, srv.WSEndpoint())
	require.NoError(t, err)
	defer wsClient.Close()
	ipcClient, err := gethrpc.DialIPC(ctx, ipc)
	require.NoError(t, err)
	defer ipcClient.Close()

	var s string
	require.NoError(t, httpClient.Call(&s, "test_echo", "a"))
	require.ErrorContains(t, httpClient.Call(&s, "other_echo", "a"), "the method other_echo does not exist/is not available")
	require.ErrorContains(t, wsClient.Call(&s, "test_echo", "a"), "the method test_echo does not exist/is not available")
	require.NoError(t, wsClient.Call(&s, "other_echo", "a"))
	require.NoError(t, ipcClient.Call(&s, "test_echo", "a"))
	require.NoError(t, ipcClient.Call(&s, "other_echo", "a"))

	var modules map[string]string
	modules = nil
	require.NoError(t, httpClient.Call(&modules, "rpc_modules"))
	require.Equal(t, map[string]string{"test": "1.0", "rpc": "1.0"}, modules)
	modules = nil
	require.NoError(t, wsClient.Call(&modules, "rpc_modules"))
	require.Equal(t, map[string]string{"other": "1.0", "rpc": "1.0"}, modules)
	modules = nil
	require.NoError(t, ipcClient.Call(&modules, "rpc_modules"))
	require.Equal(t, map[string]string{"test": "1.0", "other": "1.0", "rpc": "1.0"}, modules)

	ch := make(chan int, 1)
	_, err = wsClient.Subscribe(ctx, "test", ch, "events", 1)
	require.ErrorContains(t, err, `no "events" subscription in test namespace`)
}

// TestStreams_SubscriptionsOverWebSocketAndIPC is D4 and D5.
func TestStreams_SubscriptionsOverWebSocketAndIPC(t *testing.T) {
	ipc := filepath.Join(t.TempDir(), "sonic.ipc")
	srv, reg := module(t, transport.Config{WS: &transport.WSConfig{Host: "127.0.0.1", Origins: []string{"*"}}, IPC: ipc}, transport.FrontConfig{})
	info, err := os.Stat(ipc)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	ctx := context.Background()
	for _, url := range []string{srv.WSEndpoint(), ipc} {
		client, err := gethrpc.DialContext(ctx, url)
		require.NoError(t, err)
		ch := make(chan int, 10)
		sub, err := client.Subscribe(ctx, "test", ch, "events", 3)
		require.NoError(t, err)
		for want := range 3 {
			select {
			case got := <-ch:
				require.Equal(t, want, got)
			case <-time.After(5 * time.Second):
				t.Fatalf("%s: no notification", url)
			}
		}
		sub.Unsubscribe()
		var s string
		require.NoError(t, client.Call(&s, "test_echo", "a"))
		batch := []gethrpc.BatchElem{{Method: "test_echo", Args: []any{"b"}, Result: new(string)}, {Method: "test_fail", Result: new(string)}}
		require.NoError(t, client.BatchCall(batch))
		require.Equal(t, "b", *batch[0].Result.(*string))
		require.EqualError(t, batch[1].Error, "boom")
		client.Close()
	}
	require.Eventually(t, func() bool {
		families, err := reg.Gather()
		require.NoError(t, err)
		for _, f := range families {
			if f.GetName() == "rpc_connections" {
				for _, m := range f.GetMetric() {
					if m.GetGauge().GetValue() != 0 {
						return false
					}
				}
				return len(f.GetMetric()) == 2
			}
		}
		return false
	}, 5*time.Second, 10*time.Millisecond, "connections are counted per transport and closed")
}

// TestWebSocket_OriginCheck is D11: a disallowed origin is refused.
func TestWebSocket_OriginCheck(t *testing.T) {
	srv, _ := module(t, transport.Config{WS: &transport.WSConfig{Host: "127.0.0.1", Origins: []string{"https://allowed.example"}}}, transport.FrontConfig{})
	header := http.Header{"Origin": {"https://evil.example"}}
	_, resp, err := websocket.DefaultDialer.Dial(srv.WSEndpoint(), header)
	require.Error(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
	header = http.Header{"Origin": {"https://allowed.example"}}
	conn, _, err := websocket.DefaultDialer.Dial(srv.WSEndpoint(), header)
	require.NoError(t, err)
	conn.Close()
}

// TestWebSocket_AClientThatDoesNotReadIsDisconnected is D4 and D11.
func TestWebSocket_AClientThatDoesNotReadIsDisconnected(t *testing.T) {
	srv, _ := module(t, transport.Config{WS: &transport.WSConfig{Host: "127.0.0.1", Origins: []string{"*"}}}, transport.FrontConfig{})
	conn, _, err := websocket.DefaultDialer.Dial(srv.WSEndpoint(), nil)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"jsonrpc":"2.0","id":1,"method":"test_subscribe","params":["events",100000]}`)))
	// The client reads nothing; the subscription's queue overflows and the
	// server closes the connection.
	time.Sleep(time.Second)
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(15*time.Second)))
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			var closeErr *websocket.CloseError
			if errors.As(err, &closeErr) {
				require.Contains(t, closeErr.Text, "fell behind")
			}
			return
		}
	}
}

// TestWebSocket_SubscriptionCapPerConnection is D4.
func TestWebSocket_SubscriptionCapPerConnection(t *testing.T) {
	srv, _ := module(t, transport.Config{WS: &transport.WSConfig{Host: "127.0.0.1", Origins: []string{"*"}}}, transport.FrontConfig{})
	client, err := gethrpc.DialContext(context.Background(), srv.WSEndpoint())
	require.NoError(t, err)
	defer client.Close()
	for range 1000 {
		_, err := client.Subscribe(context.Background(), "test", make(chan int, 1), "events", 0)
		require.NoError(t, err)
	}
	_, err = client.Subscribe(context.Background(), "test", make(chan int, 1), "events", 0)
	require.ErrorContains(t, err, "too many subscriptions, the limit is 1000")
}

// TestIPC_ForwardsWhatTheModuleDoesNotServe is D5 and X3: on IPC the node's
// handler answers the namespaces that stay with it; elsewhere only the p2p
// facts are forwarded.
func TestIPC_ForwardsWhatTheModuleDoesNotServe(t *testing.T) {
	ipc := filepath.Join(t.TempDir(), "sonic.ipc")
	forward := func(_ context.Context, call dispatch.Call) (json.RawMessage, error) {
		switch call.Method {
		case "rpc_modules":
			return json.RawMessage(`{"admin":"1.0","net":"1.0"}`), nil
		case "admin_peers", "net_peerCount":
			return json.RawMessage(`"node:` + call.Method + `"`), nil
		}
		return nil, dispatch.MethodNotFound(call.Method)
	}
	srv, _ := module(t, transport.Config{HTTP: httpConfig(time.Second), IPC: ipc}, transport.FrontConfig{
		Forward: forward, ForwardEverywhere: []string{"net_peerCount"},
	})
	ctx := context.Background()
	ipcClient, err := gethrpc.DialIPC(ctx, ipc)
	require.NoError(t, err)
	defer ipcClient.Close()
	httpClient, err := gethrpc.DialContext(ctx, srv.HTTPEndpoint())
	require.NoError(t, err)
	defer httpClient.Close()

	var s string
	require.NoError(t, ipcClient.Call(&s, "admin_peers"))
	require.Equal(t, "node:admin_peers", s)
	require.ErrorContains(t, httpClient.Call(&s, "admin_peers"), "the method admin_peers does not exist/is not available")
	require.NoError(t, httpClient.Call(&s, "net_peerCount"))
	require.Equal(t, "node:net_peerCount", s)
	require.ErrorContains(t, ipcClient.Call(&s, "admin_nope"), "the method admin_nope does not exist/is not available")

	var modules map[string]string
	modules = nil
	require.NoError(t, ipcClient.Call(&modules, "rpc_modules"))
	require.Contains(t, modules, "admin")
	modules = nil
	require.NoError(t, httpClient.Call(&modules, "rpc_modules"))
	require.NotContains(t, modules, "admin")
	require.Contains(t, modules, "net")
}

// TestServer_SharedPortAndClose: HTTP and WebSocket on one port listen on the
// WebSocket host, as go-ethereum's node; Close ends the endpoints.
func TestServer_SharedPortAndClose(t *testing.T) {
	srv, _ := module(t, transport.Config{HTTP: httpConfig(time.Second), WS: &transport.WSConfig{Host: "127.0.0.1", Origins: []string{"*"}}}, transport.FrontConfig{})
	require.NotEmpty(t, srv.HTTPEndpoint())
	require.Equal(t, strings.TrimPrefix(srv.HTTPEndpoint(), "http://"), strings.TrimPrefix(srv.WSEndpoint(), "ws://"), "ports equal: one listener")
	client, err := gethrpc.DialContext(context.Background(), srv.WSEndpoint())
	require.NoError(t, err)
	var s string
	require.NoError(t, client.Call(&s, "test_echo", "a"))
	client.Close()
	endpoint := srv.HTTPEndpoint()
	srv.Close(context.Background())
	_, err = http.Post(endpoint, "application/json", strings.NewReader(`{}`))
	require.Error(t, err)
	require.Empty(t, srv.HTTPEndpoint())
}
