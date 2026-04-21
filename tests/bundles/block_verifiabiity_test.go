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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBundle_BlocksWithBundlesCanBeReplayedAndVerified(t *testing.T) {
	net := GetIntegrationTestNetWithBundlesEnabled(t)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())

	senders := tests.MakeAccountsWithBalance(t, net, 3, big.NewInt(1e18))

	blockNumber, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	envelope, _, plan := bundle.NewBuilder().
		WithSigner(signer).
		SetEarliest(blockNumber).
		AllOf(
			bundle.AllOf(
				Step(t, net, senders[0], &types.AccessListTx{}),
				Step(t, net, senders[1], &types.AccessListTx{Gas: 1}),
			).WithFlags(bundle.EF_TolerateFailed),
			Step(t, net, senders[2], &types.AccessListTx{}),
		).
		BuildEnvelopeBundleAndPlan()

	// Send the bundle.
	_, err = net.Send(envelope)
	require.NoError(t, err)

	// Wait for the bundle to be processed.
	info, err := WaitForBundleExecution(t.Context(), client.Client(), plan.Hash())
	require.NoError(t, err)
	require.EqualValues(t, 1, info.Count)

	// Collect all blocks from genesis through the bundle's block.
	blocks := make([]*types.Block, 0, int(info.Block)+1)
	for blockNumber := range int64(info.Block) + 1 {
		block, err := client.BlockByNumber(t.Context(), big.NewInt(blockNumber))
		require.NoError(t, err)
		blocks = append(blocks, block)
	}

	// Re-process the blocks and verify each replayed block has the same hash.
	genesis := net.GetJsonGenesis()
	require.NotNil(t, genesis, "network must be started with JSON genesis")
	tests.VerifyBlocks(t, genesis, blocks)
}
