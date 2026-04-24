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
	"slices"

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

	blockRange, err := validateBlockRange(currentBlock.NumberU64(), args.BlockRange)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bundle: %w", err)
	}
	args.BlockRange = &blockRange

	gasPrice := a.suggestGasPrice(currentBlock)
	chainID := a.b.ChainID()
	signer := types.LatestSignerForChainID(chainID)

	flatTxs := make([]ethapi.TransactionArgs, 0)
	err = traverse(args,
		func(step RPCExecutionStepProposal) error {
			flatTxs = append(flatTxs, step.TransactionArgs)
			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bundle: %w", err)
	}

	// Estimate gas for all transactions if any has an uninitialized gas limit.
	var gasLimits []hexutil.Uint64
	if slices.ContainsFunc(flatTxs, func(tx ethapi.TransactionArgs) bool {
		return tx.Gas == nil || *tx.Gas == 0 || (tx.GasPrice == nil && tx.MaxFeePerGas == nil)
	}) {
		estimated, err := a.EstimateGasForTransactions(ctx, flatTxs, nil, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare bundle: gas estimation failed: %w", err)
		}
		gasLimits = estimated.GasLimits
	}

	cursor := 0
	// Fill transaction defaults
	ready, err := transform(args,
		func(step RPCExecutionStepProposal) (RPCExecutionStepProposal, error) {
			var txArgs ethapi.TransactionArgs
			if len(gasLimits) > 0 && cursor < len(gasLimits) {
				txArgs = fillTransactionDefaults(step.TransactionArgs, &gasLimits[cursor], gasPrice)
				flatTxs[cursor] = txArgs
			} else {
				txArgs = fillTransactionDefaults(step.TransactionArgs, nil, gasPrice)
				flatTxs[cursor] = txArgs
			}

			if txArgs.Nonce == nil {
				return step, fmt.Errorf("transaction %d is missing nonce", cursor)
			}

			if txArgs.To == nil {
				return step, fmt.Errorf("transaction %d is missing to", cursor)
			}

			if txArgs.From == nil {
				return step, fmt.Errorf("transaction %d is missing from", cursor)
			}
			cursor++

			return RPCExecutionStepProposal{
				TolerateFailed:  step.TolerateFailed,
				TolerateInvalid: step.TolerateInvalid,
				TransactionArgs: txArgs,
			}, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set proposed transactions defaults: %w", err)
	}

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

// fillTransactionDefaults sets default gas and gas price values for a transaction if they are not already set.
func fillTransactionDefaults(args ethapi.TransactionArgs, gas *hexutil.Uint64, gasPrice *hexutil.Big) ethapi.TransactionArgs {

	// Set default gas limit if missing or zero.
	if args.Gas == nil || *args.Gas == 0 {
		args.Gas = gas
	}

	// Set default gas price if missing and not a type 2 transaction.
	if args.GasPrice == nil && args.MaxFeePerGas == nil {
		if args.MaxPriorityFeePerGas == nil {
			args.GasPrice = gasPrice
		} else {
			args.MaxFeePerGas = gasPrice
		}
	}

	return args
}

// validateBlockRange checks that the provided block range is valid and within
// allowed limits, defaulting to a sensible range if not provided.
func validateBlockRange(currentBlock uint64, blockRange *RPCRange) (RPCRange, error) {
	if blockRange == nil {
		maxRange := bundle.MakeMaxRangeStartingAt(currentBlock + 1)
		return RPCRange{
			Earliest: hexutil.Uint64(maxRange.Earliest),
			Latest:   hexutil.Uint64(maxRange.Latest),
		}, nil
	}

	if blockRange.Latest < hexutil.Uint64(currentBlock) {
		return RPCRange{}, fmt.Errorf("invalid block range: latest block %d is earlier than current block %d", blockRange.Latest, currentBlock)
	}

	if uint64(blockRange.Latest) < uint64(blockRange.Earliest) {
		return RPCRange{}, fmt.Errorf("invalid block range: latest block %d is earlier than earliest block %d", blockRange.Latest, blockRange.Earliest)
	}

	if uint64(blockRange.Latest-blockRange.Earliest+1) > bundle.MaxBlockRange {
		return RPCRange{}, fmt.Errorf("invalid block range: range %d is too large; must be at most %d blocks", blockRange.Latest-blockRange.Earliest+1, bundle.MaxBlockRange)
	}

	return RPCRange{
		Earliest: blockRange.Earliest,
		Latest:   blockRange.Latest,
	}, nil
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
