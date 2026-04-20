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
	"fmt"
	"math/big"
	"slices"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// RPCExecutionProposal is the JSON-serializable representation of the execution proposal
// that is returned by the API. It is designed to be easily serializable to JSON
// and human-readable for integration purposes.
//
// Each leaf step contains the same fields as the eth_call transaction arguments,
// with the addition of the tolerateFailed and tolerateInvalid flags that
// indicate whether a transaction is allowed to fail or be invalid without
// causing the entire proposal to be rejected.
//
// Steps can be nested into groups with optional oneOf and tolerateFailures
// semantics. An example of the JSON representation:
//
//	{
//	  "blockRange": {
//	    "earliest": "0xbc614e",
//	    "latest": "0xbc61b2"
//	  },
//	  "steps": [
//	    {
//	      "from": "0xabc1230000000000000000000000000000000001",
//	      "to": "0xdef4560000000000000000000000000000000002",
//	      "gas": "0x5208",
//	      "value": "0xde0b6b3a7640000",
//	      "chainId": "0x1"
//	    },
//	    {
//	      "oneOf": true,
//	      "steps": [
//	        {
//	          "tolerateFailed": true,
//	          "from": "0xabc1230000000000000000000000000000000001",
//	          "to": "0x1111111111111111111111111111111111111111",
//	          "data": "0xa9059cbb",
//	          "chainId": "0x1"
//	        },
//	        {
//	          "tolerateInvalid": true,
//	          "from": "0xabc1230000000000000000000000000000000001",
//	          "to": "0x2222222222222222222222222222222222222222",
//	          "data": "0xa9059cbb",
//	          "chainId": "0x1"
//	        }
//	      ]
//	    }
//	  ]
//	}
type RPCExecutionProposal struct {
	BlockRange RPCRange `json:"blockRange"`
	RPCExecutionPlanGroup
}

type RPCExecutionStepProposal struct {
	TolerateFailed  bool `json:"tolerateFailed,omitempty"`
	TolerateInvalid bool `json:"tolerateInvalid,omitempty"`
	ethapi.TransactionArgs
}

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

	proposal := &RPCExecutionProposal{
		BlockRange: RPCRange{
			Earliest: hexutil.Uint64(plan.Range.Earliest),
			Latest:   hexutil.Uint64(plan.Range.Latest),
		},
		RPCExecutionPlanGroup: visitor.result,
	}

	return proposal, nil
}

func convertVisitorLeafIntoRPCExecutionPlanProposalLeaf(
	signer types.Signer,
	txBundle *bundle.TransactionBundle,
	flags bundle.ExecutionFlags,
	txRef bundle.TxReference,
) (*RPCExecutionStepProposal, error) {
	tx, ok := txBundle.Transactions[txRef]
	if !ok {
		return nil, fmt.Errorf("transaction reference not found in bundle transactions: %v", txRef)
	}
	txArgs, err := convertToTransactionArgs(signer, tx)
	if err != nil {
		return nil, err
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

	return &RPCExecutionStepProposal{
		TolerateFailed:  flags&bundle.EF_TolerateFailed != 0,
		TolerateInvalid: flags&bundle.EF_TolerateInvalid != 0,
		TransactionArgs: txArgs,
	}, nil
}

// convertToTransactionArgs converts a types.Transaction to ethapi.TransactionArgs, which is the format used in the execution proposal.
// If members of the transaction are not set (e.g. GasPrice for a type 2 transaction), they will be omitted from the resulting TransactionArgs.
//
// This function is meant for testing purposes, therefore not exported
func convertToTransactionArgs(signer types.Signer, tx *types.Transaction) (ethapi.TransactionArgs, error) {

	sender, err := types.Sender(signer, tx)
	if err != nil {
		return ethapi.TransactionArgs{}, fmt.Errorf("failed to derive sender for transaction: %w", err)
	}

	res := ethapi.TransactionArgs{
		ChainID: (*hexutil.Big)(tx.ChainId()),
		From:    &sender,
		To:      tx.To(),
	}

	if tx.Nonce() != 0 {
		res.Nonce = toPtr(hexutil.Uint64(tx.Nonce()))
	}

	if tx.To() == nil && tx.Data() != nil {
		res.Input = toPtr(hexutil.Bytes(tx.Data()))
	}
	if tx.To() != nil && tx.Data() != nil {
		res.Data = toPtr(hexutil.Bytes(tx.Data()))
	}

	if tx.Value() != nil && tx.Value().Cmp(big.NewInt(0)) > 0 {
		res.Value = toPtr(hexutil.Big(*tx.Value()))
	}

	if tx.Gas() != 0 {
		res.Gas = toPtr(hexutil.Uint64(tx.Gas()))
	}

	// Type 1 tx

	if tx.Type() >= types.AccessListTxType && len(tx.AccessList()) > 0 {
		res.AccessList = toPtr(tx.AccessList())
	}

	// Type 2 txs, dynamic fees

	switch tx.Type() {
	case types.LegacyTxType, types.AccessListTxType:
		if tx.GasPrice().Cmp(big.NewInt(0)) > 0 {
			res.GasPrice = toPtr(hexutil.Big(*tx.GasPrice()))
		}
	case types.DynamicFeeTxType, types.BlobTxType, types.SetCodeTxType:
		if tx.GasTipCap().Cmp(big.NewInt(0)) > 0 {
			res.MaxPriorityFeePerGas = toPtr(hexutil.Big(*tx.GasTipCap()))
		}
		if tx.GasFeeCap().Cmp(big.NewInt(0)) > 0 {
			res.MaxFeePerGas = toPtr(hexutil.Big(*tx.GasFeeCap()))
		}
	}

	// Type 3 txs, blobs

	if tx.Type() == types.BlobTxType && len(tx.BlobHashes()) > 0 {
		return ethapi.TransactionArgs{}, fmt.Errorf("blob transactions are not supported in execution proposals")
	}

	// Type 4 txs, set code

	if tx.Type() == types.SetCodeTxType && len(tx.SetCodeAuthorizations()) > 0 {
		res.AuthorizationList = slices.Clone(tx.SetCodeAuthorizations())
	}

	return res, nil
}

func toPtr[T any](v T) *T {
	return &v
}
