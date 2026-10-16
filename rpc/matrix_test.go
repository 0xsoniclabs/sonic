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

package rpc_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

// served is every method the module registers by name, in sorted order.
func registeredMethods(m *rpc.Module) []string {
	names := m.Dispatcher().Methods()
	sort.Strings(names)
	return names
}

// newMatrixModule builds a module over a core with blocks, transactions and a
// real state, without transports.
func newMatrixModule(t *testing.T) (*rpc.Module, *tests.FakeCore, prometheus.Gatherer) {
	t.Helper()
	fc, cs := tests.NewCarmenCore(t, tests.WithBlocks(3))
	cs.SetBalance(common.HexToAddress("0x1"), big.NewInt(1e18)).SetCode(common.HexToAddress("0x2"), []byte{0x60, 0x01, 0x60, 0x00, 0x55, 0x00})
	fc.AddBlock(matrixTx(t, 0), matrixTx(t, 1))
	fc.AddBlock()
	fc.AddPoolTransaction(matrixTx(t, 2))
	reg := prometheus.NewRegistry()
	cfg := rpc.Config{Limits: dispatch.DefaultConfig(), ClientVersion: "matrix"}
	cfg.Limits.Timeout = 5 * time.Second
	module, err := rpc.New(cfg, fc, rpc.WithMetrics(dispatch.NewMetrics(reg)))
	require.NoError(t, err)
	require.NoError(t, module.Start())
	t.Cleanup(func() { require.NoError(t, module.Stop()) })
	return module, fc, reg
}

func matrixTx(t *testing.T, nonce uint64) *types.Transaction { return signedTx(t, nonce) }

// panics sums rpc_panics_total.
func panics(t *testing.T, g prometheus.Gatherer) float64 {
	t.Helper()
	total, _ := panicsByMethod(t, g)
	return total
}

// panicsByMethod sums rpc_panics_total and lists the methods that panicked.
func panicsByMethod(t *testing.T, g prometheus.Gatherer) (float64, map[string]float64) {
	t.Helper()
	families, err := g.Gather()
	require.NoError(t, err)
	var total float64
	by := map[string]float64{}
	for _, f := range families {
		if f.GetName() != "rpc_panics_total" {
			continue
		}
		for _, m := range f.GetMetric() {
			total += m.GetCounter().GetValue()
			for _, l := range m.GetLabel() {
				by[l.GetValue()] += m.GetCounter().GetValue()
			}
		}
	}
	return total, by
}

// argumentShapes returns parameter lists for a method: for each arity the
// method accepts, one list per kind of value.
func argumentShapes(d *dispatch.Dispatcher, method string) []json.RawMessage {
	params, required, _ := d.Signature(method)
	values := []string{
		`null`, `""`, `"0x"`, `"0x1"`, `"latest"`, `"pending"`, `"earliest"`, `1`, `-1`, `true`, `{}`, `[]`,
		`"0x` + strings.Repeat("00", 20) + `"`,
		`"0x` + strings.Repeat("00", 32) + `"`,
		`{"from":"0x0000000000000000000000000000000000000001","to":"0x0000000000000000000000000000000000000002"}`,
		`{"blockHash":"0x` + strings.Repeat("00", 32) + `"}`,
		`{"fromBlock":"0x0","toBlock":"0x2"}`,
		`[[{"to":"0x0000000000000000000000000000000000000002"},["trace"]]]`,
	}
	var shapes []json.RawMessage
	for arity := required; arity <= len(params); arity++ {
		for _, v := range values {
			list := make([]string, arity)
			for i := range list {
				list[i] = v
			}
			shapes = append(shapes, json.RawMessage("["+strings.Join(list, ",")+"]"))
		}
		// mixed: an address first, then the values of the later positions
		if arity >= 2 {
			for _, v := range values {
				list := make([]string, arity)
				list[0] = `"0x` + strings.Repeat("00", 20) + `"`
				for i := 1; i < arity; i++ {
					list[i] = v
				}
				shapes = append(shapes, json.RawMessage("["+strings.Join(list, ",")+"]"))
			}
		}
	}
	return append(shapes, nil, json.RawMessage(`null`), json.RawMessage(`{}`), json.RawMessage(`[null,null,null,null,null,null]`))
}

// Q3: every method answers without panic whatever a Core method returns, and
// whatever the shape of its arguments.
func TestNilSafety_InjectionMatrix(t *testing.T) {
	for _, mode := range []tests.Mode{tests.Normal, tests.Fail, tests.NotFound, tests.Nil} {
		t.Run(fmt.Sprint("mode", mode), func(t *testing.T) {
			module, fc, reg := newMatrixModule(t)
			fc.Inject("*", mode)
			d := module.Dispatcher()
			for _, method := range registeredMethods(module) {
				if strings.HasPrefix(method, "debug_") && !strings.HasPrefix(method, "debug_trace") && !strings.HasPrefix(method, "debug_getRaw") {
					continue
				}
				for _, params := range argumentShapes(d, method) {
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					_, _ = d.Serve(ctx, method, params)
					cancel()
				}
			}
			total, by := panicsByMethod(t, reg)
			require.Zero(t, total, "methods panicked: %v", by)
		})
	}
}

