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

// Package rpc is the RPC module of the node: it wires the namespaces to the
// dispatcher over one narrow interface to the node, and starts and stops the
// transports.
package rpc

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/eth"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/filters"
	"github.com/0xsoniclabs/sonic/rpc/net"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
	"github.com/0xsoniclabs/sonic/rpc/sonic"
	"github.com/0xsoniclabs/sonic/rpc/transport"
	"github.com/0xsoniclabs/sonic/rpc/txpool"
	"github.com/0xsoniclabs/sonic/rpc/web3"
)

// Config composes the settings of the module's packages. The composition
// root fills each from the node configuration.
type Config struct {
	Dispatch dispatch.Config
	Chain    chain.Config
	Exec     exec.Config
	Oracle   oracle.Config

	// ClientVersion is the name the node gives itself, web3_clientVersion.
	ClientVersion string

	// HTTP is the HTTP endpoint the module serves, nil for none.
	HTTP *HTTPEndpoint
}

// DefaultConfig returns the default settings, without endpoints.
func DefaultConfig() Config {
	return Config{
		Dispatch: dispatch.DefaultConfig(),
		Chain:    chain.DefaultConfig(),
		Exec:     exec.DefaultConfig(),
		Oracle:   oracle.DefaultConfig(),
	}
}

// HTTPEndpoint is an HTTP endpoint the module listens on.
type HTTPEndpoint struct {
	// Address is host:port.
	Address string
	// Namespaces are the namespaces it serves, every one when empty.
	Namespaces []string
	Timeouts   transport.Timeouts
	transport.HTTPConfig
}

// Module is the RPC module.
type Module struct {
	config      Config
	dispatcher  *dispatch.Dispatcher
	chain       *chain.Chain
	environment *exec.Environment
	front       *transport.Front
	http        *transport.Server
}

// New returns the module over the node, with its metrics registered on the
// registerer.
func New(config Config, nodeCore core.Core, registerer prometheus.Registerer) *Module {
	metrics := dispatch.NewMetrics(registerer)
	dispatcher := dispatch.New(config.Dispatch, metrics)
	served := chain.New(config.Chain, nodeCore, metrics)
	environment := exec.NewEnvironment(nodeCore, served, config.Exec, config.Oracle)
	registerNamespaces(dispatcher, environment, nodeCore, config)

	enabled := map[string][]string{}
	if config.HTTP != nil {
		enabled[transport.HTTP] = config.HTTP.Namespaces
	}
	module := &Module{
		config:      config,
		dispatcher:  dispatcher,
		chain:       served,
		environment: environment,
		front:       transport.NewFront(dispatcher, enabled),
	}
	if config.HTTP != nil {
		module.http = transport.NewServer(config.HTTP.Address, config.HTTP.Timeouts, dispatcher)
		module.http.EnableHTTP(module.front, config.HTTP.HTTPConfig, served)
	}
	return module
}

// registerNamespaces adds the methods of every namespace the module serves.
func registerNamespaces(dispatcher *dispatch.Dispatcher, environment *exec.Environment, nodeCore core.Core, config Config) {
	eth.Register(dispatcher, environment, nodeCore, "eth")
	eth.Register(dispatcher, environment, nodeCore, "ftm")
	filters.Register(dispatcher, environment, nodeCore, "eth", "ftm")
	net.Register(dispatcher, environment)
	web3.Register(dispatcher, config.ClientVersion)
	sonic.Register(dispatcher, environment)
	txpool.Register(dispatcher, environment, nodeCore)
}

// Forward makes the module hand every call it does not serve to the node's
// own handler.
func (module *Module) Forward(forwarder transport.Forwarder) {
	module.front.Forward(forwarder)
}

// Start reads the head and starts the loops and the endpoints.
func (module *Module) Start() error {
	if err := module.chain.Start(module.dispatcher); err != nil {
		return fmt.Errorf("failed to start the RPC module: %w", err)
	}
	module.dispatcher.Loop("oracle", module.environment.Oracle().Sample)
	if module.http != nil {
		if err := module.http.Start(); err != nil {
			module.dispatcher.Stop()
			return fmt.Errorf("failed to start the HTTP endpoint: %w", err)
		}
	}
	return nil
}

// Stop closes the endpoints, waits for the requests in flight and stops the
// loops. Once it returned, the module reads nothing of the node.
func (module *Module) Stop() error {
	if module.http != nil {
		module.http.Stop()
	}
	module.dispatcher.Stop()
	return nil
}

// HTTPEndpoint returns the URL of the HTTP endpoint, empty without one.
func (module *Module) HTTPEndpoint() string {
	if module.http == nil {
		return ""
	}
	return "http://" + module.http.Address()
}

// ServesExternally tells whether the module serves an endpoint other
// processes can reach.
func (module *Module) ServesExternally() bool {
	return module.http != nil
}
