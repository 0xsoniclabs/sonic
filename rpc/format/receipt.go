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

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// Receipt forms the receipt of a transaction of a block. The receipt and its
// logs are shared; the logs are copied before the transaction's position is
// put into them.
func Receipt(header *evmcore.EvmHeader, transaction *types.Transaction, index uint64, receipt *types.Receipt, chainID *big.Int) map[string]any {
	logs := make([]*types.Log, len(receipt.Logs))
	for i, shared := range receipt.Logs {
		log := *shared
		log.TxHash = transaction.Hash()
		log.BlockHash = header.Hash
		log.BlockNumber = header.Number.Uint64()
		log.TxIndex = uint(index)
		if log.Topics == nil {
			log.Topics = []common.Hash{}
		}
		logs[i] = &log
	}
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
		"logs":              logs,
		"logsBloom":         &receipt.Bloom,
		"type":              hexutil.Uint(transaction.Type()),
		"effectiveGasPrice": effectiveGasPrice(header.BaseFee, transaction),
	}
	if receipt.Logs == nil {
		fields["logs"] = [][]*types.Log{}
	}
	if len(receipt.PostState) > 0 {
		fields["root"] = hexutil.Bytes(receipt.PostState)
	} else {
		fields["status"] = hexutil.Uint(receipt.Status)
	}
	if transaction.To() == nil {
		fields["contractAddress"] = receipt.ContractAddress
	}
	return fields
}

// effectiveGasPrice is the price a transaction paid per gas: nothing for an
// internal or a sponsored one, as block processing charges it.
func effectiveGasPrice(baseFee *big.Int, transaction *types.Transaction) hexutil.Uint64 {
	switch {
	case internaltx.IsInternal(transaction) || subsidies.IsSponsorshipRequest(transaction):
		return 0
	case baseFee == nil:
		return hexutil.Uint64(transaction.GasPrice().Uint64())
	default:
		tip := transaction.EffectiveGasTipValue(baseFee)
		return hexutil.Uint64(new(big.Int).Add(baseFee, tip).Uint64())
	}
}