// P1: a panic in any Core method is contained: the request is answered -32603
// and counted, nothing else breaks.
func TestPanicInjection_EveryCoreMethodIsContained(t *testing.T) {
	for _, coreMethod := range tests.CoreMethods {
		if coreMethod == "SubscribeHead" || coreMethod == "SubscribeTxs" {
			continue // called once at start
		}
		t.Run(coreMethod, func(t *testing.T) {
			module, fc, reg := newMatrixModule(t)
			fc.Inject(coreMethod, tests.Panic)
			d := module.Dispatcher()
			for _, method := range registeredMethods(module) {
				if strings.HasPrefix(method, "debug_") && !strings.HasPrefix(method, "debug_trace") && !strings.HasPrefix(method, "debug_getRaw") {
					continue
				}
				for _, params := range argumentShapes(d, method)[:6] {
					_, err := d.Serve(context.Background(), method, params)
					if err != nil && err.Message == "method handler crashed" {
						require.Equal(t, dispatch.CodeInternal, err.Code)
					}
				}
			}
			// The module serves after the panics.
			res, err := d.Serve(context.Background(), "web3_clientVersion", nil)
			require.Nil(t, err)
			require.JSONEq(t, `"matrix"`, string(res))
			_ = reg
		})
	}
}

// Q2: a cancelled request returns promptly, however slow the core is.
func TestCancellation_EveryMethodReturnsWhenCancelled(t *testing.T) {
	module, fc, _ := newMatrixModule(t)
	fc.SetDelay(20 * time.Millisecond) // every Core call takes this long
	d := module.Dispatcher()
	for _, method := range registeredMethods(module) {
		if strings.HasPrefix(method, "debug_") && !strings.HasPrefix(method, "debug_trace") && !strings.HasPrefix(method, "debug_getRaw") {
			continue
		}
		shapes := argumentShapes(d, method)
		for _, params := range []json.RawMessage{shapes[0], shapes[4], shapes[len(shapes)-2]} {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			start := time.Now()
			_, _ = d.Serve(ctx, method, params)
			require.Less(t, time.Since(start), 3*time.Second, "%s did not return when cancelled", method)
		}
	}
}

// The inventory of section 6: every method the plan marks "port" or "local" is
// registered, nothing else is.
var inventory = map[string][]string{
	"eth": {
		"chainId", "blockNumber", "syncing", "etherbase", "coinbase", "hashrate",
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
		"getLogs", "getRules", "currentEpoch", "config",
	},
	"txpool": {"content", "contentFrom", "status", "inspect"},
	"sonic":  {"estimateGasForTransactions", "prepareBundle", "submitBundle", "getBundleInfo"},
	"debug": {
		"getBlockRlp", "printBlock", "traceTransaction", "traceBlockByNumber", "traceBlockByHash", "traceCall",
		"getRawBlock", "getRawHeader", "getRawReceipts", "getRawTransaction",
	},
	"trace": {"transaction", "get", "block", "call", "callMany", "filter"},
	"net":   {"version"},
	"web3":  {"clientVersion", "sha3"},
}

// profiling methods of go-ethereum's debug handler, answered with an error
var disabledDebug = []string{
	"verbosity", "vmodule", "memStats", "gcStats", "cpuProfile", "startCPUProfile", "stopCPUProfile",
	"goTrace", "startGoTrace", "stopGoTrace", "blockProfile", "setBlockProfileRate", "writeBlockProfile",
	"mutexProfile", "setMutexProfileFraction", "writeMutexProfile", "writeMemProfile", "stacks",
	"freeOSMemory", "setGCPercent", "setMemoryLimit",
}

// notServed lists the methods the plan removes (X1, X2) with their reason; they
// are answered as unknown by the module.
var notServed = map[string]string{
	"eth_stop": "S1: exported by accident", "ftm_stop": "S1: exported by accident",
	"eth_accounts": "X1: no wallets", "eth_sign": "X1: no wallets", "eth_signTransaction": "X1: no wallets",
	"eth_sendTransaction": "X1: no wallets", "eth_fillTransaction": "X1: no wallets", "eth_resend": "X1: no wallets",
	"eth_getEpochBlock": "X2: consensus internal", "eth_getEpochStats": "X2: consensus internal",
	"debug_chaindbProperty": "X1: no database properties", "debug_chaindbCompact": "X1: no compaction", "debug_setHead": "X1: no rewind",
	"net_listening": "p2p fact: legacy handler", "net_peerCount": "p2p fact: legacy handler",
}

