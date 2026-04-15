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

func TestBundle_ExecutionFlagsOfSingleTxAreInterpretedCorrectly(t *testing.T) {
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())

	revertAddress, revertInput := tests.MustDeployRevertContractAndGetMethodCallParameters(t, net)

	sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
	sender2 := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	successfulTx := types.AccessListTx{}
	failingTx := types.AccessListTx{
		To:   &revertAddress,
		Gas:  1_000_000,
		Data: revertInput,
	}
	invalidTx := types.AccessListTx{
		Gas: 1, // insufficient gas
	}

	cases := []struct {
		name            string
		tx              types.AccessListTx
		flags           bundle.ExecutionFlags
		expectTolerated bool
		expectInBlock   bool
		expectedStatus  uint64
	}{
		{
			name:            "Default/SuccessfulTx",
			tx:              successfulTx,
			flags:           bundle.EF_Default,
			expectTolerated: true,
			expectInBlock:   true,
			expectedStatus:  types.ReceiptStatusSuccessful,
		},
		{
			name:            "Default/FailingTx",
			tx:              failingTx,
			flags:           bundle.EF_Default,
			expectTolerated: false,
		},
		{
			name:            "Default/InvalidTx",
			tx:              invalidTx,
			flags:           bundle.EF_Default,
			expectTolerated: false,
		},
		{
			name:            "TolerateInvalid/SuccessfulTx",
			tx:              successfulTx,
			flags:           bundle.EF_TolerateInvalid,
			expectTolerated: true,
			expectInBlock:   true,
			expectedStatus:  types.ReceiptStatusSuccessful,
		},
		{
			name:            "TolerateInvalid/FailingTx",
			tx:              failingTx,
			flags:           bundle.EF_TolerateInvalid,
			expectTolerated: false,
		},
		{
			name:            "TolerateInvalid/InvalidTx",
			tx:              invalidTx,
			flags:           bundle.EF_TolerateInvalid,
			expectTolerated: true,
			expectInBlock:   false,
		},
		{
			name:            "TolerateFailed/SuccessfulTx",
			tx:              successfulTx,
			flags:           bundle.EF_TolerateFailed,
			expectTolerated: true,
			expectInBlock:   true,
			expectedStatus:  types.ReceiptStatusSuccessful,
		},
		{
			name:            "TolerateFailed/FailingTx",
			tx:              failingTx,
			flags:           bundle.EF_TolerateFailed,
			expectTolerated: true,
			expectInBlock:   true,
			expectedStatus:  types.ReceiptStatusFailed,
		},
		{
			name:            "TolerateFailed/InvalidTx",
			tx:              invalidTx,
			flags:           bundle.EF_TolerateFailed,
			expectTolerated: false,
		},
		{
			name:            "TolerateInvalidTolerateFailed/SuccessfulTx",
			tx:              successfulTx,
			flags:           bundle.EF_TolerateInvalid | bundle.EF_TolerateFailed,
			expectTolerated: true,
			expectInBlock:   true,
			expectedStatus:  types.ReceiptStatusSuccessful,
		},
		{
			name:            "TolerateInvalidTolerateFailed/FailingTx",
			tx:              failingTx,
			flags:           bundle.EF_TolerateInvalid | bundle.EF_TolerateFailed,
			expectTolerated: true,
			expectInBlock:   true,
			expectedStatus:  types.ReceiptStatusFailed,
		},
		{
			name:            "TolerateInvalidTolerateFailed/InvalidTx",
			tx:              invalidTx,
			flags:           bundle.EF_TolerateInvalid | bundle.EF_TolerateFailed,
			expectTolerated: true,
			expectInBlock:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			block, err := client.BlockNumber(t.Context())
			require.NoError(t, err)

			// Create the bundle: AllOf([c.flags]c.tx, successfulTx)
			// The second transaction is needed for the cases with an invalid
			// transaction to check whether it was tolerated or not.
			envelope, bundle, plan := bundle.NewBuilder().
				WithSigner(signer).
				SetEarliest(block).
				AllOf(
					bundle.Step(
						sender.PrivateKey,
						tests.SetTransactionDefaults(t, net, &c.tx, sender),
					).WithFlags(c.flags),
					bundle.Step(
						sender2.PrivateKey,
						tests.SetTransactionDefaults(t, net, &successfulTx, sender2),
					),
				).
				BuildEnvelopeBundleAndPlan()

			// Check bundle status before submission.
			_, err = GetBundleInfo(t.Context(), client.Client(), plan.Hash())
			require.ErrorIs(t, err, ethereum.NotFound)

			// Run the bundle.
			require.NoError(t, client.SendTransaction(t.Context(), envelope))

			// Wait for the bundle to be processed.
			info, err := WaitForBundleExecution(t.Context(), client.Client(), plan.Hash())
			require.NoError(t, err)

			// Verify that there is no receipt for the envelope itself.
			_, err = client.TransactionReceipt(t.Context(), envelope.Hash())
			require.ErrorIs(t, err, ethereum.NotFound)

			// If the transaction is not expected to be tolerated, the whole
			// outer group should be rejected, and thus no transactions should
			// be included in a block.
			if !c.expectTolerated {
				require.Zero(t, info.Count)
				return
			}

			// If the transaction is expected to be tolerated but not included
			// in a block, only the successful transaction that follows it
			// should be included, but not the transaction itself.
			if !c.expectInBlock {
				require.Equal(t, 1, int(info.Count))
				return
			}

			// The transactions itself and the successful transaction that
			// follows it should be included.
			require.Equal(t, 1+1, int(info.Count))

			// Check that the transaction is in the block as advertised.
			txs := bundle.GetTransactionsInReferencedOrder()
			receipt, err := net.GetReceipt(txs[0].Hash())
			require.NoError(t, err)
			require.Equal(t, c.expectedStatus, receipt.Status)
			require.EqualValues(t, info.Block, receipt.BlockNumber.Uint64())
		})
	}
}
