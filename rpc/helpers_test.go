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
	"path/filepath"
	"strings"
	"testing"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/transport"
)

// testConfig is the configuration of the test modules: HTTP and WebSocket on
// one ephemeral port, and a unix socket in the directory of the test.
func testConfig(t *testing.T) rpc.Config {
	return rpc.Config{
		Limits:        dispatch.DefaultConfig(),
		ClientVersion: "test/v0",
		HTTP: &transport.HTTPConfig{
			Host: "127.0.0.1",
			WS: &transport.WSConfig{
				StreamConfig: transport.StreamConfig{MaxSubscriptions: 4},
				Origins:      []string{"http://allowed.example"},
			},
			VHosts: []string{"localhost", "allowed.example"},
		},
		IPC: &rpc.IPCConfig{Path: filepath.Join(t.TempDir(), "sonic.ipc")},
	}
}

// start creates a module over the core, starts it and stops it with the test.
// Its metrics are registered nowhere unless an option says otherwise.
func start(t *testing.T, cfg rpc.Config, c core.Core, opts ...rpc.Option) *rpc.Module {
	t.Helper()
	module, err := rpc.New(cfg, c, append([]rpc.Option{rpc.WithMetrics(dispatch.NewMetrics(nil))}, opts...)...)
	require.NoError(t, err)
	require.NoError(t, module.Start())
	t.Cleanup(func() { require.NoError(t, module.Stop()) })
	return module
}

// startModule starts a module with the test configuration changed by the
// mutate functions, and returns it with its HTTP address and its socket.
func startModule(t *testing.T, c core.Core, mutate ...func(*rpc.Config)) (*rpc.Module, string, string) {
	t.Helper()
	cfg := testConfig(t)
	for _, m := range mutate {
		m(&cfg)
	}
	module := start(t, cfg, c)
	return module, module.HTTPAddr(), module.IPCPath()
}

// startWithMetrics is startModule with the metrics in the registry.
func startWithMetrics(t *testing.T, c core.Core, reg *prometheus.Registry) (*rpc.Module, string, string) {
	t.Helper()
	module := start(t, testConfig(t), c, rpc.WithMetrics(dispatch.NewMetrics(reg)))
	return module, module.HTTPAddr(), module.IPCPath()
}

// dialHTTP connects a client to the HTTP address; it is closed with the test.
func dialHTTP(t *testing.T, addr string) *gethrpc.Client {
	t.Helper()
	client, err := gethrpc.DialHTTP("http://" + addr)
	require.NoError(t, err)
	t.Cleanup(client.Close)
	return client
}

// dialWS connects a client to the WebSocket address; it is closed with the
// test.
func dialWS(t *testing.T, addr string) *gethrpc.Client {
	t.Helper()
	client, err := gethrpc.DialWebsocket(context.Background(), "ws://"+addr, "")
	require.NoError(t, err)
	t.Cleanup(client.Close)
	return client
}

// dialIPC connects a client to the socket; it is closed with the test.
func dialIPC(t *testing.T, path string) *gethrpc.Client {
	t.Helper()
	client, err := gethrpc.DialIPC(context.Background(), path)
	require.NoError(t, err)
	t.Cleanup(client.Close)
	return client
}

// scrape holds the series of a registry: the name with its labels in the
// order of the metric family, to the value. A histogram has a _count and a
// _sum series.
type scrape map[string]float64

func gather(t *testing.T, g prometheus.Gatherer) scrape {
	t.Helper()
	families, err := g.Gather()
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

// panicsByMethod sums rpc_panics_total and lists the methods that panicked.
func panicsByMethod(t *testing.T, g prometheus.Gatherer) (float64, map[string]float64) {
	t.Helper()
	var total float64
	by := map[string]float64{}
	for key, value := range gather(t, g) {
		if method, ok := strings.CutPrefix(key, "rpc_panics_total{method="); ok {
			total += value
			by[strings.TrimSuffix(method, "}")] += value
		}
	}
	return total, by
}
