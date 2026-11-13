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

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/format"
)

// getTransactionReceipt answers the receipt of a transaction of a block.
func (h *handlers) getTransactionReceipt(ctx context.Context, params hashParams) (map[string]any, error) {
	head := h.environment.Head()
	transaction, number, index, err := h.environment.TransactionByHash(head, params.Hash)
	if exec.IsMissing(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	header, err := exec.NoneIfMissing(h.environment.HeaderAt(head, number))
	if header == nil || err != nil {
		return nil, err
	}
	receipts, err := exec.NoneIfMissing(h.environment.ReceiptsAt(head, number))
	if receipts == nil || err != nil || index >= uint64(len(receipts)) {
		return nil, err
	}
	return format.Receipt(header, transaction, index, receipts[index], h.environment.ChainID()), nil
}

type blockReferenceParams struct {
	Block chain.BlockReference
}

// getBlockReceipts answers the receipts of a block. A reference with neither
// a number nor a hash names block 0, as the legacy handler reads it.
func (h *handlers) getBlockReceipts(ctx context.Context, params blockReferenceParams) ([]map[string]any, error) {
	head := h.environment.Head()
	number, err := h.environment.Resolve(head, params.Block)
	switch {
	case errors.Is(err, chain.ErrMissingSelector):
		number = 0
	case errors.Is(err, chain.ErrUnknownHash):
		return nil, unknownHashError(*params.Block.Hash)
	case err != nil:
		return nil, err
	}
	block, err := exec.NoneIfMissing(h.environment.BlockAt(head, number))
	if block == nil || err != nil {
		return nil, err
	}
	receipts, err := exec.NoneIfMissing(h.environment.ReceiptsAt(head, number))
	if receipts == nil || err != nil {
		return nil, err
	}
	header := block.Header()
	formed := make([]map[string]any, len(receipts))
	for i, receipt := range receipts {
		formed[i] = format.Receipt(header, block.Transactions[i], uint64(i), receipt, h.environment.ChainID())
	}
	return formed, nil
}
