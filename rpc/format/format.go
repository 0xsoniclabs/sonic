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
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
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

// NewTransaction returns the JSON form of a transaction at its position in a
// block; a zero block hash means the transaction is not in a block.
func NewTransaction(tx *types.Transaction, blockHash common.Hash, blockNumber, index uint64, baseFee, chainID *big.Int) *Transaction {
	// For replay-protected transactions use the most permissive signer, as
	// signers accept the transactions of their predecessors; the others are
	// homestead transactions with a chain id of zero.
	var signer types.Signer
	if tx.Protected() {
		signer = types.LatestSignerForChainID(tx.ChainId())
	} else {
		signer = types.HomesteadSigner{}
	}
	from, _ := internaltx.Sender(signer, tx)

	// An internal transaction has no signature to take the chain id from.
	txChainID := tx.ChainId()
	if from == (common.Address{}) {
		txChainID = chainID
	}
	v, r, s := tx.RawSignatureValues()
	result := &Transaction{
		Type:     hexutil.Uint64(tx.Type()),
		From:     from,
		Gas:      hexutil.Uint64(tx.Gas()),
		GasPrice: (*hexutil.Big)(tx.GasPrice()),
		Hash:     tx.Hash(),
		Input:    hexutil.Bytes(tx.Data()),
		Nonce:    hexutil.Uint64(tx.Nonce()),
		To:       tx.To(),
		Value:    (*hexutil.Big)(tx.Value()),
		ChainID:  (*hexutil.Big)(txChainID),
		V:        (*hexutil.Big)(v),
		R:        (*hexutil.Big)(r),
		S:        (*hexutil.Big)(s),
	}
	if blockHash != (common.Hash{}) {
		result.BlockHash = &blockHash
		result.BlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(blockNumber))
		result.TransactionIndex = (*hexutil.Uint64)(&index)
	}

	copyAccessList := func() {
		al := tx.AccessList()
		result.Accesses = &al
	}
	copyDynamicPricingFields := func() {
		result.GasFeeCap = (*hexutil.Big)(tx.GasFeeCap())
		result.GasTipCap = (*hexutil.Big)(tx.GasTipCap())
		// A mined transaction pays min(tip, feeCap - baseFee) + baseFee.
		if baseFee != nil && blockHash != (common.Hash{}) {
			price := utils.BigMin(new(big.Int).Add(tx.GasTipCap(), baseFee), tx.GasFeeCap())
			result.GasPrice = (*hexutil.Big)(price)
		} else {
			result.GasPrice = (*hexutil.Big)(tx.GasFeeCap())
		}
	}

	switch tx.Type() {
	case types.AccessListTxType:
		copyAccessList()
	case types.DynamicFeeTxType:
		copyAccessList()
		copyDynamicPricingFields()
	case types.BlobTxType:
		// Sonic accepts blob transactions without blobs; they are priced as
		// dynamic fee transactions.
		copyAccessList()
		copyDynamicPricingFields()
		result.MaxFeePerBlobGas = (*hexutil.Big)(tx.BlobGasFeeCap())
		result.BlobVersionedHashes = tx.BlobHashes()
		if result.BlobVersionedHashes == nil {
			result.BlobVersionedHashes = make([]common.Hash, 0)
		}
	case types.SetCodeTxType:
		copyAccessList()
		copyDynamicPricingFields()
		result.AuthorizationList = tx.SetCodeAuthorizations()
	}
	return result
}

// NewPendingTransaction returns the JSON form of a transaction that is not in
// a block.
func NewPendingTransaction(tx *types.Transaction, baseFee, chainID *big.Int) *Transaction {
	return NewTransaction(tx, common.Hash{}, 0, 0, baseFee, chainID)
}

// TransactionAt returns the JSON form of the transaction at an index of a
// block, nil if the block has none there.
func TransactionAt(b *evmcore.EvmBlock, index uint64, chainID *big.Int) *Transaction {
	if index >= uint64(len(b.Transactions)) {
		return nil
	}
	return NewTransaction(b.Transactions[index], b.Hash, b.NumberU64(), index, b.BaseFee, chainID)
}

