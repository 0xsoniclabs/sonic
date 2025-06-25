package inter

import (
	"math"
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
			got := GetEffectiveGasLimit(d, rate, math.MaxUint64)
			want := rate * uint64(d) / uint64(time.Second)
			require.Equal(t, want, got, "rate %d, delay %v", rate, d)
		}
	}
}

func TestGetEffectiveGasLimit_IsZeroForNegativeDelay(t *testing.T) {
	blockLimit := uint64(math.MaxUint64)
	require.Equal(t, uint64(0), GetEffectiveGasLimit(-1*time.Nanosecond, 100, blockLimit))
	require.Equal(t, uint64(0), GetEffectiveGasLimit(-1*time.Second, 100, blockLimit))
	require.Equal(t, uint64(0), GetEffectiveGasLimit(-1*time.Hour, 100, blockLimit))
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
		got := GetEffectiveGasLimit(d, rate, math.MaxUint64)
		want := GetEffectiveGasLimit(maxAccumulationTime, rate, math.MaxUint64)
		require.Equal(t, want, got, "delay %v", d)
	}
}

func TestGetEffectiveGasLimit_IsCappedByBlockGasLimit(t *testing.T) {
	delta := 100 * time.Millisecond
	rate := uint64(100_000)
	allocation := rate * uint64(delta) / uint64(time.Second)

	limits := []uint64{
		0,
		1,
		allocation - 1,
		allocation,
		allocation + 1,
		math.MaxUint64,
	}

	for _, blockLimit := range limits {
		got := GetEffectiveGasLimit(delta, rate, blockLimit)
		want := min(allocation, blockLimit)
		require.Equal(t, want, got, "block limit %d", blockLimit)
	}
}
