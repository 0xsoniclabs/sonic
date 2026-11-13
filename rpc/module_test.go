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
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/rpc"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/0xsoniclabs/sonic/rpc/transport"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func startModule(t *testing.T, config rpc.Config) *rpc.Module {
	t.Helper()
	module := rpc.New(config, tests.NewFakeCore(), gethrpc.NewServer(), prometheus.NewRegistry())
	require.NoError(t, module.Start())
	return module
}

func moduleConfig(t *testing.T, httpModules []string) rpc.Config {
	t.Helper()
	config := rpc.DefaultConfig()
	config.HTTP = &rpc.HTTPConfig{HTTPConfig: transport.HTTPConfig{Endpoint: "127.0.0.1:0"}, Modules: httpModules}
	config.IPCPath = filepath.Join(t.TempDir(), "sonic.ipc")
	config.ClientVersion = "Sonic/test"
	return config
}

func TestModule_ServesItsNamespacesPerTransport(t *testing.T) {
	config := moduleConfig(t, []string{"eth", "web3"})
	module := startModule(t, config)
	defer module.Stop()
	endpoint, err := module.HTTPEndpoint()
	require.NoError(t, err)

	cases := map[string]struct {
		url     string
		modules map[string]string
		version bool
	}{
		"http": {endpoint, map[string]string{"eth": "1.0", "web3": "1.0", "rpc": "1.0"}, false},
		"ipc":  {config.IPCPath, map[string]string{"eth": "1.0", "web3": "1.0", "net": "1.0", "debug": "1.0", "rpc": "1.0"}, true},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			client, err := gethrpc.Dial(test.url)
			require.NoError(t, err)
			defer client.Close()
			var modules map[string]string
			require.NoError(t, client.Call(&modules, "rpc_modules"))
			require.Equal(t, test.modules, modules)
			var chainID string
			require.NoError(t, client.Call(&chainID, "eth_chainId"))
			require.Equal(t, "0xfa3", chainID)
			var version string
			err = client.Call(&version, "net_version")
			require.Equal(t, test.version, err == nil, "net is served where enabled")
		})
	}
}

func TestModule_StopLeavesNoGoroutineBehind(t *testing.T) {
	before := runtime.NumGoroutine()
	module := startModule(t, moduleConfig(t, nil))
	endpoint, err := module.HTTPEndpoint()
	require.NoError(t, err)
	client, err := gethrpc.Dial(endpoint)
	require.NoError(t, err)
	var number string
	require.NoError(t, client.CallContext(context.Background(), &number, "eth_blockNumber"))
	client.Close()
	module.Stop()
	for deadline := time.Now().Add(5 * time.Second); runtime.NumGoroutine() > before; time.Sleep(10 * time.Millisecond) {
		require.True(t, time.Now().Before(deadline), "goroutines left after Stop")
	}
}

func TestModule_WithoutHTTPHasNoEndpoint(t *testing.T) {
	module := startModule(t, rpc.DefaultConfig())
	defer module.Stop()
	_, err := module.HTTPEndpoint()
	require.EqualError(t, err, "the RPC module serves no HTTP endpoint")
}

func TestModule_FailsToStartOnAnEndpointInUse(t *testing.T) {
	first := startModule(t, moduleConfig(t, nil))
	defer first.Stop()
	endpoint, err := first.HTTPEndpoint()
	require.NoError(t, err)
	config := rpc.DefaultConfig()
	config.HTTP = &rpc.HTTPConfig{HTTPConfig: transport.HTTPConfig{Endpoint: endpoint[len("http://"):]}}
	second := rpc.New(config, tests.NewFakeCore(), gethrpc.NewServer(), prometheus.NewRegistry())
	require.Error(t, second.Start())
}

func TestModule_FailsToStartWithoutAHead(t *testing.T) {
	fake := tests.NewFakeCore()
	fake.SetMode("Head", tests.Fail)
	module := rpc.New(rpc.DefaultConfig(), fake, gethrpc.NewServer(), prometheus.NewRegistry())
	require.ErrorIs(t, module.Start(), tests.ErrInjected)
}
