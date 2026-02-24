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

package bundles

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestCreateAccessList_PreservesBundleOnlyMarker verifies that eth_createAccessList
// does not strip the bundle.BundleOnly sentinel address from the access list.
// BundleOnly marks a transaction as bundle-only (must execute within a bundle,
// not standalone). The RPC must preserve it so the bundle constraint survives
// access-list recreation by wallets or tooling.
func TestCreateAccessList_PreservesBundleOnlyMarker(t *testing.T) {
	t.Parallel()

	session := tests.StartIntegrationTestNet(t)

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	sender := tests.MakeAccountWithBalance(t, session, big.NewInt(1e18))

	// callCreateAccessList is a helper that calls eth_createAccessList with the
	// given access list and returns the resulting access list.
	callCreateAccessList := func(t *testing.T, acl *types.AccessList) types.AccessList {
		t.Helper()
		rpcTx := ethapi.TransactionArgs{
			From:       tests.AsPointer(sender.Address()),
			To:         &common.Address{0x42},
			Value:      (*hexutil.Big)(big.NewInt(1)),
			AccessList: acl,
		}
		// mirrors the unexported ethapi.accessListResult
		type accessListResult struct {
			AccessList *types.AccessList `json:"accessList"`
			Error      string            `json:"error,omitempty"`
			GasUsed    hexutil.Uint64    `json:"gasUsed"`
		}
		var result accessListResult
		require.NoError(t, client.Client().Call(&result, "eth_createAccessList", rpcTx, "latest"))
		require.NotNil(t, result.AccessList)
		return *result.AccessList
	}

	t.Run("BundleOnly without storage keys is preserved", func(t *testing.T) {
		result := callCreateAccessList(t, &types.AccessList{
			{Address: bundle.BundleOnly, StorageKeys: []common.Hash{}},
		})
		require.True(t, accessListContainsAddress(result, bundle.BundleOnly),
			"BundleOnly must remain in access list")
	})

	t.Run("BundleOnly with execution plan hash is preserved", func(t *testing.T) {
		planHash := common.Hash{0xde, 0xad, 0xbe, 0xef}
		result := callCreateAccessList(t, &types.AccessList{
			{Address: bundle.BundleOnly, StorageKeys: []common.Hash{planHash}},
		})
		require.True(t, accessListContainsStorageKey(result, bundle.BundleOnly, planHash),
			"BundleOnly with execution plan hash must remain in access list with its storage key")
	})

	t.Run("BundleOnly not injected when absent from input", func(t *testing.T) {
		result := callCreateAccessList(t, &types.AccessList{})
		require.False(t, accessListContainsAddress(result, bundle.BundleOnly),
			"BundleOnly must not appear in access list if not provided by caller")
	})

	t.Run("BundleOnly not injected when access list is nil", func(t *testing.T) {
		result := callCreateAccessList(t, nil)
		require.False(t, accessListContainsAddress(result, bundle.BundleOnly),
			"BundleOnly must not appear in access list if nil")
	})
}

// accessListContainsAddress reports whether list has an entry for addr.
func accessListContainsAddress(list types.AccessList, addr common.Address) bool {
	for _, entry := range list {
		if entry.Address == addr {
			return true
		}
	}
	return false
}

// accessListContainsStorageKey reports whether list has an entry for addr that
// includes key among its storage keys.
func accessListContainsStorageKey(list types.AccessList, addr common.Address, key common.Hash) bool {
	for _, entry := range list {
		if entry.Address != addr {
			continue
		}
		for _, k := range entry.StorageKeys {
			if k == key {
				return true
			}
		}
	}
	return false
}
