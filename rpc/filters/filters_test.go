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

package filters

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/format"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

var (
	emitter = common.HexToAddress("0xe1")
	other   = common.HexToAddress("0xe2")
	topicA  = common.HexToHash("0xa")
	topicB  = common.HexToHash("0xb")
)

// logChain is a fake chain whose blocks 1 to 3 each hold one transaction
// with a log of the emitter, and block 2 one of another address too.
type logChain struct {
	fc  *tests.FakeCore
	h   *tests.Harness
	s   *System
	w   *tests.Wallet
	txs []*types.Transaction
}

func newLogChain(t *testing.T, mutate ...tests.Mutate) *logChain {
	t.Helper()
	c := &logChain{fc: tests.NewFakeCore(), w: tests.NewWallet(t)}
	for i := range 3 {
		tx := c.w.Transfer(t, uint64(i), emitter, big.NewInt(1))
		logs := []*types.Log{{Address: emitter, Topics: []common.Hash{topicA}}}
		if i == 1 {
			logs = append(logs, &types.Log{Address: other, Topics: []common.Hash{topicB}})
		}
		c.fc.AddBlock(types.Transactions{tx}, logs)
		c.txs = append(c.txs, tx)
	}
	c.fc.Publish(3)
	c.h = tests.NewHarness(t, c.fc, func(h *tests.Harness) { c.s = Register(h.D, h.Env, c.fc, "eth") }, mutate...)
	// The filter loops follow the chain from here on.
	tests.WaitFor(t, func() bool { return c.h.Chain.Heads().Len() >= 2 })
	return c
}

// emit adds a block with one log of the emitter and publishes it.
func (c *logChain) emit(t *testing.T) *types.Transaction {
	t.Helper()
	tx := c.w.Transfer(t, uint64(len(c.txs)), emitter, big.NewInt(1))
	c.txs = append(c.txs, tx)
	block := c.fc.AddAndPublish(types.Transactions{tx}, []*types.Log{{Address: emitter, Topics: []common.Hash{topicA}}})
	c.h.WaitHead(block.NumberU64())
	return tx
}

func TestGetLogs_IndexedAndScanned(t *testing.T) {
	c := newLogChain(t)
	var logs []types.Log
	c.h.MustCall(&logs, "eth_getLogs", map[string]any{"fromBlock": "0x0", "toBlock": "latest", "address": emitter})
	require.Len(t, logs, 3)
	for i, l := range logs {
		require.Equal(t, uint64(i+1), l.BlockNumber)
		require.Equal(t, c.txs[i].Hash(), l.TxHash)
		require.NotZero(t, l.BlockTimestamp)
	}
	c.h.MustCall(&logs, "eth_getLogs", map[string]any{"fromBlock": "0x0", "toBlock": "latest"})
	require.Len(t, logs, 4, "a query without criteria scans the receipts")
	c.h.MustCall(&logs, "eth_getLogs", map[string]any{"fromBlock": "0x1", "topics": []any{topicB}})
	require.Len(t, logs, 1)
	require.Equal(t, other, logs[0].Address)
	c.h.MustCall(&logs, "eth_getLogs", map[string]any{"fromBlock": "0x1", "topics": []any{[]any{topicA, topicB}}})
	require.Len(t, logs, 4)
	require.Equal(t, "[]", string(c.h.MustCall(nil, "eth_getLogs", map[string]any{"fromBlock": "0x3", "toBlock": "0x1"})))
	require.Equal(t, "[]", string(c.h.MustCall(nil, "eth_getLogs", map[string]any{"address": common.Address{9}})))

	block2, err := c.fc.Block(2)
	require.NoError(t, err)
	c.h.MustCall(&logs, "eth_getLogs", map[string]any{"blockHash": block2.Hash})
	require.Len(t, logs, 2)
	require.Equal(t, uint(0), logs[1].TxIndex)
	e := c.h.Fails("eth_getLogs", map[string]any{"blockHash": common.Hash{1}})
	require.Equal(t, "block with hash 0x0100000000000000000000000000000000000000000000000000000000000000 not found", e.Message)
	e = c.h.Fails("eth_getLogs", map[string]any{"blockHash": block2.Hash, "fromBlock": "0x1"})
	require.Equal(t, -32602, e.Code)
}

