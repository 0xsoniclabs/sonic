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

	"github.com/0xsoniclabs/sonic/api/ethapi"
	rpctest "github.com/0xsoniclabs/sonic/api/rpc_test"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func Test_ExecutionProposal_canBeConstructedFromBuilderBundle(t *testing.T) {

	signer := types.LatestSignerForChainID(big.NewInt(2))
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	// txStep generates a JSON object for a standard AccessListTx step
	// with optional flag prefix (e.g. `"tolerateFailed": true`).
	txStep := func(flags string) string {
		prefix := ""
		if flags != "" {
			prefix = flags + ","
		}
		return fmt.Sprintf(`{
			%s
			"from": "REPLACE_ADDRESS",
			"to": null,
			"gas": "0x10cc",
			"gasPrice": null,
			"maxFeePerGas": null,
			"maxPriorityFeePerGas": null,
			"value": null,
			"nonce": null,
			"data": null,
			"input": null,
			"chainId": "0x2",
			"maxFeePerBlobGas": null,
			"blobs": null,
			"commitments": null,
			"proofs": null,
			"authorizationList": null
		}`, prefix)
	}

	s := txStep("")
	sTF := txStep(`"tolerateFailed": true`)
	sTI := txStep(`"tolerateInvalid": true`)
	sTFI := txStep(`"tolerateFailed": true, "tolerateInvalid": true`)

	tests := map[string]struct {
		bundle bundle.TransactionBundle
		json   string
	}{
		"simple bundle": {
			bundle: bundle.NewBuilder().
				WithSigner(signer).
				With(bundle.Step(key, &types.AccessListTx{})).
				BuildBundle(),
			json: fmt.Sprintf(`{
				"blockRange":{"first":"0x0","length":"0x400"},
				"steps":[%s]
			}`, s),
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
			json: fmt.Sprintf(`{
				"blockRange":{"first":"0x0","length":"0x400"},
				"steps":[{"steps":[%s,%s]}]
			}`, s, s),
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
			json: fmt.Sprintf(`{
				"blockRange":{"first":"0x0","length":"0x400"},
				"steps":[{"oneOf":true,"steps":[{"steps":[%s]}]}]
			}`, s),
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
			json: fmt.Sprintf(`{
				"blockRange":{"first":"0x0","length":"0x400"},
				"steps":[{"steps":[%s,%s,%s]}]
			}`, sTF, sTI, sTFI),
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
			json: fmt.Sprintf(`{
				"blockRange":{"first":"0x0","length":"0x400"},
				"steps":[{"steps":[
					{"oneOf":true,"steps":[%s]},
					{"tolerateFailures":true,"oneOf":true,"steps":[%s]},
					{"steps":[%s]},
					{"tolerateFailures":true,"steps":[%s]}
				]}]
			}`, s, s, s, s),
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

			json := strings.ReplaceAll(tt.json, "REPLACE_ADDRESS", strings.ToLower(crypto.PubkeyToAddress(key.PublicKey).Hex()))

			expectJsonEqual(t, json, proposal)

			var deserialized RPCExecutionProposal
			expectCanBeDeserialized(t, &deserialized, json)

			require.Equal(t, *proposal, deserialized)
		})
	}
}

func TestConvertToTransactionArgs_convertsTxsToTransactionArgs(t *testing.T) {
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
		// Set code tx authorization
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

			args, err := convertToTransactionArgs(signer, tx)
			require.NoError(t, err)

			_, err = json.Marshal(args)
			require.NoError(t, err)

			json := fmt.Sprintf(tt.json, crypto.PubkeyToAddress(key.PublicKey).Hex())
			expectJsonEqual(t, json, args)
		})
	}
}

