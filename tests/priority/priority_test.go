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

// Package priority contains end-to-end integration tests for the transaction
// priorities feature.
package priority

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestPriority_PrioritizedTransactionsAreScheduledFirst demonstrates the
// end-to-end behavior of the transaction priorities feature: a configurable
// on-chain registry designates some senders as prioritized, and the resulting
// blocks schedule those transactions ahead of ordinary ones, ordered by
// (level, weight), regardless of submission order.
//
// It runs in both block-formation modes. In single-proposer mode this also
// exercises the authoritative override: even though the proposer schedules the
// transactions, block formation re-derives and enforces the priority order.
func TestPriority_PrioritizedTransactionsAreScheduledFirst(t *testing.T) {
	t.Run("legacy", func(t *testing.T) {
		testPrioritiesScheduledFirst(t, false)
	})
	t.Run("single-proposer", func(t *testing.T) {
		testPrioritiesScheduledFirst(t, true)
	})
}

func testPrioritiesScheduledFirst(t *testing.T, singleProposer bool) {
	require := require.New(t)

	net, client, signer := netClientSignerWithPriorities(t, func(u *opera.Upgrades) {
		u.SingleProposerBlockFormation = singleProposer
	})
	defer client.Close()

	// The registry must have been deployed in genesis.
	code, err := client.CodeAt(t.Context(), registry.GetAddress(), nil)
	require.NoError(err)
	require.NotEmpty(code, "priority registry must be deployed")

	// Define prioritized accounts, each with a distinct (level, weight). They are
	// declared in the order we expect them to appear within a block.
	type prioritized struct {
		account *tests.Account
		level   int64
		weight  int64
	}
	prios := []prioritized{
		{level: 2, weight: 50}, // highest level -> first
		{level: 1, weight: 90},
		{level: 1, weight: 10},
	}
	prioByAddr := map[common.Address]prioritized{}
	for i := range prios {
		acc := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
		prios[i].account = acc
		prioByAddr[acc.Address()] = prios[i]

		setPrioritized(t, net, acc.Address(),
			prios[i].level, prios[i].weight, common.Hash{byte(i + 1)})
	}

	// Ordinary background traffic from fresh, unregistered senders.
	const numNormal = 4
	const txsPerNormal = 3
	burst := types.Transactions(buildOrdinaryTraffic(t, net, signer, numNormal, txsPerNormal))

	// Append one tx per prioritized account (nonce 0) at the end of the burst
	// so that a naive submission-order scheduler would not place them first.
	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(err)
	sink := common.Address{0x99}
	for i := range prios {
		burst = append(burst, types.MustSignNewTx(prios[i].account.PrivateKey, signer, &types.LegacyTx{
			Nonce:    0,
			To:       &sink,
			Value:    big.NewInt(1),
			Gas:      21000,
			GasPrice: gasPrice,
		}))
	}

	hashes, err := net.SendAll(burst)
	require.NoError(err)
	waitForReceipts(t, net, hashes)

	// Inspect all blocks: within each block, prioritized user transactions must
	// form a prefix (appear before any ordinary user transaction) and be ordered
	// by (level desc, weight desc). At least one block must mix both classes to
	// prove that prioritized transactions actually jumped ahead.
	latest, err := client.BlockNumber(t.Context())
	require.NoError(err)

	mixSeen := false
	for n := uint64(0); n <= latest; n++ {
		block, err := client.BlockByNumber(t.Context(), new(big.Int).SetUint64(n))
		require.NoError(err)

		sawNormal := false
		prioInBlock := make([]prioritized, 0)
		for _, tx := range block.Transactions() {
			if internaltx.IsInternal(tx) {
				continue
			}
			sender, err := types.Sender(signer, tx)
			require.NoError(err)
			if p, ok := prioByAddr[sender]; ok {
				require.False(sawNormal,
					"block %d: prioritized tx scheduled after an ordinary tx", n)
				prioInBlock = append(prioInBlock, p)
			} else {
				sawNormal = true
			}
		}

		if len(prioInBlock) > 0 && sawNormal {
			mixSeen = true
		}
		for i := 1; i < len(prioInBlock); i++ {
			prev, cur := prioInBlock[i-1], prioInBlock[i]
			ordered := prev.level > cur.level ||
				(prev.level == cur.level && prev.weight >= cur.weight)
			require.True(ordered,
				"block %d: prioritized txs not ordered by (level, weight): %+v before %+v",
				n, prev, cur)
		}
	}

	require.True(mixSeen,
		"expected at least one block containing both prioritized and ordinary transactions")
}

