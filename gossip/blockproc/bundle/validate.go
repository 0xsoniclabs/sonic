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
	"fmt"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/core/types"
)

// ValidateTransactionBundle validates a transaction bundle transaction.
// It checks that the transaction is a valid bundle transaction and that all transactions in the bundle belong to the same execution plan.
// If the transaction is a valid transaction bundle, it returns nil.
// If the transaction is not a bundle transaction, or if bundle transactions are not enabled, it returns nil (no error).
func ValidateTransactionBundle(
	tx *types.Transaction,
	signer types.Signer,
	upgrades opera.Upgrades) error {

	if !upgrades.TransactionBundles {
		// feature not enabled, nothing to validate
		return nil
	}

	if !IsTransactionBundle(tx) {
		// not a bundle transaction, nothing to validate
		return nil
	}

	txBundle, err := Decode(tx.Data())
	if err != nil {
		return fmt.Errorf("failed to decode transaction bundle: %v", err)
	}

	// TODO: this function shall validate bundle correctness,
	// the current implementation is preliminary to enable prototyping.
	// This code needs to be developed

	plan, err := txBundle.ExtractExecutionPlan(signer)
	if err != nil {
		return err
	}

	planHash := plan.Hash()
	for _, tx := range txBundle.Bundle {
		// check that all transactions in the bundle belong to the same execution plan
		if !BelongsToExecutionPlan(tx, planHash) {
			return fmt.Errorf("transaction %s does not belong to the execution plan", tx.Hash().Hex())
		}
	}

	return nil
}
