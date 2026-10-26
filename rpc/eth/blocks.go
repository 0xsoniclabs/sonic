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
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/format"
)

type blockByNumberArgs struct {
	Block  chain.BlockNumber `rpc:"block"`
	FullTx bool              `rpc:"fullTx"`
}

type blockByHashArgs struct {
	Hash   common.Hash `rpc:"hash"`
	FullTx bool        `rpc:"fullTx"`
}

type hashArg struct {
	Hash common.Hash `rpc:"hash"`
}

type blockNumberIndexArgs struct {
	Block chain.BlockNumber `rpc:"block"`
	Index hexutil.Uint      `rpc:"index"`
}

type blockHashIndexArgs struct {
	Hash  common.Hash  `rpc:"hash"`
	Index hexutil.Uint `rpc:"index"`
}

type blockRefArg struct {
	Block chain.BlockRef `rpc:"block"`
}

func (a *api) getBlockByNumber(ctx context.Context, p blockByNumberArgs) (any, error) {
	return a.block(ctx, chain.NumberRef(p.Block), p.FullTx)
}

func (a *api) getBlockByHash(ctx context.Context, p blockByHashArgs) (any, error) {
	return a.block(ctx, chain.HashRef(p.Hash), p.FullTx)
}

// block answers a block with its transactions; a missing block is null.
func (a *api) block(ctx context.Context, ref chain.BlockRef, fullTx bool) (any, error) {
	head := a.env.Latest(ctx)
	block, err := a.env.FindBlock(head, ref)
	if err != nil {
		return exec.NullIfNotFound(block, err)
	}
	receipts, err := a.env.ReceiptsAt(head, block.NumberU64())
	if err != nil && !exec.IsMissing(err) {
		return nil, err
	}
	return format.Block(block, receipts, fullTx, a.env.ChainID()), nil
}

func (a *api) getHeaderByNumber(ctx context.Context, p blockNumberArg) (any, error) {
	return a.header(ctx, chain.NumberRef(p.Block))
}

func (a *api) getHeaderByHash(ctx context.Context, p hashArg) (any, error) {
	return a.header(ctx, chain.HashRef(p.Hash))
}

// header answers a header with the fields its receipts give; a missing block,
// or one whose receipts are missing, is null, and a hash the node does not
// know an error, as in the reference. It reads the block, as the reference
// did: the transactions root is derived from its transactions.
func (a *api) header(ctx context.Context, ref chain.BlockRef) (any, error) {
	head := a.env.Latest(ctx)
	block, err := a.env.FindBlock(head, ref)
	if errors.Is(err, chain.ErrUnknownHash) {
		return nil, exec.UnknownHashError(*ref.Hash)
	}
	if err != nil {
		return exec.NullIfNotFound(block, err)
	}
	receipts, err := a.env.ReceiptsAt(head, block.NumberU64())
	if err != nil {
		return exec.NullIfNotFound(receipts, err)
	}
	return block.Header().ToJson(receipts), nil
}

// getBlockReceipts answers the receipts of a block. A block selected by a hash
// the node does not know is an error, as in the reference; by a number, null.
func (a *api) getBlockReceipts(ctx context.Context, p blockRefArg) (any, error) {
	head := a.env.Latest(ctx)
	ref := p.Block
	if ref.Number == nil && ref.Hash == nil {
		ref = chain.NumberRef(0)
	}
	block, err := a.env.FindBlock(head, ref)
	switch {
	case errors.Is(err, chain.ErrUnknownHash):
		return nil, exec.UnknownHashError(*ref.Hash)
	case err != nil:
		return exec.NullIfNotFound(block, err)
	}
	receipts, err := a.env.ReceiptsAt(head, block.NumberU64())
	if err != nil {
		return exec.NullIfNotFound(receipts, err)
	}
	header := block.Header()
	signer := a.env.Signer()
	result := make([]map[string]any, len(receipts))
	for i, receipt := range receipts {
		if i >= len(block.Transactions) {
			return nil, core.ErrNotFound
		}
		result[i] = format.Receipt(header, block.Transactions[i], uint64(i), receipt, signer)
	}
	return result, nil
}

// txCount answers the number of transactions of a block, null if it is
// missing. Like the reference, it answers any failure of the lookup as a
// missing block.
func (a *api) txCount(ctx context.Context, ref chain.BlockRef) (any, error) {
	block, err := a.env.FindBlock(a.env.Latest(ctx), ref)
	if err != nil {
		return nil, nil
	}
	n := hexutil.Uint(len(block.Transactions))
	return &n, nil
}

func (a *api) getBlockTransactionCountByNumber(ctx context.Context, p blockNumberArg) (any, error) {
	return a.txCount(ctx, chain.NumberRef(p.Block))
}

func (a *api) getBlockTransactionCountByHash(ctx context.Context, p hashArg) (any, error) {
	return a.txCount(ctx, chain.HashRef(p.Hash))
}

// uncleCount answers zero uncles for a block that exists, null otherwise.
// Like the reference, it answers any failure of the lookup as a missing block.
func (a *api) uncleCount(ctx context.Context, ref chain.BlockRef) (any, error) {
	if _, err := a.env.FindHeader(a.env.Latest(ctx), ref); err != nil {
		return nil, nil
	}
	n := hexutil.Uint(0)
	return &n, nil
}

