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

package gossip

import (
	"errors"
	"fmt"

	"github.com/Fantom-foundation/lachesis-base/hash"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
)

// Ledger is the consensus-agnostic block processing component.
type Ledger interface {
	// Head returns a handle to the block the consensus head pointer points to,
	// through which the next block is begun. It fails if that block is not in
	// the store.
	Head() (Head, error)
}

// Head is a handle to the head block and the only way to begin the next one,
// so the next block is built on the same parent the consensus validated
// against.
type Head interface {
	// Block returns the head block.
	Block() *inter.Block

	// BeginBlock opens the live state at the head's state root. It fails if
	// a block was already begun on this head, params.Number does not follow
	// the head, the head is stale, or the live state does not match it.
	BeginBlock(params BlockParams) (ExecutingBlock, error)
}

// ExecutingBlock is a block being executed. It owns the live state.
type ExecutingBlock interface {
	// stateDB returns the block's live state. It is unexported so that only
	// the gossip package can reach it, e.g. for the consensus callback.
	stateDB() state.StateDB
}

var (
	_ Ledger         = (*ledger)(nil)
	_ Head           = (*head)(nil)
	_ ExecutingBlock = (*executingBlock)(nil)
)

// BlockParams holds the header inputs for the next block.
type BlockParams struct {
	Number uint64
}

// ledger is the main implementation of Ledger.
type ledger struct {
	store *Store
}

// NewLedger creates a Ledger over the given store.
func NewLedger(store *Store) Ledger {
	return &ledger{store: store}
}

func (l *ledger) Head() (Head, error) {
	headIdx := l.store.GetLatestBlockIndex()
	block := l.store.GetBlock(headIdx)
	if block == nil {
		return nil, fmt.Errorf("head block %d not found in store", headIdx)
	}
	return &head{ledger: l, block: block}, nil
}

// head is the main implementation of Head.
type head struct {
	ledger   *ledger
	block    *inter.Block
	consumed bool
}

func (h *head) Block() *inter.Block {
	return h.block
}

func (h *head) BeginBlock(params BlockParams) (ExecutingBlock, error) {
	if h.consumed {
		return nil, errors.New("head already consumed")
	}
	if params.Number != h.block.Number+1 {
		return nil, fmt.Errorf("block %d does not follow head %d",
			params.Number, h.block.Number)
	}

	latest := h.ledger.store.GetLatestBlock()
	if latest == nil || latest.Hash() != h.block.Hash() {
		return nil, fmt.Errorf("block %d is no longer the ledger's head",
			h.block.Number)
	}
	statedb, err := h.ledger.store.evm.GetLiveStateDb(hash.Hash(h.block.StateRoot))
	if err != nil {
		return nil, err
	}
	h.consumed = true
	return &executingBlock{statedb: statedb}, nil
}

// executingBlock is the main implementation of ExecutingBlock.
type executingBlock struct {
	statedb state.StateDB
}

func (eb *executingBlock) stateDB() state.StateDB {
	return eb.statedb
}
