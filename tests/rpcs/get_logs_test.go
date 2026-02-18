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

package rpcs

import (
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/indexed_logs"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestGetLogFilters(t *testing.T) {
	const N = 5

	// This test starts a network and installs contracts producing a systematic
	// list of log messages that are then retrieved using the `eth_getLogs` RPC
	// method. The test verifies that the retrieved logs match the expected logs,
	// ensuring that the log retrieval functionality works correctly.

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{})

	// Deploy multiple instances of the `IndexedLogs` contract, which emits a
	// Cartesian product of logs to be sliced by filter tests.
	sources := []*indexed_logs.IndexedLogs{}
	sourceAddrs := []common.Address{}
	for range N {
		source, receipt, err := tests.DeployContract(net, indexed_logs.DeployIndexedLogs)
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
		sources = append(sources, source)
		sourceAddrs = append(sourceAddrs, receipt.ContractAddress)
	}

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	startBlock, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	// Generate logs to be filtered for.
	blockHashes := []common.Hash{}
	for _, source := range sources {
		receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
			return source.EmitCartesianProduct(opts, big.NewInt(N))
		})
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
		blockHashes = append(blockHashes, receipt.BlockHash)
	}

	endBlock, err := client.BlockNumber(t.Context())
	require.NoError(t, err)
	require.GreaterOrEqual(t, endBlock, startBlock+N)

	// Retrieve all logs using the `eth_getLogs` RPC method.
	createdLogs, err := client.FilterLogs(t.Context(), ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(startBlock)),
		ToBlock:   big.NewInt(int64(endBlock)),
	})
	require.NoError(t, err)

	// There should be N^4 + N^3 + N^2 + N + N logs in total, as each contract
	// emits a Cartesian product of logs based on the input parameter N.
	require.EqualValues(t, N*N*N*N+N*N*N+N*N+N+N, len(createdLogs))

	// Get all logs, including those created during genesis, as the full set of
	// logs test filters will be applied on.
	allLogs, err := client.FilterLogs(t.Context(), ethereum.FilterQuery{})
	require.NoError(t, err)
	for _, log := range createdLogs {
		require.Contains(t, allLogs, log)
	}

	// Retrieve the ABI of the contract to identify the log event signatures.
	abi, err := indexed_logs.IndexedLogsMetaData.GetAbi()
	require.NoError(t, err)
	eventIDs := []common.Hash{
		abi.Events["Log1"].ID,
		abi.Events["Log2"].ID,
		abi.Events["Log3"].ID,
		abi.Events["Log4"].ID,
	}

	toTopic := func(i int) common.Hash {
		return common.BigToHash(big.NewInt(int64(i)))
	}

	tests := map[string]ethereum.FilterQuery{
		"all logs": {
			// Default accepts everything.
		},
		"no logs": {
			FromBlock: big.NewInt(int64(endBlock + 1)),
		},

		// Test filtering by block hash.

		"logs from block adding the first set of logs": {
			BlockHash: &blockHashes[0],
		},
		"logs from block adding the second set of logs": {
			BlockHash: &blockHashes[1],
		},

		// Test block ranges.

		"logs from start block only": {
			FromBlock: big.NewInt(int64(startBlock)),
			ToBlock:   big.NewInt(int64(startBlock)),
		},
		"logs from start block +1 only": {
			FromBlock: big.NewInt(int64(startBlock + 1)),
			ToBlock:   big.NewInt(int64(startBlock + 1)),
		},
		"logs from start block to start block +1": {
			FromBlock: big.NewInt(int64(startBlock)),
			ToBlock:   big.NewInt(int64(startBlock + 1)),
		},
		"logs from start block +1 to end block": {
			FromBlock: big.NewInt(int64(startBlock + 1)),
			ToBlock:   big.NewInt(int64(endBlock)),
		},
		"logs from start block to end block": {
			FromBlock: big.NewInt(int64(startBlock)),
			ToBlock:   big.NewInt(int64(endBlock)),
		},
		"logs from start block to middle block": {
			FromBlock: big.NewInt(int64(startBlock)),
			ToBlock:   big.NewInt(int64((startBlock + endBlock) / 2)),
		},
		"logs from middle block to end block": {
			FromBlock: big.NewInt(int64((startBlock + endBlock) / 2)),
			ToBlock:   big.NewInt(int64(endBlock)),
		},

		// Test filtering by contract address.

		"logs from first contract": {
			Addresses: []common.Address{sourceAddrs[0]},
		},
		"logs from second contract": {
			Addresses: []common.Address{sourceAddrs[1]},
		},
		"logs from third and fourth contract": {
			Addresses: []common.Address{sourceAddrs[2], sourceAddrs[3]},
		},
		"logs from first, second and fourth contract": {
			Addresses: []common.Address{sourceAddrs[0], sourceAddrs[1], sourceAddrs[3]},
		},
		"logs from all contracts": {
			Addresses: sourceAddrs,
		},
		"logs from non-existing contract": {
			Addresses: []common.Address{{1, 2, 3, 4, 5, 6, 7, 8}},
		},
		// Test filtering by log topics.

		"logs produced by event Log0": {
			Topics: [][]common.Hash{{eventIDs[0]}},
		},
		"logs produced by event Log1": {
			Topics: [][]common.Hash{{eventIDs[1]}},
		},
		"logs produced by event Log0 or Log1": {
			Topics: [][]common.Hash{{eventIDs[0], eventIDs[1]}},
		},
		"logs produced by event Log0, Log1 or Log2": {
			Topics: [][]common.Hash{{eventIDs[0], eventIDs[1], eventIDs[2]}},
		},
		"logs produced by event Log0, Log1, Log2 or Log3": {
			Topics: [][]common.Hash{eventIDs},
		},
		"log produced by non-existing event": {
			Topics: [][]common.Hash{{common.Hash{1, 2, 3, 4, 5, 6, 7, 8}}},
		},

		// Testing filtering by indexed log parameters.

		"logs with first indexed parameter equal to 0": {
			Topics: [][]common.Hash{nil, {toTopic(0)}}, // nil => ignore Log type
		},
		"logs with first indexed parameter equal to 1": {
			Topics: [][]common.Hash{nil, {toTopic(1)}},
		},
		"logs with first indexed parameter equal to 1 or 2": {
			Topics: [][]common.Hash{nil, {toTopic(1), toTopic(2)}},
		},
		"log with first indexed parameter equal to non-existing value": {
			Topics: [][]common.Hash{nil, {toTopic(100)}},
		},

		"logs with second indexed parameter equal to 0": {
			Topics: [][]common.Hash{nil, nil, {toTopic(0)}}, // nil => ignore Log type and first indexed parameter
		},
		"logs with second indexed parameter equal to 1": {
			Topics: [][]common.Hash{nil, nil, {toTopic(1)}},
		},
		"logs with second indexed parameter equal to 1 or 2": {
			Topics: [][]common.Hash{nil, nil, {toTopic(1), toTopic(2)}},
		},
		"log with second indexed parameter equal to non-existing value": {
			Topics: [][]common.Hash{nil, nil, {toTopic(100)}},
		},

		"logs with third indexed parameter equal to 0": {
			Topics: [][]common.Hash{nil, nil, nil, {toTopic(0)}}, // nil => ignore Log type and first two indexed parameters
		},
		"logs with third indexed parameter equal to 1": {
			Topics: [][]common.Hash{nil, nil, nil, {toTopic(1)}},
		},
		"logs with third indexed parameter equal to 1 or 2": {
			Topics: [][]common.Hash{nil, nil, nil, {toTopic(1), toTopic(2)}},
		},
		"log with third indexed parameter equal to non-existing value": {
			Topics: [][]common.Hash{nil, nil, nil, {toTopic(100)}},
		},

		"logs with first indexed parameter equal to 1 and second indexed parameter equal to 2": {
			Topics: [][]common.Hash{nil, {toTopic(1)}, {toTopic(2)}},
		},
		"logs with first indexed parameter equal to 0 or 2 and second indexed parameter equal to 1 or 3": {
			Topics: [][]common.Hash{nil, {toTopic(0), toTopic(2)}, {toTopic(1), toTopic(3)}},
		},

		"full criteria combination": {
			FromBlock: big.NewInt(int64(startBlock + 2)),
			ToBlock:   big.NewInt(int64(startBlock + 4)),
			Topics: [][]common.Hash{
				{eventIDs[2], eventIDs[3]},
				{toTopic(1), toTopic(2)},
				nil,
				{toTopic(1), toTopic(3), toTopic(100)},
			},
		},

		// Too long topics list (there are only 3 index parameters + the source
		// event ID, so the 5th topic group is invalid, leading to an empty result).
		"logs with too long topics list": {
			Topics: [][]common.Hash{nil, nil, nil, nil, {toTopic(0)}},
		},
	}

	numFull := 0
	numEmpty := 0
	for name, query := range tests {
		t.Run(name, func(t *testing.T) {
			logs, err := client.FilterLogs(t.Context(), query)
			require.NoError(t, err)

			// Verify that the retrieved logs match the expected logs.
			expectedLogs := filterLogs(allLogs, query)
			require.EqualValues(t, expectedLogs, logs)

			if len(logs) == len(allLogs) {
				numFull++
			}
			if len(logs) == 0 {
				numEmpty++
			}
		})
	}

	// Smoke-test that the reference filter implementation is not broken.
	require.Equal(t, 1, numFull, "exactly one test case should return the full set of logs")
	require.Less(t, numEmpty, len(tests)-1, "at least one test case should return a true subset of logs")
}

func filterLogs(logs []types.Log, query ethereum.FilterQuery) []types.Log {
	filter := filter{query}
	return slices.DeleteFunc(slices.Clone(logs), func(log types.Log) bool {
		return !filter.matches(log)
	})
}

type filter struct {
	ethereum.FilterQuery
}

func (f *filter) matches(log types.Log) bool {
	if f.BlockHash != nil && log.BlockHash != *f.BlockHash {
		return false
	}
	if f.FromBlock != nil && log.BlockNumber < f.FromBlock.Uint64() {
		return false
	}
	if f.ToBlock != nil && log.BlockNumber > f.ToBlock.Uint64() {
		return false
	}
	// Check address match, if any addresses are specified in the query.
	if len(f.Addresses) > 0 {
		match := false
		for _, addr := range f.Addresses {
			if log.Address == addr {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	// Check topic match, if any topics are specified in the query.
	for i, topicGroup := range f.Topics {
		if len(topicGroup) == 0 {
			continue
		}
		if i >= len(log.Topics) {
			return false
		}
		match := false
		for _, topic := range topicGroup {
			if log.Topics[i] == topic {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	return true
}
