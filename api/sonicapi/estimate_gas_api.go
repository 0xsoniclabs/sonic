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
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

//go:generate mockgen -source=estimate_gas_api.go -destination=estimate_gas_api_mock.go -package=sonicapi

// MAX_BUNDLE_TRANSACTIONS is the maximum number of transactions
// that can be included in a bundle for gas estimation.
const MAX_BUNDLE_TRANSACTIONS = 16

type BundleGasLimits struct {
	// GasLimits contains the estimated gas limit for each transaction in the
	// bundle, in the same order as the input transactions.
	GasLimits []hexutil.Uint64 `json:"gasLimits"`
}

// EstimateGasForTransactions implements the `sonic_estimateGasForTransactions` RPC method.
// It estimates the gas required for each provided transaction,
// applying state changes from previous transactions when estimating subsequent ones.
// Transactions that become invalid or fail during execution for later estimations are ignored.
// This method can help getting gas estimates for mutually depending transactions in bundles.
func (a *PublicBundleAPI) EstimateGasForTransactions(
	ctx context.Context,
	args []ethapi.TransactionArgs,
	blockNrOrHash *rpc.BlockNumberOrHash,
	overrides *ethapi.StateOverride,
	blockOverrides *ethapi.BlockOverrides,
) (BundleGasLimits, error) {

	if len(args) > MAX_BUNDLE_TRANSACTIONS {
		return BundleGasLimits{}, fmt.Errorf("too many transactions to estimate gas for: got %d, max is %d", len(args), MAX_BUNDLE_TRANSACTIONS)
	}

	bNrOrHash := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
	if blockNrOrHash != nil {
		bNrOrHash = *blockNrOrHash
	}

	gasCap := a.b.RPCGasCap()
	eval := &estimator{
		ctx:            ctx,
		b:              a.b,
		blockNrOrHash:  bNrOrHash,
		overrides:      overrides,
		blockOverrides: blockOverrides,
		gasCap:         gasCap,
	}

	gasLimits, err := doEstimateGasForTransactions(args, eval)
	if err != nil {
		return BundleGasLimits{}, err
	}
	return BundleGasLimits{GasLimits: gasLimits}, nil
}

// GasEstimator is an interface for estimating gas for a transaction with given pre-state.
type GasEstimator interface {
	EstimateGas(args ethapi.TransactionArgs, preArgs []ethapi.TransactionArgs) (hexutil.Uint64, error)
}

// estimator is a struct that implements the GasEstimator interface using the ethapi.DoEstimateGas function.
type estimator struct {
	ctx            context.Context
	b              ethapi.Backend
	blockNrOrHash  rpc.BlockNumberOrHash
	overrides      *ethapi.StateOverride
	blockOverrides *ethapi.BlockOverrides
	gasCap         uint64
}

// EstimateGas estimates the gas for a transaction given the transaction arguments and the pre-state defined by preArgs.
func (e *estimator) EstimateGas(args ethapi.TransactionArgs, preArgs []ethapi.TransactionArgs) (hexutil.Uint64, error) {
	gas, err := ethapi.DoEstimateGas(e.ctx, e.b, args, e.blockNrOrHash, e.overrides, e.blockOverrides, e.gasCap, preArgs)
	if err != nil {
		return 0, err
	}

	return gas, nil
}

// doEstimateGasForTransactions estimates the gas for a list of transactions using the provided GasEstimator.
// It applies the state changes from each transaction to the subsequent ones when estimating gas.
func doEstimateGasForTransactions(
	args []ethapi.TransactionArgs,
	eval GasEstimator,
) ([]hexutil.Uint64, error) {
	gasLimits := make([]hexutil.Uint64, len(args))
	preArgs := make([]ethapi.TransactionArgs, 0, len(args))
	for i, arg := range args {
		gas, err := eval.EstimateGas(arg, preArgs)
		if err != nil {
			return nil, err
		}

		preArgs = append(preArgs, arg)
		preArgs[len(preArgs)-1].Gas = (*hexutil.Uint64)(&gas)

		gasLimits[i] = gas +
			hexutil.Uint64(params.TxAccessListAddressGas) + // add gas for bundle only address
			hexutil.Uint64(params.TxAccessListStorageKeyGas) // add gas for execution plan hash
	}
	return gasLimits, nil
}
