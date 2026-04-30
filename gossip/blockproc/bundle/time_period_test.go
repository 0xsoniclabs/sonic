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

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestMakeUnrestrictedTimePeriod_CreatesTheMaximumTimePeriod(t *testing.T) {
	period := MakeUnrestrictedTimePeriod()
	require.Zero(t, period.Start)
	require.Equal(t, uint64(math.MaxUint64), period.Duration)
}

func TestTimePeriod_IsInPeriod_ReturnsTrueIfTimeIsInPeriod(t *testing.T) {
	tests := map[string]struct {
		period TimePeriod
		time   inter.Timestamp
		want   bool
	}{
		"within range": {
			period: TimePeriod{Start: 10, Duration: 10},
			time:   15,
			want:   true,
		},
		"at start": {
			period: TimePeriod{Start: 10, Duration: 20},
			time:   10,
			want:   true,
		},
		"at end": {
			period: TimePeriod{Start: 10, Duration: 20},
			time:   29,
			want:   true,
		},
		"before start": {
			period: TimePeriod{Start: 10, Duration: 20},
			time:   9,
			want:   false,
		},
		"after end": {
			period: TimePeriod{Start: 10, Duration: 20},
			time:   30,
			want:   false,
		},
		"single timestamp": {
			period: TimePeriod{Start: 10, Duration: 1},
			time:   10,
			want:   true,
		},
		"single timestamp before": {
			period: TimePeriod{Start: 10, Duration: 1},
			time:   9,
			want:   false,
		},
		"single timestamp after": {
			period: TimePeriod{Start: 10, Duration: 1},
			time:   11,
			want:   false,
		},
		"empty period before": {
			period: TimePeriod{Start: 20, Duration: 0},
			time:   19,
			want:   false,
		},
		"empty period after": {
			period: TimePeriod{Start: 20, Duration: 0},
			time:   20,
			want:   false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := test.period.IsInPeriod(test.time)
			require.Equal(t, test.want, got)
		})
	}
}

func TestTimePeriod_IsBeforePeriod_ReturnsTrueIfTimeIsBeforePeriod(t *testing.T) {
	for start := range inter.Timestamp(10) {
		for cur := range inter.Timestamp(10) {
			r := TimePeriod{Start: start, Duration: 10}
			want := cur < start
			got := r.IsBeforePeriod(cur)
			require.Equal(t, want, got)
		}
	}
}

func TestTimePeriod_IsAfterPeriod_ReturnsTrueIfTimeIsAfterPeriod(t *testing.T) {
	for start := range inter.Timestamp(10) {
		for duration := range uint64(10) {
			for cur := range inter.Timestamp(10) {
				r := TimePeriod{Start: start, Duration: duration}
				want := cur >= start+inter.Timestamp(duration)
				got := r.IsAfterPeriod(cur)
				require.Equal(t, want, got)
			}
		}
	}
}

func TestTimePeriod_IsAfterPeriod_HandlesOverflow(t *testing.T) {
	p := TimePeriod{Start: math.MaxUint64 - 1, Duration: 2}
	require.Zero(t, p.Start+inter.Timestamp(p.Duration)) // overflow occurs
	require.False(t, p.IsAfterPeriod(math.MaxUint64))
	require.False(t, p.IsAfterPeriod(math.MaxUint64-1))
	require.False(t, p.IsAfterPeriod(0))
}

func TestTimePeriod_EncodingAndDecodingIsAligned(t *testing.T) {
	require := require.New(t)
	tests := []TimePeriod{
		{0, 0}, {10, 20}, {20, 10},
		{0, math.MaxUint16}, {math.MaxUint64, 0},
		{math.MaxUint64, math.MaxUint16},
		MakeUnrestrictedTimePeriod(),
	}

	for _, cur := range tests {
		var buf bytes.Buffer
		require.NoError(cur.encode(&buf))

		var decoded TimePeriod
		require.NoError(decoded.decode(&buf))
		require.Equal(cur, decoded)
	}
}

func TestTimePeriod_encode_encodesValuesUsingRlp(t *testing.T) {
	require := require.New(t)
	tests := []TimePeriod{
		{0, 0}, {10, 20}, {20, 10},
		{0, math.MaxUint16}, {math.MaxUint64, 0},
		{math.MaxUint64, math.MaxUint16},
	}

	for _, cur := range tests {
		var buf bytes.Buffer
		require.NoError(cur.encode(&buf))

		type pair struct {
			A inter.Timestamp
			B uint64
		}

		type single struct { // < for the empty-wrapper
			P pair
		}

		want, err := rlp.EncodeToBytes(single{pair{cur.Start, cur.Duration}})
		require.NoError(err)

		require.Equal(want[:], buf.Bytes())
	}
}

func TestTimePeriod_encode_UnrestrictedHasCompactEncoding(t *testing.T) {
	require := require.New(t)

	encode := func(p TimePeriod) []byte {
		var res bytes.Buffer
		require.NoError(p.encode(&res))
		return res.Bytes()
	}

	empty := encode(TimePeriod{})
	unrestricted := encode(MakeUnrestrictedTimePeriod())
	restricted := encode(TimePeriod{
		Start:    math.MaxUint64,
		Duration: math.MaxUint64,
	})

	require.Len(empty, 4)        // 4 bytes for empty
	require.Len(unrestricted, 2) // 2 for no limits
	require.Len(restricted, 20)  // 20 bytes max
}

func TestTimePeriod_encode_FailingWriter_ReturnsIssue(t *testing.T) {
	ctrl := gomock.NewController(t)
	writer := NewMockWriter(ctrl)

	issue := fmt.Errorf("injected issue")
	writer.EXPECT().Write(gomock.Any()).Return(0, issue)

	r := TimePeriod{Start: 10, Duration: 20}
	err := r.encode(writer)
	require.ErrorIs(t, err, issue)
}

func TestTimePeriod_decode_ReadsRlpEncodedUint64Values(t *testing.T) {
	require := require.New(t)
	tests := []TimePeriod{
		{0, 0}, {10, 20}, {20, 10},
		{0, math.MaxUint16}, {math.MaxUint64, 0},
		{math.MaxUint64, math.MaxUint16},
	}

	for _, cur := range tests {
		type pair struct {
			A inter.Timestamp
			B uint64
		}

		type single struct { // < for the empty-wrapper
			P pair
		}

		data, err := rlp.EncodeToBytes(single{pair{cur.Start, cur.Duration}})
		require.NoError(err)

		var r TimePeriod
		err = r.decode(bytes.NewReader(data))
		require.NoError(err)
		require.Equal(cur, r)
	}
}

func TestTimePeriod_decode_FailingReader_ReturnsIssue(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)

	issue := fmt.Errorf("injected issue")
	reader.EXPECT().Read(gomock.Any()).Return(0, issue)

	var p TimePeriod
	err := p.decode(reader)
	require.ErrorIs(t, err, issue)
}
