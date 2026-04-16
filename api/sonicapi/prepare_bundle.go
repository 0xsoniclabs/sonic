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
	evmcore "github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/gossip/gasprice/gaspricelimits"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// PrepareBundleArgs represents the arguments for the `sonic_prepareBundle` RPC method.
type PrepareBundleArgs struct {
	// Transactions specifies the ordered list of transactions to be included in the bundle.
	Transactions []ethapi.TransactionArgs `json:"transactions"`
	// EarliestBlock specifies the earliest block number at which the bundle can be executed. This allows
	// users to set a lower bound on when their bundle should be considered for execution, ensuring it is
	// not included in blocks before a certain point in time.
	//
	// If left unspecified, the bundle will be eligible for execution starting from the next block after submission.
	EarliestBlock *hexutil.Uint64 `json:"earliestBlock"`
	// LatestBlock specifies the latest block number at which the bundle can be executed. This allows users
	// to set an upper bound on when their bundle should be considered for execution, ensuring it is
	// not included in blocks after a certain point in time. If the bundle is not executed by this block,
	// it will be considered expired and will not be executed.
	//
	// If left unspecified, the bundle will be eligible for execution until 1024 blocks after EarliestBlock.
	LatestBlock *hexutil.Uint64 `json:"latestBlock"`
}

// RPCPreparedBundle is the return type of the `sonic_prepareBundle` RPC method
type RPCPreparedBundle struct {
	// Transactions specifies the ordered list of transactions to be included in the bundle.
	// These must be signed exactly as provided by the `sonic_prepareBundle` RPC method; any modification
	// will invalidate the execution plan and result in an ill-formed bundle.
	Transactions []ethapi.TransactionArgs `json:"transactions"`
	// ExecutionPlan contains the execution plan that each bundled transaction references. This is provided
	// for verification purposes; users may independently compute and validate the execution plan hash.
	ExecutionPlan RPCExecutionPlan `json:"executionPlan"`
}

// PrepareBundle implements the `sonic_prepareBundle` RPC method.
// This function streamlines the creation of transaction bundles by preparing an execution plan
// based on the provided transaction order, to be executed within a specified block range.
//
// It accepts a list of unsigned transactions, constructs the corresponding execution plan,
// and updates each transaction to include the bundler-only marker, ensuring they are executed
// exclusively as part of the specified plan.
//
// Bundled transactions with uninitialized gas limits will have their gas estimated by this method, which will take into account
// potential state changes from previous transactions in the bundle. However, users can also choose to set gas limits on their own;
// in this case, the provided gas limits will be used without modification.
//
// Bundled transactions with uninitialized gas price fields (GasPrice for access list transactions,
// or both MaxFeePerGas and MaxPriorityFeePerGas for EIP-1559 capable transactions) will have their gas price
// set to the current suggested gas price by this method.
// However, users can also choose to set gas price fields on their own; in this case,
// the provided gas price fields will be used without modification, even if this is zero.
//
// The returned transactions must be signed without altering any fields; any modification may
// invalidate the execution plan and prevent the bundle from being executed.
func (a *PublicBundleAPI) PrepareBundle(
	ctx context.Context,
	args PrepareBundleArgs,
) (*RPCPreparedBundle, error) {

	gasCap := a.b.RPCGasCap()
	basefee := a.b.MinGasPrice()

	// Estimate gas for all transactions if any has an uninitialized gas limit.
	var gasLimits []hexutil.Uint64
	for _, tx := range args.Transactions {
		if tx.Gas == nil || *tx.Gas == 0 {
			estimated, err := a.EstimateGasForTransactions(ctx, args.Transactions, nil, nil, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to prepare bundle: gas estimation failed: %w", err)
			}
			gasLimits = estimated.GasLimits
			break
		}
	}

	var currentBlock *evmcore.EvmBlock
	if block := a.b.CurrentBlock(); block != nil {
		currentBlock = block
	} else {
		return nil, fmt.Errorf("failed to prepare bundle: unable to retrieve current block number")
	}

	gasPrice := a.suggestGasPrice(currentBlock)
	fillTransactionDefaults(args.Transactions, gasLimits, gasPrice)

	chainID := a.b.ChainID()
	signer := types.LatestSignerForChainID(chainID)

	// Build execution steps and TxReference map.
	steps := make([]bundle.ExecutionStep, len(args.Transactions))
	for i, txArgs := range args.Transactions {

		if txArgs.Nonce == nil {
			return nil, fmt.Errorf("failed to prepare bundle: transaction %d is missing nonce", i)
		}

		msg, err := txArgs.ToMessage(gasCap, basefee, log.Root())
		if err != nil {
			return nil, fmt.Errorf("failed to prepare bundle: transaction %d conversion error: %w", i, err)
		}

		tx, err := asTransaction(msg)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare bundle: transaction %d conversion error: %w", i, err)
		}

		steps[i] = bundle.NewTxStep(bundle.TxReference{
			From: msg.From,
			Hash: signer.Hash(tx),
		})
	}

	blockRange, err := resolveBlockRange(currentBlock.NumberU64(), args.EarliestBlock, args.LatestBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bundle: %w", err)
	}

	var root bundle.ExecutionStep
	if len(steps) == 1 {
		root = steps[0]
	} else {
		root = bundle.NewAllOfStep(steps...)
	}

	plan := bundle.ExecutionPlan{
		Root:  root,
		Range: blockRange,
	}

	injectPlanHashIntoAccessLists(args.Transactions, plan.Hash())

	result := RPCPreparedBundle{
		Transactions:  args.Transactions,
		ExecutionPlan: NewRPCExecutionPlan(plan),
	}
	return &result, nil
}

