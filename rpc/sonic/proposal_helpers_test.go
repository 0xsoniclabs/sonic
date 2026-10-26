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
func createProposalRequestFromBundle(signer types.Signer, txBundle *bundle.TransactionBundle) (*RPCExecutionProposal, error) {
	plan := txBundle.Plan

	visitor := makeExecutionPlanVisitor(func(flags bundle.ExecutionFlags, txRef bundle.TxReference) (any, error) {
		return convertVisitorLeafIntoRPCExecutionPlanProposalLeaf(signer, txBundle, flags, txRef)
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
	txBundle *bundle.TransactionBundle,
	flags bundle.ExecutionFlags,
	txRef bundle.TxReference,
) (RPCExecutionStepProposal, error) {
	empty := RPCExecutionStepProposal{}

	tx, ok := txBundle.Transactions[txRef]
	if !ok {
		return empty, fmt.Errorf("transaction reference not found in bundle transactions: %v", txRef)
	}
	txArgs, err := convertToTransactionArgs(signer, tx)
	if err != nil {
		return empty, err
	}

	// remove bundle markers from access list
	if txArgs.AccessList != nil {
		cleaned := make(types.AccessList, 0, len(*txArgs.AccessList))
		for _, entry := range *txArgs.AccessList {
			if entry.Address != bundle.BundleOnly {
				cleaned = append(cleaned, entry)
			}
		}
		if len(cleaned) == 0 {
			txArgs.AccessList = nil
		} else {
			txArgs.AccessList = &cleaned
		}
	}

	return RPCExecutionStepProposal{
		TolerateFailed:  flags&bundle.EF_TolerateFailed != 0,
		TolerateInvalid: flags&bundle.EF_TolerateInvalid != 0,
		CallArgs:        txArgs,
	}, nil
}

// convertToTransactionArgs converts a types.Transaction to exec.CallArgs, which is the format used in the execution proposal.
// If members of the transaction are not set (e.g. GasPrice for a type 2 transaction), they will be omitted from the resulting TransactionArgs.
//
// This function is meant for testing purposes, therefore not exported
func convertToTransactionArgs(signer types.Signer, tx *types.Transaction) (exec.CallArgs, error) {

	sender, err := types.Sender(signer, tx)
	if err != nil {
		return exec.CallArgs{}, fmt.Errorf("failed to derive sender for transaction: %w", err)
	}

	res := exec.CallArgs{
		ChainID: (*hexutil.Big)(tx.ChainId()),
		From:    &sender,
		To:      tx.To(),
	}

	if tx.Nonce() != 0 {
		res.Nonce = new(hexutil.Uint64(tx.Nonce()))
	}

	if tx.To() == nil && tx.Data() != nil {
		res.Input = new(hexutil.Bytes(tx.Data()))
	}
	if tx.To() != nil && tx.Data() != nil {
		res.Data = new(hexutil.Bytes(tx.Data()))
	}

	if tx.Value() != nil && tx.Value().Cmp(big.NewInt(0)) > 0 {
		res.Value = new(hexutil.Big(*tx.Value()))
	}

	if tx.Gas() != 0 {
		res.Gas = new(hexutil.Uint64(tx.Gas()))
	}

	// Type 1 tx

	if tx.Type() >= types.AccessListTxType && len(tx.AccessList()) > 0 {
		res.AccessList = new(tx.AccessList())
	}

	// Type 2 txs, dynamic fees

	switch tx.Type() {
	case types.LegacyTxType, types.AccessListTxType:
		if tx.GasPrice().Cmp(big.NewInt(0)) > 0 {
			res.GasPrice = new(hexutil.Big(*tx.GasPrice()))
		}
	case types.DynamicFeeTxType, types.BlobTxType, types.SetCodeTxType:
		if tx.GasTipCap().Cmp(big.NewInt(0)) > 0 {
			res.MaxPriorityFeePerGas = new(hexutil.Big(*tx.GasTipCap()))
		}
		if tx.GasFeeCap().Cmp(big.NewInt(0)) > 0 {
			res.MaxFeePerGas = new(hexutil.Big(*tx.GasFeeCap()))
		}
	}

	// Type 3 txs, blobs

	if tx.Type() == types.BlobTxType && len(tx.BlobHashes()) > 0 {
		return exec.CallArgs{}, fmt.Errorf("blob transactions are not supported in execution proposals")
	}

	// Type 4 txs, set code

	if tx.Type() == types.SetCodeTxType && len(tx.SetCodeAuthorizations()) > 0 {
		res.AuthorizationList = slices.Clone(tx.SetCodeAuthorizations())
	}

	return res, nil
}
