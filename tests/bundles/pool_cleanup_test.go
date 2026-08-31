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

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
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
