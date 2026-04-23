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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_GetBundleHistoryHash_InitialState_ReturnsZeroBlockAndHash(t *testing.T) {
	ctr := gomock.NewController(t)
	be := NewMockBundleApiBackend(ctr)
	api := NewPublicBundleAPI(be)

	be.EXPECT().GetProcessedBundleHistoryHash().Return(uint64(0), common.Hash{})

	res, err := api.GetBundleHistoryHash(t.Context())
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, hexutil.Uint64(0), res.Block)
	require.Equal(t, common.Hash{}, res.Hash)
}

func Test_GetBundleHistoryHash_NonZeroState_ReturnsCorrectBlockAndHash(t *testing.T) {
	ctr := gomock.NewController(t)
	be := NewMockBundleApiBackend(ctr)
	api := NewPublicBundleAPI(be)

	expectedBlock := uint64(42)
	expectedHash := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	be.EXPECT().GetProcessedBundleHistoryHash().Return(expectedBlock, expectedHash)

	res, err := api.GetBundleHistoryHash(t.Context())
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, hexutil.Uint64(expectedBlock), res.Block)
	require.Equal(t, expectedHash, res.Hash)
}

func Test_GetBundleHistoryHash_ReturnsEthereumConformantJSON(t *testing.T) {
	expectJsonEqual(t, `{
		"block": "0x2a",
		"hash":  "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	}`,
		RPCBundleHistoryHash{
			Block: hexutil.Uint64(42),
			Hash:  common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
		})
}
