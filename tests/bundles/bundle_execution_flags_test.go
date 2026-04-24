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
	"slices"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBundle_ExecutionFlagsOfSingleTxAreInterpretedCorrectly(t *testing.T) {

	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
		// This tests submits clearly invalid bundles. TxPool will reject
		// them directly, to test that they are correctly ignored, pool
		// validation needs to be disabled.
		ClientExtraArguments: []string{"--disable-txPool-validation"},
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())

	revertAddress, revertInput := tests.MustDeployRevertContractAndGetMethodCallParameters(t, net)

	senders := tests.MakeAccountsWithBalance(t, net, 2, big.NewInt(1e18))

	successfulTx := types.AccessListTx{}
	failingTx := types.AccessListTx{
		To:   &revertAddress,
		Gas:  1_000_000,
		Data: revertInput,
	}
	invalidTx := types.AccessListTx{
		Gas: 1, // insufficient gas
	}

	cases := map[string]struct {
		tx    types.AccessListTx
		flags bundle.ExecutionFlags
		// Whether the whole bundle is expected to be rolled back. If this is
		// the case, further expectations are ignored.
		expectRollback bool
		// The bundle contains case.tx and a successful transaction. This flag
		// sets the expectation for the first transaction only.
		expectInBlock bool
	}{
		"Default/SuccessfulTx": {
			tx:             successfulTx,
			flags:          bundle.EF_Default,
			expectRollback: false,
			expectInBlock:  true,
		},
		"Default/FailingTx": {
			tx:             failingTx,
			flags:          bundle.EF_Default,
			expectRollback: true,
		},
		"Default/InvalidTx": {
			tx:             invalidTx,
			flags:          bundle.EF_Default,
			expectRollback: true,
		},
		"TolerateInvalid/SuccessfulTx": {
			tx:             successfulTx,
			flags:          bundle.EF_TolerateInvalid,
			expectRollback: false,
			expectInBlock:  true,
		},
		"TolerateInvalid/FailingTx": {
			tx:             failingTx,
			flags:          bundle.EF_TolerateInvalid,
			expectRollback: true,
		},
		"TolerateInvalid/InvalidTx": {
			tx:             invalidTx,
			flags:          bundle.EF_TolerateInvalid,
			expectRollback: false,
			expectInBlock:  false,
		},
		"TolerateFailed/SuccessfulTx": {
			tx:             successfulTx,
			flags:          bundle.EF_TolerateFailed,
			expectRollback: false,
			expectInBlock:  true,
		},
		"TolerateFailed/FailingTx": {
			tx:             failingTx,
			flags:          bundle.EF_TolerateFailed,
			expectRollback: false,
			expectInBlock:  true,
		},
		"TolerateFailed/InvalidTx": {
			tx:             invalidTx,
			flags:          bundle.EF_TolerateFailed,
			expectRollback: true,
		},
		"TolerateInvalidTolerateFailed/SuccessfulTx": {
			tx:             successfulTx,
			flags:          bundle.EF_TolerateInvalid | bundle.EF_TolerateFailed,
			expectRollback: false,
			expectInBlock:  true,
		},
		"TolerateInvalidTolerateFailed/FailingTx": {
			tx:             failingTx,
			flags:          bundle.EF_TolerateInvalid | bundle.EF_TolerateFailed,
			expectRollback: false,
			expectInBlock:  true,
		},
		"TolerateInvalidTolerateFailed/InvalidTx": {
			tx:             invalidTx,
			flags:          bundle.EF_TolerateInvalid | bundle.EF_TolerateFailed,
			expectRollback: false,
			expectInBlock:  false,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			blockNumber, err := client.BlockNumber(t.Context())
			require.NoError(t, err)

			// Create the bundle: AllOf([c.flags]c.tx, successfulTx)
			// The second transaction is needed for the cases with an invalid
			// transaction to check whether it was tolerated or not.
			envelope, bundle, plan := bundle.NewBuilder().
				WithSigner(signer).
				SetEarliest(blockNumber).
				AllOf(
					Step(t, net, senders[0], &c.tx).WithFlags(c.flags),
					Step(t, net, senders[1], &successfulTx),
				).
				BuildEnvelopeBundleAndPlan()

			// Send the bundle.
			_, err = net.Send(envelope)
			require.NoError(t, err)

			if c.expectRollback {
				timeoutCtx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
				defer cancel()

				_, err := WaitForBundleExecution(timeoutCtx, client.Client(), plan.Hash())
				require.ErrorIs(t, err, context.DeadlineExceeded)
				return
			}

			// Wait for the bundle to be processed.
			info, err := WaitForBundleExecution(t.Context(), client.Client(), plan.Hash())
			require.NoError(t, err)

			blockTxsHashes := getBlockTxsHashes(t, client, big.NewInt(info.Block.Int64()))
			bundleTxs := bundle.GetTransactionsInReferencedOrder()

			// If the transaction is expected to not be included in the block,
			// only the successful transaction that follows it should be included.
			if !c.expectInBlock {
				require.Equal(t, 1, int(info.Count))
				require.NotContains(t, blockTxsHashes, bundleTxs[0].Hash())
				require.Contains(t, blockTxsHashes, bundleTxs[1].Hash())
				return
			}

			// The transactions itself and the successful transaction that
			// follows it should be included.
			require.Equal(t, 2, int(info.Count))
			require.Contains(t, blockTxsHashes, bundleTxs[0].Hash())
			require.Contains(t, blockTxsHashes, bundleTxs[1].Hash())
		})
	}
}

