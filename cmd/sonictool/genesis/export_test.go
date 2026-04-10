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

package genesis

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/genesis"
	"github.com/0xsoniclabs/sonic/opera/genesisstore"
	"github.com/0xsoniclabs/sonic/opera/genesisstore/fileshash"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

func TestExportBundles_WritesIntoWriter(t *testing.T) {

	tests := map[string]struct {
		storeSetup func(*gossip.Store)
	}{
		"empty store": {
			storeSetup: func(store *gossip.Store) {},
		},
		"store with history hash but no bundles": {
			storeSetup: func(store *gossip.Store) {
				store.SetProcessedBundlesHistoryHash(1, common.Hash{0x42})
			},
		},
		"store with bundles": {
			storeSetup: func(store *gossip.Store) {
				store.AddProcessedBundles(1, map[common.Hash]bundle.PositionInBlock{
					{1}: {Offset: 0, Count: 2},
					{2}: {Offset: 2, Count: 1},
				})
				store.AddProcessedBundles(2, map[common.Hash]bundle.PositionInBlock{
					{3}: {Offset: 0, Count: 1},
				})
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			store := setupBundleStore(t)
			writer := newDryRunWriter(t)

			tc.storeSetup(store)

			err := exportBundles(context.Background(), store, writer, 10)
			require.NoError(t, err)
			// Even with no bundles, the history hash is always written.
			require.Greater(t, writer.uncompressedSize, uint64(0),
				"history hash should always be written")

		})
	}
}

func TestExportBundles_DataIntegrity(t *testing.T) {
	store := setupBundleStore(t)

	store.AddProcessedBundles(1, map[common.Hash]bundle.PositionInBlock{
		{0x10}: {Offset: 0, Count: 1},
	})
	store.SetProcessedBundlesHistoryHash(5, common.Hash{0xde, 0xad})

	// Manually compute expected size.
	blockNum, histHash := store.GetProcessedBundleHistoryHash()
	histBytes, err := rlp.EncodeToBytes(bundle.HistoryHash{
		BlockNumber: blockNum,
		Hash:        histHash,
	})
	require.NoError(t, err)
	expectedSize := uint64(len(histBytes))

	for _, info := range store.EnumerateProcessedBundles() {
		b, err := rlp.EncodeToBytes(info)
		require.NoError(t, err)
		expectedSize += uint64(len(b))
	}

	writer := newDryRunWriter(t)
	err = exportBundles(context.Background(), store, writer, 100)
	require.NoError(t, err)
	require.Equal(t, expectedSize, writer.uncompressedSize,
		"written bytes should match manually encoded data")
}

func TestExportBundles_MoreBundlesProducesMoreData(t *testing.T) {
	// Store with 1 bundle
	store1 := setupBundleStore(t)
	store1.AddProcessedBundles(1, map[common.Hash]bundle.PositionInBlock{
		{1}: {Offset: 0, Count: 1},
	})
	writer1 := newDryRunWriter(t)
	err := exportBundles(context.Background(), store1, writer1, 10)
	require.NoError(t, err)

	// Store with 3 bundles
	store3 := setupBundleStore(t)
	store3.AddProcessedBundles(1, map[common.Hash]bundle.PositionInBlock{
		{1}: {Offset: 0, Count: 1},
		{2}: {Offset: 1, Count: 1},
		{3}: {Offset: 2, Count: 1},
	})
	writer3 := newDryRunWriter(t)
	err = exportBundles(context.Background(), store3, writer3, 10)
	require.NoError(t, err)

	require.Greater(t, writer3.uncompressedSize, writer1.uncompressedSize,
		"3 bundles should produce more data than 1 bundle")
}

