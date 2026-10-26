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

	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// IPCServer serves JSON-RPC on a unix socket, as go-ethereum's node does:
// newline-delimited JSON, the socket readable by its owner only (D5).
type IPCServer struct {
	path       string
	router     Router
	limits     BatchLimits
	dispatcher *dispatch.Dispatcher
	served     *connections

	mutex    sync.Mutex
	listener net.Listener
}

// NewIPCServer returns a server for the socket at the path.
func NewIPCServer(path string, router Router, limits BatchLimits, dispatcher *dispatch.Dispatcher) *IPCServer {
	return &IPCServer{path: path, router: router, limits: limits, dispatcher: dispatcher, served: newConnections()}
}

// Path is the path of the socket.
func (server *IPCServer) Path() string {
	return server.path
}

// Start listens on the socket and serves its connections.
func (server *IPCServer) Start() error {
	listener, err := listenIPC(server.path)
	if err != nil {
		return err
	}
	server.mutex.Lock()
	server.listener = listener
	server.mutex.Unlock()
	if !server.dispatcher.Run("ipc server", func(context.Context) { server.accept(listener) }) {
		_ = listener.Close()
		return dispatch.ErrStopping
	}
	log.Info("IPC endpoint opened", "url", server.path)
	return nil
}

// listenIPC listens on a unix socket, removing a leftover one, and makes it
// readable by its owner only, as go-ethereum's node does.
func listenIPC(path string) (net.Listener, error) {
	if maximum := len(syscall.RawSockaddrUnix{}.Path); len(path)+1 > maximum {
		log.Warn(fmt.Sprintf("The ipc endpoint is longer than %d characters. ", maximum-1), "endpoint", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0751); err != nil {
		return nil, err
	}
	_ = os.Remove(path)
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0600); err != nil {
		_ = listener.Close()
		return nil, err
	}
	return listener, nil
}

// accept serves every connection until the listener closes.
func (server *IPCServer) accept(listener net.Listener) {
	for {
		connection, err := listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			log.Warn("IPC accept failed", "err", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		ctx, done, open := server.served.add()
		if !open {
			_ = connection.Close()
			return
		}
		codec := newIPCCodec(connection)
		started := server.dispatcher.Go(ctx, "ipc connection", func(ctx context.Context) {
			defer done()
			newStream(server.router, IPC, server.limits, codec, server.dispatcher).serve(ctx)
		})
		if !started {
			done()
			codec.close(dispatch.ErrStopping)
		}
	}
}

// Stop closes the socket and the connections and waits for them.
func (server *IPCServer) Stop() {
	server.mutex.Lock()
	listener := server.listener
	server.listener = nil
	server.mutex.Unlock()
	if listener == nil {
		return
	}
	_ = listener.Close()
	server.served.stop()
	log.Info("IPC endpoint closed", "url", server.path)
}

// ipcCodec reads and writes newline-delimited JSON on a connection.
type ipcCodec struct {
	connection net.Conn
	decoder    *json.Decoder
	encoder    *json.Encoder
	closeOnce  sync.Once
}

func newIPCCodec(connection net.Conn) *ipcCodec {
	return &ipcCodec{connection: connection, decoder: json.NewDecoder(connection), encoder: json.NewEncoder(connection)}
}

func (codec *ipcCodec) read() (json.RawMessage, error) {
	var raw json.RawMessage
	if err := codec.decoder.Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (codec *ipcCodec) write(value any, deadline time.Time) error {
	_ = codec.connection.SetWriteDeadline(deadline)
	return codec.encoder.Encode(value)
}

// close ends the connection; a unix socket has no way to tell the reason.
func (codec *ipcCodec) close(error) {
	codec.closeOnce.Do(func() { _ = codec.connection.Close() })
}
