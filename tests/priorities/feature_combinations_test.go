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

package priorities

import (
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	testbundles "github.com/0xsoniclabs/sonic/tests/bundles"
	"github.com/0xsoniclabs/sonic/tests/gas_subsidies"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestPriorities_PriorityOfBundlesIsThePriorityOfTheirEnvelope(t *testing.T) {
	const numBundles = 5
	const txsPerBundle = 2

	cases := map[string]struct {
		// true => the envelope sender is prioritized
		// false => the bundle sender is prioritized
		prioritizeEnvelopeSender bool
		expectPrioritized        bool
	}{
		"prioritized envelope sender prioritized all txs in bundle": {
			prioritizeEnvelopeSender: true,
			expectPrioritized:        true,
		},
		"prioritized bundled txs sender does not prioritize txs in bundle": {
			prioritizeEnvelopeSender: false,
			expectPrioritized:        false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			net, client, signer := netClientSignerWithPriorities(t, func(u *opera.Upgrades) {
				u.TransactionBundles = true
			})
			defer client.Close()

			envelopeSenders := tests.MakeAccountsWithBalance(t, net, numBundles, big.NewInt(1e18))
			bundleSenders := tests.MakeAccountsWithBalance(t, net, numBundles, big.NewInt(1e18))

			prioritizedSenders := bundleSenders
			if tc.prioritizeEnvelopeSender {
				prioritizedSenders = envelopeSenders
			}
			for i, sender := range prioritizedSenders {
				setPrioritized(t, net, sender.Address(), 1, 1, uint64(i)+1)
			}

			block, err := client.BlockNumber(t.Context())
			require.NoError(err)

			envelopes := make([]*types.Transaction, numBundles)
			planHashes := make([]common.Hash, numBundles)
			bundled := map[common.Hash]bool{}
			for i, sender := range bundleSenders {
				steps := make([]bundle.BuilderStep, txsPerBundle)
				for n := range steps {
					steps[n] = testbundles.Step(t, net, sender, &types.AccessListTx{
						Nonce: uint64(n),
						To:    &common.Address{},
						Value: big.NewInt(1),
						Gas:   21_000,
					})
				}

				envelope, txBundle, plan := bundle.NewBuilder().
					WithSigner(signer).
					SetEarliest(block).
					SetEnvelopeSenderKey(envelopeSenders[i].PrivateKey).
					AllOf(steps...).
					BuildEnvelopeBundleAndPlan()

				envelopes[i] = envelope
				planHashes[i] = plan.Hash()
				for _, tx := range txBundle.Transactions {
					bundled[tx.Hash()] = true
				}
			}

			ordinaryTxs := buildOrdinaryTraffic(t, net, 20, 5)

			afterBlock, err := client.BlockNumber(t.Context())
			require.NoError(err)

			sendShuffledToPool(t, net, slices.Concat(envelopes, ordinaryTxs))

			infos, err := testbundles.WaitForBundleExecutions(t.Context(), client.Client(), planHashes)
			require.NoError(err)
			for _, info := range infos {
				require.EqualValues(txsPerBundle, info.Count)
			}

			requirePriorityAppliedSince(t, net, afterBlock, tc.expectPrioritized,
				func(tx *types.Transaction) bool {
					return bundled[tx.Hash()]
				})
		})
	}
}

func TestPriorities_PrioritizedSponsoredTransactionsAlsoPrioritizesPaymentTransaction(t *testing.T) {
	const numTxs = 5

	require := require.New(t)

	net, client, signer := netClientSignerWithPriorities(t, func(u *opera.Upgrades) {
		u.GasSubsidies = true
	})
	defer client.Close()

	sponsee := tests.NewAccount()
	gas_subsidies.Fund(t, net, sponsee.Address(), big.NewInt(1e18))
	setPrioritized(t, net, sponsee.Address(), 1, 1, 1)

	sponsoredTxs := make([]*types.Transaction, numTxs)
	for i := range sponsoredTxs {
		sponsoredTxs[i] = types.MustSignNewTx(sponsee.PrivateKey, signer, &types.LegacyTx{
			Nonce:    uint64(i),
			To:       &common.Address{},
			Value:    big.NewInt(0),
			Gas:      21_000,
			GasPrice: big.NewInt(0),
		})
		require.True(subsidies.IsSponsorshipRequest(sponsoredTxs[i]))
	}

	ordinaryTxs := buildOrdinaryTraffic(t, net, 20, 5)

	afterBlock, err := client.BlockNumber(t.Context())
	require.NoError(err)

	hashes := sendShuffledToPool(t, net, slices.Concat(sponsoredTxs, ordinaryTxs))

	receipts, err := net.GetReceipts(hashes)
	require.NoError(err)
	receiptByHash := make(map[common.Hash]*types.Receipt, len(receipts))
	for _, receipt := range receipts {
		require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
		receiptByHash[receipt.TxHash] = receipt
	}

	// Verify that each sponsored tx is followed by its payment tx.
	for _, tx := range sponsoredTxs {
		receipt := receiptByHash[tx.Hash()]
		block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
		require.NoError(err)
		txs := block.Transactions()
		require.Less(int(receipt.TransactionIndex)+1, len(txs))

		payment := txs[receipt.TransactionIndex+1]
		require.True(internaltx.IsInternal(payment))
		paymentReceipt, err := net.GetReceipt(payment.Hash())
		require.NoError(err)
		require.Equal(types.ReceiptStatusSuccessful, paymentReceipt.Status)
	}

	// Payment txs are internal and therefore skipped by requirePriorityAppliedSince.
	requirePriorityAppliedSince(t, net, afterBlock, true,
		func(tx *types.Transaction) bool {
			from, err := types.Sender(signer, tx)
			require.NoError(err)
			return from == sponsee.Address()
		})
}
