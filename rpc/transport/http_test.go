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
	"bytes"
	"compress/gzip"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/node"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

// envelopeBodies are requests whose answers the module gives as go-ethereum's
// server does: calls, notifications, batches, malformed messages and the
// errors of each.
var envelopeBodies = map[string]string{
	"call":                        `{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["hi"]}`,
	"string id":                   `{"jsonrpc":"2.0","id":"a","method":"test_add","params":[1,2]}`,
	"null id":                     `{"jsonrpc":"2.0","id":null,"method":"test_echo","params":["hi"]}`,
	"object id":                   `{"jsonrpc":"2.0","id":{},"method":"test_echo","params":["hi"]}`,
	"array id":                    `{"jsonrpc":"2.0","id":[1],"method":"test_echo","params":["hi"]}`,
	"html in id":                  `{"jsonrpc":"2.0","id":"<&>","method":"test_echo","params":["<b>"]}`,
	"notification":                `{"jsonrpc":"2.0","method":"test_echo","params":["hi"]}`,
	"wrong version":               `{"jsonrpc":"1.0","id":1,"method":"test_echo","params":["hi"]}`,
	"no version":                  `{"id":1,"method":"test_echo","params":["hi"]}`,
	"no method":                   `{"jsonrpc":"2.0","id":1}`,
	"method of the wrong type":    `{"jsonrpc":"2.0","id":1,"method":5}`,
	"unknown method":              `{"jsonrpc":"2.0","id":1,"method":"test_unknown"}`,
	"unknown namespace":           `{"jsonrpc":"2.0","id":1,"method":"nope_method"}`,
	"method name too long":        `{"jsonrpc":"2.0","id":1,"method":"test_` + strings.Repeat("a", 2100) + `"}`,
	"error with code and data":    `{"jsonrpc":"2.0","id":1,"method":"test_fail"}`,
	"panic":                       `{"jsonrpc":"2.0","id":1,"method":"test_panic"}`,
	"null result":                 `{"jsonrpc":"2.0","id":1,"method":"test_nothing"}`,
	"too many params":             `{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["a","b"]}`,
	"missing param":               `{"jsonrpc":"2.0","id":1,"method":"test_add","params":[1]}`,
	"object params":               `{"jsonrpc":"2.0","id":1,"method":"test_echo","params":{"text":"hi"}}`,
	"wrong param type":            `{"jsonrpc":"2.0","id":1,"method":"test_add","params":["1",2]}`,
	"modules":                     `{"jsonrpc":"2.0","id":1,"method":"rpc_modules"}`,
	"subscribe":                   `{"jsonrpc":"2.0","id":1,"method":"eth_subscribe","params":["newHeads"]}`,
	"response is ignored":         `{"jsonrpc":"2.0","id":1,"result":"0x1"}`,
	"subscription notice ignored": `{"jsonrpc":"2.0","method":"eth_subscription","params":{"subscription":"0x1","result":1}}`,
	"null":                        `null`,
	"number":                      `5`,
	"string":                      `"call"`,
	"empty object":                `{}`,
	"invalid json":                `{"jsonrpc":"2.0",`,
	"trailing garbage":            `{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["hi"]} garbage`,
	"empty body":                  ``,
	"empty batch":                 `[]`,
	"batch": `[{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["a"]},` +
		`{"jsonrpc":"2.0","method":"test_echo","params":["b"]},` +
		`{"jsonrpc":"2.0","id":3,"method":"test_unknown"},null,5,` +
		`{"jsonrpc":"2.0","id":4,"result":1},{"jsonrpc":"2.0","id":5,"method":"test_panic"}]`,
	"batch of responses only":     `[{"jsonrpc":"2.0","id":1,"result":1}]`,
	"batch of notifications only": `[{"jsonrpc":"2.0","method":"test_echo","params":["b"]}]`,
	"batch over the item limit": `[{"jsonrpc":"2.0","id":7,"result":1},{"jsonrpc":"2.0","id":8,"method":"test_echo","params":["a"]},` +
		`{"jsonrpc":"2.0","id":9,"method":"test_echo","params":["b"]},{"jsonrpc":"2.0","id":10,"method":"test_echo","params":["c"]},` +
		`{"jsonrpc":"2.0","id":11,"method":"test_echo","params":["d"]}]`,
	"batch over the response limit": `[{"jsonrpc":"2.0","id":1,"method":"test_large","params":[60]},` +
		`{"jsonrpc":"2.0","id":2,"method":"test_large","params":[60]},{"jsonrpc":"2.0","method":"test_echo","params":["n"]},` +
		`{"jsonrpc":"2.0","id":3,"method":"test_echo","params":["c"]}]`,
}

