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
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/logger"
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
	const lastBlock = 1100
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
	const lastBlock = 1030
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
		uint64(1023))

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

func TestImportProcessedBundles_LogsCrit_OnDuplicateExecutionPlan(t *testing.T) {
	store, err := NewMemStore(t)
	require.NoError(t, err)
	log := logger.NewMockLogger(gomock.NewController(t))
	store.Log = log
	log.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	const block = 5
	duplicateErr := fmt.Errorf("duplicate execution plan hash %s", common.Hash{0x01})
	expectCrit(log, "Invalid processed bundles in genesis", "block", uint64(block), "err", duplicateErr)

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
	require.Panics(t, func() { _ = store.importProcessedBundles(bundles) })
}

func TestImportProcessedBundles_LogsCrit_WhenBundlesHaveNoHistoryHash(t *testing.T) {
	store, _, log, _, _ := storeTableLogMocks(t)
	const msg = "Bundles were processed but no history hash was found in genesis"
	log.EXPECT().Crit(msg).Do(func(msg string, _ ...any) { panic(msg) })

	bundles := fakeProcessedBundles{
		infos: []bundle.ExecutionInfo{{BlockNumber: 1, ExecutionPlanHash: common.Hash{0x01}}},
	}
	require.Panics(t, func() { _ = store.importProcessedBundles(bundles) })
}

func TestImportProcessedBundles_ReportsError_WhenRestoringHistoryHashFails(t *testing.T) {
	store, table, log, _, _ := storeTableLogMocks(t)
	log.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	latest := bundle.HistoryHash{BlockNumber: 7, Hash: common.Hash{0x42}}
	injectedErr := errors.New("put error")
	table.EXPECT().Put(nil, gomock.Any()).Return(nil)
	table.EXPECT().Put(getBundleHistoryHashKey(latest.BlockNumber), latest.Hash.Bytes()).
		Return(injectedErr)

	bundles := fakeProcessedBundles{
		historyHashes: &bundle.BundleGenesisHistoryHashes{Latest: latest, Oldest: latest},
	}
	err := store.importProcessedBundles(bundles)
	require.ErrorIs(t, err, injectedErr)
}