func TestBundle_AllOfGroupSucceedsIfAllStepsTolerated(t *testing.T) {
	net := GetIntegrationTestNetWithBundlesEnabled(t)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())

	senders := tests.MakeAccountsWithBalance(t, net, 4, big.NewInt(1e18))

	revertAddress, revertInput := tests.MustDeployRevertContractAndGetMethodCallParameters(t, net)

	successfulTx := types.AccessListTx{}
	failingTx := types.AccessListTx{
		To:   &revertAddress,
		Gas:  1_000_000,
		Data: revertInput,
	}

	cases := map[string]struct {
		tolerateFailed bool
		secondTx       types.AccessListTx
		expectInBlock  []bool
	}{
		"Default/SuccessfulStep": {
			tolerateFailed: false,
			secondTx:       successfulTx,
			expectInBlock:  []bool{true, true, true, true},
		},
		"Default/FailingStep": {
			tolerateFailed: false,
			// It does not matter whether the transaction fails or is invalid,
			// since it will be reported as non-tolerated in both cases.
			secondTx:      failingTx,
			expectInBlock: []bool{false, false, false, true},
		},
		"TolerateFailed/SuccessfulStep": {
			tolerateFailed: true,
			secondTx:       successfulTx,
			expectInBlock:  []bool{true, true, true, true},
		},
		"TolerateFailed/FailingStep": {
			tolerateFailed: true,
			// It does not matter whether the transaction fails or is invalid,
			// since it will be reported as non-tolerated in both cases.
			secondTx:      failingTx,
			expectInBlock: []bool{false, false, true, true},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			blockNumber, err := client.BlockNumber(t.Context())
			require.NoError(t, err)

			flags := bundle.EF_Default
			if c.tolerateFailed {
				flags = bundle.EF_TolerateFailed
			}

			// The bundle structure adds transactions and groups
			// to guarantee that the bundle yields observable results
			envelope, bundle, plan := bundle.NewBuilder().
				WithSigner(signer).
				SetEarliest(blockNumber).
				With(
					bundle.AllOf(
						bundle.AllOf(
							// This is the group under test
							bundle.AllOf(
								// This transaction is needed to check the AllOf
								// does not stop after first successful step.
								// It also serves to check that the transaction
								// is reverted if a later tx fails
								Step(t, net, senders[0], &successfulTx),
								// Input transaction defining the failing condition
								// of the group under test
								Step(t, net, senders[1], &c.secondTx),
							).WithFlags(flags),
							// This tx is needed to detect when the group under
							// test was tolerated
							Step(t, net, senders[2], &successfulTx),
						).WithFlags(bundle.EF_TolerateFailed),
						// This tx is needed to produce observable results
						// in case the group under test was not tolerated
						Step(t, net, senders[3], &successfulTx),
					),
				).
				BuildEnvelopeBundleAndPlan()

			// Send the bundle.
			_, err = net.Send(envelope)
			require.NoError(t, err)

			// Wait for the bundle to be processed.
			info, err := WaitForBundleExecution(t.Context(), client.Client(), plan.Hash())
			require.NoError(t, err)

			bundleTxs := bundle.GetTransactionsInReferencedOrder()
			blockTxsHashes := getBlockTxsHashes(t, client, big.NewInt(info.Block.Int64()))

			expectedCount := 0
			for i, tx := range bundleTxs {
				require.Equal(t, c.expectInBlock[i], slices.Contains(blockTxsHashes, tx.Hash()))
				if c.expectInBlock[i] {
					require.Equal(t, blockTxsHashes[int(info.Position)+expectedCount], tx.Hash())
					expectedCount++
				}
			}
			require.Equal(t, expectedCount, int(info.Count))
		})
	}
}

