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
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
)

// The limits and timings of WebSocket connections, go-ethereum's.
const (
	wsReadBuffer       = 1024
	wsWriteBuffer      = 1024
	wsMessageLimit     = 32 * 1024 * 1024
	wsPingInterval     = 30 * time.Second
	wsPingWriteTimeout = 5 * time.Second
	wsPongTimeout      = 30 * time.Second
	// wsCloseReasonLimit is the longest reason a close frame carries.
	wsCloseReasonLimit = 123
)

// WSConfig is the configuration of the WebSocket endpoint, as the node holds
// it.
type WSConfig struct {
	// Endpoint is the host and port to listen on, which may be the HTTP
	// endpoint's.
	Endpoint string
	// PathPrefix is the path the connections are accepted on, the root if
	// empty.
	PathPrefix string
	// Origins are the origins a browser may connect from; without any,
	// localhost's.
	Origins []string
	Batch   BatchLimits
}

// WebSocket serves JSON-RPC to WebSocket connections: the features of IPC,
// over the namespaces its front enables.
type WebSocket struct {
	dispatcher  *dispatch.Dispatcher
	front       *Front
	prefix      string
	limits      BatchLimits
	upgrader    websocket.Upgrader
	connections prometheus.Gauge

	mutex sync.Mutex
	open  map[*wsStream]struct{}
}

// NewWebSocket creates the WebSocket handler of an endpoint.
func NewWebSocket(dispatcher *dispatch.Dispatcher, config WSConfig, front *Front, registerer prometheus.Registerer) (*WebSocket, error) {
	if err := ValidatePrefix("WebSocket", config.PathPrefix); err != nil {
		return nil, err
	}
	return &WebSocket{
		dispatcher: dispatcher,
		front:      front,
		prefix:     config.PathPrefix,
		limits:     config.Batch,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  wsReadBuffer,
			WriteBufferSize: wsWriteBuffer,
			CheckOrigin:     originChecker(config.Origins),
		},
		connections: connectionsGauge(registerer).WithLabelValues("ws"),
		open:        map[*wsStream]struct{}{},
	}, nil
}

// originChecker allows a connection without an Origin header, as only
// browsers send one, and one from an allowed origin, as go-ethereum does.
func originChecker(allowed []string) func(*http.Request) bool {
	var origins []string
	allowAll := false
	for _, origin := range allowed {
		allowAll = allowAll || origin == "*"
		if origin != "" {
			origins = append(origins, origin)
		}
	}
	if len(origins) == 0 {
		origins = append(origins, "http://localhost")
		if hostname, err := os.Hostname(); err == nil {
			origins = append(origins, "http://"+hostname)
		}
	}
	return func(request *http.Request) bool {
		if _, sent := request.Header["Origin"]; !sent {
			return true
		}
		origin := strings.ToLower(request.Header.Get("Origin"))
		if allowAll || originIsAllowed(origins, origin) {
			return true
		}
		log.Warn("Rejected WebSocket connection", "origin", origin)
		return false
	}
}

// originIsAllowed matches an origin against the allowed ones, each of which
// may leave out the scheme, the host or the port, as go-ethereum matches
// them.
func originIsAllowed(allowed []string, origin string) bool {
	return slices.ContainsFunc(allowed, func(rule string) bool { return ruleAllowsOrigin(rule, origin) })
}

func ruleAllowsOrigin(rule, origin string) bool {
	ruleScheme, ruleHostname, rulePort, err := parseOrigin(rule)
	if err != nil {
		log.Warn("Error parsing allowed origin specification", "spec", rule, "error", err)
		return false
	}
	scheme, hostname, port, err := parseOrigin(origin)
	if err != nil {
		log.Warn("Error parsing browser 'Origin' field", "Origin", origin, "error", err)
		return false
	}
	return (ruleScheme == "" || ruleScheme == scheme) &&
		(ruleHostname == "" || ruleHostname == hostname) &&
		(rulePort == "" || rulePort == port)
}

// parseOrigin splits an origin as go-ethereum does: without "://", what
// url.Parse takes for a scheme is the host and its opaque part the port.
func parseOrigin(origin string) (scheme, hostname, port string, err error) {
	parsed, err := url.Parse(strings.ToLower(origin))
	if err != nil {
		return "", "", "", err
	}
	if strings.Contains(origin, "://") {
		return parsed.Scheme, parsed.Hostname(), parsed.Port(), nil
	}
	hostname, port = parsed.Scheme, parsed.Opaque
	if hostname == "" {
		hostname = origin
	}
	return "", hostname, port, nil
}

