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

package eth_test

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/tests"
)

var logger = common.HexToAddress("0xc7")

func newSimulationChain(t *testing.T) stateChain {
	t.Helper()
	accounts := map[common.Address]tests.Account{logger: {Code: []byte{0x60, 0x00, 0x60, 0x00, 0xa0, 0x00}}}
	for address, account := range callAccounts {
		accounts[address] = account
	}
	return newStateChain(t, accounts)
}

func blocks(calls ...[]any) map[string]any {
	stateCalls := make([]any, len(calls))
	for index, blockCalls := range calls {
		stateCalls[index] = map[string]any{"calls": blockCalls}
	}
	return map[string]any{"blockStateCalls": stateCalls}
}

func simulate(t *testing.T, chain stateChain, options map[string]any, parameters ...any) []map[string]any {
	t.Helper()
	raw := chain.harness.Result("eth_simulateV1", append([]any{options}, parameters...)...)
	return tests.Decode[[]map[string]any](t, raw)
}

func TestSimulateV1_RunsCallsInBlocksAboveTheBase(t *testing.T) {
	chain := newSimulationChain(t)
	before := chain.core.States()
	results := simulate(t, chain, blocks([]any{callTo(contract)}, []any{callTo(contract), callTo(logger)}))
	require.Len(t, results, 2)
	require.Equal(t, "0x3", results[0]["number"])
	require.Equal(t, "0x4", results[1]["number"])

	first := results[0]["calls"].([]any)[0].(map[string]any)
	require.Equal(t, common.BigToHash(big.NewInt(8)).Hex(), first["returnData"])
	require.Equal(t, "0x1", first["status"])
	require.Equal(t, []any{}, first["logs"])
	second := results[1]["calls"].([]any)[0].(map[string]any)
	require.Equal(t, common.BigToHash(big.NewInt(9)).Hex(), second["returnData"], "a block sees the state the one before left")

	logged := results[1]["calls"].([]any)[1].(map[string]any)["logs"].([]any)[0].(map[string]any)
	require.Equal(t, results[1]["hash"], logged["blockHash"], "a log takes the hash of its block")
	require.Equal(t, "0x4", logged["blockNumber"])
	require.Equal(t, strings.ToLower(logger.Hex()), strings.ToLower(logged["address"].(string)))

	after := chain.core.States()
	require.Equal(t, before.Opened+1, after.Opened, "one state for all blocks")
	require.Equal(t, after.Opened, after.Released)
}

func TestSimulateV1_NumbersTimesAndFillsTheBlocks(t *testing.T) {
	chain := newSimulationChain(t)
	options := map[string]any{"blockStateCalls": []any{
		map[string]any{"blockOverrides": map[string]any{"blockNumber": "0x5"}},
		map[string]any{"blockOverrides": map[string]any{"timestamp": "0x7fffffff"}},
	}}
	results := simulate(t, chain, options, "latest")
	numbers := []string{}
	for _, block := range results {
		numbers = append(numbers, block["number"].(string))
	}
	require.Equal(t, []string{"0x3", "0x4", "0x5", "0x6"}, numbers, "the gap before block 5 is filled")
	require.Equal(t, "0x7fffffff", results[3]["timestamp"])
	require.Equal(t, results[0]["hash"], results[1]["parentHash"])
}

// TestSimulateV1_BlockHashSeesTheSimulatedBlocks: the walk of GetHashFn is
// kept here, over the simulated blocks and the chain below the base.
func TestSimulateV1_BlockHashSeesTheSimulatedBlocks(t *testing.T) {
	chain := newSimulationChain(t)
	number := func(value int64) any {
		return callTo(blockHasher, "input", common.BigToHash(big.NewInt(value)).Hex())
	}
	results := simulate(t, chain, blocks([]any{}, []any{number(3), number(2), number(1), number(4)}))
	answers := results[1]["calls"].([]any)
	require.Equal(t, results[0]["hash"], answers[0].(map[string]any)["returnData"], "the simulated block before")
	require.Equal(t, chain.blocks[1].Hex(), answers[1].(map[string]any)["returnData"], "the base")
	require.Equal(t, chain.blocks[0].Hex(), answers[2].(map[string]any)["returnData"], "a block of the chain")
	require.Equal(t, common.Hash{}.Hex(), answers[3].(map[string]any)["returnData"], "the block itself")
}

func TestSimulateV1_ReportsAFailedCallInItsResult(t *testing.T) {
	chain := newSimulationChain(t)
	results := simulate(t, chain, blocks([]any{callTo(reverter), callTo(looper, "gas", "0x10000")}))
	reverted := results[0]["calls"].([]any)[0].(map[string]any)
	require.Equal(t, "0x0", reverted["status"])
	require.Equal(t, map[string]any{"message": "execution reverted", "code": float64(-32000), "data": "0xdeadbeef"}, reverted["error"])
	looped := results[0]["calls"].([]any)[1].(map[string]any)
	require.Equal(t, map[string]any{"message": "out of gas", "code": float64(-32015)}, looped["error"], "go-ethereum's interpreter runs a simulation")
}

