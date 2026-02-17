// Copyright 2025 Sonic Operations Ltd
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

package topicsdb_test

import (
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/integration"
	"github.com/0xsoniclabs/sonic/topicsdb"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/memorydb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestWithLeapJoin_IntegrationTest_FindLogs(t *testing.T) {

	// Test against an in-memory and a real DB instance.
	dbs := map[string]kvdb.Store{
		"memory":    memorydb.New(),
		"gossip-db": openFreshGossipDatabase(t),
	}

	algorithms := map[string]func(kvdb.Store) topicsdb.Index{
		"leap-join": topicsdb.NewWithLeapJoin,
	}

	logs := []*types.Log{}
	for b := range uint64(10) {
		for i := range 10 {
			for j := range 10 {
				for k := range 10 {
					logs = append(logs, &types.Log{
						BlockNumber: b,
						Address:     common.Address{byte(i)},
						Topics:      []common.Hash{{byte(j)}, {byte(k)}},
						Data:        []byte{byte(i), byte(j), byte(k)},
						// The TxHash is needed to give each log entry a unique key
						// in the index.
						TxHash: common.Hash{byte(b), byte(i), byte(j), byte(k)},
					})
				}
			}
		}
	}

	patterns := map[string]struct {
		addresses       []common.Address
		topics          [][]common.Hash
		expectedResults int
	}{
		"one address": {
			addresses:       []common.Address{{5}},
			expectedResults: 100,
		},
		"two addresses": {
			addresses:       []common.Address{{3}, {7}},
			expectedResults: 200,
		},
		"one address, one topic": {
			addresses:       []common.Address{{2}},
			topics:          [][]common.Hash{{{4}}},
			expectedResults: 10,
		},
		"one address, two topics": {
			addresses:       []common.Address{{1}},
			topics:          [][]common.Hash{{{2}}, {{3}}},
			expectedResults: 1,
		},
		"two addresses, one topic": {
			addresses:       []common.Address{{0}, {9}},
			topics:          [][]common.Hash{{{5}}},
			expectedResults: 20,
		},
		"two addresses, two topics": {
			addresses:       []common.Address{{4}, {6}},
			topics:          [][]common.Hash{{{7}}, {{8}}},
			expectedResults: 2,
		},
		"one address, two options for topic 1": {
			addresses:       []common.Address{{8}},
			topics:          [][]common.Hash{{{1}, {2}}, {{3}}},
			expectedResults: 2,
		},
		"one address, two options for topic 2": {
			addresses:       []common.Address{{7}},
			topics:          [][]common.Hash{{{4}}, {{5}, {6}}},
			expectedResults: 2,
		},
		"one address, two options for both topics": {
			addresses:       []common.Address{{5}},
			topics:          [][]common.Hash{{{7}, {8}}, {{2}, {3}}},
			expectedResults: 4,
		},
		"one address, two options for first topic, arbitrary second topic": {
			addresses:       []common.Address{{3}},
			topics:          [][]common.Hash{{{0}, {1}}, {}},
			expectedResults: 20,
		},
		"arbitrary address, one first topic, arbitrary second topic": {
			addresses:       []common.Address{},
			topics:          [][]common.Hash{{{9}}, {}},
			expectedResults: 100,
		},
		"arbitrary address, arbitrary first topic, one second topic": {
			addresses:       []common.Address{},
			topics:          [][]common.Hash{{}, {{0}}},
			expectedResults: 100,
		},
		"arbitrary address, two options for first topic, arbitrary second topic": {
			addresses:       []common.Address{},
			topics:          [][]common.Hash{{{2}, {3}}, {}},
			expectedResults: 200,
		},
		"arbitrary address, arbitrary first topic, two options for second topic": {
			addresses:       []common.Address{},
			topics:          [][]common.Hash{{}, {{4}, {5}}},
			expectedResults: 200,
		},
		"arbitrary address, two options for both topics": {
			addresses:       []common.Address{},
			topics:          [][]common.Hash{{{6}, {7}}, {{8}, {9}}},
			expectedResults: 40,
		},
		"non-existing address": {
			addresses:       []common.Address{{99}},
			expectedResults: 0,
		},
		"non-existing topic 1": {
			addresses:       []common.Address{},
			topics:          [][]common.Hash{{{99}}},
			expectedResults: 0,
		},
		"non-existing topic 2": {
			addresses:       []common.Address{},
			topics:          [][]common.Hash{{}, {{99}}},
			expectedResults: 0,
		},
		"requesting more topics than exist in logs": {
			addresses:       []common.Address{},
			topics:          [][]common.Hash{{}, {}, {{1}}},
			expectedResults: 0,
		},
	}

	for dbName, db := range dbs {
		t.Run(dbName, func(t *testing.T) {
			for algName, alg := range algorithms {
				t.Run(algName, func(t *testing.T) {
					index := alg(db)

					// Push logs into the index.
					require.NoError(t, index.Push(logs...))

					for ptnName, pattern := range patterns {
						t.Run(ptnName, func(t *testing.T) {

							// Merge address and topic patterns.
							p := [][]common.Hash{}
							addressPatterns := []common.Hash{}
							for _, addr := range pattern.addresses {
								addressPatterns = append(
									addressPatterns,
									common.BytesToHash(addr.Bytes()),
								)
							}
							p = append(p, addressPatterns)
							p = append(p, pattern.topics...)

							// Pad pattern to 3 elements (address + 2 topics)
							// for simplicity.
							for len(p) < 3 {
								p = append(p, []common.Hash{})
							}

							ranges := map[string]struct {
								from, to int
							}{
								"empty range":     {from: 5, to: 4},
								"block zero":      {from: 0, to: 0},
								"single block":    {from: 2, to: 2},
								"full range":      {from: 0, to: 9},
								"exceeding range": {from: 0, to: 20},
							}

							for rangeName, r := range ranges {
								t.Run(rangeName, func(t *testing.T) {
									from := idx.Block(r.from)
									to := idx.Block(r.to)

									if to == 0 && algName == "thread-pool" {
										// The upper bound of 0 is used to
										// indicate "no upper bound" in the
										// thread pool algorithm.
										t.Skip("thread pool algorithm does not support an upper bound of 0")
									}

									numBlocks := min(max(r.to-r.from+1, 0), 10)
									expectedResults := pattern.expectedResults * numBlocks

									// run the join algorithm
									got, err := index.FindInBlocks(
										t.Context(), from, to, p,
									)
									require.NoError(t, err)
									require.Equal(t, expectedResults, len(got))

									// verify results
									want := filter(logs, from, to, p)
									require.Equal(t, expectedResults, len(want))
									require.ElementsMatch(t, want, got)
								})
							}
						})
					}
				})
			}
		})
	}
}

// filter filters out logs that match the given pattern. This is used to verify
// the results of the join algorithm.
func filter(
	logs []*types.Log,
	from, to idx.Block,
	pattern [][]common.Hash,
) []*types.Log {
	return slices.DeleteFunc(slices.Clone(logs), func(log *types.Log) bool {
		if log.BlockNumber < uint64(from) || log.BlockNumber > uint64(to) {
			return true
		}
		return !matchesPattern(log, pattern)
	})
}

// matchesPattern checks if a log matches the given pattern. It is a building
// block for the filter function.
func matchesPattern(
	log *types.Log,
	pattern [][]common.Hash,
) bool {
	// There must be the right number of topics in the log.
	if len(log.Topics)+1 != len(pattern) {
		return false
	}

	// Check the address.
	if len(pattern) == 0 {
		return true
	}
	if len(pattern[0]) > 0 {
		addresses := make([]common.Address, len(pattern[0]))
		for i, h := range pattern[0] {
			addresses[i] = common.BytesToAddress(h.Bytes())
		}
		if !slices.Contains(addresses, log.Address) {
			return false
		}
	}

	// Check the remaining topics.
	for i, sub := range pattern[1:] {
		if len(sub) == 0 {
			continue
		}
		if !slices.Contains(sub, log.Topics[i]) {
			return false
		}
	}
	return true
}

func TestMatchesPattern_Examples(t *testing.T) {
	tests := map[string]struct {
		log     *types.Log
		pattern [][]common.Hash
		want    bool
	}{
		"matches address and topics": {
			log: &types.Log{
				Address: common.Address{1},
				Topics:  []common.Hash{{2}, {3}},
			},
			pattern: [][]common.Hash{
				{{12: 1}}, // 20-byte address padded to 32 bytes
				{{2}},
				{{3}},
			},
			want: true,
		},
		"matches address, arbitrary topics": {
			log: &types.Log{
				Address: common.Address{1},
				Topics:  []common.Hash{{2}, {3}},
			},
			pattern: [][]common.Hash{
				{{12: 1}},
				{},
				{},
			},
			want: true,
		},
		"matches arbitrary address, matches topics": {
			log: &types.Log{
				Address: common.Address{1},
				Topics:  []common.Hash{{2}, {3}},
			},
			pattern: [][]common.Hash{
				{},
				{{2}},
				{{3}},
			},
			want: true,
		},
		"matches arbitrary address, matches one topic, arbitrary other topic": {
			log: &types.Log{
				Address: common.Address{1},
				Topics:  []common.Hash{{2}, {3}},
			},
			pattern: [][]common.Hash{
				{},
				{{2}},
				{},
			},
			want: true,
		},
		"does not match address": {
			log: &types.Log{
				Address: common.Address{1},
				Topics:  []common.Hash{{2}, {3}},
			},
			pattern: [][]common.Hash{
				{{99}},
				{{2}},
				{{3}},
			},
			want: false,
		},
		"does not match topic": {
			log: &types.Log{
				Address: common.Address{1},
				Topics:  []common.Hash{{2}, {3}},
			},
			pattern: [][]common.Hash{
				{{1}},
				{{99}},
				{{3}},
			},
			want: false,
		},
		"too few topics": {
			log: &types.Log{
				Address: common.Address{1},
				Topics:  []common.Hash{{2}, {3}},
			},
			pattern: [][]common.Hash{
				{{12: 1}}, // 20-byte address padded to 32 bytes
				{{2}},
				{{3}},
				{{4}},
			},
			want: false,
		},
		"too many topics": {
			log: &types.Log{
				Address: common.Address{1},
				Topics:  []common.Hash{{2}, {3}},
			},
			pattern: [][]common.Hash{
				{{12: 1}}, // 20-byte address padded to 32 bytes
				{{2}},
			},
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := matchesPattern(tc.log, tc.pattern)
			require.Equal(t, tc.want, got)
		})
	}
}

func openFreshGossipDatabase(t testing.TB) kvdb.Store {
	require := require.New(t)
	// Open a real, pebble based DB.
	producer, err := integration.GetDbProducer(
		t.TempDir(),
		integration.DBCacheConfig{},
	)
	require.NoError(err)
	t.Cleanup(func() {
		require.NoError(producer.Close())
	})

	db, err := producer.OpenDB("gossip")
	require.NoError(err)
	t.Cleanup(func() {
		require.NoError(db.Close())
	})

	return db
}
