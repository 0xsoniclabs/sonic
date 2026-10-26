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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// SubmitBundleArguments represents the arguments for the `sonic_submitBundle` RPC method.
type SubmitBundleArguments struct {
	// SignedTransactions is the list of transactions that have been signed using
	// the transaction arguments returned by the `sonic_prepareBundle` method.
	// These transactions must be included in the bundle exactly as they were prepared;
	// any modification will invalidate the execution plan and result in an ill-formed bundle.
	SignedTransactions []hexutil.Bytes `json:"signedTransactions"`
	// ExecutionPlan contains the execution plan that each bundled transaction references.
	// This value must be provided as returned by the `sonic_prepareBundle` method;
	// any modification will invalidate the execution plan and result in an ill-formed bundle.
	ExecutionPlan RPCExecutionPlanComposable `json:"executionPlan,omitempty"`
}

// submitBundle implements the `sonic_submitBundle` RPC method, which submits a prepared bundle for execution.
// Returns the hash of the execution plan and an error if any.
// The current block is the served head (section 6).
func (handler *handlers) submitBundle(ctx context.Context, parameters dispatch.Arguments) (any, error) {
	arguments := dispatch.Argument[SubmitBundleArguments](parameters, 0)

	if len(arguments.SignedTransactions) == 0 {
		return nil, fmt.Errorf("signedTransactions must not be empty")
	}

	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}

	if _, err := sanitizeBlockRange(head.Number, &arguments.ExecutionPlan.BlockRange); err != nil {
		return nil, fmt.Errorf("invalid block range: %w", err)
	}

	bundlePlan, err := ToBundleExecutionPlan(arguments.ExecutionPlan)
	if err != nil {
		return nil, fmt.Errorf("invalid execution plan: %w", err)
	}

	references := bundlePlan.Root.GetTransactionReferencesInReferencedOrder()
	if len(references) != len(arguments.SignedTransactions) {
		return nil, fmt.Errorf("executionPlan steps count (%d) must match signedTransactions count (%d)",
			len(references), len(arguments.SignedTransactions))
	}

	// Decode bundled transactions and build TxReference map.
	transactions := make(map[bundle.TxReference]*types.Transaction, len(arguments.SignedTransactions))
	for index, encoded := range arguments.SignedTransactions {
		transaction := new(types.Transaction)
		if err := transaction.UnmarshalBinary(encoded); err != nil {
			return nil, fmt.Errorf("failed to decode bundled transaction %d: %w", index, err)
		}
		transactions[references[index]] = transaction
	}

	transactionBundle := bundle.TransactionBundle{
		Transactions: transactions,
		Plan:         bundlePlan,
	}

	// Encode the bundle and compute if gas limits are sufficient to cover
	// both the payload and the data-related gas costs.
	data, err := transactionBundle.Encode()
	if err != nil {
		return nil, fmt.Errorf("failed to encode bundle: %w", err)
	}

	// Make a one use key to sign the bundle
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate single use signing key: %w", err)
	}

	// Calculate the gas limit for the bundle transaction
	gas, err := bundle.CalculateEnvelopeGas(transactionBundle, data, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate envelope gas: %w", err)
	}

	// Sign the bundle transaction with the one-use key and send it to the network
	signer := handler.environment.Signer()
	transaction, err := types.SignNewTx(key, signer,
		&types.DynamicFeeTx{
			To:    &bundle.BundleProcessor,
			Nonce: 0,
			Data:  data,
			Gas:   gas,
		})
	if err != nil {
		return nil, fmt.Errorf("failed to sign bundle transaction: %w", err)
	}

	// Submit the transaction to the network
	if _, err := handler.environment.Send(ctx, transaction); err != nil {
		return nil, fmt.Errorf("failed to submit bundle transaction: %w", err)
	}

	return bundlePlan.Hash(), nil
}