func TestConvertToTransactionArgs_returnsErrors(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))
	otherChainSigner := types.LatestSignerForChainID(big.NewInt(2))

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	tests := map[string]struct {
		tx *types.Transaction
	}{
		"invalid signature": {
			tx: types.MustSignNewTx(key, otherChainSigner, &types.LegacyTx{}),
		},
		"blob tx with invalid blob hash": {
			tx: types.MustSignNewTx(key, signer, &types.BlobTx{
				BlobHashes: []common.Hash{{1}},
			}),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {

			_, err = convertToTransactionArgs(signer, tt.tx)
			require.Error(t, err)
		})
	}
}

func TestCreateProposalRequestFromBundle(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	// Build a simple bundle with one transaction
	tx := types.MustSignNewTx(key, signer, &types.AccessListTx{
		Nonce:   1,
		Value:   big.NewInt(100),
		Gas:     21000,
		ChainID: big.NewInt(1),
	})
	bndl := bundle.NewBuilder().
		WithSigner(signer).
		With(bundle.Step(key, tx)).
		BuildBundle()

	proposal, err := createProposalRequestFromBundle(signer, &bndl)
	require.NoError(t, err)
	require.NotNil(t, proposal)

	// Check that the proposal contains the expected block range and steps
	require.EqualValues(t, *rpctest.ToHexUint64(0), proposal.BlockRange.First)
	require.EqualValues(t, *rpctest.ToHexUint64(1024), proposal.BlockRange.Length)
	require.Len(t, proposal.Steps, 1)

	// Nested bundle (AllOf with two steps)
	tx2 := types.MustSignNewTx(key, signer, &types.AccessListTx{
		Nonce:   2,
		Value:   big.NewInt(200),
		Gas:     22000,
		ChainID: big.NewInt(1),
	})
	nestedBndl := bundle.NewBuilder().
		WithSigner(signer).
		With(bundle.AllOf(
			bundle.Step(key, tx),
			bundle.Step(key, tx2),
		)).
		BuildBundle()

	proposal2, err := createProposalRequestFromBundle(signer, &nestedBndl)
	require.NoError(t, err)
	require.NotNil(t, proposal2)
	require.Len(t, proposal2.Steps, 1)
}

func TestCreateProposalRequestFromBundle_CanYieldErrors(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))

	tests := map[string]struct {
		bundle bundle.TransactionBundle
	}{
		"plan references missing transaction": {
			bundle: bundle.TransactionBundle{
				Plan: bundle.ExecutionPlan{
					Root: bundle.NewTxStep(bundle.TxReference{
						Hash: common.Hash{123},
					}),
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := createProposalRequestFromBundle(signer, &tt.bundle)
			require.Error(t, err)
		})
	}
}

