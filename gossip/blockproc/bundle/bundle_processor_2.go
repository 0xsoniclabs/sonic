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

//go:generate mockgen -source=bundle_processor_2.go -destination=bundle_processor_2_mock.go -package=bundle

// RunBundle2 executes the transactions in the bundle using the provided
// TransactionRunner. It returns true if the bundle execution is considered
// successful, and false otherwise.
//
// This is the canonical implementation of the bundle execution logic, which
// defines the semantic of the execution plan and flags.
func RunBundle2(
	bundle *TransactionBundle2,
	runner TransactionRunner,
) bool {
	return runGroup(
		&bundle.Plan.Group,
		bundle.Transactions,
		runner,
	)
}

func runGroup(
	group *Group,
	transactions map[TxReference]*types.Transaction,
	runner TransactionRunner,
) bool {
	snapshot := runner.CreateSnapshot()

	var success bool
	if group.Flags.IsOneOf() {
		success = runOneOfGroup(group, transactions, runner)
	} else {
		success = runAllOfGroup(group, transactions, runner)
	}

	if !success {
		runner.RevertToSnapshot(snapshot)
	}

	return success
}

func runAllOfGroup(
	group *Group,
	transactions map[TxReference]*types.Transaction,
	runner TransactionRunner,
) bool {
	for _, step := range group.Steps {
		result := runStep(step, transactions, runner)
		if !isTolerated(result, group.Flags) {
			return false
		}
	}
	return true
}

func runOneOfGroup(
	group *Group,
	transactions map[TxReference]*types.Transaction,
	runner TransactionRunner,
) bool {
	for _, step := range group.Steps {
		result := runStep(step, transactions, runner)
		if isTolerated(result, group.Flags) {
			return true
		}
	}
	return false
}

func runStep(
	step GroupOrTransaction,
	transactions map[TxReference]*types.Transaction,
	runner TransactionRunner,
) core_types.TransactionResult {
	switch s := step.(type) {
	case *Group:
		success := runGroup(s, transactions, runner)
		if success {
			return core_types.TransactionResultSuccessful
		}
		return core_types.TransactionResultFailed
	case *TxReference:
		tx, found := transactions[*s]
		if !found {
			return core_types.TransactionResultInvalid
		}
		return runner.Run(tx)
	default:
		return core_types.TransactionResultInvalid
	}
}
