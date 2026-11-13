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

// Package format holds the JSON forms of blocks, transactions and receipts
// that several namespaces of the RPC module answer with.
package format

import (
	"math/big"

	"github.com/0xsoniclabs/sonic/utils"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// Transaction is the JSON form of a transaction.
type Transaction struct {
	BlockHash           *common.Hash                 `json:"blockHash"`
	BlockNumber         *hexutil.Big                 `json:"blockNumber"`
	From                common.Address               `json:"from"`
	Gas                 hexutil.Uint64               `json:"gas"`
	GasPrice            *hexutil.Big                 `json:"gasPrice"`
	GasFeeCap           *hexutil.Big                 `json:"maxFeePerGas,omitempty"`
	GasTipCap           *hexutil.Big                 `json:"maxPriorityFeePerGas,omitempty"`
	Hash                common.Hash                  `json:"hash"`
	Input               hexutil.Bytes                `json:"input"`
	Nonce               hexutil.Uint64               `json:"nonce"`
	To                  *common.Address              `json:"to"`
	TransactionIndex    *hexutil.Uint64              `json:"transactionIndex"`
	Value               *hexutil.Big                 `json:"value"`
	Type                hexutil.Uint64               `json:"type"`
	Accesses            *types.AccessList            `json:"accessList,omitempty"`
	ChainID             *hexutil.Big                 `json:"chainId,omitempty"`
	V                   *hexutil.Big                 `json:"v"`
	R                   *hexutil.Big                 `json:"r"`
	S                   *hexutil.Big                 `json:"s"`
	MaxFeePerBlobGas    *hexutil.Big                 `json:"maxFeePerBlobGas"`
	BlobVersionedHashes []common.Hash                `json:"blobVersionedHashes"`
	AuthorizationList   []types.SetCodeAuthorization `json:"authorizationList,omitempty"`
}

// Position is where a transaction is in the chain.
type Position struct {
	BlockHash   common.Hash
	BlockNumber uint64
	Index       uint64
	// BaseFee is the base fee of the block, which prices a dynamic fee
	// transaction; nil before London.
	BaseFee *big.Int
}

// NewTransaction forms a transaction of a block.
func NewTransaction(transaction *types.Transaction, position Position, chainID *big.Int) *Transaction {
	result := newTransaction(transaction, chainID)
	result.BlockHash = &position.BlockHash
	result.BlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(position.BlockNumber))
	result.TransactionIndex = (*hexutil.Uint64)(&position.Index)
	if position.BaseFee != nil && result.GasFeeCap != nil {
		// price = min(tip, gasFeeCap - baseFee) + baseFee
		price := utils.BigMin(new(big.Int).Add(transaction.GasTipCap(), position.BaseFee), transaction.GasFeeCap())
		result.GasPrice = (*hexutil.Big)(price)
	}
	return result
}

// NewPendingTransaction forms a transaction that is in no block, priced by
// its fee cap.
func NewPendingTransaction(transaction *types.Transaction, chainID *big.Int) *Transaction {
	return newTransaction(transaction, chainID)
}

func newTransaction(transaction *types.Transaction, chainID *big.Int) *Transaction {
	// A replay-protected transaction is read with the most permissive signer,
	// since signers are backwards-compatible; one without protection with
	// the homestead signer, as its chain id is zero.
	var signer types.Signer = types.HomesteadSigner{}
	if transaction.Protected() {
		signer = types.LatestSignerForChainID(transaction.ChainId())
	}
	from, _ := internaltx.Sender(signer, transaction)
	transactionChainID := transaction.ChainId()
	if from == (common.Address{}) {
		// The zero sender is an internal transaction, whose signature does
		// not carry the chain id.
		transactionChainID = chainID
	}
	v, r, s := transaction.RawSignatureValues()
	result := &Transaction{
		Type:     hexutil.Uint64(transaction.Type()),
		From:     from,
		Gas:      hexutil.Uint64(transaction.Gas()),
		GasPrice: (*hexutil.Big)(transaction.GasPrice()),
		Hash:     transaction.Hash(),
		Input:    hexutil.Bytes(transaction.Data()),
		Nonce:    hexutil.Uint64(transaction.Nonce()),
		To:       transaction.To(),
		Value:    (*hexutil.Big)(transaction.Value()),
		ChainID:  (*hexutil.Big)(transactionChainID),
		V:        (*hexutil.Big)(v),
		R:        (*hexutil.Big)(r),
		S:        (*hexutil.Big)(s),
	}
	switch transaction.Type() {
	case types.AccessListTxType:
		addAccessList(transaction, result)
	case types.DynamicFeeTxType:
		addAccessList(transaction, result)
		addFeeCaps(transaction, result)
	case types.BlobTxType:
		// The network supports blob transactions without blobs, which are
		// dynamic fee transactions.
		addAccessList(transaction, result)
		addFeeCaps(transaction, result)
		result.MaxFeePerBlobGas = (*hexutil.Big)(transaction.BlobGasFeeCap())
		result.BlobVersionedHashes = transaction.BlobHashes()
		if result.BlobVersionedHashes == nil {
			result.BlobVersionedHashes = []common.Hash{}
		}
	case types.SetCodeTxType:
		addAccessList(transaction, result)
		addFeeCaps(transaction, result)
		result.AuthorizationList = transaction.SetCodeAuthorizations()
	}
	return result
}

func addAccessList(transaction *types.Transaction, result *Transaction) {
	accessList := transaction.AccessList()
	result.Accesses = &accessList
}

// addFeeCaps adds the fee caps of a dynamic fee, and prices the transaction
// by its fee cap, as one in no block.
func addFeeCaps(transaction *types.Transaction, result *Transaction) {
	result.GasFeeCap = (*hexutil.Big)(transaction.GasFeeCap())
	result.GasTipCap = (*hexutil.Big)(transaction.GasTipCap())
	result.GasPrice = (*hexutil.Big)(transaction.GasFeeCap())
}
