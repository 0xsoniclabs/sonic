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
	"io"
	"math/big"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/0xsoniclabs/sonic/rpc/transport"
)

// fixture is a started module over the replay chain, with one installed
// filter of each kind and a signed bundle ready to submit.
type fixture struct {
	t        *testing.T
	r        *tests.ReplayChain
	m        *Module
	reg      *prometheus.Registry
	rawTx    hexutil.Bytes
	pooled   *types.Transaction
	bundle   map[string]any
	planHash common.Hash
}

func newFixture(t *testing.T, mutate ...func(*Config)) *fixture {
	t.Helper()
	r := tests.NewReplayChain(t)
	cfg := DefaultConfig()
	cfg.ClientVersion = "sonic/test"
	for _, m := range mutate {
		m(&cfg)
	}
	reg := prometheus.NewRegistry()
	m := New(r.Core, cfg, reg)
	require.NoError(t, m.Start())
	t.Cleanup(func() { _ = m.Stop() })
	f := &fixture{t: t, r: r, m: m, reg: reg}
	f.rawTx = tests.Encode(t, r.Wallet.Transfer(t, 2, r.Recipient, big.NewInt(1)))
	r.Core.SetNonce(r.Wallet.Address, 2)
	f.pooled = r.Wallet.Transfer(t, 3, r.Recipient, big.NewInt(1))
	r.Core.SetPool(map[common.Address]types.Transactions{r.Wallet.Address: {f.pooled}}, nil)
	f.prepareBundle()
	r.Core.SetBundle(f.planHash, core.BundleInfo{BlockNumber: r.Block.NumberU64(), Count: 1})
	return f
}

// prepareBundle prepares and signs a bundle of one transfer.
func (f *fixture) prepareBundle() {
	var prepared struct {
		Transactions  []exec.CallArgs `json:"transactions"`
		ExecutionPlan json.RawMessage `json:"executionPlan"`
	}
	f.must(&prepared, "sonic_prepareBundle", map[string]any{
		"steps": []any{map[string]any{"from": f.r.Wallet.Address, "to": f.r.Recipient, "value": "0x1", "nonce": "0x2"}},
	})
	tx, err := prepared.Transactions[0].ToTransaction()
	require.NoError(f.t, err)
	signed, err := types.SignTx(tx, tests.Signer(), f.r.Wallet.Key)
	require.NoError(f.t, err)
	f.bundle = map[string]any{
		"signedTransactions": []hexutil.Bytes{tests.Encode(f.t, signed)},
		"executionPlan":      prepared.ExecutionPlan,
	}
	var hash common.Hash
	f.must(&hash, "sonic_submitBundle", f.bundle)
	f.planHash = hash
}

func (f *fixture) call(ctx context.Context, method string, params ...any) (json.RawMessage, error) {
	raw, err := json.Marshal(params)
	require.NoError(f.t, err)
	return f.m.d.Serve(ctx, dispatch.Call{Method: method, Params: raw})
}

func (f *fixture) must(result any, method string, params ...any) {
	f.t.Helper()
	res, err := f.call(context.Background(), method, params...)
	require.NoError(f.t, err, method)
	if result != nil {
		require.NoError(f.t, json.Unmarshal(res, result), method)
	}
}

// filterIDs installs one filter of each kind.
func (f *fixture) filterIDs() map[string]string {
	ids := map[string]string{}
	var id string
	f.must(&id, "eth_newBlockFilter")
	ids["block"] = id
	f.must(&id, "eth_newPendingTransactionFilter")
	ids["pending"] = id
	f.must(&id, "eth_newFilter", map[string]any{"address": f.r.Writer})
	ids["logs"] = id
	return ids
}

