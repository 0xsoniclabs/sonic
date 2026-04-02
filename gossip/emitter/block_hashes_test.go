// Copyright 2024 The Sonic Authors
// This file is part of the Sonic library.
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

package emitter

import (
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/gossip/emitter/config"
	"github.com/0xsoniclabs/sonic/inter"
)

func TestAddBlockHashes_NothingNewToReport(t *testing.T) {
	ctrl := gomock.NewController(t)
	world := NewMockExternal(ctrl)

	em := &Emitter{
		config: config.Config{},
		world:  World{External: world},
	}

	// latestBlock == 0, start == 1, so start > latestBlock
	world.EXPECT().GetLatestBlockIndex().Return(idx.Block(0))

	event := &inter.MutableEventPayload{}
	em.addBlockHashes(event)

	// No block hashes should be set
	require.Empty(t, event.BlockHashes().Hashes)
}

func TestAddBlockHashes_CollectsConsecutiveBlocks(t *testing.T) {
	ctrl := gomock.NewController(t)
	world := NewMockExternal(ctrl)

	em := &Emitter{
		config: config.Config{},
		world:  World{External: world},
	}

	world.EXPECT().GetLatestBlockIndex().Return(idx.Block(3))

	block1 := inter.NewBlockBuilder().
		WithNumber(1).
		WithEpoch(idx.Epoch(1)).
		WithBaseFee(big.NewInt(0)).
		Build()
	block2 := inter.NewBlockBuilder().
		WithNumber(2).
		WithEpoch(idx.Epoch(1)).
		WithBaseFee(big.NewInt(0)).
		Build()
	block3 := inter.NewBlockBuilder().
		WithNumber(3).
		WithEpoch(idx.Epoch(1)).
		WithBaseFee(big.NewInt(0)).
		Build()

	world.EXPECT().GetBlock(idx.Block(1)).Return(block1)
	world.EXPECT().GetBlock(idx.Block(2)).Return(block2)
	world.EXPECT().GetBlock(idx.Block(3)).Return(block3)

	event := &inter.MutableEventPayload{}
	em.addBlockHashes(event)

	bh := event.BlockHashes()
	require.Equal(t, idx.Block(1), bh.Start)
	require.Equal(t, idx.Epoch(1), bh.Epoch)
	require.Len(t, bh.Hashes, 3)
	require.Equal(t, hash.Hash(block1.Hash()), bh.Hashes[0])
	require.Equal(t, hash.Hash(block2.Hash()), bh.Hashes[1])
	require.Equal(t, hash.Hash(block3.Hash()), bh.Hashes[2])
}

func TestAddBlockHashes_StopsAtEpochBoundary(t *testing.T) {
	ctrl := gomock.NewController(t)
	world := NewMockExternal(ctrl)

	em := &Emitter{
		config: config.Config{},
		world:  World{External: world},
	}

	world.EXPECT().GetLatestBlockIndex().Return(idx.Block(3))

	block1 := inter.NewBlockBuilder().
		WithNumber(1).
		WithEpoch(idx.Epoch(1)).
		WithBaseFee(big.NewInt(0)).
		Build()
	block2 := inter.NewBlockBuilder().
		WithNumber(2).
		WithEpoch(idx.Epoch(2)). // different epoch
		WithBaseFee(big.NewInt(0)).
		Build()

	world.EXPECT().GetBlock(idx.Block(1)).Return(block1)
	world.EXPECT().GetBlock(idx.Block(2)).Return(block2)
	// block 3 should not be requested since we stopped at epoch boundary

	event := &inter.MutableEventPayload{}
	em.addBlockHashes(event)

	bh := event.BlockHashes()
	require.Equal(t, idx.Block(1), bh.Start)
	require.Equal(t, idx.Epoch(1), bh.Epoch)
	require.Len(t, bh.Hashes, 1)
}

