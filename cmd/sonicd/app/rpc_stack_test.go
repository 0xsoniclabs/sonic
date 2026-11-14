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
	"math/big"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/gossip/gasprice"
	"github.com/0xsoniclabs/sonic/rpc"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	rpcexec "github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
	"github.com/0xsoniclabs/sonic/rpc/transport"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/ethereum/go-ethereum/node"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestChooseRPCStack_PicksTheConfiguredStack(t *testing.T) {
	cfg := &config.Config{}
	cfg.Opera.RPCStack = gossip.LegacyRPCStack
	require.IsType(t, legacyStack{}, chooseRPCStack(cfg))
	cfg.Opera.RPCStack = gossip.ModuleRPCStack
	require.IsType(t, &moduleStack{}, chooseRPCStack(cfg))
}

// One case per endpoint layout the node's configuration can hold.
func TestTakeOverRPC_MovesEveryEndpointToTheModule(t *testing.T) {
	timeouts := gethrpc.HTTPTimeouts{ReadTimeout: 1 * time.Second, ReadHeaderTimeout: 2 * time.Second, WriteTimeout: 3 * time.Second, IdleTimeout: 4 * time.Second}
	cases := map[string]struct {
		node node.Config
		want rpc.Config
	}{
		"nothing": {
			node: node.Config{},
			want: rpc.DefaultConfig(),
		},
		"http": {
			node: node.Config{
				HTTPHost: "0.0.0.0", HTTPPort: 8545, HTTPPathPrefix: "/rpc", HTTPCors: []string{"*"},
				HTTPVirtualHosts: []string{"a.example"}, HTTPModules: []string{"eth"}, HTTPTimeouts: timeouts,
				BatchRequestLimit: 10, BatchResponseMaxSize: 20,
			},
			want: func() rpc.Config {
				config := rpc.DefaultConfig()
				config.HTTP = &rpc.HTTPConfig{
					HTTPConfig: transport.HTTPConfig{
						Endpoint: "0.0.0.0:8545", PathPrefix: "/rpc", CorsAllowedOrigins: []string{"*"},
						VirtualHosts: []string{"a.example"},
						Timeouts:     transport.Timeouts{Read: time.Second, ReadHeader: 2 * time.Second, Write: 3 * time.Second, Idle: 4 * time.Second},
						Batch:        transport.BatchLimits{Items: 10, ResponseBytes: 20},
					},
					Modules: []string{"eth"},
				}
				return config
			}(),
		},
		"ipv6 http": {
			node: node.Config{HTTPHost: "::1", HTTPPort: 1},
			want: func() rpc.Config {
				config := rpc.DefaultConfig()
				config.HTTP = &rpc.HTTPConfig{HTTPConfig: transport.HTTPConfig{Endpoint: "[::1]:1"}}
				return config
			}(),
		},
		"ipc name in the data directory": {
			node: node.Config{DataDir: "/data", IPCPath: "sonic.ipc"},
			want: func() rpc.Config {
				config := rpc.DefaultConfig()
				config.IPCPath = filepath.Join("/data", "sonic.ipc")
				return config
			}(),
		},
		"ipc path": {
			node: node.Config{DataDir: "/data", IPCPath: "/run/sonic.ipc"},
			want: func() rpc.Config {
				config := rpc.DefaultConfig()
				config.IPCPath = "/run/sonic.ipc"
				return config
			}(),
		},
		"websocket is not served yet": {
			node: node.Config{WSHost: "127.0.0.1", WSPort: 8546},
			want: rpc.DefaultConfig(),
		},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			nodeConfig := test.node
			require.Equal(t, test.want, takeOverRPC(&nodeConfig))
			require.False(t, nodeConfig.ExtRPCEnabled(), "the node serves no external RPC")
			require.Empty(t, nodeConfig.IPCEndpoint(), "the node opens no IPC endpoint")
		})
	}
}

