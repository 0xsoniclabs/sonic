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

package utils

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// GetTxData extracts the inner TxData from a given transaction, which is
// handy for mutating transactions in various contexts.
func GetTxData(tx *types.Transaction) types.TxData {

	// TODO: consider adding a modification to Sonic's go-ethereum fork to
	// enable a direct call to tx.inner.copy(), having the same effect.

	// Manually create a copy of the transactions's inner data type.
	var txData types.TxData
	v, r, s := tx.RawSignatureValues()
	switch tx.Type() {
	case types.LegacyTxType:
		txData = &types.LegacyTx{
			Nonce:    tx.Nonce(),
			GasPrice: tx.GasPrice(),
			Gas:      tx.Gas(),
			To:       tx.To(),
			Value:    tx.Value(),
			Data:     tx.Data(),
			V:        v,
			R:        r,
			S:        s,
		}
	case types.AccessListTxType:
		txData = &types.AccessListTx{
			ChainID:    tx.ChainId(),
			Nonce:      tx.Nonce(),
			GasPrice:   tx.GasPrice(),
			Gas:        tx.Gas(),
			To:         tx.To(),
			Value:      tx.Value(),
			Data:       tx.Data(),
			AccessList: tx.AccessList(),
			V:          v,
			R:          r,
			S:          s,
		}
	case types.DynamicFeeTxType:
		txData = &types.DynamicFeeTx{
			ChainID:    tx.ChainId(),
			Nonce:      tx.Nonce(),
			GasTipCap:  tx.GasTipCap(),
			GasFeeCap:  tx.GasFeeCap(),
			Gas:        tx.Gas(),
			To:         tx.To(),
			Value:      tx.Value(),
			Data:       tx.Data(),
			AccessList: tx.AccessList(),
			V:          v,
			R:          r,
			S:          s,
		}
	case types.BlobTxType:
		txData = &types.BlobTx{
			ChainID:    mustToUint256(tx.ChainId()),
			Nonce:      tx.Nonce(),
			GasTipCap:  mustToUint256(tx.GasTipCap()),
			GasFeeCap:  mustToUint256(tx.GasFeeCap()),
			Gas:        tx.Gas(),
			To:         *tx.To(),
			Value:      mustToUint256(tx.Value()),
			Data:       tx.Data(),
			AccessList: tx.AccessList(),
			BlobFeeCap: mustToUint256(tx.BlobGasFeeCap()),
			BlobHashes: tx.BlobHashes(),
			V:          mustToUint256(v),
			R:          mustToUint256(r),
			S:          mustToUint256(s),
		}

	case types.SetCodeTxType:
		txData = &types.SetCodeTx{
			ChainID:    mustToUint256(tx.ChainId()),
			Nonce:      tx.Nonce(),
			GasTipCap:  mustToUint256(tx.GasTipCap()),
			GasFeeCap:  mustToUint256(tx.GasFeeCap()),
			Gas:        tx.Gas(),
			To:         *tx.To(),
			Value:      mustToUint256(tx.Value()),
			Data:       tx.Data(),
			AccessList: tx.AccessList(),
			AuthList:   tx.SetCodeAuthorizations(),
			V:          mustToUint256(v),
			R:          mustToUint256(r),
			S:          mustToUint256(s),
		}
	}
	return txData
}

func mustToUint256(value *big.Int) *uint256.Int {
	if value == nil {
		return nil
	}
	if value.Sign() < 0 {
		panic(fmt.Sprintf("out of uint256 domain: %v", value))
	}
	res, overflow := uint256.FromBig(value)
	if overflow {
		panic(fmt.Sprintf("out of uint256 domain: %v", value))
	}
	return res
}