// TestGetLogs_Limits are V2 and V4, checked before a block is read, and F6.
func TestGetLogs_Limits(t *testing.T) {
	c := newLogChain(t, func(cfg *tests.Configs) {
		cfg.Exec.LogQueryParameterLimit = 2
		cfg.Exec.IndexedLogsBlockRangeLimit = 1
		cfg.Exec.UnindexedLogsBlockRangeLimit = 1
		cfg.Exec.LogQueryResultLimit = 1
	})
	headers, finds := c.fc.Calls("Header"), c.fc.Calls("FindLogs")
	require.Equal(t, "too many query parameters, the limit is 2",
		c.h.Fails("eth_getLogs", map[string]any{"address": []any{emitter, other}, "topics": []any{topicA}}).Message)
	require.Equal(t, "too many query parameters, the limit is 2",
		c.h.Fails("eth_newFilter", map[string]any{"address": []any{emitter, other}, "topics": []any{topicA}}).Message)
	require.Equal(t, "too wide blocks range, the limit is 1", c.h.Fails("eth_getLogs", map[string]any{"fromBlock": "0x0", "address": emitter}).Message)
	require.Equal(t, "too wide blocks range, the limit is 1", c.h.Fails("eth_getLogs", map[string]any{"fromBlock": "0x0"}).Message)
	require.Equal(t, headers, c.fc.Calls("Header"))
	require.Equal(t, finds, c.fc.Calls("FindLogs"))

	require.Equal(t, "too many results, consider narrowing your query criteria, the limit is 1",
		c.h.Fails("eth_getLogs", map[string]any{"fromBlock": "0x1", "toBlock": "0x2"}).Message)
	var logs []types.Log
	c.h.MustCall(&logs, "eth_getLogs", map[string]any{"fromBlock": "0x2", "toBlock": "0x2"})
	require.Len(t, logs, 2, "a single block has no result limit")
}

func TestGetLogs_AScanEndsAtTheFirstBlockItCannotRead(t *testing.T) {
	c := newLogChain(t)
	c.fc.SetMissing(2)
	var logs []types.Log
	c.h.MustCall(&logs, "eth_getLogs", map[string]any{"fromBlock": "0x1", "toBlock": "0x3"})
	require.Len(t, logs, 1)
	require.Equal(t, uint64(1), logs[0].BlockNumber)
}

func TestFilters_BlockFilter(t *testing.T) {
	c := newLogChain(t)
	var id string
	c.h.MustCall(&id, "eth_newBlockFilter")
	require.Equal(t, "[]", string(c.h.MustCall(nil, "eth_getFilterChanges", id)))
	c.emit(t)
	c.emit(t)
	var hashes []common.Hash
	tests.WaitFor(t, func() bool {
		var got []common.Hash
		c.h.MustCall(&got, "eth_getFilterChanges", id)
		hashes = append(hashes, got...)
		return len(hashes) == 2
	})
	block5, err := c.fc.Block(5)
	require.NoError(t, err)
	require.Equal(t, block5.Hash, hashes[1])
	require.Equal(t, "[]", string(c.h.MustCall(nil, "eth_getFilterChanges", id)))

	require.Equal(t, "filter not found", c.h.Fails("eth_getFilterLogs", id).Message, "a block filter has no logs")
	require.Equal(t, "true", string(c.h.MustCall(nil, "eth_uninstallFilter", id)))
	require.Equal(t, "false", string(c.h.MustCall(nil, "eth_uninstallFilter", id)))
	require.Equal(t, "filter not found", c.h.Fails("eth_getFilterChanges", id).Message)
}

// TestFilters_PendingTransactions holds F7: whole transactions carry no
// block fields and are priced by their own gas price.
func TestFilters_PendingTransactions(t *testing.T) {
	c := newLogChain(t)
	var hashesID, fullID string
	c.h.MustCall(&hashesID, "eth_newPendingTransactionFilter")
	c.h.MustCall(&fullID, "eth_newPendingTransactionFilter", true)
	require.Equal(t, "null", string(c.h.MustCall(nil, "eth_getFilterChanges", hashesID)), "none is null, as in the reference")

	tx := c.w.Transfer(t, 9, emitter, big.NewInt(1))
	tests.WaitFor(t, func() bool { return c.fc.Calls("SubscribeTxs") >= 1 })
	c.fc.EmitTxs(types.Transactions{tx})
	var hashes []common.Hash
	tests.WaitFor(t, func() bool {
		c.h.MustCall(&hashes, "eth_getFilterChanges", hashesID)
		return len(hashes) == 1
	})
	require.Equal(t, tx.Hash(), hashes[0])
	var full []format.Transaction
	c.h.MustCall(&full, "eth_getFilterChanges", fullID)
	require.Len(t, full, 1)
	require.Nil(t, full[0].BlockHash)
	require.Equal(t, tx.GasPrice(), full[0].GasPrice.ToInt())
}

