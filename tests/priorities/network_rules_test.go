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
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type upgrades struct{ TransactionPriorities bool }
type rulesDiff struct{ Upgrades upgrades }

func TestPriorities_PriorityCanBeEnabledAndDisabled(t *testing.T) {
	for name, upgrade := range opera.GetAllHardForksInOrder() {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			upgrade.TransactionPriorities = true
			net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
				Upgrades: &upgrade,
			})
			signer := types.LatestSignerForChainID(net.GetChainId())

			client, err := net.GetClient()
			require.NoError(err)
			defer client.Close()

			prioAccount := newFundedPrioritizedAccount(t, net, 1, 1, 0xaa)
			isPrioAccount := func(tx *types.Transaction) bool {
				from, err := types.Sender(signer, tx)
				require.NoError(err)
				return from == prioAccount.Address()
			}
			buildPrioTxs := func() []*types.Transaction {
				nonce, err := client.PendingNonceAt(t.Context(), prioAccount.Address())
				require.NoError(err)

				txs := make([]*types.Transaction, 5)
				for i := range txs {
					txs[i] = newSignedTx(t, net, prioAccount, nonce+uint64(i), 1, 21_000, nil)
				}
				return txs
			}

			// --- TransactionPriorities enabled ---
			rules := tests.GetNetworkRules(t, net)
			require.True(rules.Upgrades.TransactionPriorities)

			requirePriorityHasEffect(t, net, buildPrioTxs(), true, isPrioAccount)

			// --- TransactionPriorities disabled ---
			rulesUpdate := rulesDiff{Upgrades: upgrades{TransactionPriorities: false}}
			tests.UpdateNetworkRules(t, net, rulesUpdate)
			net.AdvanceEpoch(t, 1)

			rules = tests.GetNetworkRules(t, net)
			require.False(rules.Upgrades.TransactionPriorities)

			requirePriorityHasEffect(t, net, buildPrioTxs(), false, isPrioAccount)

			// --- TransactionPriorities enabled ---
			rulesUpdate = rulesDiff{Upgrades: upgrades{TransactionPriorities: true}}
			tests.UpdateNetworkRules(t, net, rulesUpdate)
			net.AdvanceEpoch(t, 1)

			rules = tests.GetNetworkRules(t, net)
			require.True(rules.Upgrades.TransactionPriorities)

			requirePriorityHasEffect(t, net, buildPrioTxs(), true, isPrioAccount)
		})
	}
}
