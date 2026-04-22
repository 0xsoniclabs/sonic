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
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/gossip/gasprice/gaspricelimits"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// PrepareBundleTxStep is a leaf node in the structured bundle input, representing
// a single transaction together with its execution flags.
type PrepareBundleTxStep struct {
	Transaction     ethapi.TransactionArgs `json:"transaction"`
	TolerateFailed  bool                   `json:"tolerateFailed,omitempty"`
	TolerateInvalid bool                   `json:"tolerateInvalid,omitempty"`
}

// PrepareBundleGroupStep is an interior node in the structured bundle input,
// grouping child steps as either AllOf (default) or OneOf.
type PrepareBundleGroupStep struct {
	OneOf            bool                `json:"oneOf,omitempty"`
	TolerateFailures bool                `json:"tolerateFailures,omitempty"`
	Steps            []PrepareBundleStep `json:"steps"`
}

// PrepareBundleStep is a polymorphic step in the structured bundle input.
// A step is either a transaction leaf (has a "transaction" JSON field) or
// a group of sub-steps (has a "steps" JSON field).
type PrepareBundleStep struct {
	Tx    *PrepareBundleTxStep
	Group *PrepareBundleGroupStep
}

// UnmarshalJSON discriminates between a leaf tx step and a group step
// based on the presence of "transaction" vs "steps" in the JSON object.
func (s *PrepareBundleStep) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to parse bundle step: %w", err)
	}
	_, hasTx := raw["transaction"]
	_, hasSteps := raw["steps"]
	switch {
	case hasTx && !hasSteps:
		s.Tx = new(PrepareBundleTxStep)
		return json.Unmarshal(data, s.Tx)
	case hasSteps && !hasTx:
		s.Group = new(PrepareBundleGroupStep)
		return json.Unmarshal(data, s.Group)
	case hasTx && hasSteps:
		return fmt.Errorf("bundle step must have either 'transaction' or 'steps', not both")
	default:
		return fmt.Errorf("bundle step must have either 'transaction' or 'steps'")
	}
}

// PrepareBundleArgs represents the arguments for the `sonic_prepareBundle` RPC method.
// The root level is a group (embeds PrepareBundleGroupStep) whose Steps contain
// transaction leaves and/or nested groups.
type PrepareBundleArgs struct {
	PrepareBundleGroupStep
	// EarliestBlock specifies the earliest block number at which the bundle can be executed.
	// If left unspecified, defaults to the next block after submission.
	EarliestBlock *hexutil.Uint64 `json:"earliestBlock"`
	// LatestBlock specifies the latest block number at which the bundle can be executed.
	// If left unspecified, defaults to EarliestBlock + MaxBlockRange - 1.
	LatestBlock *hexutil.Uint64 `json:"latestBlock"`
}

