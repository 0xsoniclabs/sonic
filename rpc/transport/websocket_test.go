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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func startModuleWebSocket(t *testing.T, config WSConfig) string {
	t.Helper()
	dispatcher := newTestDispatcher(t)
	ws, err := NewWebSocket(dispatcher, config, NewFront(dispatcher, nil, "ws.api"), prometheus.NewRegistry())
	require.NoError(t, err)
	server, err := StartWebSocket(dispatcher, config.Endpoint, ws, prometheus.NewRegistry())
	require.NoError(t, err)
	t.Cleanup(server.Stop)
	return "ws://" + server.Address().String()
}

func startReferenceWebSocket(t *testing.T, batch BatchLimits) string {
	t.Helper()
	server := newReference(t)
	server.SetBatchLimits(batch.Items, batch.ResponseBytes)
	reference := httptest.NewServer(server.WebsocketHandler([]string{"*"}))
	t.Cleanup(reference.Close)
	t.Cleanup(server.Stop)
	return "ws" + strings.TrimPrefix(reference.URL, "http")
}

// exchangeFrames writes a frame to a WebSocket and reads the frames it gets,
// in order, then how the connection goes on: open, or closed.
func exchangeFrames(t *testing.T, endpoint, request string, answers int) []string {
	t.Helper()
	connection, _, err := websocket.DefaultDialer.Dial(endpoint, nil)
	require.NoError(t, err)
	defer func() { _ = connection.Close() }()
	require.NoError(t, connection.WriteMessage(websocket.TextMessage, []byte(request)))
	var got []string
	for range answers {
		require.NoError(t, connection.SetReadDeadline(time.Now().Add(5*time.Second)))
		_, frame, err := connection.ReadMessage()
		require.NoError(t, err)
		got = append(got, ids.ReplaceAllString(strings.TrimSpace(string(frame)), `"id"`))
	}
	require.NoError(t, connection.SetReadDeadline(time.Now().Add(100*time.Millisecond)))
	_, _, err = connection.ReadMessage()
	if _, closed := err.(*websocket.CloseError); closed || !strings.Contains(err.Error(), "timeout") {
		got = append(got, "closed")
	}
	return got
}

func TestWebSocket_AnswersAsGoEthereumServer(t *testing.T) {
	batch := BatchLimits{Items: 3, ResponseBytes: 1000}
	reference := startReferenceWebSocket(t, batch)
	module := startModuleWebSocket(t, WSConfig{Endpoint: "127.0.0.1:0", Batch: batch})
	cases := map[string]struct {
		request string
		answers int
	}{
		"call":           {`{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["hi"]}`, 1},
		"batch":          {envelopeBodies["batch"], 1},
		"batch too long": {largeBatch(4), 1},
		"subscription":   {`{"jsonrpc":"2.0","id":1,"method":"test_subscribe","params":["ticks",2]}`, 3},
		"unknown kind":   {`{"jsonrpc":"2.0","id":1,"method":"test_subscribe","params":["other"]}`, 1},
		"syntax error":   {`{"jsonrpc":"2.0",]`, 1},
		"two values":     {`{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["a"]}{"jsonrpc":"2.0","id":2,"method":"test_echo","params":["b"]}`, 1},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, exchangeFrames(t, reference, test.request, test.answers), exchangeFrames(t, module, test.request, test.answers))
		})
	}
}

func TestWebSocket_ServesNoPlainRequest(t *testing.T) {
	module := startModuleWebSocket(t, WSConfig{Endpoint: "127.0.0.1:0", PathPrefix: "/ws"})
	response, err := http.Post("http"+strings.TrimPrefix(module, "ws")+"/ws", contentType, strings.NewReader(`{}`))
	require.NoError(t, err)
	_ = response.Body.Close()
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	_, response, err = websocket.DefaultDialer.Dial(module+"/other", nil)
	require.Error(t, err, "a WebSocket on another path is not served")
	require.Equal(t, http.StatusOK, response.StatusCode)
	connection, _, err := websocket.DefaultDialer.Dial(module+"/ws", nil)
	require.NoError(t, err)
	_ = connection.Close()
}

func TestWebSocket_ChecksOriginsAsGoEthereum(t *testing.T) {
	cases := []struct {
		allowed []string
		origin  string
		allow   bool
	}{
		{[]string{"*"}, "https://any.example", true},
		{nil, "http://localhost", true},
		{nil, "http://other", false},
		{[]string{"example.com"}, "https://example.com:8080", true},
		{[]string{"https://example.com"}, "http://example.com", false},
		{[]string{"example.com:8080"}, "https://example.com:8081", false},
		{[]string{"example.com:8080"}, "https://example.com:8080", true},
	}
	for _, test := range cases {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Origin", test.origin)
		require.Equal(t, test.allow, originChecker(test.allowed)(request), "%v %s", test.allowed, test.origin)
	}
	require.True(t, originChecker(nil)(httptest.NewRequest(http.MethodGet, "/", nil)), "a request without origin")
}
