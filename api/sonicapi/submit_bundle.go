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
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// SubmitBundleArgs represents the arguments for the `sonic_submitBundle` RPC method.
type SubmitBundleArgs struct {
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

// SubmitBundle implements the `sonic_submitBundle` RPC method, which submits a prepared bundle for execution.
// Returns the hash of the execution plan and an error if any.
func (a *PublicBundleAPI) SubmitBundle(
	ctx context.Context,
	args SubmitBundleArgs,
) (common.Hash, error) {

	if len(args.SignedTransactions) == 0 {
		return common.Hash{}, fmt.Errorf("signedTransactions must not be empty")
	}

	plan, err := toBundleExecutionPlan(args.ExecutionPlan)
	if err != nil {
		return common.Hash{}, fmt.Errorf("invalid executionPlan: %w", err)
	}

	if plan.Range.Latest < plan.Range.Earliest {
		return common.Hash{}, fmt.Errorf("latest block number cannot be smaller than earliest block number")
	}

	refs := plan.Root.GetTransactionReferencesInReferencedOrder()
	if len(refs) != len(args.SignedTransactions) {
		return common.Hash{}, fmt.Errorf("executionPlan steps count (%d) must match signedTransactions count (%d)",
			len(refs), len(args.SignedTransactions))
	}

	// Decode bundled transactions and build TxReference map.
	transactions := make(map[bundle.TxReference]*types.Transaction, len(args.SignedTransactions))
	var totalGas uint64
	for i, encodedTx := range args.SignedTransactions {
		tx := new(types.Transaction)
		if err := tx.UnmarshalBinary(encodedTx); err != nil {
			return common.Hash{}, fmt.Errorf("failed to decode bundled transaction %d: %w", i, err)
		}
		transactions[refs[i]] = tx
		totalGas += tx.Gas()
	}

	txBundle := bundle.TransactionBundle{
		Transactions: transactions,
		Plan:         plan,
	}

	// Encode the bundle and compute if gas limits are sufficient to cover
	// both the payload and the data-related gas costs.
	data, err := txBundle.Encode()
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to encode bundle: %w", err)
	}
	minGas, err := core.IntrinsicGas(data, nil, nil, false, true, true, true)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to finalize bundle: could not calculate intrinsic gas: %w", err)
	}
	floorDataGas, err := core.FloorDataGas(data)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to finalize bundle: could not calculate floor data gas: %w", err)
	}
	totalGas = max(totalGas, minGas, floorDataGas)

	// Make a one use key to sign the bundle
	key, err := crypto.GenerateKey()
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to generate single use signing key: %w", err)
	}

	// Sign the bundle transaction with the one-use key and send it to the network
	signer := types.LatestSignerForChainID(a.b.ChainID())
	tx, err := types.SignNewTx(key, signer,
		&types.DynamicFeeTx{
			To:    &bundle.BundleProcessor,
			Nonce: 0,
			Data:  data,
			Gas:   totalGas,
		})
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to sign bundle transaction: %w", err)
	}

	// Validate generated transaction
	_, validatedPlan, err := bundle.ValidateEnvelope(signer, tx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to validate bundle transaction: %w", err)
	}

	// Submit the transaction to the network
	_, err = ethapi.SubmitTransaction(ctx, a.b, tx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to submit bundle transaction: %w", err)
	}

	return validatedPlan.Hash(), nil
}
