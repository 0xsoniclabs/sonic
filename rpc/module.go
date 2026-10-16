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

// Package rpc is the node's RPC module. It reaches the node only through
// core.Core and serves JSON-RPC over HTTP, WebSocket and a unix socket. Methods
// the module does not serve yet are forwarded to the node's legacy handler.
package rpc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

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

// Config configures the module.
type Config struct {
	// Limits are the semantic limits of the methods.
	Limits dispatch.Config
	// ClientVersion is the answer of web3_clientVersion.
	ClientVersion string
	// HTTP configures the HTTP transport; nil keeps it off. It serves
	// WebSocket as well if its WS field is set.
	HTTP *transport.HTTPConfig
	// WS configures a WebSocket listener of its own, for the case that the
	// WebSocket endpoint is not the HTTP endpoint; nil keeps it off.
	WS *transport.HTTPConfig
	// IPC configures the unix socket transport; nil keeps it off.
	IPC *IPCConfig
	// LegacyNamespaces are the namespaces the legacy handler adds, listed by
	// rpc_modules.
	LegacyNamespaces []string
	// ShutdownGrace is how long Stop waits for requests in flight.
	ShutdownGrace time.Duration
}

// legacyIPCMethods are the methods of the namespaces the module serves that the
// node's legacy handler keeps answering on IPC: the wallet methods and the
// consensus internals that are removed (X1, X2).
var legacyIPCMethods = []string{
	"eth_accounts", "eth_sign", "eth_signTransaction", "eth_sendTransaction", "eth_fillTransaction", "eth_resend",
	"eth_getEpochBlock", "eth_getEpochStats",
	"ftm_accounts", "ftm_sign", "ftm_signTransaction", "ftm_sendTransaction", "ftm_fillTransaction", "ftm_resend",
	"ftm_getEpochBlock", "ftm_getEpochStats",
	"debug_chaindbProperty", "debug_chaindbCompact", "debug_setHead",
}

// IPCConfig configures the unix socket transport.
type IPCConfig struct {
	// Path is the path of the socket.
	Path string
	transport.StreamConfig
}

// Module is the RPC module of a node.
type Module struct {
	cfg     Config
	core    core.Core
	d       *dispatch.Dispatcher
	chain   *dispatch.Chain
	oracle  *oracle.Oracle
	front   *transport.Front
	http    *transport.HTTPServer
	ws      *transport.HTTPServer
	ipc     *transport.IPCServer
	filters *filters.API
	mu      sync.Mutex
	state   int
	legacy  transport.Forwarder
}

const (
	stateNew = iota
	stateRunning
	stateStopped
)

// Option configures a Module.
type Option func(*Module)

// WithLegacy sets the handler the module forwards to for methods it does not
// serve.
func WithLegacy(f transport.Forwarder) Option { return func(m *Module) { m.legacy = f } }

// WithMetrics sets the Prometheus collectors; the default is the default
// registry.
func WithMetrics(metrics *dispatch.Metrics) Option {
	return func(m *Module) { m.d = dispatch.New(m.cfg.Limits, metrics) }
}

// New creates the module and registers the namespaces. Nothing runs until
// Start.
func New(cfg Config, c core.Core, opts ...Option) (*Module, error) {
	m := &Module{cfg: cfg, core: c}
	for _, opt := range opts {
		opt(m)
	}
	if m.d == nil {
		m.d = dispatch.New(cfg.Limits, dispatch.DefaultMetrics())
	}
	if m.cfg.ShutdownGrace <= 0 {
		m.cfg.ShutdownGrace = 5 * time.Second
	}
	m.chain = dispatch.NewChain(c, cfg.Limits, m.d.Metrics())

	m.oracle = oracle.New(oracle.Config{
		MaxGasPrice:      cfg.Limits.OracleMaxGasPrice,
		DefaultCertainty: cfg.Limits.OracleDefaultCertainty,
	}, c)
	env := exec.NewEnv(m.chain, m.oracle)
	eth.Register(m.d, env, "eth")
	eth.Register(m.d, env, "ftm")
	m.filters = filters.New(env)
	m.filters.Register(m.d, "eth")
	m.filters.Register(m.d, "ftm")
	sonic.Register(m.d, env)
	debug.Register(m.d, env)
	trace.Register(m.d, env)
	txpool.Register(m.d, m.chain)
	net.Register(m.d, c)
	web3.Register(m.d, cfg.ClientVersion)

	frontOpts := []transport.FrontOption{
		transport.WithLegacyNamespaces(cfg.LegacyNamespaces...),
		// The consensus internals and the node's wallets are for IPC only; the
		// p2p facts are served on every transport.
		transport.WithLegacyOnIPC(cfg.LegacyNamespaces, legacyIPCMethods),
		transport.WithLegacyEverywhere("net_listening", "net_peerCount"),
	}
	if m.legacy != nil {
		frontOpts = append(frontOpts, transport.WithLegacy(m.legacy))
	}
	m.front = transport.NewFront(m.d, frontOpts...)

	if cfg.HTTP != nil {
		httpCfg := *cfg.HTTP
		if httpCfg.Health == nil {
			httpCfg.Health = m.health
		}
		srv, err := transport.NewHTTPServer(httpCfg, m.front)
		if err != nil {
			return nil, err
		}
		m.http = srv
	}
	if cfg.WS != nil {
		wsCfg := *cfg.WS
		wsCfg.DisableRPC = true
		srv, err := transport.NewHTTPServer(wsCfg, m.front)
		if err != nil {
			return nil, err
		}
		m.ws = srv
	}
	if cfg.IPC != nil {
		m.ipc = transport.NewIPCServer(cfg.IPC.Path, cfg.IPC.StreamConfig, m.front)
	}
	return m, nil
}