func TestAddBlockHashes_StopsAtNilBlock(t *testing.T) {
	ctrl := gomock.NewController(t)
	world := NewMockExternal(ctrl)

	em := &Emitter{
		config: config.Config{},
		world:  World{External: world},
	}

	world.EXPECT().GetLatestBlockIndex().Return(idx.Block(3))
	world.EXPECT().GetBlock(idx.Block(1)).Return((*inter.Block)(nil))

	event := &inter.MutableEventPayload{}
	em.addBlockHashes(event)

	require.Empty(t, event.BlockHashes().Hashes)
}

func TestAddBlockHashes_RespectsMaxBlockHashesPerEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	world := NewMockExternal(ctrl)

	em := &Emitter{
		config: config.Config{},
		world:  World{External: world},
	}

	// Make more blocks available than the maximum
	numBlocks := maxBlockHashesPerEvent + 10
	world.EXPECT().GetLatestBlockIndex().Return(idx.Block(numBlocks))

	for i := idx.Block(1); i <= idx.Block(maxBlockHashesPerEvent); i++ {
		block := inter.NewBlockBuilder().
			WithNumber(uint64(i)).
			WithEpoch(idx.Epoch(1)).
			WithBaseFee(big.NewInt(0)).
			Build()
		world.EXPECT().GetBlock(i).Return(block)
	}

	event := &inter.MutableEventPayload{}
	em.addBlockHashes(event)

	bh := event.BlockHashes()
	require.Len(t, bh.Hashes, maxBlockHashesPerEvent)
}

func TestAddBlockHashes_PersistsAndReadsTip(t *testing.T) {
	ctrl := gomock.NewController(t)
	world := NewMockExternal(ctrl)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "bvs")

	em := &Emitter{
		config: config.Config{
			PrevBlockVotesFile: config.FileConfig{
				Path: filePath,
			},
		},
		world: World{External: world},
	}
	em.emittedBvsFile = openPrevActionFile(filePath, false)
	defer func() { _ = em.emittedBvsFile.Close() }()

	// First call: collect blocks 1-2
	world.EXPECT().GetLatestBlockIndex().Return(idx.Block(2))
	block1 := inter.NewBlockBuilder().
		WithNumber(1).
		WithEpoch(idx.Epoch(1)).
		WithBaseFee(big.NewInt(0)).
		Build()
	block2 := inter.NewBlockBuilder().
		WithNumber(2).
		WithEpoch(idx.Epoch(1)).
		WithBaseFee(big.NewInt(0)).
		Build()
	world.EXPECT().GetBlock(idx.Block(1)).Return(block1)
	world.EXPECT().GetBlock(idx.Block(2)).Return(block2)

	event1 := &inter.MutableEventPayload{}
	em.addBlockHashes(event1)
	require.Len(t, event1.BlockHashes().Hashes, 2)

	// Second call: only block 3 should be collected (tip was persisted at 2)
	world.EXPECT().GetLatestBlockIndex().Return(idx.Block(3))
	block3 := inter.NewBlockBuilder().
		WithNumber(3).
		WithEpoch(idx.Epoch(1)).
		WithBaseFee(big.NewInt(0)).
		Build()
	world.EXPECT().GetBlock(idx.Block(3)).Return(block3)

	event2 := &inter.MutableEventPayload{}
	em.addBlockHashes(event2)
	bh := event2.BlockHashes()
	require.Equal(t, idx.Block(3), bh.Start)
	require.Len(t, bh.Hashes, 1)
}

func TestReadLastBlockHashesTip_ReturnsZeroWithNilFile(t *testing.T) {
	em := &Emitter{}
	require.Equal(t, idx.Block(0), em.readLastBlockHashesTip())
}

func TestWriteLastBlockHashesTip_NoOpWithNilFile(t *testing.T) {
	em := &Emitter{}
	// Should not panic
	em.writeLastBlockHashesTip(42)
}

func TestReadWriteLastBlockHashesTip_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "bvs")

	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0600)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	em := &Emitter{emittedBvsFile: f}

	em.writeLastBlockHashesTip(123)
	require.Equal(t, idx.Block(123), em.readLastBlockHashesTip())

	em.writeLastBlockHashesTip(456)
	require.Equal(t, idx.Block(456), em.readLastBlockHashesTip())
}
