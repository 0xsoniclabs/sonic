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
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

// RPCPreparedBundle is a bundle ready to be signed: its transactions, in
// depth-first order, to be signed without modification, and its plan.
type RPCPreparedBundle struct {
	Transactions  []exec.CallArgs            `json:"transactions"`
	ExecutionPlan RPCExecutionPlanComposable `json:"executionPlan"`
}

type prepareArgs struct {
	Proposal RPCExecutionProposal `rpc:"proposal"`
}

// prepareBundle fills in the range, gas and prices of a proposed bundle, and
// marks each transaction as part of the bundle's plan.
func (a *api) prepareBundle(ctx context.Context, p prepareArgs) (any, error) {
	args := p.Proposal
	head := a.env.Latest(ctx)
	blockRange, err := sanitizeBlockRange(head.Number(), args.BlockRange)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bundle: %w", err)
	}
	args.BlockRange = &blockRange

	flatTxs, needGasLimit, needGasPrice, err := flattenTransactions(args)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bundle: %w", err)
	}
	var gasLimits []hexutil.Uint64
	if len(needGasLimit) > 0 {
		if err := validateGasEstimationCompatibility(args.RPCExecutionPlanGroup); err != nil {
			return nil, fmt.Errorf("failed to prepare bundle: %w", err)
		}
		estimated, err := a.estimate(ctx, head, flatTxs, chain.LatestRef(), nil, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare bundle: gas estimation failed: %w", err)
		}
		gasLimits = estimated.GasLimits
	}

	cursor := 0
	gasPrice := (*hexutil.Big)(a.env.SuggestedGasPrice(head))
	ready, err := transform(args,
		func(step RPCExecutionStepProposal) (RPCExecutionStepProposal, error) {
			if _, ok := needGasLimit[cursor]; ok {
				step.Gas = &gasLimits[cursor]
			}
			if _, ok := needGasPrice[cursor]; ok {
				if step.MaxFeePerGas == nil && step.GasPrice == nil {
					if step.MaxPriorityFeePerGas == nil {
						step.GasPrice = gasPrice
					} else {
						step.MaxFeePerGas = gasPrice
					}
				}
			}
			flatTxs[cursor] = step.CallArgs
			cursor++
			return RPCExecutionStepProposal{
				TolerateFailed:  step.TolerateFailed,
				TolerateInvalid: step.TolerateInvalid,
				CallArgs:        step.CallArgs,
			}, nil
		},
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set proposed transactions defaults: %w", err)
	}
	plan, err := convertProposalToPlan(a.env.Signer(), ready)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bundle: %w", err)
	}
	injectPlanHashIntoAccessLists(flatTxs, plan.Hash())
	rpcPlan, err := NewRPCExecutionPlanComposable(plan)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bundle: %w", err)
	}
	return &RPCPreparedBundle{Transactions: flatTxs, ExecutionPlan: rpcPlan}, nil
}

// flattenTransactions lists the transactions of a proposal in depth-first
// order, with the positions of those that need a gas limit and a gas price.
func flattenTransactions(args RPCExecutionProposal) ([]exec.CallArgs, map[int]hexutil.Uint64, map[int]struct{}, error) {
	needGasLimit := map[int]hexutil.Uint64{}
	needGasPrice := map[int]struct{}{}
	flatTxs := make([]exec.CallArgs, 0)
	_, err := transform(args,
		func(tx RPCExecutionStepProposal) (RPCExecutionStepProposal, error) {
			if tx.Nonce == nil {
				return tx, fmt.Errorf("proposed transaction is missing nonce")
			}
			if tx.From == nil {
				return tx, fmt.Errorf("proposed transaction is missing from field")
			}
			if tx.GasPrice != nil && tx.MaxFeePerGas != nil {
				return tx, fmt.Errorf("proposed transaction cannot have both gasPrice and maxFeePerGas set")
			}
			flatTxs = append(flatTxs, tx.CallArgs)
			if tx.Gas == nil || *tx.Gas == 0 {
				needGasLimit[len(flatTxs)-1] = 0
			}
			if tx.GasPrice == nil && tx.MaxFeePerGas == nil {
				needGasPrice[len(flatTxs)-1] = struct{}{}
			}
			return tx, nil
		},
		0,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to flatten transactions: %w", err)
	}
	return flatTxs, needGasLimit, needGasPrice, nil
}

// validateGasEstimationCompatibility refuses to estimate a plan whose
// transactions may not all run: the state of the later ones is uncertain.
func validateGasEstimationCompatibility(group RPCExecutionPlanGroup) error {
	if group.TolerateFailures {
		return fmt.Errorf("gas estimation not supported when tolerateFailures is set: failed transactions leave state uncertain for subsequent transactions")
	}
	if group.OneOf {
		return fmt.Errorf("gas estimation not supported when oneOf is set: only one branch executes, making state uncertain for sibling branches")
	}
	for _, step := range group.Steps {
		switch s := step.(type) {
		case RPCExecutionStepProposal:
			if s.TolerateFailed {
				return fmt.Errorf("gas estimation not supported when tolerateFailed is set: failed transaction leaves state uncertain for subsequent transactions")
			}
			if s.TolerateInvalid {
				return fmt.Errorf("gas estimation not supported when tolerateInvalid is set: invalid transaction may be skipped, leaving state uncertain for subsequent transactions")
			}
		case RPCExecutionPlanGroup:
			if err := validateGasEstimationCompatibility(s); err != nil {
				return err
			}
		}
	}
	return nil
}

// injectPlanHashIntoAccessLists marks each transaction as part of the plan:
// the bundle-only address with the plan's hash in its access list.
func injectPlanHashIntoAccessLists(txs []exec.CallArgs, planHash common.Hash) {
	for i, tx := range txs {
		var accessList types.AccessList
		if tx.AccessList != nil {
			accessList = *tx.AccessList
		}
		accessList = append(accessList, types.AccessTuple{
			Address:     bundle.BundleOnly,
			StorageKeys: []common.Hash{planHash},
		})
		tx.AccessList = &accessList
		txs[i] = tx
	}
}
