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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/rpc"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/transport"
)

// rpcTakeover remembers what the RPC module took over from the node's own
// servers: the node's HTTP, WebSocket and IPC servers stay off and the module
// listens on the configured endpoints instead.
type rpcTakeover struct {
	// http is the configuration of the module's HTTP transport, which serves
	// WebSocket too if the operator configured the same endpoint for both; nil
	// if no HTTP endpoint is configured.
	http *transport.HTTPConfig
	// ws is the module's WebSocket listener of its own; nil if the endpoint is
	// the HTTP endpoint or WebSocket is off.
	ws *transport.HTTPConfig
	// ipc is the module's unix socket; nil if IPC is off.
	ipc *rpc.IPCConfig

	// legacyEngine is true if the operator chose the legacy engine: the node's
	// own servers stay on and there is no module.
	legacyEngine bool

	// original holds the endpoint settings the operator configured, to put
	// them back where the configuration is reported.
	original struct {
		httpHost, wsHost, ipcPath string
		wsPort                    int
		wsOrigins                 []string
	}

	stack  *node.Node
	module *rpc.Module

	mu     sync.Mutex
	client *gethrpc.Client // < in-process client of the node's own handler
}

// takeOverRPC rewrites the node's endpoint configuration so that the module
// can listen on the endpoints the operator configured. It must run before the
// node is created.
func takeOverRPC(cfg *node.Config) *rpcTakeover {
	t := &rpcTakeover{}
	t.original.httpHost = cfg.HTTPHost
	t.original.wsHost = cfg.WSHost
	t.original.wsPort = cfg.WSPort
	t.original.wsOrigins = cfg.WSOrigins
	t.original.ipcPath = cfg.IPCPath

	timeouts := transport.Timeouts{
		Read:       cfg.HTTPTimeouts.ReadTimeout,
		ReadHeader: cfg.HTTPTimeouts.ReadHeaderTimeout,
		Write:      cfg.HTTPTimeouts.WriteTimeout,
		Idle:       cfg.HTTPTimeouts.IdleTimeout,
	}
	wsConfig := func() *transport.WSConfig {
		return &transport.WSConfig{
			StreamConfig: transport.StreamConfig{EndpointConfig: transport.EndpointConfig{
				Modules:          append([]string(nil), cfg.WSModules...),
				BatchLimit:       cfg.BatchRequestLimit,
				BatchResponseMax: cfg.BatchResponseMaxSize,
			}},
			Origins: append([]string(nil), cfg.WSOrigins...),
		}
	}

	wsShared := cfg.WSHost != "" && cfg.WSHost == cfg.HTTPHost && cfg.WSPort == cfg.HTTPPort
	if cfg.HTTPHost != "" {
		t.http = &transport.HTTPConfig{
			Host:             cfg.HTTPHost,
			Port:             cfg.HTTPPort,
			Prefix:           cfg.HTTPPathPrefix,
			Cors:             append([]string(nil), cfg.HTTPCors...),
			VHosts:           append([]string(nil), cfg.HTTPVirtualHosts...),
			Modules:          append([]string(nil), cfg.HTTPModules...),
			Timeouts:         timeouts,
			BatchLimit:       cfg.BatchRequestLimit,
			BatchResponseMax: cfg.BatchResponseMaxSize,
		}
		if wsShared {
			t.http.WS = wsConfig()
			t.http.WSPrefix = cfg.WSPathPrefix
		}
	}
	if cfg.WSHost != "" && !wsShared {
		t.ws = &transport.HTTPConfig{
			Host:     cfg.WSHost,
			Port:     cfg.WSPort,
			Timeouts: timeouts,
			WS:       wsConfig(),
			WSPrefix: cfg.WSPathPrefix,
			VHosts:   append([]string(nil), cfg.HTTPVirtualHosts...),
		}
	}
	if path := cfg.IPCEndpoint(); path != "" {
		t.ipc = &rpc.IPCConfig{Path: path}
	}

	// The node's own servers stay off.
	cfg.HTTPHost = ""
	cfg.WSHost = ""
	cfg.IPCPath = ""
	return t
}

// restore puts the endpoint settings the operator configured back into the
// node configuration, for the places that report the configuration.
func (t *rpcTakeover) restore(cfg *node.Config) {
	if t.legacyEngine {
		return
	}
	cfg.HTTPHost = t.original.httpHost
	cfg.WSHost = t.original.wsHost
	cfg.WSPort = t.original.wsPort
	cfg.WSOrigins = t.original.wsOrigins
	cfg.IPCPath = t.original.ipcPath
}

