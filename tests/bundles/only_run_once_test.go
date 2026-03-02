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
	"crypto/ecdsa"
	"fmt"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestBundles_RunOnlyOnce_AnExecutionPlanSubmittedMultipleTimesInDifferentEnvelopesIsOnlyProcessedOnce(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true

	session := sharedNetwork.GetIntegrationTestNetSession(t, upgrades)

	accountFactory := MakeAccountFactory(session)
	accounts := accountFactory.CreateMultiple(t, 2)

	signer := types.LatestSignerForChainID(session.GetChainId())

	// This test case creates multiple envelops sending the same transaction
	// bundle of the following shape:
	//
	// 	   							OneOf(A,B)
	//
	// The system should ensure that the bundle is only ever executed once,
	// even if multiple envelops carrying the same bundle are submitted.
	// Since in the test setup A is always successful, B should never be
	// included in a block, even if the bundle is attempted to be processed
	// multiple times.

	// Create a bundle that runs a transaction A or B, but not both. If the
	// bundle is only processed once, there should only be a receipt for A but
	// none for B.
	b := bundle.NewBuilder(signer).
		SetFlags(bundle.EF_OneOf).
		With(
			Step(t, session, accounts[0], &types.AccessListTx{
				To:  &common.Address{},
				Gas: 21000,
			}),
			Step(t, session, accounts[1], &types.AccessListTx{
				To:  &common.Address{},
				Gas: 21000,
			}),
		).BuildBundle()

	txA := b.Transactions[0]
	txB := b.Transactions[1]

	// Pack the same bundle into multiple envelops.
	envelopes := []*types.Transaction{}
	for range 100 {
		envelopes = append(envelopes, bundle.MustWrapIntoEnvelope(signer, b))
	}

	// Submit the same bundle multiple times using different envelops.
	accepted, rejected, err := session.TrySendAll(envelopes)
	require.NoError(err)
	require.NotEmpty(accepted)
	for _, issue := range rejected {
		require.ErrorContains(issue, "bundle is not executable")
	}

	// Transaction A should be executed successfully.
	receiptA, err := session.GetReceipt(txA.Hash())
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receiptA.Status)

	// Transaction B should not be executed.
	client, err := session.GetClient()
	require.NoError(err)
	defer client.Close()

	receiptB, err := client.TransactionReceipt(t.Context(), txB.Hash())
	require.ErrorIs(err, ethereum.NotFound, "Got receipt A: %+v, receipt B: %+v", receiptA, receiptB)
}

func TestBundles_RunOnlyOnce_AnExecutionPlanSubmittedMultipleTimesInTheSameBundleIsOnlyProcessedOnce(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true

	session := sharedNetwork.GetIntegrationTestNetSession(t, upgrades)

	client, err := session.GetClient()
	require.NoError(err)
	defer client.Close()

	accountFactory := MakeAccountFactory(session)
	accounts := accountFactory.CreateMultiple(t, 2)

	signer := types.LatestSignerForChainID(session.GetChainId())

	// This test creates multiple envelops sending a transaction bundle of
	// the following shape:
	//
	//         RunAll(Env(OneOf(A,B)), Env(OneOf(A,B)))
	//
	// RunAll is a AllOf bundle tolerating failed and invalid transactions,
	// ensuring that all transactions in the bundle are executed, at all time.
	// The inner OneOf bundle ensures that only A or B can be executed, but not
	// both. Furthermore, it should be enforced that the OneOf(A,B) bundle is
	// only ever executed at most once. Thus, only A should be executed, never
	// B, even if the bundle containing them is attempted to be processed
	// multiple times.

	// Create the OneOf(A,B) bundle making sure that only A or B are executed.
	inner := bundle.NewBuilder(signer).
		SetFlags(bundle.EF_OneOf).
		With(
			Step(t, session, accounts[0], &types.AccessListTx{
				To:  &common.Address{},
				Gas: 21000,
			}),
			Step(t, session, accounts[1], &types.AccessListTx{
				To:  &common.Address{},
				Gas: 21000,
			}),
		).BuildBundle()

	txA := inner.Transactions[0]
	txB := inner.Transactions[1]

	// Create multiple execution plans running the inner bundle multiple times.
	envelopes := []*types.Transaction{}
	planHashes := []common.Hash{}
	for range 100 {
		keys := MustGenerateKeys(2)
		envelope, plan := bundle.NewBuilder(signer).
			SetFlags(bundle.EF_AllOf|bundle.EF_TolerateFailed|bundle.EF_TolerateInvalid).
			With(
				bundle.Step(keys[0], bundle.MustWrapIntoEnvelope(signer, inner)),
				bundle.Step(keys[1], bundle.MustWrapIntoEnvelope(signer, inner)),
			).
			BuildEnvelopeAndPlan()

		envelopes = append(envelopes, envelope)
		planHashes = append(planHashes, plan.Hash())
	}

	// Submit the same bundle multiple times using different envelops.
	accepted, rejected, err := session.TrySendAll(envelopes)
	require.NoError(err)
	require.NotEmpty(accepted)
	for _, issue := range rejected {
		require.ErrorContains(issue, "bundle is not executable")
	}

	// Wait for all plans to complete (not all may get executed, since filtered in pool)
	ctxt, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	infos, err := waitForBundlesExecution(ctxt, client.Client(), planHashes)
	if err != nil {
		require.ErrorIs(context.DeadlineExceeded, err)
	}

	// All the all-of plans should be executed, but only one with an accepted transaction.
	var acceptedInfo *ethapi.RPCBundleInfo
	for _, info := range infos {
		if info != nil && *info.Count > 0 {
			require.Nil(acceptedInfo, "more than one all-of bundle had an affect")
			acceptedInfo = info
		}
	}
	require.NotNil(acceptedInfo, "no all-of bundle had an affect")

	// Transaction A should be executed successfully.
	receiptA, err := session.GetReceipt(txA.Hash())
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receiptA.Status)
	require.EqualValues(*acceptedInfo.Block, receiptA.BlockNumber.Uint64())
	require.EqualValues(*acceptedInfo.Position, receiptA.TransactionIndex)

	// Transaction B should not be executed.
	receiptB, err := client.TransactionReceipt(t.Context(), txB.Hash())
	require.ErrorIs(err, ethereum.NotFound, "Got receipt A: %+v, receipt B: %+v", receiptA, receiptB)
}

