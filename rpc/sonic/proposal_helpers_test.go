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

// Test helpers of the reference's execution_proposal.go: the proposal a
// bundle comes from.

import (
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

// createProposalRequestFromBundle creates an RPCExecutionProposal from a bundle.TransactionBundle,
// which is the internal representation of a transaction bundle used in the execution engine.
// This function is meant for testing purposes, therefore not exported.
func createProposalRequestFromBundle(signer types.Signer, transactionBundle *bundle.TransactionBundle) (*RPCExecutionProposal, error) {
	plan := transactionBundle.Plan

	visitor := makeExecutionPlanVisitor(func(flags bundle.ExecutionFlags, reference bundle.TxReference) (any, error) {
		return convertVisitorLeafIntoRPCExecutionPlanProposalLeaf(signer, transactionBundle, flags, reference)
	})
	err := plan.Root.Accept(visitor)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution proposal: %w", err)
	}

	blockRange := fromBundleRange(plan.Range)
	proposal := &RPCExecutionProposal{
		BlockRange:            &blockRange,
		RPCExecutionPlanGroup: visitor.result,
	}

	return proposal, nil
}

func convertVisitorLeafIntoRPCExecutionPlanProposalLeaf(
	signer types.Signer,
	transactionBundle *bundle.TransactionBundle,
	flags bundle.ExecutionFlags,
	reference bundle.TxReference,
) (RPCExecutionStepProposal, error) {
	empty := RPCExecutionStepProposal{}

	transaction, ok := transactionBundle.Transactions[reference]
	if !ok {
		return empty, fmt.Errorf("transaction reference not found in bundle transactions: %v", reference)
	}
	arguments, err := convertToCallArguments(signer, transaction)
	if err != nil {
		return empty, err
	}

	// remove bundle markers from access list
	if arguments.AccessList != nil {
		cleaned := make(types.AccessList, 0, len(*arguments.AccessList))
		for _, entry := range *arguments.AccessList {
			if entry.Address != bundle.BundleOnly {
				cleaned = append(cleaned, entry)
			}
		}
		if len(cleaned) == 0 {
			arguments.AccessList = nil
		} else {
			arguments.AccessList = &cleaned
		}
	}

	return RPCExecutionStepProposal{
		TolerateFailed:  flags&bundle.EF_TolerateFailed != 0,
		TolerateInvalid: flags&bundle.EF_TolerateInvalid != 0,
		CallArguments:   arguments,
	}, nil
}

// convertToCallArguments converts a types.Transaction to exec.CallArguments, which is the format used in the execution proposal.
// If members of the transaction are not set (e.g. GasPrice for a type 2 transaction), they will be omitted from the resulting CallArguments.
//
// This function is meant for testing purposes, therefore not exported
func convertToCallArguments(signer types.Signer, transaction *types.Transaction) (exec.CallArguments, error) {

	sender, err := types.Sender(signer, transaction)
	if err != nil {
		return exec.CallArguments{}, fmt.Errorf("failed to derive sender for transaction: %w", err)
	}

	result := exec.CallArguments{
		ChainID:   (*hexutil.Big)(transaction.ChainId()),
		From:      &sender,
		Recipient: transaction.To(),
	}

	if transaction.Nonce() != 0 {
		result.Nonce = new(hexutil.Uint64(transaction.Nonce()))
	}

	if transaction.To() == nil && transaction.Data() != nil {
		result.Input = new(hexutil.Bytes(transaction.Data()))
	}
	if transaction.To() != nil && transaction.Data() != nil {
		result.Data = new(hexutil.Bytes(transaction.Data()))
	}

	if transaction.Value() != nil && transaction.Value().Cmp(big.NewInt(0)) > 0 {
		result.Value = new(hexutil.Big(*transaction.Value()))
	}

	if transaction.Gas() != 0 {
		result.Gas = new(hexutil.Uint64(transaction.Gas()))
	}

	// Type 1 tx

	if transaction.Type() >= types.AccessListTxType && len(transaction.AccessList()) > 0 {
		result.AccessList = new(transaction.AccessList())
	}

	// Type 2 txs, dynamic fees

	switch transaction.Type() {
	case types.LegacyTxType, types.AccessListTxType:
		if transaction.GasPrice().Cmp(big.NewInt(0)) > 0 {
			result.GasPrice = new(hexutil.Big(*transaction.GasPrice()))
		}
	case types.DynamicFeeTxType, types.BlobTxType, types.SetCodeTxType:
		if transaction.GasTipCap().Cmp(big.NewInt(0)) > 0 {
			result.MaxPriorityFeePerGas = new(hexutil.Big(*transaction.GasTipCap()))
		}
		if transaction.GasFeeCap().Cmp(big.NewInt(0)) > 0 {
			result.MaxFeePerGas = new(hexutil.Big(*transaction.GasFeeCap()))
		}
	}

	// Type 3 txs, blobs

	if transaction.Type() == types.BlobTxType && len(transaction.BlobHashes()) > 0 {
		return exec.CallArguments{}, fmt.Errorf("blob transactions are not supported in execution proposals")
	}

	// Type 4 txs, set code

	if transaction.Type() == types.SetCodeTxType && len(transaction.SetCodeAuthorizations()) > 0 {
		result.AuthorizationList = slices.Clone(transaction.SetCodeAuthorizations())
	}

	return result, nil
}