func (a *api) getUncleCountByBlockNumber(ctx context.Context, p blockNumberArg) (any, error) {
	return a.uncleCount(ctx, chain.NumberRef(p.Block))
}

func (a *api) getUncleCountByBlockHash(ctx context.Context, p hashArg) (any, error) {
	return a.uncleCount(ctx, chain.HashRef(p.Hash))
}

// Sonic blocks have no uncles.
func noUncleByNumber(context.Context, blockNumberIndexArgs) (any, error) { return nil, nil }
func noUncleByHash(context.Context, blockHashIndexArgs) (any, error)     { return nil, nil }

// txAt answers a transaction of a block by its index; encode turns it into its
// JSON or binary form. A missing block or index is the zero form: null, or
// "0x" for the binary one. Like the reference, it answers any failure of the
// lookup as a missing block.
func txAt[T any](ctx context.Context, a *api, ref chain.BlockRef, index hexutil.Uint, encode func(*evmcore.EvmBlock, uint64) T) (any, error) {
	block, err := a.env.FindBlock(a.env.Latest(ctx), ref)
	if err != nil {
		var none T
		return none, nil
	}
	return encode(block, uint64(index)), nil
}

func (a *api) transactionAt(block *evmcore.EvmBlock, index uint64) *format.Transaction {
	return format.TransactionAt(block, index, a.env.ChainID())
}

func (a *api) getTransactionByBlockNumberAndIndex(ctx context.Context, p blockNumberIndexArgs) (any, error) {
	return txAt(ctx, a, chain.NumberRef(p.Block), p.Index, a.transactionAt)
}

func (a *api) getTransactionByBlockHashAndIndex(ctx context.Context, p blockHashIndexArgs) (any, error) {
	return txAt(ctx, a, chain.HashRef(p.Hash), p.Index, a.transactionAt)
}

func (a *api) getRawTransactionByBlockNumberAndIndex(ctx context.Context, p blockNumberIndexArgs) (any, error) {
	return txAt(ctx, a, chain.NumberRef(p.Block), p.Index, format.RawTransactionAt)
}

func (a *api) getRawTransactionByBlockHashAndIndex(ctx context.Context, p blockHashIndexArgs) (any, error) {
	return txAt(ctx, a, chain.HashRef(p.Hash), p.Index, format.RawTransactionAt)
}

// getTransactionByHash answers a committed transaction, or a transaction of
// the pool, or null.
func (a *api) getTransactionByHash(ctx context.Context, p hashArg) (any, error) {
	head := a.env.Latest(ctx)
	tx, number, index, err := a.env.TxByHash(head, p.Hash)
	switch {
	case err == nil:
		header, err := a.env.HeaderAt(head, number)
		if err != nil {
			return exec.NullIfNotFound(header, err)
		}
		return format.NewTransaction(tx, header.Hash, number, index, header.BaseFee, a.env.ChainID()), nil
	case !exec.IsMissing(err):
		return nil, err
	}
	pooled, err := a.poolTransaction(p.Hash)
	if err != nil {
		return exec.NullIfNotFound(pooled, err)
	}
	return format.NewPendingTransaction(pooled, a.env.SuggestedGasPrice(head), a.env.ChainID()), nil
}

// poolTransaction looks a transaction up in the pool; nil is absence.
func (a *api) poolTransaction(hash common.Hash) (*types.Transaction, error) {
	tx, err := a.pool.PoolTransaction(hash)
	if err == nil && tx == nil {
		err = core.ErrNotFound
	}
	return tx, err
}

// getRawTransactionByHash answers the binary form of a committed or pooled
// transaction; one found nowhere is "0x", as in the reference.
func (a *api) getRawTransactionByHash(ctx context.Context, p hashArg) (any, error) {
	tx, _, _, err := a.env.TxByHash(a.env.Latest(ctx), p.Hash)
	if exec.IsMissing(err) {
		tx, err = a.poolTransaction(p.Hash)
		if exec.IsMissing(err) {
			return hexutil.Bytes(nil), nil
		}
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	enc, err := tx.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return hexutil.Bytes(enc), nil
}

// getTransactionReceipt answers the receipt of a committed transaction, or
// null.
func (a *api) getTransactionReceipt(ctx context.Context, p hashArg) (any, error) {
	head := a.env.Latest(ctx)
	tx, number, index, err := a.env.TxByHash(head, p.Hash)
	if err != nil {
		return exec.NullIfNotFound(tx, err)
	}
	return a.receipt(head, tx, number, index)
}

// receipt answers the receipt of a transaction at its position in a block, or
// null if it is missing.
func (a *api) receipt(head *exec.Head, tx *types.Transaction, number, index uint64) (any, error) {
	block, err := a.env.BlockAt(head, number)
	if err != nil {
		return exec.NullIfNotFound(block, err)
	}
	receipts, err := a.env.ReceiptsAt(head, number)
	if err != nil {
		return exec.NullIfNotFound(receipts, err)
	}
	if uint64(len(receipts)) <= index {
		return nil, nil
	}
	return format.Receipt(block.Header(), tx, index, receipts[index], a.env.Signer()), nil
}
