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

package core

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ReceiptSource and TransactionSource are the parts of the client these checks need, kept as
// interfaces so that a test can serve them without a network.
type (
	ReceiptSource interface {
		TransactionReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error)
	}
	TransactionSource interface {
		TransactionByHash(ctx context.Context, hash common.Hash) (*types.Transaction, bool, error)
	}
)

// CheckBlockInvariants verifies what every block must satisfy whatever was injected into it. These
// need no model and no knowledge of the generated input, so a violation is unambiguous, and they
// target the bookkeeping a dropped transaction invites: receipt indices sliding out of step,
// cumulative gas counting a skipped transaction.
func CheckBlockInvariants(
	ctx context.Context,
	client ReceiptSource,
	blocks []*types.Block,
) error {
	for i, block := range blocks {
		if err := CheckOneBlock(ctx, client, block); err != nil {
			return fmt.Errorf("block %d: %w", block.NumberU64(), err)
		}
		if i == 0 {
			continue
		}

		switch previous := blocks[i-1]; {
		case block.NumberU64() != previous.NumberU64()+1:
			return fmt.Errorf(
				"block numbers are not contiguous: %d follows %d",
				block.NumberU64(), previous.NumberU64(),
			)
		case block.ParentHash() != previous.Hash():
			return fmt.Errorf(
				"block %d does not chain to block %d: parent %v, expected %v",
				block.NumberU64(), previous.NumberU64(), block.ParentHash(), previous.Hash(),
			)
		case block.Time() < previous.Time():
			return fmt.Errorf(
				"block %d has timestamp %d, before block %d's %d",
				block.NumberU64(), block.Time(), previous.NumberU64(), previous.Time(),
			)
		}
	}
	return nil
}

// CheckOneBlock verifies the receipts of a single block against its transactions.
func CheckOneBlock(
	ctx context.Context,
	client ReceiptSource,
	block *types.Block,
) error {
	cumulative := uint64(0)
	for i, tx := range block.Transactions() {
		receipt, err := client.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			return fmt.Errorf("transaction %d (%v) has no receipt: %w", i, tx.Hash(), err)
		}
		cumulative += receipt.GasUsed

		switch {
		case int(receipt.TransactionIndex) != i:
			return fmt.Errorf(
				"transaction %d (%v) has receipt index %d", i, tx.Hash(), receipt.TransactionIndex,
			)
		case receipt.BlockHash != block.Hash():
			return fmt.Errorf(
				"transaction %d (%v) has receipt block hash %v, expected %v",
				i, tx.Hash(), receipt.BlockHash, block.Hash(),
			)
		case receipt.BlockNumber.Cmp(block.Number()) != 0:
			return fmt.Errorf(
				"transaction %d (%v) has receipt block number %v, expected %v",
				i, tx.Hash(), receipt.BlockNumber, block.Number(),
			)
		case receipt.GasUsed > tx.Gas():
			return fmt.Errorf(
				"transaction %d (%v) used %d gas, more than its limit of %d",
				i, tx.Hash(), receipt.GasUsed, tx.Gas(),
			)
		case receipt.CumulativeGasUsed != cumulative:
			return fmt.Errorf(
				"transaction %d (%v) reports cumulative gas %d, expected %d",
				i, tx.Hash(), receipt.CumulativeGasUsed, cumulative,
			)
		}
	}

	switch {
	case block.GasUsed() != cumulative:
		return fmt.Errorf(
			"block reports %d gas used, but its receipts sum to %d", block.GasUsed(), cumulative,
		)
	case block.GasUsed() > block.GasLimit():
		return fmt.Errorf(
			"block used %d gas, above its limit of %d", block.GasUsed(), block.GasLimit(),
		)
	}
	return nil
}

// CheckAbsent verifies that a transaction which produced no receipt left no trace at all, closing the
// gap between "no receipt was found" and "it had no effect".
func CheckAbsent(ctx context.Context, client TransactionSource, hash common.Hash) error {
	tx, _, err := client.TransactionByHash(ctx, hash)
	if err == nil {
		return fmt.Errorf(
			"transaction %v produced no receipt but is still retrievable (type %d)", hash, tx.Type(),
		)
	}
	return nil
}

// StateChange is the observed effect of an injection on one account.
type StateChange struct {
	nonceBefore, nonceAfter     uint64
	balanceBefore, balanceAfter *big.Int
}

// AccountExpectation is what should have happened to one account over an iteration: the transactions
// of its that produced a receipt, and the most that transactions producing none may nonetheless have
// taken from it, which is zero unless known defect 2 applies.
type AccountExpectation struct {
	executed             []ExecutedTx
	maxUnreceiptedCharge *big.Int
}

// ExecutedTx records what one executed transaction should have cost its sender. TransfersValue is
// false when the value never left the account, because it went to the account itself or execution
// reverted; BlobFee is what was paid for blob gas, which the receipt does not report.
type ExecutedTx struct {
	Receipt        *types.Receipt
	Value          *big.Int
	TransfersValue bool
	BlobFee        *big.Int
}

// AccountObservation records what actually happened to an account, as opposed to what was merely
// permitted. It is evidence of a known defect, kept because accepting a range silently would leave
// the defect invisible on a green run.
type AccountObservation struct {
	unreceiptedCharge *big.Int
}

// CheckAccounting verifies that an account's nonce and balance moved by exactly what its transactions
// account for, the balance half being a conservation check: no wei may appear from nothing, and none
// may vanish beyond the fees and transfers recorded. The nonce must match exactly; the balance is a
// range only where a known defect widens it, and collapses to exact equality otherwise.
func CheckAccounting(
	address common.Address,
	change StateChange,
	expectation AccountExpectation,
) (AccountObservation, error) {
	observed := AccountObservation{unreceiptedCharge: new(big.Int)}

	spent := new(big.Int)
	for _, tx := range expectation.executed {
		spent.Add(spent, new(big.Int).Mul(gas(tx.Receipt.GasUsed), tx.Receipt.EffectiveGasPrice))
		spent.Add(spent, tx.BlobFee)
		if tx.TransfersValue {
			spent.Add(spent, tx.Value)
		}
	}

	expectedNonce := change.nonceBefore + uint64(len(expectation.executed))
	if change.nonceAfter != expectedNonce {
		return observed, fmt.Errorf(
			"account %v had %d transaction(s) executed, so its nonce should be %d, but it is %d",
			address, len(expectation.executed), expectedNonce, change.nonceAfter,
		)
	}

	allowance := expectation.maxUnreceiptedCharge
	if allowance == nil {
		allowance = new(big.Int)
	}

	highestBalance := new(big.Int).Sub(change.balanceBefore, spent)
	lowestBalance := new(big.Int).Sub(highestBalance, allowance)
	if change.balanceAfter.Cmp(lowestBalance) < 0 || change.balanceAfter.Cmp(highestBalance) > 0 {
		expected := highestBalance.String()
		if allowance.Sign() > 0 {
			expected = fmt.Sprintf("between %v and %v", lowestBalance, highestBalance)
		}
		return observed, fmt.Errorf(
			"account %v started with %v and its %d executed transaction(s) account for %v, "+
				"so it should hold %v, but it holds %v (off by %v)",
			address, change.balanceBefore, len(expectation.executed), spent,
			expected, change.balanceAfter,
			new(big.Int).Sub(change.balanceAfter, highestBalance),
		)
	}
	observed.unreceiptedCharge = new(big.Int).Sub(highestBalance, change.balanceAfter)

	return observed, nil
}
