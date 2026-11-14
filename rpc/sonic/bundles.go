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
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type prepareParams struct {
	Proposal executionProposal
}

// preparedBundle is a bundle's transactions, to be signed as they are, in
// depth-first order, and its execution plan.
type preparedBundle struct {
	Transactions  []exec.CallArguments `json:"transactions"`
	ExecutionPlan executionPlan        `json:"executionPlan"`
}

// prepareBundle turns a proposal into a bundle to sign: the blocks it may run
// in, from the block after the head unless named, the gas of every
// transaction that names none, estimated with the transactions before it run,
// the suggested gas price of every transaction that names no fee cap, and
// the plan's marker in every access list.
func (h *handlers) prepareBundle(ctx context.Context, params prepareParams) (*preparedBundle, error) {
	proposal := params.Proposal.group
	head := h.environment.Head()
	blocks, err := sanitizeBlockRange(head.NumberU64(), params.Proposal.BlockRange)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bundle: %w", err)
	}
	transactions, err := flatten(proposal)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bundle: %w", err)
	}
	var limits []hexutil.Uint64
	if slices.ContainsFunc(transactions, needsGas) {
		if err := checkSequentialEstimation(proposal); err != nil {
			return nil, fmt.Errorf("failed to prepare bundle: %w", err)
		}
		estimated, err := h.estimateGasForTransactions(ctx, estimateParams{Transactions: transactions})
		if err != nil {
			return nil, fmt.Errorf("failed to prepare bundle: gas estimation failed: %w", err)
		}
		limits = estimated.GasLimits
	}
	price := (*hexutil.Big)(h.environment.SuggestedGasPrice(head))
	position := 0
	proposal, err = transform(proposal, func(step proposedStep) (proposedStep, error) {
		if needsGas(step.CallArguments) {
			step.Gas = &limits[position]
		}
		if step.GasPrice == nil && step.MaxFeePerGas == nil {
			if step.MaxPriorityFeePerGas == nil {
				step.GasPrice = price
			} else {
				step.MaxFeePerGas = price
			}
		}
		transactions[position] = step.CallArguments
		position++
		return step, nil
	}, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to set proposed transactions defaults: %w", err)
	}
	plan, err := proposalToPlan(types.LatestSignerForChainID(h.environment.ChainID()), blocks, proposal)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bundle: %w", err)
	}
	addPlanMarker(transactions, plan.Hash())
	formed, err := newExecutionPlan(plan)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bundle: %w", err)
	}
	return &preparedBundle{Transactions: transactions, ExecutionPlan: formed}, nil
}

type submitParams struct {
	Bundle submittedBundle
}

// submittedBundle is a prepared bundle with its transactions signed.
type submittedBundle struct {
	SignedTransactions []hexutil.Bytes `json:"signedTransactions"`
	ExecutionPlan      executionPlan   `json:"executionPlan,omitempty"`
}

// submitBundle wraps the signed transactions of a bundle into a transaction
// to the bundle processor, signed with a key of its own, sends it, and
// answers the hash of the bundle's plan.
func (h *handlers) submitBundle(ctx context.Context, params submitParams) (common.Hash, error) {
	submitted := params.Bundle
	if len(submitted.SignedTransactions) == 0 {
		return common.Hash{}, errors.New("signedTransactions must not be empty")
	}
	head := h.environment.Head()
	if _, err := sanitizeBlockRange(head.NumberU64(), &submitted.ExecutionPlan.BlockRange); err != nil {
		return common.Hash{}, fmt.Errorf("invalid block range: %w", err)
	}
	plan, err := submitted.ExecutionPlan.toBundle()
	if err != nil {
		return common.Hash{}, fmt.Errorf("invalid execution plan: %w", err)
	}
	references := plan.Root.GetTransactionReferencesInReferencedOrder()
	if len(references) != len(submitted.SignedTransactions) {
		return common.Hash{}, fmt.Errorf("executionPlan steps count (%d) must match signedTransactions count (%d)",
			len(references), len(submitted.SignedTransactions))
	}
	transactions := make(map[bundle.TxReference]*types.Transaction, len(references))
	for i, encoded := range submitted.SignedTransactions {
		transaction := new(types.Transaction)
		if err := transaction.UnmarshalBinary(encoded); err != nil {
			return common.Hash{}, fmt.Errorf("failed to decode bundled transaction %d: %w", i, err)
		}
		transactions[references[i]] = transaction
	}
	envelope, err := envelopeOf(bundle.TransactionBundle{Transactions: transactions, Plan: plan}, h.environment.ChainID())
	if err != nil {
		return common.Hash{}, err
	}
	if err := h.environment.Send(ctx, envelope); err != nil {
		return common.Hash{}, fmt.Errorf("failed to submit bundle transaction: %w", err)
	}
	return plan.Hash(), nil
}

