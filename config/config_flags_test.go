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

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/node"
	"github.com/stretchr/testify/require"
	cli "gopkg.in/urfave/cli.v1"

	"github.com/0xsoniclabs/sonic/config/flags"
)

// makeAllConfigsFromArgs runs MakeAllConfigs with the given command line
// arguments. If tomlContent is not empty, it is written into a config file
// which is passed to the --config flag.
func makeAllConfigsFromArgs(t *testing.T, tomlContent string, args ...string) *Config {
	t.Helper()

	cmdLine := []string{"sonicd", "--" + flags.DataDirFlag.Name, t.TempDir()}
	if tomlContent != "" {
		configFile := filepath.Join(t.TempDir(), "config.toml")
		require.NoError(t, os.WriteFile(configFile, []byte(tomlContent), 0600))
		cmdLine = append(cmdLine, "--"+flags.ConfigFileFlag.Name, configFile)
	}
	cmdLine = append(cmdLine, args...)

	var cfg *Config
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		flags.DataDirFlag,
		flags.ConfigFileFlag,
		flags.EnableThrottlingFlag,
		flags.BatchRequestLimit,
		flags.BatchResponseMaxSize,
	}
	app.Action = func(ctx *cli.Context) error {
		var err error
		cfg, err = MakeAllConfigs(ctx)
		return err
	}
	require.NoError(t, app.Run(cmdLine))
	require.NotNil(t, cfg)
	return cfg
}

func TestMakeAllConfigs_EventThrottler_FlagOverridesConfigFileOverridesDefault(t *testing.T) {
	const (
		tomlEnabled  = "[Emitter.ThrottlerConfig]\nEnabled = true\n"
		tomlDisabled = "[Emitter.ThrottlerConfig]\nEnabled = false\n"
	)

	tests := map[string]struct {
		toml     string
		args     []string
		expected bool
	}{
		"default is enabled": {
			expected: true,
		},
		"config file disables": {
			toml:     tomlDisabled,
			expected: false,
		},
		"config file enables": {
			toml:     tomlEnabled,
			expected: true,
		},
		"flag disables": {
			args:     []string{"--event-throttler=false"},
			expected: false,
		},
		"flag enables": {
			args:     []string{"--event-throttler=true"},
			expected: true,
		},
		"bare flag enables": {
			args:     []string{"--event-throttler"},
			expected: true,
		},
		"flag disables over config file enabling": {
			toml:     tomlEnabled,
			args:     []string{"--event-throttler=false"},
			expected: false,
		},
		"flag enables over config file disabling": {
			toml:     tomlDisabled,
			args:     []string{"--event-throttler=true"},
			expected: true,
		},
		"bare flag enables over config file disabling": {
			toml:     tomlDisabled,
			args:     []string{"--event-throttler"},
			expected: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := makeAllConfigsFromArgs(t, test.toml, test.args...)
			require.Equal(t, test.expected, cfg.Emitter.ThrottlerConfig.Enabled)
		})
	}
}

func TestMakeAllConfigs_RpcBatchLimits_DefaultsMatchGethDefaults(t *testing.T) {
	cfg := makeAllConfigsFromArgs(t, "")

	require.NotZero(t, node.DefaultConfig.BatchRequestLimit)
	require.NotZero(t, node.DefaultConfig.BatchResponseMaxSize)
	require.Equal(t, node.DefaultConfig.BatchRequestLimit, cfg.Node.BatchRequestLimit)
	require.Equal(t, node.DefaultConfig.BatchResponseMaxSize, cfg.Node.BatchResponseMaxSize)
}

func TestMakeAllConfigs_RpcBatchLimits_FlagDefaultsMatchConfigDefaults(t *testing.T) {
	cfg := makeAllConfigsFromArgs(t, "")

	require.Equal(t, flags.BatchRequestLimit.Value, cfg.Node.BatchRequestLimit)
	require.Equal(t, flags.BatchResponseMaxSize.Value, cfg.Node.BatchResponseMaxSize)
}

func TestMakeAllConfigs_RpcBatchLimits_FlagOverridesConfigFileOverridesDefault(t *testing.T) {
	const toml = "[Node]\nBatchRequestLimit = 11\nBatchResponseMaxSize = 2222\n"

	tests := map[string]struct {
		toml                 string
		args                 []string
		expectedRequestLimit int
		expectedResponseSize int
	}{
		"config file sets both": {
			toml:                 toml,
			expectedRequestLimit: 11,
			expectedResponseSize: 2222,
		},
		"flags set both": {
			args:                 []string{"--rpc.batch-request-limit=33", "--rpc.batch-response-max-size=4444"},
			expectedRequestLimit: 33,
			expectedResponseSize: 4444,
		},
		"flags override config file": {
			toml:                 toml,
			args:                 []string{"--rpc.batch-request-limit=33", "--rpc.batch-response-max-size=4444"},
			expectedRequestLimit: 33,
			expectedResponseSize: 4444,
		},
		"single flag overrides only its own value": {
			toml:                 toml,
			args:                 []string{"--rpc.batch-request-limit=33"},
			expectedRequestLimit: 33,
			expectedResponseSize: 2222,
		},
		"flag can disable limits": {
			args:                 []string{"--rpc.batch-request-limit=0", "--rpc.batch-response-max-size=0"},
			expectedRequestLimit: 0,
			expectedResponseSize: 0,
		},
		"config file can disable limits": {
			toml:                 "[Node]\nBatchRequestLimit = 0\nBatchResponseMaxSize = 0\n",
			expectedRequestLimit: 0,
			expectedResponseSize: 0,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := makeAllConfigsFromArgs(t, test.toml, test.args...)
			require.Equal(t, test.expectedRequestLimit, cfg.Node.BatchRequestLimit)
			require.Equal(t, test.expectedResponseSize, cfg.Node.BatchResponseMaxSize)
		})
	}
}
