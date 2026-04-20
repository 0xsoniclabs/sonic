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

const (
	// MaxNestingDepth defines the maximum allowed nesting depth of execution
	// steps. This constant is critical to consensus, as it influences the
	// decision of whether a bundle is valid and can be computed or invalid and
	// must be rejected. It can thus only be altered as part of a hard-fork.
	//
	// The main intention of adding a limit to the number of nesting levels is
	// to provide a guaranteed upper limit for nesting valid execution plans to
	// implementations, enabling them to reason about implementation trade-offs.
	// In particular, the resource usage of recursive operations can be
	// considered bound and effectively tested.
	//
	// The chosen value of 16 is somewhat arbitrary, but motivated by providing
	// generous room for nesting while keeping the number of levels low enough
	// to be easily testable and to not cause issues for implementations.
	MaxNestingDepth = 16
)

var ErrWrongEnvelopeGasLimit = errors.New("gas limit of envelope does not match gas limit of payload")

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

// validateEnvelopeInternal is an internal version of ValidateEnvelope enabling
// the injection of custom steps for testing.
func validateEnvelopeInternal(
	signer types.Signer,
	envelopeTx *types.Transaction,
	calculateIntrinsicGas func(data []byte, accessList types.AccessList) (uint64, error),
	calculateFloorGas func(data []byte) (uint64, error),
) (*TransactionBundle, *ExecutionPlan, error) {
	if !IsEnvelope(envelopeTx) {
		return nil, nil, fmt.Errorf("not an envelope transaction")
	}

	txBundle, err := decode(signer, envelopeTx.Data())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode transaction bundle: %v", err)
	}

	// TODO: this function shall validate bundle correctness,
	// the current implementation is preliminary to enable prototyping.
	// This code needs to be developed
	// Things to be checked include: (not complete)
	//  - consistent use of chain IDs
	//  - all bundled transactions are marked as bundle-only
	//  - all transactions referenced in the plan are included in the bundle
	//  - all transactions in the bundle are referenced in the plan
	//  - etc. ...

	plan := txBundle.Plan
	if err := validateRange(plan.Range); err != nil {
		return nil, nil, err
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
		return nil, nil, fmt.Errorf("%w: envelope gas limit is %d but should be %d", ErrWrongEnvelopeGasLimit, envelopeTx.Gas(), gasNeeded)
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

// validateBundle checks that the given transaction bundle is valid, meaning
// that it is well-formed and consistent.
func validateBundle(
	signer types.Signer,
	bundle TransactionBundle,
) error {

	// check the execution plan for validity
	if err := validatePlan(bundle.Plan); err != nil {
		return err
	}

	// check that there are no nil transactions in the bundle
	for _, tx := range bundle.Transactions {
		if tx == nil {
			return fmt.Errorf("invalid nil transaction in bundle")
		}
	}

	// check that signer is not nil before using it
	if signer == nil {
		return fmt.Errorf("signer is nil")
	}

	// make sure that the reference keys in the index match the transactions
	for ref, tx := range bundle.Transactions {
		sender, err := types.Sender(signer, tx)
		if err != nil {
			return fmt.Errorf("invalid transaction in bundle: %v", err)
		}
		if ref.From != sender {
			return fmt.Errorf("sender in transaction reference does not match actual sender")
		}

		strippedTx, err := removeBundleOnlyMark(tx)
		if err != nil {
			return fmt.Errorf("invalid transaction in bundle: %v", err)
		}
		if ref.Hash != signer.Hash(strippedTx) {
			return fmt.Errorf("content of transaction does not match transaction hash")
		}
	}

	// check that all transactions in the bundle agree to the execution plan
	planHash := bundle.Plan.Hash()
	for _, tx := range bundle.Transactions {
		if !belongsToExecutionPlan(tx, planHash) {
			return fmt.Errorf("contains transaction not approving the execution plan")
		}
	}

	// check that all transactions referenced by the plan are present in the bundle
	references := map[TxReference]struct{}{}
	for _, ref := range bundle.Plan.Root.GetTransactionReferencesInReferencedOrder() {
		references[ref] = struct{}{}
	}
	for ref := range references {
		if _, found := bundle.Transactions[ref]; !found {
			return fmt.Errorf("missing transaction referenced by the execution plan")
		}
	}

	// check that there are no extra transactions not referenced by the plan
	for ref := range bundle.Transactions {
		if _, found := references[ref]; !found {
			return fmt.Errorf("contains transaction not referenced by the execution plan")
		}
	}

	return nil
}

// validatePlan checks that the given execution plan is valid.
func validatePlan(plan ExecutionPlan) error {
	if err := validateStep(plan.Root); err != nil {
		return fmt.Errorf("invalid execution plan: %v", err)
	}
	if err := validateRange(plan.Range); err != nil {
		return fmt.Errorf("invalid block range: %v", err)
	}
	return nil
}

// validateStep checks that the given execution step is valid.
func validateStep(step ExecutionStep) error {
	return validateStepInternal(step, 0)
}

func validateStepInternal(
	step ExecutionStep,
	depth int,
) error {

	// Check limit of maximum nesting.
	if depth > MaxNestingDepth {
		return fmt.Errorf("exceeds maximum nesting depth of execution steps")
	}

	// The step must be either a single or a group, not neither or both.
	if !step.valid() {
		return fmt.Errorf("malformed execution step")
	}

	// Check properties of the single step variant.
	if single := step.single; single != nil {
		if !single.flags.Valid() {
			return fmt.Errorf("invalid execution flags in step")
		}
		return nil
	}

	// Check properties of the group step variant.
	if group := step.group; group != nil {
		for _, subStep := range group.steps {
			if err := validateStepInternal(subStep, depth+1); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateRange checks that the given block range is valid, i.e. that it is not
// empty and does not exceed the maximum allowed range.
func validateRange(r BlockRange) error {
	size := r.Size()
	if size == 0 {
		return fmt.Errorf("invalid empty block range [%d,%d]", r.Earliest, r.Latest)
	}
	if size > MaxBlockRange {
		return fmt.Errorf("invalid block range, duration %d, limit %d", size, MaxBlockRange)
	}
	return nil
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
