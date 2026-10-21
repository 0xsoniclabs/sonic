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
	"encoding/json"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/0xsoniclabs/sonic/rpc/transport"
)

// startModule runs a module over the core with HTTP and WebSocket on one
// ephemeral port, and a unix socket.
func startModule(t *testing.T, fc *tests.FakeCore, mutate ...func(*rpc.Config)) (*rpc.Module, string, string) {
	t.Helper()
	sock := filepath.Join(t.TempDir(), "sonic.ipc")
	cfg := rpc.Config{
		Limits:        dispatch.DefaultConfig(),
		ClientVersion: "test/v0",
		HTTP: &transport.HTTPConfig{
			Host: "127.0.0.1",
			WS: &transport.WSConfig{
				StreamConfig: transport.StreamConfig{MaxSubscriptions: 4},
				Origins:      []string{"http://allowed.example"},
			},
			VHosts: []string{"localhost", "allowed.example"},
		},
		IPC: &rpc.IPCConfig{Path: sock},
	}
	for _, m := range mutate {
		m(&cfg)
	}
	module, err := rpc.New(cfg, fc, rpc.WithMetrics(dispatch.NewMetrics(nil)))
	require.NoError(t, err)
	require.NoError(t, module.Start())
	t.Cleanup(func() { require.NoError(t, module.Stop()) })
	return module, module.HTTPAddr(), sock
}

func dialWS(t *testing.T, addr string) *gethrpc.Client {
	t.Helper()
	client, err := gethrpc.DialWebsocket(context.Background(), "ws://"+addr, "")
	require.NoError(t, err)
	t.Cleanup(client.Close)
	return client
}

func TestModule_ServesHTTPWebSocketAndIPC(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(3))
	_, addr, sock := startModule(t, fc)

	httpClient, err := gethrpc.DialHTTP("http://" + addr)
	require.NoError(t, err)
	defer httpClient.Close()
	ipcClient, err := gethrpc.DialIPC(context.Background(), sock)
	require.NoError(t, err)
	defer ipcClient.Close()
	wsClient := dialWS(t, addr)

	for name, client := range map[string]*gethrpc.Client{"http": httpClient, "ws": wsClient, "ipc": ipcClient} {
		var n hexutil.Uint64
		require.NoError(t, client.Call(&n, "eth_blockNumber"), name)
		require.EqualValues(t, 3, n, name)

		var id hexutil.Big
		require.NoError(t, client.Call(&id, "ftm_chainId"), name)
		require.Equal(t, int64(opera.FakeNetworkID), id.ToInt().Int64(), name)

		var version string
		require.NoError(t, client.Call(&version, "web3_clientVersion"), name)
		require.Equal(t, "test/v0", version, name)
		require.NoError(t, client.Call(&version, "net_version"), name)
		require.Equal(t, "4003", version, name)
	}

	// The unix socket is for its owner only.
	info, err := os.Stat(sock)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestModule_HealthEndpointReportsHeadAndLag(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(2))
	_, addr, _ := startModule(t, fc)
	fc.SetProgress(core.Progress{CurrentBlock: 5})

	resp, err := http.Get("http://" + addr + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	var h transport.Health
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&h))
	require.EqualValues(t, 2, h.Head)
	require.EqualValues(t, 5, h.Committed)
	require.EqualValues(t, 3, h.Lag)
}

func TestModule_NewHeadsSubscription(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	_, addr, sock := startModule(t, fc)

	for name, client := range map[string]*gethrpc.Client{
		"ws": dialWS(t, addr),
		"ipc": func() *gethrpc.Client {
			c, err := gethrpc.DialIPC(context.Background(), sock)
			require.NoError(t, err)
			t.Cleanup(c.Close)
			return c
		}(),
	} {
		heads := make(chan map[string]any, 8)
		sub, err := client.EthSubscribe(context.Background(), heads, "newHeads")
		require.NoError(t, err, name)

		fc.AddBlock()
		select {
		case head := <-heads:
			require.Equal(t, hexutil.EncodeUint64(fc.Latest()), head["number"], name)
		case <-time.After(2 * time.Second):
			t.Fatalf("%s: no notification", name)
		}
		sub.Unsubscribe()
	}
}

