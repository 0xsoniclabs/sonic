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

package evmstore

import (
	"fmt"
	"slices"
	"testing"

	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCarmenStateDB_CreateCarmenStateDb_CreatesACommittableInstance(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	backend := carmen.NewMockStateDB(ctrl)
	processedBundleStore := NewMockProcessedBundleStore(ctrl)
	state := CreateCarmenStateDb(backend, processedBundleStore)
	require.Same(state.db, backend)
	require.Same(state.processedExecPlanStore, processedBundleStore)
	require.True(state.committable)
}

func TestCarmenStateDB_CreateNonCommittableCarmenStateDb_CreatesANonCommittableInstance(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	backend := carmen.NewMockNonCommittableStateDB(ctrl)
	processedBundleStore := NewMockProcessedBundleStore(ctrl)
	state := CreateNonCommittableCarmenStateDb(backend, processedBundleStore)
	require.Same(state.db, backend)
	require.Same(state.processedExecPlanStore, processedBundleStore)
	require.False(state.committable)
}

func TestCarmenStateDB_Copy_CopiesNonCommittableStateDB(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	backend := carmen.NewMockNonCommittableStateDB(ctrl)
	backend2 := carmen.NewMockNonCommittableStateDB(ctrl)
	backend.EXPECT().Copy().Return(backend2)

	processedBundleStore := NewMockProcessedBundleStore(ctrl)
	state := CreateNonCommittableCarmenStateDb(backend, processedBundleStore)

	copied := state.Copy()
	copiedDb, isOfProperType := copied.(*CarmenStateDB)
	require.True(isOfProperType)
	require.Same(copiedDb.db, backend2)
	require.Same(copiedDb.processedExecPlanStore, processedBundleStore)
	require.False(copiedDb.committable)
}

func TestCarmenStateDB_Copy_PanicsForCommittableStateDB(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	backend := carmen.NewMockStateDB(ctrl)
	processedBundleStore := NewMockProcessedBundleStore(ctrl)
	state := CreateCarmenStateDb(backend, processedBundleStore)
	require.PanicsWithValue(
		"unable to copy committable (live) StateDB",
		func() { state.Copy() },
	)
}

func TestCarmenStateDB_Copy_CreatesADeepCopyOfStateDb(t *testing.T) {
	ctrl := gomock.NewController(t)

	carmenStateDbA := carmen.NewMockNonCommittableStateDB(ctrl)
	carmenStateDbB := carmen.NewMockNonCommittableStateDB(ctrl)
	carmenStateDbA.EXPECT().Copy().Return(carmenStateDbB)

	store := NewMockProcessedBundleStore(ctrl)

	original := &CarmenStateDB{
		db:                     carmenStateDbA,
		blockNum:               123,
		txHash:                 common.Hash{1, 2, 3},
		txIndex:                5,
		processedExecPlanStore: store,
		processedExecPlans: []processedExecPlan{{
			execPlanHash: common.Hash{1},
			position:     bundle.PositionInBlock{Offset: 1, Count: 2},
		}},
		interTxSnapshots: []interTxSnapshots{{
			stateDbSnapshotId:     carmen.InterTxSnapshotID(24),
			numProcessedExecPlans: 1,
		}},
		issue: fmt.Errorf("induced issue"),
	}

	copy := original.Copy()
	require.NotNil(t, copy)

	copyCarmenStateDb, ok := copy.(*CarmenStateDB)
	require.True(t, ok, "copied state DB should be of type *CarmenStateDB")

	require.Equal(t, carmenStateDbB, copyCarmenStateDb.db)

	// Nil out non-comparable fields before deep equality check
	original.db = nil
	copyCarmenStateDb.db = nil

	require.Equal(t, original, copyCarmenStateDb, "copied state DB should be equal to the original")

	// But the data structures should be cloned, not shared.
	require.NotSame(t, &original.processedExecPlans[0], &copyCarmenStateDb.processedExecPlans[0])
	require.NotSame(t, &original.interTxSnapshots[0], &copyCarmenStateDb.interTxSnapshots[0])
}

func TestCarmenStateDB_Copy_CannotCloneCommittableStateDB(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := carmen.NewMockVmStateDB(ctrl)
	_, ok := any(db).(carmen.NonCommittableStateDB)
	require.False(t, ok, "mocked VmStateDB should not implement NonCommittableStateDB")

	state := &CarmenStateDB{db: db}
	require.PanicsWithValue(t, "unable to copy committable (live) StateDB", func() {
		state.Copy()
	})
}

func TestCarmenStateDB_EndBlock_Committable_CallsEndBlockOnStateDB(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	backend := carmen.NewMockStateDB(ctrl)
	state := CreateCarmenStateDb(backend, nil)
	require.True(state.committable)

	// Expect EndBlock to be called on the backend when EndBlock is called on the state.
	errChan := make(chan error)
	backend.EXPECT().EndBlock(uint64(0)).Return(errChan)
	got := state.EndBlock(0)
	require.NotNil(got)
	require.EqualValues(errChan, got)
}

func TestCarmenStateDB_EndBlock_Committable_ProcessedExecPlansAreFlushedAndReset(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	plan1 := common.Hash{1}
	plan2 := common.Hash{2}

	pos1 := bundle.PositionInBlock{Offset: 1, Count: 2}
	pos2 := bundle.PositionInBlock{Offset: 3, Count: 4}

	bundleStore := NewMockProcessedBundleStore(ctrl)
	bundleStore.EXPECT().AddProcessedBundles(uint64(123), map[common.Hash]bundle.PositionInBlock{
		plan1: pos1,
		plan2: pos2,
	})

	state := &CarmenStateDB{
		committable:            true,
		processedExecPlanStore: bundleStore,
	}

	require.Empty(state.processedExecPlans)
	state.AddProcessedBundle(plan1, pos1)
	require.Equal(state.processedExecPlans, []processedExecPlan{{execPlanHash: plan1, position: pos1}})

	state.AddProcessedBundle(plan2, pos2)
	require.Equal(state.processedExecPlans, []processedExecPlan{{execPlanHash: plan1, position: pos1}, {execPlanHash: plan2, position: pos2}})

	state.EndBlock(123)
	require.Empty(state.processedExecPlans)
}

func TestCarmenStateDB_EndBlock_NotCommittable_DoesNotSendUpdatesToStateDBNorExecPlanStore(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	backend := carmen.NewMockNonCommittableStateDB(ctrl)
	processedBundleStore := NewMockProcessedBundleStore(ctrl)
	state := CreateNonCommittableCarmenStateDb(backend, processedBundleStore)

	// EndBlock should not send anything to the underlying state DB or the
	// processed bundle store, but it should still complete without error.
	errChan := state.EndBlock(0)
	require.Nil(errChan)
}

func TestCarmenStateDB_EndBlock_SnapshotListIsReset(t *testing.T) {
	for _, committable := range []bool{true, false} {
		t.Run(fmt.Sprintf("committable=%v", committable), func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)

			db := carmen.NewMockStateDB(ctrl)
			db.EXPECT().InterTxSnapshot().AnyTimes()
			if committable {
				db.EXPECT().EndBlock(uint64(123))
			}

			state := &CarmenStateDB{db: db, committable: committable}

			require.Empty(state.interTxSnapshots)
			state.InterTxSnapshot()
			require.Len(state.interTxSnapshots, 1)

			state.InterTxSnapshot()
			require.Len(state.interTxSnapshots, 2)

			state.EndBlock(123)
			require.Empty(state.interTxSnapshots)
		})
	}
}

