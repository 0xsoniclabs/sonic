package topicsdb_test

import (
	"encoding/binary"
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/integration"
	"github.com/0xsoniclabs/sonic/topicsdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/memorydb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestWithLeapJoin_IntegrationTest_FindLogs(t *testing.T) {

	// Test against an in-memory and a real DB instance.
	tests := map[string]kvdb.Store{
		"memory":    memorydb.New(),
		"gossip-db": openFreshGossipDatabase(t),
	}

	logs := []*types.Log{}
	for i := range 10 {
		for j := range 10 {
			for k := range 10 {
				logs = append(logs, &types.Log{
					Address: common.Address{byte(i)},
					Topics:  []common.Hash{{byte(j)}, {byte(k)}},
					Data:    []byte{byte(i), byte(j), byte(k)},
					// The TxHash is needed to give each log entry a unique key
					// in the index.
					TxHash: common.Hash{byte(i), byte(j), byte(k)},
				})
			}
		}
	}

	// TODO: also filter by block numbers

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

	for name, db := range tests {
		t.Run(name, func(t *testing.T) {
			index := topicsdb.NewWithLeapJoin(db)

			// Push logs into the index.
			require.NoError(t, index.Push(logs...))

			for ptnName, pattern := range patterns {
				t.Run(ptnName, func(t *testing.T) {

					// Merge address and topic patterns.
					p := [][]common.Hash{}
					addressPatterns := []common.Hash{}
					for _, addr := range pattern.addresses {
						addressPatterns = append(addressPatterns, common.BytesToHash(addr.Bytes()))
					}
					p = append(p, addressPatterns)
					p = append(p, pattern.topics...)

					// run the Leap Join search
					got, err := index.FindInBlocks(
						t.Context(), 0, math.MaxUint64, p,
					)
					require.NoError(t, err)
					require.Equal(t, pattern.expectedResults, len(got))

					// verify results
					want := filter(logs, p)
					require.Equal(t, pattern.expectedResults, len(want))
					require.ElementsMatch(t, want, got)
				})
			}
		})
	}
}

func filter(
	logs []*types.Log,
	pattern [][]common.Hash,
) []*types.Log {
	filtered := []*types.Log{}
	for _, log := range logs {
		if matchesPattern(log, pattern) {
			filtered = append(filtered, log)
		}
	}
	return filtered
}

func matchesPattern(
	log *types.Log,
	pattern [][]common.Hash,
) bool {
	// There must be enough topics in the log
	if len(log.Topics)+1 < len(pattern) {
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

/*
func TestWithLeapJoin_CanInteractWithRealDB(t *testing.T) {
	require := require.New(t)

	db := openFreshGossipDatabase(t)

	//index := NewWithLeapJoin(db)
}
*/

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

func benchmark_LargeQueryProcessing(
	b *testing.B,
	numAlternatives int,
) {
	require := require.New(b)

	// Build a topics DB and populate it with logs.
	db := openFreshGossipDatabase(b)
	index := topicsdb.NewWithLeapJoin(db)
	addressPattern := []common.Hash{}
	for i := range numAlternatives {
		data := [32]byte{}
		binary.BigEndian.PutUint32(data[28:], uint32(i))
		/*
			log := &types.Log{
				Address: common.Address(data[12:]),
				TxHash:  common.Hash(data),
			}
		*/
		addressPattern = append(addressPattern, common.BytesToHash(data[:]))
		//require.NoError(index.Push(log))
	}

	filterPattern := [][]common.Hash{addressPattern}
	for b.Loop() {
		/*res*/ _, err := index.FindInBlocks(b.Context(), 0, math.MaxInt64, filterPattern)
		require.NoError(err)
		//require.Equal(numAlternatives, len(res))
	}
}

func Benchmark_LargeQueryProcessing(b *testing.B) {
	for i := 1; i <= 1<<16; i *= 2 {
		b.Run(fmt.Sprintf("query_size=%d", i), func(b *testing.B) {
			benchmark_LargeQueryProcessing(b, i)
		})
	}
}
