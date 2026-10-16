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
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
)

// isSponsorshipRequest tells a transaction that asks for sponsorship from a
// pre-allocated pool: a non-internal transaction with a recipient and a
// maximum gas price of zero. It is the node's subsidies.IsSponsorshipRequest;
// a test keeps the two equal.
func isSponsorshipRequest(tx *types.Transaction) bool {
	return tx != nil &&
		!internaltx.IsInternal(tx) &&
		tx.To() != nil &&
		tx.GasPrice().Sign() == 0
}

// formatTxReceipt encodes a transaction receipt into the expected API output.
func formatTxReceipt(chainID uint64, header *evmcore.EvmHeader, tx *types.Transaction, txIndex uint64, receipt *types.Receipt) map[string]any {
	// Clone the logs before adding transaction meta data to avoid data races
	// due to concurrent accesses.
	logs := slices.Clone(receipt.Logs)
	for i := range logs {
		l := new(types.Log)
		*l = *logs[i] // shallow copy
		logs[i] = l
		l.TxHash = tx.Hash()
		l.BlockHash = header.Hash
		l.BlockNumber = header.Number.Uint64()
		l.TxIndex = uint(txIndex) /* logs cache poisoning hot fix */
		if l.Topics == nil {
			l.Topics = []common.Hash{}
		}
	}

	// Derive the sender.
	signer := types.LatestSignerForChainID(new(big.Int).SetUint64(chainID))
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
	// Assign the effective gas price paid.
	if internaltx.IsInternal(tx) || isSponsorshipRequest(tx) {
		fields["effectiveGasPrice"] = hexutil.Uint64(0)
	} else if header.BaseFee == nil {
		fields["effectiveGasPrice"] = hexutil.Uint64(tx.GasPrice().Uint64())
	} else {
		gasTip := tx.EffectiveGasTipValue(header.BaseFee)
		gasPrice := new(big.Int).Add(header.BaseFee, gasTip)
		fields["effectiveGasPrice"] = hexutil.Uint64(gasPrice.Uint64())
	}
	// Assign receipt status or post state.
	if len(receipt.PostState) > 0 {
		fields["root"] = hexutil.Bytes(receipt.PostState)
	} else {
		fields["status"] = hexutil.Uint(receipt.Status)
	}
	if receipt.Logs == nil {
		fields["logs"] = [][]*types.Log{}
	}
	// Transactions without a recipient deploy a contract.
	if tx.To() == nil {
		fields["contractAddress"] = receipt.ContractAddress
	}

	return fields
}
