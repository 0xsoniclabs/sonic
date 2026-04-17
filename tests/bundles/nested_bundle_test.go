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
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBundle_NestedBundlesCanBeExecuted(t *testing.T) {
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())

	sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	blockNumber, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	tx := tests.SetTransactionDefaults(t, net, &types.AccessListTx{}, sender)

	innerEnvelope, innerBundle, _ := bundle.NewBuilder().
		WithSigner(signer).
		SetEarliest(blockNumber).
		AllOf(bundle.Step(sender.PrivateKey, tx)).
		BuildEnvelopeBundleAndPlan()

	outerEnvelope, _, outerPlan := bundle.NewBuilder().
		WithSigner(signer).
		SetEarliest(blockNumber).
		AllOf(bundle.Step(sender.PrivateKey, innerEnvelope)).
		BuildEnvelopeBundleAndPlan()

	// Check bundle status before submission.
	_, err = GetBundleInfo(t.Context(), client.Client(), outerPlan.Hash())
	require.ErrorIs(t, err, ethereum.NotFound)

	// Run the bundle.
	require.NoError(t, client.SendTransaction(t.Context(), outerEnvelope))

	// Wait for the bundle to be processed.
	info, err := WaitForBundleExecution(t.Context(), client.Client(), outerPlan.Hash())
	require.NoError(t, err)

	// Verify that there is no receipt for the envelopes themselves.
	_, err = client.TransactionReceipt(t.Context(), outerEnvelope.Hash())
	require.ErrorIs(t, err, ethereum.NotFound)
	_, err = client.TransactionReceipt(t.Context(), innerEnvelope.Hash())
	require.ErrorIs(t, err, ethereum.NotFound)

	blockTxsHashes := getBlockTxsHashes(t, client, big.NewInt(info.Block.Int64()))

	bundleTxs := innerBundle.GetTransactionsInReferencedOrder()

	require.Equal(t, 1, int(info.Count))
	require.Contains(t, blockTxsHashes, bundleTxs[0].Hash())
}
