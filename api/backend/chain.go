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
package backend

import (
	"context"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rpc"
)

// GetBlockContext returns a new vm.BlockContext based on the given header and backend
func GetBlockContext(ctx context.Context, backend Backend, header *evmcore.EvmHeader) vm.BlockContext {
	chain := chainContext{
		ctx: ctx,
		b:   backend,
	}
	return evmcore.NewEVMBlockContext(header, &chain, nil)
}

// chainContextBackend provides methods required to implement ChainContext.
type chainContextBackend interface {
	HeaderByNumber(context.Context, rpc.BlockNumber) (*evmcore.EvmHeader, error)
}

// chainContext is an implementation of core.chainContext. It's main use-case
// is instantiating a vm.BlockContext without having access to the BlockChain object.
type chainContext struct {
	b   chainContextBackend
	ctx context.Context
}

func (context *chainContext) Header(hash common.Hash, number uint64) *evmcore.EvmHeader {
	// This method is called to get the hash for a block number when executing the BLOCKHASH
	// opcode. Hence no need to search for non-canonical blocks.
	header, err := context.b.HeaderByNumber(context.ctx, rpc.BlockNumber(number))
	if header == nil || err != nil {
		return nil
	}
	if header.Hash != hash {
		return nil
	}
	return header
}
