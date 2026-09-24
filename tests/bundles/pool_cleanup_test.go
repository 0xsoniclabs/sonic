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
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/revert"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// bundleSteps are the building blocks the test cases below compose their bundle
// of. Steps not referenced by the resulting bundle are dropped by the builder.
type bundleSteps struct {
	tested     bundle.BuilderStep // < also submitted to the pool, never executed
	reverting  bundle.BuilderStep // < fails its execution
	succeeding bundle.BuilderStep // < executes fine, from an unrelated account
}

// TestBundles_BundleOnlyTxOfEvaluatedBundleDoesNotBlockTheSendersNonce checks
// that a bundle-only transaction in the pool is dropped once its bundle got
// evaluated without executing it. Such a transaction can never be included in a
// block on its own, so keeping it would block the nonce of its sender until it
// times out of the pool.
func TestBundles_BundleOnlyTxOfEvaluatedBundleDoesNotBlockTheSendersNonce(t *testing.T) {

	// Each case builds a bundle which is executable as a whole, and thus
	// accepted by the pool, but which does not execute the tested transaction.
	// All of them add exactly one transaction to the block.
	cases := map[string]func(bundleSteps) bundle.BuilderStep{
		"tested transaction first in a group failing later": func(s bundleSteps) bundle.BuilderStep {
			return bundle.OneOf(bundle.AllOf(s.tested, s.reverting), s.succeeding)
		},
		"tested transaction after a failing transaction in a group": func(s bundleSteps) bundle.BuilderStep {
			return bundle.OneOf(bundle.AllOf(s.reverting, s.tested), s.succeeding)
		},
		"tested transaction as alternative to a succeeding transaction": func(s bundleSteps) bundle.BuilderStep {
			return bundle.OneOf(s.succeeding, s.tested)
		},
	}

	net := GetIntegrationTestNetWithBundlesEnabled(t)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())

	revertAddress := tests.MustDeployContract(t, net, revert.DeployRevert)
	// doRevert refunds the gas it does not use, keeping the bundles below
	// efficient enough for the pool to accept them.
	revertInput := tests.MustGetMethodParameters(t, revert.RevertMetaData, "doRevert")

	// The tested transaction is identified by the value it transfers.
	recipient := common.Address{0x42}
	testedValue := big.NewInt(100)

	for name, makeBundle := range cases {
		t.Run(name, func(t *testing.T) {
			accounts := tests.MakeAccountsWithBalance(t, net, 3, big.NewInt(1e18))
			sender, reverter, other := accounts[0], accounts[1], accounts[2]

			nonce, err := client.PendingNonceAt(t.Context(), sender.Address())
			require.NoError(t, err)

			blockNumber, err := client.BlockNumber(t.Context())
			require.NoError(t, err)

			envelope, txBundle, plan := bundle.NewBuilder().
				WithSigner(signer).
				SetEarliest(blockNumber).
				With(makeBundle(bundleSteps{
					tested: Step(t, net, sender, &types.AccessListTx{
						Nonce: nonce,
						To:    &recipient,
						Value: testedValue,
					}),
					reverting: Step(t, net, reverter, &types.AccessListTx{
						To:   &revertAddress,
						Gas:  100_000,
						Data: revertInput,
					}),
					succeeding: Step(t, net, other, &types.AccessListTx{}),
				})).
				BuildEnvelopeBundleAndPlan()

			var testedTx *types.Transaction
			for _, tx := range txBundle.GetTransactionsInReferencedOrder() {
				if tx.Value().Cmp(testedValue) == 0 {
					require.Nil(t, testedTx, "the tested transaction must be unique")
					testedTx = tx
				}
			}
			require.NotNil(t, testedTx, "the bundle must contain the tested transaction")
			require.True(t, bundle.IsBundleOnly(testedTx))
			require.Equal(t, nonce, testedTx.Nonce())

			// The bundle-only transaction is also known to the pool, as it is
			// gossiped between nodes and may be submitted by the participants
			// of the bundle themselves.
			_, err = net.Send(testedTx)
			require.NoError(t, err)

			// While the bundle is in flight, the pool holds the transaction.
			require.Eventually(t, func() bool {
				pending, err := client.PendingNonceAt(t.Context(), sender.Address())
				return err == nil && pending == nonce+1
			}, 10*time.Second, 50*time.Millisecond,
				"the bundle-only transaction should be pending in the pool",
			)

			// Run the bundle, the tested transaction must not reach the block.
			_, err = net.Send(envelope)
			require.NoError(t, err)

			info, err := WaitForBundleExecution(t.Context(), client.Client(), plan.Hash())
			require.NoError(t, err)
			require.Equal(t, 1, int(info.Count),
				"only the succeeding transaction should be included")

			block := big.NewInt(info.Block.Int64())
			_, err = client.TransactionReceipt(t.Context(), testedTx.Hash())
			require.ErrorIs(t, err, ethereum.NotFound)
			require.NotContains(t, getBlockTxsHashes(t, client, block), testedTx.Hash())

			onChainNonce, err := client.NonceAt(t.Context(), sender.Address(), block)
			require.NoError(t, err)
			require.Equal(t, nonce, onChainNonce,
				"the skipped transaction must not consume a nonce")

			// The evaluated transaction must be evicted from the pool.
			ctxt, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			err = tests.WaitFor(ctxt, func(ctxt context.Context) (bool, error) {
				pending, err := client.PendingNonceAt(ctxt, sender.Address())
				if err != nil {
					return false, err
				}
				return pending == nonce, nil
			})
			assert.NoError(t, err,
				"the pending nonce must return to %d after the bundle got evaluated", nonce,
			)

			// The envelope is dropped as well, it can not run a second time.
			require.Eventually(t, func() bool {
				_, _, err := client.TransactionByHash(t.Context(), envelope.Hash())
				return errors.Is(err, ethereum.NotFound)
			}, 30*time.Second, 100*time.Millisecond,
				"the envelope of an evaluated bundle should be dropped from the pool",
			)

			// Gossip may re-introduce the transaction, it must be rejected.
			_, err = net.Send(testedTx)
			require.ErrorContains(t, err, "bundle has already been processed",
				"a bundle-only transaction of an evaluated bundle must not be accepted again")

			// A regular transaction using the reported nonce must be executable.
			followUp := tests.CreateTransaction(t, net, &types.AccessListTx{
				To:    &recipient,
				Value: big.NewInt(1),
			}, sender)
			followUpHash, err := net.Send(followUp)
			require.NoError(t, err)
			receipt, err := net.TryGetReceipt(20*time.Second, followUpHash)
			require.NoError(t, err,
				"a regular transaction must not be blocked by the skipped bundle-only transaction")
			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
		})
	}
}

