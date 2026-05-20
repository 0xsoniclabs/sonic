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

package inter

import (
	"testing"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/utils/cser"
)

func TestBlockHashes_LastBlock(t *testing.T) {
	bh := BlockHashes{
		Start:  idx.Block(5),
		Hashes: make([]hash.Hash, 3),
	}
	require.Equal(t, idx.Block(7), bh.LastBlock())
}

func TestBlockHashes_Hash_IsDeterministic(t *testing.T) {
	bh := BlockHashes{
		Start:  idx.Block(1),
		Epoch:  idx.Epoch(10),
		Hashes: []hash.Hash{{1}, {2}, {3}},
	}
	h1 := bh.Hash()
	h2 := bh.Hash()
	require.Equal(t, h1, h2)
}

func TestBlockHashes_Hash_DiffersForDifferentInputs(t *testing.T) {
	base := BlockHashes{
		Start:  idx.Block(1),
		Epoch:  idx.Epoch(10),
		Hashes: []hash.Hash{{1}, {2}},
	}
	tests := map[string]BlockHashes{
		"different start": {
			Start:  idx.Block(2),
			Epoch:  idx.Epoch(10),
			Hashes: []hash.Hash{{1}, {2}},
		},
		"different epoch": {
			Start:  idx.Block(1),
			Epoch:  idx.Epoch(11),
			Hashes: []hash.Hash{{1}, {2}},
		},
		"different hashes": {
			Start:  idx.Block(1),
			Epoch:  idx.Epoch(10),
			Hashes: []hash.Hash{{1}, {3}},
		},
		"different length": {
			Start:  idx.Block(1),
			Epoch:  idx.Epoch(10),
			Hashes: []hash.Hash{{1}},
		},
	}
	baseHash := base.Hash()
	for name, other := range tests {
		t.Run(name, func(t *testing.T) {
			require.NotEqual(t, baseHash, other.Hash())
		})
	}
}

func TestBlockHashes_MarshalUnmarshalCSER_RoundTrip(t *testing.T) {
	tests := map[string]BlockHashes{
		"empty": {
			Start:  idx.Block(0),
			Epoch:  idx.Epoch(0),
			Hashes: []hash.Hash{},
		},
		"single hash": {
			Start:  idx.Block(100),
			Epoch:  idx.Epoch(5),
			Hashes: []hash.Hash{{1, 2, 3}},
		},
		"multiple hashes": {
			Start:  idx.Block(500),
			Epoch:  idx.Epoch(42),
			Hashes: []hash.Hash{{1}, {2}, {3}, {4}, {5}},
		},
	}

	for name, original := range tests {
		t.Run(name, func(t *testing.T) {
			data, err := cser.MarshalBinaryAdapter(original.MarshalCSER)
			require.NoError(t, err)

			var restored BlockHashes
			err = cser.UnmarshalBinaryAdapter(data, restored.UnmarshalCSER)
			require.NoError(t, err)

			require.Equal(t, original.Start, restored.Start)
			require.Equal(t, original.Epoch, restored.Epoch)
			require.Equal(t, len(original.Hashes), len(restored.Hashes))
			for i := range original.Hashes {
				require.Equal(t, original.Hashes[i], restored.Hashes[i])
			}
		})
	}
}

func TestBlockHashes_UnmarshalCSER_RejectsTooLargeAlloc(t *testing.T) {
	// Craft a CSER payload with an impossibly large hash count using
	// MarshalBinaryAdapter with a custom writer function.
	tooLargeNum := uint32(ProtocolMaxMsgSize/32 + 1)
	craftedData, err := cser.MarshalBinaryAdapter(func(w *cser.Writer) error {
		w.U64(1)           // start
		w.U32(1)           // epoch
		w.U32(tooLargeNum) // num (too large)
		return nil
	})
	require.NoError(t, err)

	var restored BlockHashes
	err = cser.UnmarshalBinaryAdapter(craftedData, restored.UnmarshalCSER)
	require.ErrorIs(t, err, cser.ErrTooLargeAlloc)
}
