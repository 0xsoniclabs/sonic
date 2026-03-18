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
	"github.com/ethereum/go-ethereum/core/types"
)

//go:generate mockgen -source=bundle_processor.go -destination=bundle_processor_mock.go -package=bundle

// RunBundle executes the transactions in the bundle using the provided
// TransactionRunner. It returns true if the bundle execution is considered
// successful, and false otherwise.
//
// This is the canonical implementation of the bundle execution logic, which
// defines the semantic of the execution flags.
func RunBundle(
	bundle *BundleLayer,
	runner TransactionRunner,
) bool {
	if bundle.Flags.IsOneOf() {
		return runOneOfBundle(bundle, runner)
	}
	return runAllOfBundle(bundle, runner)
}

// TransactionResult represents the result of executing a transaction within a
// bundle. It may be one of the following:
// - TransactionResultInvalid: The transaction is invalid (e.g., fails basic validation).
// - TransactionResultFailed: The transaction is valid but fails during execution (e.g., out of gas, revert).
// - TransactionResultSuccessful: The transaction is valid and executes successfully.
type TransactionResult int

const (
	TransactionResultInvalid TransactionResult = iota
	TransactionResultFailed
	TransactionResultSuccessful
)

// TransactionRunner defines an interface for running individual transactions
// within a bundle and obtaining their results, as used by the RunBundle
// function to determine the overall success of the bundle execution.
type TransactionRunner interface {
	Run(tx *types.Transaction) TransactionResult
	CreateInterTxSnapshot() int
	RevertToInterTxSnapshot(id int)
	CreateTxSnapshot() int
	RevertToTxSnapshot(id int)
}

// runAllOfBundle executes all transactions in the bundle and returns true if
// all transactions are considered successful, false otherwise.
func runAllOfBundle(
	bundle *BundleLayer,
	runner TransactionRunner,
) bool {
	for _, unit := range bundle.Units {
		var result TransactionResult
		if tx := unit.AsTransaction(); tx != nil {
			result = runner.Run(tx.Tx)
		} else {
			r := RunBundle(unit.AsBundleLayer(), runner)
			if r {
				result = TransactionResultSuccessful
			} else {
				result = TransactionResultFailed
			}
		}
		if !isTolerated(result, bundle.Flags) {
			return false
		}
	}
	return true
}

// runOneOfBundle executes the transactions in the bundle and stops at the first
// successful transaction. It returns true if at least one transaction is
// considered successful, false otherwise.
func runOneOfBundle(
	bundle *BundleLayer,
	runner TransactionRunner,
) bool {
	for _, unit := range bundle.Units {
		var result TransactionResult
		if tx := unit.AsTransaction(); tx != nil {
			result = runner.Run(tx.Tx)
		} else {
			r := RunBundle(unit.AsBundleLayer(), runner)
			if r {
				result = TransactionResultSuccessful
			} else {
				result = TransactionResultFailed
			}
		}
		if isTolerated(result, bundle.Flags) {
			return true
		}
	}
	return false
}

func isTolerated(
	result TransactionResult,
	flags ExecutionFlag,
) bool {
	if result == TransactionResultInvalid {
		return flags.TolerateInvalid()
	}
	if result == TransactionResultFailed {
		return flags.TolerateFailed()
	}
	return result == TransactionResultSuccessful
}
