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
	"context"
	"math/big"
	"net"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/node"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/api/legacy"
	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/rpc"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/0xsoniclabs/sonic/rpc/transport"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// TestTakeOverRPC_EveryEndpointLayout is O5: the module takes the endpoints of
// every layout go-ethereum's node distinguishes, and the node opens none.
func TestTakeOverRPC_EveryEndpointLayout(t *testing.T) {
	tests := map[string]struct {
		cfg      node.Config
		http, ws bool
		ipc      string
		external bool
	}{
		"nothing":            {cfg: node.Config{}, external: false},
		"http":               {cfg: node.Config{HTTPHost: "127.0.0.1", HTTPPort: 8545}, http: true, external: true},
		"ws":                 {cfg: node.Config{WSHost: "127.0.0.1", WSPort: 8546}, ws: true, external: true},
		"http and ws":        {cfg: node.Config{HTTPHost: "127.0.0.1", HTTPPort: 8545, WSHost: "127.0.0.1", WSPort: 8546}, http: true, ws: true, external: true},
		"one port":           {cfg: node.Config{HTTPHost: "127.0.0.1", HTTPPort: 8545, WSHost: "0.0.0.0", WSPort: 8545}, http: true, ws: true, external: true},
		"ipc in the datadir": {cfg: node.Config{DataDir: "/data", IPCPath: "sonic.ipc"}, ipc: "/data/sonic.ipc"},
		"ipc absolute":       {cfg: node.Config{DataDir: "/data", IPCPath: "/run/sonic.ipc"}, ipc: "/run/sonic.ipc"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := tc.cfg
			cfg.HTTPModules, cfg.WSModules = []string{"eth"}, []string{"eth", "debug"}
			ep := takeOverRPC(&cfg)
			require.Equal(t, tc.http, ep.transport.HTTP != nil)
			require.Equal(t, tc.ws, ep.transport.WS != nil)
			require.Equal(t, tc.ipc, ep.transport.IPC)
			require.Equal(t, tc.external, ep.external)
			require.Equal(t, []string{"eth"}, ep.modules[transport.HTTP])
			require.Equal(t, []string{"eth", "debug"}, ep.modules[transport.WS])
			if tc.http {
				require.Equal(t, tc.cfg.HTTPHost, ep.transport.HTTP.Host)
				require.Equal(t, tc.cfg.HTTPPort, ep.transport.HTTP.Port)
			}
			if tc.ws {
				require.Equal(t, tc.cfg.WSHost, ep.transport.WS.Host)
			}
			require.Empty(t, cfg.HTTPHost)
			require.Empty(t, cfg.WSHost)
			require.Empty(t, cfg.IPCEndpoint())
			require.False(t, cfg.ExtRPCEnabled(), "the node serves no endpoint itself")
		})
	}
}

func testConfig() *config.Config {
	return &config.Config{Node: config.DefaultNodeConfig(), Opera: gossip.DefaultConfig(cachescale.Identity)}
}

