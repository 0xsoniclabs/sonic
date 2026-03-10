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

package bundle

import (
	"errors"
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

var ErrBundleGasLimitTooLow = errors.New("gas limit of bundle transaction does not match the sum of the gas limits of the transactions in the bundle")

// ValidateTransactionBundle validates a bundle transaction.
// It checks that the transaction is a valid bundle transaction and that all transactions in the bundle belong to the same execution plan.
// If the transaction is a valid transaction bundle, it returns the decoded transaction bundle and nil (no error).
// If the transaction is not a bundle transaction, or if bundle transactions are not enabled, it returns nil,nil (no bundle, no error).
func ValidateTransactionBundle(
	tx *types.Transaction,
	signer types.Signer,
) (*TransactionBundle, *ExecutionPlan, error) {

	if !IsTransactionBundle(tx) {
		// not a bundle transaction, nothing to validate
		return nil, nil, nil
	}

	txBundle, err := Decode(tx.Data())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode transaction bundle: %v", err)
	}

	// TODO: this function shall validate bundle correctness,
	// the current implementation is preliminary to enable prototyping.
	// This code needs to be developed

	plan, err := txBundle.ExtractExecutionPlan(signer)
	if err != nil {
		return nil, nil, err
	}

	if plan.Latest < plan.Earliest {
		return nil, nil, fmt.Errorf("invalid empty block range [%d,%d] in execution plan", plan.Earliest, plan.Latest)
	}
	rangeSize := plan.Latest - plan.Earliest + 1

	if rangeSize > MaxBlockRange {
		return nil, nil, fmt.Errorf("invalid block range in execution plan, duration %d, limit %d", rangeSize, MaxBlockRange)
	}

	planHash := plan.Hash()
	for _, tx := range txBundle.Bundle {
		// check that all transactions in the bundle belong to the same execution plan
		if !BelongsToExecutionPlan(tx, planHash) {
			return nil, nil, fmt.Errorf("transaction %s does not belong to the execution plan", tx.Hash().Hex())
		}
	}

	// if this is a nested bundle, remove the bundleOnly marker before calculating the intrinsic gas
	accessList := slices.Clone(tx.AccessList())
	accessList = slices.DeleteFunc(accessList, func(al types.AccessTuple) bool {
		return al.Address == BundleOnly
	})

	bundleGas := tx.Gas()
	// Ensure the transaction has more gas than the basic tx fee.
	intrGas, err := core.IntrinsicGas(
		tx.Data(),
		accessList,
		nil,   // code auth is not used in the bundle transaction
		false, // bundle transaction is not a contract creation
		true,  // is homestead
		true,  // is istanbul
		true,  // is shanghai
	)
	if err != nil {
		return nil, nil, err
	}
	if tx.Gas() < intrGas {
		return nil, nil, fmt.Errorf("%w, gas should be more than %v", core.ErrIntrinsicGas, intrGas)
	}
	// gas limit of the bundle has to be exactly the aggregated gas of all the
	// transactions in the bundle or the intrinsic gas of the bundle
	// transaction, whichever is higher.
	gasLimit := uint64(0)
	for _, innerTx := range txBundle.Bundle {
		gasLimit += innerTx.Gas()
	}

	// EIP-7623 part of Prague revision: Floor data gas
	// see: https://eips.ethereum.org/EIPS/eip-7623
	floorDataGas, err := core.FloorDataGas(tx.Data())
	if err != nil {
		return nil, nil, err
	}
	if tx.Gas() < floorDataGas {
		return nil, nil, fmt.Errorf("%w: gas should be more than %d", core.ErrFloorDataGas, floorDataGas)
	}

	gasNeeded := max(gasLimit, intrGas, floorDataGas)
	if bundleGas != gasNeeded {
		return nil, nil, fmt.Errorf("%w: bundle gas limit %d but needs %d", ErrBundleGasLimitTooLow, tx.Gas(), gasNeeded)
	}

	return &txBundle, &plan, nil
}
