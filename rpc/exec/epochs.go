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

package exec

import (
	"errors"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
)

// CurrentEpoch is the epoch the block after the head runs in: the open
// epoch, unless the archive lags behind the end of an epoch.
func (e *Environment) CurrentEpoch(head *evmcore.EvmBlock) (uint64, error) {
	return e.reader.EpochOf(head.NumberU64() + 1)
}

// EpochStateOf answers the states of the epoch a number or tag names, as the
// legacy reads of epoch states take it: latest, safe and finalized name the
// current epoch, earliest the first block the node keeps, and pending the
// states the node has now. The states of an epoch are those stored when it
// opened; an epoch without them is core.ErrNotFound.
func (e *Environment) EpochStateOf(head *evmcore.EvmBlock, epoch chain.BlockNumber) (core.EpochSnapshot, error) {
	switch epoch {
	case chain.PendingBlockNumber:
		return e.reader.LiveEpochState(), nil
	case chain.LatestBlockNumber, chain.SafeBlockNumber, chain.FinalizedBlockNumber:
		current, err := e.CurrentEpoch(head)
		if err != nil {
			return core.EpochSnapshot{}, err
		}
		return e.reader.EpochState(current)
	case chain.EarliestBlockNumber:
		return e.reader.EpochState(e.chain.HistoryCutoff())
	}
	return e.reader.EpochState(uint64(epoch))
}

// DAGEpoch is the epoch a read of the DAG names, as the legacy DAG reads take
// it: pending is the current epoch, latest, safe and finalized the one
// before, earliest the first block the node keeps, and a number at most the
// current epoch itself.
func (e *Environment) DAGEpoch(head *evmcore.EvmBlock, epoch chain.BlockNumber) (uint64, error) {
	if epoch == chain.EarliestBlockNumber {
		return e.chain.HistoryCutoff(), nil
	}
	current, err := e.CurrentEpoch(head)
	if err != nil {
		return 0, err
	}
	switch {
	case epoch == chain.PendingBlockNumber:
		return current, nil
	case epoch == chain.LatestBlockNumber, epoch == chain.SafeBlockNumber, epoch == chain.FinalizedBlockNumber:
		return current - 1, nil
	case epoch >= 0 && uint64(epoch) <= current:
		return uint64(epoch), nil
	}
	return 0, errors.New("epoch is not in range")
}
