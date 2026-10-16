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

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

// MaxNumEstimableTransactions is the maximum number of transactions
// that can be included in a bundle for gas estimation.
// The algorithm to estimate bundle gas is O(n^2),
// therefore an upper bound is introduced.
const MaxNumEstimableTransactions = 16

// BundleGasLimits represents the estimated gas limits for a bundle of transactions.
type BundleGasLimits struct {
	// GasLimits contains the estimated gas limit for each transaction in the
	// bundle, in the same order as the input transactions.
	GasLimits []hexutil.Uint64 `json:"gasLimits"`
}

// EstimateGasForTransactions implements the `sonic_estimateGasForTransactions` RPC method.
// It estimates the gas required for each provided transaction,
// applying state changes from previous transactions when estimating subsequent ones.
// This method can help getting gas estimates for mutually depending transactions in bundles.
func (a *API) EstimateGasForTransactions(
	ctx context.Context,
	args []exec.CallArgs,
	blockNrOrHash *dispatch.BlockRef,
	overrides *exec.StateOverride,
	blockOverrides *exec.BlockOverrides,
) (BundleGasLimits, error) {

	if len(args) > MaxNumEstimableTransactions {
		return BundleGasLimits{}, fmt.Errorf("too many transactions to estimate gas for: got %d, max is %d", len(args), MaxNumEstimableTransactions)
	}

	eval := &estimator{
		ctx:            ctx,
		env:            a.env,
		head:           a.chain.Head(),
		blockNrOrHash:  dispatch.OrLatest(blockNrOrHash),
		overrides:      overrides,
		blockOverrides: blockOverrides,
		gasCap:         a.env.Cfg.GasCap,
	}

	gasLimits, err := doEstimateGasForTransactions(args, eval)
	if err != nil {
		return BundleGasLimits{}, err
	}
	return BundleGasLimits{GasLimits: gasLimits}, nil
}

// estimator is a helper struct that holds the context and parameters
// needed to estimate gas for a list of transactions.
type estimator struct {
	ctx            context.Context
	env            *exec.Env
	head           dispatch.Head
	blockNrOrHash  dispatch.BlockRef
	overrides      *exec.StateOverride
	blockOverrides *exec.BlockOverrides
	gasCap         uint64
}

// EstimateGas estimates the gas for a transaction given the transaction arguments and the pre-state defined by preArgs.
func (e *estimator) EstimateGas(args exec.CallArgs, preArgs []exec.CallArgs) (hexutil.Uint64, error) {
	gas, err := exec.DoEstimateGas(e.ctx, e.env, e.head, args, e.blockNrOrHash, e.overrides, e.blockOverrides, e.gasCap, preArgs)
	if err != nil {
		return 0, err
	}

	return gas, nil
}

// doEstimateGasForTransactions estimates the gas for a list of transactions using the provided gas estimator.
// It applies the state changes from each transaction to the subsequent ones when estimating gas.
func doEstimateGasForTransactions(
	args []exec.CallArgs,
	eval *estimator,
) ([]hexutil.Uint64, error) {
	gasLimits := make([]hexutil.Uint64, len(args))
	preArgs := make([]exec.CallArgs, 0, len(args))
	for i, arg := range args {
		gas, err := eval.EstimateGas(arg, preArgs)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate gas for transaction %d: %w", i, err)
		}

		preArgs = append(preArgs, arg)
		preArgs[len(preArgs)-1].Gas = (*hexutil.Uint64)(&gas)

		gasLimits[i] = gas +
			hexutil.Uint64(params.TxAccessListAddressGas) + // add gas for bundle only address
			hexutil.Uint64(params.TxAccessListStorageKeyGas) // add gas for execution plan hash
	}
	return gasLimits, nil
}