func TestModule_SubscriptionIsInstalledBeforeTheIdIsReturned(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	_, addr, _ := startModule(t, fc)
	client := dialWS(t, addr)

	// Blocks produced right after the response must all arrive.
	for range 20 {
		heads := make(chan map[string]any, 64)
		sub, err := client.EthSubscribe(context.Background(), heads, "newHeads")
		require.NoError(t, err)
		first := fc.AddBlock().NumberU64()
		select {
		case head := <-heads:
			require.Equal(t, hexutil.EncodeUint64(first), head["number"])
		case <-time.After(2 * time.Second):
			t.Fatal("the block after the subscribe response was lost")
		}
		sub.Unsubscribe()
	}
}

func TestModule_LogsAndPendingTransactionSubscriptions(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	_, addr, _ := startModule(t, fc)
	client := dialWS(t, addr)

	address := common.HexToAddress("0xaa")
	other := common.HexToAddress("0xbb")
	logsCh := make(chan types.Log, 8)
	sub, err := client.EthSubscribe(context.Background(), logsCh, "logs", map[string]any{"address": address})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	hashes := make(chan common.Hash, 8)
	subTx, err := client.EthSubscribe(context.Background(), hashes, "newPendingTransactions")
	require.NoError(t, err)
	defer subTx.Unsubscribe()
	fullTxs := make(chan map[string]any, 8)
	subFull, err := client.EthSubscribe(context.Background(), fullTxs, "newPendingTransactions", true)
	require.NoError(t, err)
	defer subFull.Unsubscribe()

	fc.Publish(core.HeadEvent{Block: fc.BlockAt(1), Logs: []*types.Log{
		{Address: other, BlockNumber: 1, Index: 0, Topics: []common.Hash{}},
		{Address: address, BlockNumber: 1, Index: 1, Topics: []common.Hash{}},
	}})
	select {
	case l := <-logsCh:
		require.Equal(t, address, l.Address)
		require.EqualValues(t, 1, l.Index)
	case <-time.After(2 * time.Second):
		t.Fatal("no log")
	}

	tx := signedTx(t, 0)
	fc.AddPoolTransaction(tx)
	select {
	case h := <-hashes:
		require.Equal(t, tx.Hash(), h)
	case <-time.After(2 * time.Second):
		t.Fatal("no pending transaction hash")
	}
	select {
	case full := <-fullTxs:
		require.Equal(t, tx.Hash().Hex(), full["hash"])
		require.Nil(t, full["blockHash"])
	case <-time.After(2 * time.Second):
		t.Fatal("no pending transaction")
	}
}

func signedTx(t *testing.T, nonce uint64) *types.Transaction {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	to := common.Address{0x42}
	signer := types.LatestSignerForChainID(new(big.Int).SetUint64(opera.FakeNetworkID))
	tx, err := types.SignNewTx(key, signer, &types.DynamicFeeTx{
		ChainID: new(big.Int).SetUint64(opera.FakeNetworkID), Nonce: nonce,
		GasFeeCap: big.NewInt(2000), Gas: 21000, To: &to, Value: big.NewInt(1),
	})
	require.NoError(t, err)
	return tx
}

func TestModule_SubscriptionErrors(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	_, addr, _ := startModule(t, fc)
	client := dialWS(t, addr)

	var id string
	err := client.Call(&id, "eth_subscribe", "nope")
	require.ErrorContains(t, err, `no "nope" subscription in eth namespace`)
	require.Equal(t, -32601, err.(gethrpc.Error).ErrorCode())

	err = client.Call(&id, "eth_subscribe", "logs", map[string]any{"fromBlock": "0x5", "toBlock": "0x1"})
	require.ErrorContains(t, err, "invalid from and to block combination")

	err = client.Call(&id, "eth_subscribe")
	require.ErrorContains(t, err, "non-array args")
	err = client.Call(&id, "eth_subscribe", 5)
	require.ErrorContains(t, err, "expected subscription name as first argument")

	var ok bool
	err = client.Call(&ok, "eth_unsubscribe", "0x1234")
	require.ErrorContains(t, err, "subscription not found")
}

