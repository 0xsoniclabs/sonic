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
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_GetBundleInfo_UnknownBundle_ReturnsNonErrorEmptyAnswer(t *testing.T) {

	ctr := gomock.NewController(t)
	be := NewMockBundleApiBackend(ctr)
	api := NewPublicBundleAPI(be)

	hash := common.Hash{123}
	be.EXPECT().GetBundleExecutionInfo(hash)
	res, err := api.GetBundleInfo(t.Context(), hash)
	require.NoError(t, err)
	require.Nil(t, res)
}

func Test_GetBundleInfo_KnownBundle_ReturnsInfo(t *testing.T) {

	ctr := gomock.NewController(t)
	be := NewMockBundleApiBackend(ctr)
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
	require.EqualValues(t, 1, uint64(res.Position))
	require.EqualValues(t, 2, uint64(res.Count))
}

func Test_GetBundleInfo_ReturnsEthereumConformantJSON(t *testing.T) {
	expectJsonEqual(t, `{
		  "block":"0x7b",
		  "position":"0xa",
		  "count":"0xb"
	}`,
		RPCBundleInfo{
			Block:    rpc.BlockNumber(123),
			Position: hexutil.Uint(10),
			Count:    hexutil.Uint(11),
		})
}

func expectJsonEqual[T any](t testing.TB, expected string, value T) {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err, "failed to marshal BundleRPCInfo to JSON")

	var j1, j2 T
	err = json.Unmarshal(encoded, &j1)
	require.NoError(t, err, "failed to unmarshal JSON back to %T", value)
	err = json.Unmarshal([]byte(expected), &j2)
	require.NoError(t, err, "failed to unmarshal JSON back to %T", value)
	v := reflect.DeepEqual(j1, j2)
	expected = strings.ReplaceAll(expected, " ", "")
	expected = strings.ReplaceAll(expected, "\n", "")
	expected = strings.ReplaceAll(expected, "\t", "")
	if !v {
		fmt.Println("Expected JSON", expected)
		fmt.Println("Actual   JSON", string(encoded))
		t.FailNow()
	}
}
