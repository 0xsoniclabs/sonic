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

package bundle

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestIsBundledOnly_IdentifiesBundleOnlyTransactions_OfAllTypes(t *testing.T) {
	bundleOnlyMarker := types.AccessList{{Address: BundleOnly}}

	require.False(t, IsBundleOnly(types.NewTx(&types.LegacyTx{})))
	require.False(t, IsBundleOnly(types.NewTx(&types.AccessListTx{})))
	require.False(t, IsBundleOnly(types.NewTx(&types.DynamicFeeTx{})))
	require.False(t, IsBundleOnly(types.NewTx(&types.BlobTx{})))
	require.False(t, IsBundleOnly(types.NewTx(&types.SetCodeTx{})))

	require.True(t, IsBundleOnly(types.NewTx(&types.AccessListTx{
		AccessList: bundleOnlyMarker,
	})))
	require.True(t, IsBundleOnly(types.NewTx(&types.DynamicFeeTx{
		AccessList: bundleOnlyMarker,
	})))
	require.True(t, IsBundleOnly(types.NewTx(&types.BlobTx{
		AccessList: bundleOnlyMarker,
	})))
	require.True(t, IsBundleOnly(types.NewTx(&types.SetCodeTx{
		AccessList: bundleOnlyMarker,
	})))
}

func TestIsEnvelope_IdentifiesEnvelopes(t *testing.T) {
	tests := map[string]struct {
		tx       types.TxData
		expected bool
	}{
		"normal tx": {
			tx:       &types.LegacyTx{},
			expected: false,
		},
		"bundle tx": {
			tx: &types.LegacyTx{
				To: &BundleProcessor,
			},
			expected: true,
		},
		"not bundle address": {
			tx: &types.LegacyTx{
				To: &common.Address{0x01},
			},
			expected: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tx := types.NewTx(test.tx)
			result := IsEnvelope(tx)
			require.Equal(t, test.expected, result)
		})
	}
}

func TestOpenEnvelope_SuccessfullyDecodesEnvelopes(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	tests := map[string]TransactionBundle{
		"empty all-of": NewBuilder().AllOf().BuildBundle(),
		"empty one-of": NewBuilder().OneOf().BuildBundle(),
		"bundle with transactions": NewBuilder().AllOf(
			Step(key, &types.AccessListTx{Nonce: 1}),
			Step(key, &types.DynamicFeeTx{Nonce: 2}),
		).BuildBundle(),
	}

	for name, bundle := range tests {
		t.Run(name, func(t *testing.T) {
			payload, err := bundle.encode()
			require.NoError(t, err)
			envelope := types.NewTx(&types.LegacyTx{
				To:   &BundleProcessor,
				Data: payload,
			})

			unpacked, err := OpenEnvelope(signer, envelope)
			require.NoError(t, err)

			// Transactions can not be compared using require.Equal, so we
			// check them explicitly first before replacing them in the unpacked
			// bundle for the final equality check.
			require.Equal(t, len(bundle.Transactions), len(unpacked.Transactions))
			originalTxs := bundle.GetTransactionsInReferencedOrder()
			unpackedTxs := unpacked.GetTransactionsInReferencedOrder()
			for i, tx := range originalTxs {
				require.Equal(t, tx.Hash(), unpackedTxs[i].Hash())
			}
			unpacked.Transactions = bundle.Transactions

			require.Equal(t, bundle, unpacked)
		})
	}
}

func TestOpenEnvelope_FailsIfNotAnEnvelope(t *testing.T) {
	notEnvelope := types.NewTx(&types.LegacyTx{})
	require.False(t, IsEnvelope(notEnvelope))

	_, err := OpenEnvelope(nil, notEnvelope)
	require.ErrorContains(t, err, "not an envelope")
}

