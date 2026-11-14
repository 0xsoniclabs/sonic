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

// serve serves a connection until it ends.
func (s *IPCServer) serve(ctx context.Context, connection net.Conn) {
	defer s.untrack(connection)
	decoder := json.NewDecoder(connection)
	decoder.UseNumber()
	serveStream(ctx, s.dispatcher, s.front, BatchLimits{}, &ipcStream{connection: connection, decoder: decoder})
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

// ipcStream is an IPC connection: newline-delimited JSON, written one
// message at a time.
type ipcStream struct {
	mutex      sync.Mutex
	connection net.Conn
	decoder    *json.Decoder
}

func (s *ipcStream) read() (json.RawMessage, error) {
	var raw json.RawMessage
	err := s.decoder.Decode(&raw)
	if syntaxErr, isSyntax := errors.AsType[*json.SyntaxError](err); isSyntax {
		s.write(errorMessage(invalidMessage(syntaxErr.Error())), true)
	}
	return raw, err
}

func (s *ipcStream) write(value any, _ bool) {
	_ = s.send(value)
}

func (s *ipcStream) send(message any) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_ = s.connection.SetWriteDeadline(time.Now().Add(streamWriteTimeout))
	return json.NewEncoder(s.connection).Encode(message)
}

// end tells the client the reason in an error without an id, and closes the
// connection.
func (s *ipcStream) end(reason error) {
	s.write(errorMessage(&dispatch.Error{Code: dispatch.CodeDefault, Message: reason.Error()}), true)
	s.close()
}

func (s *ipcStream) close() {
	_ = s.connection.Close()
}

// connectionsGauge counts the open connections by transport.
func connectionsGauge(registerer prometheus.Registerer) *prometheus.GaugeVec {
	return dispatch.Register(registerer, prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "rpc_connections",
		Help: "Open connections, by transport.",
	}, []string{"transport"}))
}