func TestBundles_RunOnlyOnce_FailedBundlesCanBeRetried(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true

	session := sharedNetwork.GetIntegrationTestNetSession(t, upgrades)

	client, err := session.GetClient()
	require.NoError(err)
	defer client.Close()

	accountFactory := MakeAccountFactory(session)
	account := accountFactory.Create(t)

	signer := types.LatestSignerForChainID(session.GetChainId())

	// This test creates a single envelope sending a transaction bundle of
	// the following shape:
	//
	//         RunAll(Env(OneOf(A1)), A0, Env(OneOf(A1)))
	//
	// RunAll is a AllOf bundle tolerating failed and invalid transactions,
	// ensuring that all steps in the bundle are processed, at all time.
	//
	// Transaction A0 uses nonce 0 and A1 uses nonce 1 of the same account.
	// Thus, A0 is required to enable A1. The first OneOf(A1) thus fails but
	// the second copy of OneOf(A1) should succeed.

	inner := bundle.NewBuilder(signer).
		SetFlags(bundle.EF_OneOf).
		With(
			Step(t, session, account, &types.AccessListTx{
				To:    &common.Address{},
				Nonce: 1,
				Gas:   21000,
			}),
		).BuildBundle()

	txA1 := inner.Transactions[0]

	keys := MustGenerateKeys(2)

	outer, txBundle, plan := bundle.NewBuilder(signer).
		SetFlags(bundle.EF_AllOf|bundle.EF_TolerateFailed|bundle.EF_TolerateInvalid).
		With(
			bundle.Step(keys[0], bundle.MustWrapIntoEnvelope(signer, inner)),
			Step(t, session, account, &types.AccessListTx{
				To:    &common.Address{},
				Nonce: 0,
				Gas:   21000,
			}),
			bundle.Step(keys[1], bundle.MustWrapIntoEnvelope(signer, inner)),
		).BuildEnvelopeBundleAndPlan()

	txA0 := txBundle.Transactions[1]

	// Submit the outer bundle and wait for the execution to complete.
	_, err = session.Send(outer)
	require.NoError(err)

	// Wait for all plans to complete.
	info, err := waitForBundleExecution(t.Context(), client.Client(), plan.Hash())
	require.NoError(err)

	// We should see two transactions accepted - A0 and A1 in that order.
	require.EqualValues(2, *info.Count)

	// Transaction A0 should be executed successfully.
	receiptA0, err := session.GetReceipt(txA0.Hash())
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receiptA0.Status)
	require.EqualValues(*info.Block, receiptA0.BlockNumber.Uint64())
	require.EqualValues(*info.Position, receiptA0.TransactionIndex)

	// Followed by A1.
	receiptA1, err := session.GetReceipt(txA1.Hash())
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receiptA1.Status)
	require.EqualValues(*info.Block, receiptA1.BlockNumber.Uint64())
	require.EqualValues(*info.Position+1, receiptA1.TransactionIndex)
}

func MustGenerateKeys(n int) []*ecdsa.PrivateKey {
	keys := make([]*ecdsa.PrivateKey, n)
	for i := range keys {
		key, err := crypto.GenerateKey()
		if err != nil {
			panic(fmt.Sprintf("failed to generate new key: %v", err))
		}
		keys[i] = key
	}
	return keys
}
