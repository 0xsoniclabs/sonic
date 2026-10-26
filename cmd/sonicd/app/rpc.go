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
	"github.com/0xsoniclabs/sonic/rpc/transport"
)

// newRPCModule creates the RPC module of the node, takes over the endpoints
// it serves and registers it with the node, which starts it after the
// gossip service and stops it before.
func newRPCModule(stack *node.Node, service *gossip.Service, configuration *config.Config) (*rpc.Module, error) {
	moduleConfig := rpcModuleConfig(configuration)
	moduleConfig.ClientVersion = stack.Server().Name
	moduleConfig.HTTP = takeOverRPC(stack.Config())

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
	return moduleConfig
}

// takeOverRPC moves the HTTP endpoint from the node's configuration to the
// module and returns it, nil when the module does not take it. The node then
// starts no HTTP server. An HTTP endpoint that shares its port with WebSocket
// stays with the node, whose WebSocket server it is, as long as WebSocket is
// the node's.
//
// Taking over rewrites the configuration everything in the node reads, so its
// readers learn from the module whether external RPC is served: see
// externalRPCServed.
func takeOverRPC(nodeConfig *node.Config) *rpc.HTTPEndpoint {
	if nodeConfig.HTTPHost == "" {
		return nil
	}
	if nodeConfig.WSHost != "" && nodeConfig.WSPort == nodeConfig.HTTPPort {
		return nil
	}
	timeouts := nodeConfig.HTTPTimeouts
	if timeouts != (gethrpc.HTTPTimeouts{}) {
		node.CheckTimeouts(&timeouts)
	}
	endpoint := &rpc.HTTPEndpoint{
		Address:    net.JoinHostPort(nodeConfig.HTTPHost, strconv.Itoa(nodeConfig.HTTPPort)),
		Namespaces: nodeConfig.HTTPModules,
		Timeouts: transport.Timeouts{
			Read:       timeouts.ReadTimeout,
			ReadHeader: timeouts.ReadHeaderTimeout,
			Write:      timeouts.WriteTimeout,
			Idle:       timeouts.IdleTimeout,
		},
		HTTPConfig: transport.HTTPConfig{
			PathPrefix:   nodeConfig.HTTPPathPrefix,
			CORSOrigins:  nodeConfig.HTTPCors,
			VirtualHosts: nodeConfig.HTTPVirtualHosts,
			Batch: transport.BatchLimits{
				RequestLimit:    nodeConfig.BatchRequestLimit,
				ResponseMaxSize: nodeConfig.BatchResponseMaxSize,
			},
		},
	}
	nodeConfig.HTTPHost = ""
	return endpoint
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