// TestTakeOverRPC_TheModuleServesEveryLayout starts the module in each layout
// and calls it over every endpoint it opened.
func TestTakeOverRPC_TheModuleServesEveryLayout(t *testing.T) {
	shared := freePort(t)
	layouts := map[string]node.Config{
		"http":        {HTTPHost: "127.0.0.1", HTTPPort: freePort(t)},
		"ws":          {WSHost: "127.0.0.1", WSPort: freePort(t)},
		"http and ws": {HTTPHost: "127.0.0.1", HTTPPort: freePort(t), WSHost: "127.0.0.1", WSPort: freePort(t)},
		"one port":    {HTTPHost: "127.0.0.2", HTTPPort: shared, WSHost: "127.0.0.1", WSPort: shared},
		"ipc":         {IPCPath: "sonic.ipc"},
	}
	for name, layout := range layouts {
		t.Run(name, func(t *testing.T) {
			cfg := testConfig()
			layout.DataDir = t.TempDir()
			layout.HTTPVirtualHosts = []string{"*"}
			layout.WSOrigins = []string{"*"}
			cfg.Node = layout
			ep := takeOverRPC(&cfg.Node)
			stack, err := node.New(&cfg.Node)
			require.NoError(t, err)
			defer stack.Close()
			module := rpc.New(tests.NewFakeCore(), rpcConfig(cfg, ep, stack), prometheus.NewRegistry())
			require.NoError(t, module.Start())
			defer func() { _ = module.Stop() }()
			urls := []string{module.HTTPEndpoint(), module.WSEndpoint(), module.IPCEndpoint()}
			opened := 0
			for _, url := range urls {
				if url == "" {
					continue
				}
				opened++
				client, err := gethrpc.Dial(url)
				require.NoError(t, err, url)
				var id hexutil.Big
				require.NoError(t, client.Call(&id, "eth_chainId"), url)
				require.Equal(t, int64(tests.ChainID), id.ToInt().Int64())
				client.Close()
			}
			require.Positive(t, opened)
			if name == "one port" {
				require.Equal(t, "ws://127.0.0.1:"+strconv.Itoa(shared), module.WSEndpoint(), "the shared endpoint listens on the WebSocket host")
			}
			if name == "ipc" {
				require.Equal(t, filepath.Join(layout.DataDir, "sonic.ipc"), module.IPCEndpoint())
			}
		})
	}
}

// TestModuleChainReader_ReadsTheChainThroughTheModule is O5 for the node's
// own reader of the chain, the self-derivation of hardware wallets.
func TestModuleChainReader_ReadsTheChainThroughTheModule(t *testing.T) {
	r := tests.NewReplayChain(t)
	module := rpc.New(r.Core, rpc.DefaultConfig(), prometheus.NewRegistry())
	require.NoError(t, module.Start())
	defer func() { _ = module.Stop() }()
	reader := moduleChainReader{module}
	ctx := context.Background()

	balance, err := reader.BalanceAt(ctx, r.Writer, nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), balance.Int64())
	code, err := reader.CodeAt(ctx, r.Writer, big.NewInt(1))
	require.NoError(t, err)
	require.Equal(t, tests.StorageWritingCode, code)
	nonce, err := reader.NonceAt(ctx, r.Wallet.Address, nil)
	require.NoError(t, err)
	require.Zero(t, nonce)
	slot, err := reader.StorageAt(ctx, r.Writer, common.Hash{}, nil)
	require.NoError(t, err)
	require.Len(t, slot, 32)
	_, err = reader.BalanceAt(ctx, r.Writer, big.NewInt(9))
	require.EqualError(t, err, "header not found")
}

// TestModuleWallet_DefaultsComeFromTheModule is ADR5: the legacy wallet
// methods take their tip and gas estimate from the module.
func TestModuleWallet_DefaultsComeFromTheModule(t *testing.T) {
	r := tests.NewReplayChain(t)
	module := rpc.New(r.Core, rpc.DefaultConfig(), prometheus.NewRegistry())
	require.NoError(t, module.Start())
	defer func() { _ = module.Stop() }()
	wallet := moduleWallet{module}
	ctx := context.Background()

	tip, err := wallet.SuggestGasTipCap(ctx)
	require.NoError(t, err)
	require.NotNil(t, tip)
	gas, err := wallet.EstimateGas(ctx, legacy.TransactionArgs{From: &r.Wallet.Address, To: &r.Recipient}, gethrpc.BlockNumberOrHashWithNumber(gethrpc.PendingBlockNumber))
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(21000), gas)
	gas, err = wallet.EstimateGas(ctx, legacy.TransactionArgs{From: &r.Wallet.Address, To: &r.Recipient}, gethrpc.BlockNumberOrHashWithHash(r.Block.Hash, false))
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(21000), gas)
	_, err = wallet.EstimateGas(ctx, legacy.TransactionArgs{From: &r.Wallet.Address, To: &r.Recipient}, gethrpc.BlockNumberOrHashWithHash(common.Hash{1}, false))
	require.Error(t, err)
}
