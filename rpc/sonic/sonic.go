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
// Package sonic serves the sonic namespace: the gas of a sequence of
// transactions, and the preparation, submission and tracking of bundles.
package sonic

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

// MaxNumEstimableTransactions bounds the transactions of one
// sonic_estimateGasForTransactions.
const MaxNumEstimableTransactions = 16

type api struct {
	env *exec.Env
}

// Register adds the sonic methods.
func Register(d *dispatch.Dispatcher, env *exec.Env) {
	a := &api{env: env}
	d.Add("sonic_estimateGasForTransactions", dispatch.Func(a.estimateGasForTransactions))
	d.Add("sonic_prepareBundle", dispatch.Func(a.prepareBundle))
	d.Add("sonic_submitBundle", dispatch.Func(a.submitBundle))
	d.Add("sonic_getBundleInfo", dispatch.Func(a.getBundleInfo))
}

// BundleGasLimits holds the gas limit of each transaction of a bundle, in the
// order of the transactions.
type BundleGasLimits struct {
	GasLimits []hexutil.Uint64 `json:"gasLimits"`
}

type estimateArgs struct {
	Args           []exec.CallArgs      `rpc:"args"`
	Block          *chain.BlockRef      `rpc:"block"`
	StateOverrides *exec.StateOverride  `rpc:"stateOverrides"`
	BlockOverrides *exec.BlockOverrides `rpc:"blockOverrides"`
}

func (a *api) estimateGasForTransactions(ctx context.Context, p estimateArgs) (any, error) {
	return a.estimate(ctx, a.env.Latest(ctx), p.Args, chain.OrLatest(p.Block), p.StateOverrides, p.BlockOverrides)
}

// estimate estimates each transaction on the state the ones before it leave,
// and adds the gas of the bundle markers in its access list.
func (a *api) estimate(ctx context.Context, head *exec.Head, args []exec.CallArgs, ref chain.BlockRef, overrides *exec.StateOverride, blockOverrides *exec.BlockOverrides) (BundleGasLimits, error) {
	if len(args) > MaxNumEstimableTransactions {
		return BundleGasLimits{}, fmt.Errorf("too many transactions to estimate gas for: got %d, max is %d", len(args), MaxNumEstimableTransactions)
	}
	gasCap := a.env.Config().GasCap
	gasLimits := make([]hexutil.Uint64, len(args))
	preArgs := make([]exec.CallArgs, 0, len(args))
	for i, arg := range args {
		gas, err := a.env.EstimateGas(ctx, head, arg, ref, overrides, blockOverrides, gasCap, preArgs)
		if err != nil {
			return BundleGasLimits{}, fmt.Errorf("failed to estimate gas for transaction %d: %w", i, err)
		}
		preArgs = append(preArgs, arg)
		preArgs[len(preArgs)-1].Gas = (*hexutil.Uint64)(&gas)
		gasLimits[i] = gas +
			hexutil.Uint64(params.TxAccessListAddressGas) + // the bundle-only address
			hexutil.Uint64(params.TxAccessListStorageKeyGas) // the execution plan hash
	}
	return BundleGasLimits{GasLimits: gasLimits}, nil
}

// sanitizeBlockRange fills in the range of a bundle relative to the current
// block: by default, the longest range from the next block on.
func sanitizeBlockRange(currentBlock uint64, blockRange *RPCRange) (RPCRange, error) {
	nextBlock := max(currentBlock+1, currentBlock)
	if blockRange == nil {
		maxRange := bundle.MakeMaxRangeStartingAt(nextBlock)
		return RPCRange{First: hexutil.Uint64(maxRange.First), Length: hexutil.Uint64(maxRange.Length)}, nil
	}
	first := uint64(blockRange.First)
	if first == 0 {
		first = nextBlock
	}
	limit := uint64(blockRange.Length)
	if limit == 0 {
		limit = bundle.MaxBlockRangeLength
	} else if limit > bundle.MaxBlockRangeLength {
		return RPCRange{}, fmt.Errorf("invalid block range: length %d is too large; must be at most %d blocks", limit, bundle.MaxBlockRangeLength)
	}
	r := bundle.BlockRange{First: first, Length: limit}
	if r.IsAfterRange(currentBlock) {
		return RPCRange{}, fmt.Errorf("invalid block range: the specified range (first: %d, length: %d) is not in the future relative to current block %d", first, limit, currentBlock)
	}
	return RPCRange{First: hexutil.Uint64(first), Length: hexutil.Uint64(limit)}, nil
}

