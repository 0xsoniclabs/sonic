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

// Package rpc is the RPC module of the node: its namespaces served over its
// own transports, reaching the node only through rpc/core. Module wires them
// and starts and stops them.
package rpc

import (
	"context"
	"errors"
	"net/http"

	"github.com/0xsoniclabs/sonic/rpc/abft"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dag"
	"github.com/0xsoniclabs/sonic/rpc/debug"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/eth"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/filters"
	"github.com/0xsoniclabs/sonic/rpc/net"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
	"github.com/0xsoniclabs/sonic/rpc/sonic"
	"github.com/0xsoniclabs/sonic/rpc/testonly"
	"github.com/0xsoniclabs/sonic/rpc/transport"
	"github.com/0xsoniclabs/sonic/rpc/txpool"
	"github.com/0xsoniclabs/sonic/rpc/web3"
	"github.com/prometheus/client_golang/prometheus"
)

// Config is the configuration of the module.
type Config struct {
	Dispatch dispatch.Config
	Chain    chain.Config
	Exec     exec.Config
	Oracle   oracle.Config
	// HTTP is the HTTP endpoint, none if nil.
	HTTP *HTTPConfig
	// IPCPath is the path of the IPC socket, none if empty.
	IPCPath string
	// ClientVersion is the node's name as its peers see it.
	ClientVersion string
}

// HTTPConfig is the HTTP endpoint with the namespaces it serves.
type HTTPConfig struct {
	transport.HTTPConfig
	// Modules is the API list of the endpoint; all namespaces if empty.
	Modules []string
}

// DefaultConfig is the configuration of a module without endpoints.
func DefaultConfig() Config {
	return Config{Dispatch: dispatch.DefaultConfig(), Exec: exec.DefaultConfig(), Oracle: oracle.DefaultConfig()}
}

// Module is the RPC module of a node.
type Module struct {
	config     Config
	core       core.Core
	node       http.Handler
	registerer prometheus.Registerer
	dispatcher *dispatch.Dispatcher
	http       *transport.HTTPServer
	ipc        *transport.IPCServer
}

// New creates a module over the node's core. The node's own handler answers
// the node's administration and profiling forwarded over IPC.
func New(config Config, node core.Core, nodeHandler http.Handler, registerer prometheus.Registerer) *Module {
	return &Module{config: config, core: node, node: nodeHandler, registerer: registerer}
}

// Start serves the module's namespaces on its endpoints.
func (m *Module) Start() error {
	m.dispatcher = dispatch.New(m.config.Dispatch, m.registerer)
	served := chain.New(m.config.Chain, m.core, m.registerer)
	suggester := oracle.New(m.config.Oracle, m.core, func() uint64 { return served.Head().NumberU64() })
	environment := exec.NewEnvironment(m.config.Exec, served, m.core, suggester)
	m.register(environment)
	if err := served.Start(m.dispatcher); err != nil {
		m.dispatcher.Stop()
		return err
	}
	m.dispatcher.Loop("gas price oracle", suggester.Sample)
	if err := m.startTransports(); err != nil {
		m.Stop()
		return err
	}
	return nil
}

func (m *Module) register(environment *exec.Environment) {
	dispatch.Add(m.dispatcher, "rpc_modules", modules)
	eth.Register(m.dispatcher, environment, m.core)
	filters.Register(m.dispatcher, environment, m.core)
	net.Register(m.dispatcher, environment, m.core)
	web3.Register(m.dispatcher, m.config.ClientVersion)
	debug.Register(m.dispatcher, environment)
	sonic.Register(m.dispatcher, environment, m.core)
	txpool.Register(m.dispatcher, environment, m.core)
	testonly.Register(m.dispatcher, environment, m.core)
	dag.Register(m.dispatcher, environment, m.core)
	abft.Register(m.dispatcher, environment, m.core)
}

// modules answers the namespaces served on the transport of the call.
func modules(ctx context.Context, _ struct{}) (map[string]string, error) {
	return transport.Modules(ctx), nil
}

func (m *Module) startTransports() error {
	var err error
	if m.config.HTTP != nil {
		front := transport.NewFront(m.dispatcher, m.config.HTTP.Modules, "http.api")
		if m.http, err = transport.StartHTTP(m.dispatcher, m.config.HTTP.HTTPConfig, front, m.registerer); err != nil {
			return err
		}
	}
	if m.config.IPCPath != "" {
		front := transport.NewIPCFront(m.dispatcher, m.node)
		if m.ipc, err = transport.StartIPC(m.dispatcher, m.config.IPCPath, front, m.registerer); err != nil {
			return err
		}
	}
	return nil
}

// HTTPEndpoint is the URL of the HTTP endpoint, without its path prefix.
func (m *Module) HTTPEndpoint() (string, error) {
	if m.http == nil {
		return "", errors.New("the RPC module serves no HTTP endpoint")
	}
	return "http://" + m.http.Address().String(), nil
}

// Stop closes the endpoints, lets the calls in flight finish, and returns
// once no goroutine of the module runs.
func (m *Module) Stop() {
	if m.http != nil {
		m.http.Stop()
	}
	if m.ipc != nil {
		m.ipc.Stop()
	}
	m.dispatcher.Stop()
}
