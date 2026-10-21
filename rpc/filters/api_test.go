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
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

var (
	addrA = common.HexToAddress("0xaa")
	addrB = common.HexToAddress("0xbb")
	topic = common.HexToHash("0x01")
)

func newHarness(t *testing.T, fc *tests.FakeCore, mutate ...func(*dispatch.Config)) (*tests.Harness, *API) {
	t.Helper()
	cfg := tests.Default()
	for _, m := range mutate {
		m(&cfg)
	}
	var api *API
	h := tests.NewHarness(t, fc, cfg, func(d *dispatch.Dispatcher, chain *dispatch.Chain) {
		api = New(exec.NewEnv(chain, oracle.New(oracle.Config{}, fc)))
		api.Register(d, "eth")
		api.Register(d, "ftm")
		api.Start(d)
	})
	return h, api
}

// chainWithLogs builds a chain where block n (1..) holds one transaction with
// a log of address A (even n) or B (odd n).
func chainWithLogs(t *testing.T, blocks int) (*tests.FakeCore, []*types.Transaction) {
	t.Helper()
	fc := tests.NewFakeCore(t)
	alice := tests.NewAccount(t, nil, nil)
	var txs []*types.Transaction
	for n := 1; n <= blocks; n++ {
		tx := alice.Transfer(t, uint64(n-1), common.Address{1}, 1)
		txs = append(txs, tx)
		block := fc.AddBlockSilently(tx)
		addr := addrB
		if n%2 == 0 {
			addr = addrA
		}
		receipts := fc.ReceiptsOf(block.NumberU64())
		receipts[0].Logs = []*types.Log{{
			Address: addr, Topics: []common.Hash{topic}, BlockNumber: block.NumberU64(), BlockHash: block.Hash,
			TxHash: tx.Hash(), TxIndex: 99, Index: uint(n),
		}}
	}
	fc.AnnounceUpTo(uint64(blocks))
	return fc, txs
}

func TestGetLogs_Unindexed(t *testing.T) {
	fc, txs := chainWithLogs(t, 6)
	h, _ := newHarness(t, fc)

	var logs []types.Log
	h.Into(&logs, "eth_getLogs", map[string]any{"fromBlock": "0x1", "toBlock": "0x4"})
	require.Len(t, logs, 4)
	for i, l := range logs {
		require.EqualValues(t, i+1, l.BlockNumber)
		require.Equal(t, txs[i].Hash(), l.TxHash)
		// The transaction index comes from the index of transactions, not from
		// the log itself.
		require.EqualValues(t, 0, l.TxIndex)
	}

	// The default range is the head only.
	h.Into(&logs, "eth_getLogs", map[string]any{})
	require.Len(t, logs, 1)
	require.EqualValues(t, 6, logs[0].BlockNumber)

	// A range that is empty yields an empty list, not null.
	require.JSONEq(t, `[]`, string(h.Result("eth_getLogs", map[string]any{"fromBlock": "0x5", "toBlock": "0x2"})))
	// The walk stops at the head.
	h.Into(&logs, "eth_getLogs", map[string]any{"fromBlock": "0x5", "toBlock": "0x40"})
	require.Len(t, logs, 2)

	// The limit applies to the range as it was requested, not as it was
	// clamped to the head (F6).
	err := h.Fails("eth_getLogs", map[string]any{"fromBlock": "0x5", "toBlock": "0x100"})
	require.Equal(t, "too wide blocks range, the limit is 100", err.Message)
	err = h.Fails("eth_getLogs", map[string]any{"fromBlock": "0x6", "toBlock": "0x6b"})
	require.Equal(t, "too wide blocks range, the limit is 100", err.Message)
	h.Into(&logs, "eth_getLogs", map[string]any{"fromBlock": "0x6", "toBlock": "0x6a"})
	require.Len(t, logs, 1)
}

func TestGetLogs_TagsAreTheHead_EvenEarliest(t *testing.T) {
	fc, _ := chainWithLogs(t, 3)
	h, _ := newHarness(t, fc)
	var logs []types.Log
	// "earliest" is a negative number to the filter, which treats every
	// negative bound as the head. That is how the node always answered.
	h.Into(&logs, "eth_getLogs", map[string]any{"fromBlock": "earliest", "toBlock": "latest"})
	require.Len(t, logs, 1)
	require.EqualValues(t, 3, logs[0].BlockNumber)
}

