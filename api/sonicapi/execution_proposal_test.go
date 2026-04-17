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

package sonicapi

import (
	"encoding/json"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func Test_ExecutionProposal_canBeConstructedFromBuilderBundle(t *testing.T) {

	signer := types.LatestSignerForChainID(big.NewInt(2))
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	tests := map[string]struct {
		bundle bundle.TransactionBundle
		json   string
	}{
		"empty bundle": {
			bundle: bundle.NewBuilder().WithSigner(signer).BuildBundle(),
			json: `{
		 		"blockRange":{"earliest":"0x0","latest":"0x3ff"},
		 		"root":{"group":{}}
		 	}`,
		},
		"simple bundle": {
			bundle: bundle.NewBuilder().
				WithSigner(signer).
				With(bundle.Step(key, &types.AccessListTx{})).
				BuildBundle(),
			json: `{
		 		"blockRange":{"earliest":"0x0","latest":"0x3ff"},
		 		"root":{
		 			"single":{
		 				"chainId": "0x2",
		 				"from": "REPLACE_ADDRESS",
		 				"gas": "0x10cc"
		 			}
		 		}
		 	}`,
		},
		"bundle with two transactions": {
			bundle: bundle.NewBuilder().
				WithSigner(signer).
				With(
					bundle.AllOf(
						bundle.Step(key, &types.AccessListTx{}),
						bundle.Step(key, &types.AccessListTx{}),
					),
				).
				BuildBundle(),
			json: `{
				"blockRange":{"earliest":"0x0","latest":"0x3ff"},
				"root":{
					"group":{
						"steps":[
							{
								"single":{
									"chainId": "0x2",
									"from": "REPLACE_ADDRESS",
									"gas": "0x10cc"
								}
							},
							{
								"single":{
									"chainId": "0x2",
									"from": "REPLACE_ADDRESS",
									"gas": "0x10cc"
								}
							}
						]
					}
				}
			}`,
		},
		"nested bundle": {
			bundle: bundle.NewBuilder().
				WithSigner(signer).
				With(
					bundle.OneOf(
						bundle.AllOf(
							bundle.Step(key, &types.AccessListTx{}),
						),
					),
				).
				BuildBundle(),
			json: `{
				"blockRange":{"earliest":"0x0","latest":"0x3ff"},
				"root":{
					"group":{
						"oneOf": true,
						"steps":[
							{
								"group":{
									"steps":[
										{
											"single":{
												"chainId": "0x2",
												"from": "REPLACE_ADDRESS",
												"gas": "0x10cc"
											}
										}
									]
								}
							}
						]
					}
				}
			}`,
		},
		"bundle with flags in transactions": {
			bundle: bundle.NewBuilder().
				WithSigner(signer).
				With(
					bundle.AllOf(
						bundle.Step(key, &types.AccessListTx{}).
							WithFlags(bundle.EF_TolerateFailed),
						bundle.Step(key, &types.AccessListTx{}).
							WithFlags(bundle.EF_TolerateInvalid),
						bundle.Step(key, &types.AccessListTx{}).
							WithFlags(bundle.EF_TolerateFailed|bundle.EF_TolerateInvalid),
					),
				).
				BuildBundle(),
			json: `{
				"blockRange":{"earliest":"0x0","latest":"0x3ff"},
				"root":{
					"group":{
						"steps":[
							{
								"single":{
									"chainId": "0x2",
									"from": "REPLACE_ADDRESS",
									"gas": "0x10cc",
									"tolerateFailed": true
								}
							},
							{
								"single":{
									"chainId": "0x2",
									"from": "REPLACE_ADDRESS",
									"gas": "0x10cc",
									"tolerateInvalid": true
								}
							},
							{
								"single":{
									"chainId": "0x2",
									"from": "REPLACE_ADDRESS",
									"gas": "0x10cc",
									"tolerateFailed": true,
									"tolerateInvalid": true
								}
							}
						]
					}
				}
			}`,
		},
		"bundle with flags in groups": {
			bundle: bundle.NewBuilder().
				WithSigner(signer).
				With(
					bundle.AllOf(
						bundle.OneOf(
							bundle.Step(key, &types.AccessListTx{}),
						),
						bundle.OneOf(
							bundle.Step(key, &types.AccessListTx{}),
						).WithFlags(bundle.EF_TolerateFailed),
						bundle.AllOf(
							bundle.Step(key, &types.AccessListTx{}),
						),
						bundle.AllOf(
							bundle.Step(key, &types.AccessListTx{}),
						).WithFlags(bundle.EF_TolerateFailed),
					),
				).
				BuildBundle(),
			json: `{
				"blockRange":{"earliest":"0x0","latest":"0x3ff"},
				"root":{
					"group":{
						"steps":[
							{
								"group":{
									"oneOf": true,
									"steps":[
										{
											"single":{
												"chainId": "0x2",
												"from": "REPLACE_ADDRESS",
												"gas": "0x10cc"
											}
										}
									]
								}
							},
							{
								"group":{
									"oneOf": true,
									"tolerateFailures": true,
									"steps":[
										{
											"single":{
												"chainId": "0x2",
												"from": "REPLACE_ADDRESS",
												"gas": "0x10cc"
											}
										}
									]
								}
							},
							{
								"group":{
									"steps":[
										{
											"single":{
												"chainId": "0x2",
												"from": "REPLACE_ADDRESS",
												"gas": "0x10cc"
											}
										}
									]
								}
							},
							{
								"group":{
									"tolerateFailures": true,
									"steps":[
										{
											"single":{
												"chainId": "0x2",
												"from": "REPLACE_ADDRESS",
												"gas": "0x10cc"
											}
										}
									]
								}
							}
						]
					}
				}
			}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {

			// check that signatures and keys are not misplaced
			for _, tx := range tt.bundle.Transactions {
				sender, err := signer.Sender(tx)
				require.NoError(t, err)
				require.Equal(t, crypto.PubkeyToAddress(key.PublicKey), sender)
			}

			proposal, err := createProposalRequestFromBundle(signer, &tt.bundle)
			require.NoError(t, err)
			require.NotNil(t, proposal)

			json := strings.ReplaceAll(tt.json, "REPLACE_ADDRESS", crypto.PubkeyToAddress(key.PublicKey).Hex())

			expectJsonEqual(t, json, proposal)
		})
	}
}

func TestConvertToTransactionArgs(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	address := common.Address{1}

	tests := map[string]struct {
		tx   *types.Transaction
		json string
	}{
		// empty transactions
		"empty legacy tx": {
			tx: types.NewTx(&types.LegacyTx{}),
			json: `{
				"chainId": "0x1",
				"from": "%s"
			}`,
		},
		"empty access list tx": {
			tx: types.NewTx(&types.AccessListTx{}),
			json: `{
				"chainId": "0x1",
				"from": "%s"
			}`,
		},
		"empty dynamic fee tx": {
			tx: types.NewTx(&types.DynamicFeeTx{}),
			json: `{
				"chainId": "0x1",
				"from": "%s"
			}`,
		},
		"empty blob tx": {
			tx: types.NewTx(&types.BlobTx{}),
			json: `{
				"chainId": "0x1",
				"from": "%s",
				"to": "0x0000000000000000000000000000000000000000"
			}`,
		},
		"empty set code tx": {
			tx: types.NewTx(&types.SetCodeTx{}),
			json: `{
				"chainId": "0x1",
				"from": "%s",
				"to": "0x0000000000000000000000000000000000000000"
			}`,
		},
		// trivial transactions
		"trivial legacy tx": {
			tx: types.NewTx(&types.LegacyTx{
				To:    &address,
				Nonce: 10,
				Value: big.NewInt(12123),
				Data:  []byte{0xAB, 0xCD},
			}),
			json: `{
				"chainId": "0x1",
				"from": "%s",
				"to": "0x0100000000000000000000000000000000000000",
				"nonce": "0xa",
				"value": "0x2f5b",
				"data": "0xabcd"
			}`,
		},
		"trivial access list tx": {
			tx: types.NewTx(&types.AccessListTx{
				To:    &address,
				Nonce: 10,
				Value: big.NewInt(123),
				Data:  []byte{0xAB, 0xCD},
			}),
			json: `{
				"chainId": "0x1",
				"from": "%s",
				"to": "0x0100000000000000000000000000000000000000",
				"nonce": "0xa",
				"value": "0x7b",
				"data": "0xabcd"
			}`,
		},
		"trivial dynamic fee tx": {
			tx: types.NewTx(&types.DynamicFeeTx{
				To:    &address,
				Nonce: 10,
				Value: big.NewInt(123),
				Data:  []byte{0xAB, 0xCD},
			}),
			json: `{
				"chainId": "0x1",
				"from": "%s",
				"to": "0x0100000000000000000000000000000000000000",
				"nonce": "0xa",
				"value": "0x7b",
				"data": "0xabcd"
			}`,
		},
		"trivial blob tx": {
			tx: types.NewTx(&types.BlobTx{
				To:    address,
				Nonce: 10,
				Value: uint256.NewInt(123),
				Data:  []byte{0xAB, 0xCD},
			}),
			json: `{
				"chainId": "0x1",
				"from": "%s",
				"to": "0x0100000000000000000000000000000000000000",
				"nonce": "0xa",
				"value": "0x7b",
				"data": "0xabcd"
			}`,
		},
		"trivial set code tx": {
			tx: types.NewTx(&types.SetCodeTx{
				To:    address,
				Nonce: 10,
				Value: uint256.NewInt(123),
				Data:  []byte{0xAB, 0xCD},
			}),
			json: `{
				"chainId": "0x1",
				"from": "%s",
				"to": "0x0100000000000000000000000000000000000000",
				"nonce": "0xa",
				"value": "0x7b",
				"data": "0xabcd"
			}`,
		},
		// Data vs Input semantics
		"contract create": {
			tx: types.NewTx(&types.LegacyTx{
				Data: slices.Repeat([]byte{0xAB}, 4),
			}),
			json: `{
				"chainId": "0x1",
				"from": "%s",
				"input": "0xabababab"
			}`,
		},
		"no create": {
			tx: types.NewTx(&types.LegacyTx{
				To:   &address,
				Data: slices.Repeat([]byte{0xAB}, 4),
			}),
			json: `{
				"chainId": "0x1",
				"from": "%s",
				"to": "0x0100000000000000000000000000000000000000",
				"data": "0xabababab"
			}`,
		},
		// GasLimit
		"With Gas limit": {
			tx: types.NewTx(&types.DynamicFeeTx{
				Gas: 21000,
			}),
			json: `{
				"chainId": "0x1",
				"from": "%s",
				"gas": "0x5208"
			}`,
		},
		// Access list semantics
		"Access list with entries": {
			tx: types.NewTx(&types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address:     address,
						StorageKeys: []common.Hash{{1}},
					},
				},
			}),
			json: `{
				"chainId": "0x1",
				"from": "%s",
				"accessList": [
					{
						"address": "0x0100000000000000000000000000000000000000",
						"storageKeys": ["0x0100000000000000000000000000000000000000000000000000000000000000"]
					}
				]
			}`,
		},
		// Gas price semantics
		"Legacy tx with gas price": {
			tx: types.NewTx(&types.LegacyTx{
				GasPrice: big.NewInt(100_000_000_000),
			}),
			json: `{
				"chainId": "0x1",
				"from": "%s",
				"gasPrice": "0x174876e800"
			}`,
		},
		"Access list tx with gas price": {
			tx: types.NewTx(&types.AccessListTx{
				GasPrice: big.NewInt(100_000_000_000),
			}),
			json: `{
				"chainId": "0x1",
				"from": "%s",
				"gasPrice": "0x174876e800"
			}`,
		},
		"Dynamic fee tx with max fee per gas and max priority fee per gas": {
			tx: types.NewTx(&types.DynamicFeeTx{
				GasFeeCap: big.NewInt(100_000_000_000),
				GasTipCap: big.NewInt(2_000_000_000),
			}),
			json: `{
				"chainId": "0x1",
				"from": "%s",
				"maxFeePerGas": "0x174876e800",
				"maxPriorityFeePerGas": "0x77359400"
			}`,
		},
		"Blob tx with max fee per gas and max priority fee per gas": {
			tx: types.NewTx(&types.BlobTx{
				GasFeeCap: uint256.NewInt(100_000_000_000),
				GasTipCap: uint256.NewInt(2_000_000_000),
			}),
			json: `{
				"chainId": "0x1",
				"from": "%s",
				"to": "0x0000000000000000000000000000000000000000",
				"maxFeePerGas": "0x174876e800",
				"maxPriorityFeePerGas": "0x77359400"
			}`,
		},
		"Set code tx with max fee per gas and max priority fee per gas": {
			tx: types.NewTx(&types.SetCodeTx{
				GasFeeCap: uint256.NewInt(100_000_000_000),
				GasTipCap: uint256.NewInt(2_000_000_000),
			}),
			json: `{
				"chainId": "0x1",
				"from": "%s",
				"to": "0x0000000000000000000000000000000000000000",
				"maxFeePerGas": "0x174876e800",
				"maxPriorityFeePerGas": "0x77359400"
			}`,
		},
		// Set code tx autorizations
		"set code tx with authorization": {
			tx: types.NewTx(&types.SetCodeTx{
				To: address,
				AuthList: []types.SetCodeAuthorization{
					{},
				},
			}),
			json: `{
				"chainId": "0x1",
				"from": "%s",
				"to": "0x0100000000000000000000000000000000000000",
				"authorizationList": [
					{
						"chainId": "0x0",
						"address": "0x0000000000000000000000000000000000000000",
						"nonce": "0x0",
						"yParity": "0x0",
						"r": "0x0",
						"s": "0x0"
					}
				]
			}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tx, err := types.SignTx(tt.tx, signer, key)
			require.NoError(t, err)

			args, err := convertToTransactonArgs(signer, tx)
			require.NoError(t, err)

			_, err = json.Marshal(args)
			require.NoError(t, err)

			json := fmt.Sprintf(tt.json, crypto.PubkeyToAddress(key.PublicKey).Hex())
			expectJsonEqual(t, json, args)
		})
	}
}