// Each key of the node's configuration the module reads reaches its field.
func TestSetModuleConfig_MapsEachKey(t *testing.T) {
	node := gossip.Config{
		RPCTimeout:           1 * time.Second,
		RPCGasCap:            2,
		RPCEVMTimeout:        3 * time.Second,
		GPO:                  gasprice.Config{MaxGasPrice: big.NewInt(4), MinGasPrice: big.NewInt(5), DefaultCertainty: 6},
		RPCTxFeeCap:          7,
		AllowUnprotectedTxs:  true,
		TxSyncDefaultTimeout: 8 * time.Second,
		TxSyncMaxTimeout:     9 * time.Second,
		EnableTestOnlyApi:    true,
	}
	var module rpc.Config
	setModuleConfig(&module, node)
	require.Equal(t, rpc.Config{
		Dispatch: dispatch.Config{RequestTimeout: 1 * time.Second},
		Exec: rpcexec.Config{RPCGasCap: 2, RPCEVMTimeout: 3 * time.Second, RPCTxFeeCap: 7, AllowUnprotectedTxs: true,
			TxSyncDefaultTimeout: 8 * time.Second, TxSyncMaxTimeout: 9 * time.Second, EnableTestOnlyApi: true},
		Oracle: oracle.Config{MaxGasPrice: big.NewInt(4), MinGasPrice: big.NewInt(5), DefaultCertainty: 6},
	}, module)
}

// The node's defaults of the keys the module reads are the module's own,
// as long as the node keeps its own literals for the legacy stack.
func TestSetModuleConfig_NodeDefaultsAreTheModulesDefaults(t *testing.T) {
	var module rpc.Config
	setModuleConfig(&module, gossip.DefaultConfig(cachescale.Identity))
	defaults := rpc.DefaultConfig()
	require.Equal(t, defaults.Dispatch, module.Dispatch)
	require.Equal(t, defaults.Exec, module.Exec)
	require.Equal(t, defaults.Oracle, module.Oracle)
}

// startFakenetNode runs a node of a one-validator fake network with the
// given arguments, waits for its IPC endpoint and answers a client of it.
func startFakenetNode(t *testing.T, args ...string) (*testcli, *gethrpc.Client) {
	t.Helper()
	dataDir := tmpdir(t)
	initFakenetDatadir(dataDir, 1)
	args = append([]string{"--fakenet", "1/1", "--datadir", dataDir, "--nat", "none", "--nodiscover",
		"--maxpeers", "0", "--port", "0"}, args...)
	cli := exec(t, args...)
	t.Cleanup(func() {
		cli.Interrupt()
		cli.WaitExit()
	})
	endpoint := filepath.Join(dataDir, "sonic.ipc")
	var client *gethrpc.Client
	require.Eventually(t, func() bool {
		var err error
		if client, err = gethrpc.Dial(endpoint); err != nil {
			return false
		}
		var modules map[string]string
		if client.Call(&modules, "rpc_modules") != nil {
			client.Close()
			return false
		}
		return true
	}, time.Minute, 100*time.Millisecond, "the node does not answer on IPC")
	t.Cleanup(client.Close)
	return cli, client
}

func TestRPCStack_ANodeServesTheModulesOfItsStack(t *testing.T) {
	cases := map[string]struct {
		args    []string
		modules []string
	}{
		"legacy by default": {nil, []string{"abft", "admin", "dag", "debug", "eth", "ftm", "net", "personal", "rpc", "sonic", "test", "trace", "txpool", "web3"}},
		"legacy":            {[]string{"--rpc.stack", "legacy"}, []string{"abft", "admin", "dag", "debug", "eth", "ftm", "net", "personal", "rpc", "sonic", "test", "trace", "txpool", "web3"}},
		"module":            {[]string{"--rpc.stack", "module"}, []string{"abft", "admin", "dag", "debug", "eth", "net", "rpc", "sonic", "test", "txpool", "web3"}},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			_, client := startFakenetNode(t, test.args...)
			var modules map[string]string
			require.NoError(t, client.Call(&modules, "rpc_modules"))
			var names []string
			for module := range modules {
				names = append(names, module)
			}
			require.ElementsMatch(t, test.modules, names)
		})
	}
}

func TestRPCStack_AnUnknownStackRefusesToStart(t *testing.T) {
	dataDir := tmpdir(t)
	cli := exec(t, "--datadir", dataDir, "--rpc.stack", "other")
	cli.ExpectExit()
	require.Contains(t, cli.StderrText(), `the RPC stack is "legacy" or "module", not "other"`)
}

func TestRPCStack_TheModuleStartsWithUnlockFlagsAndWarns(t *testing.T) {
	cli, client := startFakenetNode(t, "--rpc.stack", "module", "--http", "--http.port", "0",
		"--unlock", "0x0000000000000000000000000000000000000001", "--allow-insecure-unlock")
	var chainID string
	require.NoError(t, client.Call(&chainID, "eth_chainId"))
	cli.Interrupt()
	cli.WaitExit()
	stderr := cli.StderrText()
	for _, flag := range []string{"--unlock", "--allow-insecure-unlock"} {
		require.True(t, strings.Contains(stderr, "The "+flag+" flag has no effect with --rpc.stack module"), "no warning about %s", flag)
	}
}