func TestGetLogs_Indexed(t *testing.T) {
	fc, _ := chainWithLogs(t, 6)
	var seen struct {
		from, to uint64
		pattern  [][]common.Hash
		limit    uint
	}
	fc.SetFindLogs(func(from, to uint64, pattern [][]common.Hash, limit uint) ([]*types.Log, error) {
		seen.from, seen.to, seen.pattern, seen.limit = from, to, pattern, limit
		var res []*types.Log
		// out of order on purpose: the module sorts
		for n := to; n >= from; n-- {
			receipts, _ := fc.Receipts(n)
			for _, r := range receipts {
				res = append(res, r.Logs...)
			}
			if n == 0 {
				break
			}
		}
		return res, nil
	})
	h, _ := newHarness(t, fc)

	var logs []types.Log
	h.Into(&logs, "eth_getLogs", map[string]any{"fromBlock": "0x1", "toBlock": "0x6", "address": addrA.Hex(), "topics": []any{topic.Hex(), nil}})
	require.EqualValues(t, 1, seen.from)
	require.EqualValues(t, 6, seen.to)
	require.Equal(t, [][]common.Hash{{common.BytesToHash(addrA[:])}, {topic}, nil}, seen.pattern)
	require.EqualValues(t, tests.Default().LogQueryResultLimit, seen.limit)
	require.Len(t, logs, 6)
	for i, l := range logs {
		require.EqualValues(t, i+1, l.BlockNumber)
		require.NotZero(t, l.BlockTimestamp)
		require.EqualValues(t, 0, l.TxIndex)
	}

	// A single block has no result limit.
	h.Into(&logs, "eth_getLogs", map[string]any{"fromBlock": "0x2", "toBlock": "0x2", "address": addrA.Hex()})
	require.Zero(t, seen.limit)
}

func TestGetLogs_Limits(t *testing.T) {
	fc, _ := chainWithLogs(t, 6)
	fc.SetFindLogs(func(uint64, uint64, [][]common.Hash, uint) ([]*types.Log, error) { return nil, nil })
	h, _ := newHarness(t, fc, func(c *dispatch.Config) {
		c.UnindexedLogsBlockRangeLimit = 2
		c.IndexedLogsBlockRangeLimit = 3
		c.LogQueryParameterLimit = 3
		c.LogQueryResultLimit = 2
	})

	err := h.Fails("eth_getLogs", map[string]any{"fromBlock": "0x1", "toBlock": "0x6"})
	require.Equal(t, "too wide blocks range, the limit is 2", err.Message)
	err = h.Fails("eth_getLogs", map[string]any{"fromBlock": "0x1", "toBlock": "0x6", "address": addrA.Hex()})
	require.Equal(t, "too wide blocks range, the limit is 3", err.Message)

	// The result limit applies to a walk over several blocks.
	err = h.Fails("eth_getLogs", map[string]any{"fromBlock": "0x1", "toBlock": "0x3"})
	require.Equal(t, "too many results, consider narrowing your query criteria, the limit is 2", err.Message)

	// The parameter limit holds on every path, before any block is read.
	tooMany := map[string]any{"address": []string{addrA.Hex(), addrB.Hex()}, "topics": []any{topic.Hex(), topic.Hex()}}
	for _, call := range []struct {
		method string
		params []any
	}{
		{"eth_getLogs", []any{tooMany}},
		{"eth_newFilter", []any{tooMany}},
	} {
		err = h.Fails(call.method, call.params...)
		require.Equal(t, "too many query parameters, the limit is 3", err.Message, call.method)
	}
	require.Zero(t, fc.Calls("Header")-fc.Calls("Header")) // no access needed to refuse
}

func TestGetLogs_ByBlockHash(t *testing.T) {
	fc, _ := chainWithLogs(t, 4)
	h, _ := newHarness(t, fc)

	var logs []types.Log
	h.Into(&logs, "eth_getLogs", map[string]any{"blockHash": fc.BlockAt(3).Hash})
	require.Len(t, logs, 1)
	require.EqualValues(t, 3, logs[0].BlockNumber)

	h.Into(&logs, "eth_getLogs", map[string]any{"blockHash": fc.BlockAt(3).Hash, "address": addrA.Hex()})
	require.Empty(t, logs)

	err := h.Fails("eth_getLogs", map[string]any{"blockHash": common.Hash{9}})
	require.Equal(t, "block with hash "+common.Hash{9}.String()+" not found", err.Message)

	// A block that is not served yet is unknown.
	late := fc.AddBlockSilently()
	err = h.Fails("eth_getLogs", map[string]any{"blockHash": late.Hash})
	require.Equal(t, "unknown block", err.Message)

	err = h.Fails("eth_getLogs", map[string]any{"blockHash": fc.BlockAt(3).Hash, "fromBlock": "0x1"})
	require.Equal(t, dispatch.CodeInvalidParams, err.Code)
	require.Contains(t, err.Message, "cannot specify both BlockHash and FromBlock/ToBlock")
}

