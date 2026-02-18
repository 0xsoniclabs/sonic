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

func TestIsBundledOnly_IdentifiesBundleOnlyTransactions(t *testing.T) {
	require.False(t, IsBundleOnly(types.NewTx(&types.LegacyTx{})))
	require.True(t, IsBundleOnly(types.NewTx(&types.AccessListTx{
		AccessList: types.AccessList{
			{
				Address: BundleOnly,
			},
		},
	})))
	require.True(t, IsBundleOnly(types.NewTx(&types.DynamicFeeTx{
		AccessList: types.AccessList{
			{
				Address: BundleOnly,
			},
		},
	})))
	require.True(t, IsBundleOnly(types.NewTx(&types.BlobTx{
		AccessList: types.AccessList{
			{
				Address: BundleOnly,
			},
		},
	})))
	require.True(t, IsBundleOnly(types.NewTx(&types.SetCodeTx{
		AccessList: types.AccessList{
			{
				Address: BundleOnly,
			},
		},
	})))
}

func TestBelongsToExecutionPlan_IdentifiesTransactionsWhichSignTheExecutionPlan(t *testing.T) {

	tests := map[string]struct {
		tx                types.TxData
		executionPlanHash common.Hash
		expected          bool
	}{
		"transaction without access list": {
			tx:       &types.LegacyTx{},
			expected: false,
		},
		"transaction with bundle-only but no pan hash": {
			tx: &types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address: BundleOnly,
					},
				},
			},
			executionPlanHash: common.Hash{0x01, 0x02, 0x03},
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
						StorageKeys: []common.Hash{{0x01, 0x02, 0x03}},
					},
				},
			},
			executionPlanHash: common.Hash{0x01, 0x02, 0x03},
			expected:          false,
		},
		"transaction with bundle-only and matching plan hash": {
			tx: &types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address:     BundleOnly,
						StorageKeys: []common.Hash{{0x01, 0x02, 0x03}},
					},
				},
			},
			executionPlanHash: common.Hash{0x01, 0x02, 0x03},
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

func TestBelongsToExecutionPlan_IdentifiesMarkedTransactions(t *testing.T) {

	executionPlanHash := common.Hash{0x01, 0x02, 0x03} // dummy hash

	tests := map[string]struct {
		tx       types.TxData
		expected bool
	}{
		"without access list ": {
			tx:       &types.AccessListTx{},
			expected: false,
		},
		"without bundle-only": {
			tx: &types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address: common.HexToAddress("0x0000000000000000000000000000000000000001"),
					},
				},
			},
			expected: false,
		},
		"with bundle-only, no execution plan hash": {
			tx: &types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address: BundleOnly,
					},
				},
			},
			expected: false,
		},
		"with bundle-only, and execution plan hash": {
			tx: &types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address:     BundleOnly,
						StorageKeys: []common.Hash{executionPlanHash},
					},
				},
			},
			expected: true,
		},
		"with bundle only and others": {
			tx: &types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address: common.HexToAddress("0x0000000000000000000000000000000000000001"),
					},
					{
						Address:     BundleOnly,
						StorageKeys: []common.Hash{executionPlanHash},
					},
				},
			},
			expected: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tx := types.NewTx(test.tx)
			result := BelongsToExecutionPlan(tx, executionPlanHash)
			require.Equal(t, test.expected, result)
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

	removeBundleMark := func(tx types.TxData) (types.TxData, error) {

		switch tx := tx.(type) {
		case *types.AccessListTx:
			var accessList types.AccessList
			for _, entry := range tx.AccessList {
				if entry.Address == BundleOnly {
					continue
				}
				accessList = append(accessList, entry)
			}
			return &types.AccessListTx{
				Nonce:      tx.Nonce,
				GasPrice:   tx.GasPrice,
				Gas:        tx.Gas,
				To:         tx.To,
				Value:      tx.Value,
				Data:       tx.Data,
				AccessList: accessList,
			}, nil
		case *types.DynamicFeeTx:
			var accessList types.AccessList
			for _, entry := range tx.AccessList {
				if entry.Address == BundleOnly {
					continue
				}
				accessList = append(accessList, entry)
			}
			return &types.DynamicFeeTx{
				Nonce:      tx.Nonce,
				GasTipCap:  tx.GasTipCap,
				GasFeeCap:  tx.GasFeeCap,
				Gas:        tx.Gas,
				To:         tx.To,
				Value:      tx.Value,
				Data:       tx.Data,
				AccessList: accessList,
			}, nil
		}
		return nil, errors.New("unsupported transaction type")

	}

	for _, txData := range txs {

		tx, err := types.SignNewTx(key, signer, txData)
		require.NoError(t, err)

		bundle := TransactionBundle{
			Bundle: types.Transactions{tx},
		}
		executionPlan, err := bundle.ExtractExecutionPlan(signer)
		require.NoError(t, err)

		require.Equal(t, 1, len(executionPlan.Steps))

		withoutBundleOnlyMark, err := removeBundleMark(txData)
		require.NoError(t, err)

		hash := signer.Hash(types.NewTx(withoutBundleOnlyMark))
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

		bundle := TransactionBundle{
			Bundle: types.Transactions{
				types.NewTx(txData),
			},
		}

		ctrl := gomock.NewController(t)
		mockSigner := NewMockSigner(ctrl)

		var txType byte
		switch txData.(type) {
		case *types.LegacyTx:
			txType = types.LegacyTxType
		case *types.BlobTx:
			txType = types.BlobTxType
		case *types.SetCodeTx:
			txType = types.SetCodeTxType
		}

		_, err := bundle.ExtractExecutionPlan(mockSigner)
		require.ErrorContains(t, err,
			fmt.Sprintf("invalid bundle: unsupported transaction type %d", txType))
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
