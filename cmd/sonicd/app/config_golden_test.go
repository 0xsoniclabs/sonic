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
	"bufio"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/node"
	"github.com/stretchr/testify/require"
	cli "gopkg.in/urfave/cli.v1"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/config/flags"
	"github.com/0xsoniclabs/sonic/rpc"
	"github.com/0xsoniclabs/sonic/rpc/transport"
)

func readLines(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	require.NoError(t, scanner.Err())
	return lines
}

// TestFlags_EveryFlagOfTheReferenceIsAccepted is O4 for the command line.
func TestFlags_EveryFlagOfTheReferenceIsAccepted(t *testing.T) {
	var names []string
	for _, f := range initApp().Flags {
		for _, name := range strings.Split(f.GetName(), ",") {
			names = append(names, strings.TrimSpace(name))
		}
	}
	for _, name := range readLines(t, filepath.Join("testdata", "golden", "flags-2.2.5.txt")) {
		if name == "help" {
			continue
		}
		require.Contains(t, names, name, "flag --%s of the reference", name)
	}
}

// tomlKeys lists the keys of a TOML file as section.key.
func tomlKeys(t *testing.T, path string) []string {
	t.Helper()
	section := ""
	var keys []string
	for _, line := range readLines(t, path) {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "["):
			section = strings.Trim(line, "[]")
		case strings.Contains(line, " = "):
			keys = append(keys, section+"."+strings.SplitN(line, " = ", 2)[0])
		}
	}
	return keys
}

// TestTOML_TheConfigurationOfTheReferenceLoadsAndKeepsItsKeys is O4 for the
// configuration file: the loader rejects unknown fields, so the reference's
// file loads only if every key is still known, and the node writes every key
// the reference wrote.
func TestTOML_TheConfigurationOfTheReferenceLoadsAndKeepsItsKeys(t *testing.T) {
	golden := filepath.Join("testdata", "golden", "config-2.2.5.toml")
	var cfg config.Config
	require.NoError(t, config.LoadAllConfigs(golden, &cfg))
	require.Equal(t, uint64(50000000), cfg.Opera.RPCGasCap)
	require.Equal(t, uint(1000), cfg.Opera.FilterAPI.LogQueryParameterLimit)
	require.Equal(t, uint64(500000), cfg.Opera.GPO.DefaultCertainty)

	dumped := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, config.SaveAllConfigs(dumped, &cfg))
	ours := tomlKeys(t, dumped)
	for _, key := range tomlKeys(t, golden) {
		require.Contains(t, ours, key, "TOML key %s of the reference", key)
	}
}

// configFromArgs builds the node configuration of a command line, as sonicd
// does.
func configFromArgs(t *testing.T, args ...string) *config.Config {
	t.Helper()
	var cfg *config.Config
	app := cli.NewApp()
	app.Flags = initApp().Flags
	app.Action = func(ctx *cli.Context) error {
		var err error
		cfg, err = config.MakeAllConfigs(ctx)
		return err
	}
	cmdLine := append([]string{"sonicd", "--" + flags.DataDirFlag.Name, t.TempDir()}, args...)
	require.NoError(t, app.Run(cmdLine))
	return cfg
}

