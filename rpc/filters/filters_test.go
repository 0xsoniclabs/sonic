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

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/filters"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

var (
	emitter = common.Address{0xe0}
	topic   = common.Hash{0x01}
)

// newHarness serves a head at block 2, whose transaction emitted a log
// without topics; block 3 is stored, not served.
func newHarness(t *testing.T) *tests.Harness {
	t.Helper()
	fake := tests.NewFakeCore()
	fake.AddBlock()
	recipient := common.Address{0x42}
	transaction := tests.NewAccount().Sign(&types.LegacyTx{To: &recipient, Gas: 21000, GasPrice: big.NewInt(1e9)})
	fake.AddBlockWithReceipts(types.Transactions{transaction}, types.Receipts{{
		Status: types.ReceiptStatusSuccessful, CumulativeGasUsed: 21000, Logs: []*types.Log{{Address: emitter}},
	}})
	fake.SetHead(2)
	fake.AddBlock()
	return tests.NewHarness(t, fake, func(h *tests.Harness) {
		filters.Register(h.Dispatcher, h.Environment, h.Core)
	})
}

func TestGetLogs_PassesTheCriteriaWithTheServedHead(t *testing.T) {
	h := newHarness(t)
	block, err := h.Core.Block(2)
	require.NoError(t, err)
	latest := int64(chain.LatestBlockNumber)
	cases := map[string]struct {
		criteria map[string]any
		query    core.LogQuery
	}{
		"nothing": {map[string]any{},
			core.LogQuery{Begin: latest, End: latest, Addresses: []common.Address{}, Head: 2}},
		"a range": {map[string]any{"fromBlock": "0x1", "toBlock": "earliest", "address": emitter, "topics": []any{topic, nil, []any{topic, topic}}},
			core.LogQuery{Begin: 1, End: int64(chain.EarliestBlockNumber), Addresses: []common.Address{emitter}, Topics: [][]common.Hash{{topic}, nil, {topic, topic}}, Head: 2}},
		"a block": {map[string]any{"blockHash": block.Hash, "address": []any{emitter, emitter}, "topics": []any{[]any{topic, nil}}},
			core.LogQuery{BlockHash: &block.Hash, Begin: latest, End: latest, Addresses: []common.Address{emitter, emitter}, Topics: [][]common.Hash{nil}, Head: 2}},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			h.Result(t, "eth_getLogs", test.criteria)
			queries := h.Core.LogQueries()
			require.Equal(t, test.query, queries[len(queries)-1])
		})
	}
}

// A log without topics is answered with an empty list of them, on a copy.
func TestGetLogs_AnswersTopicsAsAList(t *testing.T) {
	h := newHarness(t)
	var logs []map[string]any
	h.Call(t, &logs, "eth_getLogs", map[string]any{"fromBlock": "0x2"})
	require.Len(t, logs, 1)
	require.Equal(t, []any{}, logs[0]["topics"])
	receipts, err := h.Core.Receipts(2)
	require.NoError(t, err)
	require.Nil(t, receipts[0].Logs[0].Topics, "the node's log is unchanged")

	require.Equal(t, "[]", h.Result(t, "eth_getLogs", map[string]any{"fromBlock": "0x1", "toBlock": "0x1"}))
	h.Core.SetMode("Logs", tests.Fail)
	require.Equal(t, tests.ErrInjected.Error(), h.Fail(t, "eth_getLogs", map[string]any{}).Message)
}

func TestGetLogs_WordsCriteriaItCannotDecode(t *testing.T) {
	h := newHarness(t)
	cases := map[string]struct {
		criteria any
		err      string
	}{
		"hash and range":        {map[string]any{"blockHash": common.Hash{}, "fromBlock": "0x1"}, "cannot specify both BlockHash and FromBlock/ToBlock, choose one or the other"},
		"short address":         {map[string]any{"address": "0x01"}, "invalid address: hex has invalid length 1 after decoding; expected 20 for address"},
		"bad address in a list": {map[string]any{"address": []any{"0x01"}}, "invalid address at index 0: hex has invalid length 1 after decoding; expected 20 for address"},
		"non-string address":    {map[string]any{"address": []any{1}}, "non-string address at index 0"},
		"address of no form":    {map[string]any{"address": 1}, "invalid addresses in query"},
		"short topic":           {map[string]any{"topics": []any{"0x01"}}, "hex has invalid length 1 after decoding; expected 32 for topic"},
		"bad topic in a list":   {map[string]any{"topics": []any{[]any{"0x01"}}}, "hex has invalid length 1 after decoding; expected 32 for topic"},
		"topic of no form":      {map[string]any{"topics": []any{1}}, "invalid topic(s)"},
		"non-string topic":      {map[string]any{"topics": []any{[]any{1}}}, "invalid topic(s)"},
		"no object":             {"x", "json: cannot unmarshal string into Go value of type filters.input"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, &dispatch.Error{Code: dispatch.CodeInvalidParams, Message: "invalid argument 0: " + test.err}, h.Fail(t, "eth_getLogs", test.criteria))
		})
	}
}
