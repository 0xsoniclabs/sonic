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

package chain_test

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/chain"
)

// inputs are client inputs of a block number or reference, valid and not.
var inputs = []string{
	`"latest"`, `"pending"`, `"earliest"`, `"safe"`, `"finalized"`,
	`"0x0"`, `"0x1"`, `"0x7fffffffffffffff"`, `"0x8000000000000000"`, `"0xffffffffffffffffff"`,
	`"0x"`, `"1"`, `"0x01"`, `"0xg"`, `"unknown"`, `1`, `null`, `true`, `{}`, `[]`,
	`"0x0000000000000000000000000000000000000000000000000000000000000001"`,
	`"0x000000000000000000000000000000000000000000000000000000000000000g"`,
	`{"blockNumber":"0x1"}`, `{"blockNumber":"latest"}`, `{"blockHash":"0x0000000000000000000000000000000000000000000000000000000000000002"}`,
	`{"blockNumber":"0x1","blockHash":"0x0000000000000000000000000000000000000000000000000000000000000002"}`,
	`{"blockNumber":"0x1","requireCanonical":true}`, `{"blockNumber":5}`,
}

func TestBlockNumber_DecodesAsGoEthereum(t *testing.T) {
	for _, input := range inputs {
		var ours chain.BlockNumber
		var theirs gethrpc.BlockNumber
		oursErr := json.Unmarshal([]byte(input), &ours)
		theirsErr := json.Unmarshal([]byte(input), &theirs)
		if theirsErr != nil {
			require.Error(t, oursErr, input)
			require.Equal(t, theirsErr.Error(), oursErr.Error(), input)
			continue
		}
		require.NoError(t, oursErr, input)
		require.Equal(t, int64(theirs), int64(ours), input)
		require.Equal(t, theirs.String(), ours.String(), input)
	}
}

func TestBlockReference_DecodesAsGoEthereum(t *testing.T) {
	for _, input := range inputs {
		var ours chain.BlockReference
		var theirs gethrpc.BlockNumberOrHash
		oursErr := json.Unmarshal([]byte(input), &ours)
		theirsErr := json.Unmarshal([]byte(input), &theirs)
		if theirsErr != nil {
			require.Error(t, oursErr, input)
			require.Equal(t, theirsErr.Error(), oursErr.Error(), input)
			continue
		}
		require.NoError(t, oursErr, input)
		theirNumber, theirHasNumber := theirs.Number()
		require.Equal(t, theirHasNumber, ours.Number != nil, input)
		if theirHasNumber {
			require.Equal(t, int64(theirNumber), int64(*ours.Number), input)
		}
		theirHash, theirHasHash := theirs.Hash()
		require.Equal(t, theirHasHash, ours.Hash != nil, input)
		if theirHasHash {
			require.Equal(t, theirHash, *ours.Hash, input)
		}
		require.Equal(t, theirs.RequireCanonical, ours.RequireCanonical, input)
		require.Equal(t, theirs.String(), ours.String(), input)
	}
}

func TestBlockNumber_ANumberAboveMaxInt64IsNoTag(t *testing.T) {
	var number chain.BlockNumber
	require.Error(t, json.Unmarshal([]byte(`"0xfffffffffffffffe"`), &number), "2^64-2 would be latest")
	require.False(t, chain.BlockNumber(0).IsTag())
	require.True(t, chain.LatestBlockNumber.IsTag())
}

func TestBlockNumber_EncodesTagsByName(t *testing.T) {
	encoded, err := json.Marshal([]chain.BlockNumber{chain.LatestBlockNumber, 16, chain.BlockNumber(-9)})
	require.NoError(t, err)
	require.Equal(t, "[\"latest\",\"0x10\",\"\\u003cinvalid -9\\u003e\"]", string(encoded))
}

func TestReferences_NameWhatTheySay(t *testing.T) {
	hash := common.HexToHash("0x01")
	require.Equal(t, hash.String(), chain.ReferenceToHash(hash).String())
	require.Equal(t, "latest", chain.Latest().String())
	require.Equal(t, "nil", chain.BlockReference{}.String())
}