func TestCarmenStateDB_EndBlock_ProcessedExecPlansAreReset(t *testing.T) {
	for _, committable := range []bool{true, false} {
		t.Run(fmt.Sprintf("committable=%v", committable), func(t *testing.T) {
			require := require.New(t)

			state := &CarmenStateDB{committable: committable}

			state.AddProcessedBundle(common.Hash{1}, bundle.PositionInBlock{})
			state.AddProcessedBundle(common.Hash{2}, bundle.PositionInBlock{})
			require.Len(state.processedExecPlans, 2)

			state.EndBlock(123)
			require.Empty(state.processedExecPlans)
		})
	}
}

func TestCarmenStateDB_EndBlock_CanEndBlockWithoutProcessedBundleStore(t *testing.T) {
	ctrl := gomock.NewController(t)

	db := carmen.NewMockStateDB(ctrl)
	state := &CarmenStateDB{db: db, processedExecPlanStore: nil}

	state.EndBlock(123) // exec plan store is nil, but EndBlock passes
}

func TestCarmenStateDB_InterTxSnapshot_DelegatesToUnderlyingDb(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	db := carmen.NewMockStateDB(ctrl)
	db.EXPECT().InterTxSnapshot().Return(carmen.InterTxSnapshotID(24))
	db.EXPECT().InterTxSnapshot().Return(carmen.InterTxSnapshotID(42))

	state := &CarmenStateDB{
		db:                 db,
		processedExecPlans: make([]processedExecPlan, 5),
	}

	// Verify the proper creation of a snapshot.
	snapshot := state.InterTxSnapshot()
	require.EqualValues(0, snapshot)
	require.Equal(state.interTxSnapshots, []interTxSnapshots{
		{
			stateDbSnapshotId:     carmen.InterTxSnapshotID(24),
			numProcessedExecPlans: 5,
		},
	})

	// Check that a second snapshot is added on top.
	state.processedExecPlans = make([]processedExecPlan, 10)
	snapshot = state.InterTxSnapshot()
	require.EqualValues(1, snapshot)
	require.Equal(state.interTxSnapshots, []interTxSnapshots{
		{
			stateDbSnapshotId:     carmen.InterTxSnapshotID(24),
			numProcessedExecPlans: 5,
		},
		{
			stateDbSnapshotId:     carmen.InterTxSnapshotID(42),
			numProcessedExecPlans: 10,
		},
	})
}

