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
	"go.uber.org/mock/gomock"
)

func TestMakeMaxRangeStartingAt_CreatesMaxRangeStartingAtGivenBlock(t *testing.T) {
	cases := map[string]struct {
		start          uint64
		expectedLatest uint64
		expectedSize   uint64
	}{
		"start at 0": {
			start:          0,
			expectedLatest: MaxBlockRange - 1,
			expectedSize:   MaxBlockRange,
		},
		"start at 1": {
			start:          1,
			expectedLatest: MaxBlockRange,
			expectedSize:   MaxBlockRange,
		},
		"start at 100": {
			start:          100,
			expectedLatest: 100 + MaxBlockRange - 1,
			expectedSize:   MaxBlockRange,
		},
		"start with max plus one blocks": {
			start:          math.MaxUint64 - MaxBlockRange - 1,
			expectedLatest: math.MaxUint64 - 2,
			expectedSize:   MaxBlockRange,
		},
		"start with max blocks": {
			start:          math.MaxUint64 - MaxBlockRange,
			expectedLatest: math.MaxUint64 - 1,
			expectedSize:   MaxBlockRange,
		},
		"start with exact left blocks": {
			start:          math.MaxUint64 - MaxBlockRange + 1,
			expectedLatest: math.MaxUint64,
			expectedSize:   MaxBlockRange,
		},
		"start with not enough blocks": {
			start:          math.MaxUint64 - MaxBlockRange + 2,
			expectedLatest: math.MaxUint64,
			expectedSize:   MaxBlockRange - 1,
		},
		"start with two blocks left": {
			start:          math.MaxUint64 - 1,
			expectedLatest: math.MaxUint64,
			expectedSize:   2,
		},
		"start with one block left": {
			start:          math.MaxUint64,
			expectedLatest: math.MaxUint64,
			expectedSize:   1,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			r := MakeMaxRangeStartingAt(c.start)
			require.Equal(t, c.start, r.Earliest)
			require.Equal(t, c.expectedLatest, r.Latest)
			require.Equal(t, c.expectedSize, r.Size())
		})
	}
}

func TestBlockRange_Size_ReturnsCorrectSize(t *testing.T) {
	tests := map[string]struct {
		blockRange BlockRange
		want       uint64
	}{
		"empty range": {
			blockRange: BlockRange{
				Earliest: 10,
				Latest:   9,
			},
			want: 0,
		},
		"single range": {
			blockRange: BlockRange{
				Earliest: 10,
				Latest:   10,
			},
			want: 1,
		},
		"two blocks range": {
			blockRange: BlockRange{
				Earliest: 10,
				Latest:   11,
			},
			want: 2,
		},
		"multiple blocks range": {
			blockRange: BlockRange{
				Earliest: 10,
				Latest:   20,
			},
			want: 11,
		},
		"large range": {
			blockRange: BlockRange{
				Earliest: 0,
				Latest:   10_000_000,
			},
			want: 10_000_001,
		},
		"large range with latest near max uint64": {
			blockRange: BlockRange{
				Earliest: 0,
				Latest:   math.MaxUint64 - 1,
			},
			want: math.MaxUint64,
		},
		"too large range is capped to prevent overflow": {
			blockRange: BlockRange{
				Earliest: 0,
				Latest:   math.MaxUint64,
			},
			want: math.MaxUint64,
		},
		"small range start near max uint64": {
			blockRange: BlockRange{
				Earliest: math.MaxUint64 - 10,
				Latest:   math.MaxUint64,
			},
			want: 11,
		},
		"small range with the last two blocks": {
			blockRange: BlockRange{
				Earliest: math.MaxUint64 - 1,
				Latest:   math.MaxUint64,
			},
			want: 2,
		},
		"single block range at max uint64": {
			blockRange: BlockRange{
				Earliest: math.MaxUint64,
				Latest:   math.MaxUint64,
			},
			want: 1,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.EqualValues(t, test.want, test.blockRange.Size())
		})
	}
}

func TestBlockRange_IsInRange_ReturnsTrueIfBlockNumberIsWithinRange(t *testing.T) {
	tests := map[string]struct {
		BlockRange BlockRange
		current    uint64
		want       bool
	}{
		"within range": {
			BlockRange: BlockRange{Earliest: 10, Latest: 20},
			current:    15,
			want:       true,
		},
		"at earliest": {
			BlockRange: BlockRange{Earliest: 10, Latest: 20},
			current:    10,
			want:       true,
		},
		"at latest": {
			BlockRange: BlockRange{Earliest: 10, Latest: 20},
			current:    20,
			want:       true,
		},
		"below range": {
			BlockRange: BlockRange{Earliest: 10, Latest: 20},
			current:    9,
			want:       false,
		},
		"above range": {
			BlockRange: BlockRange{Earliest: 10, Latest: 20},
			current:    21,
			want:       false,
		},
		"at lower end": {
			BlockRange: BlockRange{Earliest: 10, Latest: 20},
			current:    10,
			want:       true,
		},
		"at upper end": {
			BlockRange: BlockRange{Earliest: 10, Latest: 20},
			current:    20,
			want:       true,
		},
		"single block range": {
			BlockRange: BlockRange{Earliest: 10, Latest: 10},
			current:    10,
			want:       true,
		},
		"invalid range": {
			BlockRange: BlockRange{Earliest: 20, Latest: 10},
			current:    15,
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

func TestBlockRange_EncodingAndDecodingIsAligned(t *testing.T) {
	require := require.New(t)
	tests := []BlockRange{
		{0, 0}, {10, 20}, {20, 10},
		{0, math.MaxUint64}, {math.MaxUint64, 0},
		{math.MaxUint64, math.MaxUint64},
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
		{0, math.MaxUint64}, {math.MaxUint64, 0},
		{math.MaxUint64, math.MaxUint64},
	}

	for _, cur := range tests {
		var buf bytes.Buffer
		require.NoError(cur.encode(&buf))

		type pair struct {
			A, B uint64
		}

		want, err := rlp.EncodeToBytes(pair{cur.Earliest, cur.Latest})
		require.NoError(err)

		require.Equal(want[:], buf.Bytes())
	}
}

func TestBlockRange_encode_FailingWriter_ReturnsIssue(t *testing.T) {
	ctrl := gomock.NewController(t)
	writer := NewMockWriter(ctrl)

	issue := fmt.Errorf("injected issue")
	writer.EXPECT().Write(gomock.Any()).Return(0, issue)

	r := BlockRange{Earliest: 10, Latest: 20}
	err := r.encode(writer)
	require.ErrorIs(t, err, issue)
}

func TestBlockRange_decode_ReadsRlpEncodedUint64Values(t *testing.T) {
	require := require.New(t)
	tests := []BlockRange{
		{0, 0}, {10, 20}, {20, 10},
		{0, math.MaxUint64}, {math.MaxUint64, 0},
		{math.MaxUint64, math.MaxUint64},
	}

	for _, cur := range tests {
		type pair struct {
			A, B uint64
		}

		data, err := rlp.EncodeToBytes(pair{cur.Earliest, cur.Latest})
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
