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

package bundle

import (
	"bytes"
	"fmt"
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestMakeMaxRangeStartingAt_CreatesMaxRangeStartingAtGivenBlock(t *testing.T) {
	starts := []uint64{0, 1, 100, math.MaxUint64}

	for _, start := range starts {
		r := MakeMaxRangeStartingAt(start)
		require.Equal(t, start, r.First)
		require.Equal(t, MaxBlockRangeLength, r.Length)
	}
}

func TestBlockRange_IsInRange_ReturnsTrueIfBlockNumberIsWithinRange(t *testing.T) {
	tests := map[string]struct {
		BlockRange BlockRange
		current    uint64
		want       bool
	}{
		"within range": {
			BlockRange: BlockRange{First: 10, Length: 10},
			current:    15,
			want:       true,
		},
		"at earliest": {
			BlockRange: BlockRange{First: 10, Length: 20},
			current:    10,
			want:       true,
		},
		"at latest": {
			BlockRange: BlockRange{First: 10, Length: 20},
			current:    29,
			want:       true,
		},
		"below range": {
			BlockRange: BlockRange{First: 10, Length: 20},
			current:    9,
			want:       false,
		},
		"above range": {
			BlockRange: BlockRange{First: 10, Length: 20},
			current:    30,
			want:       false,
		},
		"at lower end": {
			BlockRange: BlockRange{First: 10, Length: 20},
			current:    10,
			want:       true,
		},
		"at upper end": {
			BlockRange: BlockRange{First: 10, Length: 20},
			current:    29,
			want:       true,
		},
		"single block range": {
			BlockRange: BlockRange{First: 10, Length: 1},
			current:    10,
			want:       true,
		},
		"single block range before": {
			BlockRange: BlockRange{First: 10, Length: 1},
			current:    9,
			want:       false,
		},
		"single block range after": {
			BlockRange: BlockRange{First: 10, Length: 1},
			current:    11,
			want:       false,
		},
		"empty range before": {
			BlockRange: BlockRange{First: 20, Length: 0},
			current:    19,
			want:       false,
		},
		"empty range after": {
			BlockRange: BlockRange{First: 20, Length: 0},
			current:    20,
			want:       false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := test.BlockRange.IsInRange(test.current)
			require.Equal(t, test.want, got)
		})
	}
}

func TestBlockRange_IsBeforeRange_ReturnsTrueIfBlockNumberIsBeforeRange(t *testing.T) {
	for first := range uint64(10) {
		for cur := range uint64(10) {
			r := BlockRange{First: first, Length: 10}
			want := cur < first
			got := r.IsBeforeRange(cur)
			require.Equal(t, want, got)
		}
	}
}

func TestBlockRange_IsAfterRange_ReturnsTrueIfBlockNumberIsAfterRange(t *testing.T) {
	for first := range uint64(10) {
		for length := range uint64(10) {
			for cur := range uint64(10) {
				r := BlockRange{First: first, Length: length}
				want := cur >= first+length
				got := r.IsAfterRange(cur)
				require.Equal(t, want, got)
			}
		}
	}
}

func TestBlockRange_IsAfterRange_HandlesOverflow(t *testing.T) {
	r := BlockRange{First: math.MaxUint64 - 1, Length: 2}
	require.Zero(t, r.First+uint64(r.Length)) // overflow occurs
	require.False(t, r.IsAfterRange(math.MaxUint64))
	require.False(t, r.IsAfterRange(math.MaxUint64-1))
	require.False(t, r.IsAfterRange(0))
}

func TestBlockRange_String_ReturnsFormattedString(t *testing.T) {
	tests := map[string]struct {
		BlockRange BlockRange
		want       string
	}{
		"zero range": {
			BlockRange: BlockRange{First: 0, Length: 0},
			want:       "[0,+0)",
		},
		"non-zero range": {
			BlockRange: BlockRange{First: 10, Length: 20},
			want:       "[10,+20)",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := test.BlockRange.String()
			require.Equal(t, test.want, got)
		})
	}
}

func TestBlockRange_EncodingAndDecodingIsAligned(t *testing.T) {
	require := require.New(t)
	tests := []BlockRange{
		{0, 0}, {10, 20}, {20, 10},
		{0, math.MaxUint16}, {math.MaxUint64, 0},
		{math.MaxUint64, math.MaxUint16},
	}

	for _, cur := range tests {
		var buf bytes.Buffer
		require.NoError(cur.encode(&buf))

		var decoded BlockRange
		require.NoError(decoded.decode(&buf))
		require.Equal(cur, decoded)
	}
}

func TestBlockRange_encode_encodesBoundsUsingRlp(t *testing.T) {
	require := require.New(t)
	tests := []BlockRange{
		{0, 0}, {10, 20}, {20, 10},
		{0, math.MaxUint16}, {math.MaxUint64, 0},
		{math.MaxUint64, math.MaxUint16},
	}

	for _, cur := range tests {
		var buf bytes.Buffer
		require.NoError(cur.encode(&buf))

		type pair struct {
			A, B uint64
		}

		want, err := rlp.EncodeToBytes(pair{cur.First, cur.Length})
		require.NoError(err)

		require.Equal(want[:], buf.Bytes())
	}
}

func TestBlockRange_encode_FailingWriter_ReturnsIssue(t *testing.T) {
	ctrl := gomock.NewController(t)
	writer := NewMockWriter(ctrl)

	issue := fmt.Errorf("injected issue")
	writer.EXPECT().Write(gomock.Any()).Return(0, issue)

	r := BlockRange{First: 10, Length: 20}
	err := r.encode(writer)
	require.ErrorIs(t, err, issue)
}

func TestBlockRange_decode_ReadsRlpEncodedUint64Values(t *testing.T) {
	require := require.New(t)
	tests := []BlockRange{
		{0, 0}, {10, 20}, {20, 10},
		{0, math.MaxUint16}, {math.MaxUint64, 0},
		{math.MaxUint64, math.MaxUint16},
	}

	for _, cur := range tests {
		type pair struct {
			A, B uint64
		}

		data, err := rlp.EncodeToBytes(pair{cur.First, cur.Length})
		require.NoError(err)

		var r BlockRange
		err = r.decode(bytes.NewReader(data))
		require.NoError(err)
		require.Equal(cur, r)
	}
}

func TestBlockRange_decode_FailingReader_ReturnsIssue(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)

	issue := fmt.Errorf("injected issue")
	reader.EXPECT().Read(gomock.Any()).Return(0, issue)

	var r BlockRange
	err := r.decode(reader)
	require.ErrorIs(t, err, issue)
}
