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
	"github.com/0xsoniclabs/sonic/evmcore/core_types"
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
	bundle *TransactionBundle,
	runner TransactionRunner,
) bool {
	return runStep(&bundle.Plan.Root, bundle.Transactions, runner)
}

// TransactionRunner defines an interface for running individual transactions
// within a bundle and obtaining their results, as used by the RunBundle
// function to determine the overall success of the bundle execution.
type TransactionRunner interface {
	Run(tx *types.Transaction) core_types.TransactionResult
	CreateSnapshot() int
	RevertToSnapshot(id int)
}

// runStep executes a single execution step, which may be a transaction or a
// group of steps (one-of or all-of). It returns true if the step is considered
// successful based on the execution result and its execution flags, and false
// otherwise. The transaction index map is required to resolve transaction
// references for steps that execute transactions.
func runStep(
	step *ExecutionStep,
	transactions map[TxReference]*types.Transaction,
	runner TransactionRunner,
) bool {
	var result core_types.TransactionResult
	if step.txRef == nil {
		if step.oneOf {
			result = runOneOfGroup(step.steps, transactions, runner)
		} else {
			result = runAllOfGroup(step.steps, transactions, runner)
		}
	} else {
		result = runTransaction(*step.txRef, transactions, runner)
	}
	return isTolerated(result, step.flags)
}

// runAllOfGroup executes a group of steps where all steps must be successful
// for the group to be considered successful. If any step fails, the entire
// group is reverted to the state before the group execution began, and the
// function returns a failed result.
func runAllOfGroup(
	steps []ExecutionStep,
	transactions map[TxReference]*types.Transaction,
	runner TransactionRunner,
) core_types.TransactionResult {
	snapshot := runner.CreateSnapshot()
	for i := range steps {
		if !runStep(&(steps[i]), transactions, runner) {
			runner.RevertToSnapshot(snapshot)
			return core_types.TransactionResultFailed
		}
	}
	return core_types.TransactionResultSuccessful
}

// runOneOfGroup executes a group of steps where at least one step must be
// successful for the group to be considered successful. If all steps fail, the
// entire group is reverted to the state before the group execution began, and
// the function returns a failed result. After the first successful step,
// processing of the group stops and the function returns a successful result.
func runOneOfGroup(
	steps []ExecutionStep,
	transactions map[TxReference]*types.Transaction,
	runner TransactionRunner,
) core_types.TransactionResult {
	snapshot := runner.CreateSnapshot()
	for i := range steps {
		if runStep(&(steps[i]), transactions, runner) {
			return core_types.TransactionResultSuccessful
		}
	}
	runner.RevertToSnapshot(snapshot)
	return core_types.TransactionResultFailed
}

// runTransaction executes a single transaction referenced by txRef using the
// provided TransactionRunner. It returns the result of the transaction
// execution. If the transaction reference is not found in the transactions map,
// it signals an invalid transaction result.
func runTransaction(
	txRef TxReference,
	transactions map[TxReference]*types.Transaction,
	runner TransactionRunner,
) core_types.TransactionResult {
	tx, found := transactions[txRef]
	if !found {
		return core_types.TransactionResultInvalid
	}
	return runner.Run(tx)
}

// isTolerated determines whether a transaction result is considered successful
// based on the execution flags. It returns true if the result is successful or
// if it is invalid/failed but the corresponding tolerance flag is set, and false
// otherwise.
func isTolerated(
	result core_types.TransactionResult,
	flags ExecutionFlags,
) bool {
	if result == core_types.TransactionResultInvalid {
		return flags.TolerateInvalid()
	}
	if result == core_types.TransactionResultFailed {
		return flags.TolerateFailed()
	}
	return result == core_types.TransactionResultSuccessful
}
