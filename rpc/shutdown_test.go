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
	"log/slog"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/0xsoniclabs/sonic/rpc/transport"
)

// logRecorder collects the messages logged while a test runs.
type logRecorder struct {
	mu      sync.Mutex
	records []string
}

func (r *logRecorder) Enabled(context.Context, slog.Level) bool { return true }
func (r *logRecorder) Handle(_ context.Context, rec slog.Record) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	line := rec.Message
	rec.Attrs(func(a slog.Attr) bool {
		line += " " + a.Key + "=" + a.Value.String()
		return true
	})
	r.records = append(r.records, line)
	return nil
}
func (r *logRecorder) WithAttrs([]slog.Attr) slog.Handler { return r }
func (r *logRecorder) WithGroup(string) slog.Handler      { return r }

func (r *logRecorder) contains(parts ...string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, line := range r.records {
		all := true
		for _, p := range parts {
			all = all && strings.Contains(line, p)
		}
		if all {
			return true
		}
	}
	return false
}

func recordLogs(t *testing.T) *logRecorder {
	r := &logRecorder{}
	log.SetDefault(log.NewLogger(r))
	t.Cleanup(func() { log.SetDefault(log.NewLogger(log.DiscardHandler())) })
	return r
}

// TestStop_DuringAnEVMRunAtTheGasCap is Q1: the interpreter does not poll for
// the cancel, so Stop waits for the run, bounded by the gas cap, and logs the
// request it waits for after the grace period.
func TestStop_DuringAnEVMRunAtTheGasCap(t *testing.T) {
	logs := recordLogs(t)
	f := newFixture(t, func(c *Config) {
		c.Dispatch.RequestTimeout = 50 * time.Millisecond
		c.Exec.GasCap = 300_000_000
	})
	spinner := common.HexToAddress("0x5555")
	f.r.Core.SetAccount(spinner, tests.Account{Code: tests.SpinningCode})
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = f.call(context.Background(), "eth_call", map[string]any{"to": spinner, "gas": hexutil.Uint64(300_000_000)}, "latest")
	}()
	require.Eventually(t, func() bool { return testutilInflight(f) == 1 }, 5*time.Second, time.Millisecond)
	start := time.Now()
	require.NoError(t, f.m.Stop())
	took := time.Since(start)
	<-done
	require.Less(t, took, 30*time.Second, "the node's shutdown timeout")
	require.True(t, logs.contains("RPC requests still running after the grace period", "name=eth_call", "age="))
}

func testutilInflight(f *fixture) float64 {
	return tests.Scrape(f.t, f.reg, "rpc_inflight", nil)
}

// TestStop_DuringABlockedCoreCall is Q1 for a Core call that ignores the
// cancel.
func TestStop_DuringABlockedCoreCall(t *testing.T) {
	logs := recordLogs(t)
	f := newFixture(t, func(c *Config) { c.Dispatch.RequestTimeout = 50 * time.Millisecond })
	release := f.r.Core.Gate("State")
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = f.call(context.Background(), "eth_getBalance", f.r.Wallet.Address, "latest")
	}()
	require.Eventually(t, func() bool { return testutilInflight(f) == 1 }, 5*time.Second, time.Millisecond)
	stopped := make(chan time.Duration)
	go func() {
		start := time.Now()
		_ = f.m.Stop()
		stopped <- time.Since(start)
	}()
	time.Sleep(300 * time.Millisecond)
	select {
	case <-stopped:
		t.Fatal("Stop returned while a Core call still ran")
	default:
	}
	release()
	took := <-stopped
	<-done
	require.Less(t, took, 5*time.Second)
	require.True(t, logs.contains("RPC requests still running after the grace period", "name=eth_getBalance"))
}

