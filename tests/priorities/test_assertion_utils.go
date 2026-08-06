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
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// requirePriorityHasEffect proves that the currently installed priority
// classifier is (or isn't, per `expectPrioritized`) consulted on the
// block-formation path. It submits `prioritizedTxs` together with a burst of
// ordinary background traffic from fresh, unrelated senders in a single batch
// through the tx pool, and asserts via requirePriorityAppliedSince that
// `isPrioritized` transactions are scheduled ahead of ordinary ones (or not).
// The senders of `prioritizedTxs` must be disjoint from the background-traffic
// senders, which is always the case.
func requirePriorityHasEffect(
	t *testing.T,
	net *tests.IntegrationTestNet,
	prioritizedTxs []*types.Transaction,
	expectPrioritized bool,
	isPrioritized func(*types.Transaction) bool,
) {
	t.Helper()
	require := require.New(t)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	// Pre-build ordinary background traffic (fresh accounts are funded now
	// so their funding txs don't compete for block space with the actual
	// test batch).
	ordinaryTxs := buildOrdinaryTraffic(t, net, 20, 5)

	afterBlock, err := client.BlockNumber(t.Context())
	require.NoError(err)

	// Add prioritized + ordinary txs to the pool in a single batch so they all
	// become available before the emitter builds its first event, making their
	// relative ordering deterministic.
	batch := append([]*types.Transaction{}, ordinaryTxs...)
	batch = append(batch, prioritizedTxs...)

	hashes, err := net.SendAllToPool(t.Context(), batch)
	require.NoError(err)

	receipts, err := net.GetReceipts(hashes)
	require.NoError(err)
	require.Len(receipts, len(hashes))
	for _, receipt := range receipts {
		require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
	}

	requirePriorityAppliedSince(t, net, afterBlock, expectPrioritized, isPrioritized)
}

// buildOrdinaryTraffic constructs a set of signed transfers from freshly
// created, non-prioritized accounts.
func buildOrdinaryTraffic(
	t *testing.T,
	net *tests.IntegrationTestNet,
	numAccounts int,
	txsPerAccount int,
) types.Transactions {
	t.Helper()

	txs := make([]*types.Transaction, 0, numAccounts*int(txsPerAccount))
	for i := 0; i < numAccounts; i++ {
		acc := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
		for n := 0; n < txsPerAccount; n++ {
			txs = append(txs, newSignedTx(t, net, acc, uint64(n), 1, 21000, nil))
		}
	}
	return txs
}

// requirePriorityAppliedSince scans the user transactions of every block after
// `afterBlock`, in block-then-index order, and asserts on the global ordering of
// prioritized (per `isPrioritized`) vs ordinary transactions. Both classes must
// be present for the check to be meaningful. `afterBlock` is meant to be the
// head observed just before the batch was submitted; that block is excluded
// because it still holds the test's setup transactions (e.g. the funding of the
// ordinary senders), which would otherwise count as ordinary traffic.
//
//   - expectPrioritized == true: all prioritized txs come before all ordinary
//     ones (no ordinary tx precedes a prioritized one).
//   - expectPrioritized == false: at least one ordinary tx precedes a
//     prioritized one (proof that no reordering ran).
//
// Callers should submit the whole batch via SendAllToPool, so this global
// ordering is deterministic and does not depend on arrival times of individual
// transactions in the pool and whether emission ran between those or not.
func requirePriorityAppliedSince(
	t *testing.T,
	net *tests.IntegrationTestNet,
	afterBlock uint64,
	expectPrioritized bool,
	isPrioritized func(*types.Transaction) bool,
) {
	t.Helper()
	require := require.New(t)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	latest, err := client.BlockNumber(t.Context())
	require.NoError(err)

	signer := types.LatestSignerForChainID(net.GetChainId())
	prioritizedSenders := map[common.Address]struct{}{}
	ordinarySenders := map[common.Address]struct{}{}
	numPrioritized := 0
	ordinaryBeforePriority := false
	for n := afterBlock + 1; n <= latest; n++ {
		block, err := client.BlockByNumber(t.Context(), new(big.Int).SetUint64(n))
		require.NoError(err)
		for _, tx := range block.Transactions() {
			if internaltx.IsInternal(tx) {
				continue
			}
			from, err := types.Sender(signer, tx)
			require.NoError(err)
			if isPrioritized(tx) {
				numPrioritized++
				prioritizedSenders[from] = struct{}{}
				if len(ordinarySenders) > 0 {
					ordinaryBeforePriority = true
				}
			} else {
				ordinarySenders[from] = struct{}{}
			}
		}
	}

	// For the test to be meaningful the chance of a false positive must be
	// extremely low. Transactions of one sender are scheduled in nonce order, so
	// only a single transaction per sender competes for the next slot. Without
	// prioritization the chance of a slot going to a prioritized transaction is
	// therefore at most `num_prioritized_senders/num_senders`, and all
	// prioritized transactions come first only if that happens `num_prioritized`
	// times in a row. This is a conservative estimate: the ratio only shrinks as
	// prioritized senders run out of transactions, so the true chance is lower.
	numSenders := len(prioritizedSenders) + len(ordinarySenders)
	prioritizedChance := float64(len(prioritizedSenders)) / float64(numSenders)
	falsePositiveChance := math.Pow(prioritizedChance, float64(numPrioritized))
	require.Less(falsePositiveChance, 1e-6,
		"false positive chance %.1e with %d prioritized and %d ordinary senders too high, "+
			"use more prioritized transactions and/or ordinary senders to reduce it",
		falsePositiveChance, len(prioritizedSenders), len(ordinarySenders))

	require.Equal(expectPrioritized, !ordinaryBeforePriority)
}
