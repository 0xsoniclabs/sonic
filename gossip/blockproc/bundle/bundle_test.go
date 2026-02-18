// Copyright 2025 Sonic Operations Ltd
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
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestExecutionPlan_Hash_ComputesDeterministicHash(t *testing.T) {

	step1 := ExecutionStep{
		From: common.HexToAddress("0x0000000000000000000000000000000000000001"),
		Hash: common.Hash{0x01},
	}
	step2 := ExecutionStep{
		From: common.HexToAddress("0x0000000000000000000000000000000000000002"),
		Hash: common.Hash{0x02},
	}

	tests := map[string]ExecutionPlan{
		"empty plan": {},
		"plan with transactions": {
			Steps: []ExecutionStep{step1, step2},
		},
		"plan with flag 1": {
			Steps: []ExecutionStep{step1},
			Flags: 0x1,
		},
		"plan with flag 2": {
			Steps: []ExecutionStep{step1},
			Flags: 0x2,
		},
		"plan with flag 3": {
			Steps: []ExecutionStep{step1},
			Flags: 0x3,
		},
	}

	seenHashes := make(map[common.Hash]struct{})
	for name, executionPlan := range tests {
		t.Run(name, func(t *testing.T) {

			transactions := make([]any, len(executionPlan.Steps))
			for i, step := range executionPlan.Steps {
				transactions[i] = []any{step.From, step.Hash}
			}
			manualSerialize := []any{transactions, executionPlan.Flags}

			hasher := crypto.NewKeccakState()
			require.NoError(t, rlp.Encode(hasher, manualSerialize))
			computed := common.BytesToHash(hasher.Sum(nil))

			require.Equal(t, executionPlan.Hash(), computed)
			require.NotContains(t, seenHashes, computed, "hash should be unique for different plans")
			seenHashes[computed] = struct{}{}
		})
	}
}

func TestIsTransactionBundle_IdentifiesBundles(t *testing.T) {
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
				To: &BundleAddress,
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
			result := IsTransactionBundle(tx)
			require.Equal(t, test.expected, result)
		})
	}
}

func TestIsBundledOnly_IdentifiesBundleOnlyTransactions_OfAllTypes(t *testing.T) {
	bundleOnlyMarker := types.AccessList{{Address: BundleOnly}}
	require.False(t, IsBundleOnly(types.NewTx(&types.LegacyTx{})))
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

func TestBelongsToExecutionPlan_IdentifiesTransactionsWhichSignTheExecutionPlan(t *testing.T) {

	executionPlanHash := common.Hash{0x01, 0x02, 0x03}

	tests := map[string]struct {
		tx                types.TxData
		executionPlanHash common.Hash
		expected          bool
	}{
		"transaction without access list": {
			tx:       &types.LegacyTx{},
			expected: false,
		},
		"transaction with bundle-only but no plan hash": {
			tx: &types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address: BundleOnly,
					},
				},
			},
			executionPlanHash: executionPlanHash,
			expected:          false,
		},
		"fragmented access list": {
			tx: &types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address: BundleOnly,
					},
					{
						Address:     common.HexToAddress("0x0000000000000000000000000000000000000001"),
						StorageKeys: []common.Hash{executionPlanHash},
					},
				},
			},
			executionPlanHash: executionPlanHash,
			expected:          false,
		},
		"transaction with bundle-only and matching plan hash": {
			tx: &types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address:     BundleOnly,
						StorageKeys: []common.Hash{executionPlanHash},
					},
				},
			},
			executionPlanHash: executionPlanHash,
			expected:          true,
		},
		"transaction with multiple accepted plans": {
			tx: &types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address:     BundleOnly,
						StorageKeys: []common.Hash{{0x0A}},
					},
					{
						Address:     BundleOnly,
						StorageKeys: []common.Hash{executionPlanHash},
					},
					{
						Address:     BundleOnly,
						StorageKeys: []common.Hash{{0x0B}},
					},
				},
			},
			executionPlanHash: executionPlanHash,
			expected:          true,
		},
		"transaction with multiple accepted plans compact": {
			tx: &types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address:     BundleOnly,
						StorageKeys: []common.Hash{{0x0A}, executionPlanHash, {0x0B}},
					},
				},
			},
			executionPlanHash: executionPlanHash,
			expected:          true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tx := types.NewTx(test.tx)
			require.Equal(t, test.expected,
				BelongsToExecutionPlan(tx, test.executionPlanHash))
		})
	}

}

