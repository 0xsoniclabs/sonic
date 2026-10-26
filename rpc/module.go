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

// Package rpc is the RPC module of sonicd: it serves the JSON-RPC namespaces
// over HTTP, WebSocket and IPC, and reaches the node through core.Core only.
package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/debug"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/eth"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/filters"
	"github.com/0xsoniclabs/sonic/rpc/net"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
	"github.com/0xsoniclabs/sonic/rpc/sonic"
	"github.com/0xsoniclabs/sonic/rpc/trace"
	"github.com/0xsoniclabs/sonic/rpc/transport"
	"github.com/0xsoniclabs/sonic/rpc/txpool"
	"github.com/0xsoniclabs/sonic/rpc/web3"
)

// Config composes the configurations of the module's packages; the node
// fills each from its own configuration.
type Config struct {
	Dispatch  dispatch.Config
	Chain     chain.Config
	Exec      exec.Config
	Oracle    oracle.Config
	Transport transport.Config
	Front     transport.FrontConfig
	// ClientVersion is the answer of web3_clientVersion.
	ClientVersion string
}

// DefaultConfig holds the defaults of every package, and no endpoint.
func DefaultConfig() Config {
	return Config{
		Dispatch: dispatch.DefaultConfig(),
		Chain:    chain.DefaultConfig(),
		Exec:     exec.DefaultConfig(),
		Oracle:   oracle.DefaultConfig(),
	}
}

// Module is the RPC module.
type Module struct {
	d      *dispatch.Dispatcher
	chain  *chain.Chain
	server *transport.Server
	front  *transport.Front
	grace  time.Duration
}

// New wires the module over the node's core. The metrics go to reg, the
// node's default registry.
func New(c core.Core, cfg Config, reg prometheus.Registerer) *Module {
	d := dispatch.New(cfg.Dispatch, dispatch.NewMetrics(reg))
	ch := chain.New(c, cfg.Chain, d)
	o := oracle.New(cfg.Oracle, c, func() (*opera.Rules, error) {
		return c.Rules(ch.Latest().Number() + 1)
	})
	d.Loop("oracle", o.Run)
	env := exec.NewEnv(c, ch, o, d, cfg.Exec)

	for _, ns := range []string{"eth", "ftm"} {
		eth.Register(d, env, c, c, ns)
	}
	filters.Register(d, env, c, "eth", "ftm")
	debug.Register(d, env)
	trace.Register(d, env)
	txpool.Register(d, env, c)
	sonic.Register(d, env)
	net.Register(d, env)
	web3.Register(d, cfg.ClientVersion)

	front := transport.NewFront(d, cfg.Front)
	m := &Module{
		d:     d,
		chain: ch,
		front: front,
		grace: cfg.Dispatch.RequestTimeout,
	}
	if m.grace <= 0 {
		m.grace = dispatch.DefaultConfig().RequestTimeout
	}
	m.server = transport.NewServer(d, front, cfg.Transport, m.health)
	return m
}

// health is the answer of /health: the served head and its lag behind the
// committed head.
func (m *Module) health() any {
	return struct {
		Head hexutil.Uint64 `json:"head"`
		Lag  uint64         `json:"lag"`
	}{Head: hexutil.Uint64(m.chain.Latest().Number()), Lag: m.chain.Lag()}
}

// Start reads the head, starts the loops and opens the endpoints.
func (m *Module) Start() error {
	if err := m.chain.Start(); err != nil {
		return err
	}
	m.d.Start()
	return m.server.Start()
}

// Stop closes the endpoints, waits for the requests in flight up to the
// request deadline, cancels what is left and waits for it: once it returns,
// nothing of the module runs and the node may close its stores.
func (m *Module) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), m.grace)
	defer cancel()
	m.server.Close(ctx)
	m.d.Stop(m.grace)
	return nil
}

// Call serves one call in process, for the node's own clients of its RPC.
func (m *Module) Call(ctx context.Context, result any, method string, params ...any) error {
	raw, err := json.Marshal(params)
	if err != nil {
		return err
	}
	res, err := m.d.Serve(ctx, dispatch.Call{Method: method, Params: raw})
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(res, result); err != nil {
		return fmt.Errorf("decoding the answer of %s: %w", method, err)
	}
	return nil
}

// HTTPEndpoint is the address of the HTTP endpoint the module serves.
func (m *Module) HTTPEndpoint() string { return m.server.HTTPEndpoint() }

// WSEndpoint is the address of the WebSocket endpoint the module serves.
func (m *Module) WSEndpoint() string { return m.server.WSEndpoint() }

// IPCEndpoint is the path of the IPC socket the module serves.
func (m *Module) IPCEndpoint() string { return m.server.IPCEndpoint() }
