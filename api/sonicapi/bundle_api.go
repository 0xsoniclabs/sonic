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

package sonicapi

import (
	"fmt"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// sanitizeBlockRange checks that the provided block range is valid and within
// allowed limits, defaulting to a sensible range if not provided.
func sanitizeBlockRange(currentBlock uint64, blockRange *RPCRange) (RPCRange, error) {
	nextBlock := max(currentBlock+1, currentBlock)
	if blockRange == nil {
		maxRange := bundle.MakeMaxRangeStartingAt(nextBlock)
		return RPCRange{
			First:  hexutil.Uint64(maxRange.First),
			Length: hexutil.Uint64(maxRange.Length),
		}, nil
	}

	// If first is not specified, set it to currentBlock + 1.
	first := uint64(blockRange.First)
	if first == 0 {
		first = nextBlock
	}

	// Sanitize the length of the range.
	limit := uint64(blockRange.Length)
	if limit == 0 { // < user did not specify, use maximum allowed range
		limit = bundle.MaxBlockRangeLength
	} else if limit > bundle.MaxBlockRangeLength {
		return RPCRange{}, fmt.Errorf("invalid block range: length %d is too large; must be at most %d blocks", limit, bundle.MaxBlockRangeLength)
	}

	// Check whether the user given limit is already in the past
	r := bundle.BlockRange{First: first, Length: limit}
	if r.IsAfterRange(currentBlock) {
		return RPCRange{}, fmt.Errorf("invalid block range: the specified range (first: %d, length: %d) is not in the future relative to current block %d", first, limit, currentBlock)
	}

	return RPCRange{
		First:  hexutil.Uint64(first),
		Length: hexutil.Uint64(limit),
	}, nil
}
