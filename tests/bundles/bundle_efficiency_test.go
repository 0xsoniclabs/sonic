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
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBundle_BundlesWithTooLowEfficiencyAreRejected(t *testing.T) {

	t.Run("by the pool", func(t *testing.T) {
		net := GetIntegrationTestNetWithBundlesEnabled(t)
		testBundle_BundlesWithTooLowEfficiencyAreRejected(t, net, expectRejectedByPool)
	})

	t.Run("by the emitter", func(t *testing.T) {
		upgrades := opera.GetBrioUpgrades()
		upgrades.TransactionBundles = true

		net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
			Upgrades: &upgrades,
			ClientExtraArguments: []string{
				"--disable-txPool-validation",
			},
		})
		testBundle_BundlesWithTooLowEfficiencyAreRejected(t, net, expectRejectedByEmitter)
	})
}

func testBundle_BundlesWithTooLowEfficiencyAreRejected(t *testing.T, net *tests.IntegrationTestNet,
	expectRejected func(t *testing.T, net *tests.IntegrationTestNet, envelope *types.Transaction),
) {

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())

	// Bundle structure: OneOf(AllOf(invalidTx1, ..., invalidTxN), validTx))
	// The invalid transactions are wrapped in an AllOf step, so their state
	// changes are reverted and there are no receipts for them, but their gas
	// cost is still counted toward the bundle execution cost.
	// The outer OneOf keeps executing the steps because they are all invalid,
	// until the last step with the valid transaction is executed, which
	// produces a successful receipt. The bundle is then accepted or rejected
	// based on the overall efficiency, which depends on the number of invalid
	// transactions.
	cases := map[string]struct {
		InvalidTxCount       int
		ExpectBelowThreshold bool
	}{
		"Bundle with too low efficiency": {
			InvalidTxCount:       10,
			ExpectBelowThreshold: true,
		},
		"Bundle with efficiency above threshold": {
			InvalidTxCount:       1,
			ExpectBelowThreshold: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			senders := tests.MakeAccountsWithBalance(t, net, tc.InvalidTxCount+1, big.NewInt(1e18))

			blockNumber, err := client.BlockNumber(t.Context())
			require.NoError(t, err)

			steps := make([]bundle.BuilderStep, 0, tc.InvalidTxCount+1)
			for i := range tc.InvalidTxCount {
				// The invalid transactions need to pass the pre-execution
				// checks, in order to count towards the execution cost.
				steps = append(steps, bundle.AllOf(Step(t, net, senders[i], &types.AccessListTx{
					Data: []byte{0xfe}, // INVALID opcode as init code
					Gas:  60_000,
				})))
			}
			// The valid transaction.
			steps = append(steps, Step(t, net, senders[tc.InvalidTxCount], &types.AccessListTx{}))

			envelope, bundle, plan := bundle.NewBuilder().
				WithSigner(signer).
				SetEarliest(blockNumber).
				With(bundle.OneOf(steps...)).
				BuildEnvelopeBundleAndPlan()

			execGas := uint64(0)
			for _, tx := range bundle.GetTransactionsInReferencedOrder() {
				execGas += tx.Gas()
			}
			usedGas := bundle.GetTransactionsInReferencedOrder()[tc.InvalidTxCount].Gas()

			if tc.ExpectBelowThreshold {
				require.Less(t, float64(usedGas)/float64(execGas), evmcore.MinBundleEfficiency)
				expectRejected(t, net, envelope)
			} else {

				require.GreaterOrEqual(t, float64(usedGas)/float64(execGas), evmcore.MinBundleEfficiency)

				_, err = net.Send(envelope)
				require.NoError(t, err)

				// Wait for the bundle to be processed.
				info, err := WaitForBundleExecution(t.Context(), client.Client(), plan.Hash())
				require.NoError(t, err)

				bundleTxs := bundle.GetTransactionsInReferencedOrder()
				blockTxsHashes := getBlockTxsHashes(t, client, big.NewInt(info.Block.Int64()))

				successfulTxHash := bundleTxs[tc.InvalidTxCount].Hash()
				require.EqualValues(t, info.Count, 1)
				require.Contains(t, blockTxsHashes, successfulTxHash)

				receipt, err := client.TransactionReceipt(t.Context(), successfulTxHash)
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
			}
		})
	}
}

func expectRejectedByPool(t *testing.T, net *tests.IntegrationTestNet, envelope *types.Transaction) {
	t.Helper()
	_, err := net.Send(envelope)
	require.ErrorContains(t, err, "bundle trial-run failed")
}

func expectRejectedByEmitter(t *testing.T, net *tests.IntegrationTestNet, envelope *types.Transaction) {
	t.Helper()
	_, err := net.Send(envelope)
	require.NoError(t, err, "bundle trial-run failed")

	timedCtx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	_, err = WaitForBundleExecution(timedCtx, client.Client(), envelope.Hash())
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
