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
	return runStep(bundle.Plan.Root, bundle.Transactions, runner)
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
	step ExecutionStep,
	transactions map[TxReference]*types.Transaction,
	runner TransactionRunner,
) bool {
	if !step.valid() {
		return false
	}
	if step.single != nil {
		return runSingle(*step.single, transactions, runner)
	}
	return runGroup(*step.group, transactions, runner)
}

// runGroup executes a group of steps, which can be either one-of or all-of,
// based on the group's execution semantic. It returns true if the group
// execution is considered tolerated, and false otherwise.
func runGroup(
	group group,
	transactions map[TxReference]*types.Transaction,
	runner TransactionRunner,
) bool {
	var success bool
	if group.oneOf {
		success = runOneOfGroup(group.steps, transactions, runner)
	} else {
		success = runAllOfGroup(group.steps, transactions, runner)
	}
	if group.tolerateFailed {
		return true
	}
	return success
}

// runAllOfGroup executes a group of steps where all steps must be successful
// for the group to be considered successful. If any step fails, the entire
// group is reverted to the state before the group execution began, and the
// function returns false. If all transactions succeed, true is returned.
func runAllOfGroup(
	steps []ExecutionStep,
	transactions map[TxReference]*types.Transaction,
	runner TransactionRunner,
) bool {
	snapshot := runner.CreateSnapshot()
	for _, step := range steps {
		if !runStep(step, transactions, runner) {
			runner.RevertToSnapshot(snapshot)
			return false
		}
	}
	return true
}

// runOneOfGroup executes a group of steps where at least one step must be
// successful for the group to be considered successful. If all steps fail, the
// entire group is reverted to the state before the group execution began, and
// the function returns false. After the first successful step, processing of
// the group stops and the function returns true.
func runOneOfGroup(
	steps []ExecutionStep,
	transactions map[TxReference]*types.Transaction,
	runner TransactionRunner,
) bool {
	snapshot := runner.CreateSnapshot()
	for _, step := range steps {
		if runStep(step, transactions, runner) {
			return true
		}
	}
	runner.RevertToSnapshot(snapshot)
	return false
}

// runSingle executes a single transaction step and returns true if the
// execution result is considered tolerated based on the execution flags, and
// false otherwise. If the transaction reference is not found in the transactions
// map, it is considered an invalid transaction result, and the function returns
// true if the TolerateInvalid flag is set, and false otherwise.
func runSingle(
	single single,
	transactions map[TxReference]*types.Transaction,
	runner TransactionRunner,
) bool {
	var result core_types.TransactionResult
	tx, found := transactions[single.txRef]
	if !found {
		result = core_types.TransactionResultInvalid
	} else {
		result = runner.Run(tx)
	}
	return isTolerated(result, single.flags)
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