// testLimits are the batch limits of the envelope tests.
var testLimits = BatchLimits{Items: 4, ResponseBytes: 100}

func TestHTTP_AnswersAsGoEthereumServer(t *testing.T) {
	reference := newReference(t)
	reference.SetBatchLimits(testLimits.Items, testLimits.ResponseBytes)
	module := &httpHandler{front: NewFront(newTestDispatcher(t), nil, "http.api"), limits: testLimits}
	for name, body := range envelopeBodies {
		t.Run(name, func(t *testing.T) {
			want := serveRecorded(reference, body)
			got := serveRecorded(module, body)
			require.Equal(t, want.Code, got.Code)
			require.Equal(t, want.Body.String(), got.Body.String())
			require.Equal(t, want.Header(), got.Header())
		})
	}
}

func serveRecorded(handler http.Handler, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	request.Header.Set("content-type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

// The HTTP requests that are not JSON-RPC calls, through the whole stack of
// each server: health checks, methods, content types, sizes, paths, hosts,
// CORS and compression.
func TestHTTP_StackAnswersAsGoEthereumServer(t *testing.T) {
	config := HTTPConfig{
		PathPrefix:         "/rpc",
		CorsAllowedOrigins: []string{"https://allowed.example"},
		VirtualHosts:       []string{"allowed.example"},
	}
	referenceURL := startReferenceHTTP(t, config)
	moduleURL := startModuleHTTP(t, config)
	call := `{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["hi"]}`
	cases := map[string]struct {
		method, path, body string
		headers            map[string]string
	}{
		"call":                  {http.MethodPost, "/rpc", call, nil},
		"call below the prefix": {http.MethodPost, "/rpc/sub", call, nil},
		"root without prefix":   {http.MethodPost, "/", call, nil},
		"other path":            {http.MethodPost, "/other", call, nil},
		"health check":          {http.MethodGet, "/rpc", "", nil},
		"get with a query":      {http.MethodGet, "/rpc?x=1", "", nil},
		"put":                   {http.MethodPut, "/rpc", call, nil},
		"delete":                {http.MethodDelete, "/rpc", call, nil},
		"options":               {http.MethodOptions, "/rpc", "", map[string]string{"content-type": ""}},
		"wrong content type":    {http.MethodPost, "/rpc", call, map[string]string{"content-type": "text/plain"}},
		"json-rpc content type": {http.MethodPost, "/rpc", call, map[string]string{"content-type": "application/json-rpc"}},
		"content type params":   {http.MethodPost, "/rpc", call, map[string]string{"content-type": "application/json; charset=utf-8"}},
		"body too large":        {http.MethodPost, "/rpc", `"` + strings.Repeat("a", httpBodyLimit) + `"`, nil},
		"allowed host":          {http.MethodPost, "/rpc", call, map[string]string{"host": "allowed.example"}},
		"allowed host and port": {http.MethodPost, "/rpc", call, map[string]string{"host": "allowed.example:80"}},
		"refused host":          {http.MethodPost, "/rpc", call, map[string]string{"host": "evil.example"}},
		"ip host":               {http.MethodPost, "/rpc", call, map[string]string{"host": "10.1.2.3"}},
		"allowed origin":        {http.MethodPost, "/rpc", call, map[string]string{"origin": "https://allowed.example"}},
		"refused origin":        {http.MethodPost, "/rpc", call, map[string]string{"origin": "https://evil.example"}},
		"preflight": {http.MethodOptions, "/rpc", "", map[string]string{
			"origin": "https://allowed.example", "access-control-request-method": "POST"}},
		"gzip":             {http.MethodPost, "/rpc", call, map[string]string{"accept-encoding": "gzip"}},
		"gzip of an error": {http.MethodPost, "/rpc", `{"jsonrpc":"2.0",`, map[string]string{"accept-encoding": "gzip"}},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			want := send(t, referenceURL, test.method, test.path, test.body, test.headers)
			got := send(t, moduleURL, test.method, test.path, test.body, test.headers)
			require.Equal(t, want, got)
		})
	}
}