func TestExportBundles_ContextCancelledImmediately(t *testing.T) {
	store := setupBundleStore(t)
	store.AddProcessedBundles(1, map[common.Hash]bundle.PositionInBlock{
		{1}: {Offset: 0, Count: 1},
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	writer := newDryRunWriter(t)
	err := exportBundles(ctx, store, writer, 10)
	require.ErrorIs(t, err, context.Canceled)
}

func TestExportBundles_ContextCancelledAfterFirstBundle(t *testing.T) {
	store := setupBundleStore(t)
	store.AddProcessedBundles(1, map[common.Hash]bundle.PositionInBlock{
		{1}: {Offset: 0, Count: 1},
		{2}: {Offset: 1, Count: 1},
		{3}: {Offset: 2, Count: 1},
	})

	// Allow 1 ctx.Err() check to pass (after first bundle write), then cancel.
	ctx := &cancelAfterNChecks{
		Context:      context.Background(),
		allowedCalls: 1,
	}

	writer := newDryRunWriter(t)
	err := exportBundles(ctx, store, writer, 10)
	require.ErrorIs(t, err, context.Canceled)
	require.Greater(t, writer.uncompressedSize, uint64(0),
		"some data should have been written before cancellation")
}

func TestExportBundles_HistoryHash_WriteError(t *testing.T) {
	store := setupBundleStore(t)

	writer := &unitWriter{}
	writer.fileshasher = fileshash.WrapWriter(nil, genesisstore.FilesHashPieceSize,
		func(int) fileshash.TmpWriter {
			return &failingTmpWriter{}
		},
	)

	err := exportBundles(context.Background(), store, writer, 10)
	require.Error(t, err)
}

func TestExportBundles_WrittenWithRealFile(t *testing.T) {
	store := setupBundleStore(t)
	store.AddProcessedBundles(1, map[common.Hash]bundle.PositionInBlock{
		{0xaa}: {Offset: 0, Count: 2},
	})
	store.SetProcessedBundlesHistoryHash(1, common.Hash{0xff})

	tmpDir := t.TempDir()
	outFile, err := os.CreateTemp(tmpDir, "export-bundles-*")
	require.NoError(t, err)
	defer func() { require.NoError(t, outFile.Close()) }()

	writer := newUnitWriter(outFile)
	err = writer.Start(genesis.Header{}, "bundles", tmpDir)
	require.NoError(t, err)

	err = exportBundles(context.Background(), store, writer, 10)
	require.NoError(t, err)
	require.Greater(t, writer.uncompressedSize, uint64(0))
}

func TestExportBundles_DeterministicOutput(t *testing.T) {
	// Running export twice with the same data should produce the same hash.
	makeStore := func() *gossip.Store {
		s := setupBundleStore(t)
		s.AddProcessedBundles(1, map[common.Hash]bundle.PositionInBlock{
			{0x01}: {Offset: 0, Count: 1},
		})
		s.SetProcessedBundlesHistoryHash(1, common.Hash{0x42})
		return s
	}

	writer1 := newDryRunWriter(t)
	err := exportBundles(context.Background(), makeStore(), writer1, 10)
	require.NoError(t, err)

	writer2 := newDryRunWriter(t)
	err = exportBundles(context.Background(), makeStore(), writer2, 10)
	require.NoError(t, err)

	require.Equal(t, writer1.fileshasher.Root(), writer2.fileshasher.Root(),
		"same input should produce same output hash")
}

func TestExportBundles_HistoryHashAlwaysWrittenFirst(t *testing.T) {
	storeEmpty := setupBundleStore(t)
	writerEmpty := newDryRunWriter(t)
	err := exportBundles(context.Background(), storeEmpty, writerEmpty, 10)
	require.NoError(t, err)
	histOnlySize := writerEmpty.uncompressedSize

	storeWithBundles := setupBundleStore(t)
	storeWithBundles.AddProcessedBundles(1, map[common.Hash]bundle.PositionInBlock{
		{1}: {Offset: 0, Count: 1},
	})
	writerWithBundles := newDryRunWriter(t)
	err = exportBundles(context.Background(), storeWithBundles, writerWithBundles, 10)
	require.NoError(t, err)

	require.Greater(t, writerWithBundles.uncompressedSize, histOnlySize,
		"bundles should add data beyond just the history hash")
}

// ------------- tooling for tests -------------

// failingTmpWriter implements fileshash.TmpWriter but always fails on Write.
type failingTmpWriter struct{}

func (f *failingTmpWriter) Read(p []byte) (int, error)                   { return 0, errors.New("read error") }
func (f *failingTmpWriter) Write(p []byte) (int, error)                  { return 0, errors.New("write error") }
func (f *failingTmpWriter) Seek(offset int64, whence int) (int64, error) { return 0, nil }
func (f *failingTmpWriter) Close() error                                 { return nil }
func (f *failingTmpWriter) Drop() error                                  { return nil }

// Ensure failingTmpWriter satisfies the interface.
var _ fileshash.TmpWriter = (*failingTmpWriter)(nil)
var _ io.ReadWriteSeeker = (*failingTmpWriter)(nil)

// cancelAfterNChecks is a context.Context wrapper that returns
// context.Canceled after allowedCalls calls to Err().
type cancelAfterNChecks struct {
	context.Context
	mu           sync.Mutex
	allowedCalls int
	calls        int
}

func (c *cancelAfterNChecks) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	if c.calls > c.allowedCalls {
		return context.Canceled
	}
	return nil
}

// newDryRunWriter creates a unitWriter in dry-run mode (nil plain)
// which writes data to DevNull-backed tmp files. Useful for testing
// export logic without writing real files.
func newDryRunWriter(t *testing.T) *unitWriter {
	t.Helper()
	w := newUnitWriter(nil)
	err := w.Start(genesis.Header{}, "test", "")
	require.NoError(t, err)
	return w
}

// setupBundleStore creates a gossip.Store with a current epoch state set
// and optionally populated with processed bundles and a history hash.
func setupBundleStore(t *testing.T) *gossip.Store {
	t.Helper()
	store, err := gossip.NewMemStore(t)
	require.NoError(t, err)

	rules := opera.FakeNetRules(opera.Upgrades{})
	store.SetBlockEpochState(
		iblockproc.BlockState{},
		iblockproc.EpochState{Epoch: 1, Rules: rules},
	)
	return store
}
