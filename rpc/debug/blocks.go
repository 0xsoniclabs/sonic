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

package debug

import (
	"context"
	"errors"
	"fmt"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/rlp"
)

type numberParams struct {
	Number uint64
}

// getBlockRlp answers the RLP encoding of a block, in hex without prefix.
func (h *handlers) getBlockRlp(ctx context.Context, params numberParams) (string, error) {
	block, _ := h.block(params.Number)
	if block == nil {
		return "", blockNotFound(params.Number)
	}
	encoded, err := rlp.EncodeToBytes(block)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", encoded), nil
}

// printBlock answers a dump of a block's fields.
func (h *handlers) printBlock(ctx context.Context, params numberParams) (string, error) {
	block, err := h.block(params.Number)
	if err != nil {
		return "", err
	}
	if block == nil {
		return "", blockNotFound(params.Number)
	}
	return spew.Sdump(block), nil
}

// block reads a block by its number, nil if there is none. The legacy handler
// takes the number for a block number of a client, so that a number above
// math.MaxInt64 names a tag, and one that names no tag no block; answers
// keep that cast.
func (h *handlers) block(number uint64) (*evmcore.EvmBlock, error) {
	reference := chain.ReferenceTo(chain.BlockNumber(number))
	block, err := exec.NoneIfMissing(h.environment.FindBlock(h.environment.Head(), reference))
	if errors.Is(err, chain.ErrInvalidTag) {
		return nil, nil
	}
	return block, err
}

func blockNotFound(number uint64) error {
	return fmt.Errorf("block #%d not found", number)
}