func TestSimulateV1_TracesTransfers(t *testing.T) {
	chain := newSimulationChain(t)
	options := blocks([]any{callTo(readAccount, "value", "0x10")})
	options["traceTransfers"] = true
	results := simulate(t, chain, options)
	logs := results[0]["calls"].([]any)[0].(map[string]any)["logs"].([]any)
	require.Len(t, logs, 1)
	transfer := logs[0].(map[string]any)
	require.Equal(t, "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", strings.ToLower(transfer["address"].(string)))
	require.Equal(t, common.BytesToHash(holder.Bytes()).Hex(), transfer["topics"].([]any)[1])
	require.Equal(t, common.BytesToHash(readAccount.Bytes()).Hex(), transfer["topics"].([]any)[2])
}

func TestSimulateV1_AnswersFullTransactionsWithTheirSender(t *testing.T) {
	chain := newSimulationChain(t)
	options := blocks([]any{callTo(contract)})
	options["returnFullTransactions"] = true
	results := simulate(t, chain, options)
	transaction := results[0]["transactions"].([]any)[0].(map[string]any)
	require.Equal(t, strings.ToLower(holder.Hex()), strings.ToLower(transaction["from"].(string)))
}

func TestSimulateV1_AppliesTheStateOverridesOfEachBlock(t *testing.T) {
	chain := newSimulationChain(t)
	options := map[string]any{"blockStateCalls": []any{
		map[string]any{"calls": []any{callTo(contract)}},
		map[string]any{
			"stateOverrides": map[string]any{contract.Hex(): map[string]any{"stateDiff": map[string]any{common.Hash{}.Hex(): common.BigToHash(big.NewInt(99)).Hex()}}},
			"calls":          []any{callTo(contract)},
		},
	}}
	results := simulate(t, chain, options)
	require.Equal(t, common.BigToHash(big.NewInt(100)).Hex(), results[1]["calls"].([]any)[0].(map[string]any)["returnData"])
}

// TestSimulateV1_RefusesWithTheCodesOfTheSpecification holds ADR6 for the
// block count, and the codes of eth_simulateV1.
func TestSimulateV1_RefusesWithTheCodesOfTheSpecification(t *testing.T) {
	chain := newSimulationChain(t)
	many := func(count int) []any {
		calls := make([]any, count)
		for index := range calls {
			calls[index] = callTo(readAccount)
		}
		return calls
	}
	tooManyBlocks := make([][]any, 257)
	cases := map[string]struct {
		options any
		code    int
		message string
	}{
		"no blocks":            {map[string]any{"blockStateCalls": []any{}}, -32602, "empty input"},
		"too many blocks":      {blocks(tooManyBlocks...), -38026, "too many blocks"},
		"too many in a block":  {blocks(many(5001)), -38026, "too many calls in block: 5001 > 5000"},
		"too many calls":       {blocks(many(4000), many(4000), many(4000)), -38026, "too many calls: 12000 > 10000"},
		"too far above":        {map[string]any{"blockStateCalls": []any{map[string]any{"blockOverrides": map[string]any{"blockNumber": "0x103"}}}}, -38026, "too many blocks"},
		"numbers out of order": {map[string]any{"blockStateCalls": []any{map[string]any{"blockOverrides": map[string]any{"blockNumber": "0x2"}}}}, -38020, "block numbers must be in order: 2 <= 2"},
		"times out of order":   {map[string]any{"blockStateCalls": []any{map[string]any{"blockOverrides": map[string]any{"timestamp": "0x1"}}}}, -38021, fmt.Sprintf("block timestamps must be in order: 1 <= %d", 1_700_000_002)},
		"gas above the block":  {blocks([]any{callTo(readAccount, "gas", "0x1c9c381")}), -38015, "block gas limit reached: 30000001 >= 30000000"},
		"a nonce too high":     {map[string]any{"validation": true, "blockStateCalls": []any{map[string]any{"calls": []any{callTo(readAccount, "nonce", "0x5")}}}}, -38011, "err: nonce too high: address " + holder.Hex() + ", tx: 5 state: 0 (supplied gas 30000000)"},
	}
	for name, test := range cases {
		failure := chain.harness.Failure("eth_simulateV1", test.options)
		require.Equal(t, test.code, failure.Code, name)
		require.Equal(t, test.message, failure.Message, name)
	}
	require.Equal(t, "header not found", chain.harness.Failure("eth_simulateV1", blocks([]any{}), "0x9").Message)
}