// TestPriority_PriorityReorderingOverwritesNonceOrdering documents an
// intentional consequence of block-formation reordering (see §11 of the
// priorities design doc): when the priority machinery hoists a high-nonce
// transaction ahead of a same-sender lower-nonce non-prioritized one, the
// hoisted transaction is skipped in that block (nonce too high). Because the
// two transactions are proposed atomically (bypassing the tx pool via
// ForceEmit) the skipped transaction is not automatically re-tried, so it
// ends up with no receipt at all. Without prioritization the two transactions
// are executed in nonce order and both land in the same block.
func TestPriority_PriorityReorderingOverwritesNonceOrdering(t *testing.T) {
	require := require.New(t)

	net, client, signer := netClientSignerWithPriorities(t, nil)
	defer client.Close()

	reg, err := registry.NewRegistry(registry.GetAddress(), client)
	require.NoError(err)

	// The registry's `maxGas` filter suppresses priority for any transaction
	// whose gas limit exceeds the threshold. This lets us have two txs from
	// the same sender where only the low-gas one is classified as
	// prioritized.
	const gasThreshold = uint64(25_000)
	receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return reg.SetMaxGas(opts, new(big.Int).SetUint64(gasThreshold))
	})
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(err)
	sink := common.Address{0x99}

	signFrom := func(acc *tests.Account, nonce, gasLimit uint64) *types.Transaction {
		tx := types.NewTx(&types.LegacyTx{
			Nonce:    nonce,
			To:       &sink,
			Value:    big.NewInt(1),
			Gas:      gasLimit,
			GasPrice: gasPrice,
		})
		signed, err := types.SignTx(tx, signer, acc.PrivateKey)
		require.NoError(err)
		return signed
	}

	cases := map[string]struct {
		prioritized bool
	}{
		"prio reordering skips the hoisted tx": {prioritized: true},
		"no prio preserves nonce order":        {prioritized: false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
			if tc.prioritized {
				setPrioritized(t, net, sender.Address(), 1, 1, common.Hash{0xaa})
			}

			// tx1: gas limit above the threshold -> never prioritized.
			// tx2: gas limit at the threshold    -> prioritized iff the sender is registered.
			tx1 := signFrom(sender, 0, gasThreshold+5_000)
			tx2 := signFrom(sender, 1, gasThreshold)

			// Propose both transactions atomically into a single event so
			// they are guaranteed to be considered together for the same
			// block. This bypasses the tx pool, so a transaction that is
			// skipped during block formation is not automatically re-tried.
			_, err := net.ForceEmitAll(t.Context(), []*types.Transaction{tx1, tx2})
			require.NoError(err)

			// tx1 (nonce 0) must succeed in both scenarios.
			receipt1, err := net.GetReceipt(tx1.Hash())
			require.NoError(err)
			require.Equal(types.ReceiptStatusSuccessful, receipt1.Status)

			if tc.prioritized {
				// Prioritization hoists tx2 ahead of tx1 in the block; tx2
				// is skipped for nonce-too-high and, since it was not
				// submitted through the tx pool, it never gets re-tried.
				_, err := net.TryGetReceipt(3*time.Second, tx2.Hash())
				require.ErrorIs(err, context.DeadlineExceeded,
					"tx2 must have no receipt: prio reordering skipped it")
			} else {
				// Without prioritization the base ordering is by nonce, so
				// both txs land in the same block, tx1 before tx2.
				receipt2, err := net.GetReceipt(tx2.Hash())
				require.NoError(err)
				require.Equal(types.ReceiptStatusSuccessful, receipt2.Status)
				require.Equal(receipt1.BlockNumber.Uint64(), receipt2.BlockNumber.Uint64(),
					"without prio both txs must land in the same block")
				require.Less(receipt1.TransactionIndex, receipt2.TransactionIndex,
					"without prio tx1 (lower nonce) must precede tx2")
			}
		})
	}
}
