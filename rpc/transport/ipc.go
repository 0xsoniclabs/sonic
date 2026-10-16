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
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// IPCServer serves JSON-RPC over a unix socket, as newline-delimited JSON. The
// socket is readable by its owner only.
type IPCServer struct {
	path   string
	server *streamServer

	mu       sync.Mutex
	listener net.Listener
	conns    map[*ipcCodec]struct{}
	closing  bool
	wg       sync.WaitGroup
}

// NewIPCServer creates the IPC transport on the socket path.
func NewIPCServer(path string, cfg StreamConfig, front *Front) *IPCServer {
	cfg.Name = "ipc"
	cfg.Local = true
	return &IPCServer{path: path, server: newStreamServer(cfg, front), conns: map[*ipcCodec]struct{}{}}
}

// Start creates the socket and accepts connections in the background.
func (s *IPCServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		return nil
	}
	l, err := ipcListen(s.path)
	if err != nil {
		return err
	}
	if !s.server.d.Go("ipc-accept", func(context.Context) { s.acceptLoop(l) }) {
		_ = l.Close()
		return errors.New("dispatcher is stopping")
	}
	s.listener = l
	log.Info("IPC endpoint opened", "url", s.path)
	return nil
}

// ipcListen creates a unix socket on the endpoint, replacing a leftover.
func ipcListen(endpoint string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(endpoint), 0751); err != nil {
		return nil, err
	}
	_ = os.Remove(endpoint)
	l, err := net.Listen("unix", endpoint)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(endpoint, 0600); err != nil {
		_ = l.Close()
		return nil, fmt.Errorf("failed to restrict the IPC socket: %w", err)
	}
	return l, nil
}

func (s *IPCServer) acceptLoop(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				log.Warn("RPC accept error", "err", err)
				continue
			}
			return
		}
		codec := newIPCCodec(conn)
		s.mu.Lock()
		if s.closing {
			s.mu.Unlock()
			_ = conn.Close()
			return
		}
		s.conns[codec] = struct{}{}
		s.wg.Add(1)
		s.mu.Unlock()
		if !s.server.d.Go("ipc-conn", func(ctx context.Context) {
			defer s.wg.Done()
			defer func() {
				s.mu.Lock()
				delete(s.conns, codec)
				s.mu.Unlock()
			}()
			s.server.serve(ctx, codec)
		}) {
			s.wg.Done()
			_ = conn.Close()
			return
		}
	}
}

// Stop closes the socket and the connections and waits for them.
func (s *IPCServer) Stop() error {
	s.mu.Lock()
	l := s.listener
	s.listener = nil
	s.closing = true
	conns := make([]*ipcCodec, 0, len(s.conns))
	for c := range s.conns {
		conns = append(conns, c)
	}
	s.mu.Unlock()
	var err error
	if l != nil {
		err = l.Close()
		_ = os.Remove(s.path)
		log.Info("IPC endpoint closed", "url", s.path)
	}
	for _, c := range conns {
		_ = c.close()
	}
	s.wg.Wait()
	return err
}

// ipcCodec carries newline-delimited JSON over a stream connection.
type ipcCodec struct {
	conn net.Conn
	dec  *json.Decoder
	mu   sync.Mutex
}

func newIPCCodec(conn net.Conn) *ipcCodec {
	dec := json.NewDecoder(conn)
	dec.UseNumber()
	return &ipcCodec{conn: conn, dec: dec}
}

func (c *ipcCodec) readMessage() (json.RawMessage, error) {
	var raw json.RawMessage
	if err := c.dec.Decode(&raw); err != nil {
		if errors.Is(err, net.ErrClosed) {
			return nil, io.EOF
		}
		return nil, err
	}
	return raw, nil
}

func (c *ipcCodec) writeMessage(data []byte, deadline time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.SetWriteDeadline(deadline)
	_, err := c.conn.Write(append(data, '\n'))
	return err
}

func (c *ipcCodec) close() error { return c.conn.Close() }

func (c *ipcCodec) remote() string { return c.conn.RemoteAddr().String() }