// requests holds, for every method the module serves, a request with every
// optional argument given: the shape the matrices run and shorten.
func (f *fixture) requests(filters map[string]string) map[string][]any {
	r := f.r
	from, block, txHash := r.Wallet.Address, r.Block.Hash, r.Call.Hash()
	call := map[string]any{"from": from, "to": r.Writer, "gas": "0x20000"}
	traceConfig := map[string]any{"tracer": "callTracer", "timeout": "1s"}
	overrides := map[string]any{r.Recipient.Hex(): map[string]any{"balance": "0x10"}}
	blockOverrides := map[string]any{"number": "0x2"}
	return map[string][]any{
		"debug_getBlockRlp":                          {1},
		"debug_printBlock":                           {1},
		"debug_traceBlockByHash":                     {block, traceConfig},
		"debug_traceBlockByNumber":                   {"0x1", traceConfig},
		"debug_traceCall":                            {call, "latest", traceConfig},
		"debug_traceTransaction":                     {txHash, traceConfig},
		"eth_blobBaseFee":                            {},
		"eth_blockNumber":                            {},
		"eth_call":                                   {call, "latest", overrides, blockOverrides},
		"eth_chainId":                                {},
		"eth_coinbase":                               {},
		"eth_config":                                 {},
		"eth_createAccessList":                       {call, "latest"},
		"eth_currentEpoch":                           {},
		"eth_estimateGas":                            {call, "latest", overrides, blockOverrides},
		"eth_etherbase":                              {},
		"eth_feeHistory":                             {"0x2", "latest", []float64{10, 50}},
		"eth_gasPrice":                               {},
		"eth_getAccount":                             {from, "latest"},
		"eth_getBalance":                             {from, "latest"},
		"eth_getBlockByHash":                         {block, true},
		"eth_getBlockByNumber":                       {"0x1", true},
		"eth_getBlockReceipts":                       {"0x1"},
		"eth_getBlockTransactionCountByHash":         {block},
		"eth_getBlockTransactionCountByNumber":       {"0x1"},
		"eth_getCode":                                {r.Writer, "latest"},
		"eth_getFilterChanges":                       {filters["block"]},
		"eth_getFilterLogs":                          {filters["logs"]},
		"eth_getHeaderByHash":                        {block},
		"eth_getHeaderByNumber":                      {"0x1"},
		"eth_getLogs":                                {map[string]any{"fromBlock": "0x0", "toBlock": "latest", "address": r.Writer}},
		"eth_getProof":                               {r.Writer, []string{"0x0"}, "latest"},
		"eth_getRawTransactionByBlockHashAndIndex":   {block, "0x1"},
		"eth_getRawTransactionByBlockNumberAndIndex": {"0x1", "0x1"},
		"eth_getRawTransactionByHash":                {f.pooled.Hash()},
		"eth_getRules":                               {"latest"},
		"eth_getStorageAt":                           {r.Writer, "0x0", "latest"},
		"eth_getTransactionByBlockHashAndIndex":      {block, "0x1"},
		"eth_getTransactionByBlockNumberAndIndex":    {"0x1", "0x1"},
		"eth_getTransactionByHash":                   {txHash},
		"eth_getTransactionCount":                    {from, "pending"},
		"eth_getTransactionReceipt":                  {txHash},
		"eth_getUncleByBlockHashAndIndex":            {block, "0x0"},
		"eth_getUncleByBlockNumberAndIndex":          {"0x1", "0x0"},
		"eth_getUncleCountByBlockHash":               {block},
		"eth_getUncleCountByBlockNumber":             {"0x1"},
		"eth_hashrate":                               {},
		"eth_maxPriorityFeePerGas":                   {},
		"eth_newBlockFilter":                         {},
		"eth_newFilter":                              {map[string]any{"address": r.Writer, "topics": []any{r.Log.Topics[0]}}},
		"eth_newPendingTransactionFilter":            {true},
		"eth_pendingTransactions":                    {},
		"eth_sendRawTransaction":                     {f.rawTx},
		"eth_sendRawTransactionSync":                 {f.rawTx, "0x1"},
		"eth_simulateV1": {map[string]any{
			"blockStateCalls": []any{map[string]any{"calls": []any{map[string]any{"from": from, "to": r.Returner}}}},
		}, "latest"},
		"eth_syncing":                      {},
		"eth_uninstallFilter":              {filters["pending"]},
		"net_version":                      {},
		"rpc_modules":                      {},
		"sonic_estimateGasForTransactions": {[]any{call}, "latest", overrides, blockOverrides},
		"sonic_getBundleInfo":              {f.planHash},
		"sonic_prepareBundle": {map[string]any{
			"steps": []any{map[string]any{"from": from, "to": r.Recipient, "value": "0x1", "nonce": "0x2"}},
		}},
		"sonic_submitBundle": {f.bundle},
		"trace_block":        {"0x1"},
		"trace_call":         {call, []string{"trace"}, "latest", map[string]any{"stateOverrides": overrides}},
		"trace_callMany":     {[]any{[]any{call, []string{"trace"}}}, "latest", map[string]any{"stateOverrides": overrides}},
		"trace_filter":       {map[string]any{"fromBlock": "0x1", "toBlock": "0x1", "fromAddress": []any{from}}},
		"trace_get":          {txHash, []string{"0x0"}},
		"trace_transaction":  {txHash},
		"txpool_content":     {},
		"txpool_contentFrom": {from},
		"txpool_inspect":     {},
		"txpool_status":      {},
		"web3_clientVersion": {},
		"web3_sha3":          {"0x01"},
	}
}

