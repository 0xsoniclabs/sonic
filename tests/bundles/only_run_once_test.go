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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBundle_RunOnlyOnce_ExecutionPlanSubmittedMultipleTimesInDifferentEnvelopesIsOnlyProcessedOnce(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())

	senders := tests.MakeAccountsWithBalance(t, net, 2, big.NewInt(1e18))

	// This test creates multiple envelopes with the same transaction bundle
	// of the following shape:
	//
	// 								OneOf(A,B)
	//
	// The system should ensure that the bundle is only ever executed once,
	// even if multiple envelopes carrying the same bundle are submitted.
	// Since in the test setup A is always successful, B should never be
	// included in a block, even if the bundle is attempted to be processed
	// multiple times.

	// Create a bundle that runs a transaction A or B, but not both. If the
	// bundle is only processed once, there should only be a receipt for A but
	// none for B.
	b := bundle.NewBuilder().
		WithSigner(signer).
		OneOf(
			Step(t, net, senders[0], &types.AccessListTx{}),
			Step(t, net, senders[1], &types.AccessListTx{}),
		).BuildBundle()

	// Pack the same bundle into multiple envelopes.
	envelopes := []*types.Transaction{}
	for range 100 {
		envelopes = append(envelopes, bundle.MustWrapIntoEnvelope(signer, &b))
	}

	// Submit the same bundle multiple times using different envelopes.
	_, err = net.SendAll(envelopes)
	require.NoError(err)

	bundleTxs := b.GetTransactionsInReferencedOrder()

	// Transaction A should be executed successfully.
	receiptA, err := net.GetReceipt(bundleTxs[0].Hash())
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receiptA.Status)

	// Transaction B should not be executed.
	receiptB, err := client.TransactionReceipt(t.Context(), bundleTxs[1].Hash())
	require.ErrorIs(err, ethereum.NotFound, "Got receipt A: %+v, receipt B: %+v", receiptA, receiptB)
}

func TestBundle_RunOnlyOnce_NestedBundleSubmittedMultipleTimesInSameBundleIsOnlyProcessedOnce(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())

	senders := tests.MakeAccountsWithBalance(t, net, 4, big.NewInt(1e18))

	// This test creates multiple envelopes with a transaction bundle of
	// the following shape:
	//
	//       AllOf(Env(OneOf(A,B)), [TolerateInvalid]Env(OneOf(A,B)))
	//
	// The system should ensure that the bundle OneOf(A,B) is only ever
	// executed once, although envelopes containing it are included twice in
	// the top level bundle. Since in the test setup A is always successful,
	// B should never be included in a block.

	// Create the OneOf(A,B) bundle making sure that only A or B are executed.
	innerEnvelope, innerBundle, _ := bundle.NewBuilder().
		WithSigner(signer).
		OneOf(
			Step(t, net, senders[0], &types.AccessListTx{}),
			Step(t, net, senders[1], &types.AccessListTx{}),
		).BuildEnvelopeBundleAndPlan()

	// Create execution plan running the inner bundle multiple times.
	envelope, plan := bundle.NewBuilder().
		WithSigner(signer).
		AllOf(
			bundle.Step(senders[2].PrivateKey, innerEnvelope),
			bundle.Step(senders[3].PrivateKey, innerEnvelope).WithFlags(bundle.EF_TolerateInvalid),
		).
		BuildEnvelopeAndPlan()

	// Submit the outer envelope.
	_, err = net.Send(envelope)
	require.NoError(err)

	// Wait for the bundle to be processed.
	outerInfo, err := WaitForBundleExecution(t.Context(), client.Client(), plan.Hash())
	require.NoError(err)

	require.EqualValues(1, outerInfo.Count)

	bundleTxs := innerBundle.GetTransactionsInReferencedOrder()

	// Transaction A should be executed successfully.
	receiptA, err := net.GetReceipt(bundleTxs[0].Hash())
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receiptA.Status)
	require.EqualValues(outerInfo.Block, receiptA.BlockNumber.Uint64())
	require.EqualValues(outerInfo.Position, receiptA.TransactionIndex)

	// Transaction B should not be executed.
	receiptB, err := client.TransactionReceipt(t.Context(), bundleTxs[1].Hash())
	require.ErrorIs(err, ethereum.NotFound, "Got receipt A: %+v, receipt B: %+v", receiptA, receiptB)
}

func TestBundle_RunOnlyOnce_FailedGroupsCanBeRetried(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())

	sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	// This test creates a single envelope with a transaction bundle of
	// the following shape:
	//
	//       AllOf([TolerateFailed]OneOf(A1), A0, OneOf(A1))
	//
	// Transaction A0 uses nonce 0 and A1 uses nonce 1 of the same account.
	// Thus, A0 is required to enable A1. The first OneOf(A1) thus fails but
	// the second copy of OneOf(A1) should succeed.

	envelope, bundle, plan := bundle.NewBuilder().
		WithSigner(signer).
		AllOf(
			bundle.OneOf(Step(t, net, sender, &types.AccessListTx{
				To:    &common.Address{},
				Nonce: 1,
				Gas:   21000,
			})).WithFlags(bundle.EF_TolerateFailed),
			Step(t, net, sender, &types.AccessListTx{
				To:    &common.Address{},
				Nonce: 0,
				Gas:   21000,
			}),
			bundle.OneOf(Step(t, net, sender, &types.AccessListTx{
				To:    &common.Address{},
				Nonce: 1,
				Gas:   21000,
			})),
		).BuildEnvelopeBundleAndPlan()

	// Submit the outer bundle and wait for the execution to complete.
	_, err = net.Send(envelope)
	require.NoError(err)

	// Wait for all plans to complete.
	info, err := WaitForBundleExecution(t.Context(), client.Client(), plan.Hash())
	require.NoError(err)

	// We should see two transactions accepted - A0 and A1 in that order.
	require.EqualValues(2, info.Count)

	bundleTxs := bundle.GetTransactionsInReferencedOrder()

	// Transaction A0 should be executed successfully.
	receiptA0, err := net.GetReceipt(bundleTxs[1].Hash())
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receiptA0.Status)
	require.EqualValues(info.Block, receiptA0.BlockNumber.Uint64())
	require.EqualValues(info.Position, receiptA0.TransactionIndex)

	// Followed by A1.
	receiptA1, err := net.GetReceipt(bundleTxs[2].Hash())
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receiptA1.Status)
	require.EqualValues(info.Block, receiptA1.BlockNumber.Uint64())
	require.EqualValues(info.Position+1, receiptA1.TransactionIndex)
}
