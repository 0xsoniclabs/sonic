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
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// GetTxData extracts the inner TxData from a given transaction, which is
// handy for mutating transactions in various contexts.
func GetTxData(tx *types.Transaction) (types.TxData, error) {
	// TODO: consider adding a modification to Sonic's go-ethereum fork to
	// enable a direct call to tx.inner.copy(), having the same effect.
	if tx == nil {
		return nil, fmt.Errorf("transaction is nil")
	}
	return getTxDataInternal(tx)
}

func getTxDataInternal(tx txDataSource) (types.TxData, error) {

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
		chainId, err1 := toUint256(tx.ChainId())
		gasTipCap, err2 := toUint256(tx.GasTipCap())
		gasFeeCap, err3 := toUint256(tx.GasFeeCap())
		value, err4 := toUint256(tx.Value())
		blobFeeCap, err5 := toUint256(tx.BlobGasFeeCap())
		v, err6 := toUint256(v)
		r, err7 := toUint256(r)
		s, err8 := toUint256(s)

		err := errors.Join(err1, err2, err3, err4, err5, err6, err7, err8)
		if err != nil {
			return nil, err
		}

		if tx.To() == nil {
			return nil, fmt.Errorf("blob transactions must have a recipient")
		}

		txData = &types.BlobTx{
			ChainID:    chainId,
			Nonce:      tx.Nonce(),
			GasTipCap:  gasTipCap,
			GasFeeCap:  gasFeeCap,
			Gas:        tx.Gas(),
			To:         *tx.To(),
			Value:      value,
			Data:       tx.Data(),
			AccessList: tx.AccessList(),
			BlobFeeCap: blobFeeCap,
			BlobHashes: tx.BlobHashes(),
			V:          v,
			R:          r,
			S:          s,
		}

	case types.SetCodeTxType:

		chainId, err1 := toUint256(tx.ChainId())
		gasTipCap, err2 := toUint256(tx.GasTipCap())
		gasFeeCap, err3 := toUint256(tx.GasFeeCap())
		value, err4 := toUint256(tx.Value())
		v, err5 := toUint256(v)
		r, err6 := toUint256(r)
		s, err7 := toUint256(s)

		err := errors.Join(err1, err2, err3, err4, err5, err6, err7)
		if err != nil {
			return nil, err
		}

		if tx.To() == nil {
			return nil, fmt.Errorf("set code transactions must have a recipient")
		}

		txData = &types.SetCodeTx{
			ChainID:    chainId,
			Nonce:      tx.Nonce(),
			GasTipCap:  gasTipCap,
			GasFeeCap:  gasFeeCap,
			Gas:        tx.Gas(),
			To:         *tx.To(),
			Value:      value,
			Data:       tx.Data(),
			AccessList: tx.AccessList(),
			AuthList:   tx.SetCodeAuthorizations(),
			V:          v,
			R:          r,
			S:          s,
		}

	default:
		return nil, fmt.Errorf("unsupported transaction type: %d", tx.Type())

	}
	return txData, nil
}

type txDataSource interface {
	ChainId() *big.Int
	Nonce() uint64
	GasPrice() *big.Int
	GasFeeCap() *big.Int
	GasTipCap() *big.Int
	Gas() uint64
	To() *common.Address
	Value() *big.Int
	Data() []byte
	AccessList() types.AccessList
	BlobGasFeeCap() *big.Int
	BlobHashes() []common.Hash
	SetCodeAuthorizations() []types.SetCodeAuthorization
	Type() uint8
	RawSignatureValues() (v, r, s *big.Int)
}

func toUint256(value *big.Int) (*uint256.Int, error) {
	if value == nil {
		return nil, nil
	}
	if value.Sign() < 0 {
		return nil, fmt.Errorf("out of uint256 domain: %v", value)
	}
	res, overflow := uint256.FromBig(value)
	if overflow {
		return nil, fmt.Errorf("out of uint256 domain: %v", value)
	}
	return res, nil
}
