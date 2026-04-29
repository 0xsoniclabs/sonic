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
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/gossip/gasprice/gaspricelimits"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// RPCPreparedBundle is the return type of the sonic_prepareBundle RPC method.
type RPCPreparedBundle struct {
	// Transactions is the flat depth-first ordered list; must be signed without modification.
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
	args RPCExecutionProposal,
) (*RPCPreparedBundle, error) {

	var currentBlock *evmcore.EvmBlock
	if block := a.b.CurrentBlock(); block != nil {
		currentBlock = block
	} else {
		return nil, fmt.Errorf("failed to prepare bundle: unable to retrieve current block number")
	}

	blockRange, err := sanitizeBlockRange(currentBlock.NumberU64(), args.BlockRange)
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
		estimated, err := a.EstimateGasForTransactions(ctx, flatTxs, nil, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare bundle: gas estimation failed: %w", err)
		}
		gasLimits = estimated.GasLimits
	}

	cursor := 0
	gasPrice := a.suggestGasPrice(currentBlock)
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
			flatTxs[cursor] = step.TransactionArgs
			cursor++

			return RPCExecutionStepProposal{
				TolerateFailed:  step.TolerateFailed,
				TolerateInvalid: step.TolerateInvalid,
				TransactionArgs: step.TransactionArgs,
			}, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set proposed transactions defaults: %w", err)
	}

	chainID := a.b.ChainID()
	signer := types.LatestSignerForChainID(chainID)
	plan, err := convertProposalToPlan(signer, ready)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bundle: %w", err)
	}

	injectPlanHashIntoAccessLists(flatTxs, plan.Hash())

	rpcPlan, err := NewRPCExecutionPlanComposable(plan)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bundle: %w", err)
	}

	return &RPCPreparedBundle{
		Transactions:  flatTxs,
		ExecutionPlan: rpcPlan,
	}, nil
}

// flattenTransactions traverses the execution proposal and extracts a map of
// transactions indexed by the depth-first order position in the tree,
// while also tracking which transactions need gas limit and gas price defaults filled in.
func flattenTransactions(args RPCExecutionProposal) ([]ethapi.TransactionArgs, map[int]hexutil.Uint64, map[int]struct{}, error) {
	needGasLimit := map[int]hexutil.Uint64{}
	needGasPrice := map[int]struct{}{}
	flatTxs := make([]ethapi.TransactionArgs, 0)
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

			flatTxs = append(flatTxs, tx.TransactionArgs)
			if tx.Gas == nil || *tx.Gas == 0 {
				needGasLimit[len(flatTxs)-1] = 0
			}
			if tx.GasPrice == nil && tx.MaxFeePerGas == nil {
				needGasPrice[len(flatTxs)-1] = struct{}{}
			}
			return tx, nil
		},
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to flatten transactions: %w", err)
	}
	return flatTxs, needGasLimit, needGasPrice, nil
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

// injectPlanHashIntoAccessLists appends the BundleOnly access-list entry with planHash to every tx.
func injectPlanHashIntoAccessLists(txs []ethapi.TransactionArgs, planHash common.Hash) {
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

// suggestGasPrice returns the suggested gas price based on the current block's base fee.
func (a *PublicBundleAPI) suggestGasPrice(block *evmcore.EvmBlock) *hexutil.Big {
	price := block.Header().BaseFee
	price = gaspricelimits.GetSuggestedGasPriceForNewTransactions(price)
	return (*hexutil.Big)(price)
}
