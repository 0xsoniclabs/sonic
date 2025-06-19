package inter

import (
	"math/big"
	"time"
)

// We put a strict cap of 2 second on the maximum time gas can be accumulated
// for a single block. Thus, if the delay between two blocks is less than 2
// seconds, gas is accumulated linearly. If the delay is longer than 2 seconds,
// we cap the gas to the maximum accumulation time. This is to limit the maximum
// block size to at most 2 seconds worth of gas.
const maxAccumulationTime = 2 * time.Second

// GetEffectiveGasLimit computes the effective gas limit for the next block.
// This is the time since the last block times the targeted network throughput.
// The result is capped to the gas that corresponds to a maximum accumulation
// time of maxAccumulationTime.
func GetEffectiveGasLimit(
	delta time.Duration,
	targetedThroughput uint64,
) uint64 {
	if delta <= 0 {
		return 0
	}
	if delta > maxAccumulationTime {
		delta = maxAccumulationTime
	}
	return new(big.Int).Div(
		new(big.Int).Mul(
			big.NewInt(int64(targetedThroughput)),
			big.NewInt(int64(delta.Nanoseconds())),
		),
		big.NewInt(int64(time.Second.Nanoseconds())),
	).Uint64()
}
