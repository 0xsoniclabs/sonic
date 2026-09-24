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

package integration

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// BenchmarkStore_HasBundleRecentlyBeenProcessed measures the lookup the tx
// pool performs per bundle-only transaction, on the DB stack of a node holding
// a full window of processed bundles.
func BenchmarkStore_HasBundleRecentlyBeenProcessed(b *testing.B) {
	const bundlesPerBlock = 10

	for _, cacheMiB := range []uint64{1, 480} {
		dbs, err := GetDbProducer(b.TempDir(), DBCacheConfig{
			Cache:   cacheMiB << 20,
			Fdlimit: 100,
		})
		require.NoError(b, err)
		store, err := gossip.NewStore(dbs, gossip.DefaultStoreConfig(cachescale.Identity))
		require.NoError(b, err)

		var processed []common.Hash
		for block := range bundle.MaxBlockRangeLength {
			bundles := map[common.Hash]bundle.PositionInBlock{}
			for range bundlesPerBlock {
				hash := randomHash()
				bundles[hash] = bundle.PositionInBlock{}
				processed = append(processed, hash)
			}
			store.AddProcessedBundles(block, bundles)
		}
		require.NoError(b, dbs.Flush([]byte("bench"))) // < read from pebble, not from memory
		processed = processed[bundlesPerBlock:]        // < the first block got pruned

		b.Run(fmt.Sprintf("cache=%dMiB/processed", cacheMiB), func(b *testing.B) {
			for i := range b.N {
				if !store.HasBundleRecentlyBeenProcessed(processed[i%len(processed)]) {
					b.Fatal("processed bundle not found")
				}
			}
		})
		b.Run(fmt.Sprintf("cache=%dMiB/pending", cacheMiB), func(b *testing.B) {
			pending := make([]common.Hash, 1024)
			for i := range pending {
				pending[i] = randomHash()
			}
			b.ResetTimer()
			for i := range b.N {
				if store.HasBundleRecentlyBeenProcessed(pending[i%len(pending)]) {
					b.Fatal("pending bundle reported as processed")
				}
			}
		})
		require.NoError(b, store.Close())
	}
}

func randomHash() common.Hash {
	var hash common.Hash
	for i := range hash {
		hash[i] = byte(rand.Uint32())
	}
	return hash
}
