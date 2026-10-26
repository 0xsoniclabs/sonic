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

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

// maxEstimableTransactions bounds the transactions of one estimate: each is
// estimated after all before it, so the work grows with their square.
const maxEstimableTransactions = 16

// bundleGasLimits is the answer of sonic_estimateGasForTransactions.
type bundleGasLimits struct {
	GasLimits []hexutil.Uint64 `json:"gasLimits"`
}

// estimateGasForTransactions implements sonic_estimateGasForTransactions.
func (handler *handlers) estimateGasForTransactions(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	gasLimits, err := handler.estimateTransactions(head, dispatch.Argument[[]exec.CallArguments](arguments, 0), exec.Estimation{
		Block:          dispatch.ArgumentOr(arguments, 1, chain.Latest()),
		StateOverrides: dispatch.ArgumentOr[exec.StateOverride](arguments, 2, nil),
		BlockOverrides: dispatch.Argument[*exec.BlockOverrides](arguments, 3),
	})
	if err != nil {
		return nil, err
	}
	return bundleGasLimits{GasLimits: gasLimits}, nil
}

// estimateTransactions estimates each transaction after the ones before it
// with their estimated gas, on the block and with the overrides of the
// estimation, and adds the gas of the bundle-only address and of the
// execution plan hash.
func (handler *handlers) estimateTransactions(head exec.Head, transactions []exec.CallArguments, estimation exec.Estimation) ([]hexutil.Uint64, error) {
	if len(transactions) > maxEstimableTransactions {
		return nil, fmt.Errorf("too many transactions to estimate gas for: got %d, max is %d", len(transactions), maxEstimableTransactions)
	}
	estimation.Previous = make([]exec.CallArguments, 0, len(transactions))
	gasLimits := make([]hexutil.Uint64, len(transactions))
	for index, transaction := range transactions {
		estimation.Arguments = transaction
		gas, err := handler.environment.EstimateGas(head, estimation)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate gas for transaction %d: %w", index, err)
		}
		transaction.Gas = &gas
		estimation.Previous = append(estimation.Previous, transaction)
		gasLimits[index] = gas + hexutil.Uint64(params.TxAccessListAddressGas) + hexutil.Uint64(params.TxAccessListStorageKeyGas)
	}
	return gasLimits, nil
}