func Test_convertVisitorLeafIntoRPCExecutionPlanProposalLeaf_ConvertsToProposalLeaf(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	sender := crypto.PubkeyToAddress(key.PublicKey)

	tx1 := types.MustSignNewTx(key, signer, &types.AccessListTx{
		Nonce: 1,
		Value: big.NewInt(100),
	})

	tests := map[string]struct {
		txBundle bundle.TransactionBundle
		expected []RPCExecutionStepProposal
	}{
		"simple bundle": {
			txBundle: bundle.NewBuilder().
				WithSigner(signer).
				With(bundle.Step(key, tx1)).
				BuildBundle(),
			expected: []RPCExecutionStepProposal{{
				TransactionArgs: ethapi.TransactionArgs{
					ChainID: rpctest.ToHexBigInt(big.NewInt(1)),
					From:    &sender,
					Nonce:   rpctest.ToHexUint64(1),
					Value:   rpctest.ToHexBigInt(big.NewInt(100)),
					Gas:     rpctest.ToHexUint64(4300),
				},
			}},
		},
		"bundle with transaction including access list": {
			txBundle: bundle.NewBuilder().
				WithSigner(signer).
				With(bundle.Step(key, types.MustSignNewTx(key, signer, &types.AccessListTx{
					Nonce: 1,
					Value: big.NewInt(100),
					AccessList: types.AccessList{
						{
							Address:     common.Address{1},
							StorageKeys: []common.Hash{{1}},
						},
					},
				}))).
				BuildBundle(),
			expected: []RPCExecutionStepProposal{{
				TransactionArgs: ethapi.TransactionArgs{
					ChainID: rpctest.ToHexBigInt(big.NewInt(1)),
					From:    &sender,
					Nonce:   rpctest.ToHexUint64(1),
					Value:   rpctest.ToHexBigInt(big.NewInt(100)),
					Gas:     rpctest.ToHexUint64(4300),
					AccessList: &types.AccessList{
						{
							Address:     common.Address{1},
							StorageKeys: []common.Hash{{1}},
						},
					},
				},
			}},
		},
		"bundle with transaction including access list with marker": {
			txBundle: bundle.NewBuilder().
				WithSigner(signer).
				With(bundle.Step(key, types.MustSignNewTx(key, signer, &types.AccessListTx{
					Nonce: 1,
					Value: big.NewInt(100),
					AccessList: types.AccessList{
						{
							Address: bundle.BundleOnly,
						},
					},
				}))).
				BuildBundle(),
			expected: []RPCExecutionStepProposal{{
				TransactionArgs: ethapi.TransactionArgs{
					ChainID: rpctest.ToHexBigInt(big.NewInt(1)),
					From:    &sender,
					Nonce:   rpctest.ToHexUint64(1),
					Value:   rpctest.ToHexBigInt(big.NewInt(100)),
					Gas:     rpctest.ToHexUint64(4300),
				},
			}},
		},
		"bundle with transaction including access list with marker and other access list entries": {
			txBundle: bundle.NewBuilder().
				WithSigner(signer).
				With(bundle.Step(key, types.MustSignNewTx(key, signer, &types.AccessListTx{
					Nonce: 1,
					Value: big.NewInt(100),
					AccessList: types.AccessList{
						{
							Address: bundle.BundleOnly,
						},
						{
							Address:     common.Address{1},
							StorageKeys: []common.Hash{{1}},
						},
					},
				}))).
				BuildBundle(),
			expected: []RPCExecutionStepProposal{{
				TransactionArgs: ethapi.TransactionArgs{
					ChainID: rpctest.ToHexBigInt(big.NewInt(1)),
					From:    &sender,
					Nonce:   rpctest.ToHexUint64(1),
					Value:   rpctest.ToHexBigInt(big.NewInt(100)),
					Gas:     rpctest.ToHexUint64(4300),
					AccessList: &types.AccessList{
						{
							Address:     common.Address{1},
							StorageKeys: []common.Hash{{1}},
						},
					},
				},
			}},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {

			for i, ref := range tt.txBundle.Plan.Root.GetTransactionReferencesInReferencedOrder() {

				result, err := convertVisitorLeafIntoRPCExecutionPlanProposalLeaf(
					signer,
					&tt.txBundle,
					0,
					ref,
				)
				require.NoError(t, err)
				require.Equal(t, tt.expected[i], result)

			}
		})
	}
}

func Test_convertVisitorLeafIntoRPCExecutionPlanProposalLeaf_canReturnErrors(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	txBundle := bundle.NewBuilder().
		WithSigner(signer).
		//  Blob tx with hashes is not supported
		With(bundle.Step(key, &types.BlobTx{BlobHashes: []common.Hash{{}}})).
		BuildBundle()

	tests := map[string]struct {
		txBundle      bundle.TransactionBundle
		txRef         bundle.TxReference
		expectedError string
	}{
		"transaction reference not found in bundle transactions returns error": {
			txBundle:      txBundle,
			txRef:         bundle.TxReference{Hash: common.Hash{123}},
			expectedError: "transaction reference not found in bundle transactions",
		},
		"bundle with non-convertible transaction type returns error": {
			txBundle:      txBundle,
			txRef:         txBundle.Plan.Root.GetTransactionReferencesInReferencedOrder()[0],
			expectedError: "blob transactions are not supported in execution proposals",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := convertVisitorLeafIntoRPCExecutionPlanProposalLeaf(signer, &tt.txBundle, 0, tt.txRef)
			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

func Test_transform(t *testing.T) {

	proposal := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				RPCExecutionStepProposal{},
				RPCExecutionPlanGroup{
					Steps: []any{
						RPCExecutionStepProposal{},
					},
				},
				RPCExecutionStepProposal{},
			},
		},
	}

	newProposal, err := transform(proposal,
		func(step RPCExecutionStepProposal) (RPCExecutionStepProposal, error) {
			step.TolerateFailed = true
			return step, nil
		})
	require.NoError(t, err)

	expected := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				RPCExecutionStepProposal{TolerateFailed: true},
				RPCExecutionPlanGroup{
					Steps: []any{
						RPCExecutionStepProposal{TolerateFailed: true},
					},
				},
				RPCExecutionStepProposal{TolerateFailed: true},
			},
		},
	}

	require.Equal(t, expected, newProposal)
}