func TestModule_SubscriptionsAreCappedPerConnection(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	_, addr, _ := startModule(t, fc) // the cap is 4
	client := dialWS(t, addr)

	var subs []*gethrpc.ClientSubscription
	for range 4 {
		sub, err := client.EthSubscribe(context.Background(), make(chan map[string]any, 1), "newHeads")
		require.NoError(t, err)
		subs = append(subs, sub)
	}
	_, err := client.EthSubscribe(context.Background(), make(chan map[string]any, 1), "newHeads")
	require.ErrorContains(t, err, "too many subscriptions")

	// A freed slot can be used again.
	subs[0].Unsubscribe()
	require.Eventually(t, func() bool {
		sub, err := client.EthSubscribe(context.Background(), make(chan map[string]any, 1), "newHeads")
		if err == nil {
			sub.Unsubscribe()
		}
		return err == nil
	}, 2*time.Second, 20*time.Millisecond)

	// A second connection has its own budget.
	other := dialWS(t, addr)
	_, err = other.EthSubscribe(context.Background(), make(chan map[string]any, 1), "newHeads")
	require.NoError(t, err)
}

func TestModule_SubscriptionIdsBelongToTheConnection(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	_, addr, _ := startModule(t, fc)
	a, b := dialWS(t, addr), dialWS(t, addr)

	var id string
	sub, err := a.EthSubscribe(context.Background(), make(chan map[string]any, 1), "newHeads")
	require.NoError(t, err)
	defer sub.Unsubscribe()
	var raw json.RawMessage
	require.NoError(t, a.Call(&raw, "eth_subscribe", "newHeads"))
	require.NoError(t, json.Unmarshal(raw, &id))

	var ok bool
	err = b.Call(&ok, "eth_unsubscribe", id)
	require.ErrorContains(t, err, "subscription not found")
	require.NoError(t, a.Call(&ok, "eth_unsubscribe", id))
	require.True(t, ok)
}

func TestModule_SubscriptionsAreNotAvailableOverHTTP(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	_, addr, _ := startModule(t, fc)
	client, err := gethrpc.DialHTTP("http://" + addr)
	require.NoError(t, err)
	defer client.Close()
	var id string
	err = client.Call(&id, "eth_subscribe", "newHeads")
	require.ErrorContains(t, err, "notifications not supported")
}

func TestModule_PolledFilters(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	_, addr, _ := startModule(t, fc)
	client, err := gethrpc.DialHTTP("http://" + addr)
	require.NoError(t, err)
	defer client.Close()

	var blockFilter, txFilter, fullTxFilter, logFilter string
	require.NoError(t, client.Call(&blockFilter, "eth_newBlockFilter"))
	require.NoError(t, client.Call(&txFilter, "eth_newPendingTransactionFilter"))
	require.NoError(t, client.Call(&fullTxFilter, "eth_newPendingTransactionFilter", true))
	require.NoError(t, client.Call(&logFilter, "eth_newFilter", map[string]any{"address": common.HexToAddress("0xaa")}))
	require.True(t, strings.HasPrefix(blockFilter, "0x"))
	require.NotEqual(t, blockFilter, txFilter)

	// Nothing happened yet.
	var hashes []common.Hash
	require.NoError(t, client.Call(&hashes, "eth_getFilterChanges", blockFilter))
	require.Empty(t, hashes)
	var raw json.RawMessage
	require.NoError(t, client.Call(&raw, "eth_getFilterChanges", txFilter))
	require.Equal(t, "null", string(raw)) // the answer of a pending transaction filter with nothing new

	block := fc.AddBlock()
	tx := signedTx(t, 0)
	fc.AddPoolTransaction(tx)
	fc.Publish(core.HeadEvent{Block: block, Logs: []*types.Log{
		{Address: common.HexToAddress("0xaa"), BlockNumber: 2, Topics: []common.Hash{}},
	}})

	require.Eventually(t, func() bool {
		var hs []common.Hash
		if err := client.Call(&hs, "eth_getFilterChanges", blockFilter); err != nil {
			return false
		}
		hashes = append(hashes, hs...)
		return len(hashes) >= 1
	}, 2*time.Second, 10*time.Millisecond)
	require.Equal(t, block.Hash, hashes[0])

	var txHashes []common.Hash
	require.Eventually(t, func() bool {
		return client.Call(&txHashes, "eth_getFilterChanges", txFilter) == nil && len(txHashes) == 1
	}, 2*time.Second, 10*time.Millisecond)
	require.Equal(t, tx.Hash(), txHashes[0])

	var full []map[string]any
	require.Eventually(t, func() bool {
		return client.Call(&full, "eth_getFilterChanges", fullTxFilter) == nil && len(full) == 1
	}, 2*time.Second, 10*time.Millisecond)
	require.Nil(t, full[0]["blockNumber"])

	var logs []types.Log
	require.Eventually(t, func() bool {
		return client.Call(&logs, "eth_getFilterChanges", logFilter) == nil && len(logs) == 1
	}, 2*time.Second, 10*time.Millisecond)

	// The changes are delivered once.
	require.NoError(t, client.Call(&hashes, "eth_getFilterChanges", blockFilter))
	require.Empty(t, hashes)

	var found bool
	require.NoError(t, client.Call(&found, "eth_uninstallFilter", blockFilter))
	require.True(t, found)
	require.NoError(t, client.Call(&found, "eth_uninstallFilter", blockFilter))
	require.False(t, found)
	err = client.Call(&hashes, "eth_getFilterChanges", blockFilter)
	require.ErrorContains(t, err, "filter not found")
	// Only log filters have logs.
	err = client.Call(&logs, "eth_getFilterLogs", txFilter)
	require.ErrorContains(t, err, "filter not found")
}

