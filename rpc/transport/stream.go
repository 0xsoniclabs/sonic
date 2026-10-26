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
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// streamConn is a connection that carries many frames and notifications:
// WebSocket or IPC.
type streamConn interface {
	// read reads the next frame.
	read() (json.RawMessage, error)
	// send writes one message, within the write timeout.
	send(msg []byte) error
	// close closes the connection, telling the client the reason if the
	// transport can; it may be called more than once.
	close(reason string) error
}

// stream serves one stream connection. Each frame is served on a goroutine of
// its own, as in go-ethereum; answers and notifications are written one at a
// time, and a failed write closes the connection.
type stream struct {
	frames frames
	conn   streamConn
	state  *dispatch.Stream

	mu sync.Mutex
}

// serveStream serves a connection until it closes or the dispatcher stops,
// then ends its subscriptions: their producers are cancelled first and then
// waited for.
func serveStream(ctx context.Context, d *dispatch.Dispatcher, f frames, conn streamConn) {
	connections := d.Metrics().Connections(string(f.kind))
	connections.Inc()
	defer connections.Dec()

	s := &stream{frames: f, conn: conn}
	s.state = d.NewStream(ctx, dispatch.StreamLimits{MaxSubscriptions: maxSubscriptions, Buffer: subscriptionBuffer}, s.send, s.drop)
	stop := context.AfterFunc(s.state.Context(), func() { _ = conn.close("server stopped") })
	defer func() {
		stop()
		_ = conn.close("")
		s.state.Close()
	}()

	for {
		raw, err := conn.read()
		if err != nil {
			var syntaxErr *json.SyntaxError
			if errors.As(err, &syntaxErr) {
				_ = s.send((&response{id: nullID, err: &parseErr{err.Error()}}).encode())
			}
			return
		}
		ok := s.state.Go("rpc frame", func(ctx context.Context) {
			s.frames.serveFrame(ctx, raw, s, 0, s.state.Request)
		})
		if !ok {
			return
		}
	}
}

// write implements writer for the frames of the connection.
func (s *stream) write(msg []byte, _ bool) {
	_ = s.send(msg)
}

// send writes one message with its newline; a failed write closes the
// connection: its client does not read.
func (s *stream) send(msg []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.conn.send(append(msg, '\n')); err != nil {
		_ = s.conn.close("")
		return err
	}
	return nil
}

// drop closes the connection with a reason: a subscription of it lost
// events, or its client fell behind.
func (s *stream) drop(reason string) {
	log.Info("Closing RPC connection", "transport", s.frames.kind, "reason", reason)
	_ = s.conn.close(reason)
}
