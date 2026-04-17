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

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBundle_BundleContainingAnyEmptyGroupIsRejected(t *testing.T) {
	t.Parallel()

	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())

	senders := tests.MakeAccountsWithBalance(t, net, 4, big.NewInt(1e18))

	tx := types.AccessListTx{}

	cases := map[string]struct {
		root         bundle.BuilderStep
		ExpectReject bool
	}{
		"AllOf/NonEmpty": {
			root: bundle.AllOf(
				bundle.Step(
					senders[0].PrivateKey,
					tests.SetTransactionDefaults(t, net, &tx, senders[0]),
				),
			),
			ExpectReject: false,
		},
		"AllOf/Empty": {
			root:         bundle.AllOf(),
			ExpectReject: true,
		},
		"OneOf/NonEmpty": {
			root: bundle.OneOf(
				bundle.Step(
					senders[1].PrivateKey,
					tests.SetTransactionDefaults(t, net, &tx, senders[1]),
				),
			),
			ExpectReject: false,
		},
		"OneOf/Empty": {
			root:         bundle.OneOf(),
			ExpectReject: true,
		},
		"Layered/NonEmpty": {
			root: bundle.AllOf(
				bundle.AllOf(
					bundle.Step(
						senders[2].PrivateKey,
						tests.SetTransactionDefaults(t, net, &tx, senders[2]),
					),
				),
			),
			ExpectReject: false,
		},
		"Layered/EmptyAndNonEmptySubGroups": {
			root: bundle.AllOf(
				bundle.AllOf(
					bundle.Step(
						senders[3].PrivateKey,
						tests.SetTransactionDefaults(t, net, &tx, senders[3]),
					),
				),
				bundle.AllOf(),
			),
			ExpectReject: false,
		},
		"Layered/OnlyEmptySubGroups": {
			root: bundle.AllOf(
				bundle.AllOf(),
			),
			ExpectReject: true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			blockNumber, err := client.BlockNumber(t.Context())
			require.NoError(t, err)

			envelope, bundle, plan := bundle.NewBuilder().
				WithSigner(signer).
				SetEarliest(blockNumber).
				With(c.root).
				BuildEnvelopeBundleAndPlan()

			// Send the bundle.
			require.NoError(t, client.SendTransaction(t.Context(), envelope))

			// Wait for the bundle to be processed.
			timeout, timeoutCancel := context.WithTimeout(t.Context(), 1*time.Second)
			defer timeoutCancel()
			info, err := WaitForBundleExecution(timeout, client.Client(), plan.Hash())

			if c.ExpectReject {
				require.ErrorIs(t, err, context.DeadlineExceeded)
				return
			}

			require.NoError(t, err)

			blockTxsHashes := getBlockTxsHashes(t, client, big.NewInt(info.Block.Int64()))
			bundleTxs := bundle.GetTransactionsInReferencedOrder()

			require.Equal(t, 1, int(info.Count))
			require.Contains(t, blockTxsHashes, bundleTxs[0].Hash())
		})
	}
}