// RPCPreparedBundle is the return type of the `sonic_prepareBundle` RPC method.
type RPCPreparedBundle struct {
	// Transactions is the flat ordered list of transactions (depth-first leaf order)
	// to be signed. They must be signed without altering any fields; any modification
	// will invalidate the execution plan.
	Transactions []ethapi.TransactionArgs `json:"transactions"`
	// ExecutionPlan contains the structured execution plan that each bundled transaction
	// references. This is provided for verification purposes.
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

	flatTxs := collectLeafTxArgsFromGroup(args.PrepareBundleGroupStep)

	if len(flatTxs) == 0 {
		return &RPCPreparedBundle{}, nil
	}

	gasCap := a.b.RPCGasCap()
	basefee := a.b.MinGasPrice()

	// Estimate gas for all transactions if any has an uninitialized gas limit.
	var gasLimits []hexutil.Uint64
	for _, tx := range flatTxs {
		if tx.Gas == nil || *tx.Gas == 0 {
			estimated, err := a.EstimateGasForTransactions(ctx, flatTxs, nil, nil, nil)
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
	fillTransactionDefaults(flatTxs, gasLimits, gasPrice)

	chainID := a.b.ChainID()
	signer := types.LatestSignerForChainID(chainID)

	builder := &prepareBundleBuilder{
		flatTxArgs: flatTxs,
		gasCap:     gasCap,
		basefee:    basefee,
		signer:     signer,
	}
	root, err := builder.buildGroupStep(&args.PrepareBundleGroupStep)
	if err != nil {
		return nil, err
	}

	blockRange, err := resolveBlockRange(currentBlock.NumberU64(), args.EarliestBlock, args.LatestBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bundle: %w", err)
	}

	plan := bundle.ExecutionPlan{
		Root:  root,
		Range: blockRange,
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

// prepareBundleBuilder threads a cursor through the flat filled tx list while
// recursively building the bundle.ExecutionStep tree from the structured input.
type prepareBundleBuilder struct {
	flatTxArgs []ethapi.TransactionArgs
	cursor     int
	gasCap     uint64
	basefee    *big.Int
	signer     types.Signer
}

func (b *prepareBundleBuilder) buildStep(step PrepareBundleStep) (bundle.ExecutionStep, error) {
	if step.Tx != nil {
		return b.buildTxStep(step.Tx)
	}
	return b.buildGroupStep(step.Group)
}

func (b *prepareBundleBuilder) buildTxStep(tx *PrepareBundleTxStep) (bundle.ExecutionStep, error) {
	idx := b.cursor
	txArgs := b.flatTxArgs[idx]
	b.cursor++

	if txArgs.Nonce == nil {
		return bundle.ExecutionStep{}, fmt.Errorf("failed to prepare bundle: transaction %d is missing nonce", idx)
	}

	msg, err := txArgs.ToMessage(b.gasCap, b.basefee, log.Root())
	if err != nil {
		return bundle.ExecutionStep{}, fmt.Errorf("failed to prepare bundle: transaction %d conversion error: %w", idx, err)
	}

	bundleTx, err := asTransaction(msg)
	if err != nil {
		return bundle.ExecutionStep{}, fmt.Errorf("failed to prepare bundle: transaction %d conversion error: %w", idx, err)
	}

	step := bundle.NewTxStep(bundle.TxReference{
		From: msg.From,
		Hash: b.signer.Hash(bundleTx),
	})

	var flags bundle.ExecutionFlags
	if tx.TolerateFailed {
		flags |= bundle.EF_TolerateFailed
	}
	if tx.TolerateInvalid {
		flags |= bundle.EF_TolerateInvalid
	}
	if flags != bundle.EF_Default {
		step = step.WithFlags(flags)
	}
	return step, nil
}

func (b *prepareBundleBuilder) buildGroupStep(g *PrepareBundleGroupStep) (bundle.ExecutionStep, error) {
	subSteps := make([]bundle.ExecutionStep, len(g.Steps))
	for i, s := range g.Steps {
		sub, err := b.buildStep(s)
		if err != nil {
			return bundle.ExecutionStep{}, err
		}
		subSteps[i] = sub
	}

	// Unwrap single child with no group modifiers — mirrors toBundleExecutionGroup convention.
	if !g.OneOf && !g.TolerateFailures && len(subSteps) == 1 {
		return subSteps[0], nil
	}

	group := bundle.NewGroupStep(g.OneOf, subSteps...)
	if g.TolerateFailures {
		group = group.WithFlags(bundle.EF_TolerateFailed)
	}
	return group, nil
}

// collectLeafTxArgsFromGroup does a depth-first walk of the group, collecting
// all leaf TransactionArgs in order for gas estimation and default-filling.
func collectLeafTxArgsFromGroup(g PrepareBundleGroupStep) []ethapi.TransactionArgs {
	var out []ethapi.TransactionArgs
	for _, step := range g.Steps {
		collectLeafTxArgs(step, &out)
	}
	return out
}

func collectLeafTxArgs(step PrepareBundleStep, out *[]ethapi.TransactionArgs) {
	if step.Tx != nil {
		*out = append(*out, step.Tx.Transaction)
		return
	}
	if step.Group != nil {
		for _, sub := range step.Group.Steps {
			collectLeafTxArgs(sub, out)
		}
	}
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

	if r.Latest-r.Earliest+1 > bundle.MaxBlockRange {
		return bundle.BlockRange{}, fmt.Errorf("invalid block range: range %d is too large; must be at most %d blocks", r.Latest-r.Earliest+1, bundle.MaxBlockRange)
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

	// when gasprice and gas fee cap or gas tip cap are both set, return error
	if msg.GasPrice != nil && msg.GasPrice.Sign() != 0 && (msg.GasFeeCap != nil || msg.GasTipCap != nil) {
		return nil, fmt.Errorf("cannot set both gas price and gas fee cap or gas tip cap")
	}

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