func TestBundle_OneOfGroupSucceedsOnFirstToleratedStep(t *testing.T) {
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
		// This tests submits clearly invalid bundles. TxPool will reject
		// them directly, to test that they are correctly ignored, pool
		// validation needs to be disabled.
		ClientExtraArguments: []string{"--disable-txPool-validation"},
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())

	senders := tests.MakeAccountsWithBalance(t, net, 3, big.NewInt(1e18))

	revertAddress, revertInput := tests.MustDeployRevertContractAndGetMethodCallParameters(t, net)

	successfulTx := types.AccessListTx{}
	failingTx := types.AccessListTx{
		To:   &revertAddress,
		Gas:  1_000_000,
		Data: revertInput,
	}

	// The bundle structure is: AllOf(OneOf[flags](firstTx, secondTx), successfulTx)
	// The group under test is the inner OneOf. The trailing successful
	// transaction in the outer group is needed to check whether the inner
	// group was tolerated or not.
	// expectInBlock is indexed by the position of the transaction in the
	// bundle, in reference order.
	cases := map[string]struct {
		tolerateFailed bool
		firstTx        types.AccessListTx
		secondTx       types.AccessListTx
		expectInBlock  [3]bool
	}{
		"Default/FirstTolerated": {
			tolerateFailed: false,
			firstTx:        successfulTx,
			secondTx:       successfulTx,
			expectInBlock:  [3]bool{true, false, true},
		},
		"Default/SecondTolerated": {
			tolerateFailed: false,
			firstTx:        failingTx,
			secondTx:       successfulTx,
			expectInBlock:  [3]bool{true, true, true},
		},
		"Default/OnlyNonTolerated": {
			// Note: this test does not yield anything observable, timeout expected
			tolerateFailed: false,
			firstTx:        failingTx,
			secondTx:       failingTx,
			expectInBlock:  [3]bool{false, false, false},
		},
		"TolerateFailed/FirstTolerated": {
			tolerateFailed: true,
			firstTx:        successfulTx,
			secondTx:       successfulTx,
			expectInBlock:  [3]bool{true, false, true},
		},
		"TolerateFailed/SecondTolerated": {
			tolerateFailed: true,
			firstTx:        failingTx,
			secondTx:       successfulTx,
			expectInBlock:  [3]bool{true, true, true},
		},
		"TolerateFailed/OnlyNonTolerated": {
			tolerateFailed: true,
			firstTx:        failingTx,
			secondTx:       failingTx,
			expectInBlock:  [3]bool{false, false, true},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			blockNumber, err := client.BlockNumber(t.Context())
			require.NoError(t, err)

			flags := bundle.EF_Default
			if c.tolerateFailed {
				flags = bundle.EF_TolerateFailed
			}

			envelope, bundle, plan := bundle.NewBuilder().
				WithSigner(signer).
				SetEarliest(blockNumber).
				AllOf(
					bundle.OneOf(
						Step(t, net, senders[0], &c.firstTx),
						Step(t, net, senders[1], &c.secondTx),
					).WithFlags(flags),
					Step(t, net, senders[2], &successfulTx),
				).
				BuildEnvelopeBundleAndPlan()

			// Send the bundle.
			_, err = net.Send(envelope)
			require.NoError(t, err)

			if !slices.Contains(c.expectInBlock[:], true) {
				timeoutCtx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
				defer cancel()

				_, err := WaitForBundleExecution(timeoutCtx, client.Client(), plan.Hash())
				require.ErrorIs(t, err, context.DeadlineExceeded)

				return
			}

			// Wait for the bundle to be processed.
			info, err := WaitForBundleExecution(t.Context(), client.Client(), plan.Hash())
			require.NoError(t, err)

			bundleTxs := bundle.GetTransactionsInReferencedOrder()
			blockTxsHashes := getBlockTxsHashes(t, client, big.NewInt(info.Block.Int64()))

			expectedCount := 0
			for i, tx := range bundleTxs {
				require.Equal(t, c.expectInBlock[i], slices.Contains(blockTxsHashes, tx.Hash()))
				if c.expectInBlock[i] {
					require.Equal(t, blockTxsHashes[int(info.Position)+expectedCount], tx.Hash())
					expectedCount++
				}
			}
			require.Equal(t, expectedCount, int(info.Count))
		})
	}
}
