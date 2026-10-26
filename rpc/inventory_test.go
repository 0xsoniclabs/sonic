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
	"go/ast"
	"go/token"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/0xsoniclabs/sonic/rpc/transport"
)

// testPackages are the packages whose tests a method's handler tests live
// in, by namespace.
var testPackages = map[string][]string{
	"eth":    {"rpc/eth", "rpc/filters"},
	"debug":  {"rpc/debug"},
	"trace":  {"rpc/trace"},
	"txpool": {"rpc/txpool"},
	"sonic":  {"rpc/sonic"},
	"net":    {"rpc/net"},
	"web3":   {"rpc/web3"},
	"rpc":    {"rpc/transport"},
}

// TestHandlerTests_NameEveryMethod is T4 and Q4: every method the module
// serves is named by a test of its namespace's package. The ftm mirror is the
// eth handlers registered again, checked by TestModule_TheFtmNamespaceMirrorsEth.
func TestHandlerTests_NameEveryMethod(t *testing.T) {
	named := map[string]map[string]bool{}
	for _, f := range moduleFiles(t) {
		if !f.test {
			continue
		}
		if named[f.pkg] == nil {
			named[f.pkg] = map[string]bool{}
		}
		ast.Inspect(f.ast, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if s, err := strconv.Unquote(lit.Value); err == nil {
					named[f.pkg][s] = true
				}
			}
			return true
		})
	}
	m := New(tests.NewFakeCore(), DefaultConfig(), prometheus.NewRegistry())
	for _, method := range m.d.Methods() {
		ns, _, _ := strings.Cut(method, "_")
		if ns == "ftm" {
			continue
		}
		pkgs, ok := testPackages[ns]
		require.True(t, ok, "no test package for namespace %s", ns)
		found := slices.ContainsFunc(pkgs, func(pkg string) bool { return named[pkg][method] })
		if !found {
			t.Errorf("no handler test names %s (looked in %v)", method, pkgs)
		}
	}
}

// inventory is section 6 as data: the methods the module serves, the eth
// methods mirrored as ftm.
var inventory = []string{
	// eth
	"eth_chainId", "eth_blockNumber", "eth_syncing", "eth_gasPrice", "eth_maxPriorityFeePerGas", "eth_feeHistory",
	"eth_blobBaseFee", "eth_getBalance", "eth_getTransactionCount", "eth_getCode", "eth_getStorageAt", "eth_getAccount",
	"eth_getProof", "eth_call", "eth_estimateGas", "eth_createAccessList", "eth_simulateV1", "eth_getBlockByNumber",
	"eth_getBlockByHash", "eth_getHeaderByNumber", "eth_getHeaderByHash", "eth_getBlockReceipts",
	"eth_getBlockTransactionCountByNumber", "eth_getBlockTransactionCountByHash", "eth_getUncleByBlockNumberAndIndex",
	"eth_getUncleByBlockHashAndIndex", "eth_getUncleCountByBlockNumber", "eth_getUncleCountByBlockHash",
	"eth_getTransactionByBlockNumberAndIndex", "eth_getTransactionByBlockHashAndIndex",
	"eth_getRawTransactionByBlockNumberAndIndex", "eth_getRawTransactionByBlockHashAndIndex", "eth_getTransactionByHash",
	"eth_getRawTransactionByHash", "eth_getTransactionReceipt", "eth_sendRawTransaction", "eth_sendRawTransactionSync",
	"eth_pendingTransactions", "eth_newFilter", "eth_newBlockFilter", "eth_newPendingTransactionFilter",
	"eth_getFilterChanges", "eth_getFilterLogs", "eth_uninstallFilter", "eth_getLogs", "eth_subscribe", "eth_unsubscribe",
	"eth_getRules", "eth_currentEpoch", "eth_config", "eth_etherbase", "eth_coinbase", "eth_hashrate",
	// txpool, sonic
	"txpool_content", "txpool_contentFrom", "txpool_status", "txpool_inspect",
	"sonic_estimateGasForTransactions", "sonic_prepareBundle", "sonic_submitBundle", "sonic_getBundleInfo",
	// debug, trace
	"debug_getBlockRlp", "debug_printBlock", "debug_traceTransaction", "debug_traceBlockByNumber", "debug_traceBlockByHash",
	"debug_traceCall", "trace_transaction", "trace_get", "trace_block", "trace_call", "trace_callMany", "trace_filter",
	// net, web3, rpc
	"net_version", "web3_clientVersion", "web3_sha3", "rpc_modules",
}

// profilingShadows are go-ethereum's profiling methods of the debug
// namespace, which the module answers as not enabled, as the reference does
// in a build without the enable_debug tag.
var profilingShadows = []string{
	"debug_verbosity", "debug_vmodule", "debug_memStats", "debug_gcStats", "debug_cpuProfile", "debug_startCPUProfile",
	"debug_stopCPUProfile", "debug_goTrace", "debug_startGoTrace", "debug_stopGoTrace", "debug_blockProfile",
	"debug_setBlockProfileRate", "debug_writeBlockProfile", "debug_mutexProfile", "debug_setMutexProfileFraction",
	"debug_writeMutexProfile", "debug_writeMemProfile", "debug_stacks", "debug_freeOSMemory", "debug_setGCPercent",
	"debug_setMemoryLimit",
}

