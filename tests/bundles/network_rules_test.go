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
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBundles_BundleOnlyTransactionsAreNotExecutedStartingFromBrio(t *testing.T) {
	brioAndBundlesUpgrades := opera.GetBrioUpgrades()
	brioAndBundlesUpgrades.TransactionBundles = true

	testCases := map[string]struct {
		upgrades  opera.Upgrades
		processed bool
	}{
		"pre-Brio": {
			upgrades:  opera.GetAllegroUpgrades(),
			processed: true,
		},
		"Brio": {
			upgrades:  opera.GetBrioUpgrades(),
			processed: false,
		},
		"Brio and Bundles": {
			upgrades:  brioAndBundlesUpgrades,
			processed: false,
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
				Upgrades: &test.upgrades,
			})
			client, err := net.GetClient()
			require.NoError(t, err)
			defer client.Close()

			sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
			recipient := tests.NewAccount()
			recipientAddr := recipient.Address()
			blockNumber, err := client.BlockNumber(t.Context())
			require.NoError(t, err)
			balance, err := client.BalanceAt(t.Context(), recipientAddr, big.NewInt(int64(blockNumber)))
			require.NoError(t, err)
			require.Equal(t, uint64(0), balance.Uint64(),
				"recipient should have 0 balance before transaction")

			// Create a bundle-only transaction (has BundleOnly marker in access list)
			// which transfers some funds.
			bundleOnlyTx := tests.CreateTransaction(t, net, &types.AccessListTx{
				To:    &recipientAddr,
				Value: big.NewInt(100),
				AccessList: types.AccessList{{
					Address:     bundle.BundleOnly,
					StorageKeys: []common.Hash{},
				}},
			}, sender)
			require.True(t, bundle.IsBundleOnly(bundleOnlyTx))

			txHash, err := net.Send(bundleOnlyTx)
			require.NoError(t, err)
			if test.processed {
				// The transaction should be processed and recipient should receive the funds.
				receipt, err := net.GetReceipt(txHash)
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status,
					"transaction should succeed")

				balance, err = client.BalanceAt(t.Context(), recipientAddr, receipt.BlockNumber)
				require.NoError(t, err)
				require.Equal(t, uint64(100), balance.Uint64(),
					"recipient should receive 100")
			} else {
				// Submit a normal tx from a different account to advance at least one block.
				other := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
				normalTx := tests.CreateTransaction(t, net, &types.AccessListTx{}, other)
				normalHash, err := net.Send(normalTx)
				require.NoError(t, err)
				_, err = net.GetReceipt(normalHash)
				require.NoError(t, err)

				// After a block advanced, the bundle-only tx should not have been executed.
				blockNumber, err = client.BlockNumber(t.Context())
				require.NoError(t, err)
				balance, err = client.BalanceAt(t.Context(), recipientAddr, big.NewInt(int64(blockNumber)))
				require.NoError(t, err)
				require.Equal(t, uint64(0), balance.Uint64(),
					"recipient should still have 0 balance")
			}
		})
	}
}

func TestBundles_BundlesCanBeEnabledAndDisabledStartingFromBrio(t *testing.T) {
	// Start with Brio but bundles initially disabled.
	net := tests.StartIntegrationTestNetWithFakeGenesis(t,
		tests.IntegrationTestNetOptions{
			Upgrades: tests.AsPointer(opera.GetBrioUpgrades()),
		},
	)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())

	// Helper to build a fresh bundle envelope using new funded accounts.
	buildEnvelope := func(t *testing.T) *types.Transaction {
		t.Helper()
		senderA := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
		block, err := client.BlockNumber(t.Context())
		require.NoError(t, err)
		return bundle.NewBuilder().
			WithSigner(signer).
			SetEarliest(block).
			AllOf(Step(t, net, senderA, &types.AccessListTx{})).
			Build()
	}

	// Verify that bundles are initially disabled.
	rules := tests.GetNetworkRules(t, net)
	require.True(t, rules.Upgrades.Brio, "Brio should be enabled")
	require.False(t, rules.Upgrades.TransactionBundles,
		"TransactionBundles should be disabled by default with Brio")

	envelope := buildEnvelope(t)
	_, err = net.Send(envelope)
	require.ErrorContains(t, err, "bundled transactions are disabled")

	// Enable bundles via network rules update.
	type rulesType struct {
		Upgrades struct{ TransactionBundles bool }
	}
	tests.UpdateNetworkRules(t, net, rulesType{
		Upgrades: struct{ TransactionBundles bool }{TransactionBundles: true},
	})
	net.AdvanceEpoch(t, 1)

	rules = tests.GetNetworkRules(t, net)
	require.True(t, rules.Upgrades.TransactionBundles,
		"TransactionBundles should be enabled after rules update")

	envelope = buildEnvelope(t)
	_, err = net.Send(envelope)
	require.NoError(t, err, "bundle envelopes should be accepted when bundles are enabled")

	// Disable bundles again via network rules update.
	tests.UpdateNetworkRules(t, net, rulesType{
		Upgrades: struct{ TransactionBundles bool }{TransactionBundles: false},
	})
	net.AdvanceEpoch(t, 1)

	rules = tests.GetNetworkRules(t, net)
	require.False(t, rules.Upgrades.TransactionBundles,
		"TransactionBundles should be disabled after rules update")

	envelope = buildEnvelope(t)
	_, err = net.Send(envelope)
	require.ErrorContains(t, err, "bundled transactions are disabled")
}
