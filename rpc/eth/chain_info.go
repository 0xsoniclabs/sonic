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

package eth

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func (h *handlers) chainID(context.Context, struct{}) (hexutil.Uint64, error) {
	return hexutil.Uint64(h.environment.ChainID().Uint64()), nil
}

func (h *handlers) blockNumber(context.Context, struct{}) (hexutil.Uint64, error) {
	return hexutil.Uint64(h.environment.Head().NumberU64()), nil
}

// syncedAge is how old the node's last block may be for the node to count as
// synchronised; far above the longest interval between blocks.
const syncedAge = 90 * time.Minute

// syncing answers false when the node is synchronised, and otherwise how far
// it and its best peer are.
func (h *handlers) syncing(context.Context, struct{}) (any, error) {
	progress := h.node.Progress()
	if time.Since(progress.CurrentBlockTime.Time()) <= syncedAge {
		return false, nil
	}
	return map[string]any{
		"startingBlock":    hexutil.Uint64(0),
		"currentEpoch":     hexutil.Uint64(progress.CurrentEpoch),
		"currentBlock":     hexutil.Uint64(progress.CurrentBlock),
		"currentBlockHash": progress.CurrentBlockHash.Hex(),
		"currentBlockTime": hexutil.Uint64(progress.CurrentBlockTime),
		"highestBlock":     hexutil.Uint64(progress.HighestBlock),
		"highestEpoch":     hexutil.Uint64(progress.HighestEpoch),
		"pulledStates":     hexutil.Uint64(0),
		"knownStates":      hexutil.Uint64(0),
	}, nil
}

// zeroAddress answers the etherbase and coinbase: the network has neither.
func zeroAddress(context.Context, struct{}) (common.Address, error) {
	return common.Address{}, nil
}

// zeroHashrate answers the hashrate: the network does not mine.
func zeroHashrate(context.Context, struct{}) (hexutil.Uint64, error) {
	return 0, nil
}
