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
	"errors"
	"testing"

	cc "github.com/0xsoniclabs/carmen/go/common"
	"github.com/0xsoniclabs/carmen/go/common/amount"
	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestStore_UpdateTotalSupply_StoresSnapshotOfGivenBlock(t *testing.T) {
	ctrl := gomock.NewController(t)
	state := carmen.NewMockState(ctrl)
	stateDb := carmen.NewMockStateDB(ctrl)
	store := &Store{carmenState: state, liveStateDb: stateDb}

	stateDb.EXPECT().Check().Return(nil)
	state.EXPECT().TotalSupply().Return(amount.New(100), nil)
	state.EXPECT().GetBalance(cc.Address{}).Return(amount.New(3), nil)
	state.EXPECT().GetBalance(burnAddress).Return(amount.New(5), nil)

	require.NoError(t, store.UpdateTotalSupply(7))
	snapshot, err := store.GetTotalSupply()
	require.NoError(t, err)
	require.Equal(t, &TotalSupplySnapshot{
		Block:       7,
		Total:       amount.New(100),
		ZeroBalance: amount.New(3),
		DeadBalance: amount.New(5),
	}, snapshot)
}

func TestStore_UpdateTotalSupply_FailingLiveStateDbCheck_ClearsSnapshot(t *testing.T) {
	ctrl := gomock.NewController(t)
	state := carmen.NewMockState(ctrl)
	stateDb := carmen.NewMockStateDB(ctrl)
	store := &Store{carmenState: state, liveStateDb: stateDb}
	store.totalSupply.Store(&TotalSupplySnapshot{Block: 6})

	// The state must not be read, since it may not reflect the given block.
	issue := errors.New("injected issue")
	stateDb.EXPECT().Check().Return(issue)

	require.ErrorIs(t, store.UpdateTotalSupply(7), issue)
	_, err := store.GetTotalSupply()
	require.ErrorIs(t, err, ErrTotalSupplyUnavailable)
}

func TestStore_UpdateTotalSupply_ClosedStore_ReturnsError(t *testing.T) {
	store := &Store{}
	require.ErrorContains(t, store.UpdateTotalSupply(7), "not open")
	_, err := store.GetTotalSupply()
	require.ErrorIs(t, err, ErrTotalSupplyUnavailable)
}
