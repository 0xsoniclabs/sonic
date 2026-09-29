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

package gossip

import (
	"errors"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// fakeProcessedBundles provides processed bundles as exported from a store.
type fakeProcessedBundles struct {
	infos         []bundle.ExecutionInfo
	historyHashes *bundle.BundleGenesisHistoryHashes
}

func (f fakeProcessedBundles) ForEach(fn func(bundle.ExecutionInfo) bool) {
	for _, info := range f.infos {
		if !fn(info) {
			return
		}
	}
}

func (f fakeProcessedBundles) GetHistoryHashes() (bundle.BundleGenesisHistoryHashes, bool) {
	if f.historyHashes == nil {
		return bundle.BundleGenesisHistoryHashes{}, false
	}
	return *f.historyHashes, true
}

// exportProcessedBundles mirrors the genesis export of processed bundles.
func exportProcessedBundles(t *testing.T, store *Store) fakeProcessedBundles {
	t.Helper()
	res := fakeProcessedBundles{infos: store.EnumerateProcessedBundles()}
	oldestBlock, oldestHash, ok := store.GetEarliestBundleHistoryHash()
	if ok {
		latestBlock, latestHash := store.GetLatestProcessedBundleHistoryHash()
		res.historyHashes = &bundle.BundleGenesisHistoryHashes{
			Latest: bundle.HistoryHash{BlockNumber: latestBlock, Hash: latestHash},
			Oldest: bundle.HistoryHash{BlockNumber: oldestBlock, Hash: oldestHash},
		}
	}
	return res
}

func TestImportProcessedBundles_RestoresHistoryHash_WhenNoBundlesAreRetained(t *testing.T) {
	require := require.New(t)

	source, err := NewMemStore(t)
	require.NoError(err)

	// a bundle is executed and enough empty blocks are produced to prune it
	// while the history hash keeps being updated for every block
	source.AddProcessedBundles(1, map[common.Hash]bundle.PositionInBlock{
		{0x01}: {Offset: 0, Count: 1},
	})
	const lastBlock = ProcessedBundlesRetention + 100
	for block := uint64(2); block <= lastBlock; block++ {
		source.AddProcessedBundles(block, nil)
	}
	exported := exportProcessedBundles(t, source)
	require.Empty(exported.infos, "bundle should have been pruned")
	require.NotNil(exported.historyHashes)

	_, wantHash := source.GetLatestProcessedBundleHistoryHash()
	require.NotEqual(common.Hash{}, wantHash)

	target, err := NewMemStore(t)
	require.NoError(err)
	require.NoError(target.importProcessedBundles(exported))

	gotBlock, gotHash := target.GetLatestProcessedBundleHistoryHash()
	require.Equal(uint64(lastBlock), gotBlock)
	require.Equal(wantHash, gotHash)

	// the imported state can be exported again
	require.Equal(exportProcessedBundles(t, target).historyHashes.Latest,
		exported.historyHashes.Latest)

	// processing further blocks keeps both stores aligned
	for block := uint64(lastBlock + 1); block <= lastBlock+10; block++ {
		source.AddProcessedBundles(block, nil)
		target.AddProcessedBundles(block, nil)
	}
	_, wantHash = source.GetLatestProcessedBundleHistoryHash()
	_, gotHash = target.GetLatestProcessedBundleHistoryHash()
	require.Equal(wantHash, gotHash)
}

func TestImportProcessedBundles_ReproducesHistoryHash_WhenBundlesAreRetained(t *testing.T) {
	require := require.New(t)

	source, err := NewMemStore(t)
	require.NoError(err)
	source.AddProcessedBundles(3, map[common.Hash]bundle.PositionInBlock{
		{0x01}: {Offset: 0, Count: 1},
	})
	for block := uint64(4); block <= 10; block++ {
		source.AddProcessedBundles(block, nil)
	}
	exported := exportProcessedBundles(t, source)
	require.Len(exported.infos, 1)

	target, err := NewMemStore(t)
	require.NoError(err)
	require.NoError(target.importProcessedBundles(exported))

	_, wantHash := source.GetLatestProcessedBundleHistoryHash()
	_, gotHash := target.GetLatestProcessedBundleHistoryHash()
	require.Equal(wantHash, gotHash)
	require.True(target.HasBundleRecentlyBeenProcessed(common.Hash{0x01}))
}

func TestImportProcessedBundles_KeepsZeroHistoryHash_WhenNoBundlesWereEverExecuted(t *testing.T) {
	require := require.New(t)

	target, err := NewMemStore(t)
	require.NoError(err)
	require.NoError(target.importProcessedBundles(fakeProcessedBundles{}))

	block, hash := target.GetLatestProcessedBundleHistoryHash()
	require.Zero(block)
	require.Equal(common.Hash{}, hash)
	_, _, found := target.GetEarliestBundleHistoryHash()
	require.False(found)
}

func TestPositionsByExecutionPlan_IndexesPositions(t *testing.T) {
	positions, err := positionsByExecutionPlan([]bundle.ExecutionInfo{
		{ExecutionPlanHash: common.Hash{0x01}, Position: bundle.PositionInBlock{Offset: 0, Count: 1}},
		{ExecutionPlanHash: common.Hash{0x02}, Position: bundle.PositionInBlock{Offset: 1, Count: 2}},
	})
	require.NoError(t, err)
	require.Equal(t, map[common.Hash]bundle.PositionInBlock{
		{0x01}: {Offset: 0, Count: 1},
		{0x02}: {Offset: 1, Count: 2},
	}, positions)
}

func TestPositionsByExecutionPlan_RejectsDuplicateExecutionPlans(t *testing.T) {
	_, err := positionsByExecutionPlan([]bundle.ExecutionInfo{
		{ExecutionPlanHash: common.Hash{0x01}},
		{ExecutionPlanHash: common.Hash{0x01}},
	})
	require.ErrorContains(t, err, "duplicate execution plan hash")
}

func TestImportProcessedBundles_ReplaysFromOldestHash_WhenRangeExceedsRetentionWindow(t *testing.T) {
	require := require.New(t)

	source, err := NewMemStore(t)
	require.NoError(err)
	source.AddProcessedBundles(1, map[common.Hash]bundle.PositionInBlock{
		{0x01}: {Offset: 0, Count: 1},
	})
	const lastBlock = ProcessedBundlesRetention + 6
	for block := uint64(2); block <= lastBlock; block++ {
		source.AddProcessedBundles(block, nil)
	}
	source.AddProcessedBundles(lastBlock+1, map[common.Hash]bundle.PositionInBlock{
		{0x02}: {Offset: 0, Count: 1},
	})
	exported := exportProcessedBundles(t, source)
	require.Len(exported.infos, 1)
	require.GreaterOrEqual(
		exported.historyHashes.Latest.BlockNumber-exported.historyHashes.Oldest.BlockNumber,
		ProcessedBundlesRetention-1)

	target, err := NewMemStore(t)
	require.NoError(err)
	require.NoError(target.importProcessedBundles(exported))

	_, wantHash := source.GetLatestProcessedBundleHistoryHash()
	_, gotHash := target.GetLatestProcessedBundleHistoryHash()
	require.Equal(wantHash, gotHash)
	require.True(target.HasBundleRecentlyBeenProcessed(common.Hash{0x02}))
}

func TestImportProcessedBundles_FailsOnHistoryHashMismatch(t *testing.T) {
	require := require.New(t)

	source, err := NewMemStore(t)
	require.NoError(err)
	source.AddProcessedBundles(1, map[common.Hash]bundle.PositionInBlock{
		{0x01}: {Offset: 0, Count: 1},
	})
	exported := exportProcessedBundles(t, source)
	exported.historyHashes.Latest.Hash = common.Hash{0x42}

	target, err := NewMemStore(t)
	require.NoError(err)
	err = target.importProcessedBundles(exported)
	require.ErrorContains(err, "reproduced latest bundle history hash does not match genesis")
}

func TestImportProcessedBundles_ReportsError_OnDuplicateExecutionPlan(t *testing.T) {
	store, err := NewMemStore(t)
	require.NoError(t, err)

	const block = 5
	bundles := fakeProcessedBundles{
		infos: []bundle.ExecutionInfo{
			{BlockNumber: block, ExecutionPlanHash: common.Hash{0x01}},
			{BlockNumber: block, ExecutionPlanHash: common.Hash{0x01}},
		},
		historyHashes: &bundle.BundleGenesisHistoryHashes{
			Latest: bundle.HistoryHash{BlockNumber: block},
			Oldest: bundle.HistoryHash{BlockNumber: block},
		},
	}
	err = store.importProcessedBundles(bundles)
	require.ErrorContains(t, err, "invalid processed bundles in genesis at block 5")
	require.ErrorContains(t, err, "duplicate execution plan hash")
}

func TestImportProcessedBundles_ReportsError_OnDuplicateExecutionPlanInReplayedBlock(t *testing.T) {
	store, err := NewMemStore(t)
	require.NoError(t, err)

	const base, block = 4, 5
	bundles := fakeProcessedBundles{
		infos: []bundle.ExecutionInfo{
			{BlockNumber: block, ExecutionPlanHash: common.Hash{0x01}},
			{BlockNumber: block, ExecutionPlanHash: common.Hash{0x01}},
		},
		historyHashes: &bundle.BundleGenesisHistoryHashes{
			Oldest: bundle.HistoryHash{BlockNumber: base, Hash: common.Hash{0x42}},
			Latest: bundle.HistoryHash{BlockNumber: block},
		},
	}
	err = store.importProcessedBundles(bundles)
	require.ErrorContains(t, err, "invalid processed bundles in genesis at block 5")
	require.ErrorContains(t, err, "duplicate execution plan hash")
}

func TestImportProcessedBundles_ReportsError_WhenBundlesHaveNoHistoryHash(t *testing.T) {
	store, _, _, _, _ := storeTableLogMocks(t)

	bundles := fakeProcessedBundles{
		infos: []bundle.ExecutionInfo{{BlockNumber: 1, ExecutionPlanHash: common.Hash{0x01}}},
	}
	err := store.importProcessedBundles(bundles)
	require.ErrorContains(t, err, "bundles were processed but no history hash was found in genesis")
}

func TestImportProcessedBundles_ReportsError_WhenRestoringHistoryBaseFails(t *testing.T) {
	store, table, log, batch, _ := storeTableLogMocks(t)
	log.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	latest := bundle.HistoryHash{BlockNumber: 7, Hash: common.Hash{0x42}}
	injectedErr := errors.New("put error")
	table.EXPECT().NewBatch().Return(batch)
	batch.EXPECT().Put(nil, gomock.Any()).Return(nil)
	batch.EXPECT().Put(getBundleHistoryHashKey(latest.BlockNumber), latest.Hash.Bytes()).
		Return(injectedErr)

	bundles := fakeProcessedBundles{
		historyHashes: &bundle.BundleGenesisHistoryHashes{Latest: latest, Oldest: latest},
	}
	err := store.importProcessedBundles(bundles)
	require.ErrorIs(t, err, injectedErr)
}

func TestImportProcessedBundles_ReportsError_WhenRestoringBundlesOfBaseFails(t *testing.T) {
	store, table, log, batch, _ := storeTableLogMocks(t)
	log.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	const block = 5
	injectedErr := errors.New("entry put error")
	table.EXPECT().NewBatch().Return(batch)
	batch.EXPECT().Put(getEntryKey(common.Hash{0x01}), gomock.Any()).Return(injectedErr)
	batch.EXPECT().Put(getIndexKey(block, common.Hash{0x01}), gomock.Any()).Return(nil)
	// no batch.Write() is expected

	base := bundle.HistoryHash{BlockNumber: block, Hash: common.Hash{0x42}}
	bundles := fakeProcessedBundles{
		infos:         []bundle.ExecutionInfo{{BlockNumber: block, ExecutionPlanHash: common.Hash{0x01}}},
		historyHashes: &bundle.BundleGenesisHistoryHashes{Latest: base, Oldest: base},
	}
	err := store.importProcessedBundles(bundles)
	require.ErrorContains(t, err, "failed to restore processed bundles of block 5")
	require.ErrorIs(t, err, injectedErr)
}

func TestImportProcessedBundles_ReportsError_WhenWritingHistoryBaseFails(t *testing.T) {
	store, table, log, batch, _ := storeTableLogMocks(t)
	log.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	injectedErr := errors.New("batch write error")
	table.EXPECT().NewBatch().Return(batch)
	batch.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil).Times(2)
	batch.EXPECT().Write().Return(injectedErr)

	base := bundle.HistoryHash{BlockNumber: 7, Hash: common.Hash{0x42}}
	bundles := fakeProcessedBundles{
		historyHashes: &bundle.BundleGenesisHistoryHashes{Latest: base, Oldest: base},
	}
	err := store.importProcessedBundles(bundles)
	require.ErrorIs(t, err, injectedErr)
}

