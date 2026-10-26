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

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/chain"
)

// TraceCallConfig configures debug_traceCall, trace_call and trace_callMany:
// a tracer, overrides of the state and the block, and the position in the
// block the call runs at.
type TraceCallConfig struct {
	tracers.TraceConfig
	StateOverrides *StateOverride
	BlockOverrides *BlockOverrides
	TxIndex        *hexutil.Uint
}

// ParentState opens the archive at the block before a block of the chain.
func (e *Env) ParentState(head *Head, block *evmcore.EvmBlock) (state.StateDB, error) {
	st, err := e.StateAtNumber(head, block.NumberU64()-1)
	if err != nil {
		return nil, HeaderNotFound(err)
	}
	return st, nil
}

// PreBlock creates the EVM of a block for replaying its transactions on the
// state before it, and runs the system calls of the block's start: the
// EIP-2935 parent hash from Prague on.
func (e *Env) PreBlock(ctx context.Context, block *evmcore.EvmBlock, st state.StateDB) (*vm.EVM, error) {
	cfg, err := e.VMConfig(ctx, block.NumberU64())
	if err != nil {
		return nil, fmt.Errorf("failed to get vm config: %w", err)
	}
	cfg.NoBaseFee = true
	header := block.Header()
	evm := vm.NewEVM(e.BlockContext(header), st, e.ChainConfig(block.NumberU64()), cfg)
	if evm.ChainConfig().IsPrague(block.Number, uint64(block.Time.Unix())) {
		evmcore.ProcessParentBlockHash(block.ParentHash, evm, st)
	}
	return evm, nil
}

// StateAtTransaction replays a block up to a transaction and returns the
// transaction's message and the state it runs on. The caller releases the
// state; it is released here on failure, a panic included.
func (e *Env) StateAtTransaction(head *Head, block *evmcore.EvmBlock, txIndex int) (*core.Message, state.StateDB, error) {
	if block.NumberU64() == 0 {
		return nil, nil, errors.New("no transaction in genesis")
	}
	if txIndex > len(block.Transactions) {
		return nil, nil, fmt.Errorf("transaction index %d out of range for block %#x", txIndex, block.Hash)
	}
	st, err := e.ParentState(head, block)
	if err != nil {
		return nil, nil, err
	}
	handedOver := false
	defer func() {
		if !handedOver {
			st.Release()
		}
	}()
	evm, err := e.PreBlock(head.Context(), block, st)
	if err != nil {
		return nil, nil, err
	}
	if txIndex == 0 && len(block.Transactions) == 0 {
		handedOver = true
		return nil, st, nil
	}
	signer := types.LatestSignerForChainID(evm.ChainConfig().ChainID)
	for i, tx := range block.Transactions {
		msg, err := evmcore.TxAsMessage(tx, signer, block.BaseFee)
		if err != nil {
			return nil, nil, err
		}
		if i == txIndex {
			handedOver = true
			return msg, st, nil
		}
		// Sonic supports blob transactions without blob data only.
		if msg.BlobHashes != nil {
			if len(msg.BlobHashes) > 0 {
				continue
			}
			msg.BlobHashes = nil
		}
		st.SetTxContext(tx.Hash(), i)
		if _, err := core.ApplyMessage(evm, msg, core.NewGasPool(tx.Gas())); err != nil {
			return nil, nil, fmt.Errorf("transaction %#x failed: %v", tx.Hash(), err)
		}
		st.EndTransaction()
	}
	return nil, nil, fmt.Errorf("transaction index %d out of range for block %#x", txIndex, block.Hash)
}

// TraceBlock is the block a traced call runs on: "header not found" for a
// missing one, as the tracers phrase it.
func (e *Env) TraceBlock(head *Head, ref chain.BlockRef) (*evmcore.EvmBlock, error) {
	if ref.Number == nil && ref.Hash == nil {
		return nil, errors.New("invalid arguments; neither block number nor hash specified")
	}
	block, err := e.FindBlock(head, ref)
	if err != nil {
		return nil, HeaderNotFound(err)
	}
	return block, nil
}

// TxMessage converts the arguments of a traced call to its message and to the
// legacy transaction the tracers report it as.
func (e *Env) TxMessage(args *CallArgs, block *evmcore.EvmBlock) (*types.Transaction, *core.Message, error) {
	msg, err := args.ToMessage(e.cfg.GasCap, block.BaseFee)
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
