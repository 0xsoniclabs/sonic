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

package sonicapi

import (
	"errors"
	"testing"

	"github.com/0xsoniclabs/carmen/go/common/amount"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_TotalSupply_ReturnsValuesOfTheSameBlock(t *testing.T) {
	ctr := gomock.NewController(t)
	be := NewMockBundleApiBackend(ctr)
	api := NewPublicBundleAPI(be)

	be.EXPECT().GetTotalSupply().Return(&evmstore.TotalSupplySnapshot{
		Block:       42,
		Total:       amount.New(1000),
		ZeroBalance: amount.New(30),
		DeadBalance: amount.New(12),
	}, nil)

	res, err := api.TotalSupply(t.Context())
	require.NoError(t, err)
	require.EqualValues(t, 42, res.BlockNumber)
	require.EqualValues(t, 1000, (*uint256.Int)(res.TotalSupply).Uint64())
	require.EqualValues(t, 958, (*uint256.Int)(res.TotalSupplyWithoutBurned).Uint64())
	require.EqualValues(t, 30, (*uint256.Int)(res.BurnedZero).Uint64())
	require.EqualValues(t, 12, (*uint256.Int)(res.BurnedDead).Uint64())
}

func Test_TotalSupply_BackendError_IsForwarded(t *testing.T) {
	ctr := gomock.NewController(t)
	be := NewMockBundleApiBackend(ctr)
	api := NewPublicBundleAPI(be)

	issue := errors.New("no snapshot")
	be.EXPECT().GetTotalSupply().Return(nil, issue)

	_, err := api.TotalSupply(t.Context())
	require.ErrorIs(t, err, issue)
}

func Test_TotalSupply_BurnedExceedsTotal_ReturnsError(t *testing.T) {
	ctr := gomock.NewController(t)
	be := NewMockBundleApiBackend(ctr)
	api := NewPublicBundleAPI(be)

	be.EXPECT().GetTotalSupply().Return(&evmstore.TotalSupplySnapshot{
		Total:       amount.New(10),
		ZeroBalance: amount.New(8),
		DeadBalance: amount.New(8),
	}, nil)

	_, err := api.TotalSupply(t.Context())
	require.ErrorContains(t, err, "exceeds the total supply")
}

func Test_TotalSupply_BurnedOverflow_ReturnsError(t *testing.T) {
	ctr := gomock.NewController(t)
	be := NewMockBundleApiBackend(ctr)
	api := NewPublicBundleAPI(be)

	be.EXPECT().GetTotalSupply().Return(&evmstore.TotalSupplySnapshot{
		Total:       amount.Max(),
		ZeroBalance: amount.Max(),
		DeadBalance: amount.New(1),
	}, nil)

	_, err := api.TotalSupply(t.Context())
	require.ErrorContains(t, err, "overflows")
}
