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
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/format"
)

// findBlock returns the block a number or a hash argument names, nil for one
// the module does not have.
func (handler *handlers) findBlock(ctx context.Context, reference chain.BlockReference) (exec.Head, *evmcore.EvmBlock, error) {
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return exec.Head{}, nil, err
	}
	block, err := exec.NullIfAbsent(handler.environment.FindBlock(head, reference))
	return head, block, err
}

func numberArgument(arguments dispatch.Arguments) chain.BlockReference {
	return chain.ReferenceToNumber(dispatch.Argument[chain.BlockNumber](arguments, 0))
}

func hashArgument(arguments dispatch.Arguments) chain.BlockReference {
	return chain.ReferenceToHash(dispatch.Argument[common.Hash](arguments, 0))
}

// blockReceipts returns the receipts of a block, nil for a block whose
// receipts the module does not have, and the error of a node that keeps no
// transaction index.
func (handler *handlers) blockReceipts(head exec.Head, number uint64) (types.Receipts, error) {
	return exec.NullIfAbsent(handler.environment.ReceiptsAt(head, number))
}

func (handler *handlers) getBlockByNumber(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	return handler.getBlock(ctx, numberArgument(arguments), dispatch.Argument[bool](arguments, 1))
}

func (handler *handlers) getBlockByHash(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	return handler.getBlock(ctx, hashArgument(arguments), dispatch.Argument[bool](arguments, 1))
}

func (handler *handlers) getBlock(ctx context.Context, reference chain.BlockReference, fullTransactions bool) (any, error) {
	head, block, err := handler.findBlock(ctx, reference)
	if block == nil || err != nil {
		return nil, err
	}
	receipts, err := handler.blockReceipts(head, block.NumberU64())
	if err != nil {
		return nil, err
	}
	return format.Block(block, receipts, fullTransactions, handler.environment.ChainID()), nil
}

// getHeaderByNumber reads the block, not only its header: the JSON form
// carries the transactions root.
func (handler *handlers) getHeaderByNumber(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	return handler.getHeader(ctx, numberArgument(arguments))
}

// getHeaderByHash answers an unknown hash with an error, as the reference
// does, where the other lookups by hash answer null.
func (handler *handlers) getHeaderByHash(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	reference := hashArgument(arguments)
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	block, err := handler.environment.FindBlock(head, reference)
	if errors.Is(err, chain.ErrUnknownHash) {
		return nil, exec.UnknownHashError(*reference.Hash)
	}
	return handler.headerJSON(head, block, err)
}

func (handler *handlers) getHeader(ctx context.Context, reference chain.BlockReference) (any, error) {
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	block, err := handler.environment.FindBlock(head, reference)
	return handler.headerJSON(head, block, err)
}

// headerJSON answers the JSON form of the header of a block, null for a
// block or receipts the module does not have.
func (handler *handlers) headerJSON(head exec.Head, block *evmcore.EvmBlock, err error) (any, error) {
	block, err = exec.NullIfAbsent(block, err)
	if block == nil || err != nil {
		return nil, err
	}
	receipts, err := handler.blockReceipts(head, block.NumberU64())
	if receipts == nil || err != nil {
		return nil, err
	}
	return block.Header().ToJson(receipts), nil
}

// getBlockReceipts answers the receipts of a block. Like the reference, a
// reference without number or hash names block 0, and an unknown hash is an
// error.
func (handler *handlers) getBlockReceipts(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	reference := dispatch.Argument[chain.BlockReference](arguments, 0)
	if reference.Number == nil && reference.Hash == nil {
		reference = chain.ReferenceToNumber(0)
	}
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	block, err := handler.environment.FindBlock(head, reference)
	if errors.Is(err, chain.ErrUnknownHash) {
		return nil, exec.UnknownHashError(*reference.Hash)
	}
	block, err = exec.NullIfAbsent(block, err)
	if block == nil || err != nil {
		return nil, err
	}
	receipts, err := handler.blockReceipts(head, block.NumberU64())
	if receipts == nil || err != nil {
		return nil, err
	}
	header := block.Header()
	chainID := handler.environment.ChainID()
	answer := make([]map[string]any, len(receipts))
	for index, receipt := range receipts {
		if index >= len(block.Transactions) {
			return nil, errors.New("receipt without transaction")
		}
		answer[index] = format.Receipt(header, block.Transactions[index], uint64(index), receipt, chainID)
	}
	return answer, nil
}

func (handler *handlers) getBlockTransactionCountByNumber(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	return handler.transactionCount(ctx, numberArgument(arguments))
}

func (handler *handlers) getBlockTransactionCountByHash(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	return handler.transactionCount(ctx, hashArgument(arguments))
}

// transactionCount answers the number of transactions of a block, null for
// a block the module cannot read, whatever the reason, as the reference does.
func (handler *handlers) transactionCount(ctx context.Context, reference chain.BlockReference) (any, error) {
	_, block, err := handler.findBlock(ctx, reference)
	if block == nil || err != nil {
		return nil, nil
	}
	count := hexutil.Uint(len(block.Transactions))
	return &count, nil
}

// Sonic has no uncles: a block answers none and an uncle is always null.

func (handler *handlers) getUncleByBlockNumberAndIndex(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	return handler.uncle(ctx, numberArgument(arguments))
}

func (handler *handlers) getUncleByBlockHashAndIndex(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	return handler.uncle(ctx, hashArgument(arguments))
}

func (handler *handlers) uncle(ctx context.Context, reference chain.BlockReference) (any, error) {
	_, _, err := handler.findBlock(ctx, reference)
	return nil, err
}

func (handler *handlers) getUncleCountByBlockNumber(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	return handler.uncleCount(ctx, numberArgument(arguments))
}

func (handler *handlers) getUncleCountByBlockHash(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	return handler.uncleCount(ctx, hashArgument(arguments))
}

func (handler *handlers) uncleCount(ctx context.Context, reference chain.BlockReference) (any, error) {
	_, block, err := handler.findBlock(ctx, reference)
	if block == nil || err != nil {
		return nil, nil
	}
	count := hexutil.Uint(0)
	return &count, nil
}