func TestFilters_LogFilter(t *testing.T) {
	c := newLogChain(t)
	var id, pending string
	c.h.MustCall(&id, "eth_newFilter", map[string]any{"address": emitter})
	c.h.MustCall(&pending, "eth_newFilter", map[string]any{"fromBlock": "pending", "toBlock": "pending"})
	tx := c.emit(t)
	var logs []types.Log
	tests.WaitFor(t, func() bool {
		c.h.MustCall(&logs, "eth_getFilterChanges", id)
		return len(logs) == 1
	})
	require.Equal(t, tx.Hash(), logs[0].TxHash)
	require.Equal(t, "[]", string(c.h.MustCall(nil, "eth_getFilterChanges", pending)), "pending logs never arrive")

	c.h.MustCall(&logs, "eth_getFilterLogs", id)
	require.Len(t, logs, 1, "the query again: without a range, the head")
	var ranged string
	c.h.MustCall(&ranged, "eth_newFilter", map[string]any{"fromBlock": "0x0", "address": emitter})
	c.h.MustCall(&logs, "eth_getFilterLogs", ranged)
	require.Len(t, logs, 4)
	e := c.h.Fails("eth_newFilter", map[string]any{"fromBlock": "0x5", "toBlock": "0x2"})
	require.Equal(t, "invalid from and to block combination: from > to", e.Message)
}

// TestFilters_SharedLogsAreNotChanged is P8: normalising works on copies.
func TestFilters_SharedLogsAreNotChanged(t *testing.T) {
	c := newLogChain(t)
	var id string
	c.h.MustCall(&id, "eth_newFilter", map[string]any{})
	consumer := c.h.Chain.Heads().Subscribe()
	defer consumer.Close()
	tx := c.w.Transfer(t, 9, emitter, big.NewInt(1))
	block := c.fc.AddAndPublish(types.Transactions{tx}, []*types.Log{{Address: emitter}})
	c.h.WaitHead(block.NumberU64())
	ev := <-consumer.Events()
	shared := ev.Logs[0]
	require.Nil(t, shared.Topics)
	var logs []map[string]any
	tests.WaitFor(t, func() bool {
		c.h.MustCall(&logs, "eth_getFilterChanges", id)
		return len(logs) == 1
	})
	require.Equal(t, []any{}, logs[0]["topics"])
	require.Nil(t, shared.Topics, "the shared log was changed")
	c.h.MustCall(&logs, "eth_getLogs", map[string]any{"fromBlock": "0x4"})
	require.Nil(t, shared.Topics)
}

// TestFilters_Expiry is D6.
func TestFilters_Expiry(t *testing.T) {
	c := newLogChain(t)
	var polled, idle string
	c.h.MustCall(&polled, "eth_newBlockFilter")
	c.h.MustCall(&idle, "eth_newBlockFilter")
	timeout := c.h.Env.Config().FilterTimeout
	require.Equal(t, 5*time.Minute, timeout)
	c.s.mu.Lock()
	c.s.filters[idle].lastPoll = time.Now().Add(-timeout - time.Second)
	c.s.mu.Unlock()
	c.s.expire(time.Now(), timeout)
	c.h.MustCall(nil, "eth_getFilterChanges", polled)
	require.Equal(t, "filter not found", c.h.Fails("eth_getFilterChanges", idle).Message)
	require.NotEqual(t, polled, idle)
	require.Len(t, polled, len("0x")+32, "ids are 16 random bytes")
}

// TestFilters_OneCopyOfAnEventForManyFilters is I17.
func TestFilters_OneCopyOfAnEventForManyFilters(t *testing.T) {
	c := newLogChain(t)
	ids := make([]string, 1000)
	for i := range ids {
		c.h.MustCall(&ids[i], "eth_newBlockFilter")
	}
	before := c.s.blocks.len()
	c.emit(t)
	tests.WaitFor(t, func() bool { return c.s.blocks.len() == before+1 })
	for _, id := range ids[:500] {
		var hashes []common.Hash
		c.h.MustCall(&hashes, "eth_getFilterChanges", id)
		require.Len(t, hashes, 1)
	}
	c.s.expire(time.Now(), time.Hour)
	require.Equal(t, 1, c.s.blocks.len(), "kept for the filters that did not poll")
	for _, id := range ids[500:] {
		c.h.MustCall(nil, "eth_getFilterChanges", id)
	}
	c.s.expire(time.Now(), time.Hour)
	require.Zero(t, c.s.blocks.len(), "every live filter polled")
}