func Test_convertProposalToPlan(t *testing.T) {

	signer := types.LatestSignerForChainID(big.NewInt(1))

	key1, err := crypto.GenerateKey()
	require.NoError(t, err)
	address1 := crypto.PubkeyToAddress(key1.PublicKey)
	key2, err := crypto.GenerateKey()
	require.NoError(t, err)
	address2 := crypto.PubkeyToAddress(key2.PublicKey)

	const bundleMarkerCost = params.TxAccessListAddressGas + params.TxAccessListStorageKeyGas

	tests := map[string]struct {
		proposal RPCExecutionProposal
		plan     bundle.ExecutionPlan
	}{
		"simple proposal with one step": {
			proposal: RPCExecutionProposal{
				BlockRange: &RPCRange{
					First:  *rpctest.ToHexUint64(0),
					Length: *rpctest.ToHexUint64(1023),
				},
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						RPCExecutionStepProposal{
							TransactionArgs: ethapi.TransactionArgs{
								From:  &address1,
								To:    &common.Address{123},
								Nonce: rpctest.ToHexUint64(1),
								Gas:   rpctest.ToHexUint64(21000 + bundleMarkerCost),
							},
						},
					},
				},
			},
			plan: bundle.NewBuilder().
				SetEarliest(0).SetRangeLength(1023).
				WithSigner(signer).
				With(
					bundle.Step(key1, &types.AccessListTx{
						To:    &common.Address{123},
						Nonce: 1,
						Gas:   21000,
					}),
				).
				BuildBundle().Plan,
		},
		"two steps in one group": {
			proposal: RPCExecutionProposal{
				BlockRange: &RPCRange{
					First:  *rpctest.ToHexUint64(0),
					Length: *rpctest.ToHexUint64(1023),
				},
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						RPCExecutionStepProposal{
							TransactionArgs: ethapi.TransactionArgs{
								From:  &address1,
								To:    &common.Address{123},
								Nonce: rpctest.ToHexUint64(1),
								Gas:   rpctest.ToHexUint64(21000 + bundleMarkerCost),
							},
						},
						RPCExecutionStepProposal{
							TransactionArgs: ethapi.TransactionArgs{
								From:  &address2,
								To:    &common.Address{1},
								Nonce: rpctest.ToHexUint64(2),
								Gas:   rpctest.ToHexUint64(21000 + bundleMarkerCost),
							},
						},
					},
				},
			},
			plan: bundle.NewBuilder().
				SetEarliest(0).SetRangeLength(1023).
				WithSigner(signer).
				AllOf(
					bundle.Step(key1, &types.AccessListTx{
						To:    &common.Address{123},
						Nonce: 1,
						Gas:   21000,
					}),
					bundle.Step(key2, &types.AccessListTx{
						To:    &common.Address{1},
						Nonce: 2,
						Gas:   21000,
					}),
				).
				BuildBundle().Plan,
		},
		"different execution flags in steps": {
			proposal: RPCExecutionProposal{
				BlockRange: &RPCRange{
					First:  *rpctest.ToHexUint64(0),
					Length: *rpctest.ToHexUint64(1023),
				},
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						RPCExecutionStepProposal{
							TransactionArgs: ethapi.TransactionArgs{
								From:  &address1,
								To:    &common.Address{123},
								Nonce: rpctest.ToHexUint64(1),
								Gas: rpctest.ToHexUint64(21000 +
									// NOTE: builder adds marker costs
									params.TxAccessListAddressGas + params.TxAccessListStorageKeyGas),
							},
							TolerateFailed: true,
						},
						RPCExecutionStepProposal{
							TransactionArgs: ethapi.TransactionArgs{
								From:  &address2,
								To:    &common.Address{1},
								Nonce: rpctest.ToHexUint64(2),
								Gas:   rpctest.ToHexUint64(21000 + bundleMarkerCost),
							},
							TolerateInvalid: true,
						},
					},
				},
			},
			plan: bundle.NewBuilder().
				SetEarliest(0).SetRangeLength(1023).
				WithSigner(signer).
				AllOf(
					bundle.Step(key1, &types.AccessListTx{
						To:    &common.Address{123},
						Nonce: 1,
						Gas:   21000,
					}).WithFlags(bundle.EF_TolerateFailed),
					bundle.Step(key2, &types.AccessListTx{
						To:    &common.Address{1},
						Nonce: 2,
						Gas:   21000,
					}).WithFlags(bundle.EF_TolerateInvalid),
				).
				BuildBundle().Plan,
		},
		"OneOf group": {

			proposal: RPCExecutionProposal{
				BlockRange: &RPCRange{
					First:  *rpctest.ToHexUint64(0),
					Length: *rpctest.ToHexUint64(1023),
				},
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						RPCExecutionPlanGroup{
							OneOf: true,
							Steps: []any{
								RPCExecutionStepProposal{
									TransactionArgs: ethapi.TransactionArgs{
										From:  &address1,
										To:    &common.Address{123},
										Nonce: rpctest.ToHexUint64(1),
										Gas:   rpctest.ToHexUint64(21000 + bundleMarkerCost),
									},
								},
								RPCExecutionStepProposal{
									TransactionArgs: ethapi.TransactionArgs{
										From:  &address2,
										To:    &common.Address{1},
										Nonce: rpctest.ToHexUint64(2),
										Gas:   rpctest.ToHexUint64(21000 + bundleMarkerCost),
									},
								},
							},
						},
					},
				},
			},

			plan: bundle.NewBuilder().
				SetEarliest(0).SetRangeLength(1023).
				WithSigner(signer).
				OneOf(
					bundle.Step(key1, &types.AccessListTx{
						To:    &common.Address{123},
						Nonce: 1,
						Gas:   21000,
					}),
					bundle.Step(key2, &types.AccessListTx{
						To:    &common.Address{1},
						Nonce: 2,
						Gas:   21000,
					}),
				).
				BuildBundle().Plan,
		},
		"OneOf group with different execution flags in steps": {
			proposal: RPCExecutionProposal{
				BlockRange: &RPCRange{
					First:  *rpctest.ToHexUint64(0),
					Length: *rpctest.ToHexUint64(1023),
				},
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						RPCExecutionPlanGroup{
							OneOf: true,
							Steps: []any{
								RPCExecutionStepProposal{
									TransactionArgs: ethapi.TransactionArgs{
										From:  &address1,
										To:    &common.Address{123},
										Nonce: rpctest.ToHexUint64(1),
										Gas:   rpctest.ToHexUint64(21000 + bundleMarkerCost),
									},
									TolerateFailed: true,
								},
								RPCExecutionStepProposal{
									TransactionArgs: ethapi.TransactionArgs{
										From:  &address2,
										To:    &common.Address{1},
										Nonce: rpctest.ToHexUint64(2),
										Gas:   rpctest.ToHexUint64(21000 + bundleMarkerCost),
									},
									TolerateInvalid: true,
								},
							},
						},
					},
				},
			},
			plan: bundle.NewBuilder().
				SetEarliest(0).SetRangeLength(1023).
				WithSigner(signer).
				OneOf(
					bundle.Step(key1, &types.AccessListTx{
						To:    &common.Address{123},
						Nonce: 1,
						Gas:   21000,
					}).WithFlags(bundle.EF_TolerateFailed),
					bundle.Step(key2, &types.AccessListTx{
						To:    &common.Address{1},
						Nonce: 2,
						Gas:   21000,
					}).WithFlags(bundle.EF_TolerateInvalid),
				).
				BuildBundle().Plan,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			plan, err := convertProposalToPlan(signer, tt.proposal)
			require.NoError(t, err)
			require.Equal(t, tt.plan, plan)
			require.Equal(t, tt.plan.Hash(), plan.Hash())
		})
	}
}