// TestBundles_BundleOnlyTxIsEvictedWithoutItsEnvelope checks that a node drops
// a bundle-only transaction of an evaluated bundle even if its pool never got
// to see the envelope, e.g. because a wallet signed the transaction before it
// got packed into an envelope submitted to another node.
func TestBundles_BundleOnlyTxIsEvictedWithoutItsEnvelope(t *testing.T) {
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true

	// Node 0 holds enough stake to confirm blocks on its own, so node 1 can be
	// cut off while the bundle runs, keeping the envelope out of its pool.
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades:        &upgrades,
		ValidatorsStake: []uint64{99, 1},
		ModifyConfig: func(c *config.Config) {
			c.Emitter.EmitIntervals.DoublesignProtection = 0 // < emit without peers
		},
	})

	client0, err := net.GetClientConnectedToNode(0)
	require.NoError(t, err)
	defer client0.Close()
	client1, err := net.GetClientConnectedToNode(1)
	require.NoError(t, err)
	defer client1.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())
	accounts := tests.MakeAccountsWithBalance(t, net, 2, big.NewInt(1e18))
	sender, other := accounts[0], accounts[1]

	nonce, err := client0.PendingNonceAt(t.Context(), sender.Address())
	require.NoError(t, err)
	blockNumber, err := client0.BlockNumber(t.Context())
	require.NoError(t, err)

	// The bundle runs the transaction of the other account instead of the
	// tested one, so the tested transaction does not consume its nonce.
	recipient := common.Address{0x42}
	testedData := &types.AccessListTx{Nonce: nonce, To: &recipient, Value: big.NewInt(100)}
	envelope, txBundle, plan := bundle.NewBuilder().
		WithSigner(signer).
		SetEarliest(blockNumber).
		With(bundle.OneOf(
			Step(t, net, other, &types.AccessListTx{}),
			Step(t, net, sender, testedData),
		)).
		BuildEnvelopeBundleAndPlan()

	var testedTx *types.Transaction
	for _, tx := range txBundle.GetTransactionsInReferencedOrder() {
		if tx.Nonce() == nonce && tx.To() != nil && *tx.To() == recipient {
			testedTx = tx
		}
	}
	require.NotNil(t, testedTx, "the bundle must contain the tested transaction")
	require.True(t, bundle.IsBundleOnly(testedTx))

	setNodesConnected(t, net, false)

	// Only node 1 learns about the bundle-only transaction.
	require.NoError(t, client1.SendTransaction(t.Context(), testedTx))
	require.Eventually(t, func() bool {
		pending, err := client1.PendingNonceAt(t.Context(), sender.Address())
		return err == nil && pending == nonce+1
	}, 10*time.Second, 50*time.Millisecond,
		"the bundle-only transaction should be pending in the pool of node 1",
	)

	// Only node 0 learns about the envelope, and runs the bundle.
	_, err = net.Send(envelope)
	require.NoError(t, err)
	info, err := WaitForBundleExecution(t.Context(), client0.Client(), plan.Hash())
	require.NoError(t, err)
	require.Equal(t, 1, int(info.Count), "only the other transaction should be included")

	// Reconnecting while node 0 still holds the envelope would gossip it.
	require.Eventually(t, func() bool {
		_, _, err := client0.TransactionByHash(t.Context(), envelope.Hash())
		return errors.Is(err, ethereum.NotFound)
	}, 30*time.Second, 100*time.Millisecond,
		"the envelope of an evaluated bundle should be dropped from the pool of node 0",
	)
	_, _, err = client1.TransactionByHash(t.Context(), envelope.Hash())
	require.ErrorIs(t, err, ethereum.NotFound, "node 1 must never see the envelope")

	setNodesConnected(t, net, true)

	// Once node 1 caught up with the bundle, it evicts the tested transaction.
	_, err = WaitForBundleExecution(t.Context(), client1.Client(), plan.Hash())
	require.NoError(t, err)
	ctxt, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	err = tests.WaitFor(ctxt, func(ctxt context.Context) (bool, error) {
		pending, err := client1.PendingNonceAt(ctxt, sender.Address())
		return pending == nonce, err
	})
	require.NoError(t, err,
		"the pending nonce on node 1 must return to %d after the bundle got evaluated", nonce,
	)

	// A regular transaction using the reported nonce must be executable.
	followUp := tests.CreateTransaction(t, net, &types.AccessListTx{
		To:    &recipient,
		Value: big.NewInt(1),
	}, sender)
	require.NoError(t, client1.SendTransaction(t.Context(), followUp))
	receipt, err := net.TryGetReceipt(20*time.Second, followUp.Hash())
	require.NoError(t, err,
		"a regular transaction must not be blocked by the evicted bundle-only transaction")
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
}

