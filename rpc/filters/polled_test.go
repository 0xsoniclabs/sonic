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
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// addLoggingBlock stores a block whose transaction emitted a log of the
// emitter with the topic and one of another address, and serves it.
func addLoggingBlock(t *testing.T, h *tests.Harness) uint64 {
	t.Helper()
	recipient := common.Address{0x42}
	transaction := tests.NewAccount().Sign(&types.LegacyTx{To: &recipient, Gas: 21000, GasPrice: big.NewInt(1e9)})
	number := h.Core.AddBlockWithReceipts(types.Transactions{transaction}, types.Receipts{{
		Status: types.ReceiptStatusSuccessful, CumulativeGasUsed: 21000,
		Logs: []*types.Log{{Address: emitter, Topics: []common.Hash{topic}}, {Address: common.Address{0x01}}},
	}})
	h.Core.SetHead(number)
	return number
}

func newFilter(t *testing.T, h *tests.Harness, method string, params ...any) dispatch.ID {
	t.Helper()
	var id dispatch.ID
	h.Call(t, &id, method, params...)
	return id
}

// changes polls a filter until it answers something, and answers that.
func changes[Result any](t *testing.T, h *tests.Harness, id dispatch.ID) Result {
	t.Helper()
	var result Result
	require.Eventually(t, func() bool {
		encoded := h.Result(t, "eth_getFilterChanges", id)
		require.NoError(t, json.Unmarshal([]byte(encoded), &result))
		return encoded != "[]" && encoded != "null"
	}, time.Second, time.Millisecond)
	return result
}

func TestPolledFilters_ReceiveTheEventsOfTheirKind(t *testing.T) {
	h := newHarness(t)
	blocks := newFilter(t, h, "eth_newBlockFilter")
	logs := newFilter(t, h, "eth_newFilter", map[string]any{"address": emitter})
	pending := newFilter(t, h, "eth_newFilter", map[string]any{"fromBlock": "pending", "toBlock": "pending"})
	number := addLoggingBlock(t, h)
	block, err := h.Core.Block(number)
	require.NoError(t, err)

	require.Equal(t, []common.Hash{block.Hash}, changes[[]common.Hash](t, h, blocks))
	selected := changes[[]map[string]any](t, h, logs)
	require.Len(t, selected, 1)
	require.Equal(t, emitter.Hex(), common.HexToAddress(selected[0]["address"].(string)).Hex())
	require.Equal(t, "[]", h.Result(t, "eth_getFilterChanges", blocks), "a poll takes what came since the last")
	require.Equal(t, "[]", h.Result(t, "eth_getFilterChanges", pending), "the pending block never comes")

	h.Result(t, "eth_getFilterLogs", logs)
	queries := h.Core.LogQueries()
	require.Equal(t, []common.Address{emitter}, queries[len(queries)-1].Addresses, "a query of the filter's criteria")
	require.Equal(t, number, queries[len(queries)-1].Head)
}

func TestPolledFilters_ServeTransactionsAsHashesOrInFull(t *testing.T) {
	h := newHarness(t)
	hashes := newFilter(t, h, "eth_newPendingTransactionFilter")
	full := newFilter(t, h, "eth_newPendingTransactionFilter", true)
	require.Equal(t, "null", h.Result(t, "eth_getFilterChanges", hashes), "no transactions are null")
	recipient := common.Address{0x42}
	transaction := tests.NewAccount().Sign(&types.DynamicFeeTx{ChainID: big.NewInt(tests.ChainID), To: &recipient, Gas: 21000,
		GasFeeCap: big.NewInt(5e9), GasTipCap: big.NewInt(1)})
	h.Core.AddPoolTransaction(transaction)
	require.Equal(t, []common.Hash{transaction.Hash()}, changes[[]common.Hash](t, h, hashes))
	whole := changes[[]map[string]any](t, h, full)
	require.Equal(t, transaction.Hash().Hex(), whole[0]["hash"])
	require.Equal(t, "0x12a05f200", whole[0]["gasPrice"], "priced by its fee cap")
	require.Nil(t, whole[0]["blockHash"])
}

func TestPolledFilters_AnswerNotFoundForAnUnknownFilter(t *testing.T) {
	h := newHarness(t)
	blocks := newFilter(t, h, "eth_newBlockFilter")
	notFound := &dispatch.Error{Code: dispatch.CodeDefault, Message: "filter not found"}
	require.Equal(t, notFound, h.Fail(t, "eth_getFilterLogs", blocks), "only a logs filter has logs to query")
	require.Equal(t, "true", h.Result(t, "eth_uninstallFilter", blocks))
	require.Equal(t, "false", h.Result(t, "eth_uninstallFilter", blocks))
	for _, method := range []string{"eth_getFilterChanges", "eth_getFilterLogs"} {
		require.Equal(t, notFound, h.Fail(t, method, blocks))
	}
}

func TestNewFilter_RefusesWhatTheLegacyFiltersRefuse(t *testing.T) {
	h := newHarness(t)
	require.Equal(t, &dispatch.Error{Code: dispatch.CodeDefault, Message: "invalid from and to block combination: from > to"},
		h.Fail(t, "eth_newFilter", map[string]any{"fromBlock": "0x2", "toBlock": "0x1"}))
	require.Equal(t, &dispatch.Error{Code: dispatch.CodeDefault, Message: "too many query parameters, the limit is 1000"},
		h.Fail(t, "eth_newFilter", map[string]any{"topics": []any{make([]common.Hash, 1001)}}))
}

// The logs of a block are shared, so the answer changes only a copy.
func TestPolledFilters_LeaveTheSharedLogsUnchanged(t *testing.T) {
	h := newHarness(t)
	logs := newFilter(t, h, "eth_newFilter", map[string]any{})
	number := addLoggingBlock(t, h)
	answered := changes[[]map[string]any](t, h, logs)
	require.Equal(t, []any{}, answered[1]["topics"])
	receipts, err := h.Core.Receipts(number)
	require.NoError(t, err)
	require.Nil(t, receipts[0].Logs[1].Topics)
}

// When the feed of the polled filters is lost, every filter it fed answers
// not found.
func TestPolledFilters_AreDroppedWithTheirFeed(t *testing.T) {
	h := newHarness(t)
	blocks := newFilter(t, h, "eth_newBlockFilter")
	h.Core.StopFeeds()
	require.Eventually(t, func() bool {
		return h.Serve(t, "eth_getFilterChanges", blocks).Error != nil
	}, time.Second, time.Millisecond)
}