// profiling are the debug methods of the node's process, which a build
// without the enable_debug tag answers as not enabled; they take no Core
// call, so the matrices do not run them.
func isProfiling(method string) bool {
	switch method {
	case "debug_getBlockRlp", "debug_printBlock", "debug_traceBlockByHash", "debug_traceBlockByNumber",
		"debug_traceCall", "debug_traceTransaction":
		return false
	}
	return strings.HasPrefix(method, "debug_")
}

// served are the methods the matrices run: every method registered, but the
// ftm mirror (the same handlers), the subscriptions (which need a stream) and
// the profiling methods.
func (f *fixture) served() []string {
	var out []string
	for _, m := range f.m.d.Methods() {
		if strings.HasPrefix(m, "ftm_") || strings.HasSuffix(m, "_subscribe") || strings.HasSuffix(m, "_unsubscribe") || isProfiling(m) {
			continue
		}
		out = append(out, m)
	}
	return out
}

func TestModule_EveryServedMethodHasARequestOfTheMatrices(t *testing.T) {
	f := newFixture(t)
	reqs := f.requests(f.filterIDs())
	served := f.served()
	for _, m := range served {
		require.Contains(t, reqs, m, "no request for %s", m)
	}
	for m := range reqs {
		require.Contains(t, served, m, "a request for %s, which is not served", m)
	}
	for _, m := range served {
		_, err := f.call(context.Background(), m, reqs[m]...)
		if err != nil {
			t.Logf("%s: %v", m, err)
		}
	}
}

func TestModule_TheFtmNamespaceMirrorsEth(t *testing.T) {
	f := newFixture(t)
	var eth, ftm []string
	for _, m := range f.m.d.Methods() {
		if name, ok := strings.CutPrefix(m, "eth_"); ok {
			eth = append(eth, name)
		}
		if name, ok := strings.CutPrefix(m, "ftm_"); ok {
			ftm = append(ftm, name)
		}
	}
	require.Equal(t, eth, ftm)
	var a, b string
	f.must(&a, "eth_chainId")
	f.must(&b, "ftm_chainId")
	require.Equal(t, a, b)
}

// panics counts the requests answered after a panic, per method.
func (f *fixture) panics() map[string]float64 {
	families, err := f.reg.Gather()
	require.NoError(f.t, err)
	out := map[string]float64{}
	for _, family := range families {
		if family.GetName() != "rpc_requests_total" {
			continue
		}
		for _, m := range family.GetMetric() {
			labels := map[string]string{}
			for _, l := range m.GetLabel() {
				labels[l.GetName()] = l.GetValue()
			}
			if labels["result"] == "panic" {
				out[labels["method"]] += m.GetCounter().GetValue()
			}
		}
	}
	return out
}

// grown lists the methods whose count grew.
func grown(before, after map[string]float64) []string {
	var out []string
	for method, n := range after {
		if n > before[method] {
			out = append(out, method)
		}
	}
	slices.Sort(out)
	return out
}

// loopOnly are the Core methods that no request calls, with the reason.
var loopOnly = map[string]string{
	"ChainInfo":     "read by Chain.Start and the head loop with each new head; a request reads what it read",
	"Head":          "read by Chain.Start and the head loop; a request reads the served head",
	"SubscribeHead": "taken by the head loop only",
	"SubscribeTxs":  "taken by the pool loop only",
	"PendingHeads":  "read by the oracle's sampler only (I10)",
}

// TestMatrix_EveryCoreFailureIsAnsweredWithoutPanic is Q3 and P1: every
// method, with every Core method failing in every mode. A failure that is not
// a panic is answered without one; a panic of a Core method is contained,
// answered and counted, and some request reaches every Core method but those
// only a loop calls (TS8).
func TestMatrix_EveryCoreFailureIsAnsweredWithoutPanic(t *testing.T) {
	f := newFixture(t)
	served := f.served()
	for _, name := range tests.CoreMethods() {
		for _, mode := range tests.Modes {
			reqs := f.requests(f.filterIDs())
			if name == "OracleInputs" {
				// The oracle reads its inputs for a suggestion it has not
				// cached; a cached one lives for a second.
				time.Sleep(1100 * time.Millisecond)
			}
			before := f.panics()
			f.r.Core.SetMode(name, mode)
			for _, method := range served {
				o1, r1 := f.r.Core.TrackStates()
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				_, _ = f.call(ctx, method, reqs[method]...)
				cancel()
				if o2, r2 := f.r.Core.TrackStates(); o2-o1 != r2-r1 {
					t.Errorf("%s with %s in mode %v opened %d states and released %d", method, name, mode, o2-o1, r2-r1)
				}
			}
			f.r.Core.SetMode(name, tests.Normal)
			panicked := grown(before, f.panics())
			if mode != tests.Panic {
				if len(panicked) > 0 {
					t.Errorf("%s in mode %v: requests panicked: %v", name, mode, panicked)
				}
				continue
			}
			if _, ok := loopOnly[name]; ok {
				continue
			}
			if len(panicked) == 0 {
				t.Errorf("no request reached %s", name)
			}
		}
	}
	// I7 and P5: every state a request opened was released once, whatever
	// failed or panicked after it was opened.
	opened, released := f.r.Core.TrackStates()
	require.Positive(t, opened)
	require.Equal(t, opened, released)
}