func TestFilters_TheNumberOfFiltersIsBounded(t *testing.T) {
	c := newLogChain(t)
	for range maxFilters {
		c.h.MustCall(nil, "eth_newBlockFilter")
	}
	require.Equal(t, "too many filters, the limit is 100000", c.h.Fails("eth_newBlockFilter").Message)
	require.Equal(t, float64(maxFilters), tests.Scrape(t, c.h.Registry, "rpc_subscriptions", map[string]string{"kind": "block_filters"}))
}

// TestFilters_AFilterThatFellBehindTheJournalIsRemoved is I17: the journal
// keeps a bounded number of events; a filter whose events left it is
// removed, while one that polled gets every event.
func TestFilters_AFilterThatFellBehindTheJournalIsRemoved(t *testing.T) {
	c := newLogChain(t)
	var behind, current string
	c.h.MustCall(&behind, "eth_newBlockFilter")
	c.h.MustCall(&current, "eth_newBlockFilter")
	for i := range journalLimit {
		c.s.blocks.add(common.Hash{byte(i)})
	}
	var hashes []common.Hash
	c.h.MustCall(&hashes, "eth_getFilterChanges", current)
	require.Len(t, hashes, journalLimit)
	c.s.blocks.add(common.Hash{1})
	c.h.MustCall(&hashes, "eth_getFilterChanges", current)
	require.Len(t, hashes, 1)
	require.Equal(t, "filter not found", c.h.Fails("eth_getFilterChanges", behind).Message)
	require.Equal(t, journalLimit, c.s.blocks.len())
}

// TestFilters_ALostFeedRemovesTheFiltersOfItsKind is I18: when the chain's
// head loop restarts, its consumers missed blocks; the polled filters of
// blocks and logs answer filter not found, and a filter installed afterwards
// gets every event.
func TestFilters_ALostFeedRemovesTheFiltersOfItsKind(t *testing.T) {
	c := newLogChain(t)
	var blocks, logs, txs string
	c.h.MustCall(&blocks, "eth_newBlockFilter")
	c.h.MustCall(&logs, "eth_newFilter", map[string]any{})
	c.h.MustCall(&txs, "eth_newPendingTransactionFilter")

	c.fc.SetMode("ChainInfo", tests.Panic)
	c.fc.AddAndPublish(nil)
	tests.WaitFor(t, func() bool {
		return tests.Scrape(t, c.h.Registry, "rpc_loop_restarts_total", map[string]string{"loop": "head"}) >= 1
	})
	c.fc.SetMode("ChainInfo", tests.Normal)
	tests.WaitFor(t, func() bool { _, err := c.h.Call("eth_getFilterChanges", blocks); return err != nil })
	require.Equal(t, "filter not found", c.h.Fails("eth_getFilterChanges", logs).Message)
	c.h.MustCall(nil, "eth_getFilterChanges", txs)

	tests.WaitFor(t, func() bool { return c.h.Chain.Heads().Len() >= 2 })
	var fresh string
	c.h.MustCall(&fresh, "eth_newBlockFilter")
	c.emit(t)
	var hashes []common.Hash
	tests.WaitFor(t, func() bool {
		c.h.MustCall(&hashes, "eth_getFilterChanges", fresh)
		return len(hashes) == 1
	})
}

// TestFilters_ARestartedLoopRemovesTheFiltersOfItsKind is P3 for the loops
// of the filters.
func TestFilters_ARestartedLoopRemovesTheFiltersOfItsKind(t *testing.T) {
	c := newLogChain(t)
	var blocks, txs string
	c.h.MustCall(&blocks, "eth_newBlockFilter")
	c.h.MustCall(&txs, "eth_newPendingTransactionFilter")
	done := testContextDone()
	c.s.poolLoop(done)
	require.Equal(t, "filter not found", c.h.Fails("eth_getFilterChanges", txs).Message)
	c.h.MustCall(nil, "eth_getFilterChanges", blocks)
	c.s.headLoop(done)
	require.Equal(t, "filter not found", c.h.Fails("eth_getFilterChanges", blocks).Message)
}

// TestSubscriptions_NewHeadsAreBuiltOncePerBlock is I16.
func TestSubscriptions_NewHeadsAreBuiltOncePerBlock(t *testing.T) {
	c := newLogChain(t)
	limits := dispatch.StreamLimits{MaxSubscriptions: 10, Buffer: 100}
	streams := make([]*tests.Stream, 5)
	for i := range streams {
		streams[i] = c.h.Stream(limits)
		_, err := streams[i].Call("eth_subscribe", "newHeads")
		require.NoError(t, err)
	}
	receipts := c.fc.Calls("Receipts")
	c.emit(t)
	for _, s := range streams {
		var note struct {
			Params struct {
				Result struct {
					Number string `json:"number"`
				} `json:"result"`
			} `json:"params"`
		}
		require.NoError(t, json.Unmarshal(s.Next(), &note))
		require.Equal(t, "0x4", note.Params.Result.Number)
	}
	require.Equal(t, receipts+1, c.fc.Calls("Receipts"))
}

