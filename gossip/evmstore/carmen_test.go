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
	"testing"

	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCarmenStateDB_ReportedExecutionPlansAreMarkedAsExecuted(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	store := NewMockProcessedBundleStore(ctrl)
	store.EXPECT().HasBeenProcessed(gomock.Any()).Return(false).AnyTimes()

	state := &CarmenStateDB{
		processedExecPlanStore: store,
	}

	plan1 := common.Hash{1}
	plan2 := common.Hash{2}
	plan3 := common.Hash{3}

	require.False(state.HasBeenProcessed(plan1))
	require.False(state.HasBeenProcessed(plan2))
	require.False(state.HasBeenProcessed(plan3))

	state.AddProcessedBundle(plan1)

	require.True(state.HasBeenProcessed(plan1))
	require.False(state.HasBeenProcessed(plan2))
	require.False(state.HasBeenProcessed(plan3))

	state.AddProcessedBundle(plan2)

	require.True(state.HasBeenProcessed(plan1))
	require.True(state.HasBeenProcessed(plan2))
	require.False(state.HasBeenProcessed(plan3))

	// Marking a plan as processed multiple times should not cause any issues
	state.AddProcessedBundle(plan1)
	state.AddProcessedBundle(plan2)

	require.True(state.HasBeenProcessed(plan1))
	require.True(state.HasBeenProcessed(plan2))
	require.False(state.HasBeenProcessed(plan3))
}

func TestCarmenStateDB_ReportedExecutionPlansCanBeRolledBac(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	db := carmen.NewMockVmStateDB(ctrl)
	db.EXPECT().InterTxSnapshot().AnyTimes()
	db.EXPECT().RevertToInterTxSnapshot(gomock.Any()).AnyTimes()

	store := NewMockProcessedBundleStore(ctrl)
	store.EXPECT().HasBeenProcessed(gomock.Any()).Return(false).AnyTimes()

	state := &CarmenStateDB{
		db:                     db,
		processedExecPlanStore: store,
	}

	plan1 := common.Hash{1}
	plan2 := common.Hash{2}
	plan3 := common.Hash{3}

	require.False(state.HasBeenProcessed(plan1))
	require.False(state.HasBeenProcessed(plan2))
	require.False(state.HasBeenProcessed(plan3))

	s1 := state.InterTxSnapshot()
	state.AddProcessedBundle(plan1)

	s2 := state.InterTxSnapshot()
	state.AddProcessedBundle(plan2)

	require.True(state.HasBeenProcessed(plan1))
	require.True(state.HasBeenProcessed(plan2))
	require.False(state.HasBeenProcessed(plan3))

	state.RevertToInterTxSnapshot(s2)

	require.True(state.HasBeenProcessed(plan1))
	require.False(state.HasBeenProcessed(plan2))
	require.False(state.HasBeenProcessed(plan3))

	state.RevertToInterTxSnapshot(s1)

	require.False(state.HasBeenProcessed(plan1))
	require.False(state.HasBeenProcessed(plan2))
	require.False(state.HasBeenProcessed(plan3))
}

func TestCarmenStateDB_HasBeenProcessed_ConsultsUnderlyingStore(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	plan1 := common.Hash{1}
	plan2 := common.Hash{2}

	store := NewMockProcessedBundleStore(ctrl)
	store.EXPECT().HasBeenProcessed(plan1).Return(false)
	store.EXPECT().HasBeenProcessed(plan2).Return(true)

	state := &CarmenStateDB{
		processedExecPlanStore: store,
	}

	require.False(state.HasBeenProcessed(plan1))
	require.True(state.HasBeenProcessed(plan2))
}
