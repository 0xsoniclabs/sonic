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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

var ErrWrongEnvelopeGasLimitTooLow = errors.New("gas limit of envelope does not match gas limit of payload")

// ValidateEnvelope validates an envelope and its contents.
// It checks that the transaction is a valid bundle transaction and that all transactions in the bundle belong to the same execution plan.
// If the transaction is a valid transaction bundle, it returns the decoded transaction bundle and nil (no error).
// If the transaction is not a bundle transaction, or if bundle transactions are not enabled, it returns nil,nil (no bundle, no error).
func ValidateEnvelope(
	signer types.Signer,
	envelopeTx *types.Transaction,
) (*TransactionBundle, *ExecutionPlan, error) {
	return validateEnvelopeInternal(
		signer,
		envelopeTx,
		func(data []byte, accessList types.AccessList) (uint64, error) {
			return core.IntrinsicGas(
				envelopeTx.Data(),
				envelopeTx.AccessList(),
				nil,   // code auth is not used in the bundle transaction
				false, // bundle transaction is not a contract creation
				true,  // is homestead
				true,  // is istanbul
				true,  // is shanghai
			)
		},
		core.FloorDataGas,
	)
}

func validateEnvelopeInternal(
	signer types.Signer,
	envelopeTx *types.Transaction,
	calculateIntrinsicGas func(data []byte, accessList types.AccessList) (uint64, error),
	calculateFloorGas func(data []byte) (uint64, error),
) (*TransactionBundle, *ExecutionPlan, error) {
	if !IsEnvelope(envelopeTx) {
		// not a bundle transaction, nothing to validate
		return nil, nil, nil
	}

	txBundle, err := decode(envelopeTx.Data())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode transaction bundle: %v", err)
	}

	// TODO: this function shall validate bundle correctness,
	// the current implementation is preliminary to enable prototyping.
	// This code needs to be developed
	// Things to be checked include: (not complete)
	//  - consistent use of chain IDs
	//  - all bundled transactions are marked as bundle-only
	//  - etc. ...

	plan, err := txBundle.extractExecutionPlan(signer)
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

	bundleGas := envelopeTx.Gas()
	// Ensure the transaction has more gas than the basic tx fee.
	intrGas, err := calculateIntrinsicGas(
		envelopeTx.Data(),
		envelopeTx.AccessList(),
	)
	if err != nil {
		return nil, nil, err
	}
	if envelopeTx.Gas() < intrGas {
		return nil, nil, fmt.Errorf("%w, gas should be more than intrinsic gas %v", core.ErrIntrinsicGas, intrGas)
	}
	// gas limit of the bundle has to be exactly the aggregated gas of all the
	// transactions in the bundle or the intrinsic gas of the bundle
	// transaction, whichever is higher.
	gasLimit := uint64(0)
	for _, innerTx := range txBundle.Transactions {
		gasLimit += innerTx.Gas()
	}

	// EIP-7623 part of Prague revision: Floor data gas
	// see: https://eips.ethereum.org/EIPS/eip-7623
	floorDataGas, err := calculateFloorGas(envelopeTx.Data())
	if err != nil {
		return nil, nil, err
	}
	if envelopeTx.Gas() < floorDataGas {
		return nil, nil, fmt.Errorf("%w: gas should be more than floor gas %d", core.ErrFloorDataGas, floorDataGas)
	}

	gasNeeded := max(gasLimit, intrGas, floorDataGas)
	if bundleGas != gasNeeded {
		return nil, nil, fmt.Errorf("%w: envelope gas limit is %d but should be %d", ErrWrongEnvelopeGasLimitTooLow, envelopeTx.Gas(), gasNeeded)
	}

	// Check consistency of the execution plan.
	planHash := plan.Hash()
	for _, tx := range txBundle.Transactions {
		// check that all transactions in the bundle belong to the same execution plan
		if !belongsToExecutionPlan(tx, planHash) {
			return nil, nil, fmt.Errorf("transaction %s does not belong to the execution plan", tx.Hash().Hex())
		}
	}

	return &txBundle, &plan, nil
}

// --- internal utilities ---

// belongsToExecutionPlan checks if the given transaction correspond to one step in the execution plan.
func belongsToExecutionPlan(tx *types.Transaction, executionPlanHash common.Hash) bool {
	for _, entry := range tx.AccessList() {
		if entry.Address == BundleOnly &&
			slices.Contains(entry.StorageKeys, executionPlanHash) {
			return true
		}
	}
	return false
}
