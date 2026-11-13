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

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/format"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// getTransactionByHash answers a transaction of a block, or else one the pool
// holds.
func (h *handlers) getTransactionByHash(ctx context.Context, params hashParams) (*format.Transaction, error) {
	head := h.environment.Head()
	transaction, number, index, err := h.environment.TransactionByHash(head, params.Hash)
	if exec.IsMissing(err) {
		return h.pendingTransaction(params.Hash)
	}
	if err != nil {
		return nil, err
	}
	header, err := exec.NoneIfMissing(h.environment.HeaderAt(head, number))
	if header == nil || err != nil {
		return nil, err
	}
	position := format.Position{BlockHash: header.Hash, BlockNumber: number, Index: index, BaseFee: header.BaseFee}
	return format.NewTransaction(transaction, position, h.environment.ChainID()), nil
}

func (h *handlers) pendingTransaction(hash common.Hash) (*format.Transaction, error) {
	transaction, err := exec.NoneIfMissing(h.node.PoolTransaction(hash))
	if transaction == nil || err != nil {
		return nil, err
	}
	return format.NewPendingTransaction(transaction, h.environment.ChainID()), nil
}

// getRawTransactionByHash answers the binary form of a transaction of a
// block, or else of one the pool holds; "0x" for neither.
func (h *handlers) getRawTransactionByHash(ctx context.Context, params hashParams) (hexutil.Bytes, error) {
	transaction, _, _, err := h.environment.TransactionByHash(h.environment.Head(), params.Hash)
	if exec.IsMissing(err) {
		transaction, err = exec.NoneIfMissing(h.node.PoolTransaction(params.Hash))
	}
	if transaction == nil || err != nil {
		return nil, err
	}
	return transaction.MarshalBinary()
}

func (h *handlers) getTransactionByBlockNumberAndIndex(ctx context.Context, params numberAndIndexParams) (*format.Transaction, error) {
	return h.transactionAt(h.blockByNumber(params.Number), uint64(params.Index)), nil
}

func (h *handlers) getTransactionByBlockHashAndIndex(ctx context.Context, params hashAndIndexParams) (*format.Transaction, error) {
	return h.transactionAt(h.blockByHash(params.Hash), uint64(params.Index)), nil
}

// transactionAt forms the transaction at an index of a block, nil for a
// missing block or index.
func (h *handlers) transactionAt(block *evmcore.EvmBlock, index uint64) *format.Transaction {
	if block == nil || index >= uint64(len(block.Transactions)) {
		return nil
	}
	position := format.Position{BlockHash: block.Hash, BlockNumber: block.NumberU64(), Index: index, BaseFee: block.BaseFee}
	return format.NewTransaction(block.Transactions[index], position, h.environment.ChainID())
}

func (h *handlers) getRawTransactionByBlockNumberAndIndex(ctx context.Context, params numberAndIndexParams) (hexutil.Bytes, error) {
	return rawTransactionAt(h.blockByNumber(params.Number), uint64(params.Index)), nil
}

func (h *handlers) getRawTransactionByBlockHashAndIndex(ctx context.Context, params hashAndIndexParams) (hexutil.Bytes, error) {
	return rawTransactionAt(h.blockByHash(params.Hash), uint64(params.Index)), nil
}

// rawTransactionAt is the binary form of the transaction at an index of a
// block; "0x" for a missing block or index.
func rawTransactionAt(block *evmcore.EvmBlock, index uint64) hexutil.Bytes {
	if block == nil || index >= uint64(len(block.Transactions)) {
		return nil
	}
	encoded, _ := block.Transactions[index].MarshalBinary()
	return encoded
}