// moduleConfig is the module's configuration of a node configuration.
func moduleConfig(t *testing.T, cfg *config.Config) rpc.Config {
	t.Helper()
	ep := takeOverRPC(&cfg.Node)
	stack, err := node.New(&node.Config{DataDir: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(func() { _ = stack.Close() })
	return rpcConfig(cfg, ep, stack)
}

// TestFlags_EachFlagTheModuleConsumesTakesEffect is O4 for the flags of the
// module: setting each changes the module's configuration.
func TestFlags_EachFlagTheModuleConsumesTakesEffect(t *testing.T) {
	tests := map[string]struct {
		args  []string
		check func(t *testing.T, rc rpc.Config)
	}{
		"rpc.gascap":                    {[]string{"--rpc.gascap", "7"}, func(t *testing.T, rc rpc.Config) { require.Equal(t, uint64(7), rc.Exec.GasCap) }},
		"rpc.evmtimeout":                {[]string{"--rpc.evmtimeout", "3s"}, func(t *testing.T, rc rpc.Config) { require.Equal(t, 3*time.Second, rc.Exec.EVMTimeout) }},
		"rpc.txfeecap":                  {[]string{"--rpc.txfeecap", "2.5"}, func(t *testing.T, rc rpc.Config) { require.Equal(t, 2.5, rc.Exec.TxFeeCap) }},
		"rpc.timeout":                   {[]string{"--rpc.timeout", "9s"}, func(t *testing.T, rc rpc.Config) { require.Equal(t, 9*time.Second, rc.Dispatch.RequestTimeout) }},
		"rpc.log-query-parameter-limit": {[]string{"--rpc.log-query-parameter-limit", "3"}, func(t *testing.T, rc rpc.Config) { require.Equal(t, uint(3), rc.Exec.LogQueryParameterLimit) }},
		"rpc.log-query-result-limit":    {[]string{"--rpc.log-query-result-limit", "4"}, func(t *testing.T, rc rpc.Config) { require.Equal(t, uint(4), rc.Exec.LogQueryResultLimit) }},
		"rpc.batch-request-limit":       {[]string{"--rpc.batch-request-limit", "5"}, func(t *testing.T, rc rpc.Config) { require.Equal(t, 5, rc.Transport.Limits.BatchRequestLimit) }},
		"rpc.batch-response-max-size":   {[]string{"--rpc.batch-response-max-size", "6"}, func(t *testing.T, rc rpc.Config) { require.Equal(t, 6, rc.Transport.Limits.BatchResponseMaxSize) }},
		"rpc.maxresponsesize":           {[]string{"--rpc.maxresponsesize", "8"}, func(t *testing.T, rc rpc.Config) { require.Equal(t, 8, rc.Exec.MaxResponseSize) }},
		"rpc.tracefilterrange":          {[]string{"--rpc.tracefilterrange", "9"}, func(t *testing.T, rc rpc.Config) { require.Equal(t, uint64(9), rc.Exec.MaxTraceFilterRange) }},
		"rpc.structloglimit":            {[]string{"--rpc.structloglimit", "10"}, func(t *testing.T, rc rpc.Config) { require.Equal(t, 10, rc.Exec.StructLogLimit) }},
		"rpc.allow-js-tracers":          {[]string{"--rpc.allow-js-tracers"}, func(t *testing.T, rc rpc.Config) { require.True(t, rc.Exec.AllowJSTracers) }},
		"http": {[]string{"--http", "--http.addr", "127.0.0.2", "--http.port", "1234", "--http.corsdomain", "a.example", "--http.vhosts", "b.example", "--http.api", "eth,txpool", "--http.rpcprefix", "/rpc"}, func(t *testing.T, rc rpc.Config) {
			require.Equal(t, &transport.HTTPConfig{Host: "127.0.0.2", Port: 1234, Cors: []string{"a.example"}, Vhosts: []string{"b.example"}, Prefix: "/rpc",
				Timeouts: rc.Transport.HTTP.Timeouts, FallbackTimeouts: rc.Transport.HTTP.FallbackTimeouts}, rc.Transport.HTTP)
			require.Equal(t, []string{"eth", "txpool"}, rc.Front.Modules[transport.HTTP])
			require.Equal(t, 30*time.Second, rc.Transport.HTTP.Timeouts.Write)
		}},
		"ws": {[]string{"--ws", "--ws.addr", "127.0.0.3", "--ws.port", "4321", "--ws.api", "eth", "--ws.origins", "c.example", "--ws.rpcprefix", "/ws"}, func(t *testing.T, rc rpc.Config) {
			require.Equal(t, &transport.WSConfig{Host: "127.0.0.3", Port: 4321, Origins: []string{"c.example"}, Prefix: "/ws"}, rc.Transport.WS)
			require.Equal(t, []string{"eth"}, rc.Front.Modules[transport.WS])
		}},
		"ipcpath":    {[]string{"--ipcpath", "/run/other.ipc"}, func(t *testing.T, rc rpc.Config) { require.Equal(t, "/run/other.ipc", rc.Transport.IPC) }},
		"ipcdisable": {[]string{"--ipcdisable"}, func(t *testing.T, rc rpc.Config) { require.Empty(t, rc.Transport.IPC) }},
	}
	defaults := moduleConfig(t, configFromArgs(t))
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			rc := moduleConfig(t, configFromArgs(t, tc.args...))
			tc.check(t, rc)
			require.NotEqual(t, defaults, rc, "the flag changes nothing")
		})
	}
	require.Nil(t, defaults.Transport.HTTP, "no endpoint by default")
	require.NotEmpty(t, defaults.Transport.IPC, "IPC by default")
}

