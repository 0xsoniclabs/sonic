// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package emitter

import (
	"math/big"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/utils/frontierheap"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// txWithMinerFee wraps a transaction with its gas price or effective miner gasTipCap
type txWithMinerFee struct {
	tx   *txpool.LazyTransaction
	from common.Address
	fees *uint256.Int
}

// newTxWithMinerFee creates a wrapped transaction, calculating the effective
// miner gasTipCap if a base fee is provided.
// Returns error in case of a negative effective miner gasTipCap.
func newTxWithMinerFee(tx *txpool.LazyTransaction, from common.Address, baseFee *uint256.Int) (*txWithMinerFee, error) {
	tip := new(uint256.Int).Set(tx.GasTipCap)
	if baseFee != nil {
		if tx.GasFeeCap.Cmp(baseFee) < 0 {
			if !subsidies.IsSponsorshipRequest(tx.Tx) {
				return nil, types.ErrGasFeeCapTooLow
			}
		}
		tip = new(uint256.Int).Sub(tx.GasFeeCap, baseFee)
		if tip.Gt(tx.GasTipCap) {
			tip = tx.GasTipCap
		}
	}
	return &txWithMinerFee{
		tx:   tx,
		from: from,
		fees: tip,
	}, nil
}

// compareTxByPriceAndTime orders transactions by (effective miner tip desc,
// first-seen time asc), returning >0 when a precedes b.
func compareTxByPriceAndTime(a, b *txWithMinerFee) int {
	if c := a.fees.Cmp(b.fees); c != 0 {
		return c
	}
	return b.tx.Time.Compare(a.tx.Time)
}

// transactionsByPriceAndNonce is the heap of transaction sequences used to
// order the candidates of an event, see newTransactionsByPriceAndNonce.
type transactionsByPriceAndNonce = frontierheap.FrontierHeap[*txWithMinerFee]

// newTransactionsByPriceAndNonce creates a heap over the senders' transaction
// sequences that returns transactions in a profit-maximizing order (effective
// tip desc, time asc) while honouring per-sender nonce sequencing.
//
// Every transaction is wrapped up front. A transaction whose effective tip
// cannot be computed is dropped together with the sender's later nonces, which
// depend on it.
func newTransactionsByPriceAndNonce(
	txs map[common.Address][]*txpool.LazyTransaction,
	baseFee *big.Int,
) *transactionsByPriceAndNonce {
	// Convert the basefee from header format to uint256 format
	var baseFeeUint *uint256.Int
	if baseFee != nil {
		baseFeeUint = uint256.MustFromBig(baseFee)
	}

	heap := frontierheap.NewFrontierHeap(compareTxByPriceAndTime)
	for sender, senderTxs := range txs {
		sequence := make([]*txWithMinerFee, 0, len(senderTxs))
		for _, tx := range senderTxs {
			wrapped, err := newTxWithMinerFee(tx, sender, baseFeeUint)
			if err != nil {
				break // the sender's later nonces depend on the dropped transaction
			}
			sequence = append(sequence, wrapped)
		}
		heap.AddSequence(sequence)
	}
	return heap
}
