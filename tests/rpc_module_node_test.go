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
	"context"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/console"
	"github.com/stretchr/testify/require"
)

// TestConsole_OverIPCReachesTheModuleAndTheNode is the IPC console test of
// 7.5: a console attached to the IPC socket, as `sonictool cli` attaches it,
// calls the module's methods and those of the node's own handler.
func TestConsole_OverIPCReachesTheModuleAndTheNode(t *testing.T) {
	net := StartIntegrationTestNet(t)
	client, err := net.GetIpcClientConnectedToNode(0)
	require.NoError(t, err)
	defer client.Close()

	var out bytes.Buffer
	c, err := console.New(console.Config{DataDir: t.TempDir(), DocRoot: t.TempDir(), Client: client, Printer: &out})
	require.NoError(t, err)
	defer func() { _ = c.Stop(false) }()

	evaluate := func(statement string) string {
		out.Reset()
		c.Evaluate(statement)
		return strings.TrimSpace(out.String())
	}
	require.Equal(t, `"`+hexutil.EncodeBig(net.GetChainId())+`"`, evaluate("eth.chainId()"), "a method of the module")
	require.Contains(t, evaluate("admin.nodeInfo.enode"), "enode://", "a method of the node's handler")
	require.Contains(t, evaluate("rpc.modules"), "admin", "the namespaces of both")
	require.Contains(t, evaluate("rpc.modules"), "eth")
	accounts := evaluate("JSON.stringify(eth.accounts)")
	require.True(t, strings.HasPrefix(accounts, `"[`), "eth_accounts, a legacy method served by the node's handler: %s", accounts)
}

// TestNodeStop_UnderRPCLoadFinishes is the shutdown test of Q1 and D10: the
// node stops, and restarts, while clients keep calling it.
func TestNodeStop_UnderRPCLoadFinishes(t *testing.T) {
	net := StartIntegrationTestNet(t)
	_, err := net.EndowAccount(common.Address{42}, big.NewInt(100))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for range 8 {
		client, err := net.GetClient()
		require.NoError(t, err)
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer client.Close()
			for ctx.Err() == nil {
				_, _ = client.BalanceAt(ctx, common.Address{42}, nil)
				_, _ = client.BlockByNumber(ctx, nil)
			}
		}()
	}
	time.Sleep(time.Second)
	start := time.Now()
	require.NoError(t, net.Restart())
	require.Less(t, time.Since(start), 2*time.Minute)
	cancel()
	wg.Wait()

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()
	balance, err := client.BalanceAt(context.Background(), common.Address{42}, nil)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(100), balance)
}