func TestImportProcessedBundles_ReportsError_WhenReplayingBlockFails(t *testing.T) {
	store, table, log, batch, it := storeTableLogMocks(t)
	log.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	injectedErr := errors.New("batch write error")
	// the base is restored with a first batch, block 8 is replayed with a second one
	table.EXPECT().NewBatch().Return(batch).Times(2)
	// the retained bundles are recounted after restoring the base
	table.EXPECT().NewIterator([]byte{'i'}, nil).Return(it)
	it.EXPECT().Next().Return(false)
	it.EXPECT().Error().Return(nil)
	it.EXPECT().Release()
	batch.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil).Times(4)
	gomock.InOrder(
		batch.EXPECT().Write().Return(nil),
		batch.EXPECT().Write().Return(injectedErr),
	)
	table.EXPECT().Get(gomock.Any()).Return([]byte{8 + 32 - 1: 0x42}, nil)

	bundles := fakeProcessedBundles{
		historyHashes: &bundle.BundleGenesisHistoryHashes{
			Oldest: bundle.HistoryHash{BlockNumber: 7, Hash: common.Hash{0x42}},
			Latest: bundle.HistoryHash{BlockNumber: 8, Hash: common.Hash{0x43}},
		},
	}
	err := store.importProcessedBundles(bundles)
	require.ErrorContains(t, err, "failed to replay processed bundles of block 8")
	require.ErrorIs(t, err, injectedErr)
}

