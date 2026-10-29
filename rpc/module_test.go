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

package rpc

import (
	"context"
	"encoding/json"
	"net"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/0xsoniclabs/sonic/rpc/transport"
)

func newTestModule(t *testing.T, address string) *Module {
	t.Helper()
	config := DefaultConfig()
	config.ClientVersion = "Sonic/test"
	config.HTTP = &HTTPEndpoint{Address: address, HTTPConfig: transport.HTTPConfig{VirtualHosts: []string{"localhost"}}}
	return New(config, tests.NewFakeCore(t), prometheus.NewRegistry())
}

func TestModule_ServesHTTPUntilItStops(t *testing.T) {
	module := newTestModule(t, "127.0.0.1:0")
	require.True(t, module.ServesExternally())
	require.NoError(t, module.Start())
	client, err := gethrpc.Dial(module.HTTPEndpoint())
	require.NoError(t, err)
	defer client.Close()
	var version string
	require.NoError(t, client.Call(&version, "web3_clientVersion"))
	require.Equal(t, "Sonic/test", version)

	require.NoError(t, module.Stop())
	// Another server may take the freed port; it is not this module.
	err = client.Call(&version, "web3_clientVersion")
	require.True(t, err != nil || version != "Sonic/test", "the module answers after it stopped")
}

func TestModule_WithoutEndpointsServesNothingExternally(t *testing.T) {
	module := New(DefaultConfig(), tests.NewFakeCore(t), prometheus.NewRegistry())
	require.False(t, module.ServesExternally())
	require.Empty(t, module.HTTPEndpoint())
	require.NoError(t, module.Start())
	require.NoError(t, module.Stop())
}

func TestModule_StartFailsOnATakenAddress(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { require.NoError(t, listener.Close()) }()
	module := newTestModule(t, listener.Addr().String())
	require.ErrorContains(t, module.Start(), "failed to start the HTTP endpoint")
}

func TestModule_StartFailsWithoutAHead(t *testing.T) {
	fake := tests.NewFakeCore(t)
	fake.SetMode("Head", tests.Fail)
	module := New(DefaultConfig(), fake, prometheus.NewRegistry())
	require.ErrorContains(t, module.Start(), "failed to start the RPC module")
}

type stubForwarder struct{}

func (stubForwarder) Forward(context.Context, string, []json.RawMessage) (json.RawMessage, error) {
	return json.RawMessage(`"from the node"`), nil
}

func TestModule_ForwardsTheLegacyNamespacesOnIPCOnly(t *testing.T) {
	config := DefaultConfig()
	config.HTTP = &HTTPEndpoint{Address: "127.0.0.1:0", HTTPConfig: transport.HTTPConfig{VirtualHosts: []string{"localhost"}}}
	config.IPC = &IPCEndpoint{Path: filepath.Join(t.TempDir(), "sonic.ipc")}
	module := New(config, tests.NewFakeCore(t), prometheus.NewRegistry())
	module.Forward(stubForwarder{})
	require.NoError(t, module.Start())
	defer func() { require.NoError(t, module.Stop()) }()
	httpClient, err := gethrpc.Dial(module.HTTPEndpoint())
	require.NoError(t, err)
	defer httpClient.Close()
	ipcClient, err := gethrpc.DialIPC(t.Context(), config.IPC.Path)
	require.NoError(t, err)
	defer ipcClient.Close()

	var answer string
	require.NoError(t, ipcClient.Call(&answer, "dag_getHeads"))
	require.Equal(t, "from the node", answer)
	var rpcError gethrpc.Error
	require.ErrorAs(t, httpClient.Call(&answer, "dag_getHeads"), &rpcError)
	require.Equal(t, dispatch.CodeMethodNotFound, rpcError.ErrorCode())
	answer = ""
	require.NoError(t, httpClient.Call(&answer, "net_peerCount"))
	require.Equal(t, "from the node", answer, "a p2p fact")
}

// TestModule_ServesNoFtmNamespace holds D7: the ftm twin of the eth namespace
// is served on no transport, also where the configuration still names it,
// and rpc_modules does not list it.
func TestModule_ServesNoFtmNamespace(t *testing.T) {
	config := DefaultConfig()
	config.HTTP = &HTTPEndpoint{Address: "127.0.0.1:0", Namespaces: []string{"eth", "ftm"}, HTTPConfig: transport.HTTPConfig{VirtualHosts: []string{"localhost"}}}
	config.IPC = &IPCEndpoint{Path: filepath.Join(t.TempDir(), "sonic.ipc")}
	module := New(config, tests.NewFakeCore(t), prometheus.NewRegistry())
	for _, method := range module.dispatcher.Methods() {
		require.False(t, strings.HasPrefix(method, "ftm_"), "%s is registered", method)
	}
	require.NoError(t, module.Start())
	defer func() { require.NoError(t, module.Stop()) }()
	httpClient, err := gethrpc.Dial(module.HTTPEndpoint())
	require.NoError(t, err)
	defer httpClient.Close()
	ipcClient, err := gethrpc.DialIPC(t.Context(), config.IPC.Path)
	require.NoError(t, err)
	defer ipcClient.Close()

	for name, client := range map[string]*gethrpc.Client{"HTTP": httpClient, "IPC": ipcClient} {
		var chainID string
		require.NoError(t, client.Call(&chainID, "eth_chainId"), name)
		var rpcError gethrpc.Error
		require.ErrorAs(t, client.Call(&chainID, "ftm_chainId"), &rpcError, name)
		require.Equal(t, dispatch.CodeMethodNotFound, rpcError.ErrorCode(), name)
		var modules map[string]string
		require.NoError(t, client.Call(&modules, "rpc_modules"), name)
		require.Contains(t, modules, "eth", name)
		require.NotContains(t, modules, "ftm", name)
	}
	_, err = ipcClient.Subscribe(t.Context(), "ftm", make(chan any), "newHeads")
	require.ErrorContains(t, err, `no "newHeads" subscription in ftm namespace`)
}