// SubmitBundleArgs are the signed transactions of a prepared bundle and its
// execution plan, exactly as sonic_prepareBundle returned them.
type SubmitBundleArgs struct {
	SignedTransactions []hexutil.Bytes            `json:"signedTransactions"`
	ExecutionPlan      RPCExecutionPlanComposable `json:"executionPlan,omitempty"`
}

type submitArgs struct {
	Args SubmitBundleArgs `rpc:"args"`
}

// submitBundle wraps the signed transactions of a bundle into one envelope
// transaction, signed with a single-use key, and sends it.
func (a *api) submitBundle(ctx context.Context, p submitArgs) (any, error) {
	args := p.Args
	if len(args.SignedTransactions) == 0 {
		return nil, fmt.Errorf("signedTransactions must not be empty")
	}
	head := a.env.Latest(ctx)
	if _, err := sanitizeBlockRange(head.Number(), &args.ExecutionPlan.BlockRange); err != nil {
		return nil, fmt.Errorf("invalid block range: %w", err)
	}
	plan, err := ToBundleExecutionPlan(args.ExecutionPlan)
	if err != nil {
		return nil, fmt.Errorf("invalid execution plan: %w", err)
	}
	refs := plan.Root.GetTransactionReferencesInReferencedOrder()
	if len(refs) != len(args.SignedTransactions) {
		return nil, fmt.Errorf("executionPlan steps count (%d) must match signedTransactions count (%d)", len(refs), len(args.SignedTransactions))
	}
	transactions := make(map[bundle.TxReference]*types.Transaction, len(args.SignedTransactions))
	for i, encoded := range args.SignedTransactions {
		tx := new(types.Transaction)
		if err := tx.UnmarshalBinary(encoded); err != nil {
			return nil, fmt.Errorf("failed to decode bundled transaction %d: %w", i, err)
		}
		transactions[refs[i]] = tx
	}
	txBundle := bundle.TransactionBundle{Transactions: transactions, Plan: plan}
	data, err := txBundle.Encode()
	if err != nil {
		return nil, fmt.Errorf("failed to encode bundle: %w", err)
	}
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate single use signing key: %w", err)
	}
	gas, err := bundle.CalculateEnvelopeGas(txBundle, data, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate envelope gas: %w", err)
	}
	envelope, err := types.SignNewTx(key, a.env.Signer(), &types.DynamicFeeTx{
		To:    &bundle.BundleProcessor,
		Nonce: 0,
		Data:  data,
		Gas:   gas,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to sign bundle transaction: %w", err)
	}
	if _, err := a.env.Submit(ctx, envelope); err != nil {
		return nil, fmt.Errorf("failed to submit bundle transaction: %w", err)
	}
	return plan.Hash(), nil
}

// RPCBundleInfo locates the transactions a bundle left in a block.
type RPCBundleInfo struct {
	Block    hexutil.Uint64 `json:"block,omitempty"`
	Position hexutil.Uint   `json:"position,omitempty"`
	Count    hexutil.Uint   `json:"count,omitempty"`
}

type planHashArg struct {
	ExecutionPlanHash common.Hash `rpc:"executionPlanHash"`
}

// getBundleInfo answers where an executed bundle went; null for an unknown
// one, and for one whose block is not served yet, so that a client retries
// rather than act on a block it cannot read.
func (a *api) getBundleInfo(ctx context.Context, p planHashArg) (any, error) {
	info, err := a.env.BundleInfo(a.env.Latest(ctx), p.ExecutionPlanHash)
	if err != nil {
		return exec.NullIfNotFound(info, err)
	}
	return &RPCBundleInfo{
		Block:    hexutil.Uint64(info.BlockNumber),
		Position: hexutil.Uint(info.Offset),
		Count:    hexutil.Uint(info.Count),
	}, nil
}