func (m *Module) health() transport.Health {
	head := m.chain.Latest()
	committed := m.core.Progress().CurrentBlock
	h := transport.Health{Head: head, Committed: committed}
	if committed > head {
		h.Lag = committed - head
	}
	return h
}

// Dispatcher returns the dispatcher of the module.
func (m *Module) Dispatcher() *dispatch.Dispatcher { return m.d }

// Front returns the router of the module.
func (m *Module) Front() *transport.Front { return m.front }

// Chain returns the head view of the module.
func (m *Module) Chain() *dispatch.Chain { return m.chain }

// IPCPath returns the path of the unix socket, "" if there is none.
func (m *Module) IPCPath() string {
	if m.cfg.IPC == nil {
		return ""
	}
	return m.cfg.IPC.Path
}

// WSAddr returns the address of the WebSocket listener of its own, "" if there is none.
func (m *Module) WSAddr() string {
	if m.ws == nil {
		return ""
	}
	return m.ws.Addr()
}

// HTTPAddr returns the address of the HTTP listener, "" if there is none.
func (m *Module) HTTPAddr() string {
	if m.http == nil {
		return ""
	}
	return m.http.Addr()
}

// Start takes the head snapshot, starts the loops and opens the listeners.
// It satisfies the node's lifecycle interface.
func (m *Module) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state != stateNew {
		return errors.New("rpc module cannot be started twice")
	}
	if err := m.chain.Start(m.d); err != nil {
		return err
	}
	m.d.Loop("oracle", m.oracle.Run)
	m.filters.Start(m.d)
	if m.http != nil {
		if err := m.http.Start(); err != nil {
			m.stopTransports()
			m.d.Stop(0)
			return fmt.Errorf("failed to start the HTTP transport: %w", err)
		}
	}
	if m.ws != nil {
		if err := m.ws.Start(); err != nil {
			m.stopTransports()
			m.d.Stop(0)
			return fmt.Errorf("failed to start the WebSocket transport: %w", err)
		}
	}
	if m.ipc != nil {
		if err := m.ipc.Start(); err != nil {
			m.stopTransports()
			m.d.Stop(0)
			return fmt.Errorf("failed to start the IPC transport: %w", err)
		}
	}
	m.state = stateRunning
	return nil
}

// stopTransports closes the listeners and the connections.
func (m *Module) stopTransports() {
	ctx, cancel := context.WithTimeout(context.Background(), m.cfg.ShutdownGrace)
	defer cancel()
	if m.ipc != nil {
		if err := m.ipc.Stop(); err != nil {
			log.Warn("RPC IPC server did not stop cleanly", "err", err)
		}
	}
	if m.ws != nil {
		if err := m.ws.Stop(ctx); err != nil {
			log.Warn("RPC WebSocket server did not stop cleanly", "err", err)
		}
	}
	if m.http != nil {
		if err := m.http.Stop(ctx); err != nil {
			log.Warn("RPC HTTP server did not stop cleanly", "err", err)
		}
	}
}

// Stop closes the listeners, drains the requests in flight and stops every
// goroutine of the module. It must run before the node closes. It satisfies
// the node's lifecycle interface.
func (m *Module) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state == stateStopped {
		return nil
	}
	m.state = stateStopped
	m.stopTransports()
	m.d.Stop(m.cfg.ShutdownGrace)
	return nil
}
