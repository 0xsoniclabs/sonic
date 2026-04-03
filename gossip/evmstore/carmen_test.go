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
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCarmenStateDB_CreateCarmenStateDb_CreatesACommittableInstance(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	backend := carmen.NewMockStateDB(ctrl)
	state := CreateCarmenStateDb(backend)
	db, isOfProperType := state.(*CarmenStateDB)
	require.True(isOfProperType)
	require.Same(db.db, backend)
	require.True(db.committable)
}

func TestCarmenStateDB_createNonCommittableCarmenStateDb_CreatesANonCommittableInstance(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	backend := carmen.NewMockNonCommittableStateDB(ctrl)
	state := createNonCommittableCarmenStateDb(backend)
	db, isOfProperType := state.(*CarmenStateDB)
	require.True(isOfProperType)
	require.Same(db.db, backend)
	require.False(db.committable)
}

func TestCarmenStateDB_Copy_CopiesNonCommittableStateDB(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	backend := carmen.NewMockNonCommittableStateDB(ctrl)
	backend2 := carmen.NewMockNonCommittableStateDB(ctrl)
	backend.EXPECT().Copy().Return(backend2)

	state := createNonCommittableCarmenStateDb(backend)

	copied := state.Copy()
	copiedDb, isOfProperType := copied.(*CarmenStateDB)
	require.True(isOfProperType)
	require.Same(copiedDb.db, backend2)
	require.False(copiedDb.committable)
}

func TestCarmenStateDB_Copy_PanicsForCommittableStateDB(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	backend := carmen.NewMockStateDB(ctrl)
	state := CreateCarmenStateDb(backend)
	require.PanicsWithValue(
		"unable to copy committable (live) StateDB",
		func() { state.Copy() },
	)
}

func TestCarmenStateDB_EndBlock_CallsEndBlockOnCommittableStateDB(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	backend := carmen.NewMockStateDB(ctrl)
	state := CreateCarmenStateDb(backend)

	// Expect EndBlock to be called on the backend when EndBlock is called on the state.
	errChan := make(chan error)
	backend.EXPECT().EndBlock(uint64(0)).Return(errChan)
	got := state.EndBlock(0)
	require.NotNil(got)
	require.EqualValues(errChan, got)
}

func TestCarmenStateDB_EndBlock_IsNoOpForNonCommittableStateDB(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	backend := carmen.NewMockNonCommittableStateDB(ctrl)
	state := createNonCommittableCarmenStateDb(backend)

	// EndBlock should be a no-op for non-committable state DBs, so it should
	// not call the backend's EndBlock method.
	errChan := state.EndBlock(0)
	require.Nil(errChan)
}
