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

package filters_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func (chain logChain) install(t *testing.T, method string, parameters ...any) string {
	t.Helper()
	return tests.Decode[string](t, chain.harness.Result(method, parameters...))
}

func (chain logChain) changes(t *testing.T, identifier string) []any {
	t.Helper()
	return tests.Decode[[]any](t, chain.harness.Result("eth_getFilterChanges", identifier))
}

func TestBlockFilter_ReportsTheBlocksSinceTheLastPoll(t *testing.T) {
	chain := newLogChain(t)
	identifier := chain.install(t, "eth_newBlockFilter")
	require.Equal(t, []any{}, chain.changes(t, identifier), "nothing yet")
	chain.core.AppendAndPublish(tests.BlockSpec{})
	chain.core.AppendAndPublish(tests.BlockSpec{})
	tests.Eventually(t, func() bool {
		return len(tests.Decode[[]any](t, chain.harness.Result("eth_getFilterChanges", identifier))) > 0
	}, "the filter to see the blocks")
	// The poll above took one or both blocks; publish one more and check
	// that it comes last.
	third := chain.core.AppendAndPublish(tests.BlockSpec{})
	tests.Eventually(t, func() bool {
		changes := chain.changes(t, identifier)
		return len(changes) > 0 && changes[len(changes)-1] == third.Hash.Hex()
	}, "the third block")
	require.Equal(t, []any{}, chain.changes(t, identifier), "a poll empties the filter")
}

// TestPendingTransactionFilter_ReportsHashesOrTransactions holds F7.
func TestPendingTransactionFilter_ReportsHashesOrTransactions(t *testing.T) {
	chain := newLogChain(t)
	hashes := chain.install(t, "eth_newPendingTransactionFilter")
	full := chain.install(t, "eth_newPendingTransactionFilter", true)
	require.JSONEq(t, `null`, string(chain.harness.Result("eth_getFilterChanges", hashes)), "no transactions are null, as in the reference")

	signer := tests.NewSigner(t)
	recipient := common.HexToAddress("0xb0")
	pending := signer.Sign(t, &types.LegacyTx{To: &recipient, Gas: 21000, GasPrice: big.NewInt(7)})
	chain.core.SendPoolTransactions(pending)
	tests.Eventually(t, func() bool {
		changes := tests.Decode[[]any](t, chain.harness.Result("eth_getFilterChanges", hashes))
		return len(changes) == 1 && changes[0] == pending.Hash().Hex()
	}, "the hash of the pending transaction")
	answered := tests.Decode[[]map[string]any](t, chain.harness.Result("eth_getFilterChanges", full))
	require.Len(t, answered, 1)
	require.Equal(t, pending.Hash().Hex(), answered[0]["hash"])
	require.Nil(t, answered[0]["blockHash"])
	require.Equal(t, "0x7", answered[0]["gasPrice"], "priced at its own gas price")
}

// TestLogFilter_ReportsTheMatchingLogsOfNewBlocks holds P8: the logs the node
// handed out are not changed.
func TestLogFilter_ReportsTheMatchingLogsOfNewBlocks(t *testing.T) {
	chain := newLogChain(t)
	identifier := chain.install(t, "eth_newFilter", map[string]any{"address": first})
	pendingOnly := chain.install(t, "eth_newFilter", map[string]any{"fromBlock": "pending", "toBlock": "pending"})
	shared := &types.Log{Address: first}
	other := &types.Log{Address: second, Topics: []common.Hash{red}}
	chain.core.AppendAndPublish(tests.BlockSpec{Transactions: types.Transactions{chain.sent[0]}, Receipts: types.Receipts{{Status: 1, Logs: []*types.Log{shared, other}}}})
	var logs []any
	tests.Eventually(t, func() bool {
		logs = append(logs, chain.changes(t, identifier)...)
		return len(logs) > 0
	}, "the log of the new block")
	require.Len(t, logs, 1)
	require.Equal(t, []any{}, logs[0].(map[string]any)["topics"])
	require.Nil(t, shared.Topics, "the node's log keeps its null topics")
	require.Equal(t, []any{}, chain.changes(t, pendingOnly), "a filter of pending logs reports nothing, as in the reference")

	history := tests.Decode[[]any](t, chain.harness.Result("eth_getFilterLogs", identifier))
	require.Len(t, history, 1, "the history of the filter's criteria is the latest block")
}