func TestModule_CallServesAClientInTheProcess(t *testing.T) {
	module := New(DefaultConfig(), tests.NewFakeCore(t), prometheus.NewRegistry())
	require.NoError(t, module.Start())
	defer func() { require.NoError(t, module.Stop()) }()
	var hash string
	require.NoError(t, module.Call(t.Context(), &hash, "web3_sha3", "0x"))
	require.Equal(t, "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470", hash)

	var dispatchError *dispatch.Error
	require.ErrorAs(t, module.Call(t.Context(), &hash, "web3_none"), &dispatchError)
	require.Equal(t, dispatch.CodeMethodNotFound, dispatchError.Code)
	require.ErrorContains(t, module.Call(t.Context(), &hash, "web3_sha3", func() {}), "unsupported type")
	require.Error(t, module.Call(t.Context(), new(int), "web3_sha3", "0x"), "a result that does not decode")
}

// inventory is the method inventory of section 6 of the plan: the methods
// the module ports or answers locally. The eth ones are mirrored as ftm.
var inventory = struct {
	eth, others []string
}{
	eth: []string{
		"chainId", "blockNumber", "syncing",
		"gasPrice", "maxPriorityFeePerGas", "feeHistory", "blobBaseFee",
		"getBalance", "getTransactionCount", "getCode", "getStorageAt", "getAccount", "getProof",
		"call", "estimateGas", "createAccessList", "simulateV1",
		"getBlockByNumber", "getBlockByHash", "getHeaderByNumber", "getHeaderByHash", "getBlockReceipts",
		"getBlockTransactionCountByNumber", "getBlockTransactionCountByHash",
		"getUncleByBlockNumberAndIndex", "getUncleByBlockHashAndIndex", "getUncleCountByBlockNumber", "getUncleCountByBlockHash",
		"getTransactionByBlockNumberAndIndex", "getTransactionByBlockHashAndIndex",
		"getRawTransactionByBlockNumberAndIndex", "getRawTransactionByBlockHashAndIndex",
		"getTransactionByHash", "getRawTransactionByHash", "getTransactionReceipt",
		"sendRawTransaction", "sendRawTransactionSync", "pendingTransactions",
		"newFilter", "newBlockFilter", "newPendingTransactionFilter", "getFilterChanges", "getFilterLogs", "uninstallFilter",
		"getLogs", "subscribe",
		"getRules", "currentEpoch", "config",
		"etherbase", "coinbase", "hashrate",
	},
	others: []string{
		"txpool_content", "txpool_contentFrom", "txpool_status", "txpool_inspect",
		"sonic_estimateGasForTransactions", "sonic_prepareBundle", "sonic_submitBundle", "sonic_getBundleInfo",
		"debug_getBlockRlp", "debug_printBlock", "debug_traceTransaction", "debug_traceBlockByNumber", "debug_traceBlockByHash", "debug_traceCall",
		"trace_transaction", "trace_get", "trace_block", "trace_call", "trace_callMany", "trace_filter",
		"net_version", "web3_clientVersion", "web3_sha3",
	},
}

// profiling are go-ethereum's profiling methods, which the module answers as
// not enabled in a build without enable_debug.
var profiling = []string{
	"debug_verbosity", "debug_vmodule", "debug_memStats", "debug_gcStats", "debug_cpuProfile",
	"debug_startCPUProfile", "debug_stopCPUProfile", "debug_goTrace", "debug_startGoTrace",
	"debug_stopGoTrace", "debug_blockProfile", "debug_setBlockProfileRate", "debug_writeBlockProfile",
	"debug_mutexProfile", "debug_setMutexProfileFraction", "debug_writeMutexProfile",
	"debug_writeMemProfile", "debug_stacks", "debug_freeOSMemory", "debug_setGCPercent", "debug_setMemoryLimit",
}

// TestModule_RegistersExactlyTheInventory holds S1 and X with section 6 of
// the plan: the module registers the inventory, and nothing else.
func TestModule_RegistersExactlyTheInventory(t *testing.T) {
	module := New(DefaultConfig(), tests.NewFakeCore(t), prometheus.NewRegistry())
	expected := slices.Clone(inventory.others)
	for _, name := range inventory.eth {
		expected = append(expected, "eth_"+name)
	}
	if module.dispatcher.Has(profiling[0]) {
		expected = append(expected, profiling...)
	}
	registered := module.dispatcher.Methods()
	slices.Sort(expected)
	slices.Sort(registered)
	require.Equal(t, expected, registered)
}