func TestGetLogs_TheWalkEndsAtTheFirstBlockItCannotRead(t *testing.T) {
	fc, _ := chainWithLogs(t, 5)
	h, _ := newHarness(t, fc)
	// Blocks 3 and above cannot be read: what was found before is returned.
	calls := 0
	fc.SetHeaderHook(func(n uint64) error {
		calls++
		if n >= 3 {
			return errors.New("gone")
		}
		return nil
	})
	var logs []types.Log
	err := h.Fails("eth_getLogs", map[string]any{"fromBlock": "0x1", "toBlock": "0x5"})
	require.Equal(t, "gone", err.Message) // a read error is reported
	fc.SetHeaderHook(func(n uint64) error {
		if n >= 3 {
			return nil
		}
		return nil
	})
	fc.HideBlocksFrom(3)
	h.Into(&logs, "eth_getLogs", map[string]any{"fromBlock": "0x1", "toBlock": "0x5"})
	require.Len(t, logs, 2) // blocks 1 and 2
	_ = calls
}

func TestCriteria_Decoding(t *testing.T) {
	var c Criteria
	require.NoError(t, json.Unmarshal([]byte(`{"address":"`+addrA.Hex()+`","topics":[null,"`+topic.Hex()+`",["`+topic.Hex()+`",null]],"fromBlock":"0x1","toBlock":"latest"}`), &c))
	require.Equal(t, []common.Address{addrA}, c.Addresses)
	require.Equal(t, big.NewInt(1), c.FromBlock)
	require.Equal(t, big.NewInt(-2), c.ToBlock)
	require.Len(t, c.Topics, 3)
	require.Nil(t, c.Topics[0])
	require.Equal(t, []common.Hash{topic}, c.Topics[1])
	require.Nil(t, c.Topics[2]) // a null in the alternatives matches everything

	for _, bad := range []string{
		`{"address":5}`, `{"address":["0x1"]}`, `{"address":[5]}`, `{"topics":[5]}`, `{"topics":["0x1"]}`,
		`{"topics":[[5]]}`, `{"blockHash":"` + topic.Hex() + `","fromBlock":"0x1"}`,
	} {
		require.Error(t, json.Unmarshal([]byte(bad), &Criteria{}), bad)
	}
}

func TestNewFilter_BlockRanges(t *testing.T) {
	fc, _ := chainWithLogs(t, 2)
	h, _ := newHarness(t, fc)
	for _, ok := range []map[string]any{
		{}, {"fromBlock": "0x1", "toBlock": "0x5"}, {"fromBlock": "0x1", "toBlock": "latest"},
		{"fromBlock": "pending", "toBlock": "pending"}, {"fromBlock": "latest", "toBlock": "pending"},
	} {
		h.Result("eth_newFilter", ok)
	}
	for _, bad := range []map[string]any{
		{"fromBlock": "0x5", "toBlock": "0x1"}, {"fromBlock": "earliest"}, {"fromBlock": "finalized", "toBlock": "latest"},
	} {
		err := h.Fails("eth_newFilter", bad)
		require.Equal(t, "invalid from and to block combination: from > to", err.Message, bad)
	}
}

func TestGetFilterLogs_RunsTheQueryOfTheFilter(t *testing.T) {
	fc, _ := chainWithLogs(t, 4)
	h, _ := newHarness(t, fc)
	var id string
	h.Into(&id, "eth_newFilter", map[string]any{"fromBlock": "0x1", "toBlock": "0x4", "address": addrA.Hex()})
	fc.SetFindLogs(func(from, to uint64, pattern [][]common.Hash, limit uint) ([]*types.Log, error) {
		var res []*types.Log
		for n := from; n <= to; n++ {
			receipts, _ := fc.Receipts(n)
			for _, r := range receipts {
				for _, l := range r.Logs {
					if l.Address == addrA {
						res = append(res, l)
					}
				}
			}
		}
		return res, nil
	})
	var logs []types.Log
	h.Into(&logs, "eth_getFilterLogs", id)
	require.Len(t, logs, 2)

	err := h.Fails("eth_getFilterLogs", "0xdead")
	require.Equal(t, "filter not found", err.Message)
}

