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
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// listenIPC opens the unix socket at path, readable by the node's user only,
// replacing a socket a previous run left behind.
func listenIPC(path string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0751); err != nil {
		return nil, err
	}
	_ = os.Remove(path)
	l, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0600); err != nil {
		_ = l.Close()
		return nil, err
	}
	return l, nil
}

// serveIPC accepts connections until the listener closes.
func serveIPC(ctx context.Context, d *dispatch.Dispatcher, f frames, l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				log.Warn("IPC accept failed", "err", err)
			}
			return
		}
		c := &ipcConn{conn: conn, dec: json.NewDecoder(conn)}
		if !d.Go("ipc connection", func(ctx context.Context) { serveStream(ctx, d, f, c) }) {
			_ = conn.Close()
			return
		}
	}
}

// ipcConn is one connection of newline-delimited JSON.
type ipcConn struct {
	conn      net.Conn
	dec       *json.Decoder
	closeOnce sync.Once
}

func (c *ipcConn) read() (json.RawMessage, error) {
	var raw json.RawMessage
	err := c.dec.Decode(&raw)
	return raw, err
}

func (c *ipcConn) send(msg []byte) error {
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	_, err := c.conn.Write(msg)
	return err
}

func (c *ipcConn) close(string) error {
	var err error
	c.closeOnce.Do(func() { err = c.conn.Close() })
	return err
}
