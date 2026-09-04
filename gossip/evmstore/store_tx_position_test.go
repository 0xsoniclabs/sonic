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
	"math"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

func TestTxPosition_DecodeRLP_ReadsCurrentTwoFieldLayout(t *testing.T) {
	want := TxPosition{Block: 42, BlockOffset: 7}

	encoded, err := rlp.EncodeToBytes(&want)
	require.NoError(t, err)

	var got TxPosition
	require.NoError(t, rlp.DecodeBytes(encoded, &got))
	require.Equal(t, want, got)
}

func TestTxPosition_DecodeRLP_ReadsLegacyFourFieldLayout(t *testing.T) {
	// The historical layout persisted four fields:
	// [Block, Event, EventOffset, BlockOffset]. Such records must still decode,
	// keeping Block (first) and BlockOffset (last) and dropping the event fields.
	legacy := struct {
		Block       idx.Block
		Event       hash.Event
		EventOffset uint32
		BlockOffset uint32
	}{
		Block:       42,
		Event:       hash.HexToEventHash("0xdeadbeef"),
		EventOffset: 3,
		BlockOffset: 7,
	}

	encoded, err := rlp.EncodeToBytes(&legacy)
	require.NoError(t, err)

	var got TxPosition
	require.NoError(t, rlp.DecodeBytes(encoded, &got))
	require.Equal(t, TxPosition{Block: 42, BlockOffset: 7}, got)
}

func TestTxPosition_DecodeRLP_RejectsInvalidEncodings(t *testing.T) {
	encode := func(v any) []byte {
		encoded, err := rlp.EncodeToBytes(v)
		require.NoError(t, err)
		return encoded
	}

	tests := map[string]struct {
		encoded []byte
	}{
		"not a list":   {encode(uint32(1))},
		"no fields":    {encode([]uint32{})},
		"one field":    {encode([]uint32{1})},
		"three fields": {encode([]uint32{1, 2, 3})},
		"five fields":  {encode([]uint32{1, 2, 3, 4, 5})},
		"block is not an integer": {encode(struct {
			Block       []uint32
			BlockOffset uint32
		}{[]uint32{1}, 7})},
		"block offset overflows uint32": {encode(struct {
			Block       idx.Block
			BlockOffset uint64
		}{42, math.MaxUint32 + 1})},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var got TxPosition
			require.Error(t, rlp.DecodeBytes(test.encoded, &got))
		})
	}
}
