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

package sonic

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

// RPCPreparedBundle is the return type of the sonic_prepareBundle RPC method.
type RPCPreparedBundle struct {
	// Transactions is the flat depth-first ordered list; must be signed without modification.
	Transactions  []exec.CallArguments       `json:"transactions"`
	ExecutionPlan RPCExecutionPlanComposable `json:"executionPlan"`
}

// prepareBundle implements sonic_prepareBundle, as the reference's
// PrepareBundle: it builds the execution plan of a proposal whose leaves are
// unsigned transactions, estimates the missing gas limits in depth-first
// order, each after the ones before, fills the missing prices with the
// suggested gas price, and adds the bundle-only marker with the plan hash to
// every access list. The returned transactions must be signed without
// changing any field. The current block is the served head (section 6).
func (handler *handlers) prepareBundle(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	proposal := dispatch.Argument[RPCExecutionProposal](arguments, 0)
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	blockRange, err := sanitizeBlockRange(head.Number, proposal.BlockRange)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bundle: %w", err)
	}
	proposal.BlockRange = &blockRange

	flatTransactions, needGasLimit, needGasPrice, err := flattenTransactions(proposal)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bundle: %w", err)
	}
	gasLimits, err := handler.estimateMissingGas(head, proposal, flatTransactions, needGasLimit)
	if err != nil {
		return nil, err
	}
	cursor := 0
	gasPrice := (*hexutil.Big)(exec.SuggestedGasPrice(head))
	ready, err := transform(proposal,
		func(step RPCExecutionStepProposal) (RPCExecutionStepProposal, error) {
			if _, ok := needGasLimit[cursor]; ok {
				step.Gas = &gasLimits[cursor]
			}
			if _, ok := needGasPrice[cursor]; ok && step.MaxFeePerGas == nil && step.GasPrice == nil {
				if step.MaxPriorityFeePerGas == nil {
					step.GasPrice = gasPrice
				} else {
					step.MaxFeePerGas = gasPrice
				}
			}
			flatTransactions[cursor] = step.CallArguments
			cursor++
			return RPCExecutionStepProposal{
				TolerateFailed:  step.TolerateFailed,
				TolerateInvalid: step.TolerateInvalid,
				CallArguments:   step.CallArguments,
			}, nil
		},
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set proposed transactions defaults: %w", err)
	}
	plan, err := convertProposalToPlan(handler.environment.Signer(), ready)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bundle: %w", err)
	}
	injectPlanHashIntoAccessLists(flatTransactions, plan.Hash())
	planForm, err := NewRPCExecutionPlanComposable(plan)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bundle: %w", err)
	}
	return &RPCPreparedBundle{
		Transactions:  flatTransactions,
		ExecutionPlan: planForm,
	}, nil
}

// estimateMissingGas estimates the gas of every transaction, as
// sonic_estimateGasForTransactions does, when any lacks a limit.
func (handler *handlers) estimateMissingGas(head exec.Head, proposal RPCExecutionProposal, transactions []exec.CallArguments, needGasLimit map[int]hexutil.Uint64) ([]hexutil.Uint64, error) {
	if len(needGasLimit) == 0 {
		return nil, nil
	}
	if err := validateGasEstimationCompatibility(proposal.RPCExecutionPlanGroup); err != nil {
		return nil, fmt.Errorf("failed to prepare bundle: %w", err)
	}
	gasLimits, err := handler.estimateTransactions(head, transactions, exec.Estimation{Block: chain.Latest()})
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bundle: gas estimation failed: %w", err)
	}
	return gasLimits, nil
}

// flattenTransactions traverses the execution proposal and extracts a map of
// transactions indexed by the depth-first order position in the tree,
// while also tracking which transactions need gas limit and gas price defaults filled in.
func flattenTransactions(arguments RPCExecutionProposal) ([]exec.CallArguments, map[int]hexutil.Uint64, map[int]struct{}, error) {
	needGasLimit := map[int]hexutil.Uint64{}
	needGasPrice := map[int]struct{}{}
	flatTransactions := make([]exec.CallArguments, 0)
	_, err := transform(arguments,
		func(transaction RPCExecutionStepProposal) (RPCExecutionStepProposal, error) {

			if transaction.Nonce == nil {
				return transaction, fmt.Errorf("proposed transaction is missing nonce")
			}

			if transaction.From == nil {
				return transaction, fmt.Errorf("proposed transaction is missing from field")
			}

			if transaction.GasPrice != nil && transaction.MaxFeePerGas != nil {
				return transaction, fmt.Errorf("proposed transaction cannot have both gasPrice and maxFeePerGas set")
			}

			flatTransactions = append(flatTransactions, transaction.CallArguments)
			if transaction.Gas == nil || *transaction.Gas == 0 {
				needGasLimit[len(flatTransactions)-1] = 0
			}
			if transaction.GasPrice == nil && transaction.MaxFeePerGas == nil {
				needGasPrice[len(flatTransactions)-1] = struct{}{}
			}
			return transaction, nil
		},
		0,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to flatten transactions: %w", err)
	}
	return flatTransactions, needGasLimit, needGasPrice, nil
}

// validateGasEstimationCompatibility traverses the proposal group tree and returns
// an error if any flag makes sequential gas estimation unreliable.
// Gas estimation assumes all prior transactions executed and their state is visible;
// any flag that allows partial or conditional execution breaks this assumption.
func validateGasEstimationCompatibility(group RPCExecutionPlanGroup) error {
	if group.TolerateFailures {
		return fmt.Errorf("gas estimation not supported when tolerateFailures is set: failed transactions leave state uncertain for subsequent transactions")
	}
	if group.OneOf {
		return fmt.Errorf("gas estimation not supported when oneOf is set: only one branch executes, making state uncertain for sibling branches")
	}
	for _, step := range group.Steps {
		switch step := step.(type) {
		case RPCExecutionStepProposal:
			if step.TolerateFailed {
				return fmt.Errorf("gas estimation not supported when tolerateFailed is set: failed transaction leaves state uncertain for subsequent transactions")
			}
			if step.TolerateInvalid {
				return fmt.Errorf("gas estimation not supported when tolerateInvalid is set: invalid transaction may be skipped, leaving state uncertain for subsequent transactions")
			}
		case RPCExecutionPlanGroup:
			if err := validateGasEstimationCompatibility(step); err != nil {
				return err
			}
		}
	}
	return nil
}

// injectPlanHashIntoAccessLists appends the BundleOnly access-list entry with planHash to every tx.
func injectPlanHashIntoAccessLists(transactions []exec.CallArguments, planHash common.Hash) {
	for index, transaction := range transactions {
		var accessList types.AccessList
		if transaction.AccessList != nil {
			accessList = *transaction.AccessList
		}
		accessList = append(accessList, types.AccessTuple{
			Address:     bundle.BundleOnly,
			StorageKeys: []common.Hash{planHash},
		})
		transaction.AccessList = &accessList
		transactions[index] = transaction
	}
}