// attach creates the module over the service and registers it as a lifecycle
// of the node, after the service: it starts after the service and stops
// before it.
func (t *rpcTakeover) attach(stack *node.Node, svc *gossip.Service, cfg *config.Config) error {
	if t.legacyEngine {
		return nil
	}
	limits := limitsFromConfig(cfg.Opera)
	modCfg := rpc.Config{
		Limits:        limits,
		ClientVersion: stack.Server().Name,
		HTTP:          t.http,
		WS:            t.ws,
		IPC:           t.ipc,

		LegacyNamespaces: []string{"abft", "admin", "dag", "personal", "test"},
	}
	module, err := rpc.New(modCfg, svc.RPCCore(), rpc.WithLegacy(t))
	if err != nil {
		return fmt.Errorf("failed to create the RPC module: %w", err)
	}
	t.module = module
	t.stack = stack
	if t.http != nil || t.ws != nil {
		// The node's own HTTP and WebSocket servers are off, but external RPC
		// is enabled; the legacy handlers use this to refuse account unlocking.
		svc.EthAPI.SetExtRPCEnabled(true)
	}
	stack.RegisterLifecycle(t)
	return nil
}

// httpEndpoint is the HTTP endpoint clients connect to.
func (t *rpcTakeover) httpEndpoint(stack *node.Node) string {
	if t.module != nil {
		if addr := t.module.HTTPAddr(); addr != "" {
			return "http://" + addr
		}
	}
	return stack.HTTPEndpoint()
}

// Start starts the module; it satisfies node.Lifecycle.
func (t *rpcTakeover) Start() error { return t.module.Start() }

// Stop stops the module and the in-process client; it satisfies node.Lifecycle.
func (t *rpcTakeover) Stop() error {
	err := t.module.Stop()
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.client != nil {
		t.client.Close()
		t.client = nil
	}
	return err
}

var errNoLegacy = errors.New("legacy RPC handler is not available")

// Forward hands a call to the node's own RPC handler through its in-process
// client.
func (t *rpcTakeover) Forward(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, *dispatch.Error) {
	client, err := t.legacyClient()
	if err != nil {
		return nil, dispatch.NewError(dispatch.CodeInternal, "%v", err)
	}
	raw, err := dispatch.SplitParams(params)
	if err != nil {
		return nil, dispatch.InvalidParams("%v", err)
	}
	args := make([]any, len(raw))
	for i := range raw {
		args[i] = raw[i]
	}
	var result json.RawMessage
	if err := client.CallContext(ctx, &result, method, args...); err != nil {
		return nil, legacyError(err)
	}
	return result, nil
}

// legacyClient returns the in-process client of the node's own handler. It is
// created on first use, once the node is running.
func (t *rpcTakeover) legacyClient() (*gethrpc.Client, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.client != nil {
		return t.client, nil
	}
	if t.stack == nil {
		return nil, errNoLegacy
	}
	t.client = t.stack.Attach()
	return t.client, nil
}

func legacyError(err error) *dispatch.Error {
	var coded gethrpc.Error
	if errors.As(err, &coded) {
		e := &dispatch.Error{Code: coded.ErrorCode(), Message: err.Error()}
		var data gethrpc.DataError
		if errors.As(err, &data) {
			e.Data = data.ErrorData()
		}
		return e
	}
	log.Debug("Legacy RPC call failed", "err", err)
	return dispatch.NewError(dispatch.CodeInternal, "%v", err)
}

// limitsFromConfig maps the node's RPC limits to the module's.
func limitsFromConfig(g gossip.Config) dispatch.Config {
	limits := dispatch.DefaultConfig()
	limits.Timeout = g.RPCTimeout
	limits.EVMTimeout = g.RPCEVMTimeout
	limits.GasCap = g.RPCGasCap
	limits.TxFeeCap = g.RPCTxFeeCap
	limits.UnprotectedAllowed = g.AllowUnprotectedTxs
	limits.TxSyncDefaultTimeout = g.TxSyncDefaultTimeout
	limits.TxSyncMaxTimeout = g.TxSyncMaxTimeout
	limits.MaxResponseSize = g.MaxResponseSize
	limits.StructLogLimit = g.StructLogLimit
	limits.MaxTraceFilterRange = g.MaxTraceFilterRange
	limits.AllowJSTracers = g.AllowJSTracers
	limits.LogQueryParameterLimit = g.FilterAPI.LogQueryParameterLimit
	limits.LogQueryResultLimit = g.FilterAPI.LogQueryResultLimit
	limits.IndexedLogsBlockRangeLimit = uint64(g.FilterAPI.IndexedLogsBlockRangeLimit)
	limits.UnindexedLogsBlockRangeLimit = uint64(g.FilterAPI.UnindexedLogsBlockRangeLimit)
	if g.GPO.MaxGasPrice != nil {
		limits.OracleMaxGasPrice = g.GPO.MaxGasPrice
	}
	if g.GPO.DefaultCertainty != 0 {
		limits.OracleDefaultCertainty = g.GPO.DefaultCertainty
	}
	return limits
}