// setNodesConnected connects or disconnects the two nodes of the given network.
func setNodesConnected(t *testing.T, net *tests.IntegrationTestNet, connected bool) {
	t.Helper()
	require.Equal(t, 2, net.NumNodes())

	clients := make([]*tests.PooledEhtClient, 2)
	enodes := make([]string, 2)
	for i := range clients {
		client, err := net.GetClientConnectedToNode(i)
		require.NoError(t, err)
		defer client.Close()
		clients[i] = client

		var info struct {
			Enode string `json:"enode"`
		}
		require.NoError(t, client.Client().Call(&info, "admin_nodeInfo"))
		enodes[i] = info.Enode
	}

	if connected {
		require.NoError(t, clients[1].Client().Call(nil, "admin_addPeer", enodes[0]))
	} else {
		// Removing the peer on both sides also stops the dialer from redialing.
		for i, client := range clients {
			require.NoError(t, client.Client().Call(nil, "admin_removePeer", enodes[1-i]))
		}
	}

	for i, client := range clients {
		require.Eventually(t, func() bool {
			var peers []map[string]any
			err := client.Client().Call(&peers, "admin_peers")
			return err == nil && (len(peers) > 0) == connected
		}, 30*time.Second, 100*time.Millisecond,
			"node %d should have connected=%v", i, connected,
		)
	}
}
