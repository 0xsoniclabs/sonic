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

			txData := GetTxData(tx)

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

			txData := GetTxData(tx)

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

			txData := GetTxData(tx)

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

			txData := GetTxData(tx)

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

			txData := GetTxData(tx)

			restored := txData.(*types.SetCodeTx)
			require.Equal(t, original, restored)
		})
	}
}

func Test_mustToUint256_ValidInputs_ProducesSameValueResult(t *testing.T) {
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
			result := mustToUint256(test.input)
			require.Equal(t, test.expected, result)
			if test.input != nil {
				test.input.SetInt64(0)                  // mutate input to check for copying
				require.Equal(t, test.expected, result) // result should not change
			}
		})
	}
}

func Test_mustToUint256_InvalidInputs_Panics(t *testing.T) {
	tests := map[string]*big.Int{
		"-1":     new(big.Int).Sub(big.NewInt(0), big.NewInt(1)),
		"-20":    new(big.Int).Sub(big.NewInt(0), big.NewInt(20)),
		"-2^256": new(big.Int).Sub(big.NewInt(0), new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)),
		"2^256":  new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil),
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			require.PanicsWithValue(t,
				fmt.Sprintf("out of uint256 domain: %v", input), func() {
					mustToUint256(input)
				},
			)
		})
	}
}
