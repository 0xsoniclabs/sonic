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
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/ethereum/go-ethereum/node"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/rpc"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
	"github.com/0xsoniclabs/sonic/rpc/transport"
)

func TestTakeOverRPC_TakesEveryEndpoint(t *testing.T) {
	nodeConfig := node.Config{
		DataDir:              "/data",
		HTTPHost:             "127.0.0.1",
		HTTPPort:             8545,
		HTTPModules:          []string{"eth", "net"},
		HTTPCors:             []string{"https://example.com"},
		HTTPVirtualHosts:     []string{"localhost"},
		HTTPPathPrefix:       "/rpc",
		HTTPTimeouts:         gethrpc.HTTPTimeouts{ReadTimeout: 2 * time.Second, ReadHeaderTimeout: 3 * time.Second, WriteTimeout: 4 * time.Second, IdleTimeout: 5 * time.Second},
		BatchRequestLimit:    7,
		BatchResponseMaxSize: 8,
		WSHost:               "127.0.0.2",
		WSPort:               8546,
		WSModules:            []string{"eth"},
		WSOrigins:            []string{"https://wallet.example"},
		WSPathPrefix:         "/ws",
		IPCPath:              "sonic.ipc",
	}
	var moduleConfig rpc.Config
	takeOverRPC(&nodeConfig, &moduleConfig)
	batch := transport.BatchLimits{RequestLimit: 7, ResponseMaxSize: 8}
	timeouts := transport.Timeouts{Read: 2 * time.Second, ReadHeader: 3 * time.Second, Write: 4 * time.Second, Idle: 5 * time.Second}
	require.Equal(t, &rpc.HTTPEndpoint{
		Address:    "127.0.0.1:8545",
		Namespaces: []string{"eth", "net"},
		Timeouts:   timeouts,
		HTTPConfig: transport.HTTPConfig{PathPrefix: "/rpc", CORSOrigins: []string{"https://example.com"}, VirtualHosts: []string{"localhost"}, Batch: batch},
	}, moduleConfig.HTTP)
	require.Equal(t, &rpc.WebSocketEndpoint{
		Address:         "127.0.0.2:8546",
		Namespaces:      []string{"eth"},
		Timeouts:        timeouts,
		WebSocketConfig: transport.WebSocketConfig{PathPrefix: "/ws", Origins: []string{"https://wallet.example"}, Batch: batch},
	}, moduleConfig.WebSocket)
	require.Equal(t, &rpc.IPCEndpoint{Path: "/data/sonic.ipc", Batch: batch}, moduleConfig.IPC)
	require.Empty(t, nodeConfig.HTTPHost, "the node starts no server of its own")
	require.Empty(t, nodeConfig.WSHost)
	require.Empty(t, nodeConfig.IPCPath)
	require.False(t, nodeConfig.ExtRPCEnabled(), "the module, not the node's configuration, tells whether RPC is external")
}

// TestTakeOverRPC_SharesTheHTTPServerWhenThePortsMatch: the shared server
// listens on the WebSocket host, as go-ethereum's node decides.
func TestTakeOverRPC_SharesTheHTTPServerWhenThePortsMatch(t *testing.T) {
	for name, wsHost := range map[string]string{"same host": "127.0.0.1", "another host": "0.0.0.0"} {
		nodeConfig := node.Config{HTTPHost: "127.0.0.1", HTTPPort: 8545, WSHost: wsHost, WSPort: 8545}
		var moduleConfig rpc.Config
		takeOverRPC(&nodeConfig, &moduleConfig)
		require.Equal(t, wsHost+":8545", moduleConfig.HTTP.Address, name)
		require.Equal(t, moduleConfig.HTTP.Address, moduleConfig.WebSocket.Address, name)
	}
}

func TestTakeOverRPC_TakesNothingWithoutEndpoints(t *testing.T) {
	nodeConfig := node.Config{}
	var moduleConfig rpc.Config
	takeOverRPC(&nodeConfig, &moduleConfig)
	require.Nil(t, moduleConfig.HTTP)
	require.Nil(t, moduleConfig.WebSocket)
	require.Nil(t, moduleConfig.IPC)
}

func TestTakeOverRPC_SanitizesTimeoutsAsTheNode(t *testing.T) {
	nodeConfig := node.Config{HTTPHost: "127.0.0.1", HTTPTimeouts: gethrpc.HTTPTimeouts{ReadTimeout: time.Millisecond}}
	var moduleConfig rpc.Config
	takeOverRPC(&nodeConfig, &moduleConfig)
	require.Equal(t, gethrpc.DefaultHTTPTimeouts.WriteTimeout, moduleConfig.HTTP.Timeouts.Write)
	require.Equal(t, gethrpc.DefaultHTTPTimeouts.ReadTimeout, moduleConfig.HTTP.Timeouts.Read)

	nodeConfig = node.Config{HTTPHost: "127.0.0.1"}
	takeOverRPC(&nodeConfig, &moduleConfig)
	require.Equal(t, transport.Timeouts{}, moduleConfig.HTTP.Timeouts, "no timeouts stay no timeouts")
}