// TestTOML_EachKeyTheModuleConsumesTakesEffect is O4 for the TOML keys of the
// module, and section 8: the node's default of each knob is the module's.
func TestTOML_EachKeyTheModuleConsumesTakesEffect(t *testing.T) {
	keys := map[string]struct {
		set   func(*config.Config)
		check func(t *testing.T, rc rpc.Config)
	}{
		"Opera.RPCGasCap":            {func(c *config.Config) { c.Opera.RPCGasCap = 1 }, func(t *testing.T, rc rpc.Config) { require.Equal(t, uint64(1), rc.Exec.GasCap) }},
		"Opera.RPCEVMTimeout":        {func(c *config.Config) { c.Opera.RPCEVMTimeout = 2 }, func(t *testing.T, rc rpc.Config) { require.Equal(t, time.Duration(2), rc.Exec.EVMTimeout) }},
		"Opera.RPCTxFeeCap":          {func(c *config.Config) { c.Opera.RPCTxFeeCap = 3 }, func(t *testing.T, rc rpc.Config) { require.Equal(t, 3.0, rc.Exec.TxFeeCap) }},
		"Opera.RPCTimeout":           {func(c *config.Config) { c.Opera.RPCTimeout = 4 }, func(t *testing.T, rc rpc.Config) { require.Equal(t, time.Duration(4), rc.Dispatch.RequestTimeout) }},
		"Opera.TxSyncDefaultTimeout": {func(c *config.Config) { c.Opera.TxSyncDefaultTimeout = 5 }, func(t *testing.T, rc rpc.Config) { require.Equal(t, time.Duration(5), rc.Exec.TxSyncDefaultTimeout) }},
		"Opera.TxSyncMaxTimeout":     {func(c *config.Config) { c.Opera.TxSyncMaxTimeout = 6 }, func(t *testing.T, rc rpc.Config) { require.Equal(t, time.Duration(6), rc.Exec.TxSyncMaxTimeout) }},
		"Opera.AllowUnprotectedTxs":  {func(c *config.Config) { c.Opera.AllowUnprotectedTxs = true }, func(t *testing.T, rc rpc.Config) { require.True(t, rc.Exec.AllowUnprotectedTxs) }},
		"Opera.MaxResponseSize":      {func(c *config.Config) { c.Opera.MaxResponseSize = 7 }, func(t *testing.T, rc rpc.Config) { require.Equal(t, 7, rc.Exec.MaxResponseSize) }},
		"Opera.MaxTraceFilterRange":  {func(c *config.Config) { c.Opera.MaxTraceFilterRange = 8 }, func(t *testing.T, rc rpc.Config) { require.Equal(t, uint64(8), rc.Exec.MaxTraceFilterRange) }},
		"Opera.StructLogLimit":       {func(c *config.Config) { c.Opera.StructLogLimit = 9 }, func(t *testing.T, rc rpc.Config) { require.Equal(t, 9, rc.Exec.StructLogLimit) }},
		"Opera.AllowJSTracers":       {func(c *config.Config) { c.Opera.AllowJSTracers = true }, func(t *testing.T, rc rpc.Config) { require.True(t, rc.Exec.AllowJSTracers) }},
		"Opera.FilterAPI.IndexedLogsBlockRangeLimit": {func(c *config.Config) { c.Opera.FilterAPI.IndexedLogsBlockRangeLimit = 10 }, func(t *testing.T, rc rpc.Config) {
			require.Equal(t, uint64(10), rc.Exec.IndexedLogsBlockRangeLimit)
		}},
		"Opera.FilterAPI.UnindexedLogsBlockRangeLimit": {func(c *config.Config) { c.Opera.FilterAPI.UnindexedLogsBlockRangeLimit = 11 }, func(t *testing.T, rc rpc.Config) {
			require.Equal(t, uint64(11), rc.Exec.UnindexedLogsBlockRangeLimit)
		}},
		"Opera.FilterAPI.LogQueryParameterLimit": {func(c *config.Config) { c.Opera.FilterAPI.LogQueryParameterLimit = 12 }, func(t *testing.T, rc rpc.Config) { require.Equal(t, uint(12), rc.Exec.LogQueryParameterLimit) }},
		"Opera.FilterAPI.LogQueryResultLimit":    {func(c *config.Config) { c.Opera.FilterAPI.LogQueryResultLimit = 13 }, func(t *testing.T, rc rpc.Config) { require.Equal(t, uint(13), rc.Exec.LogQueryResultLimit) }},
		"Opera.GPO.MaxGasPrice":                  {func(c *config.Config) { c.Opera.GPO.MaxGasPrice = big.NewInt(14) }, func(t *testing.T, rc rpc.Config) { require.Equal(t, big.NewInt(14), rc.Oracle.MaxGasPrice) }},
		"Opera.GPO.MinGasPrice":                  {func(c *config.Config) { c.Opera.GPO.MinGasPrice = big.NewInt(15) }, func(t *testing.T, rc rpc.Config) { require.Equal(t, big.NewInt(15), rc.Oracle.MinGasPrice) }},
		"Opera.GPO.DefaultCertainty":             {func(c *config.Config) { c.Opera.GPO.DefaultCertainty = 16 }, func(t *testing.T, rc rpc.Config) { require.Equal(t, uint64(16), rc.Oracle.DefaultCertainty) }},
		"Node.BatchRequestLimit":                 {func(c *config.Config) { c.Node.BatchRequestLimit = 17 }, func(t *testing.T, rc rpc.Config) { require.Equal(t, 17, rc.Transport.Limits.BatchRequestLimit) }},
		"Node.BatchResponseMaxSize":              {func(c *config.Config) { c.Node.BatchResponseMaxSize = 18 }, func(t *testing.T, rc rpc.Config) { require.Equal(t, 18, rc.Transport.Limits.BatchResponseMaxSize) }},
		"Node.HTTPTimeouts": {func(c *config.Config) {
			c.Node.HTTPHost = "127.0.0.1"
			c.Node.HTTPTimeouts.WriteTimeout = 19 * time.Second
		}, func(t *testing.T, rc rpc.Config) {
			require.Equal(t, 19*time.Second, rc.Transport.HTTP.Timeouts.Write)
		}},
	}
	for name, k := range keys {
		t.Run(name, func(t *testing.T) {
			cfg := configFromArgs(t)
			k.set(cfg)
			k.check(t, moduleConfig(t, cfg))
		})
	}

	// Section 8: the node's defaults are the module's definitions.
	rc, want := moduleConfig(t, configFromArgs(t)), rpc.DefaultConfig()
	require.Equal(t, want.Exec, rc.Exec)
	require.Equal(t, want.Dispatch, rc.Dispatch)
	require.Equal(t, want.Oracle, rc.Oracle)
	require.Equal(t, want.Chain, rc.Chain)
	require.True(t, slices.Equal(rc.Front.ForwardEverywhere, []string{"net_listening", "net_peerCount"}))
}

// TestMode_AValidatorKeepsTheEndpointsOfTheReference is O3. In the reference,
// --mode validator drops the archive and the indexes and leaves the endpoints
// as configured; the module takes them over the same way, and serves the
// committed head when there is no archive (rpcCore.Head).
func TestMode_AValidatorKeepsTheEndpointsOfTheReference(t *testing.T) {
	for _, mode := range []string{"validator", "rpc"} {
		cfg := configFromArgs(t, "--mode", mode, "--http", "--ws")
		rc := moduleConfig(t, cfg)
		require.NotNil(t, rc.Transport.HTTP, mode)
		require.NotNil(t, rc.Transport.WS, mode)
		require.Equal(t, mode == "validator", cfg.OperaStore.EVM.StateDb.Archive == "none", mode)
	}
	cfg := configFromArgs(t, "--mode", "validator")
	rc := moduleConfig(t, cfg)
	require.Nil(t, rc.Transport.HTTP, "no public endpoint unless asked for")
	require.Nil(t, rc.Transport.WS)
}