func TestDecode_SuccessfullyUnpacksValidBundle(t *testing.T) {

	for _, flags := range []ExecutionFlag{0, 1, 2, 3} {

		executionPlanHash := common.Hash{0x01, 0x02, 0x03} // dummy hash

		bundle := TransactionBundle{
			Version: 1,
			Bundle: types.Transactions{
				types.NewTx(&types.AccessListTx{
					AccessList: types.AccessList{
						{
							Address:     BundleOnly,
							StorageKeys: []common.Hash{executionPlanHash},
						},
					},
				}),
			},
			Payment: types.NewTx(
				&types.AccessListTx{
					AccessList: types.AccessList{
						{Address: BundleOnly},
					},
				},
			),
			Flags: flags,
		}

		unpacked, err := Decode(Encode(bundle))
		require.NoError(t, err)
		require.Equal(t, bundle.Version, unpacked.Version)

		require.Equal(t, bundle.Payment.Hash(), unpacked.Payment.Hash())
		for i, tx := range bundle.Bundle {
			require.Equal(t, tx.Hash(), unpacked.Bundle[i].Hash())
		}
		require.Equal(t, bundle.Flags, unpacked.Flags)
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
			bundle := TransactionBundle{
				Version: test.version,
			}

			_, err := Decode(Encode(bundle))
			require.ErrorContains(t, err, test.expectedError)
		})
	}
}

func TestDecode_ReturnsErrorForInvalidData(t *testing.T) {
	_, err := Decode([]byte{0x01, 0x02, 0x03})
	require.ErrorContains(t, err, "failed to decode transaction bundle")

	_, err = Decode(nil)
	require.ErrorContains(t, err, "failed to decode transaction bundle")
}

//go:generate mockgen -source=bundle_test.go -destination=bundle_test_mock.go -package=bundle

type Signer interface {
	types.Signer
}

func TestExtractExecutionPlan_ExtractsStepsAndFlags(t *testing.T) {

	for _, flags := range []ExecutionFlag{0, 1, 2, 3} {

		bundle := TransactionBundle{
			Bundle: types.Transactions{
				types.NewTx(&types.AccessListTx{
					AccessList: types.AccessList{
						{
							Address:     BundleOnly,
							StorageKeys: []common.Hash{{0x01}},
						},
					},
				}),
				types.NewTx(&types.DynamicFeeTx{
					AccessList: types.AccessList{
						{
							Address:     BundleOnly,
							StorageKeys: []common.Hash{{0x01}},
						},
					},
				}),
			},
			Flags: flags,
		}

		ctrl := gomock.NewController(t)
		mockSigner := NewMockSigner(ctrl)
		mockSigner.EXPECT().Sender(gomock.Any()).Return(common.Address{0x42}, nil)
		mockSigner.EXPECT().Sender(gomock.Any()).Return(common.Address{0x43}, nil)
		mockSigner.EXPECT().Hash(gomock.Any()).Return(common.Hash{0x01})
		mockSigner.EXPECT().Hash(gomock.Any()).Return(common.Hash{0x02})

		executionPlan, err := bundle.ExtractExecutionPlan(mockSigner)
		require.NoError(t, err)

		require.Equal(t, 2, len(executionPlan.Steps))
		require.Equal(t, common.Address{0x42}, executionPlan.Steps[0].From)
		require.Equal(t, common.Hash{0x01}, executionPlan.Steps[0].Hash)
		require.Equal(t, common.Address{0x43}, executionPlan.Steps[1].From)
		require.Equal(t, common.Hash{0x02}, executionPlan.Steps[1].Hash)
		require.Equal(t, bundle.Flags, executionPlan.Flags)
	}
}

