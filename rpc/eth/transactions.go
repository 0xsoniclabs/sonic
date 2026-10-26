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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/format"
)

func indexArgument(arguments dispatch.Arguments) uint64 {
	return uint64(dispatch.Argument[hexutil.Uint](arguments, 1))
}

func (handler *handlers) getTransactionByBlockNumberAndIndex(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	return handler.transactionAt(ctx, numberArgument(arguments), indexArgument(arguments))
}

func (handler *handlers) getTransactionByBlockHashAndIndex(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	return handler.transactionAt(ctx, hashArgument(arguments), indexArgument(arguments))
}

// transactionAt answers a transaction of a block, null for a block the
// module cannot read, whatever the reason, as the reference does.
func (handler *handlers) transactionAt(ctx context.Context, reference chain.BlockReference, index uint64) (any, error) {
	_, block, err := handler.findBlock(ctx, reference)
	if block == nil || err != nil {
		return nil, nil
	}
	transaction := format.TransactionAt(block, index, handler.environment.ChainID())
	if transaction == nil {
		return nil, nil
	}
	return transaction, nil
}

func (handler *handlers) getRawTransactionByBlockNumberAndIndex(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	return handler.rawTransactionAt(ctx, numberArgument(arguments), indexArgument(arguments))
}

func (handler *handlers) getRawTransactionByBlockHashAndIndex(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	return handler.rawTransactionAt(ctx, hashArgument(arguments), indexArgument(arguments))
}

// rawTransactionAt answers the encoding of a transaction of a block. Where
// there is none it answers empty bytes, "0x", as the reference does.
func (handler *handlers) rawTransactionAt(ctx context.Context, reference chain.BlockReference, index uint64) (any, error) {
	_, block, err := handler.findBlock(ctx, reference)
	if block == nil || err != nil {
		return hexutil.Bytes(nil), nil
	}
	return format.RawTransactionAt(block, index), nil
}

// findTransaction returns a transaction of a served block with its
// position, nil for one the module does not have.
func (handler *handlers) findTransaction(head exec.Head, hash common.Hash) (*exec.Located, error) {
	return exec.NullIfAbsent(handler.environment.TransactionByHash(head, hash))
}

// poolTransaction returns a transaction of the pool, nil for one the pool
// does not hold.
func (handler *handlers) poolTransaction(hash common.Hash) (*types.Transaction, error) {
	return exec.NullIfAbsent(handler.node.PoolTransaction(hash))
}

// getTransactionByHash answers a transaction of a block, or else of the
// pool, priced at the suggested price of a new transaction.
func (handler *handlers) getTransactionByHash(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	hash := dispatch.Argument[common.Hash](arguments, 0)
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	located, err := handler.findTransaction(head, hash)
	if err != nil {
		return nil, err
	}
	chainID := handler.environment.ChainID()
	if located != nil {
		header, err := exec.NullIfAbsent(handler.environment.HeaderAt(head, located.Block))
		if header == nil || err != nil {
			return nil, err
		}
		location := format.Location{BlockHash: header.Hash, BlockNumber: located.Block, Index: located.Index}
		return format.NewTransaction(located.Transaction, location, header.BaseFee, chainID), nil
	}
	pending, err := handler.poolTransaction(hash)
	if pending == nil || err != nil {
		return nil, err
	}
	return format.PendingTransaction(pending, exec.SuggestedGasPrice(head), chainID), nil
}

// getRawTransactionByHash answers the encoding of a transaction of a block or
// of the pool, empty bytes for one the node does not have.
func (handler *handlers) getRawTransactionByHash(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	hash := dispatch.Argument[common.Hash](arguments, 0)
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	transaction, err := handler.chainOrPoolTransaction(head, hash)
	if transaction == nil || err != nil {
		return hexutil.Bytes(nil), err
	}
	encoded, err := transaction.MarshalBinary()
	return hexutil.Bytes(encoded), err
}

// chainOrPoolTransaction returns a transaction of a served block, or else of
// the pool, nil for one the node does not have.
func (handler *handlers) chainOrPoolTransaction(head exec.Head, hash common.Hash) (*types.Transaction, error) {
	located, err := handler.findTransaction(head, hash)
	if err != nil {
		return nil, err
	}
	if located != nil {
		return located.Transaction, nil
	}
	return handler.poolTransaction(hash)
}

func (handler *handlers) getTransactionReceipt(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	hash := dispatch.Argument[common.Hash](arguments, 0)
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	located, err := handler.findTransaction(head, hash)
	if located == nil || err != nil {
		return nil, err
	}
	header, err := exec.NullIfAbsent(handler.environment.HeaderAt(head, located.Block))
	if header == nil || err != nil {
		return nil, err
	}
	receipts, err := handler.blockReceipts(head, located.Block)
	if receipts == nil || err != nil {
		return nil, err
	}
	if uint64(len(receipts)) <= located.Index {
		return nil, nil
	}
	return format.Receipt(header, located.Transaction, located.Index, receipts[located.Index], handler.environment.ChainID()), nil
}
