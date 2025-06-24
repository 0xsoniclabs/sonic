// Copyright 2025 Sonic Operations Ltd
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

package ibr

import (
	"math/big"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/core/types"
)

type LlrFullBlockRecord struct {
	BlockHash  hash.Hash
	ParentHash hash.Hash
	StateRoot  hash.Hash
	Time       inter.Timestamp
	Duration   uint64
	Difficulty uint64
	GasLimit   uint64
	GasUsed    uint64
	BaseFee    *big.Int
	PrevRandao hash.Hash
	Epoch      idx.Epoch
	Txs        types.Transactions
	Receipts   []*types.ReceiptForStorage
}

type LlrIdxFullBlockRecord struct {
	LlrFullBlockRecord
	Idx idx.Block
}

// FullBlockRecordFor returns the full block record used in Genesis processing
// for the given block, list of transactions, and list of transaction receipts.
func FullBlockRecordFor(block *inter.Block, txs types.Transactions,
	rawReceipts []*types.ReceiptForStorage) *LlrFullBlockRecord {
	return &LlrFullBlockRecord{
		BlockHash:  hash.Hash(block.Hash()),
		ParentHash: hash.Hash(block.ParentHash),
		StateRoot:  hash.Hash(block.StateRoot),
		Time:       block.Time,
		Duration:   block.Duration,
		Difficulty: block.Difficulty,
		GasLimit:   block.GasLimit,
		GasUsed:    block.GasUsed,
		BaseFee:    block.BaseFee,
		PrevRandao: hash.Hash(block.PrevRandao),
		Epoch:      block.Epoch,
		Txs:        txs,
		Receipts:   rawReceipts,
	}
}
