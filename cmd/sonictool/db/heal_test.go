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

package db

import (
	"testing"

	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/flushable"
	"github.com/Fantom-foundation/lachesis-base/kvdb/memorydb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestHealGossipDb_RevertsProcessedBundles(t *testing.T) {
	// epoch e starts at block 200*e
	// a complete history retains all bundles since the first one of the chain,
	// a full window retains the bundle.MaxBlockRangeLength blocks before the target
	// in the drawings:
	// - B is the first bundle of the chain
	// - B... is the first bundle of the chain in history pruned from the store
	// - [ ] is the history retained by the store up to block 2048
	// - T is the heal target
	tests := map[string]struct {
		firstBlock uint64 // first block of the retained history
		complete   bool   // whether the retained history starts with the first bundle
		epoch      idx.Epoch
		wantBlock  uint64 // latest history block after healing, 0 if none
		wantErr    string
	}{
		//                  T   [B--------------------]
		//                800   1000               2048
		"complete history, target before first bundle": {
			firstBlock: 1000, complete: true, epoch: 4, wantBlock: 0,
		},
		//     [B-----------T-------------------------]
		//     200          800                    2048
		"complete history, partial window before target": {
			firstBlock: 200, complete: true, epoch: 4, wantBlock: 800,
		},
		// B...[------------T-------------------------]
		//     200          800                    2048
		"incomplete history, partial window before target": {
			firstBlock: 200, complete: false, epoch: 4, wantBlock: 2048,
			wantErr: "cannot revert processed bundles history to epoch 4: " +
				"not enough processed bundles history to roll back to block 800, earliest safe block is 1224",
		},
		//     [B--------------------------T----------]
		//     200                         1600    2048
		"complete history, full window before target": {
			firstBlock: 200, complete: true, epoch: 8, wantBlock: 1600,
		},
		// B...[---------------------------T----------]
		//     200                         1600    2048
		"incomplete history, full window before target": {
			firstBlock: 200, complete: false, epoch: 8, wantBlock: 1600,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			dbs, cfg := newDbsWithBundleHistory(t, test.firstBlock, test.complete)

			store, err := gossip.NewStore(dbs, cfg)
			require.NoError(err)
			wantHash, found := store.GetProcessedBundleHistoryHash(test.wantBlock)
			require.Equal(test.wantBlock != 0, found)
			require.NoError(store.Close())

			_, lastBlock, err := healGossipDb(dbs, cfg, &test.epoch, 10_000)
			if test.wantErr != "" {
				require.EqualError(err, test.wantErr)
			} else {
				require.NoError(err)
				require.Equal(idx.Block(200*test.epoch), lastBlock)
			}

			store, err = gossip.NewStore(dbs, cfg)
			require.NoError(err)
			gotBlock, gotHash := store.GetLatestProcessedBundleHistoryHash()
			require.Equal(test.wantBlock, gotBlock)
			require.Equal(wantHash, gotHash)
			wantBundle := test.complete && test.wantBlock != 0
			require.Equal(wantBundle, store.HasBundleRecentlyBeenProcessed(common.Hash{0x01}))
		})
	}
}

// newDbsWithBundleHistory creates databases of a node with epoch e starting at
// block 200*e for 10 epochs, retaining the bundle history from firstBlock to
// block 2048. A complete history starts with the first bundle of the chain at
// firstBlock; an incomplete one continues the history hash of pruned blocks.
func newDbsWithBundleHistory(
	t *testing.T,
	firstBlock uint64,
	complete bool,
) (kvdb.FlushableDBProducer, gossip.StoreConfig) {
	t.Helper()
	dbs := flushable.NewSyncedPool(memorydb.NewProducer(""), []byte{0})
	cfg := gossip.MemTestStoreConfig(t.TempDir())
	store, err := gossip.NewStore(dbs, cfg)
	require.NoError(t, err)
	if !complete {
		store.SetProcessedBundlesHistoryHash(firstBlock-1, common.Hash{0x01})
	}
	for block := firstBlock; block <= 2*bundle.MaxBlockRangeLength; block++ {
		var bundles map[common.Hash]bundle.PositionInBlock
		if complete && block == firstBlock {
			bundles = map[common.Hash]bundle.PositionInBlock{{0x01}: {}}
		}
		store.AddProcessedBundles(block, bundles)
	}
	for epoch := idx.Epoch(1); epoch <= 10; epoch++ {
		blockState := iblockproc.BlockState{
			LastBlock: iblockproc.BlockCtx{Idx: idx.Block(200 * uint64(epoch))},
		}
		epochState := iblockproc.EpochState{Epoch: epoch}
		store.SetHistoryBlockEpochState(epoch, blockState, epochState)
		store.SetBlockEpochState(blockState, epochState)
	}
	store.FlushBlockEpochState()
	require.NoError(t, store.Close())
	return dbs, cfg
}