func TestModule_FiltersExpireWithoutPolls(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	_, addr, _ := startModule(t, fc, func(c *rpc.Config) { c.Limits.FilterTimeout = 300 * time.Millisecond })
	client, err := gethrpc.DialHTTP("http://" + addr)
	require.NoError(t, err)
	defer client.Close()

	var idle, polled string
	require.NoError(t, client.Call(&idle, "eth_newBlockFilter"))
	require.NoError(t, client.Call(&polled, "eth_newBlockFilter"))
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		var hs []common.Hash
		require.NoError(t, client.Call(&hs, "eth_getFilterChanges", polled))
		time.Sleep(50 * time.Millisecond)
	}
	var hs []common.Hash
	require.ErrorContains(t, client.Call(&hs, "eth_getFilterChanges", idle), "filter not found")
	require.NoError(t, client.Call(&hs, "eth_getFilterChanges", polled))
}

func TestModule_StopLeavesNoGoroutinesBehindUnderLoad(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(2))
	sock := filepath.Join(t.TempDir(), "s.ipc")
	cfg := rpc.Config{
		Limits: dispatch.DefaultConfig(),
		HTTP:   &transport.HTTPConfig{Host: "127.0.0.1", WS: &transport.WSConfig{}},
		IPC:    &rpc.IPCConfig{Path: sock},
	}
	module, err := rpc.New(cfg, fc, rpc.WithMetrics(dispatch.NewMetrics(nil)))
	require.NoError(t, err)
	require.NoError(t, module.Start())
	addr := module.HTTPAddr()

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
			}
			// Every call is bounded: go-ethereum's client can wait forever for the
			// answer of a request that raced with the close of its connection, and
			// this loop runs while the module stops. With no socket open on the
			// server side at that point, the module is not what is waited for.
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			client, err := gethrpc.DialWebsocket(ctx, "ws://"+addr, "")
			if err != nil {
				cancel()
				continue
			}
			heads := make(chan map[string]any, 4)
			_, _ = client.EthSubscribe(ctx, heads, "newHeads")
			var n hexutil.Uint64
			_ = client.CallContext(ctx, &n, "eth_blockNumber")
			cancel()
			client.Close()
		}
	}()
	time.Sleep(200 * time.Millisecond)
	require.NoError(t, module.Stop())
	close(stop)
	<-done

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && moduleGoroutines() != "" {
		time.Sleep(10 * time.Millisecond)
	}
	require.Empty(t, moduleGoroutines(), "goroutine of the module is alive after Stop")
}

func moduleGoroutines() string {
	buf := make([]byte, 1<<22)
	buf = buf[:runtime.Stack(buf, true)]
	for _, g := range strings.Split(string(buf), "\n\n") {
		if strings.Contains(g, "testing.tRunner") || strings.Contains(g, "runtime.Stack") {
			continue
		}
		if strings.Contains(g, "sonic/rpc/dispatch.") || strings.Contains(g, "sonic/rpc/transport.") ||
			strings.Contains(g, "sonic/rpc/filters.") || strings.Contains(g, "sonic/rpc.(*Module)") {
			return g
		}
	}
	return ""
}

