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
// time of maxAccumulationTime and the given block limit.
func GetEffectiveGasLimit(
	delta time.Duration,
	targetedThroughput uint64,
	blockLimit uint64,
) uint64 {
	if delta <= 0 {
		return 0
	}
	if delta > maxAccumulationTime {
		delta = maxAccumulationTime
	}
	return min(blockLimit, new(big.Int).Div(
		new(big.Int).Mul(
			big.NewInt(int64(targetedThroughput)),
			big.NewInt(int64(delta.Nanoseconds())),
		),
		big.NewInt(int64(time.Second.Nanoseconds())),
	).Uint64())
}
