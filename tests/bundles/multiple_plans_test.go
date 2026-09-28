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

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBundle_BundleOnlyTxApprovingMultiplePlans_IsRejectedByPool(t *testing.T) {
	net := GetIntegrationTestNetWithBundlesEnabled(t)
	sender := tests.MakeAccountsWithBalance(t, net, 1, big.NewInt(1e18))[0]
	signer := types.LatestSignerForChainID(net.GetChainId())

	cases := map[string]struct {
		accessList types.AccessList
		rejected   bool
	}{
		"single plan": {
			accessList: types.AccessList{
				{Address: bundle.BundleOnly, StorageKeys: []common.Hash{{0x01}}},
			},
		},
		"multiple plans in one entry": {
			accessList: types.AccessList{
				{Address: bundle.BundleOnly, StorageKeys: []common.Hash{{0x01}, {0x02}}},
			},
			rejected: true,
		},
		"multiple plans in separate entries": {
			accessList: types.AccessList{
				{Address: bundle.BundleOnly, StorageKeys: []common.Hash{{0x01}}},
				{Address: bundle.BundleOnly, StorageKeys: []common.Hash{{0x02}}},
			},
			rejected: true,
		},
		"same plan twice in one entry": {
			accessList: types.AccessList{
				{Address: bundle.BundleOnly, StorageKeys: []common.Hash{{0x01}, {0x01}}},
			},
			rejected: true,
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			tx := types.MustSignNewTx(sender.PrivateKey, signer, tests.SetTransactionDefaults(t, net, &types.AccessListTx{
				To:         &common.Address{0x42},
				AccessList: test.accessList,
			}, sender))

			_, err := net.Send(tx)
			if test.rejected {
				require.ErrorContains(t, err, "approves multiple execution plans")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
