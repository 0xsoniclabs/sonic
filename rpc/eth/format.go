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

package eth

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
)

// RPCMarshalBlock converts the given block to the RPC output which depends on fullTx. If inclTx is true transactions are
// returned. When fullTx is true the returned block contains full transaction details, otherwise it will only contain
// transaction hashes.
func RPCMarshalBlock(block *evmcore.EvmBlock, receipts types.Receipts, inclTx bool, fullTx bool, chainId *big.Int) (*evmcore.EvmBlockJson, error) {
	size := hexutil.Uint64(block.EthBlock().Size()) // RPC encoded storage size
	json := &evmcore.EvmBlockJson{
		EvmHeaderJson: block.Header().ToJson(receipts),
		Size:          &size,
	}

	if inclTx {
		formatTx := func(tx *types.Transaction) (interface{}, error) {
			return tx.Hash(), nil
		}
		if fullTx {
			formatTx = func(tx *types.Transaction) (interface{}, error) {
				return newRPCTransactionFromBlockHash(block, tx.Hash(), chainId), nil
			}
		}
		txs := block.Transactions
		transactions := make([]interface{}, len(txs))
		var err error
		for i, tx := range txs {
			if transactions[i], err = formatTx(tx); err != nil {
				return nil, err
			}
		}
		json.Txs = transactions
	}
	json.Uncles = make([]common.Hash, 0)

	return json, nil
}

// RPCTransaction represents a transaction that will serialize to the RPC representation of a transaction
type RPCTransaction struct {
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

// newRPCTransaction returns a transaction that will serialize to the RPC
// representation, with the given location metadata set (if available).
func newRPCTransaction(tx *types.Transaction, blockHash common.Hash, blockNumber uint64, index uint64, baseFee *big.Int, chainId *big.Int) *RPCTransaction {
	// Determine the signer. For replay-protected transactions, use the most permissive
	// signer, because we assume that signers are backwards-compatible with old
	// transactions. For non-protected transactions, the homestead signer signer is used
	// because the return value of ChainId is zero for those transactions.
	var signer types.Signer
	if tx.Protected() {
		signer = types.LatestSignerForChainID(tx.ChainId())
	} else {
		signer = types.HomesteadSigner{}
	}
	from, _ := internaltx.Sender(signer, tx)

	// Set the chain id for internal transaction
	txChainId := tx.ChainId()
	if from == (common.Address{}) {
		// if the sender is the zero address,
		// this is an internal transaction and chainId
		// cannot be retrieved from the signature
		txChainId = chainId
	}
	v, r, s := tx.RawSignatureValues()
	result := &RPCTransaction{
		Type:     hexutil.Uint64(tx.Type()),
		From:     from,
		Gas:      hexutil.Uint64(tx.Gas()),
		GasPrice: (*hexutil.Big)(tx.GasPrice()),
		Hash:     tx.Hash(),
		Input:    hexutil.Bytes(tx.Data()),
		Nonce:    hexutil.Uint64(tx.Nonce()),
		To:       tx.To(),
		Value:    (*hexutil.Big)(tx.Value()),
		ChainID:  (*hexutil.Big)(txChainId),
		V:        (*hexutil.Big)(v),
		R:        (*hexutil.Big)(r),
		S:        (*hexutil.Big)(s),
	}
	if blockHash != (common.Hash{}) {
		result.BlockHash = &blockHash
		result.BlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(blockNumber))
		result.TransactionIndex = (*hexutil.Uint64)(&index)
	}

	copyAccessList := func(tx *types.Transaction, result *RPCTransaction) {
		al := tx.AccessList()
		result.Accesses = &al
	}

	copyDynamicPricingFields := func(tx *types.Transaction, result *RPCTransaction) {
		result.GasFeeCap = (*hexutil.Big)(tx.GasFeeCap())
		result.GasTipCap = (*hexutil.Big)(tx.GasTipCap())
		// if the transaction has been mined, compute the effective gas price
		if baseFee != nil && blockHash != (common.Hash{}) {
			// price = min(tip, gasFeeCap - baseFee) + baseFee
			price := utils.BigMin(new(big.Int).Add(tx.GasTipCap(), baseFee), tx.GasFeeCap())
			result.GasPrice = (*hexutil.Big)(price)
		} else {
			result.GasPrice = (*hexutil.Big)(tx.GasFeeCap())
		}
	}

	copyBlobFields := func(tx *types.Transaction, result *RPCTransaction) {
		result.MaxFeePerBlobGas = (*hexutil.Big)(tx.BlobGasFeeCap())
		result.BlobVersionedHashes = tx.BlobHashes()
		if result.BlobVersionedHashes == nil {
			result.BlobVersionedHashes = make([]common.Hash, 0)
		}
	}

	copyAuthorizationList := func(tx *types.Transaction, result *RPCTransaction) {
		result.AuthorizationList = tx.SetCodeAuthorizations()
	}

	switch tx.Type() {
	case types.AccessListTxType:
		copyAccessList(tx, result)
	case types.DynamicFeeTxType:
		copyAccessList(tx, result)
		copyDynamicPricingFields(tx, result)
	case types.BlobTxType:
		// BLOB NOTE: the current sonic network supports blobTx so long as they don not contain blobs
		// for this reason they are equivalent to the dynamic fee tx type
		copyAccessList(tx, result)
		copyDynamicPricingFields(tx, result)
		copyBlobFields(tx, result)
	case types.SetCodeTxType:
		copyAccessList(tx, result)
		copyDynamicPricingFields(tx, result)
		copyAuthorizationList(tx, result)
	}
	return result
}

// NewRPCPendingTransaction returns a transaction that will serialize to the RPC representation.
func NewRPCPendingTransaction(tx *types.Transaction, baseFee *big.Int, chainId *big.Int) *RPCTransaction {
	return newRPCPendingTransaction(tx, baseFee, chainId)
}

// newRPCPendingTransaction returns a pending transaction that will serialize to the RPC representation
func newRPCPendingTransaction(tx *types.Transaction, baseFee *big.Int, chainId *big.Int) *RPCTransaction {
	return newRPCTransaction(tx, common.Hash{}, 0, 0, baseFee, chainId)
}

// newRPCTransactionFromBlockIndex returns a transaction that will serialize to the RPC representation.
func newRPCTransactionFromBlockIndex(b *evmcore.EvmBlock, index uint64, chainId *big.Int) *RPCTransaction {
	txs := b.Transactions
	if index >= uint64(len(txs)) {
		return nil
	}
	return newRPCTransaction(txs[index], b.Hash, b.NumberU64(), index, b.BaseFee, chainId)
}

// newRPCRawTransactionFromBlockIndex returns the bytes of a transaction given a block and a transaction index.
func newRPCRawTransactionFromBlockIndex(b *evmcore.EvmBlock, index uint64) hexutil.Bytes {
	txs := b.Transactions
	if index >= uint64(len(txs)) {
		return nil
	}
	blob, _ := txs[index].MarshalBinary()
	return blob
}

// newRPCTransactionFromBlockHash returns a transaction that will serialize to the RPC representation.
func newRPCTransactionFromBlockHash(b *evmcore.EvmBlock, hash common.Hash, chainId *big.Int) *RPCTransaction {
	for idx, tx := range b.Transactions {
		if tx.Hash() == hash {
			return newRPCTransactionFromBlockIndex(b, uint64(idx), chainId)
		}
	}
	return nil
}

// accessListResult returns an optional accesslist
// Its the result of the `debug_createAccessList` RPC call.
