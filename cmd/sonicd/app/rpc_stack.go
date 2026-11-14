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

package app

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/config/flags"
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/rpc"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	rpcexec "github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
	"github.com/0xsoniclabs/sonic/rpc/transport"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/urfave/cli.v1"
)

// rpcStack is the RPC stack a node serves. One runs, picked by
// chooseRPCStack; the other registers nothing and starts nothing.
type rpcStack interface {
	// prepare changes the configuration before the node is made from it.
	prepare(cfg *config.Config)
	// attach registers the stack with the node before the node starts.
	attach(stack *node.Node, service *gossip.Service) error
	// httpEndpoint is the URL of the HTTP endpoint, once the node runs.
	httpEndpoint(stack *node.Node) string
	// servesAccounts tells whether a method of the stack uses the node's
	// accounts, which are then unlocked and opened at start.
	servesAccounts() bool
}

// chooseRPCStack picks the stack the configuration names.
func chooseRPCStack(cfg *config.Config) rpcStack {
	if cfg.Opera.RPCStack == gossip.ModuleRPCStack {
		return &moduleStack{}
	}
	return legacyStack{}
}

// legacyStack is go-ethereum's RPC server of the node, with the handlers of
// the api packages, on the node's own endpoints.
type legacyStack struct{}

// prepare gives feedback on past RPC changes.
func (legacyStack) prepare(cfg *config.Config) {
	for flag, modules := range map[string][]string{
		flags.HTTPApiFlag.Name: cfg.Node.HTTPModules,
		flags.WSApiFlag.Name:   cfg.Node.WSModules,
	} {
		for _, module := range modules {
			if strings.ToLower(module) == "ftm" {
				log.Warn(fmt.Sprintf("The 'ftm' API is deprecated, use 'eth' instead (--%s).", flag))
			}
		}
	}
}

func (legacyStack) attach(stack *node.Node, service *gossip.Service) error {
	stack.RegisterAPIs(service.APIs())
	return nil
}

func (legacyStack) httpEndpoint(stack *node.Node) string {
	return stack.HTTPEndpoint()
}

func (legacyStack) servesAccounts() bool {
	return true
}

// moduleStack is the node's RPC module, which serves every endpoint the
// configuration names in place of the node.
type moduleStack struct {
	config rpc.Config
	module *rpc.Module
}

func (m *moduleStack) prepare(cfg *config.Config) {
	m.config = takeOverRPC(&cfg.Node)
	setModuleConfig(&m.config, cfg.Opera)
}

// setModuleConfig maps the node's configuration of the RPC into the module's.
func setModuleConfig(module *rpc.Config, node gossip.Config) {
	module.Dispatch = dispatch.Config{RequestTimeout: node.RPCTimeout}
	module.Exec = rpcexec.Config{
		RPCGasCap:     node.RPCGasCap,
		RPCEVMTimeout: node.RPCEVMTimeout,
	}
	module.Oracle = oracle.Config{
		MaxGasPrice:      node.GPO.MaxGasPrice,
		MinGasPrice:      node.GPO.MinGasPrice,
		DefaultCertainty: node.GPO.DefaultCertainty,
	}
}

func (m *moduleStack) attach(stack *node.Node, service *gossip.Service) error {
	handler, err := stack.RPCHandler()
	if err != nil {
		return err
	}
	m.config.ClientVersion = stack.Server().Name
	m.module = rpc.New(m.config, gossip.NewRPCCore(service), handler, prometheus.DefaultRegisterer)
	// As a lifecycle registered after the service, the module starts once the
	// service runs and stops before it, whatever closes the node.
	stack.RegisterLifecycle(moduleLifecycle{m.module})
	return nil
}

func (m *moduleStack) httpEndpoint(*node.Node) string {
	endpoint, err := m.module.HTTPEndpoint()
	if err != nil {
		return "http://"
	}
	return endpoint
}

func (m *moduleStack) servesAccounts() bool {
	return false
}

// unlockFlags are the flags that unlock the node's accounts, which only
// methods of the legacy stack use.
var unlockFlags = []string{
	flags.UnlockedAccountFlag.Name,
	flags.PasswordFileFlag.Name,
	flags.InsecureUnlockAllowedFlag.Name,
}

// warnAboutAccountFlags warns about each account flag given to a node whose
// RPC stack serves no account: it is accepted and has no effect.
func warnAboutAccountFlags(ctx *cli.Context) {
	for _, name := range unlockFlags {
		if ctx.GlobalIsSet(name) {
			log.Warn(fmt.Sprintf("The --%s flag has no effect with --%s %s: no method it serves uses an account.",
				name, flags.RPCStackFlag.Name, gossip.ModuleRPCStack))
		}
	}
}

// moduleLifecycle runs the RPC module as a lifecycle of the node.
type moduleLifecycle struct {
	*rpc.Module
}

func (l moduleLifecycle) Stop() error {
	l.Module.Stop()
	return nil
}

// takeOverRPC moves the RPC endpoints out of the node's configuration into
// the module's, so that the node opens none and the module listens on each.
// The node then reports no external RPC.
func takeOverRPC(cfg *node.Config) rpc.Config {
	module := rpc.DefaultConfig()
	if cfg.HTTPHost != "" {
		module.HTTP = &rpc.HTTPConfig{
			HTTPConfig: transport.HTTPConfig{
				Endpoint:           net.JoinHostPort(cfg.HTTPHost, strconv.Itoa(cfg.HTTPPort)),
				PathPrefix:         cfg.HTTPPathPrefix,
				CorsAllowedOrigins: cfg.HTTPCors,
				VirtualHosts:       cfg.HTTPVirtualHosts,
				Timeouts: transport.Timeouts{
					Read:       cfg.HTTPTimeouts.ReadTimeout,
					ReadHeader: cfg.HTTPTimeouts.ReadHeaderTimeout,
					Write:      cfg.HTTPTimeouts.WriteTimeout,
					Idle:       cfg.HTTPTimeouts.IdleTimeout,
				},
				Batch: transport.BatchLimits{Items: cfg.BatchRequestLimit, ResponseBytes: cfg.BatchResponseMaxSize},
			},
			Modules: cfg.HTTPModules,
		}
		cfg.HTTPHost = ""
	}
	if cfg.WSHost != "" {
		log.Warn("The RPC module does not serve WebSocket yet; the endpoint is not opened",
			"endpoint", fmt.Sprintf("ws://%s", net.JoinHostPort(cfg.WSHost, strconv.Itoa(cfg.WSPort))))
		cfg.WSHost = ""
	}
	module.IPCPath = cfg.IPCEndpoint()
	cfg.IPCPath = ""
	return module
}
