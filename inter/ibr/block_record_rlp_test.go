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

package ibr_test

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/ibr"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

// Verify that [] is equivalent to nil for log's Topics in the genesis file
func TestNilTopicsDoesNotMatterInGenesis(t *testing.T) {
	rlp1, err := rlp.EncodeToBytes(ibr.LlrIdxFullBlockRecord{
		LlrFullBlockRecord: ibr.LlrFullBlockRecord{
			Receipts: []*types.ReceiptForStorage{
				{
					Logs: []*types.Log{
						{
							Topics: nil,
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	rlp2, err := rlp.EncodeToBytes(ibr.LlrIdxFullBlockRecord{
		LlrFullBlockRecord: ibr.LlrFullBlockRecord{
			Receipts: []*types.ReceiptForStorage{
				{
					Logs: []*types.Log{
						{
							Topics: []common.Hash{},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rlp1, rlp2) {
		t.Errorf("serialized byte slices does not match: %x != %x", rlp1, rlp2)
	}
}

// LegacyBlockRecord is the block record as encoded before gas used overrides
// were added. Genesis files containing such records must remain readable, and
// their serialization must not change. The type is exported since RLP skips
// unexported embedded fields.
type LegacyBlockRecord struct {
	BlockHash  hash.Hash
	ParentHash hash.Hash
	StateRoot  hash.Hash
	Time       inter.Timestamp
	Duration   uint64
	Difficulty uint64
	GasLimit   uint64
	GasUsed    uint64
	BaseFee    *big.Int
	PrevRandao hash.Hash
	Epoch      idx.Epoch
	Txs        types.Transactions
	Receipts   []*types.ReceiptForStorage
}

type LegacyIdxBlockRecord struct {
	LegacyBlockRecord
	Idx idx.Block
}

func TestBlockRecordRlp_RecordsWithoutOverridesKeepLegacyEncoding(t *testing.T) {
	to := common.Address{1}
	legacy := LegacyIdxBlockRecord{
		LegacyBlockRecord: LegacyBlockRecord{
			BlockHash: hash.Hash{1},
			GasUsed:   42,
			BaseFee:   big.NewInt(7),
			Txs:       types.Transactions{types.NewTx(&types.LegacyTx{To: &to})},
			Receipts:  []*types.ReceiptForStorage{{CumulativeGasUsed: 42, Logs: []*types.Log{}}},
		},
		Idx: 5,
	}
	legacyEncoded, err := rlp.EncodeToBytes(legacy)
	require.NoError(t, err)

	current := ibr.LlrIdxFullBlockRecord{
		LlrFullBlockRecord: ibr.LlrFullBlockRecord{
			BlockHash: legacy.BlockHash,
			GasUsed:   legacy.GasUsed,
			BaseFee:   legacy.BaseFee,
			Txs:       legacy.Txs,
			Receipts:  legacy.Receipts,
		},
		Idx: legacy.Idx,
	}
	currentEncoded, err := rlp.EncodeToBytes(current)
	require.NoError(t, err)
	require.Equal(t, legacyEncoded, currentEncoded)

	// Legacy records can still be decoded.
	var decoded ibr.LlrIdxFullBlockRecord
	require.NoError(t, rlp.DecodeBytes(legacyEncoded, &decoded))
	require.Equal(t, legacy.Idx, decoded.Idx)
	require.Equal(t, legacy.GasUsed, decoded.GasUsed)
	require.Nil(t, decoded.GasUsedOverrides)
}

func TestBlockRecordRlp_GasUsedOverridesRoundTrip(t *testing.T) {
	tests := map[string][]ibr.GasUsedOverride{
		"single override":    {{Index: 1, GasUsed: 20}},
		"multiple overrides": {{Index: 1, GasUsed: 20}, {Index: 4, GasUsed: 0}},
		"large values":       {{Index: 1 << 40, GasUsed: 1 << 50}},
	}
	for name, overrides := range tests {
		t.Run(name, func(t *testing.T) {
			record := ibr.LlrIdxFullBlockRecord{
				LlrFullBlockRecord: ibr.LlrFullBlockRecord{
					BaseFee:          big.NewInt(1),
					GasUsedOverrides: overrides,
				},
				Idx: 9, // must survive being encoded behind the overrides
			}
			encoded, err := rlp.EncodeToBytes(record)
			require.NoError(t, err)

			var decoded ibr.LlrIdxFullBlockRecord
			require.NoError(t, rlp.DecodeBytes(encoded, &decoded))
			require.Equal(t, overrides, decoded.GasUsedOverrides)
			require.Equal(t, record.Idx, decoded.Idx)
		})
	}
}
