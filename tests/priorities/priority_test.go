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
	"cmp"
	"encoding/binary"
	"math/big"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/magic_value_priority"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestPriorities_TransactionsAreScheduledInPriorityOrder(t *testing.T) {
	cases := map[string]struct {
		singleProposer bool
	}{
		"distributed proposers": {false},
		"single proposer":       {true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			net, client, _ := netClientSignerWithPriorities(t,
				func(u *opera.Upgrades) {
					u.SingleProposerBlockFormation = tc.singleProposer
				},
			)
			defer client.Close()

			prioByHash := make(map[common.Hash]priorities.Priority)

			ordinary := buildOrdinaryTraffic(t, net, 20, 5)
			for _, tx := range ordinary {
				prioByHash[tx.Hash()] = priorities.Priority{}
			}

			const numLevels, numWeights = 4, 4
			prioritized := make(types.Transactions, 0, numLevels*numWeights)
			for level := uint64(1); level <= numLevels; level++ {
				for weight := uint64(1); weight <= numWeights; weight++ {
					id := (level-1)*numWeights + weight
					prio := priorities.Priority{Level: level, Weight: weight}
					binary.BigEndian.PutUint64(prio.ID[8:], id)

					account := newFundedPrioritizedAccount(t, net, level, weight, id)
					tx := newSignedTx(t, net, account, 0, 1, 21_000, nil)
					prioritized = append(prioritized, tx)
					prioByHash[tx.Hash()] = prio
				}
			}

			batch := slices.Concat(ordinary, prioritized)
			rand.Shuffle(len(batch), func(i, j int) {
				batch[i], batch[j] = batch[j], batch[i]
			})

			// Add the whole batch to the pool in a single call so every transaction is
			// available before the emitter builds its first event.
			hashes, err := net.SendAllToPool(t.Context(), batch)
			require.NoError(err)

			receipts, err := net.GetReceipts(hashes)
			require.NoError(err)

			first, last := receipts[0].BlockNumber.Uint64(), receipts[0].BlockNumber.Uint64()
			for _, receipt := range receipts {
				require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
				first = min(first, receipt.BlockNumber.Uint64())
				last = max(last, receipt.BlockNumber.Uint64())
			}

			scheduled := make([]priorities.Priority, 0, len(prioByHash))
			for number := first; number <= last; number++ {
				block, err := client.BlockByNumber(t.Context(), new(big.Int).SetUint64(number))
				require.NoError(err)
				for _, tx := range block.Transactions() {
					scheduled = append(scheduled, prioByHash[tx.Hash()])
				}
			}
			require.Len(scheduled, len(ordinary)+len(prioritized))

			// The schedule must be in descending priority order.
			require.True(slices.IsSortedFunc(scheduled, func(a, b priorities.Priority) int {
				return b.Cmp(a)
			}))
		})
	}
}

func TestPriorities_PriorityOrderingPreservesNonceOrdering(t *testing.T) {
	modes := map[string]struct {
		singleProposer bool
	}{
		"distributed proposers": {false},
		"single proposer":       {true},
	}

	for modeName, mode := range modes {
		t.Run(modeName, func(t *testing.T) {
			require := require.New(t)

			net, client, _ := netClientSignerWithPriorities(t,
				func(u *opera.Upgrades) {
					u.SingleProposerBlockFormation = mode.singleProposer
				},
			)
			defer client.Close()

			// MagicValuePriority classifies a transaction as prioritized iff its
			// value equals a fixed magic constant. This lets us alternate between
			// prioritized and ordinary transactions within a single sender's nonce
			// sequence.
			impl, deployReceipt, err := tests.DeployContract(net, magic_value_priority.DeployMagicValuePriority)
			require.NoError(err)
			require.Equal(types.ReceiptStatusSuccessful, deployReceipt.Status)
			magicValue, err := impl.MAGICVALUE(nil)
			require.NoError(err)
			switchPriorityRegistry(t, net, deployReceipt.ContractAddress)

			sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

			// Every odd-nonce transaction outranks its predecessor, so honoring
			// priorities alone would reverse each pair.
			const numTxs = 10
			txs := make([]*types.Transaction, numTxs)
			for i := range txs {
				value := uint64(1)
				if i%2 == 1 {
					value = magicValue.Uint64()
				}
				txs[i] = newSignedTx(t, net, sender, uint64(i), value, 21_000, nil)
			}

			// Add the whole batch to the pool in a single call so every
			// transaction is available before the emitter builds its first event.
			hashes, err := net.SendAllToPool(t.Context(), txs)
			require.NoError(err)

			receipts, err := net.GetReceipts(hashes)
			require.NoError(err)
			for _, receipt := range receipts {
				require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
			}

			// The receipts are in nonce order, so their positions must be
			// ascending as well.
			require.True(slices.IsSortedFunc(receipts, func(a, b *types.Receipt) int {
				return cmp.Or(
					a.BlockNumber.Cmp(b.BlockNumber),
					cmp.Compare(a.TransactionIndex, b.TransactionIndex),
				)
			}))
		})
	}
}
