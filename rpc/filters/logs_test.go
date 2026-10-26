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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/filters"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

var (
	first, second, third = common.HexToAddress("0xa1"), common.HexToAddress("0xa2"), common.HexToAddress("0xa3")
	red, blue            = common.Hash{0x0e}, common.Hash{0x0b}
)

// logChain is a chain of three served blocks: the first with logs of the
// first and second address, the second with logs of the first and third,
// the third without logs.
type logChain struct {
	core    *tests.FakeCore
	harness *tests.Harness
	blocks  []common.Hash
	sent    []*types.Transaction
}

func newLogChain(t *testing.T, mutations ...tests.Mutate) logChain {
	t.Helper()
	fake := tests.NewFakeCore(t)
	signer := tests.NewSigner(t)
	transaction := func(nonce uint64) *types.Transaction {
		return signer.Sign(t, &types.LegacyTx{Nonce: nonce, To: &first, Gas: 21000, GasPrice: big.NewInt(1)})
	}
	sent := []*types.Transaction{transaction(0), transaction(1), transaction(2)}
	blocks := []common.Hash{
		fake.AppendBlock(tests.BlockSpec{Transactions: types.Transactions{sent[0]}, Receipts: types.Receipts{{Status: 1, Logs: []*types.Log{
			{Address: first, Topics: []common.Hash{red}}, {Address: second, Topics: []common.Hash{blue}},
		}}}}).Hash,
		fake.AppendBlock(tests.BlockSpec{Transactions: types.Transactions{sent[1], sent[2]}, Receipts: types.Receipts{
			{Status: 1, Logs: []*types.Log{{Address: first, Topics: []common.Hash{red, blue}}}},
			{Status: 1, Logs: []*types.Log{{Address: third}}},
		}}).Hash,
		fake.AppendBlock(tests.BlockSpec{}).Hash,
	}
	fake.PublishHead(3)
	harness := tests.NewHarness(t, fake, func(dispatcher *dispatch.Dispatcher, environment *exec.Environment) {
		filters.Register(dispatcher, environment, fake, "eth")
		filters.Register(dispatcher, environment, fake, "ftm")
	}, mutations...)
	harness.WaitForHead(3)
	return logChain{core: fake, harness: harness, blocks: blocks, sent: sent}
}

func (chain logChain) logs(t *testing.T, criteria map[string]any) []map[string]any {
	t.Helper()
	return tests.Decode[[]map[string]any](t, chain.harness.Result("eth_getLogs", criteria))
}

func limit(set func(*exec.Config)) tests.Mutate {
	return func(configs *tests.Configs) { set(&configs.Exec) }
}

// TestGetLogs_ReadsTheIndexForAnAddressOrATopic holds I11: indexed queries
// through FindLogs, each log with its transaction's position and its
// block's time, each header read once (H2).
func TestGetLogs_ReadsTheIndexForAnAddressOrATopic(t *testing.T) {
	chain := newLogChain(t)
	headers := chain.core.Calls("Header")
	logs := chain.logs(t, map[string]any{"fromBlock": "0x1", "toBlock": "latest", "address": first})
	require.Len(t, logs, 2)
	require.Equal(t, "0x1", logs[0]["blockNumber"])
	require.Equal(t, "0x2", logs[1]["blockNumber"])
	require.Equal(t, "0x0", logs[1]["transactionIndex"])
	require.Equal(t, headers+2, chain.core.Calls("Header"), "one header per block of a log")
	require.Equal(t, chain.headerTime(t, 1), logs[0]["blockTimestamp"])

	third := chain.logs(t, map[string]any{"fromBlock": "0x1", "address": []any{third.Hex()}})
	require.Equal(t, "0x1", third[0]["transactionIndex"], "the position comes from the transaction")
	require.Equal(t, []any{}, third[0]["topics"], "no null topics")

	blue := chain.logs(t, map[string]any{"fromBlock": "0x1", "topics": []any{nil, blue.Hex()}})
	require.Len(t, blue, 1)
	require.Equal(t, "0x2", blue[0]["blockNumber"])
	either := chain.logs(t, map[string]any{"fromBlock": "0x1", "topics": []any{[]any{red.Hex(), common.Hash{0x0c}.Hex()}}})
	require.Len(t, either, 2)
}

func (chain logChain) headerTime(t *testing.T, number uint64) string {
	t.Helper()
	head, err := chain.harness.Environment.Latest(t.Context())
	require.NoError(t, err)
	found, err := chain.harness.Environment.HeaderAt(head, number)
	require.NoError(t, err)
	return "0x" + big.NewInt(found.Time.Unix()).Text(16)
}