func Test_convertProposalToPlan_ReturnsErrors(t *testing.T) {

	signer := types.LatestSignerForChainID(big.NewInt(1))
	address := common.Address{1}

	validRange := &RPCRange{
		First:  *rpctest.ToHexUint64(0),
		Length: *rpctest.ToHexUint64(1023),
	}

	validStep := RPCExecutionStepProposal{
		TransactionArgs: ethapi.TransactionArgs{
			From:  &address,
			To:    &common.Address{2},
			Nonce: rpctest.ToHexUint64(1),
			Gas:   rpctest.ToHexUint64(21000),
		},
	}

	// a value that overflows uint256
	overflowingValue := new(big.Int).Lsh(big.NewInt(1), 257)

	tests := map[string]struct {
		proposal    RPCExecutionProposal
		expectedErr string
	}{
		// --- missing fields ---
		"missing block range": {
			proposal: RPCExecutionProposal{
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{validStep},
				},
			},
			expectedErr: "execution proposal must include block range",
		},
		"missing from in step": {
			proposal: RPCExecutionProposal{
				BlockRange: validRange,
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						RPCExecutionStepProposal{
							TransactionArgs: ethapi.TransactionArgs{
								To:    &common.Address{2},
								Nonce: rpctest.ToHexUint64(1),
								Gas:   rpctest.ToHexUint64(21000),
							},
						},
					},
				},
			},
			expectedErr: "transaction in bundle must include from",
		},

		// --- empty / wrong structure ---
		"empty steps": {
			proposal: RPCExecutionProposal{
				BlockRange:            validRange,
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{Steps: []any{}},
			},
			expectedErr: "proposed group must include at least one step",
		},
		"empty nested group": {
			proposal: RPCExecutionProposal{
				BlockRange: validRange,
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						RPCExecutionPlanGroup{Steps: []any{}},
					},
				},
			},
			expectedErr: "proposed group must include at least one step",
		},
		"nil step element in group": {
			proposal: RPCExecutionProposal{
				BlockRange: validRange,
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{nil},
				},
			},
			expectedErr: "invalid execution proposal level",
		},
		"wrong type in steps slice": {
			proposal: RPCExecutionProposal{
				BlockRange: validRange,
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{"not a step"},
				},
			},
			expectedErr: "invalid execution proposal level",
		},

		// --- wrong transactions (overflow in gas fee fields) ---
		"SetCodeTx with overflowing MaxFeePerGas": {
			proposal: RPCExecutionProposal{
				BlockRange: validRange,
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						RPCExecutionStepProposal{
							TransactionArgs: ethapi.TransactionArgs{
								From:              &address,
								To:                &common.Address{2},
								Nonce:             rpctest.ToHexUint64(1),
								Gas:               rpctest.ToHexUint64(21000),
								MaxFeePerGas:      (*hexutil.Big)(overflowingValue),
								AuthorizationList: []types.SetCodeAuthorization{{}},
							},
						},
					},
				},
			},
			expectedErr: "invalid MaxFeePerGas",
		},
		"SetCodeTx with negative MaxPriorityFeePerGas": {
			proposal: RPCExecutionProposal{
				BlockRange: validRange,
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						RPCExecutionStepProposal{
							TransactionArgs: ethapi.TransactionArgs{
								From:                 &address,
								To:                   &common.Address{2},
								Nonce:                rpctest.ToHexUint64(1),
								Gas:                  rpctest.ToHexUint64(21000),
								MaxFeePerGas:         (*hexutil.Big)(big.NewInt(1)),
								MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(-1)),
								AuthorizationList:    []types.SetCodeAuthorization{{}},
							},
						},
					},
				},
			},
			expectedErr: "invalid MaxPriorityFeePerGas",
		},
		"BlobTx with overflowing BlobFeeCap": {
			proposal: RPCExecutionProposal{
				BlockRange: validRange,
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						RPCExecutionStepProposal{
							TransactionArgs: ethapi.TransactionArgs{
								From:                 &address,
								To:                   &common.Address{2},
								Nonce:                rpctest.ToHexUint64(1),
								Gas:                  rpctest.ToHexUint64(21000),
								MaxFeePerGas:         (*hexutil.Big)(big.NewInt(1)),
								MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(1)),
								BlobFeeCap:           (*hexutil.Big)(overflowingValue),
								BlobHashes:           []common.Hash{{1}},
							},
						},
					},
				},
			},
			expectedErr: "invalid BlobFeeCap",
		},

		// --- wrong transaction nested in group ---
		"error propagates from nested group step": {
			proposal: RPCExecutionProposal{
				BlockRange: validRange,
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						RPCExecutionPlanGroup{
							OneOf: true,
							Steps: []any{
								RPCExecutionStepProposal{
									TransactionArgs: ethapi.TransactionArgs{
										// missing From
										To:    &common.Address{2},
										Nonce: rpctest.ToHexUint64(1),
										Gas:   rpctest.ToHexUint64(21000),
									},
								},
							},
						},
					},
				},
			},
			expectedErr: "transaction in bundle must include from",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := convertProposalToPlan(signer, tt.proposal)
			require.Error(t, err)
			require.ErrorContains(t, err, tt.expectedErr)
		})
	}
}

