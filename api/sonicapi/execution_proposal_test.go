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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
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
		"empty bundle": {
			bundle: bundle.NewBuilder().WithSigner(signer).BuildBundle(),
			json: `{
				"blockRange":{"earliest":"0x0","latest":"0x3ff"},
				"steps":[{"steps":null}]
			}`,
		},
		"simple bundle": {
			bundle: bundle.NewBuilder().
				WithSigner(signer).
				With(bundle.Step(key, &types.AccessListTx{})).
				BuildBundle(),
			json: fmt.Sprintf(`{
				"blockRange":{"earliest":"0x0","latest":"0x3ff"},
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
				"blockRange":{"earliest":"0x0","latest":"0x3ff"},
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
				"blockRange":{"earliest":"0x0","latest":"0x3ff"},
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
				"blockRange":{"earliest":"0x0","latest":"0x3ff"},
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
				"blockRange":{"earliest":"0x0","latest":"0x3ff"},
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
	require.EqualValues(t, *rpctest.ToHexUint64(0), proposal.BlockRange.Earliest)
	require.EqualValues(t, *rpctest.ToHexUint64(1023), proposal.BlockRange.Latest)
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
				require.Equal(t, tt.expected[i], *result)

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
