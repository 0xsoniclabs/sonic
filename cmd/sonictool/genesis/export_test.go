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
	"math/big"
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
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

func TestExportBundles_WritesIntoWriter(t *testing.T) {
	store := setupBundleStore(t)
	store.AddProcessedBundles(1, map[common.Hash]bundle.PositionInBlock{
		{1}: {Offset: 0, Count: 2},
	})

	writer := newDryRunWriter(t)

	err := exportBundles(context.Background(), store, writer, 10)
	require.NoError(t, err)
	// Even with no bundles, the history hash is always written.
	require.Greater(t, writer.uncompressedSize, uint64(0),
		"history hash should always be written")

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

func TestBundles_WriteError(t *testing.T) {
	store := setupBundleStore(t)
	store.AddProcessedBundles(1, map[common.Hash]bundle.PositionInBlock{
		{1}: {Offset: 0, Count: 1},
	})

	writer := &unitWriter{}
	writer.fileshasher = fileshash.WrapWriter(nil, genesisstore.FilesHashPieceSize,
		func(int) fileshash.TmpWriter {
			return &failingTmpWriter{}
		},
	)
	err := exportBundlesHash(context.Background(), store, writer, 10)
	require.Error(t, err)

	err = exportBundles(context.Background(), store, writer, 10)
	require.Error(t, err)
}

func TestBundles_RoundTrip(t *testing.T) {
	store := setupBundleStore(t)

	wantBundles := map[common.Hash]bundle.PositionInBlock{
		{0xaa}: {Offset: 0, Count: 2},
		{0xbb}: {Offset: 2, Count: 1},
	}
	store.AddProcessedBundles(1, wantBundles)
	wantHistoryHash := common.Hash{0xff}
	store.SetProcessedBundlesHistoryHash(1, wantHistoryHash)

	// Export to a real file.
	tmpDir := t.TempDir()
	outFile, err := os.CreateTemp(tmpDir, "export-bundles-*.g")
	require.NoError(t, err)
	defer func() { require.NoError(t, outFile.Close()) }()

	header := genesis.Header{}
	writer := newUnitWriter(outFile)

	err = writer.Start(header, "bh", tmpDir)
	require.NoError(t, err)
	err = exportBundlesHash(context.Background(), store, writer, 10)
	require.NoError(t, err)

	err = writer.Start(header, "bundles", tmpDir)
	require.NoError(t, err)
	err = exportBundles(context.Background(), store, writer, 10)
	require.NoError(t, err)

	// Re-open the file and read back through genesisstore.
	_, err = outFile.Seek(0, io.SeekStart)
	require.NoError(t, err)

	gs, _, err := genesisstore.OpenGenesisStore(outFile)
	require.NoError(t, err)

	// Verify history hash roundtrips correctly.
	gotHist, ok := gs.ProcessedBundles().GetHistoryHash()
	require.True(t, ok, "history hash should be present")
	require.Equal(t, uint64(1), gotHist.BlockNumber)
	require.Equal(t, wantHistoryHash, gotHist.Hash)

	// Verify bundle execution infos roundtrip correctly.
	wantInfos := store.EnumerateProcessedBundles()
	var gotInfos []bundle.ExecutionInfo
	gs.ProcessedBundles().ForEach(func(info bundle.ExecutionInfo) bool {
		gotInfos = append(gotInfos, info)
		return true
	})
	require.Equal(t, wantInfos, gotInfos,
		"exported and re-imported bundle execution infos should match")
}

func TestBundles_DeterministicOutput(t *testing.T) {
	// Running export twice with the same data should produce the same hash.

	exporter := []func(context.Context, *gossip.Store, *unitWriter, idx.Block) error{
		exportBundlesHash,
		exportBundles,
	}

	for _, exp := range exporter {
		t.Run("exporter", func(t *testing.T) {
			s := setupBundleStore(t)
			s.AddProcessedBundles(1, map[common.Hash]bundle.PositionInBlock{
				{0x01}: {Offset: 0, Count: 1},
			})
			s.SetProcessedBundlesHistoryHash(1, common.Hash{0x42})

			hashWriter1 := newDryRunWriter(t)
			err := exp(context.Background(), s, hashWriter1, 10)
			require.NoError(t, err)

			hashWriter2 := newDryRunWriter(t)
			err = exp(context.Background(), s, hashWriter2, 10)
			require.NoError(t, err)

			require.Equal(t, hashWriter1.fileshasher.Root(), hashWriter2.fileshasher.Root(),
				"same input should produce same output hash")
		})
	}
}

func TestMustRlpEncodeToByte_PanicsOnError(t *testing.T) {
	require.Panics(t, func() {
		MustRlpEncodeToByte(new(big.Int).Sub(big.NewInt(0), big.NewInt(1)))
	}, "MustRlpEncodeToByte should panic on encoding error")
}

func TestMustRlpEncodeToByte_CanEncodeHistoryHashAndExecutionInfo(t *testing.T) {

	tests := map[string]any{
		"history hash": bundle.HistoryHash{
			BlockNumber: 123,
			Hash:        common.HexToHash("0xdeadbeef"),
		},
		"execution info": bundle.ExecutionInfo{
			BlockNumber:       123,
			ExecutionPlanHash: common.Hash{0x42},
			Position:          bundle.PositionInBlock{Offset: 1, Count: 2},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			expected, err := rlp.EncodeToBytes(test)
			require.NoError(t, err)

			result := MustRlpEncodeToByte(test)
			require.Equal(t, expected, result, "MustRlpEncodeToByte should return correct encoded bytes")
		})
	}
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