// RawTransactionAt returns the binary encoding of the transaction at an index
// of a block, nil if the block has none there.
func RawTransactionAt(b *evmcore.EvmBlock, index uint64) hexutil.Bytes {
	if index >= uint64(len(b.Transactions)) {
		return nil
	}
	blob, _ := b.Transactions[index].MarshalBinary()
	return blob
}

// Block returns the JSON form of a block, with transaction hashes or, with
// fullTx, whole transactions.
func Block(block *evmcore.EvmBlock, receipts types.Receipts, fullTx bool, chainID *big.Int) *evmcore.EvmBlockJson {
	size := hexutil.Uint64(block.EthBlock().Size())
	enc := &evmcore.EvmBlockJson{
		EvmHeaderJson: block.Header().ToJson(receipts),
		Size:          &size,
	}
	txs := make([]any, len(block.Transactions))
	for i, tx := range block.Transactions {
		if fullTx {
			txs[i] = TransactionAt(block, uint64(i), chainID)
		} else {
			txs[i] = tx.Hash()
		}
	}
	enc.Txs = txs
	enc.Uncles = make([]common.Hash, 0)
	return enc
}

// Receipt returns the JSON form of the receipt of a transaction at an index of
// a block. The logs are copies: the receipt is shared with other requests.
func Receipt(header *evmcore.EvmHeader, tx *types.Transaction, txIndex uint64, receipt *types.Receipt, signer types.Signer) map[string]any {
	logs := slices.Clone(receipt.Logs)
	for i := range logs {
		l := *logs[i]
		l.TxHash = tx.Hash()
		l.BlockHash = header.Hash
		l.BlockNumber = header.Number.Uint64()
		l.TxIndex = uint(txIndex)
		if l.Topics == nil {
			l.Topics = []common.Hash{}
		}
		logs[i] = &l
	}
	from, _ := internaltx.Sender(signer, tx)

	fields := map[string]any{
		"blockHash":         header.Hash,
		"blockNumber":       hexutil.Uint64(header.Number.Uint64()),
		"transactionHash":   tx.Hash(),
		"transactionIndex":  hexutil.Uint64(txIndex),
		"from":              from,
		"to":                tx.To(),
		"gasUsed":           hexutil.Uint64(receipt.GasUsed),
		"cumulativeGasUsed": hexutil.Uint64(receipt.CumulativeGasUsed),
		"contractAddress":   nil,
		"logs":              logs,
		"logsBloom":         &receipt.Bloom,
		"type":              hexutil.Uint(tx.Type()),
	}
	switch {
	case internaltx.IsInternal(tx) || subsidies.IsSponsorshipRequest(tx):
		fields["effectiveGasPrice"] = hexutil.Uint64(0)
	case header.BaseFee == nil:
		fields["effectiveGasPrice"] = hexutil.Uint64(tx.GasPrice().Uint64())
	default:
		gasTip := tx.EffectiveGasTipValue(header.BaseFee)
		gasPrice := new(big.Int).Add(header.BaseFee, gasTip)
		fields["effectiveGasPrice"] = hexutil.Uint64(gasPrice.Uint64())
	}
	if len(receipt.PostState) > 0 {
		fields["root"] = hexutil.Bytes(receipt.PostState)
	} else {
		fields["status"] = hexutil.Uint(receipt.Status)
	}
	if receipt.Logs == nil {
		fields["logs"] = [][]*types.Log{}
	}
	if tx.To() == nil {
		fields["contractAddress"] = receipt.ContractAddress
	}
	return fields
}

// Logs returns logs as an answer lists them: never null, and with a topic list
// that is never null. A log that needs a change is copied, since the logs the
// node hands out are shared.
func Logs(logs []*types.Log) []*types.Log {
	out := make([]*types.Log, len(logs))
	for i, l := range logs {
		if l.Topics == nil {
			c := *l
			c.Topics = []common.Hash{}
			l = &c
		}
		out[i] = l
	}
	return out
}

// Hashes returns hashes as an answer lists them: never null.
func Hashes(hashes []common.Hash) []common.Hash {
	if hashes == nil {
		return []common.Hash{}
	}
	return hashes
}