// notServed is section X: the methods the module does not serve, with the
// reason.
var notServed = map[string]string{
	"eth_accounts":               "X1: no account manager",
	"eth_sign":                   "X1",
	"eth_signTransaction":        "X1",
	"eth_sendTransaction":        "X1",
	"eth_fillTransaction":        "X1",
	"eth_resend":                 "X1",
	"eth_getEpochBlock":          "X2",
	"eth_getEpochStats":          "X2",
	"debug_chaindbProperty":      "X1",
	"debug_chaindbCompact":       "X1",
	"debug_setHead":              "X1",
	"debug_getRawBlock":          "X6: the reference does not serve it",
	"debug_getRawHeader":         "X6",
	"debug_getRawReceipts":       "X6",
	"debug_getRawTransaction":    "X6",
	"debug_getBadBlocks":         "X6: the reference does not serve it; Sonic has no bad blocks",
	"dag_getEvent":               "X2",
	"dag_getEventPayload":        "X2",
	"dag_getHeads":               "X2",
	"abft_getValidators":         "X2",
	"abft_getEpochUptime":        "X2",
	"abft_getOriginatedEpochFee": "X2",
	"abft_getDowntime":           "X2",
	"personal_listAccounts":      "X1",
	"personal_sign":              "X1",
	"net_listening":              "the node's p2p facts, forwarded",
	"net_peerCount":              "the node's p2p facts, forwarded",
}

// gethMethods are the methods of go-ethereum's OpenRPC specification
// (ethereum/execution-apis) and of its txpool, net and web3 namespaces.
var gethMethods = []string{
	"eth_accounts", "eth_blobBaseFee", "eth_blockNumber", "eth_call", "eth_chainId", "eth_coinbase",
	"eth_createAccessList", "eth_estimateGas", "eth_feeHistory", "eth_gasPrice", "eth_getBalance",
	"eth_getBlockByHash", "eth_getBlockByNumber", "eth_getBlockReceipts", "eth_getBlockTransactionCountByHash",
	"eth_getBlockTransactionCountByNumber", "eth_getCode", "eth_getFilterChanges", "eth_getFilterLogs", "eth_getLogs",
	"eth_getProof", "eth_getStorageAt", "eth_getTransactionByBlockHashAndIndex",
	"eth_getTransactionByBlockNumberAndIndex", "eth_getTransactionByHash", "eth_getTransactionCount",
	"eth_getTransactionReceipt", "eth_getUncleCountByBlockHash", "eth_getUncleCountByBlockNumber",
	"eth_maxPriorityFeePerGas", "eth_newBlockFilter", "eth_newFilter", "eth_newPendingTransactionFilter",
	"eth_sendRawTransaction", "eth_sendTransaction", "eth_sign", "eth_signTransaction", "eth_simulateV1",
	"eth_syncing", "eth_uninstallFilter", "eth_config", "eth_sendRawTransactionSync",
	"debug_getBadBlocks", "debug_getRawBlock", "debug_getRawHeader", "debug_getRawReceipts", "debug_getRawTransaction",
	"txpool_content", "txpool_contentFrom", "txpool_inspect", "txpool_status",
	"net_version", "net_listening", "net_peerCount", "web3_clientVersion", "web3_sha3",
}

// TestInventory_TheServedSetIsSection6 is S1 and X: the module serves the
// methods of section 6, the eth ones mirrored as ftm, and nothing else.
func TestInventory_TheServedSetIsSection6(t *testing.T) {
	m := New(tests.NewFakeCore(), DefaultConfig(), prometheus.NewRegistry())
	want := slices.Clone(inventory)
	for _, method := range inventory {
		if name, ok := strings.CutPrefix(method, "eth_"); ok {
			want = append(want, "ftm_"+name)
		}
	}
	want = append(want, profilingShadows...)
	slices.Sort(want)
	require.Equal(t, want, m.d.Methods())
	for method := range notServed {
		require.False(t, m.d.Has(method), "%s is in section X", method)
	}
}

// TestInventory_EveryMethodOfGoEthereumIsServedOrListed is Q5.
func TestInventory_EveryMethodOfGoEthereumIsServedOrListed(t *testing.T) {
	for _, method := range gethMethods {
		_, listed := notServed[method]
		require.True(t, slices.Contains(inventory, method) || listed, "%s is neither served nor in section X", method)
	}
}

// TestInventory_WhatIsNotServedIsNotFoundOnEveryTransport is S1 and X: a
// method no Register names answers -32601 over HTTP, WebSocket and IPC.
func TestInventory_WhatIsNotServedIsNotFoundOnEveryTransport(t *testing.T) {
	cfg := DefaultConfig()
	ipc := filepath.Join(t.TempDir(), "sonic.ipc")
	cfg.Transport = transport.Config{
		HTTP: &transport.HTTPConfig{Host: "127.0.0.1", Vhosts: []string{"*"}},
		WS:   &transport.WSConfig{Host: "127.0.0.1", Port: 0, Origins: []string{"*"}},
		IPC:  ipc,
	}
	fc := tests.NewFakeCore()
	m := New(fc, cfg, prometheus.NewRegistry())
	require.NoError(t, m.Start())
	t.Cleanup(func() { _ = m.Stop() })
	ctx := context.Background()
	for _, url := range []string{m.HTTPEndpoint(), m.WSEndpoint(), m.IPCEndpoint()} {
		client, err := gethrpc.DialContext(ctx, url)
		require.NoError(t, err)
		for method := range notServed {
			if strings.HasPrefix(method, "net_") {
				continue // forwarded to the node's handler
			}
			err := client.Call(nil, method)
			var rpcErr gethrpc.Error
			require.ErrorAs(t, err, &rpcErr, "%s over %s", method, url)
			require.Equal(t, -32601, rpcErr.ErrorCode(), "%s over %s", method, url)
		}
		var chainID string
		require.NoError(t, client.Call(&chainID, "ftm_chainId"))
		client.Close()
	}
}
