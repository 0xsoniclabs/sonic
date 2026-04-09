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
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_GetBundleInfo_UnknownBundle_ReturnsNonErrorEmptyAnswer(t *testing.T) {

	ctr := gomock.NewController(t)
	be := NewMockbackend(ctr)
	api := NewPublicBundleAPI(be)

	hash := common.Hash{123}
	be.EXPECT().GetBundleExecutionInfo(hash)
	res, err := api.GetBundleInfo(t.Context(), hash)
	require.NoError(t, err)
	require.Nil(t, res)
}

func Test_GetBundleInfo_KnownBundle_ReturnsInfo(t *testing.T) {

	ctr := gomock.NewController(t)
	be := NewMockbackend(ctr)
	api := NewPublicBundleAPI(be)

	hash := common.Hash{123}
	be.EXPECT().GetBundleExecutionInfo(hash).Return(&bundle.ExecutionInfo{
		BlockNumber: 123,
		Position: bundle.PositionInBlock{
			Offset: 1,
			Count:  2,
		},
	})
	res, err := api.GetBundleInfo(t.Context(), hash)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.EqualValues(t, 123, res.Block.Int64())
	require.EqualValues(t, 1, uint64(*res.Position))
	require.EqualValues(t, 2, uint64(*res.Count))
}
