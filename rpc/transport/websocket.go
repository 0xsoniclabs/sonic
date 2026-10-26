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
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/websocket"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// The constants of go-ethereum's WebSocket server.
const (
	websocketReadBuffer       = 1024
	websocketWriteBuffer      = 1024
	websocketPingInterval     = 30 * time.Second
	websocketPingWriteTimeout = 5 * time.Second
	websocketPongTimeout      = 30 * time.Second
)

var websocketBufferPool = new(sync.Pool)

// WebSocketConfig is the WebSocket endpoint as the node's configuration
// holds it.
type WebSocketConfig struct {
	PathPrefix string
	Origins    []string
	Batch      BatchLimits
}

// websocketEndpoint upgrades requests to WebSocket connections and serves
// them until they end or the server stops.
type websocketEndpoint struct {
	router     Router
	config     WebSocketConfig
	dispatcher *dispatch.Dispatcher
	upgrader   websocket.Upgrader
	served     *connections
}

func newWebSocketEndpoint(router Router, config WebSocketConfig, dispatcher *dispatch.Dispatcher, served *connections) *websocketEndpoint {
	return &websocketEndpoint{
		router:     router,
		config:     config,
		dispatcher: dispatcher,
		served:     served,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  websocketReadBuffer,
			WriteBufferSize: websocketWriteBuffer,
			WriteBufferPool: websocketBufferPool,
			CheckOrigin:     originChecker(config.Origins),
		},
	}
}

func (endpoint *websocketEndpoint) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	ctx, done, open := endpoint.served.add()
	if !open {
		http.Error(response, "server is stopping", http.StatusServiceUnavailable)
		return
	}
	defer done()
	connection, err := endpoint.upgrader.Upgrade(response, request, nil)
	if err != nil {
		log.Debug("WebSocket upgrade failed", "err", err)
		return
	}
	codec := newWebSocketCodec(connection)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if !endpoint.dispatcher.Go(ctx, "websocket ping", codec.ping) {
		codec.close(dispatch.ErrStopping)
		return
	}
	newStream(endpoint.router, WebSocket, endpoint.config.Batch, codec, endpoint.dispatcher).serve(ctx)
}

// websocketCodec reads and writes JSON messages of a WebSocket connection
// and keeps it alive with pings, as go-ethereum's codec does.
type websocketCodec struct {
	connection *websocket.Conn
	writing    sync.Mutex
	written    chan struct{}
	pong       chan struct{}
	closed     chan struct{}
	closeOnce  sync.Once
}

func newWebSocketCodec(connection *websocket.Conn) *websocketCodec {
	connection.SetReadLimit(websocketMessageLimit)
	codec := &websocketCodec{
		connection: connection,
		written:    make(chan struct{}, 1),
		pong:       make(chan struct{}),
		closed:     make(chan struct{}),
	}
	connection.SetPongHandler(func(string) error {
		select {
		case codec.pong <- struct{}{}:
		case <-codec.closed:
		}
		return nil
	})
	return codec
}

func (codec *websocketCodec) read() (json.RawMessage, error) {
	var raw json.RawMessage
	if err := codec.connection.ReadJSON(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (codec *websocketCodec) write(value any, deadline time.Time) error {
	codec.writing.Lock()
	defer codec.writing.Unlock()
	_ = codec.connection.SetWriteDeadline(deadline)
	if err := codec.connection.WriteJSON(value); err != nil {
		return err
	}
	select {
	case codec.written <- struct{}{}:
	default:
	}
	return nil
}

// close ends the connection with a close frame that carries the reason, if
// any.
func (codec *websocketCodec) close(reason error) {
	codec.closeOnce.Do(func() {
		close(codec.closed)
		// A control message may be written while a message is being written.
		if reason != nil {
			frame := websocket.FormatCloseMessage(websocket.CloseInternalServerErr, truncateReason(reason.Error()))
			_ = codec.connection.WriteControl(websocket.CloseMessage, frame, time.Now().Add(websocketPingWriteTimeout))
		}
		_ = codec.connection.Close()
	})
}

// truncateReason fits a reason into a close frame, whose payload holds 123
// bytes of text.
func truncateReason(reason string) string {
	if len(reason) > 123 {
		return reason[:123]
	}
	return reason
}

// ping sends a ping when nothing was written for a ping interval and ends a
// connection whose client does not answer it in time.
func (codec *websocketCodec) ping(ctx context.Context) {
	timer := time.NewTimer(websocketPingInterval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-codec.closed:
			return
		case <-codec.written:
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(websocketPingInterval)
		case <-timer.C:
			codec.writing.Lock()
			_ = codec.connection.SetWriteDeadline(time.Now().Add(websocketPingWriteTimeout))
			_ = codec.connection.WriteMessage(websocket.PingMessage, nil)
			_ = codec.connection.SetReadDeadline(time.Now().Add(websocketPongTimeout))
			codec.writing.Unlock()
			timer.Reset(websocketPingInterval)
		case <-codec.pong:
			_ = codec.connection.SetReadDeadline(time.Time{})
		}
	}
}

// originChecker allows a request without an Origin header, as browsers
// always send one, and otherwise one of the allowed origins; without any,
// localhost and the host's name, as go-ethereum's server does.
func originChecker(allowed []string) func(*http.Request) bool {
	origins := map[string]bool{}
	allowAll := false
	for _, origin := range allowed {
		if origin == "*" {
			allowAll = true
		}
		if origin != "" {
			origins[origin] = true
		}
	}
	if len(origins) == 0 {
		origins["http://localhost"] = true
		if hostname, err := os.Hostname(); err == nil {
			origins["http://"+hostname] = true
		}
	}
	return func(request *http.Request) bool {
		if _, sent := request.Header["Origin"]; !sent {
			return true
		}
		origin := strings.ToLower(request.Header.Get("Origin"))
		if allowAll {
			return true
		}
		for allowedOrigin := range origins {
			if ruleAllowsOrigin(allowedOrigin, origin) {
				return true
			}
		}
		log.Warn("Rejected WebSocket connection", "origin", origin)
		return false
	}
}

func ruleAllowsOrigin(allowedOrigin, browserOrigin string) bool {
	allowedScheme, allowedHostname, allowedPort, err := parseOrigin(allowedOrigin)
	if err != nil {
		log.Warn("Error parsing allowed origin specification", "spec", allowedOrigin, "error", err)
		return false
	}
	browserScheme, browserHostname, browserPort, err := parseOrigin(browserOrigin)
	if err != nil {
		log.Warn("Error parsing browser 'Origin' field", "Origin", browserOrigin, "error", err)
		return false
	}
	return (allowedScheme == "" || allowedScheme == browserScheme) &&
		(allowedHostname == "" || allowedHostname == browserHostname) &&
		(allowedPort == "" || allowedPort == browserPort)
}

func parseOrigin(origin string) (scheme, hostname, port string, err error) {
	parsed, err := url.Parse(strings.ToLower(origin))
	if err != nil {
		return "", "", "", err
	}
	if strings.Contains(origin, "://") {
		return parsed.Scheme, parsed.Hostname(), parsed.Port(), nil
	}
	hostname = parsed.Scheme
	if hostname == "" {
		hostname = origin
	}
	return "", hostname, parsed.Opaque, nil
}
