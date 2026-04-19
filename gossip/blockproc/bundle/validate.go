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
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	// MaxBundleNestingDepth defines the maximum allowed nesting depth of
	// bundles, thus envelopes of bundles being referenced by enclosing bundles.
	// This constant is critical to consensus, as it influences the decision of
	// whether a bundle is valid and can be computed or invalid and must be
	// rejected. It can thus only be altered as part of a hard-fork.
	//
	// The main intention of adding a limit to the number of nesting levels is
	// to provide a guaranteed upper limit for nesting valid bundles to
	// implementations, enabling them to reason about implementation trade-offs.
	// In particular, the resource usage of recursive operations can be
	// considered bound and effectively tested.
	//
	// The chosen value of 2 is somewhat arbitrary, but motivated by providing
	// some room for nesting while keeping the number of levels low enough
	// to be easily testable and to not cause issues for implementations.
	MaxBundleNestingDepth = 2

	// MaxGroupNestingDepth defines the maximum allowed nesting depth of
	// execution steps. This constant is critical to consensus, as it influences
	// the decision of whether a bundle is valid and can be computed or invalid
	// and must be rejected. It can thus only be altered as part of a hard-fork.
	//
	// The main intention of adding a limit to the number of nesting levels is
	// to provide a guaranteed upper limit for nesting valid execution plans to
	// implementations, enabling them to reason about implementation trade-offs.
	// In particular, the resource usage of recursive operations can be
	// considered bound and effectively tested.
	//
	// The chosen value of 8 is somewhat arbitrary, but motivated by providing
	// some room for nesting while keeping the number of levels low enough
	// to be easily testable and to not cause issues for implementations.
	MaxGroupNestingDepth = 8
)

// ValidateEnvelope validates an envelope and its contents. Among others, it
// checks whether the envelope contains the encoding of a valid bundle and that
// the gas limit for the envelope matches the maximum processing costs of the
// bundle. If the encoded bundle contains nested bundles, those are checked
// recursively as well.
func ValidateEnvelope(
	signer types.Signer,
	envelopeTx *types.Transaction,
) (*TransactionBundle, *ExecutionPlan, error) {
	return validateEnvelopeInternal(signer, envelopeTx, 0)
}

func validateEnvelopeInternal(
	signer types.Signer,
	envelopeTx *types.Transaction,
	depth int,
) (*TransactionBundle, *ExecutionPlan, error) {
	if depth > MaxBundleNestingDepth {
		return nil, nil, fmt.Errorf("exceeds maximum nesting depth of bundles")
	}

	// Everything that is not an envelope can be sorted out quickly.
	if envelopeTx == nil || !IsEnvelope(envelopeTx) {
		return nil, nil, fmt.Errorf("not an envelope transaction")
	}

	// Input validation - the signer must not be nil.
	if signer == nil {
		return nil, nil, fmt.Errorf("signer is nil")
	}

	// Next, we check whether the bundle can be decoded.
	bundle, err := OpenEnvelope(signer, envelopeTx)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid bundle encoding: %w", err)
	}

	// Check that the bundle is valid.
	if err := validateBundle(signer, bundle); err != nil {
		return nil, nil, fmt.Errorf("invalid bundle: %w", err)
	}

	// Check that the envelope's ChainID matches the signer's ChainID.
	envelopeChainId := envelopeTx.ChainId()
	signerChainId := signer.ChainID()
	if envelopeChainId != nil && signerChainId != nil && envelopeChainId.Cmp(signerChainId) != 0 {
		return nil, nil, fmt.Errorf("envelope signed for wrong chain ID")
	}

	// Check that the bundle's gas limit matches the required gas limit.
	have := envelopeTx.Gas()
	want, err := calculateEnvelopeGas(
		bundle,
		envelopeTx.Data(),
		envelopeTx.AccessList(),
		envelopeTx.SetCodeAuthorizations(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid gas limit: %w", err)
	}
	if have != want {
		return nil, nil, fmt.Errorf("invalid gas limit: have %d, want %d", have, want)
	}

	// Check that nested envelopes are fine.
	for _, tx := range bundle.Transactions {
		if IsEnvelope(tx) {
			if _, _, err := validateEnvelopeInternal(signer, tx, depth+1); err != nil {
				return nil, nil, fmt.Errorf("invalid nested envelope: %w", err)
			}
		}
	}

	// All checks passed, return temporaries.
	return &bundle, &bundle.Plan, nil
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
	if depth > MaxGroupNestingDepth {
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
