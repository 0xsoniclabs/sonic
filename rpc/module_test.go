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
	"net"
	"testing"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/0xsoniclabs/sonic/rpc/transport"
)

func newTestModule(t *testing.T, address string) *Module {
	t.Helper()
	config := DefaultConfig()
	config.ClientVersion = "Sonic/test"
	config.HTTP = &HTTPEndpoint{Address: address, HTTPConfig: transport.HTTPConfig{VirtualHosts: []string{"localhost"}}}
	return New(config, tests.NewFakeCore(t), prometheus.NewRegistry())
}

func TestModule_ServesHTTPUntilItStops(t *testing.T) {
	module := newTestModule(t, "127.0.0.1:0")
	require.True(t, module.ServesExternally())
	require.NoError(t, module.Start())
	client, err := gethrpc.Dial(module.HTTPEndpoint())
	require.NoError(t, err)
	defer client.Close()
	var version string
	require.NoError(t, client.Call(&version, "web3_clientVersion"))
	require.Equal(t, "Sonic/test", version)

	require.NoError(t, module.Stop())
	// Another server may take the freed port; it is not this module.
	err = client.Call(&version, "web3_clientVersion")
	require.True(t, err != nil || version != "Sonic/test", "the module answers after it stopped")
}

func TestModule_WithoutEndpointsServesNothingExternally(t *testing.T) {
	module := New(DefaultConfig(), tests.NewFakeCore(t), prometheus.NewRegistry())
	require.False(t, module.ServesExternally())
	require.Empty(t, module.HTTPEndpoint())
	require.NoError(t, module.Start())
	require.NoError(t, module.Stop())
}

func TestModule_StartFailsOnATakenAddress(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { require.NoError(t, listener.Close()) }()
	module := newTestModule(t, listener.Addr().String())
	require.ErrorContains(t, module.Start(), "failed to start the HTTP endpoint")
}

func TestModule_StartFailsWithoutAHead(t *testing.T) {
	fake := tests.NewFakeCore(t)
	fake.SetMode("Head", tests.Fail)
	module := New(DefaultConfig(), fake, prometheus.NewRegistry())
	require.ErrorContains(t, module.Start(), "failed to start the RPC module")
}

type stubForwarder struct{}

func (stubForwarder) Forward(context.Context, string, []json.RawMessage) (json.RawMessage, error) {
	return json.RawMessage(`"from the node"`), nil
}

func TestModule_ForwardsWhatItDoesNotServe(t *testing.T) {
	module := newTestModule(t, "127.0.0.1:0")
	module.Forward(stubForwarder{})
	require.NoError(t, module.Start())
	defer func() { require.NoError(t, module.Stop()) }()
	client, err := gethrpc.Dial(module.HTTPEndpoint())
	require.NoError(t, err)
	defer client.Close()
	var answer string
	require.NoError(t, client.Call(&answer, "dag_getHeads"))
	require.Equal(t, "from the node", answer)
}
