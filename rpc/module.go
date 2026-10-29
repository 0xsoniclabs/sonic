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
	"context"
	"encoding/json"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

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
	// WebSocket is the WebSocket endpoint, nil for none. At the address of
	// the HTTP endpoint, both share one server, as in go-ethereum's node.
	WebSocket *WebSocketEndpoint
	// IPC is the unix socket endpoint, nil for none.
	IPC *IPCEndpoint
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

// WebSocketEndpoint is a WebSocket endpoint the module listens on.
type WebSocketEndpoint struct {
	// Address is host:port.
	Address string
	// Namespaces are the namespaces it serves, every one when empty.
	Namespaces []string
	// Timeouts are those of its server when it has one of its own.
	Timeouts transport.Timeouts
	transport.WebSocketConfig
}

// IPCEndpoint is the unix socket the module listens on. It serves every
// namespace, the node's own ones included (X3).
type IPCEndpoint struct {
	Path  string
	Batch transport.BatchLimits
}

// Module is the RPC module.
type Module struct {
	config      Config
	dispatcher  *dispatch.Dispatcher
	chain       *chain.Chain
	environment *exec.Environment
	front       *transport.Front
	http        *transport.Server
	websocket   *transport.Server
	ipc         *transport.IPCServer
}

// New returns the module over the node, with its metrics registered on the
// registerer.
func New(config Config, nodeCore core.Core, registerer prometheus.Registerer) *Module {
	metrics := dispatch.NewMetrics(registerer)
	dispatcher := dispatch.New(config.Dispatch, metrics)
	served := chain.New(config.Chain, nodeCore, metrics)
	environment := exec.NewEnvironment(nodeCore, served, config.Exec, config.Oracle)
	registerNamespaces(dispatcher, environment, nodeCore, config)

	module := &Module{
		config:      config,
		dispatcher:  dispatcher,
		chain:       served,
		environment: environment,
		front:       transport.NewFront(dispatcher, enabledNamespaces(config)),
	}
	module.makeEndpoints(served)
	return module
}

// registerNamespaces adds the methods of every namespace the module serves.
func registerNamespaces(dispatcher *dispatch.Dispatcher, environment *exec.Environment, nodeCore core.Core, config Config) {
	eth.Register(dispatcher, environment, nodeCore)
	filters.Register(dispatcher, environment, nodeCore)
	net.Register(dispatcher, environment)
	web3.Register(dispatcher, config.ClientVersion)
	sonic.Register(dispatcher, environment)
	txpool.Register(dispatcher, environment, nodeCore)
	debug.Register(dispatcher, environment)
	trace.Register(dispatcher, environment)
}

// enabledNamespaces are the namespaces each transport serves; IPC serves
// every one.
func enabledNamespaces(config Config) map[string][]string {
	enabled := map[string][]string{}
	if config.HTTP != nil {
		enabled[transport.HTTP] = config.HTTP.Namespaces
	}
	if config.WebSocket != nil {
		enabled[transport.WebSocket] = config.WebSocket.Namespaces
	}
	return enabled
}

// makeEndpoints makes the servers of the endpoints: WebSocket on the HTTP
// server when their addresses match, on a server of its own otherwise.
func (module *Module) makeEndpoints(served *chain.Chain) {
	config := module.config
	if config.HTTP != nil {
		module.http = transport.NewServer(config.HTTP.Address, config.HTTP.Timeouts, module.dispatcher)
		module.http.EnableHTTP(module.front, config.HTTP.HTTPConfig, served)
	}
	if config.WebSocket != nil {
		if module.http != nil && config.WebSocket.Address == config.HTTP.Address {
			module.http.EnableWebSocket(module.front, config.WebSocket.WebSocketConfig)
		} else {
			module.websocket = transport.NewServer(config.WebSocket.Address, config.WebSocket.Timeouts, module.dispatcher)
			module.websocket.EnableWebSocket(module.front, config.WebSocket.WebSocketConfig)
		}
	}
	if config.IPC != nil {
		module.ipc = transport.NewIPCServer(config.IPC.Path, module.front, config.IPC.Batch, module.dispatcher)
	}
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
	for _, server := range module.endpoints() {
		if err := server.Start(); err != nil {
			_ = module.Stop()
			return fmt.Errorf("failed to start the %s endpoint: %w", server.name, err)
		}
	}
	return nil
}

// endpoint is a server of the module with its name.
type endpoint struct {
	name string
	server
}

type server interface {
	Start() error
	Stop()
}

// endpoints are the servers of the module, in the order they start.
func (module *Module) endpoints() []endpoint {
	var endpoints []endpoint
	if module.http != nil {
		endpoints = append(endpoints, endpoint{"HTTP", module.http})
	}
	if module.websocket != nil {
		endpoints = append(endpoints, endpoint{"WebSocket", module.websocket})
	}
	if module.ipc != nil {
		endpoints = append(endpoints, endpoint{"IPC", module.ipc})
	}
	return endpoints
}

// Stop closes the endpoints, waits for the requests in flight and stops the
// loops. Once it returned, the module reads nothing of the node.
func (module *Module) Stop() error {
	for _, server := range module.endpoints() {
		server.Stop()
	}
	module.dispatcher.Stop()
	return nil
}

// Call serves a call of a client in the node's own process, as the
// dispatcher serves one of a transport, and decodes its result into result.
func (module *Module) Call(ctx context.Context, result any, method string, parameters ...any) error {
	encoded, err := json.Marshal(parameters)
	if err != nil {
		return err
	}
	answer, failure := module.dispatcher.Serve(ctx, method, encoded)
	if failure != nil {
		return failure
	}
	return json.Unmarshal(answer, result)
}

// HTTPEndpoint returns the URL of the HTTP endpoint, empty without one.
func (module *Module) HTTPEndpoint() string {
	if module.http == nil {
		return ""
	}
	return "http://" + module.http.Address()
}

// WebSocketEndpoint returns the URL of the WebSocket endpoint, empty
// without one.
func (module *Module) WebSocketEndpoint() string {
	switch {
	case module.websocket != nil:
		return "ws://" + module.websocket.Address()
	case module.config.WebSocket != nil && module.http != nil:
		return "ws://" + module.http.Address()
	}
	return ""
}

// ServesExternally tells whether the module serves an endpoint other
// processes can reach: HTTP or WebSocket, as go-ethereum's node decides.
func (module *Module) ServesExternally() bool {
	return module.http != nil || module.config.WebSocket != nil
}
