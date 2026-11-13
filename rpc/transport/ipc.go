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
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
)

// streamWriteTimeout bounds the write of one answer to a stream connection.
const streamWriteTimeout = 10 * time.Second

// IPCServer serves JSON-RPC over a Unix socket: newline-delimited JSON, every
// namespace of the module, and the node's administration and profiling.
type IPCServer struct {
	dispatcher  *dispatch.Dispatcher
	front       *Front
	listener    net.Listener
	connections *prometheus.GaugeVec

	mutex sync.Mutex
	open  map[net.Conn]struct{}
}

// StartIPC listens on the socket path, readable and writable by the node's
// user only, and serves the front's calls in goroutines of the dispatcher.
func StartIPC(dispatcher *dispatch.Dispatcher, path string, front *Front, registerer prometheus.Registerer) (*IPCServer, error) {
	listener, err := listenUnix(path)
	if err != nil {
		return nil, err
	}
	server := &IPCServer{
		dispatcher:  dispatcher,
		front:       front,
		listener:    listener,
		connections: connectionsGauge(registerer),
		open:        map[net.Conn]struct{}{},
	}
	if !dispatcher.Go("IPC listener", func(ctx context.Context) { server.accept(ctx) }) {
		_ = listener.Close()
		return nil, errors.New("the RPC module is stopping")
	}
	log.Info("IPC endpoint opened", "url", path)
	return server, nil
}

// listenUnix creates the socket, removing one a previous run left.
func listenUnix(path string) (net.Listener, error) {
	if limit := len(syscall.RawSockaddrUnix{}.Path); len(path)+1 > limit {
		log.Warn(fmt.Sprintf("The ipc endpoint is longer than %d characters. ", limit-1), "endpoint", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o751); err != nil {
		return nil, err
	}
	_ = os.Remove(path)
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	return listener, os.Chmod(path, 0o600)
}

func (s *IPCServer) accept(ctx context.Context) {
	for {
		connection, err := s.listener.Accept()
		if err != nil {
			return
		}
		if !s.track(connection) || !s.dispatcher.Go("IPC connection", func(ctx context.Context) { s.serve(ctx, connection) }) {
			s.untrack(connection)
			_ = connection.Close()
		}
	}
}

// serve reads the messages of a connection until it fails, and serves each in
// a goroutine of its own, as their answers may come in any order. Closing a
// connection cancels its calls first, then waits for them.
func (s *IPCServer) serve(ctx context.Context, connection net.Conn) {
	defer s.untrack(connection)
	ctx, cancel := context.WithCancel(ctx)
	var calls sync.WaitGroup
	defer func() {
		cancel()
		calls.Wait()
		_ = connection.Close()
	}()
	out := &streamWriter{connection: connection}
	served := &session{front: s.front, out: out}
	decoder := json.NewDecoder(connection)
	decoder.UseNumber()
	for {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			var syntaxErr *json.SyntaxError
			if errors.As(err, &syntaxErr) {
				out.write(errorMessage(invalidMessage(err.Error())), true)
			}
			return
		}
		messages, batch := parseMessages(raw)
		calls.Add(1)
		started := s.dispatcher.Go("IPC call", func(context.Context) {
			defer calls.Done()
			if batch {
				served.serveBatch(ctx, messages)
			} else {
				served.serveMessage(ctx, messages[0])
			}
		})
		if !started {
			calls.Done()
			return
		}
	}
}

func (s *IPCServer) track(connection net.Conn) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.open == nil {
		return false
	}
	s.open[connection] = struct{}{}
	s.connections.WithLabelValues("ipc").Inc()
	return true
}

func (s *IPCServer) untrack(connection net.Conn) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if _, open := s.open[connection]; open {
		delete(s.open, connection)
		s.connections.WithLabelValues("ipc").Dec()
	}
}

// Stop closes the listener and every connection, which ends their calls.
func (s *IPCServer) Stop() {
	_ = s.listener.Close()
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for connection := range s.open {
		_ = connection.Close()
		s.connections.WithLabelValues("ipc").Dec()
	}
	s.open = nil
	log.Info("IPC endpoint closed", "url", s.listener.Addr())
}

// streamWriter writes the answers of a stream connection, one at a time.
type streamWriter struct {
	mutex      sync.Mutex
	connection net.Conn
}

func (w *streamWriter) write(value any, isError bool) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	_ = w.connection.SetWriteDeadline(time.Now().Add(streamWriteTimeout))
	_ = json.NewEncoder(w.connection).Encode(value)
}

// connectionsGauge counts the open connections by transport.
func connectionsGauge(registerer prometheus.Registerer) *prometheus.GaugeVec {
	return dispatch.Register(registerer, prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "rpc_connections",
		Help: "Open connections, by transport.",
	}, []string{"transport"}))
}
