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
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

// blockOfNumber finds a block by the number argument. The reference cast the
// number to a block number, so a number above the largest int64 is a tag:
// 2^64-2 names the head (F1 over H6).
func (handler *handlers) blockOfNumber(ctx context.Context, arguments dispatch.Arguments) (*evmcore.EvmBlock, error) {
	number := dispatch.Argument[uint64](arguments, 0)
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	block, err := exec.NullIfAbsent(handler.environment.FindBlock(head, chain.ReferenceToNumber(chain.BlockNumber(number))))
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", number)
	}
	return block, nil
}

// getBlockRlp answers the RLP encoding of a block as the node keeps it. As
// in the reference, a block that cannot be read is not found.
func (handler *handlers) getBlockRlp(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	block, err := handler.blockOfNumber(ctx, arguments)
	if err != nil {
		return nil, fmt.Errorf("block #%d not found", dispatch.Argument[uint64](arguments, 0))
	}
	encoded, err := rlp.EncodeToBytes(block)
	if err != nil {
		return nil, err
	}
	return fmt.Sprintf("%x", encoded), nil
}

// printBlock answers a block printed as Go's spew prints it.
func (handler *handlers) printBlock(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	block, err := handler.blockOfNumber(ctx, arguments)
	if err != nil {
		return nil, err
	}
	return spew.Sdump(block), nil
}
