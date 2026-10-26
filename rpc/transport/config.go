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

import "time"

// The limits of the transports themselves, as go-ethereum's rpc package set
// them; the knobs of the node configuration arrive in Config.
const (
	httpBodyLimit = 5 * 1024 * 1024
	wsReadLimit   = 32 * 1024 * 1024

	// writeTimeout bounds one write to a stream connection; a client that
	// does not read is disconnected after it.
	writeTimeout = 10 * time.Second

	wsPingInterval     = 30 * time.Second
	wsPingWriteTimeout = 5 * time.Second
	wsPongTimeout      = 30 * time.Second

	// maxSubscriptions bounds the subscriptions of one stream connection,
	// and subscriptionBuffer the notifications one subscription queues
	// before its client is disconnected.
	maxSubscriptions   = 1000
	subscriptionBuffer = 20000

	// timeoutMargin is how long before the HTTP write timeout a running call
	// is answered "request timed out", as in go-ethereum.
	timeoutMargin = 100 * time.Millisecond
)

// Config holds the endpoints and the knobs of the node configuration, as the
// node configuration holds them: a zero value means what it means there.
type Config struct {
	HTTP   *HTTPConfig // nil: no HTTP endpoint
	WS     *WSConfig   // nil: no WebSocket endpoint
	IPC    string      // path of the socket; empty: no IPC endpoint
	Limits Limits
}

// Limits bound a batch: its calls and the size of its results.
type Limits struct {
	BatchRequestLimit    int
	BatchResponseMaxSize int
}

// HTTPConfig is the HTTP endpoint.
type HTTPConfig struct {
	Host     string
	Port     int
	Cors     []string
	Vhosts   []string
	Prefix   string
	Timeouts Timeouts
	// FallbackTimeouts replace a configured timeout below a second, as the
	// node did.
	FallbackTimeouts Timeouts
}

// WSConfig is the WebSocket endpoint.
type WSConfig struct {
	Host    string
	Port    int
	Origins []string
	Prefix  string
}

// Timeouts of an HTTP server; all zero means none.
type Timeouts struct {
	Read, ReadHeader, Write, Idle time.Duration
}

// sanitized replaces each timeout below a second by its fallback, unless no
// timeout is set at all.
func (t Timeouts) sanitized(fallback Timeouts) Timeouts {
	if t == (Timeouts{}) {
		return t
	}
	fix := func(v, def time.Duration) time.Duration {
		if v < time.Second {
			return def
		}
		return v
	}
	return Timeouts{
		Read:       fix(t.Read, fallback.Read),
		ReadHeader: fix(t.ReadHeader, fallback.ReadHeader),
		Write:      fix(t.Write, fallback.Write),
		Idle:       fix(t.Idle, fallback.Idle),
	}
}