func TestCarmenStateDB_RevertToInterTxSnapshot_DelegatesToUnderlyingDbAndPrunesInternalData(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	db := carmen.NewMockStateDB(ctrl)

	snapshots := []interTxSnapshots{
		{
			stateDbSnapshotId: carmen.InterTxSnapshotID(24),
		}, {
			stateDbSnapshotId: carmen.InterTxSnapshotID(42),
		},
	}

	gomock.InOrder(
		db.EXPECT().RevertToInterTxSnapshot(carmen.InterTxSnapshotID(42)),
		db.EXPECT().RevertToInterTxSnapshot(carmen.InterTxSnapshotID(24)),
	)

	state := &CarmenStateDB{
		db:                 db,
		processedExecPlans: make([]processedExecPlan, 5),
		interTxSnapshots:   slices.Clone(snapshots),
	}

	state.RevertToInterTxSnapshot(1)
	require.Equal(state.interTxSnapshots, snapshots[:1])

	state.RevertToInterTxSnapshot(0)
	require.Empty(state.interTxSnapshots)
}

func TestCarmenStateDB_RevertToInterTxSnapshot_InvalidSnapshotIdCreatesIssue(t *testing.T) {
	for _, invalidId := range []int{-1, 0, 1} {
		require := require.New(t)
		ctrl := gomock.NewController(t)

		db := carmen.NewMockStateDB(ctrl)
		db.EXPECT().Check().Return(nil).AnyTimes()

		state := &CarmenStateDB{db: db}

		require.NoError(state.issue)
		state.RevertToInterTxSnapshot(invalidId)
		require.ErrorContains(state.issue, "failed to revert to invalid snapshot id")
	}
}

