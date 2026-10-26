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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/evmcore"
)

// Block returns the JSON form of a block with its transactions, in full or
// as hashes. The receipts give the receipts root and the bloom; nil receipts
// leave both unset.
func Block(block *evmcore.EvmBlock, receipts types.Receipts, fullTransactions bool, chainID *big.Int) *evmcore.EvmBlockJson {
	size := hexutil.Uint64(block.EthBlock().Size())
	transactions := make([]any, len(block.Transactions))
	for index, transaction := range block.Transactions {
		if fullTransactions {
			transactions[index] = TransactionAt(block, uint64(index), chainID)
		} else {
			transactions[index] = transaction.Hash()
		}
	}
	return &evmcore.EvmBlockJson{
		EvmHeaderJson: block.Header().ToJson(receipts),
		Size:          &size,
		Txs:           transactions,
		Uncles:        make([]common.Hash, 0),
	}
}