// TestMatrix_EveryArgumentShapeIsAnsweredWithoutPanic is Q3 for arguments:
// every request shortened to each length, and with each argument null.
func TestMatrix_EveryArgumentShapeIsAnsweredWithoutPanic(t *testing.T) {
	f := newFixture(t)
	served := f.served()
	reqs := f.requests(f.filterIDs())
	before := f.panics()
	for _, method := range served {
		full := reqs[method]
		var shapes [][]any
		for n := 0; n <= len(full); n++ {
			shapes = append(shapes, full[:n])
		}
		for i := range full {
			shape := slices.Clone(full)
			shape[i] = nil
			shapes = append(shapes, shape)
		}
		for _, shape := range shapes {
			_, _ = f.call(context.Background(), method, shape...)
		}
	}
	require.Empty(t, grown(before, f.panics()))
}

// TestCancellation_EveryMethodReturnsWithinOneCoreCall is Q2: with every Core
// call slow, a cancelled request returns within about one call.
func TestCancellation_EveryMethodReturnsWithinOneCoreCall(t *testing.T) {
	f := newFixture(t)
	reqs := f.requests(f.filterIDs())
	const delay = 50 * time.Millisecond
	f.r.Core.SetDelay(delay)
	for _, method := range f.served() {
		ctx, cancel := context.WithTimeout(context.Background(), delay/2)
		start := time.Now()
		_, _ = f.call(ctx, method, reqs[method]...)
		elapsed := time.Since(start)
		cancel()
		if elapsed >= 3*delay {
			t.Errorf("%s returned %v after its cancel", method, elapsed)
		}
	}
}

func TestModule_CallServesInProcess(t *testing.T) {
	f := newFixture(t)
	var id hexutil.Big
	require.NoError(t, f.m.Call(context.Background(), &id, "eth_chainId"))
	require.Equal(t, int64(tests.ChainID), id.ToInt().Int64())
	require.NoError(t, f.m.Call(context.Background(), nil, "eth_chainId"))
	require.EqualError(t, f.m.Call(context.Background(), nil, "eth_nope"), "the method eth_nope does not exist/is not available")
	var wrong []int
	require.ErrorContains(t, f.m.Call(context.Background(), &wrong, "eth_chainId"), "decoding the answer of eth_chainId")
	require.Error(t, f.m.Call(context.Background(), nil, "eth_getBalance", make(chan int)))
}

func TestModule_HealthReportsTheHeadAndTheLag(t *testing.T) {
	r := tests.NewReplayChain(t)
	cfg := DefaultConfig()
	cfg.Transport.HTTP = &transport.HTTPConfig{Host: "127.0.0.1", Vhosts: []string{"*"}}
	m := New(r.Core, cfg, prometheus.NewRegistry())
	require.NoError(t, m.Start())
	t.Cleanup(func() { _ = m.Stop() })
	r.Core.SetProgress(core.Progress{CurrentBlock: 3})
	resp, err := http.Get(m.HTTPEndpoint() + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"head":"0x1","lag":2}`, string(body))
	require.Empty(t, m.WSEndpoint())
	require.Empty(t, m.IPCEndpoint())
}

func TestModule_StartFailsWithoutAHeadAndStopTwiceIsFine(t *testing.T) {
	fc := tests.NewFakeCore()
	fc.SetMode("Head", tests.Fail)
	cfg := DefaultConfig()
	cfg.Dispatch.RequestTimeout = 0
	m := New(fc, cfg, prometheus.NewRegistry())
	require.ErrorIs(t, m.Start(), tests.ErrFake)
	require.NoError(t, m.Stop())
	require.NoError(t, m.Stop())
}
