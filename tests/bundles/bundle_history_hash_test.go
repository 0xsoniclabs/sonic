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
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// TestBundle_HistoryHash_IsZeroBeforeExecution_AndUpdatesCorrectlyAfter verifies
// the end-to-end behaviour of sonic_getBundleHistoryHash:
//  1. The hash is zero before any bundle is executed.
//  2. After a bundle executes at block B, the hash matches the formula
//     newHash = Keccak256(oldHash || executionPlanHash || blockNum)
//     for every block from B up to the block reported by the latest query.
func TestBundle_HistoryHash_IsZeroBeforeExecution_AndUpdatesCorrectlyAfter(t *testing.T) {
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	// Step 1: Query the history hash before any bundle is executed.
	// It must be zero.
	initial, err := GetBundleHistoryHash(t.Context(), client.Client())
	require.NoError(t, err)
	require.NotNil(t, initial)
	require.Equal(t, uint64(0), uint64(initial.Block), "block should be zero before any bundle")
	require.Equal(t, common.Hash{}, initial.Hash, "hash should be zero before any bundle")

	// Step 2: Create one account and submit a single bundle.
	recipient := tests.NewAccount()

	block, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	signer := types.LatestSignerForChainID(net.GetChainId())
	recipientAddr := recipient.Address()

	envelope, _, plan := bundle.NewBuilder().
		WithSigner(signer).
		SetEarliest(block).
		AllOf(
			Step(t, net, sender, &types.AccessListTx{
				To:    &recipientAddr,
				Value: big.NewInt(1),
			}),
		).
		BuildEnvelopeBundleAndPlan()

	require.NoError(t, client.SendTransaction(t.Context(), envelope))

	// Wait until the bundle is executed and we know which block it landed in.
	info, err := WaitForBundleExecution(t.Context(), client.Client(), plan.Hash())
	require.NoError(t, err)
	bundleBlock := uint64(info.Block)
	executionPlanHash := plan.Hash()

	// Step 3: Query the latest history hash from the node.
	latest, err := GetBundleHistoryHash(t.Context(), client.Client())
	require.NoError(t, err)
	require.NotNil(t, latest)
	require.NotEqual(t, common.Hash{}, latest.Hash, "history hash must be non-zero after bundle execution")

	latestBlock := uint64(latest.Block)
	require.GreaterOrEqual(t, latestBlock, bundleBlock,
		"latest reported block must be at or after the bundle's block")

	// Step 4: Recompute the expected hash using the formula and compare.
	//
	// The formula is applied from bundleBlock (when hash first becomes non-zero)
	// up to latestBlock (as reported by the node). For the bundle block the
	// addedHash equals executionPlanHash (XOR of a single hash = itself). For
	// every subsequent block with no bundles the addedHash is zero.
	expectedHash := computeExpectedHistoryHash(bundleBlock, latestBlock, executionPlanHash)
	require.Equal(t, expectedHash, latest.Hash,
		"history hash must match the formula from block %d to %d", bundleBlock, latestBlock)
}

// computeExpectedHistoryHash recomputes the processed-bundle history hash from
// scratch, given that the first (and only) bundle was included at bundleBlock
// with the given executionPlanHash, and blocks up to latestBlock had no further
// bundles.
//
// Formula per block:
//
//	newHash = Keccak256(oldHash || addedExecPlanHash || blockNum)
//
// where addedExecPlanHash is the XOR of plan hashes executed in that block
// (executionPlanHash for bundleBlock, zero for every later block).
func computeExpectedHistoryHash(bundleBlock, latestBlock uint64, executionPlanHash common.Hash) common.Hash {
	h := common.Hash{}
	for blockNum := bundleBlock; blockNum <= latestBlock; blockNum++ {
		addedHash := common.Hash{}
		if blockNum == bundleBlock {
			addedHash = executionPlanHash
		}

		var data []byte
		data = append(data, h.Bytes()...)
		data = append(data, addedHash.Bytes()...)
		data = binary.BigEndian.AppendUint64(data, blockNum)
		h = common.Hash(crypto.Keccak256(data))
	}
	return h
}
