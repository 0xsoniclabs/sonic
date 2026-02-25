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

package dbutil_test

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/integration"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/stretchr/testify/require"
)

// Run with:
// go test ./utils/dbutil --bench BenchmarkIteratorCosts -run None --benchmem

func BenchmarkIteratorCosts(b *testing.B) {

	key := []byte("test-key")
	value := []byte("test-value")

	db := openFreshGossipDatabase(b)
	require.NoError(b, db.Put(key, value))

	for numIter := 1; numIter <= 1<<20; numIter <<= 1 {
		b.Run(fmt.Sprintf("Iterators=%d", numIter), func(b *testing.B) {
			for b.Loop() {
				iterators := make([]kvdb.Iterator, numIter)
				for i := 0; i < numIter; i++ {
					iterators[i] = db.NewIterator(nil, nil)
				}
				for _, iter := range iterators {
					require.True(b, iter.Next())
					require.Equal(b, key, iter.Key())
					require.Equal(b, value, iter.Value())
				}
				for _, iter := range iterators {
					iter.Release()
				}
			}
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
