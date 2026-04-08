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

import "math"

const (
	// MaxBlockRange is the maximum allowed block range (Latest - Earliest) for
	// allowed for the validity period of a bundle.
	MaxBlockRange = uint64(1024)
)

// BlockRange represents a range of blocks, defined by an earliest and latest
// block number. The covered block range is a closed interval [Earliest, Latest],
// meaning that the earliest and latest blocks are included in the range.
// For instance, [0,0] is a valid block range that only includes the block
// number 0, while [0,1] includes both blocks 0 and 1. An interval where Latest
// is smaller than Earliest, such as [1,0], is a valid empty range.
type BlockRange struct {
	Earliest uint64
	Latest   uint64
}

// MakeMaxRangeStartingAt creates a block range of maximum allowed size, starting
// at the given block number.
func MakeMaxRangeStartingAt(blockNum uint64) BlockRange {
	latest := blockNum + MaxBlockRange - 1
	if blockNum > math.MaxUint64-MaxBlockRange {
		// if the starting block number is too close to maxUint64,
		// we cannot create a full range of MaxBlockRange blocks without overflowing.
		// In this case, we create the largest possible range starting at blockNum,
		// which ends at the maximum uint64 value.
		latest = math.MaxUint64
	}
	return BlockRange{
		Earliest: blockNum,
		Latest:   latest,
	}
}

// Size returns the size of the block range, which is the number of blocks
// included in the range.
func (r BlockRange) Size() uint64 {
	if r.Latest < r.Earliest {
		return 0
	}
	// overflow check
	if r.Earliest == 0 && r.Latest == math.MaxUint64 {
		return math.MaxUint64
	}
	return r.Latest - r.Earliest + 1
}

// IsInRange checks if the given block number is within this block range.
// The range is a closed interval [Earliest, Latest], meaning that blocks with
// numbers from Earliest through Latest (inclusive) are considered in range.
func (r BlockRange) IsInRange(blockNum uint64) bool {
	return blockNum >= r.Earliest && blockNum <= r.Latest
}
