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
