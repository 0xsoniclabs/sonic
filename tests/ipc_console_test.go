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
	"bytes"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/console"
	"github.com/stretchr/testify/require"
)

// The JavaScript console of `sonictool attach` talks to the node over the IPC
// socket: it lists the modules and calls methods of the RPC module as well as
// those of the node's own handler.
func TestIpcConsole_ReachesTheModuleAndTheNodeHandler(t *testing.T) {
	net := StartIntegrationTestNet(t)

	client, err := net.GetIpcClient()
	require.NoError(t, err)
	defer client.Close()

	var printed bytes.Buffer
	c, err := console.New(console.Config{
		DataDir: t.TempDir(),
		DocRoot: ".",
		Client:  client,
		Printer: &printed,
	})
	require.NoError(t, err)
	defer func() { require.NoError(t, c.Stop(false)) }()

	for _, module := range []string{"eth", "net", "web3", "debug", "txpool", "admin"} {
		c.Evaluate("console.log('module " + module + " ' + typeof " + module + ")")
	}
	c.Evaluate("console.log('block', eth.blockNumber)")
	c.Evaluate("console.log('chain', eth.chainId())")
	c.Evaluate("console.log('peers', admin.peers.length)")
	c.Evaluate("console.log('version', web3.version.node)")

	out := printed.String()
	for _, module := range []string{"eth", "net", "web3", "debug", "txpool", "admin"} {
		require.Contains(t, out, "module "+module+" object", out)
	}

	// The console binds the modules it knows; the node lists all it serves.
	var modules map[string]string
	require.NoError(t, client.Call(&modules, "rpc_modules"))
	for _, module := range []string{"eth", "ftm", "net", "web3", "debug", "trace", "txpool", "sonic", "admin", "dag", "abft", "personal"} {
		require.Contains(t, modules, module)
	}
	require.Contains(t, out, "block ")
	require.NotContains(t, strings.ToLower(out), "error")
	require.Contains(t, out, "peers ")
}