// isWebSocket tells whether a request asks to upgrade to WebSocket.
func isWebSocket(request *http.Request) bool {
	return strings.EqualFold(request.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(request.Header.Get("Connection")), "upgrade")
}

// ServeHTTP upgrades a request and serves the connection in a goroutine of
// the dispatcher, after the request has returned.
func (w *WebSocket) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	connection, err := w.upgrader.Upgrade(response, request, nil)
	if err != nil {
		log.Debug("WebSocket upgrade failed", "err", err)
		return
	}
	connection.SetReadLimit(wsMessageLimit)
	stream := newWSStream(connection)
	if !w.track(stream) || !w.dispatcher.Go("WebSocket connection", func(ctx context.Context) {
		defer w.untrack(stream)
		w.dispatcher.Go("WebSocket pings", stream.ping)
		serveStream(ctx, w.dispatcher, w.front, w.limits, stream)
	}) {
		w.untrack(stream)
		stream.close()
	}
}

func (w *WebSocket) track(stream *wsStream) bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if w.open == nil {
		return false
	}
	w.open[stream] = struct{}{}
	w.connections.Inc()
	return true
}

func (w *WebSocket) untrack(stream *wsStream) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if _, open := w.open[stream]; open {
		delete(w.open, stream)
		w.connections.Dec()
	}
}

// closeAll closes every connection and refuses new ones.
func (w *WebSocket) closeAll() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	for stream := range w.open {
		stream.close()
		w.connections.Dec()
	}
	w.open = nil
}

// wsStream is a WebSocket connection: one JSON message per frame, written
// one at a time, kept alive by pings while it is idle.
type wsStream struct {
	connection *websocket.Conn
	mutex      sync.Mutex
	wrote      chan struct{}
	pong       chan struct{}
	closed     chan struct{}
	closeOnce  sync.Once
}

func newWSStream(connection *websocket.Conn) *wsStream {
	stream := &wsStream{
		connection: connection,
		wrote:      make(chan struct{}, 1),
		pong:       make(chan struct{}),
		closed:     make(chan struct{}),
	}
	connection.SetPongHandler(func(string) error {
		select {
		case stream.pong <- struct{}{}:
		case <-stream.closed:
		}
		return nil
	})
	return stream
}

// read answers the JSON value a frame begins with, as go-ethereum reads it.
func (s *wsStream) read() (json.RawMessage, error) {
	_, data, err := s.connection.ReadMessage()
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	err = json.NewDecoder(bytes.NewReader(data)).Decode(&raw)
	if syntaxErr, isSyntax := err.(*json.SyntaxError); isSyntax {
		s.write(errorMessage(invalidMessage(syntaxErr.Error())), true)
	}
	return raw, err
}

func (s *wsStream) write(value any, _ bool) {
	_ = s.send(value)
}

func (s *wsStream) send(message any) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_ = s.connection.SetWriteDeadline(time.Now().Add(streamWriteTimeout))
	err := s.connection.WriteJSON(message)
	if err == nil {
		select {
		case s.wrote <- struct{}{}:
		default:
		}
	}
	return err
}

// end tells the client the reason in a close frame, and closes the
// connection.
func (s *wsStream) end(reason error) {
	text := reason.Error()
	if len(text) > wsCloseReasonLimit {
		text = text[:wsCloseReasonLimit]
	}
	s.mutex.Lock()
	_ = s.connection.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, text),
		time.Now().Add(streamWriteTimeout))
	s.mutex.Unlock()
	s.close()
}

func (s *wsStream) close() {
	s.closeOnce.Do(func() {
		close(s.closed)
		_ = s.connection.Close()
	})
}

// ping pings the client when the connection has been idle for the ping
// interval, and closes it when no pong comes within the pong timeout, by
// the deadline of the next read.
func (s *wsStream) ping(context.Context) {
	timer := time.NewTimer(wsPingInterval)
	defer timer.Stop()
	for {
		select {
		case <-s.closed:
			return
		case <-s.wrote:
			timer.Reset(wsPingInterval)
		case <-timer.C:
			s.mutex.Lock()
			_ = s.connection.SetWriteDeadline(time.Now().Add(wsPingWriteTimeout))
			_ = s.connection.WriteMessage(websocket.PingMessage, nil)
			_ = s.connection.SetReadDeadline(time.Now().Add(wsPongTimeout))
			s.mutex.Unlock()
			timer.Reset(wsPingInterval)
		case <-s.pong:
			_ = s.connection.SetReadDeadline(time.Time{})
		}
	}
}
