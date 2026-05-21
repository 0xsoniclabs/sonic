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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestGetTxData_NilTransaction_ReportsError(t *testing.T) {
	_, err := GetTxData(nil)
	require.ErrorContains(t, err, "transaction is nil")
}

func TestGetTxData_ExtractsAllData(t *testing.T) {

	type msg struct {
		ChainId    *uint256.Int
		Nonce      uint64
		GasPrice   *uint256.Int
		GasFeeCap  *uint256.Int
		GasTipCap  *uint256.Int
		Gas        uint64
		To         *common.Address
		Value      *uint256.Int
		Data       []byte
		AccessList types.AccessList
		BlobHashes []common.Hash
		AuthList   []types.SetCodeAuthorization
		V          *uint256.Int
		R          *uint256.Int
		S          *uint256.Int
	}

	uint256Options := []*uint256.Int{uint256.NewInt(5), uint256.NewInt(100)}

	tests := make([]msg, 0)
	for _, chainId := range uint256Options {
		for _, nonce := range []uint64{0, 1} {
			for _, gasPrice := range uint256Options {
				for _, gasFeeCap := range uint256Options {
					for _, gasTipCap := range uint256Options {
						for _, gas := range []uint64{0, 21000} {
							for _, to := range []*common.Address{nil, {0x01}} {
								for _, value := range uint256Options {
									for _, accessList := range []types.AccessList{
										{},
										{{
											Address:     common.Address{1},
											StorageKeys: []common.Hash{{0x02}, {0x03}},
										}},
									} {
										for _, blobHash := range [][]common.Hash{
											{}, {{0x01}, {0x02}},
										} {
											for _, authList := range [][]types.SetCodeAuthorization{
												{}, {
													{Address: common.Address{1}, Nonce: 0x02},
													{Address: common.Address{3}, Nonce: 0x04},
												},
											} {
												for _, v := range uint256Options {
													for _, r := range uint256Options {
														for _, s := range uint256Options {
															tests = append(tests, msg{
																ChainId:    chainId,
																Nonce:      nonce,
																GasPrice:   gasPrice,
																GasFeeCap:  gasFeeCap,
																GasTipCap:  gasTipCap,
																Gas:        gas,
																To:         to,
																Value:      value,
																Data:       []byte{0x01, 0x02},
																AccessList: accessList,
																BlobHashes: blobHash,
																AuthList:   authList,
																V:          v,
																R:          r,
																S:          s,
															})
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	for _, test := range tests {

		t.Run("LegacyTx", func(t *testing.T) {
			original := &types.LegacyTx{
				Nonce:    test.Nonce,
				GasPrice: test.GasPrice.ToBig(),
				Gas:      test.Gas,
				To:       test.To,
				Value:    test.Value.ToBig(),
				Data:     test.Data,
				V:        test.V.ToBig(),
				R:        test.R.ToBig(),
				S:        test.S.ToBig(),
			}

			tx := types.NewTx(original)

			txData, err := GetTxData(tx)
			require.NoError(t, err)

			restored := txData.(*types.LegacyTx)
			require.Equal(t, original, restored)
		})
		t.Run("AccessListTx", func(t *testing.T) {
			original := &types.AccessListTx{
				ChainID:    test.ChainId.ToBig(),
				Nonce:      test.Nonce,
				GasPrice:   test.GasPrice.ToBig(),
				Gas:        test.Gas,
				To:         test.To,
				Value:      test.Value.ToBig(),
				Data:       test.Data,
				AccessList: test.AccessList,
				V:          test.V.ToBig(),
				R:          test.R.ToBig(),
				S:          test.S.ToBig(),
			}

			tx := types.NewTx(original)

			txData, err := GetTxData(tx)
			require.NoError(t, err)

			restored := txData.(*types.AccessListTx)
			require.Equal(t, original, restored)
		})

		t.Run("DynamicFeeTx", func(t *testing.T) {
			original := &types.DynamicFeeTx{
				ChainID:    test.ChainId.ToBig(),
				Nonce:      test.Nonce,
				GasFeeCap:  test.GasFeeCap.ToBig(),
				GasTipCap:  test.GasTipCap.ToBig(),
				Gas:        test.Gas,
				To:         test.To,
				Value:      test.Value.ToBig(),
				Data:       test.Data,
				AccessList: test.AccessList,
				V:          test.V.ToBig(),
				R:          test.R.ToBig(),
				S:          test.S.ToBig(),
			}

			tx := types.NewTx(original)

			txData, err := GetTxData(tx)
			require.NoError(t, err)

			restored := txData.(*types.DynamicFeeTx)
			require.Equal(t, original, restored)
		})

		t.Run("BlobTx", func(t *testing.T) {
			if test.To == nil {
				t.Skip("BlobTx requires a non-nil To address")
			}
			original := &types.BlobTx{
				ChainID:    test.ChainId,
				Nonce:      test.Nonce,
				GasFeeCap:  test.GasFeeCap,
				GasTipCap:  test.GasTipCap,
				Gas:        test.Gas,
				To:         *test.To,
				Value:      test.Value,
				Data:       test.Data,
				AccessList: test.AccessList,
				BlobFeeCap: test.GasPrice, // < reuse of deprecated field
				BlobHashes: test.BlobHashes,
				V:          test.V,
				R:          test.R,
				S:          test.S,
			}

			tx := types.NewTx(original)

			txData, err := GetTxData(tx)
			require.NoError(t, err)

			restored := txData.(*types.BlobTx)
			require.Equal(t, original, restored)
		})

		t.Run("SetCodeTx", func(t *testing.T) {
			if test.To == nil {
				t.Skip("SetCodeTx requires a non-nil To address")
			}
			original := &types.SetCodeTx{
				ChainID:    test.ChainId,
				Nonce:      test.Nonce,
				GasFeeCap:  test.GasFeeCap,
				GasTipCap:  test.GasTipCap,
				Gas:        test.Gas,
				To:         *test.To,
				Value:      test.Value,
				Data:       test.Data,
				AccessList: test.AccessList,
				AuthList:   test.AuthList,
				V:          test.V,
				R:          test.R,
				S:          test.S,
			}

			tx := types.NewTx(original)

			txData, err := GetTxData(tx)
			require.NoError(t, err)

			restored := txData.(*types.SetCodeTx)
			require.Equal(t, original, restored)
		})
	}
}

func Test_getTxDataInternal_DetectsUnsupportedTransactionType(t *testing.T) {
	txDataSource := &fakeTxDataSource{
		txType: 123, // an unsupported transaction type
	}

	_, err := getTxDataInternal(txDataSource)
	require.ErrorContains(t, err, "unsupported transaction type: 123")
}

func Test_getTxDataInternal_DetectsNilToAddressForBlobTxType(t *testing.T) {
	txDataSource := &fakeTxDataSource{
		txType: types.BlobTxType,
		to:     nil, // BlobTx requires a non-nil To address
	}

	_, err := getTxDataInternal(txDataSource)
	require.ErrorContains(t, err, "blob transactions must have a recipient")
}

func Test_getTxDataInternal_BlobTxType_DetectsOutOfDomainValues(t *testing.T) {
	values := map[string]struct {
		value *big.Int
		valid bool
	}{
		"nil":       {value: nil, valid: true},
		"zero":      {value: big.NewInt(0), valid: true},
		"positive":  {value: big.NewInt(123456789), valid: true},
		"negative":  {value: big.NewInt(-1), valid: false},
		"too large": {value: new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil), valid: false},
	}

	fields := map[string]func(*fakeTxDataSource, *big.Int){
		"ChainId":    func(f *fakeTxDataSource, v *big.Int) { f.chainId = v },
		"GasTipCap":  func(f *fakeTxDataSource, v *big.Int) { f.gasTipCap = v },
		"GasFeeCap":  func(f *fakeTxDataSource, v *big.Int) { f.gasFeeCap = v },
		"Value":      func(f *fakeTxDataSource, v *big.Int) { f.value = v },
		"BlobFeeCap": func(f *fakeTxDataSource, v *big.Int) { f.blobGasFeeCap = v },
		"V":          func(f *fakeTxDataSource, v *big.Int) { f.v = v },
		"R":          func(f *fakeTxDataSource, v *big.Int) { f.r = v },
		"S":          func(f *fakeTxDataSource, v *big.Int) { f.s = v },
	}

	for fieldName, setField := range fields {
		for valueName, value := range values {
			t.Run(fmt.Sprintf("%s=%s", fieldName, valueName), func(t *testing.T) {
				txDataSource := &fakeTxDataSource{
					txType: types.BlobTxType,
					to:     &common.Address{0x01},
				}
				setField(txDataSource, value.value)

				_, err := getTxDataInternal(txDataSource)
				if value.valid {
					require.NoError(t, err)
				} else {
					require.ErrorContains(t, err, fmt.Sprintf("out of uint256 domain: %v", value.value))
				}
			})
		}
	}
}

func Test_getTxDataInternal_DetectsNilToAddressForSetCodeTxType(t *testing.T) {
	txDataSource := &fakeTxDataSource{
		txType: types.SetCodeTxType,
		to:     nil, // SetCodeTx requires a non-nil To address
	}

	_, err := getTxDataInternal(txDataSource)
	require.ErrorContains(t, err, "set code transactions must have a recipient")
}

func Test_getTxDataInternal_SetCodeTxType_DetectsOutOfDomainValues(t *testing.T) {
	values := map[string]struct {
		value *big.Int
		valid bool
	}{
		"nil":       {value: nil, valid: true},
		"zero":      {value: big.NewInt(0), valid: true},
		"positive":  {value: big.NewInt(123456789), valid: true},
		"negative":  {value: big.NewInt(-1), valid: false},
		"too large": {value: new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil), valid: false},
	}

	fields := map[string]func(*fakeTxDataSource, *big.Int){
		"ChainId":   func(f *fakeTxDataSource, v *big.Int) { f.chainId = v },
		"GasTipCap": func(f *fakeTxDataSource, v *big.Int) { f.gasTipCap = v },
		"GasFeeCap": func(f *fakeTxDataSource, v *big.Int) { f.gasFeeCap = v },
		"Value":     func(f *fakeTxDataSource, v *big.Int) { f.value = v },
		"V":         func(f *fakeTxDataSource, v *big.Int) { f.v = v },
		"R":         func(f *fakeTxDataSource, v *big.Int) { f.r = v },
		"S":         func(f *fakeTxDataSource, v *big.Int) { f.s = v },
	}

	for fieldName, setField := range fields {
		for valueName, value := range values {
			t.Run(fmt.Sprintf("%s=%s", fieldName, valueName), func(t *testing.T) {
				txDataSource := &fakeTxDataSource{
					txType: types.SetCodeTxType,
					to:     &common.Address{0x01},
				}
				setField(txDataSource, value.value)

				_, err := getTxDataInternal(txDataSource)
				if value.valid {
					require.NoError(t, err)
				} else {
					require.ErrorContains(t, err, fmt.Sprintf("out of uint256 domain: %v", value.value))
				}
			})
		}
	}
}

func Test_toUint256_ValidInputs_ProducesSameValueResult(t *testing.T) {
	tests := map[string]struct {
		input    *big.Int
		expected *uint256.Int
	}{
		"nil input": {
			input:    nil,
			expected: nil,
		},
		"zero input": {
			input:    big.NewInt(0),
			expected: uint256.NewInt(0),
		},
		"positive input": {
			input:    big.NewInt(123456789),
			expected: uint256.NewInt(123456789),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := toUint256(test.input)
			require.NoError(t, err)
			require.Equal(t, test.expected, result)
			if test.input != nil {
				test.input.SetInt64(0)                  // mutate input to check for copying
				require.Equal(t, test.expected, result) // result should not change
			}
		})
	}
}

func Test_toUint256_InvalidInputs_ReportsError(t *testing.T) {
	tests := map[string]*big.Int{
		"-1":     new(big.Int).Sub(big.NewInt(0), big.NewInt(1)),
		"-20":    new(big.Int).Sub(big.NewInt(0), big.NewInt(20)),
		"-2^256": new(big.Int).Sub(big.NewInt(0), new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)),
		"2^256":  new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil),
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := toUint256(input)
			require.ErrorContains(t, err, fmt.Sprintf("out of uint256 domain: %v", input))
		})
	}
}

type fakeTxDataSource struct {
	chainId       *big.Int
	nonce         uint64
	gasPrice      *big.Int
	gasFeeCap     *big.Int
	gasTipCap     *big.Int
	gas           uint64
	to            *common.Address
	value         *big.Int
	data          []byte
	accessList    types.AccessList
	blobGasFeeCap *big.Int
	blobHashes    []common.Hash
	authList      []types.SetCodeAuthorization
	v             *big.Int
	r             *big.Int
	s             *big.Int
	txType        uint8
}

func (f *fakeTxDataSource) ChainId() *big.Int {
	return f.chainId
}

func (f *fakeTxDataSource) Nonce() uint64 {
	return f.nonce
}

func (f *fakeTxDataSource) GasPrice() *big.Int {
	return f.gasPrice
}

func (f *fakeTxDataSource) GasFeeCap() *big.Int {
	return f.gasFeeCap
}

func (f *fakeTxDataSource) GasTipCap() *big.Int {
	return f.gasTipCap
}

func (f *fakeTxDataSource) Gas() uint64 {
	return f.gas
}

func (f *fakeTxDataSource) To() *common.Address {
	return f.to
}

func (f *fakeTxDataSource) Value() *big.Int {
	return f.value
}

func (f *fakeTxDataSource) Data() []byte {
	return f.data
}

func (f *fakeTxDataSource) AccessList() types.AccessList {
	return f.accessList
}

func (f *fakeTxDataSource) BlobGasFeeCap() *big.Int {
	return f.blobGasFeeCap
}

func (f *fakeTxDataSource) BlobHashes() []common.Hash {
	return f.blobHashes
}

func (f *fakeTxDataSource) SetCodeAuthorizations() []types.SetCodeAuthorization {
	return f.authList
}

func (f *fakeTxDataSource) Type() uint8 {
	return f.txType
}

func (f *fakeTxDataSource) RawSignatureValues() (v, r, s *big.Int) {
	return f.v, f.r, f.s
}