func TestImportProcessedBundles_ReplaysFromOldestHash_WhenSpanIsShortAndBaseIsNotZero(t *testing.T) {
	// A genesis may start its history at any block, e.g. a pruned genesis or
	// a genesis exported shortly after importing another one. The oldest hash
	// is the base of the replay regardless of the span of the history.
	require := require.New(t)

	source, err := NewMemStore(t)
	require.NoError(err)
	source.AddProcessedBundles(1, map[common.Hash]bundle.PositionInBlock{
		{0x01}: {Offset: 0, Count: 1},
	})
	for block := uint64(2); block <= 20; block++ {
		source.AddProcessedBundles(block, nil)
	}
	source.AddProcessedBundles(21, map[common.Hash]bundle.PositionInBlock{
		{0x02}: {Offset: 0, Count: 1},
	})

	const base = uint64(15)
	baseHash, ok := source.GetProcessedBundleHistoryHash(base)
	require.True(ok)
	require.NotEqual(common.Hash{}, baseHash)
	latestBlock, latestHash := source.GetLatestProcessedBundleHistoryHash()
	exported := fakeProcessedBundles{
		infos: []bundle.ExecutionInfo{*source.GetBundleExecutionInfo(common.Hash{0x02})},
		historyHashes: &bundle.BundleGenesisHistoryHashes{
			Latest: bundle.HistoryHash{BlockNumber: latestBlock, Hash: latestHash},
			Oldest: bundle.HistoryHash{BlockNumber: base, Hash: baseHash},
		},
	}

	target, err := NewMemStore(t)
	require.NoError(err)
	require.NoError(target.importProcessedBundles(exported))

	_, gotHash := target.GetLatestProcessedBundleHistoryHash()
	require.Equal(latestHash, gotHash)
	require.True(target.HasBundleRecentlyBeenProcessed(common.Hash{0x02}))

	// the base is retained, so the imported store can be exported and rolled back
	earliest, gotBaseHash, found := target.GetEarliestBundleHistoryHash()
	require.True(found)
	require.Equal(base, earliest)
	require.Equal(baseHash, gotBaseHash)
}

func TestImportProcessedBundles_RetainsBundlesUpToOldestHash_ForReplayProtection(t *testing.T) {
	require := require.New(t)

	source, err := NewMemStore(t)
	require.NoError(err)
	source.AddProcessedBundles(3, map[common.Hash]bundle.PositionInBlock{
		{0x01}: {Offset: 0, Count: 1},
	})
	for block := uint64(4); block <= 10; block++ {
		source.AddProcessedBundles(block, nil)
	}
	exported := exportProcessedBundles(t, source)
	require.Equal(uint64(3), exported.historyHashes.Oldest.BlockNumber)

	target, err := NewMemStore(t)
	require.NoError(err)
	require.NoError(target.importProcessedBundles(exported))

	require.Equal(source.GetBundleExecutionInfo(common.Hash{0x01}),
		target.GetBundleExecutionInfo(common.Hash{0x01}))
}