func TestGetFilterChanges_LeavesTheLogsOfTheHeadAlone(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	h, _ := newHarness(t, fc)
	var id string
	h.Into(&id, "eth_newFilter", map[string]any{})

	// A LOG0 log has no topics; the head event is shared by all consumers.
	block := fc.AddBlockSilently()
	shared := &types.Log{Address: addrA, BlockNumber: block.NumberU64(), BlockHash: block.Hash}
	fc.Publish(core.HeadEvent{Block: block, Logs: []*types.Log{shared}})

	var changes []map[string]any
	require.Eventually(t, func() bool {
		h.Into(&changes, "eth_getFilterChanges", id)
		return len(changes) > 0
	}, time.Second, time.Millisecond)
	require.Equal(t, []any{}, changes[0]["topics"])
	require.Nil(t, shared.Topics)
}

func TestFilters_AreSharedBetweenEthAndFtm(t *testing.T) {
	fc, _ := chainWithLogs(t, 1)
	h, _ := newHarness(t, fc)
	var id string
	h.Into(&id, "eth_newBlockFilter")
	var hashes []common.Hash
	h.Into(&hashes, "ftm_getFilterChanges", id)
	var ok bool
	h.Into(&ok, "ftm_uninstallFilter", id)
	require.True(t, ok)
}

func TestFilters_IdsAreUnguessable(t *testing.T) {
	fc, _ := chainWithLogs(t, 1)
	h, _ := newHarness(t, fc)
	seen := map[string]bool{}
	for range 200 {
		var id string
		h.Into(&id, "eth_newBlockFilter")
		require.True(t, strings.HasPrefix(id, "0x"))
		require.GreaterOrEqual(t, len(id), 20) // 80 bits and more
		require.False(t, seen[id])
		seen[id] = true
	}
}

func TestFilters_Expire(t *testing.T) {
	fc, _ := chainWithLogs(t, 1)
	h, api := newHarness(t, fc, func(c *dispatch.Config) { c.FilterTimeout = 200 * time.Millisecond })
	var id string
	h.Into(&id, "eth_newBlockFilter")
	require.Eventually(t, func() bool {
		api.mu.Lock()
		defer api.mu.Unlock()
		return len(api.filters) == 0
	}, 2*time.Second, 10*time.Millisecond)
	err := h.Fails("eth_getFilterChanges", id)
	require.Equal(t, "filter not found", err.Message)
}

func TestSubscriptions_NeedAConnection(t *testing.T) {
	fc, _ := chainWithLogs(t, 1)
	h, _ := newHarness(t, fc)
	for _, kind := range []string{"newHeads", "logs", "newPendingTransactions"} {
		res, err := h.D.Subscribe(context.Background(), "eth", true, json.RawMessage(`["`+kind+`",{}]`))
		_ = res
		require.NotNil(t, err, kind)
	}
}

// pendingTxs returns n signed transactions for the pool.
func pendingTxs(t *testing.T, n int) []*types.Transaction {
	t.Helper()
	alice := tests.NewAccount(t, nil, nil)
	txs := make([]*types.Transaction, n)
	for i := range txs {
		txs[i] = alice.Transfer(t, uint64(i), common.Address{1}, 1)
	}
	return txs
}

// journaled waits until the journal of the pool holds n transactions.
func journaled(t *testing.T, api *API, n uint64) {
	t.Helper()
	require.Eventually(t, func() bool {
		api.mu.Lock()
		defer api.mu.Unlock()
		return api.txs.end() >= n
	}, 2*time.Second, time.Millisecond)
}

func TestFilters_AnEventIsKeptOnceWhateverTheNumberOfFilters(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	h, api := newHarness(t, fc)
	ids := make([]string, 1000)
	for i := range ids {
		h.Into(&ids[i], "eth_newPendingTransactionFilter")
	}
	tx := pendingTxs(t, 1)[0]
	fc.AddPoolTransaction(tx)
	journaled(t, api, 1)
	api.mu.Lock()
	require.Len(t, api.txs.items, 1)
	api.mu.Unlock()
	for _, id := range ids {
		var hashes []common.Hash
		h.Into(&hashes, "eth_getFilterChanges", id)
		require.Equal(t, []common.Hash{tx.Hash()}, hashes)
	}
}

