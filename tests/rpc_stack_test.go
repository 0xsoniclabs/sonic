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

package tests

import (
	"maps"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/stretchr/testify/require"
)

// A node serves the namespaces of its RPC stack: over HTTP those of its API
// list, over IPC also the node's administration.
func TestRPCStack_ANodeServesTheNamespacesOfItsStack(t *testing.T) {
	cases := map[string]struct {
		http, ipc []string
	}{
		gossip.LegacyRPCStack: {
			http: []string{"admin", "dag", "debug", "eth", "net", "rpc", "sonic", "test", "trace", "txpool", "web3"},
			ipc:  []string{"abft", "admin", "dag", "debug", "eth", "ftm", "net", "personal", "rpc", "sonic", "test", "trace", "txpool", "web3"},
		},
		gossip.ModuleRPCStack: {
			http: []string{"dag", "debug", "eth", "net", "rpc", "sonic", "test", "trace", "txpool", "web3"},
			ipc:  []string{"abft", "admin", "dag", "debug", "eth", "net", "rpc", "sonic", "test", "trace", "txpool", "web3"},
		},
	}
	for stack, want := range cases {
		t.Run(stack, func(t *testing.T) {
			net := StartIntegrationTestNet(t, IntegrationTestNetOptions{RPCStack: stack})

			client, err := net.GetClient()
			require.NoError(t, err)
			defer client.Close()
			var modules map[string]string
			require.NoError(t, client.Client().Call(&modules, "rpc_modules"))
			require.Equal(t, want.http, slices.Sorted(maps.Keys(modules)))

			ws, err := net.GetWebSocketClient()
			require.NoError(t, err)
			defer ws.Close()
			require.NoError(t, ws.Client().Call(&modules, "rpc_modules"))
			require.Equal(t, want.http, slices.Sorted(maps.Keys(modules)), "WebSocket serves the namespaces of HTTP")

			ipc, err := net.GetIpcClient(0)
			require.NoError(t, err)
			defer ipc.Close()
			require.NoError(t, ipc.Call(&modules, "rpc_modules"))
			require.Equal(t, want.ipc, slices.Sorted(maps.Keys(modules)))
		})
	}
}

func TestRPCStackOfRun_IsTheLegacyStackUnlessSet(t *testing.T) {
	cases := map[string]struct {
		value, stack string
		valid        bool
	}{
		"unset":  {"", gossip.LegacyRPCStack, true},
		"legacy": {"legacy", gossip.LegacyRPCStack, true},
		"module": {"module", gossip.ModuleRPCStack, true},
		"other":  {"other", "", false},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv(rpcStackVariable, test.value)
			stack, err := RPCStackOfRun()
			require.Equal(t, test.valid, err == nil, "%v", err)
			require.Equal(t, test.stack, stack)
		})
	}
}
