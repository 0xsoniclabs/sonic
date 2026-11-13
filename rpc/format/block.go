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

package format

import (
	"math/big"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// Block forms a block with its header, whose receipt fields come from the
// receipts, and its transactions: their hashes, or each in full.
func Block(block *evmcore.EvmBlock, receipts types.Receipts, fullTransactions bool, chainID *big.Int) *evmcore.EvmBlockJson {
	size := hexutil.Uint64(block.EthBlock().Size())
	header := block.Header()
	transactions := make([]any, len(block.Transactions))
	for i, transaction := range block.Transactions {
		if !fullTransactions {
			transactions[i] = transaction.Hash()
			continue
		}
		transactions[i] = NewTransaction(transaction, Position{
			BlockHash:   block.Hash,
			BlockNumber: block.NumberU64(),
			Index:       uint64(i),
			BaseFee:     block.BaseFee,
		}, chainID)
	}
	return &evmcore.EvmBlockJson{
		EvmHeaderJson: header.ToJson(receipts),
		Txs:           transactions,
		Size:          &size,
		Uncles:        []common.Hash{},
	}
}
