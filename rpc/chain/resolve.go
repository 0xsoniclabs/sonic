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

package chain

import (
	"errors"
	"fmt"

	"github.com/0xsoniclabs/sonic/rpc/core"
)

// The ways a reference can fail to name a block. A method maps each to the
// wording its legacy counterpart used.
var (
	// ErrNoSelector is a reference with neither a number nor a hash.
	ErrNoSelector = errors.New("reference without block number or hash")
	// ErrUnknownHash is a hash of no block the module serves.
	ErrUnknownHash = errors.New("unknown block hash")
	// ErrInvalidTag is a negative number that is no tag.
	ErrInvalidTag = errors.New("invalid block tag")
)

// Resolve turns a reference into the number of a block at the given head.
// Every tag but earliest names the head, earliest names the pruning cutoff. A
// number above the head is core.ErrNotFound, a hash above it ErrUnknownHash:
// the module serves no block it cannot serve the state of.
func (chain *Chain) Resolve(head Head, reference BlockReference) (uint64, error) {
	switch {
	case reference.Number != nil:
		return chain.ResolveNumber(head, *reference.Number)
	case reference.Hash != nil:
		number, err := chain.source.BlockNumberByHash(*reference.Hash)
		if errors.Is(err, core.ErrNotFound) {
			return 0, ErrUnknownHash
		}
		if err != nil {
			return 0, err
		}
		if number > head.Number {
			return 0, ErrUnknownHash
		}
		return number, nil
	}
	return 0, ErrNoSelector
}

// ResolveNumber turns a number or tag into the number of a block at the
// given head, as Resolve does.
func (chain *Chain) ResolveNumber(head Head, number BlockNumber) (uint64, error) {
	switch number {
	case LatestBlockNumber, PendingBlockNumber, SafeBlockNumber, FinalizedBlockNumber:
		return head.Number, nil
	case EarliestBlockNumber:
		return chain.config.HistoryPruningCutoff, nil
	}
	if number < 0 {
		return 0, fmt.Errorf("%w: %d", ErrInvalidTag, number)
	}
	if uint64(number) > head.Number {
		return 0, core.ErrNotFound
	}
	return uint64(number), nil
}
