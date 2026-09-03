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

package evmcore

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestToEvmHeader_ReportsTheBlocksDifficulty(t *testing.T) {
	for _, difficulty := range []uint64{0, 1, 0x20000} {
		block := inter.NewBlockBuilder().
			WithNumber(1).
			WithDifficulty(difficulty).
			Build()

		header := ToEvmHeader(block, common.Hash{}, opera.FakeNetRules(opera.GetBrioUpgrades()))
		require.Equal(t, new(big.Int).SetUint64(difficulty), header.Difficulty)
	}
}

func TestEvmHeader_DifficultyIsReportedByTheEthereumHeaderAndTheJsonEncoding(t *testing.T) {
	tests := map[string]struct {
		difficulty *big.Int
		want       *big.Int
	}{
		// The difficulty is a required field of both encodings, so a header
		// without one must still report a value.
		"unset":     {nil, big.NewInt(0)},
		"zero":      {big.NewInt(0), big.NewInt(0)},
		"pre-merge": {big.NewInt(0x20000), big.NewInt(0x20000)},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			header := &EvmHeader{
				Number:     big.NewInt(1),
				Difficulty: test.difficulty,
			}

			require.Equal(t, test.want, header.EthHeader().Difficulty)
			require.Equal(t, (*hexutil.Big)(test.want), header.ToJson(nil).Difficulty)
		})
	}
}

func TestConvertFromEthHeader_CarriesTheDifficulty(t *testing.T) {
	source := &EvmHeader{
		Number:     big.NewInt(1),
		Difficulty: big.NewInt(0x20000),
	}
	header := ConvertFromEthHeader(source.EthHeader())
	require.Equal(t, big.NewInt(0x20000), header.Difficulty)
}

func TestEvmHeader_ToJson_ReportsThePostCancunHeaderFields(t *testing.T) {
	// Sonic runs Cancun and Prague rules, so a client reading block JSON
	// structurally expects these fields. Sonic has neither a beacon chain nor
	// execution layer requests, so it states that rather than omitting them.
	header := &EvmHeader{Number: big.NewInt(1)}

	json := header.ToJson(nil)
	require.Equal(t, &common.Hash{}, json.ParentBeaconBlockRoot)
	require.Equal(t, &types.EmptyRequestsHash, json.RequestsHash)
}

func TestEvmHeader_ToJson_EncodesThePostCancunFieldsAsHashes(t *testing.T) {
	// The specification permits no null for either field: both must be 32 bytes
	// of hex when present.
	header := &EvmHeader{Number: big.NewInt(1)}

	encoded, err := json.Marshal(header.ToJson(nil))
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	require.Equal(t, "0x0000000000000000000000000000000000000000000000000000000000000000", decoded["parentBeaconBlockRoot"])
	require.Equal(t, types.EmptyRequestsHash.Hex(), decoded["requestsHash"])
}
