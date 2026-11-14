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

// Package sonic serves the sonic namespace of the RPC module: the bundles of
// transactions Sonic runs together.
package sonic

import (
	"context"
	"fmt"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/params"
)

const (
	// maxEstimableTransactions bounds the transactions of one estimation, as
	// each estimation runs the transactions before it.
	maxEstimableTransactions = 16
	// markerGas is the gas a transaction of a bundle spends on its access
	// list beyond what it runs: the bundle-only address and the hash of the
	// execution plan.
	markerGas = params.TxAccessListAddressGas + params.TxAccessListStorageKeyGas
)

type handlers struct {
	environment *exec.Environment
}

// Register adds the methods of the namespace to the dispatcher.
func Register(dispatcher *dispatch.Dispatcher, environment *exec.Environment) {
	h := &handlers{environment: environment}
	dispatch.Add(dispatcher, "sonic_estimateGasForTransactions", h.estimateGasForTransactions)
}

type estimateParams struct {
	Transactions   []exec.CallArguments
	Block          *chain.BlockReference
	StateOverrides *exec.StateOverride
	BlockOverrides *exec.BlockOverrides
}

type bundleGasLimits struct {
	GasLimits []hexutil.Uint64 `json:"gasLimits"`
}

// estimateGasForTransactions estimates the gas limit of each transaction of a
// bundle, on the state the transactions before it leave, at a block, the
// head if it names none, each limit with the gas of the bundle's markers.
func (h *handlers) estimateGasForTransactions(ctx context.Context, params estimateParams) (bundleGasLimits, error) {
	if len(params.Transactions) > maxEstimableTransactions {
		return bundleGasLimits{}, fmt.Errorf("too many transactions to estimate gas for: got %d, max is %d",
			len(params.Transactions), maxEstimableTransactions)
	}
	block := chain.ReferenceTo(chain.LatestBlockNumber)
	if params.Block != nil {
		block = *params.Block
	}
	estimates, err := h.environment.EstimateGasInOrder(ctx, h.environment.Head(), exec.Estimation{
		Block:          block,
		StateOverrides: params.StateOverrides,
		BlockOverrides: params.BlockOverrides,
	}, params.Transactions)
	if err != nil {
		return bundleGasLimits{}, err
	}
	for i := range estimates {
		estimates[i] += hexutil.Uint64(markerGas)
	}
	return bundleGasLimits{GasLimits: estimates}, nil
}