// The write timeout is answered before the server cuts the response off: for
// a call and for a batch, whose answers so far go out with the error.
func TestHTTP_AnswersTheWriteTimeoutAsGoEthereumServer(t *testing.T) {
	config := HTTPConfig{Timeouts: Timeouts{Read: time.Second, ReadHeader: time.Second, Write: time.Second, Idle: time.Second}}
	referenceURL := startReferenceHTTP(t, config)
	moduleURL := startModuleHTTP(t, config)
	cases := map[string]string{
		"call": `{"jsonrpc":"2.0","id":1,"method":"test_slow"}`,
		"batch": `[{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["a"]},{"jsonrpc":"2.0","id":2,"method":"test_slow"},` +
			`{"jsonrpc":"2.0","id":3,"method":"test_echo","params":["c"]},{"jsonrpc":"2.0","method":"test_echo","params":["n"]}]`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			want := send(t, referenceURL, http.MethodPost, "/", body, nil)
			got := send(t, moduleURL, http.MethodPost, "/", body, nil)
			require.Equal(t, want, got)
			require.Contains(t, got.body, "request timed out")
		})
	}
}

func TestHTTP_ValidatesThePathPrefix(t *testing.T) {
	require.NoError(t, ValidatePrefix("HTTP", ""))
	require.NoError(t, ValidatePrefix("HTTP", "/rpc"))
	require.EqualError(t, ValidatePrefix("HTTP", "rpc"), `HTTP RPC path prefix "rpc" does not contain leading "/"`)
	require.EqualError(t, ValidatePrefix("HTTP", "/rpc?x"), `HTTP RPC path prefix "/rpc?x" contains URL meta-characters`)
}

func TestSanitize_RaisesTimeoutsBelowASecond(t *testing.T) {
	require.Equal(t, defaultTimeouts, sanitize(Timeouts{Read: time.Millisecond}))
	kept := Timeouts{Read: 2 * time.Second, ReadHeader: 3 * time.Second, Write: 4 * time.Second, Idle: 5 * time.Second}
	require.Equal(t, kept, sanitize(kept))
}

type answer struct {
	status  int
	headers map[string]string
	body    string
}

// comparedHeaders are the response headers a client sees the same from both
// servers; Date differs by its nature.
var comparedHeaders = []string{"Content-Type", "Content-Encoding", "Content-Length", "Vary",
	"Access-Control-Allow-Origin", "Access-Control-Allow-Methods", "Access-Control-Allow-Headers", "Access-Control-Max-Age"}

func send(t *testing.T, url, method, path, body string, headers map[string]string) answer {
	t.Helper()
	request, err := http.NewRequest(method, url+path, strings.NewReader(body))
	require.NoError(t, err)
	request.Header.Set("content-type", "application/json")
	for key, value := range headers {
		if key == "host" {
			request.Host = value
			continue
		}
		request.Header.Set(key, value)
	}
	transport := &http.Transport{DisableCompression: true}
	response, err := (&http.Client{Transport: transport}).Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()
	raw, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	if response.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(bytes.NewReader(raw))
		require.NoError(t, err)
		raw, err = io.ReadAll(reader)
		require.NoError(t, err)
	}
	got := answer{status: response.StatusCode, headers: map[string]string{}, body: string(raw)}
	for _, header := range comparedHeaders {
		got.headers[header] = response.Header.Get(header)
	}
	return got
}

// startReferenceHTTP serves the reference over go-ethereum's HTTP stack, as
// its node mounts it.
func startReferenceHTTP(t *testing.T, config HTTPConfig) string {
	t.Helper()
	handler := node.NewHTTPHandlerStack(newReference(t), config.CorsAllowedOrigins, config.VirtualHosts, nil)
	server := &http.Server{Handler: &pathHandler{prefix: config.PathPrefix, next: handler}}
	if config.Timeouts != (Timeouts{}) {
		server.ReadTimeout, server.ReadHeaderTimeout = config.Timeouts.Read, config.Timeouts.ReadHeader
		server.WriteTimeout, server.IdleTimeout = config.Timeouts.Write, config.Timeouts.Idle
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	return "http://" + listener.Addr().String()
}

func startModuleHTTP(t *testing.T, config HTTPConfig) string {
	t.Helper()
	dispatcher := newTestDispatcher(t)
	config.Endpoint = "127.0.0.1:0"
	server, err := StartHTTP(dispatcher, config, NewFront(dispatcher, nil, "http.api"), prometheus.NewRegistry())
	require.NoError(t, err)
	t.Cleanup(server.Stop)
	return "http://" + server.Address().String()
}