func Test_RPCExecutionProposal_UnmarshalJSON_FailsOnInvalidTopLevel(t *testing.T) {
	tests := map[string]string{
		"not json at all":        `not json`,
		"top level is an array":  `[]`,
		"blockRange wrong type":  `{"blockRange": "invalid"}`,
		"steps is not an array":  `{"steps": 123}`,
		"oneOf is not a boolean": `{"oneOf": "yes", "steps": []}`,
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			var proposal RPCExecutionProposal
			err := json.Unmarshal([]byte(input), &proposal)
			require.Error(t, err)
		})
	}
}

func Test_RPCExecutionProposal_UnmarshalJSON_FailsOnInvalidFirstLevelStep(t *testing.T) {
	tests := map[string]string{
		"step is not an object": `{
			"steps": [123]
		}`,
		"step has invalid from field": `{
			"steps": [{"from": "not-an-address", "to": "0x0000000000000000000000000000000000000001"}]
		}`,
		"step has nested steps with invalid content": `{
			"steps": [{"steps": [true]}]
		}`,
		"step has nested steps with invalid flags": `{
			"steps": [{"tolerateFailed": "not a boolean"}]
		}`,
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			var proposal RPCExecutionProposal
			err := json.Unmarshal([]byte(input), &proposal)
			require.Error(t, err)
		})
	}
}

func Test_RPCExecutionProposal_UnmarshalJSON_FailsOnInvalidNestedLevelStep(t *testing.T) {
	tests := map[string]string{
		"nested group contains non-object element": `{
			"steps": [{"steps": [{"steps": [42]}]}]
		}`,
		"nested group contains invalid leaf": `{
			"steps": [{"steps": [{"steps": [{"from": "bad"}]}]}]
		}`,
		"deeply nested group has malformed steps array": `{
			"steps": [{"steps": [{"steps": "not-an-array"}]}]
		}`,
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			var proposal RPCExecutionProposal
			err := json.Unmarshal([]byte(input), &proposal)
			require.Error(t, err)
		})
	}
}