func TestSubscriptions_LogsAndPendingTransactions(t *testing.T) {
	c := newLogChain(t)
	s := c.h.Stream(dispatch.StreamLimits{MaxSubscriptions: 10, Buffer: 100})
	_, err := s.Call("eth_subscribe", "logs", map[string]any{"address": emitter})
	require.NoError(t, err)
	_, err = s.Call("eth_subscribe", "logs", map[string]any{"address": other})
	require.NoError(t, err)
	tx := c.emit(t)
	var note struct {
		Params struct {
			Result types.Log `json:"result"`
		} `json:"params"`
	}
	require.NoError(t, json.Unmarshal(s.Next(), &note))
	require.Equal(t, tx.Hash(), note.Params.Result.TxHash)

	_, err = s.Call("eth_subscribe", "newPendingTransactions", true)
	require.NoError(t, err)
	pooled := c.w.Transfer(t, 9, emitter, big.NewInt(1))
	c.fc.EmitTxs(types.Transactions{pooled})
	var full struct {
		Params struct {
			Result format.Transaction `json:"result"`
		} `json:"params"`
	}
	require.NoError(t, json.Unmarshal(s.Next(), &full))
	require.Equal(t, pooled.Hash(), full.Params.Result.Hash)

	_, err = s.Call("eth_subscribe", "logs", map[string]any{"address": []any{emitter, other}, "topics": []any{topicA}})
	require.NoError(t, err)
	_, err = s.Call("eth_subscribe", "syncing")
	require.Equal(t, `no "syncing" subscription in eth namespace`, err.Error())

	raw, err := s.Call("eth_subscribe", "newHeads")
	require.NoError(t, err)
	var id string
	require.NoError(t, json.Unmarshal(raw, &id))
	other := c.h.Stream(dispatch.StreamLimits{MaxSubscriptions: 10, Buffer: 100})
	_, err = other.Call("eth_unsubscribe", id)
	require.Equal(t, "subscription not found", err.Error(), "ids belong to their connection (V8)")
	require.Equal(t, "true", string(must(s.Call("eth_unsubscribe", id))))
}

// TestSubscriptions_ALostFeedEndsTheConnection is I18 for subscriptions.
func TestSubscriptions_ALostFeedEndsTheConnection(t *testing.T) {
	c := newLogChain(t)
	s := c.h.Stream(dispatch.StreamLimits{MaxSubscriptions: 10, Buffer: 100})
	_, err := s.Call("eth_subscribe", "logs", map[string]any{})
	require.NoError(t, err)
	c.fc.SetMode("ChainInfo", tests.Panic)
	c.fc.AddAndPublish(nil)
	select {
	case reason := <-s.Closed:
		require.Contains(t, reason, "fell behind its feed")
	case <-time.After(10 * time.Second):
		t.Fatal("the subscription went on with a gap")
	}
	c.fc.SetMode("ChainInfo", tests.Normal)
}

func TestSubscriptions_TheParameterLimitHolds(t *testing.T) {
	c := newLogChain(t, func(cfg *tests.Configs) { cfg.Exec.LogQueryParameterLimit = 1 })
	s := c.h.Stream(dispatch.StreamLimits{MaxSubscriptions: 10, Buffer: 100})
	_, err := s.Call("eth_subscribe", "logs", map[string]any{"address": []any{emitter, other}})
	require.Equal(t, "too many query parameters, the limit is 1", err.Error())
	_, err = c.h.Call("eth_subscribe", "logs", map[string]any{})
	require.Equal(t, "notifications not supported", err.Error())
}

func testContextDone() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func must(raw json.RawMessage, err error) json.RawMessage {
	if err != nil {
		panic(err)
	}
	return raw
}

// TestGetLogs_ReadsHeadersNotBlocks is I13 for the timestamps of logs.
func TestGetLogs_ReadsHeadersNotBlocks(t *testing.T) {
	c := newLogChain(t)
	blocks := c.fc.Calls("Block")
	c.h.MustCall(nil, "eth_getLogs", map[string]any{"fromBlock": "0x0", "address": emitter})
	c.h.MustCall(nil, "eth_getLogs", map[string]any{"fromBlock": "0x0"})
	require.Equal(t, blocks, c.fc.Calls("Block"))
}