func TestExtractExecutionPlan_ComputesHashOfUnmarkedBundledTransactions(t *testing.T) {
	// This test verifies that the hash of transactions included in the execution plan
	// correspond to the hash of the transaction without the bundle-only marker in the access list.

	txs := []types.TxData{

		&types.AccessListTx{
			Nonce:    1,
			GasPrice: big.NewInt(100),
			Gas:      21000,
			To:       &common.Address{0x01},
			Value:    big.NewInt(100),
			Data:     []byte{0x01, 0x02},
			AccessList: types.AccessList{
				{
					Address:     BundleOnly,
					StorageKeys: []common.Hash{{0x01}},
				},
				{
					Address: common.HexToAddress("0x0000000000000000000000000000000000000001"),
				},
			},
		},

		&types.DynamicFeeTx{
			Nonce:     2,
			GasTipCap: big.NewInt(100),
			GasFeeCap: big.NewInt(200),
			Gas:       21000,
			To:        &common.Address{0x02},
			Value:     big.NewInt(100),
			Data:      []byte{0x03, 0x04},
			AccessList: types.AccessList{
				{
					Address:     BundleOnly,
					StorageKeys: []common.Hash{{0x01}},
				},
				{
					Address: common.HexToAddress("0x0000000000000000000000000000000000000002"),
				},
			},
		},
	}

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	signer := types.LatestSignerForChainID(big.NewInt(1))

	for _, txData := range txs {

		tx, err := types.SignNewTx(key, signer, txData)
		require.NoError(t, err)

		bundle := TransactionBundle{
			Bundle: types.Transactions{tx},
		}
		executionPlan, err := bundle.ExtractExecutionPlan(signer)
		require.NoError(t, err)

		require.Equal(t, 1, len(executionPlan.Steps))

		withoutBundleOnlyMark, err := removeBundleOnlyMark(types.NewTx(txData))
		require.NoError(t, err)

		hash := signer.Hash(withoutBundleOnlyMark)
		require.Equal(t, hash, executionPlan.Steps[0].Hash)
	}
}

func TestExtractExecutionPlan_ReturnsErrorWithUnsupportedTransactionType(t *testing.T) {

	tests := []types.TxData{
		&types.LegacyTx{},
		&types.BlobTx{},
		&types.SetCodeTx{},
	}

	for _, txData := range tests {

		tx := types.NewTx(txData)
		bundle := TransactionBundle{
			Bundle: types.Transactions{tx},
		}

		ctrl := gomock.NewController(t)
		mockSigner := NewMockSigner(ctrl)
		mockSigner.EXPECT().Sender(gomock.Any()).Return(common.Address{}, nil)

		_, err := bundle.ExtractExecutionPlan(mockSigner)
		require.ErrorContains(t, err,
			fmt.Sprintf("invalid bundle: unsupported transaction type %d", tx.Type()))
	}
}

func TestExtractExecutionPlan_ReturnsErrorWithMalformedSignature(t *testing.T) {

	bundle := TransactionBundle{
		Bundle: types.Transactions{
			types.NewTx(&types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address:     BundleOnly,
						StorageKeys: []common.Hash{{0x01}},
					},
				},
			}),
		},
	}

	ctrl := gomock.NewController(t)
	mockSigner := NewMockSigner(ctrl)
	mockSigner.EXPECT().Sender(gomock.Any()).Return(common.Address{}, errors.New("invalid signature"))

	_, err := bundle.ExtractExecutionPlan(mockSigner)
	require.ErrorContains(t, err, "failed to derive sender: invalid signature")
}

func TestRemoveBundleMark_ReturnsErrorWithUnsupportedTransactionType(t *testing.T) {
	tests := []types.TxData{
		&types.LegacyTx{},
		&types.BlobTx{},
		&types.SetCodeTx{},
	}

	for _, txData := range tests {
		tx := types.NewTx(txData)
		_, err := removeBundleOnlyMark(tx)
		require.ErrorContains(t, err,
			fmt.Sprintf("invalid bundle: unsupported transaction type %d", tx.Type()))
	}
}