// TestRPCModuleConfig_MapsEveryKnob holds O4 for the mapping into the
// module's configuration: each knob set in the node configuration reaches
// the module.
func TestRPCModuleConfig_MapsEveryKnob(t *testing.T) {
	opera := gossip.DefaultConfig(cachescale.Identity)
	opera.RPCTimeout = 11 * time.Second
	opera.RPCGasCap = 12
	opera.RPCEVMTimeout = 13 * time.Second
	opera.RPCTxFeeCap = 14
	opera.AllowUnprotectedTxs = true
	opera.MaxResponseSize = 15
	opera.StructLogLimit = 16
	opera.MaxTraceFilterRange = 17
	opera.AllowJSTracers = true
	opera.FilterAPI.LogQueryParameterLimit = 18
	opera.FilterAPI.LogQueryResultLimit = 19
	opera.FilterAPI.IndexedLogsBlockRangeLimit = idx.Block(20)
	opera.FilterAPI.UnindexedLogsBlockRangeLimit = idx.Block(21)
	opera.TxSyncDefaultTimeout = 22 * time.Second
	opera.TxSyncMaxTimeout = 23 * time.Second
	opera.GPO.MaxGasPrice = big.NewInt(24)
	opera.GPO.MinGasPrice = big.NewInt(25)
	opera.GPO.DefaultCertainty = 26

	mapped := rpcModuleConfig(&config.Config{Opera: opera})
	require.Equal(t, 11*time.Second, mapped.Dispatch.RequestTimeout)
	limits := mapped.Exec
	require.Equal(t, uint64(12), limits.RPCGasCap)
	require.Equal(t, 13*time.Second, limits.RPCEVMTimeout)
	require.Equal(t, 14.0, limits.RPCTxFeeCap)
	require.True(t, limits.AllowUnprotectedTxs)
	require.Equal(t, 15, limits.MaxResponseSize)
	require.Equal(t, 16, limits.StructLogLimit)
	require.Equal(t, uint64(17), limits.MaxTraceFilterRange)
	require.True(t, limits.AllowJSTracers)
	require.Equal(t, uint(18), limits.LogQueryParameterLimit)
	require.Equal(t, uint(19), limits.LogQueryResultLimit)
	require.Equal(t, uint64(20), limits.IndexedLogsBlockRangeLimit)
	require.Equal(t, uint64(21), limits.UnindexedLogsBlockRangeLimit)
	require.Equal(t, 22*time.Second, limits.TxSyncDefaultTimeout)
	require.Equal(t, 23*time.Second, limits.TxSyncMaxTimeout)
	require.Equal(t, oracle.Config{MaxGasPrice: big.NewInt(24), MinGasPrice: big.NewInt(25), DefaultCertainty: 26}, mapped.Oracle)
}

// TestUnlock_IsRefusedWithHTTPTakenOverByTheModule holds O5: once the module
// took HTTP over, the node's configuration no longer says that external RPC
// is served, and the refusal of the reference still applies.
func TestUnlock_IsRefusedWithHTTPTakenOverByTheModule(t *testing.T) {
	datadir := tmpDatadirWithKeystore(t)
	initFakenetDatadir(datadir, 1)
	cli := exec(t,
		"--fakenet", "0/1", "--datadir", datadir, "--nat", "none", "--nodiscover", "--maxpeers", "0", "--port", "0",
		"--http", "--http.addr", "127.0.0.1", "--http.port", "0", "--ipcdisable",
		"--unlock", "f466859ead1932d743d622cb74fc058882e8648a")
	cli.ExpectExit()
	require.Contains(t, cli.StderrText(), "account unlock with HTTP access is forbidden")
}

// startNode runs a fake node in this process and returns its HTTP endpoint;
// the node stops with the test.
func startTestNode(t *testing.T, arguments ...string) string {
	t.Helper()
	datadir := tmpdir(t)
	initFakenetDatadir(datadir, 1)
	endpoints := make(chan string, 1)
	shutdown := make(chan struct{})
	done := make(chan error, 1)
	base := []string{"sonicd", "--fakenet", "1/1", "--datadir", datadir, "--nat", "none", "--nodiscover",
		"--maxpeers", "0", "--port", "0", "--ipcdisable", "--datadir.minfreedisk", "0"}
	go func() {
		done <- RunWithArgs(append(base, arguments...), &AppControl{HttpPortAnnouncement: endpoints, Shutdown: shutdown})
	}()
	t.Cleanup(func() {
		close(shutdown)
		require.NoError(t, <-done)
	})
	select {
	case endpoint := <-endpoints:
		return endpoint
	case err := <-done:
		t.Fatalf("the node stopped: %v", err)
	case <-time.After(time.Minute):
		t.Fatal("the node did not announce its HTTP endpoint")
	}
	return ""
}

func TestNode_ServesHTTPThroughTheModule(t *testing.T) {
	endpoint := startTestNode(t, "--http", "--http.addr", "127.0.0.1", "--http.port", "0", "--http.api", "eth,net,web3")
	require.True(t, strings.HasPrefix(endpoint, "http://127.0.0.1:"))
	client, err := gethrpc.Dial(endpoint)
	require.NoError(t, err)
	defer client.Close()

	var chainID string
	require.NoError(t, client.Call(&chainID, "eth_chainId"))
	require.Equal(t, "0xfa3", chainID)

	var balance string
	require.NoError(t, client.Call(&balance, "eth_getBalance", "0x0000000000000000000000000000000000000000", "latest"), "a method the module forwards")

	var listening bool
	require.NoError(t, client.Call(&listening, "net_listening"), "a p2p fact of the node")
	require.True(t, listening)

	var modules map[string]string
	require.NoError(t, client.Call(&modules, "rpc_modules"))
	require.Equal(t, map[string]string{"rpc": "1.0", "eth": "1.0", "net": "1.0", "web3": "1.0"}, modules)

	err = client.Call(new(json.RawMessage), "dag_getHeads", "latest")
	var rpcError gethrpc.Error
	require.ErrorAs(t, err, &rpcError)
	require.Equal(t, -32601, rpcError.ErrorCode(), "a namespace not enabled on HTTP")

	_, err = client.Subscribe(context.Background(), "eth", make(chan any), "newHeads")
	require.ErrorIs(t, err, gethrpc.ErrNotificationsUnsupported)
}