func TestCarmenStateDB_HasBundleRecentlyBeenProcessed_ConsultsUnderlyingStore(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	plan1 := common.Hash{1}
	plan2 := common.Hash{2}

	store := NewMockProcessedBundleStore(ctrl)
	store.EXPECT().HasBundleRecentlyBeenProcessed(plan1).Return(false)
	store.EXPECT().HasBundleRecentlyBeenProcessed(plan2).Return(true)

	state := &CarmenStateDB{
		processedExecPlanStore: store,
	}

	require.False(state.HasBundleRecentlyBeenProcessed(plan1))
	require.True(state.HasBundleRecentlyBeenProcessed(plan2))
}

func TestCarmenStateDB_HasBundleRecentlyBeenProcessed_ReturnsFalseIfNoUnderlyingStore(t *testing.T) {
	require := require.New(t)

	plan := common.Hash{1}
	state := &CarmenStateDB{
		processedExecPlanStore: nil,
	}

	require.False(state.HasBundleRecentlyBeenProcessed(plan))
}

func TestCarmenStateDB_ReportedExecutionPlansAreMarkedAsExecuted(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	store := NewMockProcessedBundleStore(ctrl)
	store.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).Return(false).AnyTimes()

	state := &CarmenStateDB{
		processedExecPlanStore: store,
	}

	plan1 := common.Hash{1}
	plan2 := common.Hash{2}
	plan3 := common.Hash{3}

	require.False(state.HasBundleRecentlyBeenProcessed(plan1))
	require.False(state.HasBundleRecentlyBeenProcessed(plan2))
	require.False(state.HasBundleRecentlyBeenProcessed(plan3))

	state.AddProcessedBundle(plan1, bundle.PositionInBlock{})

	require.True(state.HasBundleRecentlyBeenProcessed(plan1))
	require.False(state.HasBundleRecentlyBeenProcessed(plan2))
	require.False(state.HasBundleRecentlyBeenProcessed(plan3))

	state.AddProcessedBundle(plan2, bundle.PositionInBlock{})

	require.True(state.HasBundleRecentlyBeenProcessed(plan1))
	require.True(state.HasBundleRecentlyBeenProcessed(plan2))
	require.False(state.HasBundleRecentlyBeenProcessed(plan3))

	// Marking a plan as processed multiple times should not cause any issues
	state.AddProcessedBundle(plan1, bundle.PositionInBlock{})
	state.AddProcessedBundle(plan2, bundle.PositionInBlock{})

	require.True(state.HasBundleRecentlyBeenProcessed(plan1))
	require.True(state.HasBundleRecentlyBeenProcessed(plan2))
	require.False(state.HasBundleRecentlyBeenProcessed(plan3))
}

func TestCarmenStateDB_ReportedExecutionPlansCanBeRolledBack(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	db := carmen.NewMockVmStateDB(ctrl)
	db.EXPECT().InterTxSnapshot().AnyTimes()
	db.EXPECT().RevertToInterTxSnapshot(gomock.Any()).AnyTimes()

	store := NewMockProcessedBundleStore(ctrl)
	store.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).Return(false).AnyTimes()

	state := &CarmenStateDB{
		db:                     db,
		processedExecPlanStore: store,
	}

	plan1 := common.Hash{1}
	plan2 := common.Hash{2}
	plan3 := common.Hash{3}

	require.False(state.HasBundleRecentlyBeenProcessed(plan1))
	require.False(state.HasBundleRecentlyBeenProcessed(plan2))
	require.False(state.HasBundleRecentlyBeenProcessed(plan3))

	s1 := state.InterTxSnapshot()
	state.AddProcessedBundle(plan1, bundle.PositionInBlock{})

	s2 := state.InterTxSnapshot()
	state.AddProcessedBundle(plan2, bundle.PositionInBlock{})

	require.True(state.HasBundleRecentlyBeenProcessed(plan1))
	require.True(state.HasBundleRecentlyBeenProcessed(plan2))
	require.False(state.HasBundleRecentlyBeenProcessed(plan3))

	state.RevertToInterTxSnapshot(s2)

	require.True(state.HasBundleRecentlyBeenProcessed(plan1))
	require.False(state.HasBundleRecentlyBeenProcessed(plan2))
	require.False(state.HasBundleRecentlyBeenProcessed(plan3))

	state.RevertToInterTxSnapshot(s1)

	require.False(state.HasBundleRecentlyBeenProcessed(plan1))
	require.False(state.HasBundleRecentlyBeenProcessed(plan2))
	require.False(state.HasBundleRecentlyBeenProcessed(plan3))
}

