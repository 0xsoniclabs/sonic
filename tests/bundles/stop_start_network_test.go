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

	"github.com/0xsoniclabs/sonic/api/sonicapi"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBundle_GetBundleInfoSurvivesNetworkRestart(t *testing.T) {
	t.Run("Restart", func(t *testing.T) {
		net := GetIntegrationTestNetWithBundlesEnabled(t)
		testBundle_GetBundleInfoSurvivesNetworkRestart(t, net, net.Restart)
	})

	t.Run("RestartWithExportImport", func(t *testing.T) {
		net := GetIntegrationTestNetWithBundlesEnabled(t)
		testBundle_GetBundleInfoSurvivesNetworkRestart(t, net, net.RestartWithExportImport)
	})
}

func testBundle_GetBundleInfoSurvivesNetworkRestart(t *testing.T, net *tests.IntegrationTestNet, restartMethod func() error) {

	const numBundles = 10

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())
	senders := tests.MakeAccountsWithBalance(t, net, numBundles, big.NewInt(1e18))

	block, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	planHashes := make([]common.Hash, numBundles)
	for i := range planHashes {
		// Create and submit a bundle.
		envelope, plan := bundle.NewBuilder().
			WithSigner(signer).
			SetEarliest(block).
			AllOf(Step(t, net, senders[i], &types.AccessListTx{})).
			BuildEnvelopeAndPlan()

		_, err = net.Send(envelope)
		require.NoError(t, err)
		planHashes[i] = plan.Hash()
	}

	net.AdvanceEpoch(t, 2)

	// Wait for the bundle to be executed and record the info.
	infoBefore := make(map[common.Hash]*sonicapi.RPCBundleInfo)
	for i, planHash := range planHashes {
		info, err := WaitForBundleExecution(t.Context(), client.Client(), planHash)
		require.NoError(t, err)
		require.NotZero(t, info.Count, "bundle %d should be executed", i)
		infoBefore[planHash] = info
	}

	// Restart the network.
	client.Close()
	err = restartMethod()
	require.NoError(t, err, "network restart should succeed")

	// Query bundle info again after restart.
	client, err = net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	for planHash, expectedInfo := range infoBefore {

		infoAfter, err := GetBundleInfo(t.Context(), client.Client(), planHash)
		require.NoError(t, err)

		// The bundle info should be identical before and after restart.
		require.Equal(t, expectedInfo, infoAfter,
			"bundle info should be the same after network restart")
	}
}
