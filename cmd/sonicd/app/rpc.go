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
	"math/big"

	"github.com/0xsoniclabs/sonic/api/legacy"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/node"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/rpc"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/transport"
)

// rpcEndpoints are the endpoints of the node configuration the module takes
// over, and whether external clients reach the node over them.
type rpcEndpoints struct {
	transport transport.Config
	modules   map[transport.Kind][]string
	external  bool
}

// takeOverRPC moves the node's HTTP, WebSocket and IPC endpoints to the RPC
// module: it records them and blanks them in the node configuration, so that
// the node opens none of them. Whatever the node derived from these fields is
// answered by the endpoints returned here: whether external RPC is served,
// and the addresses the node reports.
func takeOverRPC(cfg *node.Config) rpcEndpoints {
	ep := rpcEndpoints{
		modules: map[transport.Kind][]string{
			transport.HTTP: cfg.HTTPModules,
			transport.WS:   cfg.WSModules,
		},
		external: cfg.ExtRPCEnabled(),
		transport: transport.Config{
			IPC: cfg.IPCEndpoint(),
			Limits: transport.Limits{
				BatchRequestLimit:    cfg.BatchRequestLimit,
				BatchResponseMaxSize: cfg.BatchResponseMaxSize,
			},
		},
	}
	if cfg.HTTPHost != "" {
		t := cfg.HTTPTimeouts
		d := gethrpc.DefaultHTTPTimeouts
		ep.transport.HTTP = &transport.HTTPConfig{
			Host:             cfg.HTTPHost,
			Port:             cfg.HTTPPort,
			Cors:             cfg.HTTPCors,
			Vhosts:           cfg.HTTPVirtualHosts,
			Prefix:           cfg.HTTPPathPrefix,
			Timeouts:         transport.Timeouts{Read: t.ReadTimeout, ReadHeader: t.ReadHeaderTimeout, Write: t.WriteTimeout, Idle: t.IdleTimeout},
			FallbackTimeouts: transport.Timeouts{Read: d.ReadTimeout, ReadHeader: d.ReadHeaderTimeout, Write: d.WriteTimeout, Idle: d.IdleTimeout},
		}
	}
	if cfg.WSHost != "" {
		ep.transport.WS = &transport.WSConfig{
			Host:    cfg.WSHost,
			Port:    cfg.WSPort,
			Origins: cfg.WSOrigins,
			Prefix:  cfg.WSPathPrefix,
		}
	}
	cfg.HTTPHost = ""
	cfg.WSHost = ""
	cfg.IPCPath = ""
	return ep
}

// rpcConfig fills the module's configuration from the node's.
func rpcConfig(cfg *config.Config, ep rpcEndpoints, stack *node.Node) rpc.Config {
	opera := cfg.Opera
	rc := rpc.DefaultConfig()
	rc.Dispatch.RequestTimeout = opera.RPCTimeout
	rc.Exec.GasCap = opera.RPCGasCap
	rc.Exec.EVMTimeout = opera.RPCEVMTimeout
	rc.Exec.TxFeeCap = opera.RPCTxFeeCap
	rc.Exec.AllowUnprotectedTxs = opera.AllowUnprotectedTxs
	rc.Exec.TxSyncDefaultTimeout = opera.TxSyncDefaultTimeout
	rc.Exec.TxSyncMaxTimeout = opera.TxSyncMaxTimeout
	rc.Exec.MaxResponseSize = opera.MaxResponseSize
	rc.Exec.StructLogLimit = opera.StructLogLimit
	rc.Exec.AllowJSTracers = opera.AllowJSTracers
	rc.Exec.MaxTraceFilterRange = opera.MaxTraceFilterRange
	rc.Exec.LogQueryParameterLimit = opera.FilterAPI.LogQueryParameterLimit
	rc.Exec.LogQueryResultLimit = opera.FilterAPI.LogQueryResultLimit
	rc.Exec.IndexedLogsBlockRangeLimit = uint64(opera.FilterAPI.IndexedLogsBlockRangeLimit)
	rc.Exec.UnindexedLogsBlockRangeLimit = uint64(opera.FilterAPI.UnindexedLogsBlockRangeLimit)
	rc.Oracle.MaxGasPrice = opera.GPO.MaxGasPrice
	rc.Oracle.MinGasPrice = opera.GPO.MinGasPrice
	rc.Oracle.DefaultCertainty = opera.GPO.DefaultCertainty
	rc.Transport = ep.transport
	rc.Front = transport.FrontConfig{
		Modules:           ep.modules,
		Forward:           nodeForwarder(stack),
		ForwardEverywhere: []string{"net_listening", "net_peerCount"},
	}
	rc.ClientVersion = stack.Server().Name
	return rc
}