// TestStop_UnderLoadLeavesNoGoroutine is Q1 and D10: with clients calling
// and subscribed on every transport, Stop returns and no goroutine of the
// module is left.
func TestStop_UnderLoadLeavesNoGoroutine(t *testing.T) {
	r := tests.NewReplayChain(t)
	require.Empty(t, moduleGoroutines())

	cfg := DefaultConfig()
	ipc := filepath.Join(t.TempDir(), "sonic.ipc")
	cfg.Transport = transport.Config{
		HTTP: &transport.HTTPConfig{Host: "127.0.0.1", Vhosts: []string{"*"}},
		WS:   &transport.WSConfig{Host: "127.0.0.1", Port: 0, Origins: []string{"*"}},
		IPC:  ipc,
	}
	m := New(r.Core, cfg, prometheus.NewRegistry())
	require.NoError(t, m.Start())

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	httpClient := &http.Client{Transport: &http.Transport{}}
	for _, url := range []string{m.HTTPEndpoint(), m.WSEndpoint(), ipc} {
		var client *gethrpc.Client
		var err error
		if strings.HasPrefix(url, "http") {
			client, err = gethrpc.DialOptions(ctx, url, gethrpc.WithHTTPClient(httpClient))
		} else {
			client, err = gethrpc.DialContext(ctx, url)
		}
		require.NoError(t, err)
		defer client.Close()
		if !strings.HasPrefix(url, "http") {
			_, err := client.Subscribe(ctx, "eth", make(chan any, 100), "newHeads")
			require.NoError(t, err)
		}
		for range 4 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for ctx.Err() == nil {
					var balance hexutil.Big
					if err := client.CallContext(ctx, &balance, "eth_getBalance", r.Wallet.Address, "latest"); err != nil {
						return
					}
					_ = client.CallContext(ctx, nil, "eth_call", map[string]any{"to": r.Returner}, "latest")
				}
			}()
		}
	}
	time.Sleep(200 * time.Millisecond)
	require.Positive(t, len(moduleGoroutines()), "the module runs")
	require.NoError(t, m.Stop())
	cancel()
	wg.Wait()
	httpClient.CloseIdleConnections()
	deadline := time.Now().Add(10 * time.Second)
	for len(moduleGoroutines()) > 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if left := moduleGoroutines(); len(left) > 0 {
		t.Fatalf("%d goroutines of the module left:\n%s", len(left), strings.Join(left, "\n\n"))
	}
}

// moduleGoroutines are the stacks of the goroutines the module runs: those
// in its code, and the connections of its servers.
func moduleGoroutines() []string {
	buf := make([]byte, 8<<20)
	buf = buf[:runtime.Stack(buf, true)]
	var found []string
	for _, stack := range strings.Split(string(buf), "\n\n") {
		for _, line := range strings.Split(stack, "\n") {
			line = strings.TrimSpace(line)
			inModule := strings.Contains(line, "/sonic/rpc/") && !strings.Contains(line, "_test.go") && strings.Contains(line, ".go:")
			if inModule || strings.HasPrefix(line, "net/http.(*conn).serve") {
				found = append(found, stack)
				break
			}
		}
	}
	return found
}

// TestRequests_ABlockedCoreCallStallsNoOtherRequest is P4.
func TestRequests_ABlockedCoreCallStallsNoOtherRequest(t *testing.T) {
	f := newFixture(t)
	release := f.r.Core.Gate("State")
	defer release()
	go func() { _, _ = f.call(context.Background(), "eth_getBalance", f.r.Wallet.Address, "latest") }()
	require.Eventually(t, func() bool { return testutilInflight(f) == 1 }, 5*time.Second, time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for _, method := range []string{"eth_chainId", "eth_blockNumber", "eth_newBlockFilter", "txpool_status"} {
		_, err := f.call(ctx, method)
		require.NoError(t, err, method)
	}
	_, err := f.call(ctx, "eth_getBlockByNumber", "latest", true)
	require.NoError(t, err)
	_, err = f.call(ctx, "eth_getLogs", map[string]any{"fromBlock": "0x0"})
	require.NoError(t, err)
}

// TestValidation_NoAllocationProportionalToAClientNumber is V9: a number a
// client chooses is bounded before anything is allocated by it.
func TestValidation_NoAllocationProportionalToAClientNumber(t *testing.T) {
	f := newFixture(t)
	requests := map[string][]any{
		"eth_feeHistory":       {"0xffffffffffffffff", "latest", []float64{50}},
		"eth_getLogs":          {map[string]any{"fromBlock": "0x0", "toBlock": "0x7fffffffffffffff"}},
		"trace_filter":         {map[string]any{"fromBlock": "0x0", "toBlock": "0x7fffffffffffffff"}},
		"eth_simulateV1":       {map[string]any{"blockStateCalls": []any{map[string]any{"blockOverrides": map[string]any{"blockNumber": "0x7fffffffffffffff"}}}}, "latest"},
		"eth_getBlockByNumber": {"0x7fffffffffffffff", true},
	}
	for method, params := range requests {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)
		before := stats.TotalAlloc
		_, _ = f.call(context.Background(), method, params...)
		runtime.ReadMemStats(&stats)
		require.Less(t, stats.TotalAlloc-before, uint64(16<<20), method)
	}
}