func TestFilters_RefuseAndForgetAsTheReference(t *testing.T) {
	chain := newLogChain(t, limit(func(config *exec.Config) { config.LogQueryParameterLimit = 1 }))
	require.Equal(t, "invalid from and to block combination: from > to", chain.harness.Failure("eth_newFilter", map[string]any{"fromBlock": "0x5", "toBlock": "0x2"}).Message)
	require.Equal(t, "invalid from and to block combination: from > to", chain.harness.Failure("eth_newFilter", map[string]any{"fromBlock": "earliest"}).Message)
	require.Equal(t, "too many query parameters, the limit is 1", chain.harness.Failure("eth_newFilter", map[string]any{"address": []any{first.Hex(), second.Hex()}}).Message)

	identifier := chain.install(t, "eth_newBlockFilter")
	require.Equal(t, "filter not found", chain.harness.Failure("eth_getFilterLogs", identifier).Message, "a block filter has no logs")
	require.JSONEq(t, `true`, string(chain.harness.Result("eth_uninstallFilter", identifier)))
	require.JSONEq(t, `false`, string(chain.harness.Result("eth_uninstallFilter", identifier)), "an uninstalled filter is forgotten")
	require.Equal(t, "filter not found", chain.harness.Failure("eth_getFilterChanges", identifier).Message)
	require.Equal(t, "filter not found", chain.harness.Failure("eth_getFilterChanges", "0x1").Message)
}

// TestFilters_ExpireWithoutAPoll holds D6.
func TestFilters_ExpireWithoutAPoll(t *testing.T) {
	chain := newLogChain(t, limit(func(config *exec.Config) { config.FilterTimeout = 100 * time.Millisecond }))
	polled := chain.install(t, "eth_newBlockFilter")
	forgotten := chain.install(t, "eth_newBlockFilter")
	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		chain.changes(t, polled)
		time.Sleep(20 * time.Millisecond)
	}
	require.Equal(t, "filter not found", chain.harness.Failure("eth_getFilterChanges", forgotten).Message)
	chain.changes(t, polled)
}

// TestFilters_OfADroppedFeedAreRemoved holds I18.
func TestFilters_OfADroppedFeedAreRemoved(t *testing.T) {
	chain := newLogChain(t)
	blocks := chain.install(t, "eth_newBlockFilter")
	logs := chain.install(t, "eth_newFilter", map[string]any{})
	subscriptions := chain.core.Calls("SubscribeHead")
	chain.core.EndFeeds()
	tests.Eventually(t, func() bool {
		_, failure := chain.harness.Call("eth_getFilterChanges", blocks)
		return failure != nil && chain.core.Calls("SubscribeHead") > subscriptions
	}, "the block filter to be removed and the head feed taken again")
	require.Equal(t, "filter not found", chain.harness.Failure("eth_getFilterChanges", logs).Message)
	for _, kind := range []string{"blockFilter", "logFilter"} {
		require.Equal(t, 1.0, tests.Labeled(t, chain.harness.Registry, "rpc_subscription_events_total", map[string]string{"kind": kind, "outcome": "lost"}), kind)
		require.Zero(t, tests.Labeled(t, chain.harness.Registry, "rpc_subscriptions", map[string]string{"kind": kind}), kind)
	}

	after := chain.install(t, "eth_newBlockFilter")
	tests.Eventually(t, func() bool {
		block := chain.core.AppendAndPublish(tests.BlockSpec{})
		changes := chain.changes(t, after)
		return len(changes) > 0 && changes[len(changes)-1] == block.Hash.Hex()
	}, "a new filter to see the blocks after the drop")
}

// TestFilters_AreCounted holds MON2 for the polled filters: the filters of
// each kind and the events their polls answer.
func TestFilters_AreCounted(t *testing.T) {
	chain := newLogChain(t)
	count := func(name string, labels map[string]string) float64 {
		return tests.Labeled(t, chain.harness.Registry, name, labels)
	}
	blocks := chain.install(t, "eth_newBlockFilter")
	chain.install(t, "eth_newBlockFilter")
	transactions := chain.install(t, "eth_newPendingTransactionFilter")
	require.Equal(t, 2.0, count("rpc_subscriptions", map[string]string{"kind": "blockFilter"}))
	require.Equal(t, 1.0, count("rpc_subscriptions", map[string]string{"kind": "pendingTransactionFilter"}))

	chain.core.AppendAndPublish(tests.BlockSpec{})
	answered := 0
	tests.Eventually(t, func() bool {
		answered += len(chain.changes(t, blocks))
		return answered > 0
	}, "the block")
	require.Equal(t, float64(answered), count("rpc_subscription_events_total", map[string]string{"kind": "blockFilter", "outcome": "delivered"}))

	require.True(t, tests.Decode[bool](t, chain.harness.Result("eth_uninstallFilter", transactions)))
	require.Zero(t, count("rpc_subscriptions", map[string]string{"kind": "pendingTransactionFilter"}))
}
