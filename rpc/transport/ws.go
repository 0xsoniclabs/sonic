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

var wsBufferPool = new(sync.Pool)

// wsHandler upgrades requests to WebSocket connections and serves them.
type wsHandler struct {
	d        *dispatch.Dispatcher
	frames   frames
	upgrader websocket.Upgrader
}

func newWSHandler(d *dispatch.Dispatcher, f frames, origins []string) *wsHandler {
	return &wsHandler{
		d:      d,
		frames: f,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			WriteBufferPool: wsBufferPool,
			CheckOrigin:     wsHandshakeValidator(origins),
		},
	}
}

func (h *wsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Debug("WebSocket upgrade failed", "err", err)
		return
	}
	conn.SetReadLimit(wsReadLimit)
	c := &wsConn{conn: conn, pingReset: make(chan struct{}, 1), done: make(chan struct{})}
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Time{})
		return nil
	})
	served := h.d.Go("ws connection", func(ctx context.Context) {
		defer close(c.done)
		h.d.Go("ws ping", func(ctx context.Context) { c.pingLoop(ctx) })
		serveStream(ctx, h.d, h.frames, c)
	})
	if !served {
		_ = conn.Close()
	}
}

// wsConn is one WebSocket connection. It pings a connection that wrote
// nothing for a while and closes it when the pong does not come.
type wsConn struct {
	conn      *websocket.Conn
	writeMu   sync.Mutex
	pingReset chan struct{}
	closeOnce sync.Once
	done      chan struct{}
}

func (c *wsConn) read() (json.RawMessage, error) {
	var raw json.RawMessage
	err := c.conn.ReadJSON(&raw)
	return raw, err
}

func (c *wsConn) send(msg []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		return err
	}
	select {
	case c.pingReset <- struct{}{}:
	default:
	}
	return nil
}

func (c *wsConn) close(reason string) error {
	var err error
	c.closeOnce.Do(func() {
		if reason != "" {
			if len(reason) > 120 {
				reason = reason[:120]
			}
			c.writeMu.Lock()
			_ = c.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, reason), time.Now().Add(wsPingWriteTimeout))
			c.writeMu.Unlock()
		}
		err = c.conn.Close()
	})
	return err
}

func (c *wsConn) pingLoop(ctx context.Context) {
	timer := time.NewTimer(wsPingInterval)
	defer timer.Stop()
	for {
		select {
		case <-c.done:
			return
		case <-ctx.Done():
			return
		case <-c.pingReset:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(wsPingInterval)
		case <-timer.C:
			c.writeMu.Lock()
			_ = c.conn.SetWriteDeadline(time.Now().Add(wsPingWriteTimeout))
			_ = c.conn.WriteMessage(websocket.PingMessage, nil)
			_ = c.conn.SetReadDeadline(time.Now().Add(wsPongTimeout))
			c.writeMu.Unlock()
			timer.Reset(wsPingInterval)
		}
	}
}

// wsHandshakeValidator accepts a request without an Origin header, and one
// whose origin a rule allows; with no rules, localhost and the host's name.
func wsHandshakeValidator(allowedOrigins []string) func(*http.Request) bool {
	origins := make(map[string]struct{})
	allowAll := false
	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAll = true
		}
		if origin != "" {
			origins[origin] = struct{}{}
		}
	}
	if len(origins) == 0 {
		origins["http://localhost"] = struct{}{}
		if hostname, err := os.Hostname(); err == nil {
			origins["http://"+hostname] = struct{}{}
		}
	}
	return func(r *http.Request) bool {
		if _, ok := r.Header["Origin"]; !ok {
			return true
		}
		origin := strings.ToLower(r.Header.Get("Origin"))
		if allowAll {
			return true
		}
		for allowed := range origins {
			if ruleAllowsOrigin(allowed, origin) {
				return true
			}
		}
		log.Warn("Rejected WebSocket connection", "origin", origin)
		return false
	}
}

func ruleAllowsOrigin(allowedOrigin, browserOrigin string) bool {
	allowedScheme, allowedHostname, allowedPort, err := parseOriginURL(allowedOrigin)
	if err != nil {
		log.Warn("Error parsing allowed origin specification", "spec", allowedOrigin, "error", err)
		return false
	}
	browserScheme, browserHostname, browserPort, err := parseOriginURL(browserOrigin)
	if err != nil {
		log.Warn("Error parsing browser 'Origin' field", "Origin", browserOrigin, "error", err)
		return false
	}
	if allowedScheme != "" && allowedScheme != browserScheme {
		return false
	}
	if allowedHostname != "" && allowedHostname != browserHostname {
		return false
	}
	if allowedPort != "" && allowedPort != browserPort {
		return false
	}
	return true
}

func parseOriginURL(origin string) (scheme, hostname, port string, err error) {
	parsed, err := url.Parse(strings.ToLower(origin))
	if err != nil {
		return "", "", "", err
	}
	if strings.Contains(origin, "://") {
		return parsed.Scheme, parsed.Hostname(), parsed.Port(), nil
	}
	hostname = parsed.Scheme
	port = parsed.Opaque
	if hostname == "" {
		hostname = origin
	}
	return "", hostname, port, nil
}
