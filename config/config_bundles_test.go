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
	"flag"
	"testing"

	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/0xsoniclabs/sonic/config/flags"
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/stretchr/testify/require"
	"gopkg.in/urfave/cli.v1"
)

func TestMakeAllConfigs_UsesDefaultProcessedBundlesRetention(t *testing.T) {
	cfg, err := MakeAllConfigsFromFile(newCliContextWithArgs(t), "")
	require.NoError(t, err)
	require.Equal(t, gossip.DefaultProcessedBundlesRetention, cfg.OperaStore.ProcessedBundlesRetention)
}

func TestMakeAllConfigs_AppliesProcessedBundlesRetentionFlag(t *testing.T) {
	ctx := newCliContextWithArgs(t, "--"+flags.ProcessedBundlesRetentionFlag.Name, "42")
	cfg, err := MakeAllConfigsFromFile(ctx, "")
	require.NoError(t, err)
	require.Equal(t, uint64(42), cfg.OperaStore.ProcessedBundlesRetention)
}

func TestMakeAllConfigs_NonArchiveNodeRetainsOnlyReplayProtectionWindow(t *testing.T) {
	ctx := newCliContextWithArgs(t, "--"+flags.ModeFlag.Name, "validator")
	cfg, err := MakeAllConfigsFromFile(ctx, "")
	require.NoError(t, err)
	require.Equal(t, carmen.NoArchive, cfg.OperaStore.EVM.StateDb.Archive)
	require.Equal(t, bundle.MaxBlockRangeLength, cfg.OperaStore.ProcessedBundlesRetention)
}

func TestMakeAllConfigs_RetentionFlagOverridesNonArchiveDefault(t *testing.T) {
	ctx := newCliContextWithArgs(t,
		"--"+flags.ModeFlag.Name, "validator",
		"--"+flags.ProcessedBundlesRetentionFlag.Name, "42",
	)
	cfg, err := MakeAllConfigsFromFile(ctx, "")
	require.NoError(t, err)
	require.Equal(t, uint64(42), cfg.OperaStore.ProcessedBundlesRetention)
}

// newCliContextWithArgs creates a CLI context with a temporary data directory
// and the given additional arguments.
func newCliContextWithArgs(t *testing.T, args ...string) *cli.Context {
	t.Helper()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String(flags.DataDirFlag.Name, "", "")
	fs.String(flags.ModeFlag.Name, "rpc", "")
	fs.Uint64(flags.ProcessedBundlesRetentionFlag.Name, 0, "")
	require.NoError(t, fs.Parse(append([]string{"--" + flags.DataDirFlag.Name, t.TempDir()}, args...)))
	return cli.NewContext(nil, fs, nil)
}
