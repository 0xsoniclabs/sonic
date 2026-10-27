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
	"fmt"
	"net"
	"strconv"

	"github.com/ethereum/go-ethereum/node"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/rpc"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
	"github.com/0xsoniclabs/sonic/rpc/transport"
)

// newRPCModule creates the RPC module of the node, takes over the endpoints
// it serves and registers it with the node, which starts it after the
// gossip service and stops it before.
func newRPCModule(stack *node.Node, service *gossip.Service, configuration *config.Config) (*rpc.Module, error) {
	moduleConfig := rpcModuleConfig(configuration)
	moduleConfig.ClientVersion = stack.Server().Name
	takeOverRPC(stack.Config(), &moduleConfig)

	module := rpc.New(moduleConfig, service.RPCCore(), prometheus.DefaultRegisterer)
	handler, err := stack.RPCHandler()
	if err != nil {
		return nil, fmt.Errorf("failed to reach the node's RPC handler: %w", err)
	}
	module.Forward(nodeForwarder{client: gethrpc.DialInProc(handler)})
	stack.RegisterLifecycle(module)
	return module, nil
}

// rpcModuleConfig maps the node configuration into the module's.
func rpcModuleConfig(configuration *config.Config) rpc.Config {
	opera := configuration.Opera
	moduleConfig := rpc.DefaultConfig()
	moduleConfig.Dispatch.RequestTimeout = opera.RPCTimeout

	limits := &moduleConfig.Exec
	limits.RPCGasCap = opera.RPCGasCap
	limits.RPCEVMTimeout = opera.RPCEVMTimeout
	limits.RPCTxFeeCap = opera.RPCTxFeeCap
	limits.AllowUnprotectedTxs = opera.AllowUnprotectedTxs
	limits.MaxResponseSize = opera.MaxResponseSize
	limits.StructLogLimit = opera.StructLogLimit
	limits.MaxTraceFilterRange = opera.MaxTraceFilterRange
	limits.AllowJSTracers = opera.AllowJSTracers
	limits.LogQueryParameterLimit = opera.FilterAPI.LogQueryParameterLimit
	limits.LogQueryResultLimit = opera.FilterAPI.LogQueryResultLimit
	limits.IndexedLogsBlockRangeLimit = uint64(opera.FilterAPI.IndexedLogsBlockRangeLimit)
	limits.UnindexedLogsBlockRangeLimit = uint64(opera.FilterAPI.UnindexedLogsBlockRangeLimit)
	limits.TxSyncDefaultTimeout = opera.TxSyncDefaultTimeout
	limits.TxSyncMaxTimeout = opera.TxSyncMaxTimeout
	moduleConfig.Oracle = oracle.Config(opera.GPO)
	return moduleConfig
}

// takeOverRPC moves the HTTP, WebSocket and IPC endpoints from the node's
// configuration to the module's, which serves them; the node then starts no
// server of its own. WebSocket shares the HTTP server when their ports
// match, and the shared server listens on the WebSocket host, as
// go-ethereum's node decides (Node.wsServerForPort).
//
// Taking over rewrites the configuration everything in the node reads, so its
// readers learn from the module whether external RPC is served: see
// externalRPCServed.
func takeOverRPC(nodeConfig *node.Config, moduleConfig *rpc.Config) {
	timeouts := nodeTimeouts(nodeConfig)
	batch := transport.BatchLimits{
		RequestLimit:    nodeConfig.BatchRequestLimit,
		ResponseMaxSize: nodeConfig.BatchResponseMaxSize,
	}
	if nodeConfig.HTTPHost != "" {
		moduleConfig.HTTP = &rpc.HTTPEndpoint{
			Address:    net.JoinHostPort(nodeConfig.HTTPHost, strconv.Itoa(nodeConfig.HTTPPort)),
			Namespaces: nodeConfig.HTTPModules,
			Timeouts:   timeouts,
			HTTPConfig: transport.HTTPConfig{
				PathPrefix:   nodeConfig.HTTPPathPrefix,
				CORSOrigins:  nodeConfig.HTTPCors,
				VirtualHosts: nodeConfig.HTTPVirtualHosts,
				Batch:        batch,
			},
		}
	}
	if nodeConfig.WSHost != "" {
		address := net.JoinHostPort(nodeConfig.WSHost, strconv.Itoa(nodeConfig.WSPort))
		if moduleConfig.HTTP != nil && nodeConfig.WSPort == nodeConfig.HTTPPort {
			moduleConfig.HTTP.Address = address
		}
		moduleConfig.WebSocket = &rpc.WebSocketEndpoint{
			Address:    address,
			Namespaces: nodeConfig.WSModules,
			Timeouts:   timeouts,
			WebSocketConfig: transport.WebSocketConfig{
				PathPrefix: nodeConfig.WSPathPrefix,
				Origins:    nodeConfig.WSOrigins,
				Batch:      batch,
			},
		}
	}
	if path := nodeConfig.IPCEndpoint(); path != "" {
		moduleConfig.IPC = &rpc.IPCEndpoint{Path: path, Batch: batch}
	}
	nodeConfig.HTTPHost = ""
	nodeConfig.WSHost = ""
	nodeConfig.IPCPath = ""
}

// nodeTimeouts are the server timeouts of the node's configuration, checked
// as the node checks them; no timeouts stay no timeouts.
func nodeTimeouts(nodeConfig *node.Config) transport.Timeouts {
	timeouts := nodeConfig.HTTPTimeouts
	if timeouts != (gethrpc.HTTPTimeouts{}) {
		node.CheckTimeouts(&timeouts)
	}
	return transport.Timeouts{
		Read:       timeouts.ReadTimeout,
		ReadHeader: timeouts.ReadHeaderTimeout,
		Write:      timeouts.WriteTimeout,
		Idle:       timeouts.IdleTimeout,
	}
}

// externalRPCServed tells whether the node or the module serves RPC that
// other processes can reach, what the node's configuration said before the
// module took endpoints over.
func externalRPCServed(stack *node.Node, module *rpc.Module) bool {
	return stack.Config().ExtRPCEnabled() || module.ServesExternally()
}

// httpEndpoint is the URL of the HTTP endpoint, the module's or the node's.
func httpEndpoint(stack *node.Node, module *rpc.Module) string {
	if endpoint := module.HTTPEndpoint(); endpoint != "" {
		return endpoint
	}
	return stack.HTTPEndpoint()
}

// nodeForwarder hands the calls the module does not serve to the node's own
// handler.
type nodeForwarder struct {
	client *gethrpc.Client
}

func (forwarder nodeForwarder) Forward(ctx context.Context, method string, parameters []json.RawMessage) (json.RawMessage, error) {
	arguments := make([]any, len(parameters))
	for index, parameter := range parameters {
		arguments[index] = parameter
	}
	var result json.RawMessage
	if err := forwarder.client.CallContext(ctx, &result, method, arguments...); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return json.RawMessage("null"), nil
	}
	return result, nil
}
