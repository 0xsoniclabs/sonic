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
	"github.com/0xsoniclabs/sonic/tests/contracts/add"
	"github.com/0xsoniclabs/sonic/tests/contracts/store"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBundle_StressWithManyNonceBlockedBundles(t *testing.T) {
	// Increase this number for profiling to increase load on the system.
	const N = 2 // Number of blocked bundles

	net := GetIntegrationTestNetWithBundlesEnabled(t)

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

	// Send in all bundles except the first one (with nonce 0) which will be
	// blocked until the transaction with nonce 0 is executed.
	_, err = net.SendAll(envelopes[1:])
	require.NoError(t, err)

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

func TestBundle_StressWithExpensiveInternalRollback(t *testing.T) {
	// This test runs `B` bundles of the shape:
	// AllOf(AllOf[TolerateFailed](expensive tx, failing tx), successful tx).

	// Increase this number for profiling to increase load on the system.
	const B = 1 // Number of bundles

	net := GetIntegrationTestNetWithBundlesEnabled(t)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())

	// Create all needed accounts and endow in parallel.
	accounts := tests.MakeAccountsWithBalance(t, net, B*3, big.NewInt(1e18))

	cases := map[string]struct {
		contractAddr  common.Address
		contractInput []byte
	}{
		"ComputeHeavy": {
			contractAddr: tests.MustDeployContract(t, net, add.DeployAdd),
			contractInput: tests.MustGetMethodParameters(
				t, add.AddMetaData, "add",
				// arguments: iter (128_000 iterations is about the maximum
				// number of iterations that can be executed within the
				// block gas limit)
				big.NewInt(128_000),
			),
		},
		"DbHeavy": {
			contractAddr: tests.MustDeployContract(t, net, store.DeployStore),
			contractInput: tests.MustGetMethodParameters(
				t, store.StoreMetaData, "fill",
				// arguments: from, until, value (1100 slots is about the
				// maximum number of slots that can be filled within the
				// block gas limit)
				big.NewInt(0), big.NewInt(1100), big.NewInt(1),
			),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			envelopes := make([]*types.Transaction, B)
			planHashes := make([]common.Hash, B)

			// Create B bundles.
			for i := range B {
				envelope, _, plan := bundle.NewBuilder().
					WithSigner(signer).
					AllOf(
						bundle.AllOf(
							Step(t, net, accounts[i*3], &types.AccessListTx{
								To:   &tc.contractAddr,
								Data: tc.contractInput,
							}),
							Step(t, net, accounts[i*3+1], &types.AccessListTx{Gas: 1}),
						).WithFlags(bundle.EF_TolerateFailed),
						Step(t, net, accounts[i*3+2], &types.AccessListTx{}),
					).
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

			// Check that all bundles were executed successfully but only the
			// last outer transaction ended up in the block.
			for _, info := range infos {
				require.EqualValues(t, 1, info.Count)
			}
		})
	}
}

func TestBundle_StressWithLargeBundleAndGroupNesting(t *testing.T) {
	// Increase this number for profiling to increase load on the system.
	const B = 1 // Number of bundles

	net := GetIntegrationTestNetWithBundlesEnabled(t)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())

	// Create all needed accounts and endow in parallel.
	accounts := tests.MakeAccountsWithBalance(t, net, B*3, big.NewInt(1e18))

	cases := map[string]struct {
		groupNestingDepth  int
		bundleNestingDepth int
		expectedError      string
	}{
		"MaxNestingDepth": {
			groupNestingDepth:  bundle.MaxGroupNestingDepth,
			bundleNestingDepth: bundle.MaxBundleNestingDepth,
		},
		"GroupNestingDepthExceeded": {
			groupNestingDepth:  bundle.MaxGroupNestingDepth + 1,
			bundleNestingDepth: bundle.MaxBundleNestingDepth,
			expectedError:      "exceeds maximum nesting depth of execution steps",
		},
		"BundleNestingDepthExceeded": {
			groupNestingDepth:  bundle.MaxGroupNestingDepth,
			bundleNestingDepth: bundle.MaxBundleNestingDepth + 1,
			expectedError:      "exceeds maximum nesting depth of bundles",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			envelopes := make([]*types.Transaction, B)
			planHashes := make([]common.Hash, B)

			// Create B bundles.
			for i := range B {
				leafTx := types.NewTx(tests.SetTransactionDefaults(t, net, &types.AccessListTx{}, accounts[i*3]))
				envelope, plan := nestInStepsAndBundles(signer, accounts[i*3], leafTx, tc.groupNestingDepth, tc.bundleNestingDepth)

				envelopes[i] = envelope
				planHashes[i] = plan.Hash()
			}

			// Send all bundles.
			_, err = net.SendAll(envelopes)
			if len(tc.expectedError) > 0 {
				require.ErrorContains(t, err, tc.expectedError)
				return
			}
			require.NoError(t, err)

			// Wait for all bundles to be processed.
			infos, err := WaitForBundleExecutions(t.Context(), client.Client(), planHashes)
			require.NoError(t, err)

			// Check that all bundles were executed successfully and the single nested
			// transaction ended up in the block.
			for _, info := range infos {
				require.EqualValues(t, 1, info.Count)
			}
		})
	}
}

func nestInStepsAndBundles(
	signer types.Signer,
	sender *tests.Account,
	tx *types.Transaction,
	stepDepth int,
	bundleDepth int,
) (*types.Transaction, bundle.ExecutionPlan) {
	var plan bundle.ExecutionPlan
	for _ = range bundleDepth + 1 {
		step := bundle.Step(sender.PrivateKey, tx)
		for _ = range stepDepth {
			step = bundle.AllOf(step)
		}
		tx, plan = bundle.NewBuilder().WithSigner(signer).With(step).BuildEnvelopeAndPlan()
	}

	return tx, plan
}
