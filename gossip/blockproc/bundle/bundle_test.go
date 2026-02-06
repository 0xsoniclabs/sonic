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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
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

func TestTransactionBundle_IdentifiesBundles(t *testing.T) {
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