func TestRemoveBundleMark_PreservesOriginalData(t *testing.T) {

	type msg struct {
		Nonce      uint64
		GasPrice   *big.Int
		Gas        uint64
		To         *common.Address
		Value      *big.Int
		Data       []byte
		AccessList types.AccessList
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
	for _, gasPrice := range []*big.Int{nil, big.NewInt(0), big.NewInt(200)} {
		for _, gas := range []uint64{0, 21000} {
			for _, to := range []*common.Address{nil, {0x01}} {
				for _, value := range []*big.Int{nil, big.NewInt(0), big.NewInt(100)} {
					for _, accessList := range []types.AccessList{
						nil,
						normalAccessListEntry,
					} {

						tests = append(tests, msg{
							Nonce:      1,
							GasPrice:   gasPrice,
							Gas:        gas,
							To:         to,
							Value:      value,
							Data:       []byte{0x01, 0x02},
							AccessList: accessList,
						})
					}
				}
			}
		}
	}

	for _, test := range tests {

		t.Run("preserves original members of the access list transaction", func(t *testing.T) {

			txData := &types.AccessListTx{
				Nonce:      test.Nonce,
				GasPrice:   test.GasPrice,
				Gas:        test.Gas,
				To:         test.To,
				Value:      test.Value,
				Data:       test.Data,
				AccessList: test.AccessList,
			}

			tx := types.NewTx(txData)
			modified, err := removeBundleOnlyMark(tx)
			require.NoError(t, err)

			if test.GasPrice == nil {
				test.GasPrice = big.NewInt(0)
			}
			if test.Value == nil {
				test.Value = big.NewInt(0)
			}
			if test.AccessList == nil {
				test.AccessList = types.AccessList{}
			}

			require.Equal(t, test.Nonce, modified.Nonce())
			require.Equal(t, test.GasPrice, modified.GasPrice())
			require.Equal(t, test.Gas, modified.Gas())
			require.Equal(t, test.To, modified.To())
			require.Equal(t, test.Value, modified.Value())
			require.Equal(t, test.Data, modified.Data())
			require.Equal(t, test.AccessList, modified.AccessList())
		})

		t.Run("removes bundle marker from the access list transaction", func(t *testing.T) {

			txData := &types.AccessListTx{
				Nonce:      test.Nonce,
				GasPrice:   test.GasPrice,
				Gas:        test.Gas,
				To:         test.To,
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

		t.Run("preserves original members of the dynamic fees transaction", func(t *testing.T) {

			txData := &types.DynamicFeeTx{
				Nonce:      test.Nonce,
				GasFeeCap:  test.GasPrice,
				GasTipCap:  test.GasPrice,
				Gas:        test.Gas,
				To:         test.To,
				Value:      test.Value,
				Data:       test.Data,
				AccessList: test.AccessList,
			}

			tx := types.NewTx(txData)
			modified, err := removeBundleOnlyMark(tx)
			require.NoError(t, err)

			if test.GasPrice == nil {
				test.GasPrice = big.NewInt(0)
			}
			if test.Value == nil {
				test.Value = big.NewInt(0)
			}
			if test.AccessList == nil {
				test.AccessList = types.AccessList{}
			}

			require.Equal(t, test.Nonce, modified.Nonce())
			require.Equal(t, test.GasPrice, modified.GasFeeCap())
			require.Equal(t, test.GasPrice, modified.GasTipCap())
			require.Equal(t, test.Gas, modified.Gas())
			require.Equal(t, test.To, modified.To())
			require.Equal(t, test.Value, modified.Value())
			require.Equal(t, test.Data, modified.Data())
			require.Equal(t, test.AccessList, modified.AccessList())
		})

		t.Run("removes bundle marker from the dynamic fee transaction", func(t *testing.T) {

			txData := &types.DynamicFeeTx{
				Nonce:      test.Nonce,
				GasFeeCap:  test.GasPrice,
				GasTipCap:  test.GasPrice,
				Gas:        test.Gas,
				To:         test.To,
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
	}
}
