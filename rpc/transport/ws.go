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
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/websocket"
)

const (
	wsReadBuffer       = 1024
	wsWriteBuffer      = 1024
	wsPingInterval     = 30 * time.Second
	wsPingWriteTimeout = 5 * time.Second
	wsPongTimeout      = 30 * time.Second
	// DefaultWSReadLimit is the largest accepted WebSocket message.
	DefaultWSReadLimit = 32 * 1024 * 1024
)

var wsBufferPool = new(sync.Pool)

// WSConfig configures the WebSocket transport.
type WSConfig struct {
	StreamConfig
	// Origins are the allowed origins of browsers; "*" allows all.
	Origins []string
	// ReadLimit is the largest accepted message, DefaultWSReadLimit if 0.
	ReadLimit int64
}

// WSHandler serves JSON-RPC over WebSocket connections.
type WSHandler struct {
	cfg      WSConfig
	server   *streamServer
	upgrader websocket.Upgrader

	mu      sync.Mutex
	closing bool
	conns   map[*wsCodec]struct{}
	wg      sync.WaitGroup
}

// NewWSHandler creates the handler of WebSocket upgrade requests.
func NewWSHandler(cfg WSConfig, front *Front) *WSHandler {
	cfg.Name = "ws"
	if cfg.ReadLimit <= 0 {
		cfg.ReadLimit = DefaultWSReadLimit
	}
	return &WSHandler{
		cfg:    cfg,
		server: newStreamServer(cfg.StreamConfig, front),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  wsReadBuffer,
			WriteBufferSize: wsWriteBuffer,
			WriteBufferPool: wsBufferPool,
			CheckOrigin:     originChecker(cfg.Origins),
		},
		conns: map[*wsCodec]struct{}{},
	}
}

// ServeHTTP upgrades the request and serves the connection until it ends.
func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	if h.closing {
		h.mu.Unlock()
		http.Error(w, "server is shutting down", http.StatusServiceUnavailable)
		return
	}
	h.wg.Add(1)
	h.mu.Unlock()
	defer h.wg.Done()

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Debug("WebSocket upgrade failed", "err", err)
		return
	}
	conn.SetReadLimit(h.cfg.ReadLimit)
	codec := newWSCodec(conn)

	h.mu.Lock()
	if h.closing {
		h.mu.Unlock()
		_ = conn.Close()
		return
	}
	h.conns[codec] = struct{}{}
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.conns, codec)
		h.mu.Unlock()
	}()

	h.server.serve(h.server.d.Context(), codec)
}

// Close refuses new connections, closes the open ones and waits for them.
func (h *WSHandler) Close() {
	h.mu.Lock()
	h.closing = true
	conns := make([]*wsCodec, 0, len(h.conns))
	for c := range h.conns {
		conns = append(conns, c)
	}
	h.mu.Unlock()
	for _, c := range conns {
		_ = c.close()
	}
	h.wg.Wait()
}

// wsCodec carries messages over a WebSocket connection.
type wsCodec struct {
	conn *websocket.Conn
	mu   sync.Mutex // serializes writes, pings included
}

func newWSCodec(conn *websocket.Conn) *wsCodec {
	c := &wsCodec{conn: conn}
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Time{})
	})
	return c
}

func (c *wsCodec) readMessage() (json.RawMessage, error) {
	_, data, err := c.conn.ReadMessage()
	if err != nil {
		var closeErr *websocket.CloseError
		if errors.As(err, &closeErr) || errors.Is(err, net.ErrClosed) {
			return nil, io.EOF
		}
		return nil, err
	}
	if !json.Valid(data) {
		return nil, errors.New("invalid JSON")
	}
	return data, nil
}

func (c *wsCodec) writeMessage(data []byte, deadline time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.SetWriteDeadline(deadline)
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *wsCodec) close() error { return c.conn.Close() }

func (c *wsCodec) remote() string { return c.conn.RemoteAddr().String() }

func (c *wsCodec) idleInterval() time.Duration { return wsPingInterval }

// ping probes an idle connection: the pong must arrive within the timeout or
// the read fails.
func (c *wsCodec) ping() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(wsPingWriteTimeout))
	if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		return err
	}
	return c.conn.SetReadDeadline(time.Now().Add(wsPongTimeout))
}

var _ pinger = (*wsCodec)(nil)
