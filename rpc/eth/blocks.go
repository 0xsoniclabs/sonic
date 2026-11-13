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
	"fmt"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/format"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type blockByNumberParams struct {
	Number           chain.BlockNumber
	FullTransactions bool
}

type blockByHashParams struct {
	Hash             common.Hash
	FullTransactions bool
}

type numberParams struct {
	Number chain.BlockNumber
}

type hashParams struct {
	Hash common.Hash
}

type numberAndIndexParams struct {
	Number chain.BlockNumber
	Index  hexutil.Uint
}

type hashAndIndexParams struct {
	Hash  common.Hash
	Index hexutil.Uint
}

func (h *handlers) getBlockByNumber(ctx context.Context, params blockByNumberParams) (*evmcore.EvmBlockJson, error) {
	head := h.environment.Head()
	return h.formBlock(head, chain.ReferenceTo(params.Number), params.FullTransactions)
}

func (h *handlers) getBlockByHash(ctx context.Context, params blockByHashParams) (*evmcore.EvmBlockJson, error) {
	head := h.environment.Head()
	return h.formBlock(head, chain.BlockReference{Hash: &params.Hash}, params.FullTransactions)
}

// formBlock answers a block with its receipt fields, or nil when there is
// none.
func (h *handlers) formBlock(head *evmcore.EvmBlock, reference chain.BlockReference, fullTransactions bool) (*evmcore.EvmBlockJson, error) {
	block, err := exec.NoneIfMissing(h.environment.FindBlock(head, reference))
	if block == nil || err != nil {
		return nil, err
	}
	receipts, err := exec.NoneIfMissing(h.environment.ReceiptsAt(head, block.NumberU64()))
	if err != nil {
		return nil, err
	}
	return format.Block(block, receipts, fullTransactions, h.environment.ChainID()), nil
}

func (h *handlers) getHeaderByNumber(ctx context.Context, params numberParams) (*evmcore.EvmHeaderJson, error) {
	head := h.environment.Head()
	number, err := h.environment.Resolve(head, chain.ReferenceTo(params.Number))
	if err != nil {
		return nil, err
	}
	return h.formHeader(head, number)
}

// getHeaderByHash answers an unknown hash with an error, where the other
// lookups by hash answer null, as the legacy handler does.
func (h *handlers) getHeaderByHash(ctx context.Context, params hashParams) (*evmcore.EvmHeaderJson, error) {
	head := h.environment.Head()
	number, err := h.environment.Resolve(head, chain.BlockReference{Hash: &params.Hash})
	if errors.Is(err, chain.ErrUnknownHash) {
		return nil, unknownHashError(params.Hash)
	}
	if err != nil {
		return nil, err
	}
	return h.formHeader(head, number)
}

// formHeader answers a header with its receipt fields, or nil when there is
// none or no receipts.
func (h *handlers) formHeader(head *evmcore.EvmBlock, number uint64) (*evmcore.EvmHeaderJson, error) {
	header, err := exec.NoneIfMissing(h.environment.HeaderAt(head, number))
	if header == nil || err != nil {
		return nil, err
	}
	receipts, err := exec.NoneIfMissing(h.environment.ReceiptsAt(head, number))
	if receipts == nil || err != nil {
		return nil, err
	}
	return header.ToJson(receipts), nil
}

// unknownHashError words a hash that names no block.
func unknownHashError(hash common.Hash) error {
	return fmt.Errorf("block with hash %s not found", hash.String())
}

// The counts and the uncles take a failed lookup for a missing block, as the
// legacy handlers do.

func (h *handlers) getBlockTransactionCountByNumber(ctx context.Context, params numberParams) (*hexutil.Uint, error) {
	return transactionCount(h.blockByNumber(params.Number)), nil
}

func (h *handlers) getBlockTransactionCountByHash(ctx context.Context, params hashParams) (*hexutil.Uint, error) {
	return transactionCount(h.blockByHash(params.Hash)), nil
}

func transactionCount(block *evmcore.EvmBlock) *hexutil.Uint {
	if block == nil {
		return nil
	}
	count := hexutil.Uint(len(block.Transactions))
	return &count
}

// The network has no uncles: a block has none, and a missing block no count.

func (h *handlers) getUncleByBlockNumberAndIndex(ctx context.Context, params numberAndIndexParams) (*evmcore.EvmBlockJson, error) {
	return nil, nil
}

func (h *handlers) getUncleByBlockHashAndIndex(ctx context.Context, params hashAndIndexParams) (*evmcore.EvmBlockJson, error) {
	return nil, nil
}

func (h *handlers) getUncleCountByBlockNumber(ctx context.Context, params numberParams) (*hexutil.Uint, error) {
	return uncleCount(h.blockByNumber(params.Number)), nil
}

func (h *handlers) getUncleCountByBlockHash(ctx context.Context, params hashParams) (*hexutil.Uint, error) {
	return uncleCount(h.blockByHash(params.Hash)), nil
}

func uncleCount(block *evmcore.EvmBlock) *hexutil.Uint {
	if block == nil {
		return nil
	}
	none := hexutil.Uint(0)
	return &none
}

// blockByNumber reads a block by number or tag, nil when the lookup fails.
func (h *handlers) blockByNumber(number chain.BlockNumber) *evmcore.EvmBlock {
	block, _ := h.environment.FindBlock(h.environment.Head(), chain.ReferenceTo(number))
	return block
}

// blockByHash reads a block by hash, nil when the lookup fails.
func (h *handlers) blockByHash(hash common.Hash) *evmcore.EvmBlock {
	block, _ := h.environment.FindBlock(h.environment.Head(), chain.BlockReference{Hash: &hash})
	return block
}
