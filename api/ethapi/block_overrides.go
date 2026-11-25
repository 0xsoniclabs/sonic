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

package ethapi

import (
	"context"
	"fmt"
	"math/big"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rpc"
)

// BlockOverrides is a set of header fields to override.
type BlockOverrides struct {
	Number      *hexutil.Big
	Difficulty  *hexutil.Big
	Time        *hexutil.Uint64
	GasLimit    *hexutil.Uint64
	Coinbase    *common.Address
	Random      *common.Hash
	BaseFee     *hexutil.Big
	BlobBaseFee *hexutil.Big
}

// apply overrides the given header fields into the given block context.
func (diff *BlockOverrides) apply(blockCtx *vm.BlockContext) {
	if diff == nil {
		return
	}
	if diff.Number != nil {
		blockCtx.BlockNumber = diff.Number.ToInt()
	}
	if diff.Difficulty != nil {
		blockCtx.Difficulty = diff.Difficulty.ToInt()
	}
	if diff.Time != nil {
		blockCtx.Time = uint64(*diff.Time)
	}
	if diff.GasLimit != nil {
		blockCtx.GasLimit = uint64(*diff.GasLimit)
	}
	if diff.Coinbase != nil {
		blockCtx.Coinbase = *diff.Coinbase
	}
	if diff.Random != nil {
		blockCtx.Random = diff.Random
	}
	if diff.BaseFee != nil {
		blockCtx.BaseFee = diff.BaseFee.ToInt()
	}
	if diff.BlobBaseFee != nil {
		blockCtx.BlobBaseFee = diff.BlobBaseFee.ToInt()
	}
}

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

// overriddenBlockContext returns the block context for a call made on the
// state of the base block, with the given block overrides applied. Overrides
// may be nil.
//
// If the overrides make the call run as the block directly following the base
// block, the base block becomes the parent of the call's block, so BLOCKHASH
// returns the base block's hash, as it would on chain. For any other number,
// the hash lookup stays relative to the base block.
func overriddenBlockContext(
	ctx context.Context,
	backend Backend,
	base *evmcore.EvmHeader,
	overrides *BlockOverrides,
) vm.BlockContext {
	blockCtx := GetBlockContext(ctx, backend, base)
	overrides.apply(&blockCtx)
	if isNextBlock(&blockCtx, base) {
		linkBaseBlockHash(&blockCtx, base)
	}
	return blockCtx
}

// linkBaseBlockHash makes BLOCKHASH return the base block's hash for the base
// block's number. Without it, the hash lookup starts at the base block, which
// is not its own ancestor, so its hash is unknown. Other numbers are resolved
// as before.
func linkBaseBlockHash(blockCtx *vm.BlockContext, base *evmcore.EvmHeader) {
	baseNumber := base.Number.Uint64()
	baseHash := base.Hash
	ancestors := blockCtx.GetHash
	blockCtx.GetHash = func(n uint64) common.Hash {
		if n == baseNumber {
			return baseHash
		}
		return ancestors(n)
	}
}

// applyNextBlockSystemCalls runs on the given state the system calls that
// start the block described by blockCtx, if that block directly follows the
// base block. After Prague, this is the EIP-2935 call that stores the base
// block's hash in the history contract, so that the contract agrees with
// BLOCKHASH, as it would on chain. For any other block, it does nothing.
func applyNextBlockSystemCalls(
	ctx context.Context,
	backend Backend,
	statedb state.StateDB,
	base *evmcore.EvmHeader,
	blockCtx *vm.BlockContext,
) error {
	if !isNextBlock(blockCtx, base) {
		return nil
	}
	vmConfig, err := GetVmConfig(ctx, backend, idx.Block(base.Number.Uint64()))
	if err != nil {
		return fmt.Errorf("failed to get vm config: %w", err)
	}
	vmConfig.NoBaseFee = true
	evm, _, err := backend.GetEVM(ctx, statedb, base, &vmConfig, blockCtx)
	if err != nil {
		return err
	}
	defer evm.Release()
	if evm.ChainConfig().IsPrague(blockCtx.BlockNumber, blockCtx.Time) {
		evmcore.ProcessParentBlockHash(base.Hash, evm, statedb)
	}
	return nil
}

// isNextBlock reports whether the block context describes the block directly
// following the base block.
func isNextBlock(blockCtx *vm.BlockContext, base *evmcore.EvmHeader) bool {
	if blockCtx.BlockNumber == nil || base.Number == nil {
		return false
	}
	next := new(big.Int).Add(base.Number, common.Big1)
	return blockCtx.BlockNumber.Cmp(next) == 0
}