// envelopeOf is the transaction that carries a bundle to the bundle
// processor, with the gas the bundle needs, signed with a key used once.
func envelopeOf(transactions bundle.TransactionBundle, chainID *big.Int) (*types.Transaction, error) {
	data, err := transactions.Encode()
	if err != nil {
		return nil, fmt.Errorf("failed to encode bundle: %w", err)
	}
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate single use signing key: %w", err)
	}
	gas, err := bundle.CalculateEnvelopeGas(transactions, data, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate envelope gas: %w", err)
	}
	envelope, err := types.SignNewTx(key, types.LatestSignerForChainID(chainID),
		&types.DynamicFeeTx{To: &bundle.BundleProcessor, Nonce: 0, Data: data, Gas: gas})
	if err != nil {
		return nil, fmt.Errorf("failed to sign bundle transaction: %w", err)
	}
	return envelope, nil
}

type bundleInfoParams struct {
	ExecutionPlanHash common.Hash
}

// bundleInfo tells where the transactions of an executed bundle are.
type bundleInfo struct {
	Block    hexutil.Uint64 `json:"block,omitempty"`
	Position hexutil.Uint   `json:"position,omitempty"`
	Count    hexutil.Uint   `json:"count,omitempty"`
}

// getBundleInfo answers where the transactions of the bundle of a plan are,
// once the block that holds them is served; null before, and for a bundle
// the node no longer knows.
func (h *handlers) getBundleInfo(_ context.Context, params bundleInfoParams) (*bundleInfo, error) {
	info, err := h.node.BundleInfo(params.ExecutionPlanHash)
	if exec.IsMissing(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	header, err := exec.NoneIfMissing(h.environment.HeaderAt(h.environment.Head(), info.BlockNumber))
	if header == nil || err != nil {
		return nil, err
	}
	return &bundleInfo{Block: hexutil.Uint64(info.BlockNumber), Position: hexutil.Uint(info.Offset), Count: hexutil.Uint(info.Count)}, nil
}

// sanitizeBlockRange checks the blocks a bundle may run in, which are the
// most from the block after the head unless named: they may not end before
// the head nor be more than the limit.
func sanitizeBlockRange(head uint64, blocks *blockRange) (blockRange, error) {
	next := max(head+1, head)
	if blocks == nil {
		widest := bundle.MakeMaxRangeStartingAt(next)
		return blockRange{First: hexutil.Uint64(widest.First), Length: hexutil.Uint64(widest.Length)}, nil
	}
	first := uint64(blocks.First)
	if first == 0 {
		first = next
	}
	length := uint64(blocks.Length)
	if length == 0 {
		length = bundle.MaxBlockRangeLength
	} else if length > bundle.MaxBlockRangeLength {
		return blockRange{}, fmt.Errorf("invalid block range: length %d is too large; must be at most %d blocks", length, bundle.MaxBlockRangeLength)
	}
	if (bundle.BlockRange{First: first, Length: length}).IsAfterRange(head) {
		return blockRange{}, fmt.Errorf("invalid block range: the specified range (first: %d, length: %d) is not in the future relative to current block %d", first, length, head)
	}
	return blockRange{First: hexutil.Uint64(first), Length: hexutil.Uint64(length)}, nil
}
