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
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

	var currentBlock *evmcore.EvmBlock
	if block := a.b.CurrentBlock(); block != nil {
		currentBlock = block
	} else {
		return common.Hash{}, fmt.Errorf("failed to prepare bundle: unable to retrieve current block number")
	}

	_, err := sanitizeBlockRange(currentBlock.NumberU64(), &args.ExecutionPlan.BlockRange)
	if err != nil {
		return common.Hash{}, fmt.Errorf("invalid block range: %w", err)
	}

	execPlan, err := ToBundleExecutionPlan(args.ExecutionPlan)
	if err != nil {
		return common.Hash{}, fmt.Errorf("invalid execution plan: %w", err)
	}

	refs := execPlan.Root.GetTransactionReferencesInReferencedOrder()
	if len(refs) != len(args.SignedTransactions) {
		return common.Hash{}, fmt.Errorf("executionPlan steps count (%d) must match signedTransactions count (%d)",
			len(refs), len(args.SignedTransactions))
	}

	// Decode bundled transactions and build TxReference map.
	transactions := make(map[bundle.TxReference]*types.Transaction, len(args.SignedTransactions))
	for i, encodedTx := range args.SignedTransactions {
		tx := new(types.Transaction)
		if err := tx.UnmarshalBinary(encodedTx); err != nil {
			return common.Hash{}, fmt.Errorf("failed to decode bundled transaction %d: %w", i, err)
		}
		transactions[refs[i]] = tx
	}

	txBundle := bundle.TransactionBundle{
		Transactions: transactions,
		Plan:         execPlan,
	}

	// Encode the bundle and compute if gas limits are sufficient to cover
	// both the payload and the data-related gas costs.
	data, err := txBundle.Encode()
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to encode bundle: %w", err)
	}

	// Make a one use key to sign the bundle
	key, err := crypto.GenerateKey()
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to generate single use signing key: %w", err)
	}

	// Calculate the gas limit for the bundle transaction
	gas, err := bundle.CalculateEnvelopeGas(txBundle, data, nil, nil)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to calculate envelope gas: %w", err)
	}

	// Sign the bundle transaction with the one-use key and send it to the network
	signer := types.LatestSignerForChainID(a.b.ChainID())
	tx, err := types.SignNewTx(key, signer,
		&types.DynamicFeeTx{
			To:    &bundle.BundleProcessor,
			Nonce: 0,
			Data:  data,
			Gas:   gas,
		})
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to sign bundle transaction: %w", err)
	}

	// Submit the transaction to the network
	_, err = ethapi.SubmitTransaction(ctx, a.b, tx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to submit bundle transaction: %w", err)
	}

	return execPlan.Hash(), nil
}