// startRPCModule creates the module over the node's core and registers it as
// a lifecycle of the node after the gossip service: it starts after the
// service and stops before it, leaving the service's feeds before they close.
func startRPCModule(cfg *config.Config, ep rpcEndpoints, stack *node.Node, svc *gossip.Service) *rpc.Module {
	svc.EthAPI.SetExtRPCEnabled(ep.external)
	module := rpc.New(svc.RPCCore(), rpcConfig(cfg, ep, stack), prometheus.DefaultRegisterer)
	svc.EthAPI.SetWalletDefaults(moduleWallet{module})
	stack.RegisterLifecycle(module)
	return module
}

// nodeForwarder answers a call with the node's own handler, for the
// namespaces that stay with the node.
func nodeForwarder(stack *node.Node) transport.Forwarder {
	return func(ctx context.Context, call dispatch.Call) (json.RawMessage, error) {
		handler, err := stack.RPCHandler()
		if err != nil {
			return nil, err
		}
		client := gethrpc.DialInProc(handler)
		defer client.Close()
		var params []json.RawMessage
		if len(call.Params) > 0 && string(call.Params) != "null" {
			if err := json.Unmarshal(call.Params, &params); err != nil {
				return nil, &dispatch.Error{Code: dispatch.CodeInvalidParams, Message: "non-array args"}
			}
		}
		args := make([]any, len(params))
		for i, p := range params {
			args[i] = p
		}
		var result json.RawMessage
		if err := client.CallContext(ctx, &result, call.Method, args...); err != nil {
			return nil, forwardedError(err)
		}
		if result == nil {
			result = json.RawMessage("null")
		}
		return result, nil
	}
}

// forwardedError keeps the code and data of an error of the node's handler.
func forwardedError(err error) error {
	var coded gethrpc.Error
	if !errors.As(err, &coded) {
		return err
	}
	e := &dispatch.Error{Code: coded.ErrorCode(), Message: coded.Error()}
	var withData gethrpc.DataError
	if errors.As(err, &withData) {
		e.Data = withData.ErrorData()
	}
	return e
}

// moduleChainReader reads the chain through the module, for the node's own
// clients of its RPC, such as the self-derivation of hardware wallets.
type moduleChainReader struct {
	module *rpc.Module
}

func blockArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	return hexutil.EncodeBig(number)
}

func (r moduleChainReader) BalanceAt(ctx context.Context, account common.Address, number *big.Int) (*big.Int, error) {
	var result hexutil.Big
	err := r.module.Call(ctx, &result, "eth_getBalance", account, blockArg(number))
	return (*big.Int)(&result), err
}

func (r moduleChainReader) StorageAt(ctx context.Context, account common.Address, key common.Hash, number *big.Int) ([]byte, error) {
	var result hexutil.Bytes
	err := r.module.Call(ctx, &result, "eth_getStorageAt", account, key, blockArg(number))
	return result, err
}

func (r moduleChainReader) CodeAt(ctx context.Context, account common.Address, number *big.Int) ([]byte, error) {
	var result hexutil.Bytes
	err := r.module.Call(ctx, &result, "eth_getCode", account, blockArg(number))
	return result, err
}

func (r moduleChainReader) NonceAt(ctx context.Context, account common.Address, number *big.Int) (uint64, error) {
	var result hexutil.Uint64
	err := r.module.Call(ctx, &result, "eth_getTransactionCount", account, blockArg(number))
	return uint64(result), err
}

// moduleWallet answers the defaults of the legacy wallet methods of the node's
// handler with the module: its fee suggestion and its gas estimate (ADR5).
type moduleWallet struct {
	module *rpc.Module
}

func (w moduleWallet) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	var tip hexutil.Big
	if err := w.module.Call(ctx, &tip, "eth_maxPriorityFeePerGas"); err != nil {
		return nil, err
	}
	return tip.ToInt(), nil
}

func (w moduleWallet) EstimateGas(ctx context.Context, args legacy.TransactionArgs, block gethrpc.BlockNumberOrHash) (hexutil.Uint64, error) {
	var ref any = gethrpc.LatestBlockNumber
	if number, ok := block.Number(); ok {
		ref = number
	} else if hash, ok := block.Hash(); ok {
		ref = hash
	}
	var gas hexutil.Uint64
	err := w.module.Call(ctx, &gas, "eth_estimateGas", args, ref)
	return gas, err
}
