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

package sonicapi

import (
	"context"
	"fmt"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	types "github.com/ethereum/go-ethereum/core/types"
)

// PrepareBundleArgs are the arguments for the sonic_prepareBundle RPC method.
type PrepareBundleArgs struct {
	SetTransactionsDefaults bool `json:"setTransactionsDefaults"`
	EstimateGasLimits       bool `json:"estimateGasLimits"`

	RPCExecutionProposal
}

// RPCPreparedBundle is the JSON-serializable representation of the prepared bundle that is
// returned by the sonic_prepareBundle RPC method.
type RPCPreparedBundle struct {
	Transactions  []ethapi.TransactionArgs   `json:"transactions"`
	ExecutionPlan RPCExecutionPlanComposable `json:"executionPlan"`
}

// PrepareBundle implements the `sonic_prepareBundle` RPC method.
// It accepts a structured execution plan where leaves are unsigned transactions
// (with optional execution flags), constructs the corresponding bundle execution plan,
// and updates each transaction to include the bundler-only marker.
//
// Transactions with uninitialized gas limits will have their gas estimated taking into
// account potential state changes from previous transactions in depth-first leaf order.
// Transactions with uninitialized gas price fields will have them set to the current
// suggested gas price.
//
// The returned transactions must be signed without altering any fields.
func (a *PublicBundleAPI) PrepareBundle(
	ctx context.Context,
	args PrepareBundleArgs,
) (*RPCPreparedBundle, error) {

	signer := types.LatestSignerForChainID(a.b.ChainID())

	// Estimate gas
	if args.EstimateGasLimits {
		return nil, fmt.Errorf("gas limit estimation for bundles is not yet implemented")
	}

	// Fill transaction defaults
	ready, err := transform(args.RPCExecutionProposal,
		func(step RPCExecutionStepProposal) (RPCExecutionStepProposal, error) {
			return RPCExecutionStepProposal{
				TolerateFailed:  step.TolerateFailed,
				TolerateInvalid: step.TolerateInvalid,
				TransactionArgs: fillTransactionDefaults(step.TransactionArgs, 0, nil),
			}, nil

		})
	if err != nil {
		return nil, fmt.Errorf("failed to set proposed transactions defaults")
	}

	// Convert proposal to execution plan
	plan, err := convertProposalToPlan(signer, ready)
	if err != nil {
		return nil, fmt.Errorf("failed to construct bundle execution plan: %w", err)
	}
	planHash := plan.Hash()

	// Tag transactions with execution plan
	transactions := []ethapi.TransactionArgs{}
	_, err = transform(ready, func(step RPCExecutionStepProposal) (RPCExecutionStepProposal, error) {
		tx := step.TransactionArgs
		tx.AccessList = addBundleMarkerToAccessList(planHash, tx.AccessList)
		transactions = append(transactions, tx)
		return step, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to extract transactions from execution proposal: %w", err)
	}

	// Convert execution plan to RPC format for response
	rpcPlan, err := NewRPCExecutionPlanComposable(plan)
	if err != nil {
		return nil, fmt.Errorf("failed to convert execution plan for response: %w", err)
	}

	return &RPCPreparedBundle{
		Transactions:  transactions,
		ExecutionPlan: rpcPlan,
	}, nil
}

func addBundleMarkerToAccessList(planHash common.Hash, accessList *types.AccessList) *types.AccessList {
	if accessList == nil {
		accessList = &types.AccessList{}
	}

	*accessList = append(*accessList, types.AccessTuple{
		Address: bundle.BundleOnly,
		StorageKeys: []common.Hash{
			planHash,
		},
	})

	return accessList
}

// fillTransactionDefaults fills missing gas limits and gas price fields.
// Gas limits are set from gasLimits only when tx.Gas is nil or zero. Gas price fields are
// set from gasPrice only when both tx.GasPrice and tx.MaxFeePerGas are unset.
func fillTransactionDefaults(tx ethapi.TransactionArgs, gasLimit hexutil.Uint64, gasPrice *hexutil.Big) ethapi.TransactionArgs {
	// TODO: placeholder implementation
	return tx
}

func convertProposalToPlan(signer types.Signer, proposal RPCExecutionProposal) (bundle.ExecutionPlan, error) {

	root, err := convertProposalToPlanInternal(signer, &proposal.RPCExecutionPlanGroup)
	if err != nil {
		return bundle.ExecutionPlan{}, err
	}

	return bundle.ExecutionPlan{
		Range: proposal.BlockRange.toBundleBlockRange(),
		Root:  root,
	}, nil
}

func convertProposalToPlanInternal(signer types.Signer, proposalStep any) (bundle.ExecutionStep, error) {
	empty := bundle.ExecutionStep{}

	switch step := proposalStep.(type) {
	case RPCExecutionStepProposal:
		if step.From == nil {
			return empty, fmt.Errorf("transaction in bundle must include from")
		}

		tx := step.ToTransaction()
		hash := signer.Hash(tx)

		return bundle.NewTxStep(bundle.TxReference{
			From: *step.From,
			Hash: hash,
		}).WithFlags(func() bundle.ExecutionFlags {
			flags := bundle.EF_Default
			if step.TolerateFailed {
				flags |= bundle.EF_TolerateFailed
			}
			if step.TolerateInvalid {
				flags |= bundle.EF_TolerateInvalid
			}
			return flags
		}()), nil

	case *RPCExecutionPlanGroup:

		if step.Steps == nil {
			return empty, fmt.Errorf("execution plan group must include steps")
		}

		steps := make([]bundle.ExecutionStep, len(step.Steps))
		for i, stepLevel := range step.Steps {
			step, err := convertProposalToPlanInternal(signer, stepLevel)
			if err != nil {
				return empty, fmt.Errorf("invalid execution plan level: %w", err)
			}
			steps[i] = step
		}

		return bundle.NewGroupStep(
			step.OneOf,
			steps...,
		), nil
	}

	return empty, fmt.Errorf("invalid execution proposal level: must have either executionStep or group")
}