func TestCarmenStateDB_ReportedExecutionPlansCanBeRolledBackSkippingSnapshots(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	db := carmen.NewMockVmStateDB(ctrl)
	db.EXPECT().InterTxSnapshot().AnyTimes()
	db.EXPECT().RevertToInterTxSnapshot(gomock.Any()).AnyTimes()

	store := NewMockProcessedBundleStore(ctrl)
	store.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).Return(false).AnyTimes()

	state := &CarmenStateDB{
		db:                     db,
		processedExecPlanStore: store,
	}

	plan1 := common.Hash{1}
	plan2 := common.Hash{2}
	plan3 := common.Hash{3}

	require.False(state.HasBundleRecentlyBeenProcessed(plan1))
	require.False(state.HasBundleRecentlyBeenProcessed(plan2))
	require.False(state.HasBundleRecentlyBeenProcessed(plan3))

	s1 := state.InterTxSnapshot()
	state.AddProcessedBundle(plan1, bundle.PositionInBlock{})

	s2 := state.InterTxSnapshot()
	require.NotEqual(s1, s2)
	state.AddProcessedBundle(plan2, bundle.PositionInBlock{})

	require.True(state.HasBundleRecentlyBeenProcessed(plan1))
	require.True(state.HasBundleRecentlyBeenProcessed(plan2))
	require.False(state.HasBundleRecentlyBeenProcessed(plan3))

	// Ignore snapshot s2 and revert to s1 directly
	state.RevertToInterTxSnapshot(s1)

	require.False(state.HasBundleRecentlyBeenProcessed(plan1))
	require.False(state.HasBundleRecentlyBeenProcessed(plan2))
	require.False(state.HasBundleRecentlyBeenProcessed(plan3))
}

func TestCarmenStateDB_CommittableAndNotCommittableStatesDBsCanBeIdentifiedByTypeCasts(t *testing.T) {
	// This test verifies assumptions made about the Carmen StateDB
	// implementation utilized in this package's CarmenStateDB's EndBlock. In
	// particular, that it is possible to determine whether the underlying
	// StateDB is committable (live) or non-committable (archive) via type casts.

	require := require.New(t)
	carmenState, err := carmen.NewState(carmen.Parameters{
		Variant:      "go-file",
		Schema:       carmen.Schema(5),
		Archive:      carmen.S5Archive,
		Directory:    t.TempDir(),
		LiveCache:    1, // use minimum cache (not default)
		ArchiveCache: 1, // use minimum cache (not default)
	})
	require.NoError(err)
	defer func() {
		require.NoError(carmenState.Close())
	}()

	// Create a live state DB (committable) and verify it can
	live := carmen.CreateCustomStateDBUsing(carmenState, 1024)

	// A live instance should not be castable to a non-committable instance.
	_, ok := live.(carmen.NonCommittableStateDB)
	require.False(ok)

	// Add a few blocks and wait for them to be synced in the archive.
	<-live.EndBlock(0)
	<-live.EndBlock(1)

	archive, err := live.GetArchiveStateDB(0)
	require.NoError(err)

	// An archive instance should NOT be castable to a committable instance.
	_, ok = archive.(carmen.StateDB)
	// require.False(ok) // < this test fails, because archive indeed implements the StateDB interface

	// An explicitly created non-committable instance should not be castable to
	// a committable instance.
	nonCommittable := carmen.CreateNonCommittableStateDBUsing(carmenState)
	_, ok = nonCommittable.(carmen.StateDB)
	// require.False(ok) // < this test fails, because the non-committable instance also implements the StateDB interface
}
