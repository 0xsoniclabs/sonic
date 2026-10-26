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

package format

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
)

// Receipt returns the JSON form of the receipt of a transaction of a block.
// The logs of the receipt are shared; the form holds copies that carry the
// location of the transaction.
func Receipt(header *evmcore.EvmHeader, transaction *types.Transaction, index uint64, receipt *types.Receipt, chainID *big.Int) map[string]any {
	from, _ := internaltx.Sender(types.LatestSignerForChainID(chainID), transaction)
	fields := map[string]any{
		"blockHash":         header.Hash,
		"blockNumber":       hexutil.Uint64(header.Number.Uint64()),
		"transactionHash":   transaction.Hash(),
		"transactionIndex":  hexutil.Uint64(index),
		"from":              from,
		"to":                transaction.To(),
		"gasUsed":           hexutil.Uint64(receipt.GasUsed),
		"cumulativeGasUsed": hexutil.Uint64(receipt.CumulativeGasUsed),
		"contractAddress":   nil,
		"logs":              locatedLogs(receipt.Logs, header, transaction.Hash(), index),
		"logsBloom":         &receipt.Bloom,
		"type":              hexutil.Uint(transaction.Type()),
		"effectiveGasPrice": effectiveGasPrice(transaction, header.BaseFee),
	}
	if len(receipt.PostState) > 0 {
		fields["root"] = hexutil.Bytes(receipt.PostState)
	} else {
		fields["status"] = hexutil.Uint(receipt.Status)
	}
	if receipt.Logs == nil {
		fields["logs"] = [][]*types.Log{}
	}
	if transaction.To() == nil {
		fields["contractAddress"] = receipt.ContractAddress
	}
	return fields
}

// locatedLogs copies the logs of a receipt and sets the location of their
// transaction on the copies.
func locatedLogs(logs []*types.Log, header *evmcore.EvmHeader, transactionHash common.Hash, index uint64) []*types.Log {
	located := make([]*types.Log, len(logs))
	for position, log := range logs {
		copied := *log
		copied.TxHash = transactionHash
		copied.BlockHash = header.Hash
		copied.BlockNumber = header.Number.Uint64()
		copied.TxIndex = uint(index)
		if copied.Topics == nil {
			copied.Topics = []common.Hash{}
		}
		located[position] = &copied
	}
	return located
}

// effectiveGasPrice is the price the transaction paid per gas: zero for an
// internal or sponsored transaction, as block processing charges them.
func effectiveGasPrice(transaction *types.Transaction, baseFee *big.Int) hexutil.Uint64 {
	switch {
	case internaltx.IsInternal(transaction) || subsidies.IsSponsorshipRequest(transaction):
		return 0
	case baseFee == nil:
		return hexutil.Uint64(transaction.GasPrice().Uint64())
	}
	price := new(big.Int).Add(baseFee, transaction.EffectiveGasTipValue(baseFee))
	return hexutil.Uint64(price.Uint64())
}