// D11: transport security.
func TestModule_RejectsADisallowedHost(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	_, addr, _ := startModule(t, fc)
	req, err := http.NewRequest(http.MethodPost, "http://"+addr, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	req.Host = "evil.example"
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusForbidden, resp.StatusCode)

	req, _ = http.NewRequest(http.MethodPost, "http://"+addr, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Host = "allowed.example"
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestModule_RejectsADisallowedWebSocketOrigin(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	_, addr, _ := startModule(t, fc)

	dialer := websocket.Dialer{HandshakeTimeout: time.Second}
	header := http.Header{"Origin": []string{"http://evil.example"}}
	_, resp, err := dialer.Dial("ws://"+addr, header)
	require.Error(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)

	header = http.Header{"Origin": []string{"http://allowed.example"}}
	conn, _, err := dialer.Dial("ws://"+addr, header)
	require.NoError(t, err)
	conn.Close()

	// Without Origin header (not a browser) the connection is fine.
	conn, _, err = dialer.Dial("ws://"+addr, nil)
	require.NoError(t, err)
	conn.Close()
}

func TestModule_RejectsAnOversizeBody(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	_, addr, _ := startModule(t, fc)
	body := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":["` + strings.Repeat("a", 6*1024*1024) + `"]}`)
	resp, err := http.Post("http://"+addr, "application/json", body)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
}

func TestModule_AStalledClientDoesNotHoldTheServer(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	_, addr, _ := startModule(t, fc, func(c *rpc.Config) {
		c.HTTP.Timeouts = transport.Timeouts{
			Read: time.Second, ReadHeader: time.Second, Write: time.Second, Idle: time.Second,
		}
	})
	// A client that opens a connection and sends nothing is dropped.
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, err = conn.Read(make([]byte, 1))
	require.Error(t, err) // closed by the server, not a read timeout of ours
	require.NotContains(t, err.Error(), "i/o timeout")

	// Other clients are served meanwhile.
	client, err := gethrpc.DialHTTP("http://" + addr)
	require.NoError(t, err)
	defer client.Close()
	var n hexutil.Uint64
	require.NoError(t, client.Call(&n, "eth_blockNumber"))
}

func TestModule_AClientThatDoesNotReadIsDisconnected(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	_, addr, _ := startModule(t, fc, func(c *rpc.Config) {
		c.HTTP.WS.QueueSize = 4
		c.HTTP.WS.MaxSubscriptions = 4
	})
	conn, _, err := websocket.DefaultDialer.Dial("ws://"+addr, nil)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, conn.WriteMessage(websocket.TextMessage,
		[]byte(`{"jsonrpc":"2.0","id":1,"method":"eth_subscribe","params":["newHeads"]}`)))
	_, _, err = conn.ReadMessage() // the response; from here on the client reads nothing
	require.NoError(t, err)

	// A burst of blocks overflows the small queue and the connection is cut.
	for range 100 {
		fc.AddBlock()
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
	t.Fatal("the connection of a client that does not read was not closed")
}

// D8: namespaces are enabled per transport.
func TestModule_NamespacesAreEnabledPerTransport(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	_, addr, sock := startModule(t, fc, func(c *rpc.Config) {
		c.HTTP.Modules = []string{"eth", "web3"}
		c.HTTP.WS.Modules = []string{"net"}
	})
	httpClient, err := gethrpc.DialHTTP("http://" + addr)
	require.NoError(t, err)
	defer httpClient.Close()
	wsClient := dialWS(t, addr)
	ipcClient, err := gethrpc.DialIPC(context.Background(), sock)
	require.NoError(t, err)
	defer ipcClient.Close()

	var out json.RawMessage
	require.NoError(t, httpClient.Call(&out, "eth_blockNumber"))
	require.NoError(t, httpClient.Call(&out, "web3_clientVersion"))
	err = httpClient.Call(&out, "net_version")
	require.ErrorContains(t, err, "does not exist/is not available")
	err = httpClient.Call(&out, "ftm_blockNumber") // ftm has to be named to be enabled
	require.ErrorContains(t, err, "does not exist/is not available")

	require.NoError(t, wsClient.Call(&out, "net_version"))
	err = wsClient.Call(&out, "eth_blockNumber")
	require.ErrorContains(t, err, "does not exist/is not available")
	// A namespace that is not enabled has no subscriptions.
	err = wsClient.Call(&out, "eth_subscribe", "newHeads")
	require.ErrorContains(t, err, `no "newHeads" subscription in eth namespace`)

	// IPC serves every namespace.
	require.NoError(t, ipcClient.Call(&out, "eth_blockNumber"))
	require.NoError(t, ipcClient.Call(&out, "net_version"))
	require.NoError(t, ipcClient.Call(&out, "txpool_status"))

	// rpc_modules lists what the transport serves.
	var modules map[string]string
	require.NoError(t, httpClient.Call(&modules, "rpc_modules"))
	require.Equal(t, map[string]string{"eth": "1.0", "web3": "1.0", "rpc": "1.0"}, modules)
	modules = nil
	require.NoError(t, wsClient.Call(&modules, "rpc_modules"))
	require.Equal(t, map[string]string{"net": "1.0", "rpc": "1.0"}, modules)
	modules = nil
	require.NoError(t, ipcClient.Call(&modules, "rpc_modules"))
	require.Contains(t, modules, "debug")
	require.Contains(t, modules, "trace")
	require.Contains(t, modules, "txpool")
	require.Contains(t, modules, "sonic")
}

// S1: the filter API's Stop is not callable, on any transport.
func TestModule_EthStopIsNotCallable(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	_, addr, sock := startModule(t, fc)
	httpClient, err := gethrpc.DialHTTP("http://" + addr)
	require.NoError(t, err)
	defer httpClient.Close()
	ipcClient, err := gethrpc.DialIPC(context.Background(), sock)
	require.NoError(t, err)
	defer ipcClient.Close()

	for name, client := range map[string]*gethrpc.Client{"http": httpClient, "ws": dialWS(t, addr), "ipc": ipcClient} {
		for _, method := range []string{"eth_stop", "ftm_stop"} {
			var out json.RawMessage
			err := client.Call(&out, method)
			require.ErrorContains(t, err, "does not exist/is not available", "%s %s", name, method)
			require.Equal(t, -32601, err.(gethrpc.Error).ErrorCode())
		}
		// The filter machinery still works.
		var id string
		require.NoError(t, client.Call(&id, "eth_newBlockFilter"), name)
	}
}

// I13: reads that need no transactions do not read blocks.
func TestModule_HeaderOnlyReadsDoNotReadBlocks(t *testing.T) {
	fc, cs := tests.NewCarmenCore(t, tests.WithBlocks(3))
	cs.SetBalance(common.HexToAddress("0x1"), big.NewInt(1e18))
	module, err := rpc.New(rpc.Config{Limits: dispatch.DefaultConfig()}, fc, rpc.WithMetrics(dispatch.NewMetrics(nil)))
	require.NoError(t, err)
	require.NoError(t, module.Start())
	defer func() { require.NoError(t, module.Stop()) }()
	d := module.Dispatcher()
	blockReads := fc.Calls("Block")

	zero := common.Address{}
	for _, call := range []struct {
		method string
		params string
	}{
		{"eth_getBalance", `["` + zero.Hex() + `","0x2"]`},
		{"eth_getCode", `["` + zero.Hex() + `","latest"]`},
		{"eth_getStorageAt", `["` + zero.Hex() + `","0x0","latest"]`},
		{"eth_getTransactionCount", `["` + zero.Hex() + `","latest"]`},
		{"eth_call", `[{"to":"` + zero.Hex() + `"},"latest"]`},
		{"eth_estimateGas", `[{"to":"` + zero.Hex() + `"},"latest"]`},
		{"eth_createAccessList", `[{"to":"` + zero.Hex() + `","from":"0x0000000000000000000000000000000000000001","gas":"0x30000"}]`},
		{"eth_getLogs", `[{"fromBlock":"0x1","toBlock":"0x3","address":"` + zero.Hex() + `"}]`},
		{"eth_blockNumber", `[]`},
		{"eth_gasPrice", `[]`},
	} {
		_, rerr := d.Serve(context.Background(), call.method, json.RawMessage(call.params))
		_ = rerr
	}
	require.Equal(t, blockReads, fc.Calls("Block"), "a header-only read read a block")
}

// P4: a Core call that blocks does not stall other requests.
func TestModule_ABlockedCoreCallDoesNotStallOtherRequests(t *testing.T) {
	fc, _ := tests.NewCarmenCore(t, tests.WithBlocks(2))
	module, _, _ := func() (*rpc.Module, string, string) {
		cfg := rpc.Config{Limits: dispatch.DefaultConfig()}
		m, err := rpc.New(cfg, fc, rpc.WithMetrics(dispatch.NewMetrics(nil)))
		require.NoError(t, err)
		require.NoError(t, m.Start())
		t.Cleanup(func() { require.NoError(t, m.Stop()) })
		return m, "", ""
	}()
	d := module.Dispatcher()

	release := fc.BlockCalls("Receipts")
	defer release()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = d.Serve(context.Background(), "eth_getBlockByNumber", json.RawMessage(`["latest",false]`))
	}()
	fc.WaitBlocked("Receipts")

	// Other requests, including filters and head readers, are served meanwhile.
	for _, call := range []struct{ method, params string }{
		{"eth_blockNumber", `[]`}, {"eth_chainId", `[]`}, {"eth_newBlockFilter", `[]`},
		{"eth_getBalance", `["0x0000000000000000000000000000000000000001","latest"]`},
	} {
		start := time.Now()
		_, rerr := d.Serve(context.Background(), call.method, json.RawMessage(call.params))
		require.Nil(t, rerr, call.method)
		require.Less(t, time.Since(start), time.Second, call.method)
	}
	release()
	<-done
}

// legacyStub answers every call it is handed with its method name.
type legacyStub struct{ calls []string }

func (l *legacyStub) Forward(_ context.Context, method string, _ json.RawMessage) (json.RawMessage, *dispatch.Error) {
	l.calls = append(l.calls, method)
	return json.RawMessage(`"legacy ` + method + `"`), nil
}

// X3: the wallet methods and the consensus internals are served by the legacy
// handler on the IPC socket only; the p2p facts on every transport.
func TestModule_LegacyMethodsAreServedOnIPCOnly(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	legacy := &legacyStub{}
	module, err := rpc.New(rpc.Config{
		Limits: dispatch.DefaultConfig(), LegacyNamespaces: []string{"admin", "dag", "abft", "personal", "test"},
		HTTP: &transport.HTTPConfig{Host: "127.0.0.1"}, IPC: &rpc.IPCConfig{Path: filepath.Join(t.TempDir(), "l.ipc")},
	}, fc, rpc.WithMetrics(dispatch.NewMetrics(nil)), rpc.WithLegacy(legacy))
	require.NoError(t, err)
	require.NoError(t, module.Start())
	defer func() { require.NoError(t, module.Stop()) }()

	httpClient, err := gethrpc.DialHTTP("http://" + module.HTTPAddr())
	require.NoError(t, err)
	defer httpClient.Close()
	ipcClient, err := gethrpc.DialIPC(context.Background(), module.IPCPath())
	require.NoError(t, err)
	defer ipcClient.Close()

	for _, method := range []string{
		"admin_peers", "dag_getHeads", "abft_getValidators", "personal_listAccounts", "test_addTransactions",
		"eth_accounts", "eth_sign", "ftm_accounts", "eth_getEpochBlock", "debug_setHead",
	} {
		var out string
		require.NoError(t, ipcClient.Call(&out, method), method)
		require.Equal(t, "legacy "+method, out)

		err := httpClient.Call(&out, method)
		require.ErrorContains(t, err, "does not exist/is not available", method)
	}
	for _, method := range []string{"net_listening", "net_peerCount"} {
		var out string
		require.NoError(t, httpClient.Call(&out, method), method)
		require.Equal(t, "legacy "+method, out)
	}

	// What the module serves is never forwarded, and unknown methods are
	// unknown on both.
	var out string
	require.Error(t, ipcClient.Call(&out, "eth_nope"))
	require.Error(t, httpClient.Call(&out, "zzz_nope"))
	for _, c := range legacy.calls {
		require.NotEqual(t, "eth_nope", c)
		require.NotEqual(t, "zzz_nope", c)
		require.NotEqual(t, "eth_blockNumber", c)
	}
}

func TestModule_LifecycleAndFailures(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))

	// A bad path prefix is refused at construction.
	_, err := rpc.New(rpc.Config{Limits: dispatch.DefaultConfig(), HTTP: &transport.HTTPConfig{Prefix: "rpc"}}, fc,
		rpc.WithMetrics(dispatch.NewMetrics(nil)))
	require.Error(t, err)
	_, err = rpc.New(rpc.Config{Limits: dispatch.DefaultConfig(), WS: &transport.HTTPConfig{Prefix: "rpc"}}, fc,
		rpc.WithMetrics(dispatch.NewMetrics(nil)))
	require.Error(t, err)

	// A module cannot be started twice, and stops once.
	m, err := rpc.New(rpc.Config{Limits: dispatch.DefaultConfig()}, fc, rpc.WithMetrics(dispatch.NewMetrics(nil)))
	require.NoError(t, err)
	require.Empty(t, m.HTTPAddr())
	require.Empty(t, m.WSAddr())
	require.Empty(t, m.IPCPath())
	require.NoError(t, m.Start())
	require.Error(t, m.Start())
	require.NoError(t, m.Stop())
	require.NoError(t, m.Stop())
	require.NotNil(t, m.Chain())
	require.NotNil(t, m.Front())

	// The head must be readable at start.
	broken := tests.NewFakeCore(t)
	broken.Inject("Head", tests.Fail)
	m, err = rpc.New(rpc.Config{Limits: dispatch.DefaultConfig()}, broken, rpc.WithMetrics(dispatch.NewMetrics(nil)))
	require.NoError(t, err)
	require.ErrorContains(t, m.Start(), "failed to read the head")
	require.NoError(t, m.Stop())

	// An endpoint that is taken is reported, and nothing stays open.
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer taken.Close()
	host, port, _ := net.SplitHostPort(taken.Addr().String())
	portNumber := 0
	_, _ = host, port
	for _, c := range port {
		portNumber = portNumber*10 + int(c-'0')
	}
	m, err = rpc.New(rpc.Config{Limits: dispatch.DefaultConfig(),
		HTTP: &transport.HTTPConfig{Host: "127.0.0.1", Port: portNumber}}, fc, rpc.WithMetrics(dispatch.NewMetrics(nil)))
	require.NoError(t, err)
	require.ErrorContains(t, m.Start(), "failed to start the HTTP transport")
	require.NoError(t, m.Stop())

	m, err = rpc.New(rpc.Config{Limits: dispatch.DefaultConfig(),
		WS: &transport.HTTPConfig{Host: "127.0.0.1", Port: portNumber, WS: &transport.WSConfig{}}}, fc, rpc.WithMetrics(dispatch.NewMetrics(nil)))
	require.NoError(t, err)
	require.ErrorContains(t, m.Start(), "failed to start the WebSocket transport")
	require.NoError(t, m.Stop())

	// The IPC path must be creatable.
	file := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(file, nil, 0600))
	m, err = rpc.New(rpc.Config{Limits: dispatch.DefaultConfig(), IPC: &rpc.IPCConfig{Path: filepath.Join(file, "x.ipc")}}, fc,
		rpc.WithMetrics(dispatch.NewMetrics(nil)))
	require.NoError(t, err)
	require.ErrorContains(t, m.Start(), "failed to start the IPC transport")
	require.NoError(t, m.Stop())
}

func TestModule_UsesTheDefaultMetricsRegistry(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	a, err := rpc.New(rpc.Config{Limits: dispatch.DefaultConfig()}, fc)
	require.NoError(t, err)
	b, err := rpc.New(rpc.Config{Limits: dispatch.DefaultConfig()}, fc) // collectors are shared, not registered twice
	require.NoError(t, err)
	require.NoError(t, a.Start())
	require.NoError(t, b.Start())
	_, rerr := a.Dispatcher().Serve(context.Background(), "eth_blockNumber", nil)
	require.Nil(t, rerr)
	require.NoError(t, a.Stop())
	require.NoError(t, b.Stop())

	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	names := map[string]bool{}
	for _, f := range families {
		names[f.GetName()] = true
	}
	for _, name := range []string{
		"rpc_requests_total", "rpc_duration_seconds", "rpc_response_bytes", "rpc_inflight", "rpc_head_served",
	} {
		require.True(t, names[name], name)
	}
}
