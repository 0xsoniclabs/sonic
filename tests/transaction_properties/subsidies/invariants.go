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

package subsidies

import (
	"fmt"
	"math/big"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// CheckPostTransactions verifies, for every sponsorship request that executed, that the follow-up its
// mode calls for is in the block right behind it, succeeded, and charges the fee the specification
// says: the base fee on the gas used plus the mode's overhead. A fund-backed sponsorship must be
// followed by a deductFees transaction, a network-sponsored one by nothing at all, and a tracked one
// by a track transaction.
//
// The amounts are read back out of the follow-up's own call data with the production parsers, since
// what is being checked is the amount, not the encoding.
func (d *Domain) CheckPostTransactions(obs core.Observation) (map[common.Address]*big.Int, error) {
	charged := map[common.Address]*big.Int{}

	for _, tx := range obs.Executed() {
		if !IsSponsorshipRequest(tx.Spec) {
			if err := d.requireNoFollowUp(obs, tx); err != nil {
				return nil, err
			}
			continue
		}

		block, err := blockOf(obs, tx)
		if err != nil {
			return nil, err
		}

		charge, err := d.followUp(block, tx)
		if err != nil {
			return nil, err
		}
		if charge == nil {
			continue
		}

		expected := d.Charge(tx.Receipt.GasUsed, block.BaseFee())
		if charge.Cmp(expected) != 0 {
			return nil, fmt.Errorf(
				"the %s follow-up of sponsored transaction %v charges %v, but its %d gas used at a "+
					"base fee of %v with an overhead of %d gas comes to %v",
				d.Mode, tx.Hash, charge, tx.Receipt.GasUsed, block.BaseFee(),
				d.Mode.Overhead(d.Config), expected,
			)
		}

		total, ok := charged[tx.Sender.Address()]
		if !ok {
			total = new(big.Int)
			charged[tx.Sender.Address()] = total
		}
		total.Add(total, charge)
	}
	return charged, nil
}

// followUp is what the transaction a sponsored one must be followed by charges, or nil for a mode
// that appends none, having first verified that none was appended.
func (d *Domain) followUp(block *types.Block, tx core.TxObservation) (*big.Int, error) {
	index := int(tx.Receipt.TransactionIndex)
	txs := block.Transactions()
	if index >= len(txs) || txs[index].Hash() != tx.Hash {
		return nil, fmt.Errorf(
			"sponsored transaction %v is not at index %d of block %d, where its receipt puts it",
			tx.Hash, index, block.NumberU64())
	}

	if d.Mode.PostTransactions() == 0 {
		if index+1 < len(txs) && isFollowUp(txs[index+1]) {
			return nil, fmt.Errorf(
				"sponsored transaction %v is followed by a registry call, but %s sponsorships "+
					"append none", tx.Hash, d.Mode)
		}
		return nil, nil
	}

	if index+1 >= len(txs) {
		return nil, fmt.Errorf(
			"sponsored transaction %v is the last of block %d, so its %s follow-up is missing",
			tx.Hash, block.NumberU64(), d.Mode)
	}

	amount, err := d.parseCharge(txs[index+1])
	if err != nil {
		return nil, fmt.Errorf("the transaction after sponsored %v: %w", tx.Hash, err)
	}
	return amount, nil
}

// parseCharge reads what a follow-up charges, requiring it to be the kind the mode calls for.
func (d *Domain) parseCharge(post *types.Transaction) (*big.Int, error) {
	switch d.Mode {
	case ModeFundBacked:
		if !subsidies.IsFeeChargeTransaction(post) {
			return nil, fmt.Errorf("%v is not a deductFees transaction", post.Hash())
		}
		amount, err := subsidies.ParseFeeChargeAmount(post)
		if err != nil {
			return nil, err
		}
		return amount.ToBig(), nil

	case ModeNetworkTracked:
		if !subsidies.IsTrackTransaction(post) {
			return nil, fmt.Errorf("%v is not a track transaction", post.Hash())
		}
		amount, err := subsidies.ParseTrackAmount(post)
		if err != nil {
			return nil, err
		}
		return amount.ToBig(), nil
	}
	return nil, fmt.Errorf("mode %s appends no follow-up to parse", d.Mode)
}

// requireNoFollowUp asserts that a transaction paying its own way is not followed by a registry call,
// which would mean a fund was charged for something nobody asked to have sponsored.
func (d *Domain) requireNoFollowUp(obs core.Observation, tx core.TxObservation) error {
	block, err := blockOf(obs, tx)
	if err != nil {
		return err
	}

	txs := block.Transactions()
	index := int(tx.Receipt.TransactionIndex)
	if index+1 < len(txs) && isFollowUp(txs[index+1]) {
		return fmt.Errorf(
			"transaction %v pays its own way but is followed by the registry call %v",
			tx.Hash, txs[index+1].Hash())
	}
	return nil
}

// isFollowUp reports whether a transaction is one of the registry calls a sponsorship appends.
func isFollowUp(tx *types.Transaction) bool {
	return subsidies.IsFeeChargeTransaction(tx) || subsidies.IsTrackTransaction(tx)
}

// blockOf finds the block an executed transaction landed in among those the iteration observed.
func blockOf(obs core.Observation, tx core.TxObservation) (*types.Block, error) {
	for _, block := range obs.Blocks {
		if block.Number().Cmp(tx.Receipt.BlockNumber) == 0 {
			return block, nil
		}
	}
	return nil, fmt.Errorf(
		"transaction %v reports block %v, which is not among the blocks of this iteration",
		tx.Hash, tx.Receipt.BlockNumber)
}
