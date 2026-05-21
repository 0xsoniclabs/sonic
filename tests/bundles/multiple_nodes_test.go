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

func TestBundle_GetBundleInfoIsConsistentAcrossMultipleNodes(t *testing.T) {

	// This test checks that the bundle store is correctly replicated across multiple nodes.

	const numNodes = 3
	const numBundles = 15
	const bundlesPerEpoch = 5

	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
		NumNodes: numNodes,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())

	// Submit a series of bundles and collect their execution plan hashes.
	planHashes := make([]common.Hash, numBundles)
	for i := range numBundles {
		sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
		block, err := client.BlockNumber(t.Context())
		require.NoError(t, err)

		envelope, plan := bundle.NewBuilder().
			WithSigner(signer).
			SetEarliest(block).
			AllOf(Step(t, net, sender, &types.AccessListTx{})).
			BuildEnvelopeAndPlan()

		_, err = net.Send(envelope)
		require.NoError(t, err, "failed to send bundle %d", i)
		planHashes[i] = plan.Hash()

		if i%bundlesPerEpoch == bundlesPerEpoch-1 {
			net.AdvanceEpoch(t, 1)
		}
	}

	// Wait for all bundles to be executed (using node 0).
	expectedInfos, err := WaitForBundleExecutions(t.Context(), client.Client(), planHashes)
	require.NoError(t, err)
	for i, info := range expectedInfos {
		require.NotZero(t, info.Count, "bundle %d should be executed", i)
	}

	// Verify that every node reports the same bundle info.
	for node := 1; node < numNodes; node++ {
		nodeClient, err := net.GetClientConnectedToNode(node)
		require.NoError(t, err)
		defer nodeClient.Close()

		for i, planHash := range planHashes {
			nodeInfo, err := GetBundleInfo(t.Context(), nodeClient.Client(), planHash)
			require.NoError(t, err, "node %d: failed to get info for bundle %d", node, i)
			require.Equal(t, expectedInfos[i], nodeInfo,
				"node %d: bundle info for bundle %d differs from node 0", node, i)
		}
	}
}
