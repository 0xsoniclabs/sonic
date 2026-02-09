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
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

//go:generate mockgen -source=bundle_test.go -destination=bundle_test_mock.go -package=bundle

type Signer interface {
	types.Signer
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

func TestExecutionPlan_GetCost_ComputesAggregatedCost(t *testing.T) {
	expectedOverhead := uint64(20_000)

	tests := map[string]struct {
		executionPlan ExecutionPlan
		gasPrice      uint64
		expectedCost  uint64
	}{
		"empty execution plan": {
			executionPlan: ExecutionPlan{},
			gasPrice:      100,
			expectedCost:  expectedOverhead * 100,
		},
		"execution plan with transactions": {
			executionPlan: ExecutionPlan{
				Transactions: []MetaTransaction{
					{GasLimit: 21000},
					{GasLimit: 50000},
				},
			},
			gasPrice:     100,
			expectedCost: (expectedOverhead + 21000 + 50000) * 100,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cost := test.executionPlan.GetCost(big.NewInt(int64(test.gasPrice)))
			require.Equal(t, test.expectedCost, cost.Uint64())
		})
	}
}

func TestExecutionPlan_Hash_ComputesDeterministicHash(t *testing.T) {

	executionPlan := ExecutionPlan{
		Transactions: []MetaTransaction{
			{
				To:       &common.Address{0x01},
				From:     common.Address{0x02},
				Nonce:    1,
				GasLimit: 21000,
				Value:    big.NewInt(100),
				Data:     []byte{0x01, 0x02},
			},
			{
				To:       &common.Address{0x03},
				From:     common.Address{0x04},
				Nonce:    2,
				GasLimit: 50000,
				Value:    big.NewInt(200),
				Data:     []byte{0x03, 0x04},
			},
		},
		Flags: FlagIgnoreFailed | FlagAtMostOne,
	}

	hash1 := executionPlan.Hash()
	require.Equal(t, hash1, common.HexToHash("0x7544e75f7dc56e36c27c7b74a89e31862374efc184772070f4db1a3db1677305"))
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

func TestUnpackTransactionBundle_ReturnsErrorForNonBundle(t *testing.T) {

	tx := types.NewTx(&types.LegacyTx{})
	_, err := UnpackTransactionBundle(tx)
	require.ErrorContains(t, err, "failed to unpack bundle, not a transaction bundle")

	tx = types.NewTx(&types.LegacyTx{To: &common.Address{0x01}})
	_, err = UnpackTransactionBundle(tx)
	require.ErrorContains(t, err, "failed to unpack bundle, not a transaction bundle")
}

func TestUnpackTransactionBundle_ReturnsErrorForInvalidRLP(t *testing.T) {
	tx := types.NewTx(&types.LegacyTx{
		To:   &BundleAddress,
		Data: []byte{0x01, 0x02, 0x03},
	})
	_, err := UnpackTransactionBundle(tx)
	require.ErrorContains(t, err, "failed to unpack bundle, rlp")
}

func TestUnpackTransactionBundle_SuccessfullyUnpacksValidBundle(t *testing.T) {

	bundle := TransactionBundle{
		Bundle: types.Transactions{
			types.NewTx(&types.LegacyTx{}),
			types.NewTx(&types.LegacyTx{}),
		},
		Payment: types.NewTx(&types.LegacyTx{}),
		Flags:   FlagIgnoreFailed | FlagAtMostOne,
	}

	var buf []byte
	buf, err := rlp.EncodeToBytes(bundle)
	require.NoError(t, err)

	tx := types.NewTx(&types.LegacyTx{
		To:   &BundleAddress,
		Data: buf,
	})

	unpackedBundle, err := UnpackTransactionBundle(tx)
	require.NoError(t, err)
	require.Equal(t, bundle.Flags, unpackedBundle.Flags)
	require.Len(t, unpackedBundle.Bundle, 2)
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

func TestExtractExecutionPlan_FailsIfSenderCannotBeDeduced(t *testing.T) {

	ctrl := gomock.NewController(t)
	signer := NewMockSigner(ctrl)

	signer.EXPECT().Sender(gomock.Any()).Return(
		common.Address{}, fmt.Errorf("sender cannot be deduced"),
	).AnyTimes()

	tb := TransactionBundle{
		Bundle: types.Transactions{
			types.NewTx(&types.LegacyTx{}),
		},
		Flags: FlagIgnoreFailed | FlagAtMostOne,
	}

	_, err := tb.ExtractExecutionPlan(signer)
	require.ErrorContains(t, err, "sender cannot be deduced")
}

func TestExtractExecutionPlan_SucceedsIfSenderCanBeDeduced(t *testing.T) {

	ctrl := gomock.NewController(t)
	signer := NewMockSigner(ctrl)

	signer.EXPECT().Sender(gomock.Any()).Return(
		common.Address{0x42}, nil,
	).AnyTimes()

	tb := TransactionBundle{
		Bundle: types.Transactions{
			types.NewTx(&types.LegacyTx{}),
			types.NewTx(&types.AccessListTx{}),
			types.NewTx(&types.DynamicFeeTx{}),
			types.NewTx(&types.BlobTx{}),
			types.NewTx(&types.SetCodeTx{}),
		},
		Flags: FlagIgnoreFailed | FlagAtMostOne,
	}

	executionPlan, err := tb.ExtractExecutionPlan(signer)
	require.NoError(t, err)
	require.Equal(t, 5, len(executionPlan.Transactions))
	require.Equal(t, tb.Flags, executionPlan.Flags)
	for _, meta := range executionPlan.Transactions {
		require.Equal(t, common.Address{0x42}, meta.From)
	}
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

func TestTransactionBundleFlags(t *testing.T) {

	type testCase struct {
		flags                      ExecutionFlag
		revertOnInvalid            bool
		revertOnFailed             bool
		stopAfterFirstSuccessfulTx bool
	}

	tests := map[string]testCase{}

	for _, ignoreInvalid := range []bool{false, true} {
		for _, ignoreFailed := range []bool{false, true} {
			for _, onlyOne := range []bool{false, true} {
				name := fmt.Sprintf(
					"FlagIgnoreInvalid=%t,FlagIgnoreFailed=%t,FlagAtMostOne=%t",
					ignoreInvalid, ignoreFailed, onlyOne)

				flags := ExecutionFlag(0)
				if ignoreInvalid {
					flags |= FlagIgnoreInvalid
				}
				if ignoreFailed {
					flags |= FlagIgnoreFailed
				}
				if onlyOne {
					flags |= FlagAtMostOne
				}

				tests[name] = testCase{
					flags:                      flags,
					revertOnInvalid:            !ignoreInvalid,
					revertOnFailed:             !ignoreFailed,
					stopAfterFirstSuccessfulTx: onlyOne,
				}
			}
		}
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tb := TransactionBundle{
				Flags: test.flags,
			}
			require.Equal(t, test.revertOnInvalid, tb.RevertOnInvalidTransaction())
			require.Equal(t, test.revertOnFailed, tb.RevertOnFailedTransaction())
			require.Equal(t, test.stopAfterFirstSuccessfulTx, tb.StopAfterFirstSuccessfulTransaction())
		})
	}
}
