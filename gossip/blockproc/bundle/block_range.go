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
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/rlp"
)

const (
	// MaxBlockRangeLength is the maximum allowed block range length being
	// specified in execution plans of bundles. The limit is needed to limit
	// the state information required to be maintained by validators for
	// tracking processed execution plans.
	MaxBlockRangeLength = uint64(1024)
)

// BlockRange defines a range of blocks. The range is defined by a first block
// and the length of the range. All block numbers that satisfy
//
//	First <= blockNum < First + Length
//
// are included in the range. If Length is 0, the range is empty and does not
// include any block numbers.
type BlockRange struct {
	First  uint64
	Length uint64
}

// MakeMaxRangeStartingAt creates a block range of maximum allowed size,
// starting at the given block number. The resulting range may implicitly cover
// blocks beyond math.MaxUint64 if the start block is high enough. The length
// is not adjusted to cap block ranges at math.MaxUint64.
func MakeMaxRangeStartingAt(first uint64) BlockRange {
	return BlockRange{
		First:  first,
		Length: MaxBlockRangeLength,
	}
}

// IsInRange checks if the given block number is within this block range.
// The range is the half-open interval [First, First+Length): a block number
// is in range if First <= blockNum and blockNum is strictly less than the end
// of the range. If Length is 0, the range is empty and no block number is in
// range. The implementation avoids computing First+Length directly, so the
// result remains correct even if that sum would overflow uint64.
func (r BlockRange) IsInRange(blockNum uint64) bool {
	return !r.IsBeforeRange(blockNum) && !r.IsAfterRange(blockNum)
}

// IsBeforeRange checks if the given block number is before this block range,
// meaning that it is less than the first block number of the range.
func (r BlockRange) IsBeforeRange(blockNum uint64) bool {
	return blockNum < r.First
}

// IsAfterRange checks if the given block number is after this block range.
func (r BlockRange) IsAfterRange(blockNum uint64) bool {
	// r.First + r.Length may overflow, so we subtract the first
	return blockNum >= r.First && blockNum-r.First >= r.Length
}

func (r BlockRange) String() string {
	return fmt.Sprintf("[%d,+%d)", r.First, r.Length)
}

func (r BlockRange) encode(writer io.Writer) error {
	return rlp.Encode(writer, r)
}

func (r *BlockRange) decode(reader io.Reader) error {
	return rlp.Decode(reader, r)
}
