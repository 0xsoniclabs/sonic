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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
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
			manualSerialize := []any{
				transactions,
				executionPlan.Flags,
				executionPlan.Earliest,
				executionPlan.Latest,
			}

			hasher := crypto.NewKeccakState()
			require.NoError(t, rlp.Encode(hasher, manualSerialize))
			computed := common.BytesToHash(hasher.Sum(nil))

			require.Equal(t, executionPlan.Hash(), computed)
			require.NotContains(t, seenHashes, computed, "hash should be unique for different plans")
			seenHashes[computed] = struct{}{}
		})
	}
}

func TestIsEnvelope_IdentifiesEnvelops(t *testing.T) {
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

func TestDecode_SuccessfullyUnpacksValidBundle(t *testing.T) {

	for _, flags := range []ExecutionFlag{0, 1, 2, 3} {

		executionPlanHash := common.Hash{0x01, 0x02, 0x03} // dummy hash

		bundle := TransactionBundle{
			Transactions: types.Transactions{
				types.NewTx(&types.AccessListTx{
					AccessList: types.AccessList{
						{
							Address:     BundleOnly,
							StorageKeys: []common.Hash{executionPlanHash},
						},
					},
				}),
			},
			Flags:    flags,
			Earliest: 12,
			Latest:   34,
		}

		unpacked, err := decode(bundle.Encode())
		require.NoError(t, err)

		for i, tx := range bundle.Transactions {
			require.Equal(t, tx.Hash(), unpacked.Transactions[i].Hash())
		}
		require.Equal(t, bundle.Flags, unpacked.Flags)
		require.Equal(t, bundle.Earliest, unpacked.Earliest)
		require.Equal(t, bundle.Latest, unpacked.Latest)
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
			bundle := TransactionBundle{}

			_, err := decode(encodeInternal(test.version, bundle))
			require.ErrorContains(t, err, test.expectedError)
		})
	}
}

func TestDecode_ReturnsErrorForInvalidData(t *testing.T) {
	_, err := decode([]byte{0x01, 0x02, 0x03})
	require.ErrorContains(t, err, "failed to decode transaction bundle")

	_, err = decode(nil)
	require.ErrorContains(t, err, "failed to decode transaction bundle")
}
