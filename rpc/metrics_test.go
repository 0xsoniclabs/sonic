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
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/0xsoniclabs/sonic/rpc/transport"
)

// scrape reads the series of a registry: the name with its labels in the
// order of the metric family, to the value (the sample count for histograms).
type scrape map[string]float64

func gather(t *testing.T, reg *prometheus.Registry) scrape {
	t.Helper()
	families, err := reg.Gather()
	require.NoError(t, err)
	res := scrape{}
	for _, f := range families {
		for _, m := range f.GetMetric() {
			key := f.GetName()
			for _, l := range m.GetLabel() {
				key += fmt.Sprintf("{%s=%s}", l.GetName(), l.GetValue())
			}
			switch {
			case m.Counter != nil:
				res[key] = m.GetCounter().GetValue()
			case m.Gauge != nil:
				res[key] = m.GetGauge().GetValue()
			case m.Histogram != nil:
				res[key+"_count"] = float64(m.GetHistogram().GetSampleCount())
				res[key+"_sum"] = m.GetHistogram().GetSampleSum()
			}
		}
	}
	return res
}

func startModuleWith(t *testing.T, fc *tests.FakeCore, reg *prometheus.Registry) (*rpc.Module, string, string) {
	t.Helper()
	sock := t.TempDir() + "/m.ipc"
	module, err := rpc.New(rpc.Config{
		Limits: dispatch.DefaultConfig(),
		HTTP: &transport.HTTPConfig{
			Host: "127.0.0.1",
			WS:   &transport.WSConfig{StreamConfig: transport.StreamConfig{MaxSubscriptions: 4}},
		},
		IPC: &rpc.IPCConfig{Path: sock},
	}, fc, rpc.WithMetrics(dispatch.NewMetrics(reg)))
	require.NoError(t, err)
	require.NoError(t, module.Start())
	t.Cleanup(func() { require.NoError(t, module.Stop()) })
	return module, module.HTTPAddr(), sock
}

// MON2: per method counters and histograms, HTTP, WebSocket and IPC alike.
func TestMetrics_CountRequestsOfEveryTransport(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(2))
	reg := prometheus.NewRegistry()
	_, addr, sock := startModuleWith(t, fc, reg)

	httpClient, err := gethrpc.DialHTTP("http://" + addr)
	require.NoError(t, err)
	defer httpClient.Close()
	wsClient := dialWS(t, addr)
	ipcClient, err := gethrpc.DialIPC(context.Background(), sock)
	require.NoError(t, err)
	defer ipcClient.Close()

	for _, c := range []*gethrpc.Client{httpClient, wsClient, ipcClient} {
		var n string
		require.NoError(t, c.Call(&n, "eth_blockNumber"))
	}
	var out any
	require.Error(t, httpClient.Call(&out, "eth_getBlockByNumber"))      // invalid params
	require.Error(t, wsClient.Call(&out, "eth_noSuchMethod"))            // unknown method
	require.Error(t, ipcClient.Call(&out, "eth_getBlockByNumber", 1, 2)) // invalid params

	s := gather(t, reg)
	require.EqualValues(t, 3, s["rpc_requests_total{method=eth_blockNumber}{result=ok}"])
	require.EqualValues(t, 2, s["rpc_requests_total{method=eth_getBlockByNumber}{result=invalid_params}"])
	require.EqualValues(t, 3, s["rpc_duration_seconds{method=eth_blockNumber}_count"])
	require.EqualValues(t, 3, s["rpc_response_bytes{method=eth_blockNumber}_count"])
	require.Greater(t, s["rpc_response_bytes{method=eth_blockNumber}_sum"], 0.0)
	require.EqualValues(t, 0, s["rpc_inflight"])

	// MON5: unknown names share one label.
	require.EqualValues(t, 1, s["rpc_requests_total{method=other}{result=not_found}"])
	for key := range s {
		require.NotContains(t, key, "noSuchMethod")
	}
}

// MON5: no client input becomes a label.
func TestMetrics_CardinalityIsBoundedByTheMethodTable(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	reg := prometheus.NewRegistry()
	_, addr, _ := startModuleWith(t, fc, reg)
	client, err := gethrpc.DialHTTP("http://" + addr)
	require.NoError(t, err)
	defer client.Close()

	var out any
	before := len(gather(t, reg))
	for i := range 200 {
		_ = client.Call(&out, fmt.Sprintf("eth_made_up_%d", i))
		_ = client.Call(&out, "eth_getBalance", common.BigToAddress(common.Big1), fmt.Sprintf("0x%x", i+1000))
		_ = client.Call(&out, "eth_getBalance", fmt.Sprintf("bad-address-%d", i))
	}
	after := gather(t, reg)
	require.LessOrEqual(t, len(after)-before, 6, "series grew with client input: %v", after)
	for key := range after {
		require.NotContains(t, key, "made_up")
		require.NotContains(t, key, "bad-address")
	}
}

// MON2: connections per transport, subscriptions per kind and their events.
func TestMetrics_ConnectionsAndSubscriptions(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	reg := prometheus.NewRegistry()
	_, addr, sock := startModuleWith(t, fc, reg)

	ws := dialWS(t, addr)
	ipcClient, err := gethrpc.DialIPC(context.Background(), sock)
	require.NoError(t, err)
	defer ipcClient.Close()
	var n string
	require.NoError(t, ws.Call(&n, "eth_blockNumber"))
	require.NoError(t, ipcClient.Call(&n, "eth_blockNumber"))

	require.Eventually(t, func() bool {
		s := gather(t, reg)
		return s["rpc_connections{transport=ws}"] == 1 && s["rpc_connections{transport=ipc}"] == 1
	}, 2*time.Second, 10*time.Millisecond, "%v", gather(t, reg))

	heads := make(chan map[string]any, 4)
	sub, err := ws.EthSubscribe(context.Background(), heads, "newHeads")
	require.NoError(t, err)
	require.EqualValues(t, 1, gather(t, reg)["rpc_subscriptions{kind=newHeads}"])
	fc.AddBlock()
	<-heads
	require.Eventually(t, func() bool {
		return gather(t, reg)["rpc_subscription_events_total{kind=newHeads}{outcome=sent}"] == 1
	}, 2*time.Second, 10*time.Millisecond)

	sub.Unsubscribe()
	require.Eventually(t, func() bool {
		return gather(t, reg)["rpc_subscriptions{kind=newHeads}"] == 0
	}, 2*time.Second, 10*time.Millisecond)

	ws.Close()
	require.Eventually(t, func() bool {
		return gather(t, reg)["rpc_connections{transport=ws}"] == 0
	}, 2*time.Second, 10*time.Millisecond)
}

// MON3: the served head and its lag behind the committed head.
func TestMetrics_HeadServedAndLag(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(3))
	reg := prometheus.NewRegistry()
	_, _, _ = startModuleWith(t, fc, reg)

	require.EqualValues(t, 3, gather(t, reg)["rpc_head_served"])

	// The node commits blocks the archive has not got yet.
	fc.SetProgress(core.Progress{CurrentBlock: 8})
	require.Eventually(t, func() bool {
		return gather(t, reg)["rpc_head_lag"] == 5
	}, 4*time.Second, 50*time.Millisecond, "%v", gather(t, reg))

	// The head catches up.
	fc.SetProgress(core.Progress{CurrentBlock: 3})
	require.Eventually(t, func() bool {
		return gather(t, reg)["rpc_head_lag"] == 0
	}, 4*time.Second, 50*time.Millisecond)
}
