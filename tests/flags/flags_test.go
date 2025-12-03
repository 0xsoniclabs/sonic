// Copyright 2025 Sonic Operations Ltd
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

package flags

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	sonicd "github.com/0xsoniclabs/sonic/cmd/sonicd/app"
	"github.com/0xsoniclabs/sonic/tests"

	"github.com/stretchr/testify/require"
)

func TestSonicTool_DefaultConfig_HasDefaultValues(t *testing.T) {

	net := tests.StartIntegrationTestNet(t)
	net.Stop()

	configFile := filepath.Join(net.GetDirectory(), "config.toml")
	require.NoError(t, sonicd.RunWithArgs(
		[]string{"sonicd",
			"--datadir", net.GetDirectory() + "/state",
			"--dump-config", configFile}, nil))

	f, err := os.Open(configFile)
	require.NoError(t, err)
	configFromFile, err := io.ReadAll(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	require.Contains(t, string(configFromFile), "ThrottleEvents = false")
	require.Contains(t, string(configFromFile), "ThrottlerDominantThreshold = 7.5e-01")
	require.Contains(t, string(configFromFile), "ThrottlerSkipInSameFrame = 3")
}

func TestSonicTool_CustomThrottlerConfig_AreApplied(t *testing.T) {

	net := tests.StartIntegrationTestNet(t)
	net.Stop()

	configFile := filepath.Join(net.GetDirectory(), "config.toml")
	require.NoError(t, sonicd.RunWithArgs(
		[]string{"sonicd",
			"--datadir", net.GetDirectory() + "/state",
			"--dump-config", configFile,
			"--emitter.throttle-events",
			"--emitter.throttle-dominant-threshold", "0.85",
			"--emitter.throttle-skip-in-same-frame", "5"}, nil))

	f, err := os.Open(configFile)
	require.NoError(t, err)
	configFromFile, err := io.ReadAll(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	require.Contains(t, string(configFromFile), "ThrottleEvents = true")
	require.Contains(t, string(configFromFile), "ThrottlerDominantThreshold = 8.5e-01")
	require.Contains(t, string(configFromFile), "ThrottlerSkipInSameFrame = 5")
}
