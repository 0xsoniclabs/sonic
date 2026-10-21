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
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// BlockAt returns the block with the number, nil if the served chain has none.
func (e *Env) BlockAt(head dispatch.Head, number uint64) (*evmcore.EvmBlock, error) {
	if number > head.Number {
		return nil, nil
	}
	block, err := e.Core.Block(number)
	if errors.Is(err, core.ErrNotFound) {
		return nil, nil
	}
	return block, err
}

// BlockByNumber returns the block a number or tag names; nil if there is none.
func (e *Env) BlockByNumber(head dispatch.Head, bn dispatch.BlockNumber) (*evmcore.EvmBlock, error) {
	n, err := e.Chain.Number(head, bn)
	if err != nil {
		return nil, nil
	}
	return e.BlockAt(head, n)
}

// BlockByHash returns the block with the hash; nil if there is none.
func (e *Env) BlockByHash(head dispatch.Head, hash common.Hash) (*evmcore.EvmBlock, error) {
	n, ok := e.Core.BlockNumberByHash(hash)
	if !ok {
		return nil, nil
	}
	return e.BlockAt(head, n)
}

// ErrNoBlockSelector is the answer of the methods that look a block up by a
// reference that holds neither number nor hash.
var ErrNoBlockSelector = errors.New("invalid arguments; neither block number nor hash specified")

// FindBlock returns the block a reference names; nil if there is none. A
// reference without number and hash is an error.
func (e *Env) FindBlock(head dispatch.Head, ref dispatch.BlockRef) (*evmcore.EvmBlock, error) {
	n, err := e.Chain.Resolve(head, ref)
	if errors.Is(err, dispatch.ErrNoSelector) {
		return nil, ErrNoBlockSelector
	}
	if err != nil {
		return nil, nil
	}
	return e.BlockAt(head, n)
}

// BlockByRef returns the block a reference names, "header not found" if there
// is none.
func (e *Env) BlockByRef(head dispatch.Head, ref dispatch.BlockRef) (*evmcore.EvmBlock, error) {
	block, err := e.FindBlock(head, ref)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, ErrHeaderNotFound
	}
	return block, nil
}

// ReplayOptions tune the replay of a block.
type ReplayOptions struct {
	// SkipParentBlockHash leaves out the write of the parent block hash into
	// the history storage contract (EIP-2935) before the replay.
	SkipParentBlockHash bool
}

// StateAtTransaction returns the execution environment of a certain
// transaction: the message of the transaction and the state in which it runs,
// which is the state of the parent block with the transactions before it
// applied. The history storage contract is written once before the first
// replayed transaction, as the node's earlier handlers did for the transaction
// tracers. The caller releases the state.
func StateAtTransaction(ctx context.Context, env *Env, head dispatch.Head, block *evmcore.EvmBlock, txIndex int) (*gethcore.Message, state.StateDB, error) {
	return stateAtTransaction(ctx, env, head, block, txIndex, ReplayOptions{})
}

func stateAtTransaction(ctx context.Context, env *Env, head dispatch.Head, block *evmcore.EvmBlock, txIndex int, opts ReplayOptions) (*gethcore.Message, state.StateDB, error) {
	// Short circuit if it's genesis block.
	if block.NumberU64() == 0 {
		return nil, nil, errors.New("no transaction in genesis")
	}

	// Check correct txIndex
	if txIndex > len(block.Transactions) {
		return nil, nil, fmt.Errorf("transaction index %d out of range for block %#x", txIndex, block.Hash)
	}

	// Look up the state of the parent block.
	statedb, _, err := env.StateAt(head, dispatch.RefHash(block.ParentHash))
	if err != nil {
		return nil, nil, err
	}

	if txIndex == 0 && len(block.Transactions) == 0 {
		return nil, statedb, nil
	}

	// Use the block's VM config for replaying transactions with possible no base fee
	cfg, err := env.VMConfig(block.NumberU64())
	if err != nil {
		statedb.Release()
		return nil, nil, fmt.Errorf("failed to get vm config: %w", err)
	}
	cfg.NoBaseFee = true
	header := block.Header()
	vmenv, err := env.NewEVM(head, statedb, header, &cfg, nil)
	if err != nil {
		statedb.Release()
		return nil, nil, err
	}

	// execute EIP-2935 HistoryStorage contract.
	if !opts.SkipParentBlockHash && vmenv.ChainConfig().IsPrague(block.Number, uint64(block.Time.Unix())) {
		evmcore.ProcessParentBlockHash(block.ParentHash, vmenv, statedb)
	}

	// Recompute transactions up to the target index.
	chainConfig := env.ChainConfig(block.NumberU64())
	signer := types.LatestSignerForChainID(chainConfig.ChainID)
	for i, tx := range block.Transactions {
		if err := ctx.Err(); err != nil {
			statedb.Release()
			return nil, nil, err
		}
		// Assemble the transaction call message and return if the requested offset
		msg, err := evmcore.TxAsMessage(tx, signer, block.BaseFee)
		if err != nil {
			statedb.Release()
			return nil, nil, err
		}
		if i == txIndex {
			return msg, statedb, nil
		}

		// For now, Sonic only supports Blob transactions without blob data.
		if msg.BlobHashes != nil {
			if len(msg.BlobHashes) > 0 {
				continue // blob data is not supported - this tx will be skipped
			}
			// PreCheck requires non-nil blobHashes not to be empty
			msg.BlobHashes = nil
		}

		statedb.SetTxContext(tx.Hash(), i)
		if _, err := gethcore.ApplyMessage(vmenv, msg, gethcore.NewGasPool(tx.Gas())); err != nil {
			statedb.Release()
			return nil, nil, fmt.Errorf("transaction %#x failed: %v", tx.Hash(), err)
		}
		// Ensure any modifications are committed to the state
		statedb.EndTransaction()
	}
	statedb.Release()
	return nil, nil, fmt.Errorf("transaction index %d out of range for block %#x", txIndex, block.Hash)
}

// TxAndMessage returns the transaction and the message constructed from the
// call arguments, for the tracing of a call.
func TxAndMessage(args *CallArgs, gasCap uint64, header *evmcore.EvmHeader) (*types.Transaction, *gethcore.Message, error) {
	msg, err := args.ToMessage(gasCap, header.BaseFee, log.Root())
	if err != nil {
		return nil, nil, err
	}

	tx := types.NewTx(&types.LegacyTx{
		To:       msg.To,
		Nonce:    msg.Nonce,
		Gas:      msg.GasLimit,
		GasPrice: msg.GasPrice,
		Value:    msg.Value,
		Data:     msg.Data,
	})
	return tx, msg, nil
}
