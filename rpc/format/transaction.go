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

// Package format holds the JSON forms of blocks, headers, transactions,
// receipts and logs that several namespaces answer with.
package format

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
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
	Recipient           *common.Address              `json:"to"`
	TransactionIndex    *hexutil.Uint64              `json:"transactionIndex"`
	Value               *hexutil.Big                 `json:"value"`
	Type                hexutil.Uint64               `json:"type"`
	Accesses            *types.AccessList            `json:"accessList,omitempty"`
	ChainID             *hexutil.Big                 `json:"chainId,omitempty"`
	SignatureV          *hexutil.Big                 `json:"v"`
	SignatureR          *hexutil.Big                 `json:"r"`
	SignatureS          *hexutil.Big                 `json:"s"`
	MaxFeePerBlobGas    *hexutil.Big                 `json:"maxFeePerBlobGas"`
	BlobVersionedHashes []common.Hash                `json:"blobVersionedHashes"`
	AuthorizationList   []types.SetCodeAuthorization `json:"authorizationList,omitempty"`
}

// Location is where a transaction is in the chain.
type Location struct {
	BlockHash   common.Hash
	BlockNumber uint64
	Index       uint64
}

// NewTransaction returns the JSON form of a transaction at a location. A
// zero block hash marks a transaction that is not in a block. The base fee
// of the block prices a dynamic-fee transaction; chainID is the chain id of
// an internal transaction, which carries none.
func NewTransaction(transaction *types.Transaction, location Location, baseFee *big.Int, chainID *big.Int) *Transaction {
	from, _ := internaltx.Sender(signerOf(transaction), transaction)
	transactionChainID := transaction.ChainId()
	if from == (common.Address{}) {
		transactionChainID = chainID
	}
	signatureV, signatureR, signatureS := transaction.RawSignatureValues()
	result := &Transaction{
		Type:       hexutil.Uint64(transaction.Type()),
		From:       from,
		Gas:        hexutil.Uint64(transaction.Gas()),
		GasPrice:   (*hexutil.Big)(transaction.GasPrice()),
		Hash:       transaction.Hash(),
		Input:      hexutil.Bytes(transaction.Data()),
		Nonce:      hexutil.Uint64(transaction.Nonce()),
		Recipient:  transaction.To(),
		Value:      (*hexutil.Big)(transaction.Value()),
		ChainID:    (*hexutil.Big)(transactionChainID),
		SignatureV: (*hexutil.Big)(signatureV),
		SignatureR: (*hexutil.Big)(signatureR),
		SignatureS: (*hexutil.Big)(signatureS),
	}
	inBlock := location.BlockHash != (common.Hash{})
	if inBlock {
		result.BlockHash = &location.BlockHash
		result.BlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(location.BlockNumber))
		result.TransactionIndex = (*hexutil.Uint64)(&location.Index)
	}
	switch transaction.Type() {
	case types.AccessListTxType:
		result.setAccessList(transaction)
	case types.DynamicFeeTxType:
		result.setAccessList(transaction)
		result.setDynamicPricing(transaction, baseFee, inBlock)
	case types.BlobTxType:
		// Sonic accepts blob transactions as long as they carry no blobs, so
		// they are priced as dynamic-fee transactions.
		result.setAccessList(transaction)
		result.setDynamicPricing(transaction, baseFee, inBlock)
		result.setBlobFields(transaction)
	case types.SetCodeTxType:
		result.setAccessList(transaction)
		result.setDynamicPricing(transaction, baseFee, inBlock)
		result.AuthorizationList = transaction.SetCodeAuthorizations()
	}
	return result
}

// signerOf returns the most permissive signer of a replay-protected
// transaction, and the homestead signer of one that is not, whose chain id is
// zero.
func signerOf(transaction *types.Transaction) types.Signer {
	if transaction.Protected() {
		return types.LatestSignerForChainID(transaction.ChainId())
	}
	return types.HomesteadSigner{}
}

func (result *Transaction) setAccessList(transaction *types.Transaction) {
	accessList := transaction.AccessList()
	result.Accesses = &accessList
}

// setDynamicPricing sets the fee caps and, for a transaction in a block, the
// effective price: min(tip + base fee, fee cap).
func (result *Transaction) setDynamicPricing(transaction *types.Transaction, baseFee *big.Int, inBlock bool) {
	result.GasFeeCap = (*hexutil.Big)(transaction.GasFeeCap())
	result.GasTipCap = (*hexutil.Big)(transaction.GasTipCap())
	if baseFee != nil && inBlock {
		price := utils.BigMin(new(big.Int).Add(transaction.GasTipCap(), baseFee), transaction.GasFeeCap())
		result.GasPrice = (*hexutil.Big)(price)
		return
	}
	result.GasPrice = (*hexutil.Big)(transaction.GasFeeCap())
}

func (result *Transaction) setBlobFields(transaction *types.Transaction) {
	result.MaxFeePerBlobGas = (*hexutil.Big)(transaction.BlobGasFeeCap())
	result.BlobVersionedHashes = transaction.BlobHashes()
	if result.BlobVersionedHashes == nil {
		result.BlobVersionedHashes = make([]common.Hash, 0)
	}
}

// PendingTransaction returns the JSON form of a transaction that is in no
// block.
func PendingTransaction(transaction *types.Transaction, baseFee *big.Int, chainID *big.Int) *Transaction {
	return NewTransaction(transaction, Location{}, baseFee, chainID)
}

// TransactionAt returns the JSON form of the transaction of a block at an
// index, nil if the block has no transaction there.
func TransactionAt(block *evmcore.EvmBlock, index uint64, chainID *big.Int) *Transaction {
	if index >= uint64(len(block.Transactions)) {
		return nil
	}
	location := Location{BlockHash: block.Hash, BlockNumber: block.NumberU64(), Index: index}
	return NewTransaction(block.Transactions[index], location, block.BaseFee, chainID)
}

// RawTransactionAt returns the binary encoding of the transaction of a block
// at an index, nil if the block has no transaction there.
func RawTransactionAt(block *evmcore.EvmBlock, index uint64) hexutil.Bytes {
	if index >= uint64(len(block.Transactions)) {
		return nil
	}
	encoded, _ := block.Transactions[index].MarshalBinary()
	return encoded
}