// TestGetLogs_WalksTheBlocksWithoutCriteria holds I11 for the walk.
func TestGetLogs_WalksTheBlocksWithoutCriteria(t *testing.T) {
	chain := newLogChain(t)
	require.Len(t, chain.logs(t, map[string]any{"fromBlock": "0x1", "toBlock": "0x2"}), 4)
	require.Len(t, chain.logs(t, map[string]any{"fromBlock": "0x1", "toBlock": "0x99"}), 4, "the range ends at the head")
	require.Empty(t, chain.logs(t, map[string]any{}), "latest is the third block, which has no log")
	require.Empty(t, chain.logs(t, map[string]any{"fromBlock": "0x9"}), "a range above the head is empty")
	require.Len(t, chain.logs(t, map[string]any{"fromBlock": "0x1", "toBlock": "0x2", "topics": []any{nil, nil}}), 1, "two null positions need two topics")

	chain.core.SetMode("Receipts", tests.NotFound)
	require.Empty(t, chain.logs(t, map[string]any{"fromBlock": "0x1", "toBlock": "0x2"}), "the walk ends at the first block it cannot read (F6)")
	chain.core.SetMode("Receipts", tests.Fail)
	require.Equal(t, tests.ErrInjected.Error(), chain.harness.Failure("eth_getLogs", map[string]any{"fromBlock": "0x1"}).Message)
}

func TestGetLogs_ReadsTheBlockOfAHash(t *testing.T) {
	chain := newLogChain(t)
	logs := chain.logs(t, map[string]any{"blockHash": chain.blocks[1], "topics": []any{nil, blue.Hex()}})
	require.Len(t, logs, 1)
	require.Equal(t, chain.blocks[1].Hex(), logs[0]["blockHash"])
	unknown := common.Hash{0xee}
	require.Equal(t, "block with hash "+unknown.Hex()+" not found", chain.harness.Failure("ftm_getLogs", map[string]any{"blockHash": unknown}).Message)
}

// TestGetLogs_KeepsTheLimitsOfTheReference holds V2, V4 and O1.
func TestGetLogs_KeepsTheLimitsOfTheReference(t *testing.T) {
	chain := newLogChain(t, limit(func(config *exec.Config) {
		config.IndexedLogsBlockRangeLimit = 1
		config.UnindexedLogsBlockRangeLimit = 1
		config.LogQueryParameterLimit = 2
		config.LogQueryResultLimit = 2
	}))
	cases := map[string]struct {
		criteria map[string]any
		message  string
	}{
		"an indexed range too wide": {map[string]any{"fromBlock": "0x1", "toBlock": "0x3", "address": first}, "too wide blocks range, the limit is 1"},
		"a walk too wide":           {map[string]any{"fromBlock": "0x1", "toBlock": "0x3"}, "too wide blocks range, the limit is 1"},
		"too many parameters":       {map[string]any{"address": []any{first.Hex(), second.Hex()}, "topics": []any{red.Hex()}}, "too many query parameters, the limit is 2"},
		"too many results":          {map[string]any{"fromBlock": "0x1", "toBlock": "0x2"}, "too many results, consider narrowing your query criteria, the limit is 2"},
	}
	for name, test := range cases {
		require.Equal(t, test.message, chain.harness.Failure("eth_getLogs", test.criteria).Message, name)
	}
	require.Len(t, chain.logs(t, map[string]any{"fromBlock": "0x2", "toBlock": "0x2"}), 2, "one block has no result limit")
}

// TestGetLogs_DecodesTheCriteriaAsTheReference holds V4 and F4.
func TestGetLogs_DecodesTheCriteriaAsTheReference(t *testing.T) {
	chain := newLogChain(t)
	cases := map[string]struct {
		criteria any
		message  string
	}{
		"hash and range":     {map[string]any{"blockHash": chain.blocks[0], "fromBlock": "0x1"}, "cannot specify both BlockHash and FromBlock/ToBlock, choose one or the other"},
		"short address":      {map[string]any{"address": "0x01"}, "invalid address: hex has invalid length 1 after decoding; expected 20 for address"},
		"address in a list":  {map[string]any{"address": []any{first.Hex(), "0x01"}}, "invalid address at index 1: hex has invalid length 1 after decoding; expected 20 for address"},
		"non-string address": {map[string]any{"address": []any{1}}, "non-string address at index 0"},
		"address object":     {map[string]any{"address": map[string]any{}}, "invalid addresses in query"},
		"short topic":        {map[string]any{"topics": []any{"0x01"}}, "hex has invalid length 1 after decoding; expected 32 for topic"},
		"nested topics":      {map[string]any{"topics": []any{[]any{[]any{red.Hex()}}}}, "invalid topic(s)"},
		"topic number":       {map[string]any{"topics": []any{1}}, "invalid topic(s)"},
	}
	for name, test := range cases {
		failure := chain.harness.Failure("eth_getLogs", test.criteria)
		require.Equal(t, dispatch.CodeInvalidParameters, failure.Code, name)
		require.Equal(t, "parameter 0 (criteria): "+test.message, failure.Message, name)
	}
	require.Len(t, chain.logs(t, map[string]any{"fromBlock": "0x1", "topics": []any{[]any{red.Hex(), nil}}}), 3, "a null in a list matches any topic, but a topic is needed in its position")
}