func expectedMethods() []string {
	var names []string
	for ns, methods := range inventory {
		for _, m := range methods {
			names = append(names, ns+"_"+m)
			if ns == "eth" {
				names = append(names, "ftm_"+m)
			}
		}
	}
	for _, m := range disabledDebug {
		names = append(names, "debug_"+m)
	}
	sort.Strings(names)
	return names
}

func TestRegisteredMethods_MatchTheInventory(t *testing.T) {
	module, _, _ := newMatrixModule(t)
	require.Equal(t, expectedMethods(), registeredMethods(module))

	kinds := module.Dispatcher().SubscriptionKinds()
	sort.Strings(kinds)
	require.Equal(t, []string{
		"eth/logs", "eth/newHeads", "eth/newPendingTransactions",
		"ftm/logs", "ftm/newHeads", "ftm/newPendingTransactions",
	}, kinds)
}

func TestRemovedMethodsAreUnknown(t *testing.T) {
	module, _, _ := newMatrixModule(t)
	d := module.Dispatcher()
	for method, reason := range notServed {
		require.False(t, d.Has(method), "%s (%s) is registered", method, reason)
	}
}

// T4/Q4: every registered method is named by a test of the module.
func TestEveryMethodHasATest(t *testing.T) {
	var corpus strings.Builder
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, "matrix_test.go") {
			return err
		}
		data, err := os.ReadFile(path)
		corpus.Write(data)
		return err
	})
	require.NoError(t, err)
	module, _, _ := newMatrixModule(t)
	var missing []string
	for _, method := range registeredMethods(module) {
		ns, name, _ := strings.Cut(method, "_")
		if ns == "ftm" || strings.HasPrefix(method, "debug_") && contains(disabledDebug, name) {
			continue
		}
		if !strings.Contains(corpus.String(), `"`+method+`"`) {
			missing = append(missing, method)
		}
	}
	require.Empty(t, missing, "methods without a test")
}

func contains(list []string, s string) bool {
	for _, e := range list {
		if e == s {
			return true
		}
	}
	return false
}

// Q5: every method of go-ethereum's eth, debug, txpool, net and web3 APIs that
// a Sonic node served is served by the module or is listed with its reason.
var gethMethods = []string{
	"eth_accounts", "eth_blobBaseFee", "eth_blockNumber", "eth_call", "eth_chainId", "eth_coinbase",
	"eth_createAccessList", "eth_estimateGas", "eth_feeHistory", "eth_fillTransaction", "eth_gasPrice",
	"eth_getBalance", "eth_getBlockByHash", "eth_getBlockByNumber", "eth_getBlockReceipts",
	"eth_getBlockTransactionCountByHash", "eth_getBlockTransactionCountByNumber", "eth_getCode",
	"eth_getFilterChanges", "eth_getFilterLogs", "eth_getHeaderByHash", "eth_getHeaderByNumber", "eth_getLogs",
	"eth_getProof", "eth_getRawTransactionByBlockHashAndIndex", "eth_getRawTransactionByBlockNumberAndIndex",
	"eth_getRawTransactionByHash", "eth_getStorageAt", "eth_getTransactionByBlockHashAndIndex",
	"eth_getTransactionByBlockNumberAndIndex", "eth_getTransactionByHash", "eth_getTransactionCount",
	"eth_getTransactionReceipt", "eth_getUncleByBlockHashAndIndex", "eth_getUncleByBlockNumberAndIndex",
	"eth_getUncleCountByBlockHash", "eth_getUncleCountByBlockNumber", "eth_hashrate", "eth_maxPriorityFeePerGas",
	"eth_newBlockFilter", "eth_newFilter", "eth_newPendingTransactionFilter", "eth_pendingTransactions",
	"eth_resend", "eth_sendRawTransaction", "eth_sendRawTransactionSync", "eth_sendTransaction", "eth_sign",
	"eth_signTransaction", "eth_simulateV1", "eth_syncing", "eth_uninstallFilter",
	"debug_getRawBlock", "debug_getRawHeader", "debug_getRawReceipts", "debug_getRawTransaction",
	"debug_traceBlockByHash", "debug_traceBlockByNumber", "debug_traceCall", "debug_traceTransaction",
	"debug_printBlock", "debug_chaindbProperty", "debug_chaindbCompact", "debug_setHead",
	"txpool_content", "txpool_contentFrom", "txpool_inspect", "txpool_status",
	"net_version", "net_listening", "net_peerCount",
	"web3_clientVersion", "web3_sha3",
}

func TestGethMethodsAreServedOrListed(t *testing.T) {
	module, _, _ := newMatrixModule(t)
	d := module.Dispatcher()
	for _, method := range gethMethods {
		if d.Has(method) {
			continue
		}
		_, listed := notServed[method]
		require.True(t, listed, "%s is neither served nor listed with a reason", method)
	}
}
