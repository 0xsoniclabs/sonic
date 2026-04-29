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

// MaxNumEstimableTransactions is the maximum number of transactions
// that can be included in a bundle for gas estimation.
// The algorithm to estimate bundle gas is O(n^2),
// therefore an upper bound is introduced.
const MaxNumEstimableTransactions = 16

// sanitizeBlockRange checks that the provided block range is valid and within
// allowed limits, defaulting to a sensible range if not provided.
func sanitizeBlockRange(currentBlock uint64, blockRange *RPCRange) (RPCRange, error) {
	if blockRange == nil {
		maxRange := bundle.MakeMaxRangeStartingAt(currentBlock + 1)
		return RPCRange{
			Earliest: hexutil.Uint64(maxRange.Earliest),
			Latest:   hexutil.Uint64(maxRange.Latest),
		}, nil
	}

	if blockRange.Latest == 0 && blockRange.Earliest != 0 {
		blockRange.Latest = hexutil.Uint64(uint64(blockRange.Earliest) + bundle.MaxBlockRange - 1)
		return RPCRange{
			Earliest: blockRange.Earliest,
			Latest:   blockRange.Latest,
		}, nil
	}

	// earliest not specified but latest is specified, set earliest to latest - maxRange + 1, but not less than currentBlock + 1 and check overflow
	if blockRange.Earliest == 0 && blockRange.Latest != 0 {
		if blockRange.Latest < hexutil.Uint64(currentBlock+bundle.MaxBlockRange) {
			blockRange.Earliest = hexutil.Uint64(currentBlock + 1)
		} else {
			return RPCRange{}, fmt.Errorf("invalid block range: latest block %d is too far in the future; earliest block must be at least %d", blockRange.Latest, currentBlock+1)
		}
	}

	if blockRange.Latest < hexutil.Uint64(currentBlock) {
		return RPCRange{}, fmt.Errorf("invalid block range: latest block %d is earlier than current block %d", blockRange.Latest, currentBlock)
	}

	if uint64(blockRange.Latest) < uint64(blockRange.Earliest) {
		return RPCRange{}, fmt.Errorf("invalid block range: latest block %d is earlier than earliest block %d", blockRange.Latest, blockRange.Earliest)
	}

	if uint64(blockRange.Latest-blockRange.Earliest+1) > bundle.MaxBlockRange {
		return RPCRange{}, fmt.Errorf("invalid block range: range %d is too large; must be at most %d blocks", blockRange.Latest-blockRange.Earliest+1, bundle.MaxBlockRange)
	}

	return RPCRange{
		Earliest: blockRange.Earliest,
		Latest:   blockRange.Latest,
	}, nil
}