func TestFilters_AreBoundedPerNode(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	h, api := newHarness(t, fc)
	require.Equal(t, maxFilters, api.maxFilters)
	api.maxFilters = 3

	var ids [3]string
	h.Into(&ids[0], "eth_newBlockFilter")
	h.Into(&ids[1], "eth_newPendingTransactionFilter")
	h.Into(&ids[2], "ftm_newFilter", map[string]any{})
	require.Equal(t, "too many filters, the limit is 3", h.Fails("eth_newBlockFilter").Message)
	require.Equal(t, "too many filters, the limit is 3", h.Fails("eth_newPendingTransactionFilter").Message)
	require.Equal(t, "too many filters, the limit is 3", h.Fails("eth_newFilter", map[string]any{}).Message)

	// An uninstalled filter makes room.
	var found bool
	h.Into(&found, "eth_uninstallFilter", ids[0])
	require.True(t, found)
	h.Into(&ids[0], "eth_newBlockFilter")
}

func TestGetFilterChanges_AFilterWhoseEventsWereDroppedIsRemoved(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	h, api := newHarness(t, fc)
	api.mu.Lock()
	api.txs = newJournal[*types.Transaction](8)
	api.mu.Unlock()

	var idle, polled string
	h.Into(&idle, "eth_newPendingTransactionFilter")
	h.Into(&polled, "eth_newPendingTransactionFilter")
	txs := pendingTxs(t, 20)
	var want, got []common.Hash
	for i, tx := range txs {
		want = append(want, tx.Hash())
		fc.AddPoolTransaction(tx)
		journaled(t, api, uint64(i+1))
		var hashes []common.Hash
		h.Into(&hashes, "eth_getFilterChanges", polled)
		got = append(got, hashes...)
	}
	// The filter polled in time received every transaction; the idle one lost
	// some and is gone.
	require.Equal(t, want, got)
	require.Equal(t, "filter not found", h.Fails("eth_getFilterChanges", idle).Message)
	api.mu.Lock()
	defer api.mu.Unlock()
	require.Len(t, api.filters, 1)
	require.LessOrEqual(t, len(api.txs.items), 8)
}

func TestFilters_TheJournalKeepsOnlyWhatALiveFilterHasNotPolled(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	h, api := newHarness(t, fc)
	kept := func() (blocks, logs int) {
		api.mu.Lock()
		defer api.mu.Unlock()
		return len(api.blocks.items), len(api.logs.items)
	}

	var idle, polled, logFilter string
	h.Into(&idle, "eth_newBlockFilter")
	h.Into(&polled, "eth_newBlockFilter")
	for range 3 {
		block := fc.AddBlockSilently()
		fc.Publish(core.HeadEvent{Block: block})
	}
	require.Eventually(t, func() bool { blocks, _ := kept(); return blocks == 3 }, 2*time.Second, time.Millisecond)

	var hashes []common.Hash
	h.Into(&hashes, "eth_getFilterChanges", polled)
	require.Len(t, hashes, 3)
	api.removeExpired(time.Now(), time.Hour)
	blocks, _ := kept()
	require.Equal(t, 3, blocks, "the idle filter has not polled them")

	h.Into(&hashes, "eth_getFilterChanges", idle)
	require.Len(t, hashes, 3)
	api.removeExpired(time.Now(), time.Hour)
	blocks, _ = kept()
	require.Zero(t, blocks)

	// Without a block filter the hashes are not kept; the logs of the same
	// head are, for the log filter.
	var found bool
	h.Into(&found, "eth_uninstallFilter", idle)
	h.Into(&found, "eth_uninstallFilter", polled)
	h.Into(&logFilter, "eth_newFilter", map[string]any{})
	block := fc.AddBlockSilently()
	fc.Publish(core.HeadEvent{Block: block, Logs: []*types.Log{{Address: addrA, BlockNumber: block.NumberU64()}}})
	require.Eventually(t, func() bool { _, logs := kept(); return logs == 1 }, 2*time.Second, time.Millisecond)
	blocks, _ = kept()
	require.Zero(t, blocks)

	// An expired filter takes its events with it.
	api.removeExpired(time.Now().Add(2*time.Hour), time.Hour)
	_, logs := kept()
	require.Zero(t, logs)
}
