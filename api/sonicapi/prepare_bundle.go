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
	"slices"

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

// PrepareBundleTxStep is a leaf node: one transaction with its execution flags.
type PrepareBundleTxStep struct {
	Transaction     ethapi.TransactionArgs `json:"transaction"`
	TolerateFailed  bool                   `json:"tolerateFailed,omitempty"`
	TolerateInvalid bool                   `json:"tolerateInvalid,omitempty"`
}

// PrepareBundleGroup is an interior node grouping child entries as AllOf (default) or OneOf.
type PrepareBundleGroup struct {
	OneOf            bool                 `json:"oneOf,omitempty"`
	TolerateFailures bool                 `json:"tolerateFailures,omitempty"`
	Entries          []PrepareBundleEntry `json:"entries,omitempty"`
}

// PrepareBundleEntry is a polymorphic bundle step: either a tx leaf or a group.
type PrepareBundleEntry struct {
	Tx    *PrepareBundleTxStep
	Group *PrepareBundleGroup
}

// UnmarshalJSON discriminates between a tx leaf ("transaction") and a group ("entries").
func (s *PrepareBundleEntry) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to parse bundle step: %w", err)
	}
	_, hasTx := raw["transaction"]
	_, hasEntries := raw["entries"]
	switch {
	case hasTx && !hasEntries:
		s.Tx = new(PrepareBundleTxStep)
		return json.Unmarshal(data, s.Tx)
	case hasEntries && !hasTx:
		s.Group = new(PrepareBundleGroup)
		return json.Unmarshal(data, s.Group)
	case hasTx && hasEntries:
		return fmt.Errorf("bundle step must have either 'transaction' or 'entries', not both")
	default:
		return fmt.Errorf("bundle step must have either 'transaction' or 'entries'")
	}
}

// PrepareBundleArgs are the arguments for the sonic_prepareBundle RPC method.
type PrepareBundleArgs struct {
	PrepareBundleGroup
	// Transactions is a flat all-of shorthand; mutually exclusive with Entries.
	Transactions []ethapi.TransactionArgs `json:"transactions,omitempty"`
	// EarliestBlock defaults to currentBlock+1 if unset.
	EarliestBlock *hexutil.Uint64 `json:"earliestBlock,omitempty"`
	// LatestBlock defaults to EarliestBlock+MaxBlockRange-1 if unset.
	LatestBlock *hexutil.Uint64 `json:"latestBlock,omitempty"`
}

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
	args PrepareBundleArgs,
) (*RPCPreparedBundle, error) {

	if len(args.Transactions) > 0 && len(args.Entries) > 0 {
		return nil, fmt.Errorf("cannot specify both 'transactions' and 'entries' in bundle input")
	}
	if len(args.Transactions) > 0 {
		entries := make([]PrepareBundleEntry, len(args.Transactions))
		for i, tx := range args.Transactions {
			tx := tx
			entries[i] = PrepareBundleEntry{Tx: &PrepareBundleTxStep{Transaction: tx}}
		}
		args.Entries = entries
	}

	flatTxs := collectLeafTxArgsFromGroup(args.PrepareBundleGroup)

	if len(flatTxs) == 0 {
		return &RPCPreparedBundle{}, nil
	}

	gasCap := a.b.RPCGasCap()
	basefee := a.b.MinGasPrice()

	// Estimate gas for all transactions if any has an uninitialized gas limit.
	var gasLimits []hexutil.Uint64
	if slices.ContainsFunc(flatTxs, func(tx ethapi.TransactionArgs) bool {
		return tx.Gas == nil || *tx.Gas == 0
	}) {
		estimated, err := a.EstimateGasForTransactions(ctx, flatTxs, nil, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare bundle: gas estimation failed: %w", err)
		}
		gasLimits = estimated.GasLimits
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
	root, err := builder.buildGroupStep(&args.PrepareBundleGroup)
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

// prepareBundleBuilder threads a cursor through flatTxArgs while building the ExecutionStep tree.
type prepareBundleBuilder struct {
	flatTxArgs []ethapi.TransactionArgs
	cursor     int
	gasCap     uint64
	basefee    *big.Int
	signer     types.Signer
}

func (b *prepareBundleBuilder) buildStep(step PrepareBundleEntry) (bundle.ExecutionStep, error) {
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

func (b *prepareBundleBuilder) buildGroupStep(g *PrepareBundleGroup) (bundle.ExecutionStep, error) {
	subSteps := make([]bundle.ExecutionStep, len(g.Entries))
	for i, s := range g.Entries {
		sub, err := b.buildStep(s)
		if err != nil {
			return bundle.ExecutionStep{}, err
		}
		subSteps[i] = sub
	}

	// No-modifier single-child group is elided so the RPC output matches the internal plan shape.
	if !g.OneOf && !g.TolerateFailures && len(subSteps) == 1 {
		return subSteps[0], nil
	}

	group := bundle.NewGroupStep(g.OneOf, subSteps...)
	if g.TolerateFailures {
		group = group.WithFlags(bundle.EF_TolerateFailed)
	}
	return group, nil
}

// collectLeafTxArgsFromGroup returns all leaf TransactionArgs in depth-first order.
func collectLeafTxArgsFromGroup(g PrepareBundleGroup) []ethapi.TransactionArgs {
	var out []ethapi.TransactionArgs
	for _, step := range g.Entries {
		collectLeafTxArgs(step, &out)
	}
	return out
}

// collectLeafTxArgs appends leaf TransactionArgs from step in depth-first order.
func collectLeafTxArgs(step PrepareBundleEntry, out *[]ethapi.TransactionArgs) {
	if step.Tx != nil {
		*out = append(*out, step.Tx.Transaction)
		return
	}
	if step.Group != nil {
		for _, sub := range step.Group.Entries {
			collectLeafTxArgs(sub, out)
		}
	}
}

// fillTransactionDefaults fills missing Gas and gas-price fields without overwriting explicit values.
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

// resolveBlockRange computes the effective block range from optional earliest/latest overrides.
func resolveBlockRange(currentBlock uint64, earliest, latest *hexutil.Uint64) (bundle.BlockRange, error) {

	if latest != nil && earliest != nil && uint64(*latest) < uint64(*earliest) {
		return bundle.BlockRange{}, fmt.Errorf("invalid block range: latest block %d is earlier than earliest block %d", *latest, *earliest)
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

// asTransaction converts a Message to a Transaction, rejecting blob and set-code types.
func asTransaction(msg *core.Message) (*types.Transaction, error) {

	feecapSet := msg.GasFeeCap != nil && msg.GasFeeCap.Sign() > 0
	tipcapSet := msg.GasTipCap != nil && msg.GasTipCap.Sign() > 0
	if msg.GasPrice != nil && msg.GasPrice.Sign() != 0 && (feecapSet || tipcapSet) {
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

// suggestGasPrice returns the suggested gas price based on the current block's base fee.
func (a *PublicBundleAPI) suggestGasPrice(block *evmcore.EvmBlock) *hexutil.Big {
	price := block.Header().BaseFee
	price = gaspricelimits.GetSuggestedGasPriceForNewTransactions(price)
	return (*hexutil.Big)(price)
}
