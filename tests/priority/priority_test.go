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
	"slices"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TODO: targets emitter and processor parts of priorities design.
// TestPriority_PrioritizedTransactionsAreScheduledFirst demonstrates the
// end-to-end behavior of the transaction priorities feature: the emitter
// includes prioritized transactions in earlier events than ordinary ones,
// and the block processor enforces priority ordering within each block.
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

	var configure func(*opera.Upgrades)
	if singleProposer {
		configure = func(u *opera.Upgrades) { u.SingleProposerBlockFormation = true }
	}
	net, client, signer := netClientSignerWithPriorities(t, configure)
	defer client.Close()

	// The registry must have been deployed in genesis.
	code, err := client.CodeAt(t.Context(), registry.GetAddress(), nil)
	require.NoError(err)
	require.NotEmpty(code, "priority registry must be deployed")

	// Four prioritized senders in scrambled (level, weight) order so the test
	// cannot accidentally pass due to insertion order.
	// Expected block ordering: (2,3) → (2,1) → (1,4) → (1,2).
	type prioSpec struct {
		account       *tests.Account
		level, weight int64
	}
	prios := []prioSpec{
		{level: 1, weight: 2},
		{level: 1, weight: 4},
		{level: 2, weight: 1},
		{level: 2, weight: 3},
	}
	for i := range prios {
		acc := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
		prios[i].account = acc
		setPrioritized(t, net, acc.Address(), prios[i].level, prios[i].weight, common.Hash{byte(i + 1)})
	}

	// Reduce MaxEventGas to its allowed minimum so the ordinary batch spans
	// several events (~47 simple transfers per event at this limit).
	current := tests.GetNetworkRules(t, net)
	modified := current.Copy()
	modified.Economy.Gas.MaxEventGas = opera.UpperBoundForRuleChangeGasCosts() + modified.Economy.Gas.EventGas
	tests.UpdateNetworkRules(t, net, modified)
	net.AdvanceEpoch(t, 1)

	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(err)
	highGasPrice := new(big.Int).Add(gasPrice, big.NewInt(2e9))
	lowGasPrice := new(big.Int).Add(gasPrice, big.NewInt(1e9))
	sink := common.Address{0x99}

	// Build batch: 100 ordinary txs first (rules out FIFO), then prio txs
	// with lower gas price (rules out fee ordering).
	const numNonPrio = 100
	numPrio := len(prios)
	batch := make([]*types.Transaction, 0, numNonPrio+numPrio)
	for range numNonPrio {
		acc := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
		batch = append(batch, types.MustSignNewTx(acc.PrivateKey, signer, &types.LegacyTx{
			Nonce:    0,
			To:       &sink,
			Value:    big.NewInt(1),
			Gas:      21000,
			GasPrice: highGasPrice,
		}))
	}
	type prioEntry struct{ level, weight int64 }
	prioByHash := make(map[common.Hash]prioEntry, numPrio)
	for _, p := range prios {
		tx := types.MustSignNewTx(p.account.PrivateKey, signer, &types.LegacyTx{
			Nonce:    0,
			To:       &sink,
			Value:    big.NewInt(1),
			Gas:      21000,
			GasPrice: lowGasPrice,
		})
		batch = append(batch, tx)
		prioByHash[tx.Hash()] = prioEntry{p.level, p.weight}
	}

	hashes, err := net.SendAll(batch)
	require.NoError(err)

	// Collect all receipts.
	type txResult struct {
		blockNum      uint64
		txIdx         uint
		level, weight int64
		isPrio        bool
	}
	results := make([]txResult, 0, len(hashes))
	for _, h := range hashes {
		receipt, err := net.GetReceipt(h)
		require.NoError(err)
		require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
		if entry, isPrio := prioByHash[h]; isPrio {
			results = append(results, txResult{receipt.BlockNumber.Uint64(), receipt.TransactionIndex, entry.level, entry.weight, true})
		} else {
			results = append(results, txResult{blockNum: receipt.BlockNumber.Uint64(), txIdx: receipt.TransactionIndex})
		}
	}

	var prioResults, nonPrioResults []txResult
	for _, r := range results {
		if r.isPrio {
			prioResults = append(prioResults, r)
		} else {
			nonPrioResults = append(nonPrioResults, r)
		}
	}
	require.Len(prioResults, numPrio)

	// All prio txs must land in the same block — they fit comfortably within
	// a single event (4 × 21_000 gas << MaxEventGas).
	prioBlock := prioResults[0].blockNum
	for _, r := range prioResults {
		require.Equal(prioBlock, r.blockNum, "all prio txs must land in the same block")
	}

	// Emitter check (legacy mode only): in legacy mode MaxEventGas caps the
	// number of txs per event, so the 100-tx ordinary batch spans several
	// events. The priority heap ensures prio txs land in the first event and
	// therefore in an earlier block than the last ordinary event.
	// In single-proposer mode all pending txs are typically included in a
	// single proposal so the inter-block comparison is not meaningful.
	maxNonPrioBlock := nonPrioResults[0].blockNum
	for _, r := range nonPrioResults {
		if r.blockNum > maxNonPrioBlock {
			maxNonPrioBlock = r.blockNum
		}
	}
	if !singleProposer {
		require.Less(prioBlock, maxNonPrioBlock,
			"prio txs (lower fee, submitted last) must land in earlier blocks than ordinary txs")
	}

	// Processor check: within each block, prio txs form a prefix before all
	// ordinary txs and are ordered by (level desc, weight desc). At least one
	// block must contain both classes.
	byBlock := make(map[uint64][]txResult)
	for _, r := range results {
		byBlock[r.blockNum] = append(byBlock[r.blockNum], r)
	}
	mixSeen := false
	for blockNum, txs := range byBlock {
		slices.SortFunc(txs, func(a, b txResult) int {
			if a.txIdx < b.txIdx {
				return -1
			}
			if a.txIdx > b.txIdx {
				return 1
			}
			return 0
		})
		sawNormal := false
		var prioInBlock []txResult
		for _, r := range txs {
			if r.isPrio {
				require.False(sawNormal,
					"block %d: prio tx (level=%d,weight=%d) scheduled after an ordinary tx",
					blockNum, r.level, r.weight)
				prioInBlock = append(prioInBlock, r)
			} else {
				sawNormal = true
			}
		}
		if len(prioInBlock) > 0 && sawNormal {
			mixSeen = true
		}
		for i := 1; i < len(prioInBlock); i++ {
			prev, cur := prioInBlock[i-1], prioInBlock[i]
			require.True(
				prev.level > cur.level || (prev.level == cur.level && prev.weight >= cur.weight),
				"block %d: prio tx (%d,%d) before (%d,%d) violates ordering",
				blockNum, prev.level, prev.weight, cur.level, cur.weight,
			)
		}
	}
	require.True(mixSeen, "expected at least one block with both prio and ordinary txs")
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
