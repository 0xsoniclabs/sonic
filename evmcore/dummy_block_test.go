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

func TestEvmHeader_ToJson_HashOfDecodedHeaderMatchesBlockHash(t *testing.T) {
	// Clients such as go-ethereum's ethclient decode the block JSON into a
	// types.Header and recompute the block hash from it. Every optional header
	// field present in the JSON is part of that hash, so the JSON must carry
	// exactly the fields of Sonic's canonical header - no more, no less.
	block := inter.NewBlockBuilder().
		WithNumber(12).
		WithParentHash(common.Hash{0x01}).
		WithStateRoot(common.Hash{0x02}).
		WithTime(inter.Timestamp(1_700_000_000_123_456_789)).
		WithDuration(1234).
		WithGasLimit(30_000_000).
		WithGasUsed(21_000).
		WithBaseFee(big.NewInt(1_000_000_000)).
		WithPrevRandao(common.Hash{0x03}).
		WithEpoch(7).
		Build()

	header := ToEvmHeader(block, block.ParentHash, opera.FakeNetRules(opera.GetSonicUpgrades()))
	header.TxHash = block.TransactionsHashRoot

	encoded, err := json.Marshal(header.ToJson(types.Receipts{}))
	require.NoError(t, err)

	var decoded types.Header
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	require.Nil(t, decoded.ParentBeaconRoot, "parentBeaconBlockRoot is not part of Sonic's header")
	require.Nil(t, decoded.RequestsHash, "requestsHash is not part of Sonic's header")
	require.Equal(t, block.Hash(), decoded.Hash())
}
