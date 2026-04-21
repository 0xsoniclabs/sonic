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

func TestBundle_StressWithManyNonceBlockedBundles(t *testing.T) {
	// Increase this number for profiling to increase load on the system.
	const N = 2 // Number of blocked bundles

	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())

	// Create all needed accounts and endow in parallel.
	account := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	envelopes := make([]*types.Transaction, N+1)
	bundles := make([]*bundle.TransactionBundle, N+1)
	planHashes := make([]common.Hash, N+1)

	// Create N+1 bundles with transactions with increasing nonces.
	for i := range N + 1 {
		envelope, bundle, plan := bundle.NewBuilder().
			WithSigner(signer).
			AllOf(Step(t, net, account, &types.AccessListTx{Nonce: uint64(i)})).
			BuildEnvelopeBundleAndPlan()

		envelopes[i] = envelope
		bundles[i] = bundle
		planHashes[i] = plan.Hash()
	}

	// Iteratively send in all bundles except the first one (with nonce 0)
	// which will be blocked until the transaction with nonce 0 is executed.
	for i := range N {
		_, err = net.Send(envelopes[i+1])
		require.NoError(t, err)
	}

	// Send the bundle containing the transaction with nonce 0 which unblocks
	// all the other bundles.
	_, err = net.Send(envelopes[0])
	require.NoError(t, err)

	// Wait for all bundles to be processed.
	infos, err := WaitForBundleExecutions(t.Context(), client.Client(), planHashes)
	require.NoError(t, err)

	// Check that all obtained infos match the respective transactions.
	for i, info := range infos {
		require.EqualValues(t, len(bundles[i].Transactions), info.Count)

		for j, tx := range bundles[i].GetTransactionsInReferencedOrder() {
			receipt, err := client.TransactionReceipt(t.Context(), tx.Hash())
			require.NoError(t, err)
			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
			require.Equal(t, int(receipt.BlockNumber.Uint64()), int(info.Block))
			require.Equal(t, int(receipt.TransactionIndex), int(info.Position)+j)
		}
	}
}

func TestBundle_StressWithManyRollingBackBundles(t *testing.T) {
	// Increase these numbers for profiling to increase load on the system.
	const B = 2 // Number of bundles
	const S = 1 // Number of steps per bundle

	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())

	// Create all needed accounts and endow in parallel.
	accounts := tests.MakeAccountsWithBalance(t, net, B, big.NewInt(1e18))

	envelopes := make([]*types.Transaction, B)
	planHashes := make([]common.Hash, B)

	// Create B bundles that will roll back.
	for i := range B {
		// Create S-1 successful steps and 1 invalid step causing the whole bundle to roll back.
		steps := make([]bundle.BuilderStep, S)
		for j := range S - 1 {
			steps[j] = Step(t, net, accounts[i], &types.AccessListTx{Nonce: uint64(j)})
		}
		steps[S-1] = Step(t, net, accounts[i], &types.AccessListTx{Nonce: uint64(S - 1), Gas: 1})

		envelope, _, plan := bundle.NewBuilder().
			WithSigner(signer).
			AllOf(steps...).
			BuildEnvelopeBundleAndPlan()

		envelopes[i] = envelope
		planHashes[i] = plan.Hash()
	}

	// Send all bundles.
	_, err = net.SendAll(envelopes)
	require.NoError(t, err)

	// Wait for all bundles to be processed.
	infos, err := WaitForBundleExecutions(t.Context(), client.Client(), planHashes)
	require.NoError(t, err)

	// Check that all bundles were rolled back.
	for _, info := range infos {
		require.Zero(t, info.Count)
	}
}
