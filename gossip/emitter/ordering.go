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
	"cmp"
	"math/big"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/utils/frontierheap"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// stage is the consumption stage of a transaction.
type stage uint8

const (
	stagePrioritizedMyTurn stage = iota
	stagePrioritizedNotMyTurn
	stageNotPrioritized
)

// txWithMetadata wraps a transaction with its effective miner tip, its priority
// and whether it is this validator's turn to originate it. Non-prioritized
// entries carry a zero-valued priority.
type txWithMetadata struct {
	tx       *txpool.LazyTransaction
	tip      *uint256.Int
	priority priorities.Priority
	myTurn   bool
}

// stage returns the consumption stage of the transaction.
func (t *txWithMetadata) stage() stage {
	switch {
	case !t.priority.IsPrioritized():
		return stageNotPrioritized
	case t.myTurn:
		return stagePrioritizedMyTurn
	default:
		return stagePrioritizedNotMyTurn
	}
}

// computeEffectiveTip returns the miner tip the transaction yields at the given
// base fee. A transaction not covering the base fee has no tip to offer and is
// rejected unless it requests sponsorship.
func computeEffectiveTip(tx *txpool.LazyTransaction, baseFee *uint256.Int) (*uint256.Int, error) {
	if baseFee == nil {
		return new(uint256.Int).Set(tx.GasTipCap), nil
	}
	if tx.GasFeeCap.Cmp(baseFee) < 0 && !subsidies.IsSponsorshipRequest(tx.Tx) {
		return nil, types.ErrGasFeeCapTooLow
	}
	tip := new(uint256.Int).Sub(tx.GasFeeCap, baseFee)
	if tip.Gt(tx.GasTipCap) {
		tip = tx.GasTipCap
	}
	return tip, nil
}

// compareTxByStagePriorityPriceTime orders transactions by (stage asc,
// priority level desc, priority weight desc, effective miner tip desc,
// first-seen time asc), returning >0 when a precedes b.
func compareTxByStagePriorityPriceTime(a, b *txWithMetadata) int {
	if c := cmp.Compare(b.stage(), a.stage()); c != 0 {
		return c
	}
	if c := a.priority.Cmp(b.priority); c != 0 {
		return c
	}
	if c := a.tip.Cmp(b.tip); c != 0 {
		return c
	}
	return b.tx.Time.Compare(a.tx.Time)
}

// transactionsByPriorityAndPriceAndNonce is a heap over the senders'
// transaction sequences that returns transactions in the order
// (stage asc, priority desc, effective tip desc, time asc) while honouring
// per-sender nonce sequencing.
// Because of nonce sequencing, elements are NOT necessarily in monotonic order.
type transactionsByPriorityAndPriceAndNonce = frontierheap.FrontierHeap[*txWithMetadata]

// newTransactionsByPriorityAndPriceAndNonce collects the senders' transactions
// into a transactionsByPriorityAndPriceAndNonce, wrapping each with the metadata
// the ordering compares: its effective tip at baseFee, its priority as
// classified by context and its turn status as decided by turnPolicy.
//
// A transaction whose effective tip cannot be computed is dropped together with
// the sender's later nonces, which depend on it.
//
// A nil context treats every transaction as non-prioritized and thereby disables
// priority ordering altogether.
func newTransactionsByPriorityAndPriceAndNonce(
	txs map[common.Address][]*txpool.LazyTransaction,
	baseFee *big.Int,
	context *priorityContext,
	turnPolicy func(tx *txpool.LazyTransaction) bool,
) *transactionsByPriorityAndPriceAndNonce {
	// Convert the basefee from header format to uint256 format
	var baseFeeUint *uint256.Int
	if baseFee != nil {
		baseFeeUint = uint256.MustFromBig(baseFee)
	}

	heap := frontierheap.NewFrontierHeap(compareTxByStagePriorityPriceTime)
	for _, senderTxs := range txs {
		sequence := make([]*txWithMetadata, 0, len(senderTxs))
		for _, tx := range senderTxs {
			tip, err := computeEffectiveTip(tx, baseFeeUint)
			if err != nil {
				break // the sender's later nonces depend on the dropped transaction
			}
			sequence = append(sequence, &txWithMetadata{
				tx:       tx,
				tip:      tip,
				priority: context.priorityOf(tx.Resolve()),
				myTurn:   turnPolicy(tx),
			})
		}
		heap.AddSequence(sequence)
	}
	return heap
}