func TestTransactionBundle_GetTransactionsInReferencedOrder_ReturnsTransactionsInCorrectOrder(t *testing.T) {

	ref1 := TxReference{From: common.Address{1}}
	ref2 := TxReference{From: common.Address{2}}
	ref3 := TxReference{From: common.Address{3}}
	ref4 := TxReference{From: common.Address{4}}

	tx1 := types.NewTx(&types.LegacyTx{Nonce: 1})
	tx2 := types.NewTx(&types.LegacyTx{Nonce: 2})
	tx3 := types.NewTx(&types.LegacyTx{Nonce: 3})

	index := map[TxReference]*types.Transaction{
		ref1: tx1,
		ref2: tx2,
		ref3: tx3,
		// ref4 deliberately skipped to test that missing references are handled gracefully (e.g. by returning nil in the corresponding position in the result).
	}

	tests := map[string]struct {
		plan ExecutionStep
		want []*types.Transaction
	}{
		"empty": {
			plan: ExecutionStep{},
			want: nil,
		},
		"single": {
			plan: NewTxStep(ref1),
			want: []*types.Transaction{tx1},
		},
		"allOf group": {
			plan: NewAllOfStep(
				NewTxStep(ref1),
				NewTxStep(ref2),
				NewTxStep(ref3),
				NewTxStep(ref4),
			),
			want: []*types.Transaction{tx1, tx2, tx3, nil},
		},
		"duplicate references": {
			plan: NewOneOfStep(
				NewTxStep(ref1),
				NewTxStep(ref2),
				NewTxStep(ref1),
			),
			want: []*types.Transaction{tx1, tx2, tx1},
		},
		"nested groups": {
			plan: NewOneOfStep(
				NewAllOfStep(NewTxStep(ref1), NewTxStep(ref2)),
				NewAllOfStep(NewTxStep(ref1), NewTxStep(ref3)),
				NewAllOfStep(NewTxStep(ref2), NewTxStep(ref3)),
			),
			want: []*types.Transaction{tx1, tx2, tx1, tx3, tx2, tx3},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			bundle := TransactionBundle{
				Transactions: index,
				Plan: ExecutionPlan{
					Root: tc.plan,
				},
			}

			got := bundle.GetTransactionsInReferencedOrder()
			require.Equal(tc.want, got)
		})
	}
}

func TestRemoveBundleOnlyMark_FailsOnUnsupportedTransactionType(t *testing.T) {
	tx := types.NewTx(&types.LegacyTx{})
	_, err := removeBundleOnlyMark(tx)
	require.ErrorContains(t, err, "unsupported transaction type")
}

