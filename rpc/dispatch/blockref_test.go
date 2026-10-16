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

package dispatch

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestBlockNumber_Unmarshal(t *testing.T) {
	tests := map[string]BlockNumber{
		`"latest"`: LatestBlockNumber, `"pending"`: PendingBlockNumber, `"earliest"`: EarliestBlockNumber,
		`"safe"`: SafeBlockNumber, `"finalized"`: FinalizedBlockNumber, `"0x10"`: 16, `"0x0"`: 0, `0x1f`: 31,
	}
	for in, want := range tests {
		var bn BlockNumber
		require.NoError(t, bn.UnmarshalJSON([]byte(in)), in)
		require.Equal(t, want, bn, in)
	}
	for _, in := range []string{`"nope"`, `"10"`, `"0x"`, `"0x8000000000000000"`, `"0x01"`} {
		var bn BlockNumber
		require.Error(t, bn.UnmarshalJSON([]byte(in)), in)
	}
}

func TestBlockNumber_Marshal(t *testing.T) {
	for in, want := range map[BlockNumber]string{LatestBlockNumber: `"latest"`, 255: `"0xff"`, EarliestBlockNumber: `"earliest"`} {
		out, err := json.Marshal(in)
		require.NoError(t, err)
		require.Equal(t, want, string(out))
	}
}

func TestBlockRef_Unmarshal(t *testing.T) {
	hash := common.HexToHash("0xabc")
	var ref BlockRef
	require.NoError(t, json.Unmarshal([]byte(`"latest"`), &ref))
	require.Equal(t, RefLatest(), ref)

	ref = BlockRef{}
	require.NoError(t, json.Unmarshal([]byte(`"0x5"`), &ref))
	require.Equal(t, RefNumber(5), ref)

	ref = BlockRef{}
	require.NoError(t, json.Unmarshal([]byte(`"`+hash.Hex()+`"`), &ref))
	require.Equal(t, RefHash(hash), ref)

	ref = BlockRef{}
	require.NoError(t, json.Unmarshal([]byte(`{"blockHash":"`+hash.Hex()+`","requireCanonical":true}`), &ref))
	require.Equal(t, hash, *ref.Hash)
	require.True(t, ref.RequireCanonical)

	ref = BlockRef{}
	require.NoError(t, json.Unmarshal([]byte(`{"blockNumber":"0x7"}`), &ref))
	require.Equal(t, RefNumber(7), ref)

	for _, bad := range []string{
		`{"blockNumber":"0x1","blockHash":"` + hash.Hex() + `"}`, `"zz"`, `5`, `"0x8000000000000000"`,
	} {
		require.Error(t, json.Unmarshal([]byte(bad), &BlockRef{}), bad)
	}
}

func TestBlockRef_OrLatest(t *testing.T) {
	require.Equal(t, RefLatest(), OrLatest(nil))
	ref := RefNumber(3)
	require.Equal(t, ref, OrLatest(&ref))
}