// fillTransactionDefaults fills missing gas limits and gas price fields.
// Gas limits are set from gasLimits only when tx.Gas is nil or zero. Gas price fields are
// set from gasPrice only when both tx.GasPrice and tx.MaxFeePerGas are unset.
func fillTransactionDefaults(txs []ethapi.TransactionArgs, gasLimits []hexutil.Uint64, gasPrice *hexutil.Big) {
	for i := range txs {
		if i < len(gasLimits) && (txs[i].Gas == nil || *txs[i].Gas == 0) {
			txs[i].Gas = &gasLimits[i]
		}
		if txs[i].GasPrice == nil && txs[i].MaxFeePerGas == nil {
			if txs[i].MaxPriorityFeePerGas == nil {
				txs[i].GasPrice = gasPrice
			} else {
				txs[i].MaxFeePerGas = gasPrice
			}
		}
	}
}

// resolveBlockRange computes the effective block range for a bundle given the current
// block number and optional earliest/latest overrides from the user.
func resolveBlockRange(currentBlock uint64, earliest, latest *hexutil.Uint64) (bundle.BlockRange, error) {

	if latest != nil && earliest != nil {
		if uint64(*latest) < uint64(*earliest) {
			return bundle.BlockRange{}, fmt.Errorf("invalid block range: latest block %d is earlier than earliest block %d", *latest, *earliest)
		}
		if uint64(*latest)-uint64(*earliest)+1 > bundle.MaxBlockRange {
			return bundle.BlockRange{}, fmt.Errorf("invalid block range: range %d is too large; must be at most %d blocks", uint64(*latest)-uint64(*earliest)+1, bundle.MaxBlockRange)
		}
	}

	var r bundle.BlockRange
	if earliest != nil {
		r = bundle.MakeMaxRangeStartingAt(uint64(*earliest))
	} else {
		r = bundle.MakeMaxRangeStartingAt(currentBlock + 1)
	}

	if latest != nil {
		r.Latest = uint64(*latest)
	}

	return r, nil
}

// injectPlanHashIntoAccessLists appends the bundle-only access list entry carrying
// the execution plan hash to every transaction in-place.
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

// asTransaction converts a Message to a Transaction, ensuring that unsupported
// transaction types (e.g. blob transactions) are not included in bundles.
func asTransaction(msg *core.Message) (*types.Transaction, error) {

	if len(msg.BlobHashes) != 0 || msg.BlobGasFeeCap != nil {
		return nil, fmt.Errorf("blob transactions are not supported in bundles")
	}
	if len(msg.SetCodeAuthorizations) != 0 {
		return nil, fmt.Errorf("transactions with set code authorization are not supported in bundles")
	}

	if msg.GasPrice == nil || msg.GasPrice.Sign() == 0 {
		// use dynamic fee transaction
		return types.NewTx(&types.DynamicFeeTx{
			To:         msg.To,
			Nonce:      msg.Nonce,
			Gas:        msg.GasLimit,
			GasFeeCap:  msg.GasFeeCap,
			GasTipCap:  msg.GasTipCap,
			Value:      msg.Value,
			Data:       msg.Data,
			AccessList: msg.AccessList,
		}), nil
	} else {
		// use access list transaction
		return types.NewTx(&types.AccessListTx{
			To:         msg.To,
			Nonce:      msg.Nonce,
			Gas:        msg.GasLimit,
			GasPrice:   msg.GasPrice,
			Value:      msg.Value,
			Data:       msg.Data,
			AccessList: msg.AccessList,
		}), nil
	}
}

// suggestGasPrice returns the suggested gas price for new transactions,
// which is used to fill in missing gas price fields in bundled transactions.
func (a *PublicBundleAPI) suggestGasPrice(block *evmcore.EvmBlock) *hexutil.Big {
	price := block.Header().BaseFee
	price = gaspricelimits.GetSuggestedGasPriceForNewTransactions(price)
	return (*hexutil.Big)(price)
}