func TestRemoveBundleOnlyMark_PreservesOriginalData(t *testing.T) {

	type msg struct {
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
	}

	normalAccessListEntry := types.AccessList{
		{
			Address:     common.HexToAddress("0x0000000000000000000000000000000000000001"),
			StorageKeys: []common.Hash{{0x01}, {0x02}},
		},
	}
	bundleOnlyMark := types.AccessList{
		{
			Address:     BundleOnly,
			StorageKeys: []common.Hash{{0x01}},
		},
	}

	tests := make([]msg, 0)
	for _, gasPrice := range []*uint256.Int{nil, uint256.NewInt(0), uint256.NewInt(200)} {
		for _, gasFeeCap := range []*uint256.Int{nil, uint256.NewInt(0), uint256.NewInt(200)} {
			for _, gasTipCap := range []*uint256.Int{nil, uint256.NewInt(0), uint256.NewInt(200)} {
				for _, gas := range []uint64{0, 21000} {
					for _, to := range []*common.Address{nil, {0x01}} {
						for _, value := range []*uint256.Int{nil, uint256.NewInt(0), uint256.NewInt(100)} {
							for _, accessList := range []types.AccessList{
								nil,
								normalAccessListEntry,
							} {
								for _, blobHash := range [][]common.Hash{
									nil, {{0x01}, {0x02}},
								} {
									for _, authList := range [][]types.SetCodeAuthorization{
										nil, {
											{Address: common.Address{1}, Nonce: 0x02},
											{Address: common.Address{3}, Nonce: 0x04},
										},
									} {
										tests = append(tests, msg{
											Nonce:      1,
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

	for _, test := range tests {

		t.Run("preserves original members of the access list transaction", func(t *testing.T) {

			txData := &types.AccessListTx{
				Nonce:      test.Nonce,
				GasPrice:   test.GasPrice.ToBig(),
				Gas:        test.Gas,
				To:         test.To,
				Value:      test.Value.ToBig(),
				Data:       test.Data,
				AccessList: test.AccessList,
			}

			tx := types.NewTx(txData)
			modified, err := removeBundleOnlyMark(tx)
			require.NoError(t, err)

			if test.GasPrice == nil {
				test.GasPrice = uint256.NewInt(0)
			}
			if test.Value == nil {
				test.Value = uint256.NewInt(0)
			}
			if test.AccessList == nil {
				test.AccessList = types.AccessList{}
			}

			require.Equal(t, test.Nonce, modified.Nonce())
			require.Equal(t, test.GasPrice.Uint64(), modified.GasPrice().Uint64())
			require.Equal(t, test.Gas, modified.Gas())
			require.Equal(t, test.To, modified.To())
			require.Equal(t, test.Value.Uint64(), modified.Value().Uint64())
			require.Equal(t, test.Data, modified.Data())
			require.Equal(t, test.AccessList, modified.AccessList())
		})

		t.Run("removes bundle marker from the access list transaction", func(t *testing.T) {

			txData := &types.AccessListTx{
				Nonce:      test.Nonce,
				GasPrice:   test.GasPrice.ToBig(),
				Gas:        test.Gas,
				To:         test.To,
				Value:      test.Value.ToBig(),
				Data:       test.Data,
				AccessList: append(test.AccessList, bundleOnlyMark...),
			}

			tx := types.NewTx(txData)
			modified, err := removeBundleOnlyMark(tx)
			require.NoError(t, err)

			if test.AccessList == nil {
				test.AccessList = types.AccessList{}
			}

			require.Equal(t, test.AccessList, modified.AccessList())
		})

		t.Run("preserves original members of the dynamic fees transaction", func(t *testing.T) {

			txData := &types.DynamicFeeTx{
				Nonce:      test.Nonce,
				GasFeeCap:  test.GasFeeCap.ToBig(),
				GasTipCap:  test.GasTipCap.ToBig(),
				Gas:        test.Gas,
				To:         test.To,
				Value:      test.Value.ToBig(),
				Data:       test.Data,
				AccessList: test.AccessList,
			}

			tx := types.NewTx(txData)
			modified, err := removeBundleOnlyMark(tx)
			require.NoError(t, err)

			if test.GasFeeCap == nil {
				test.GasFeeCap = uint256.NewInt(0)
			}
			if test.GasTipCap == nil {
				test.GasTipCap = uint256.NewInt(0)
			}
			if test.Value == nil {
				test.Value = uint256.NewInt(0)
			}
			if test.AccessList == nil {
				test.AccessList = types.AccessList{}
			}

			require.Equal(t, test.Nonce, modified.Nonce())
			require.Equal(t, test.GasFeeCap.Uint64(), modified.GasFeeCap().Uint64())
			require.Equal(t, test.GasTipCap.Uint64(), modified.GasTipCap().Uint64())
			require.Equal(t, test.Gas, modified.Gas())
			require.Equal(t, test.To, modified.To())
			require.Equal(t, test.Value.Uint64(), modified.Value().Uint64())
			require.Equal(t, test.Data, modified.Data())
			require.Equal(t, test.AccessList, modified.AccessList())
		})

		t.Run("removes bundle marker from the dynamic fee transaction", func(t *testing.T) {

			txData := &types.DynamicFeeTx{
				Nonce:      test.Nonce,
				GasFeeCap:  test.GasFeeCap.ToBig(),
				GasTipCap:  test.GasTipCap.ToBig(),
				Gas:        test.Gas,
				To:         test.To,
				Value:      test.Value.ToBig(),
				Data:       test.Data,
				AccessList: append(test.AccessList, bundleOnlyMark...),
			}

			tx := types.NewTx(txData)
			modified, err := removeBundleOnlyMark(tx)
			require.NoError(t, err)

			if test.AccessList == nil {
				test.AccessList = types.AccessList{}
			}

			require.Equal(t, test.AccessList, modified.AccessList())
		})

		t.Run("preserves original members of a blob transaction", func(t *testing.T) {
			if test.To == nil {
				t.Skip("receiver must not be nil for blob transactions")
			}

			txData := &types.BlobTx{
				Nonce:      test.Nonce,
				GasFeeCap:  test.GasFeeCap,
				GasTipCap:  test.GasTipCap,
				Gas:        test.Gas,
				To:         *test.To,
				Value:      test.Value,
				Data:       test.Data,
				AccessList: test.AccessList,
				BlobFeeCap: test.GasPrice,
				BlobHashes: test.BlobHashes,
			}

			tx := types.NewTx(txData)
			modified, err := removeBundleOnlyMark(tx)
			require.NoError(t, err)

			if test.GasPrice == nil {
				test.GasPrice = uint256.NewInt(0)
			}
			if test.GasFeeCap == nil {
				test.GasFeeCap = uint256.NewInt(0)
			}
			if test.GasTipCap == nil {
				test.GasTipCap = uint256.NewInt(0)
			}
			if test.Value == nil {
				test.Value = uint256.NewInt(0)
			}
			if test.AccessList == nil {
				test.AccessList = types.AccessList{}
			}
			if test.BlobHashes == nil {
				test.BlobHashes = []common.Hash{}
			}

			require.Equal(t, test.Nonce, modified.Nonce())
			require.Equal(t, test.GasFeeCap.Uint64(), modified.GasFeeCap().Uint64())
			require.Equal(t, test.GasTipCap.Uint64(), modified.GasTipCap().Uint64())
			require.Equal(t, test.Gas, modified.Gas())
			require.Equal(t, test.To, modified.To())
			require.Equal(t, test.Value.Uint64(), modified.Value().Uint64())
			require.Equal(t, test.Data, modified.Data())
			require.Equal(t, test.AccessList, modified.AccessList())
			require.Equal(t, test.GasPrice.Uint64(), modified.BlobGasFeeCap().Uint64())
			require.Equal(t, test.BlobHashes, modified.BlobHashes())
		})

		t.Run("removes bundle marker from the blob transaction", func(t *testing.T) {
			if test.To == nil {
				t.Skip("receiver must not be nil for blob transactions")
			}

			txData := &types.BlobTx{
				Nonce:      test.Nonce,
				GasFeeCap:  test.GasPrice,
				GasTipCap:  test.GasPrice,
				Gas:        test.Gas,
				To:         *test.To,
				Value:      test.Value,
				Data:       test.Data,
				AccessList: append(test.AccessList, bundleOnlyMark...),
			}

			tx := types.NewTx(txData)
			modified, err := removeBundleOnlyMark(tx)
			require.NoError(t, err)

			if test.AccessList == nil {
				test.AccessList = types.AccessList{}
			}

			require.Equal(t, test.AccessList, modified.AccessList())
		})

		t.Run("preserves original members of a set code transaction", func(t *testing.T) {
			if test.To == nil {
				t.Skip("receiver must not be nil for set code transactions")
			}

			txData := &types.SetCodeTx{
				Nonce:      test.Nonce,
				GasFeeCap:  test.GasFeeCap,
				GasTipCap:  test.GasTipCap,
				Gas:        test.Gas,
				To:         *test.To,
				Value:      test.Value,
				Data:       test.Data,
				AccessList: test.AccessList,
				AuthList:   test.AuthList,
			}

			tx := types.NewTx(txData)
			modified, err := removeBundleOnlyMark(tx)
			require.NoError(t, err)

			if test.GasFeeCap == nil {
				test.GasFeeCap = uint256.NewInt(0)
			}
			if test.GasTipCap == nil {
				test.GasTipCap = uint256.NewInt(0)
			}
			if test.Value == nil {
				test.Value = uint256.NewInt(0)
			}
			if test.AccessList == nil {
				test.AccessList = types.AccessList{}
			}
			if test.AuthList == nil {
				test.AuthList = []types.SetCodeAuthorization{}
			}

			require.Equal(t, test.Nonce, modified.Nonce())
			require.Equal(t, test.GasFeeCap.Uint64(), modified.GasFeeCap().Uint64())
			require.Equal(t, test.GasTipCap.Uint64(), modified.GasTipCap().Uint64())
			require.Equal(t, test.Gas, modified.Gas())
			require.Equal(t, test.To, modified.To())
			require.Equal(t, test.Value.Uint64(), modified.Value().Uint64())
			require.Equal(t, test.Data, modified.Data())
			require.Equal(t, test.AccessList, modified.AccessList())
			require.Equal(t, test.AuthList, modified.SetCodeAuthorizations())
		})

		t.Run("removes bundle marker from the dynamic fee transaction", func(t *testing.T) {
			if test.To == nil {
				t.Skip("receiver must not be nil for set code transactions")
			}
			txData := &types.SetCodeTx{
				Nonce:      test.Nonce,
				GasFeeCap:  test.GasFeeCap,
				GasTipCap:  test.GasTipCap,
				Gas:        test.Gas,
				To:         *test.To,
				Value:      test.Value,
				Data:       test.Data,
				AccessList: append(test.AccessList, bundleOnlyMark...),
				AuthList:   test.AuthList,
			}

			tx := types.NewTx(txData)
			modified, err := removeBundleOnlyMark(tx)
			require.NoError(t, err)

			if test.AccessList == nil {
				test.AccessList = types.AccessList{}
			}

			require.Equal(t, test.AccessList, modified.AccessList())
		})
	}
}

//go:generate mockgen -source=bundle_test.go -destination=bundle_test_mock.go -package=bundle

type Signer interface {
	types.Signer
}

func TestDecode_SuccessfullyUnpacksValidBundle(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))

	key1, err := crypto.GenerateKey()
	require.NoError(t, err)

	key2, err := crypto.GenerateKey()
	require.NoError(t, err)

	tests := map[string]TransactionBundle{
		"empty all-of": NewBuilder().AllOf().BuildBundle(),
		"empty one-of": NewBuilder().OneOf().BuildBundle(),
		"bundle with transactions": NewBuilder().AllOf(
			Step(key1, &types.AccessListTx{}),
			Step(key2, &types.DynamicFeeTx{}),
		).BuildBundle(),
		"bundle with nested transactions": NewBuilder().OneOf(
			AllOf(
				Step(key1, &types.AccessListTx{Nonce: 1}),
				Step(key2, &types.DynamicFeeTx{Nonce: 2}),
			),
			AllOf(
				Step(key1, &types.AccessListTx{Nonce: 3}),
				Step(key2, &types.DynamicFeeTx{Nonce: 4}),
			).WithFlags(EF_TolerateFailed),
		).BuildBundle(),
	}

	for name, bundle := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			encoded, err := bundle.encode()
			require.NoError(err)
			restored, err := decode(signer, encoded)
			require.NoError(err)

			require.Equal(bundle.Plan, restored.Plan)
			require.Equal(len(bundle.Transactions), len(restored.Transactions))

			for ref, tx := range bundle.Transactions {
				restoredTx, ok := restored.Transactions[ref]
				require.True(ok, "transaction reference not found in restored bundle: %v", ref)
				require.Equal(tx.Hash(), restoredTx.Hash())
			}
		})
	}
}

func TestEncoding_IsVersioned(t *testing.T) {

	tests := map[string]struct {
		version       byte
		expectedError string
	}{
		"zero version": {
			version:       0,
			expectedError: "failed to decode version",
		},
		"invalid version": {
			version:       77,
			expectedError: "unsupported bundle version: 77",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			bundle := NewBuilder().AllOf().BuildBundle()

			encoded, err := encodeInternal(test.version, &bundle)
			require.NoError(t, err)

			_, err = decode(nil, encoded)
			require.ErrorContains(t, err, test.expectedError)
		})
	}
}

func TestEncode_NonEncodableTransactions_ReturnsError(t *testing.T) {
	require := require.New(t)
	nonEncodableTransaction := types.NewTx(&types.LegacyTx{
		Value: new(big.Int).Sub(big.NewInt(0), big.NewInt(1)), // negative value is not encodable
	})

	var buffer bytes.Buffer
	issue := rlp.Encode(&buffer, nonEncodableTransaction)
	require.Error(issue)

	bundle := TransactionBundle{
		Transactions: map[TxReference]*types.Transaction{
			{}: nonEncodableTransaction,
		},
	}

	_, err := bundle.encode()
	require.ErrorIs(err, issue)
	require.ErrorContains(err, "failed to encode transaction bundle")
}

func TestDecode_ReturnsErrorForInvalidData(t *testing.T) {
	_, err := decode(nil, []byte{0x01, 0x02, 0x03})
	require.ErrorContains(t, err, "failed to decode transaction bundle")

	_, err = decode(nil, nil)
	require.ErrorContains(t, err, "failed to decode transaction bundle")
}

func TestDecode_LegacyTxDataInBundle_FailsDecodingSinceBundleOnlyMarkCanNotBeRemoved(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	legacyTx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		To:   &BundleProcessor,
		Data: []byte{0x01, 0x02},
	})

	bundle := TransactionBundle{
		Transactions: map[TxReference]*types.Transaction{
			{}: legacyTx,
		},
		Plan: ExecutionPlan{
			Root: NewTxStep(TxReference{}),
		},
	}

	encoded, err := bundle.encode()
	require.NoError(t, err)

	_, err = decode(signer, encoded)
	require.ErrorContains(t, err, "failed to remove bundle-only mark")
}

func TestDecode_CorruptedExecutionPlanEncoding_DetectedDuringDecoding(t *testing.T) {
	bundle := TransactionBundle{
		Plan: ExecutionPlan{
			Root: NewTxStep(TxReference{}),
		},
	}

	encoded, err := bundle.encode()
	require.NoError(t, err)

	_, err = decode(nil, encoded)
	require.NoError(t, err)

	// If the execution plan data is corrupted, the issue is detected.
	encoded[len(encoded)-1] = 0x17 // < last byte contains length of sub-step list
	_, err = decode(nil, encoded)
	require.ErrorContains(t, err, "failed to decode execution plan")
}
