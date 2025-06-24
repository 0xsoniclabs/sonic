// Copyright 2025 Sonic Operations Ltd
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
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetEffectiveGasLimit_IsProportionalToDelay(t *testing.T) {
	rates := []uint64{0, 1, 20, 1234, 10_000_000_000} // < gas/sec
	delay := []time.Duration{
		0, 1 * time.Nanosecond, 50 * time.Microsecond,
		100 * time.Millisecond, 1500 * time.Millisecond,
	}

	for _, rate := range rates {
		for _, d := range delay {
			got := GetEffectiveGasLimit(d, rate)
			want := rate * uint64(d) / uint64(time.Second)
			require.Equal(t, want, got, "rate %d, delay %v", rate, d)
		}
	}
}

func TestGetEffectiveGasLimit_IsZeroForNegativeDelay(t *testing.T) {
	require.Equal(t, uint64(0), GetEffectiveGasLimit(-1*time.Nanosecond, 100))
	require.Equal(t, uint64(0), GetEffectiveGasLimit(-1*time.Second, 100))
	require.Equal(t, uint64(0), GetEffectiveGasLimit(-1*time.Hour, 100))
}

func TestGetEffectiveGasLimit_IsCappedAtMaximumAccumulationTime(t *testing.T) {
	rate := uint64(100)
	maxAccumulationTime := maxAccumulationTime
	for _, d := range []time.Duration{
		maxAccumulationTime,
		maxAccumulationTime + 1*time.Nanosecond,
		maxAccumulationTime + 1*time.Second,
		maxAccumulationTime + 1*time.Hour,
	} {
		got := GetEffectiveGasLimit(d, rate)
		want := GetEffectiveGasLimit(maxAccumulationTime, rate)
		require.Equal(t, want, got, "delay %v", d)
	}
}
