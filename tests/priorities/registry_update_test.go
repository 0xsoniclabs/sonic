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
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/magic_value_priority"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestPriorities_SwapContract_PrioritizationChanges(t *testing.T) {
	require := require.New(t)

	net, client, signer := netClientSignerWithPriorities(t, nil)
	defer client.Close()

	registered := newFundedPrioritizedAccount(t, net, 1, 0, 0xaa)
	unregistered := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	// MagicValuePriority classifies a transaction as prioritized iff its value
	// equals a fixed magic constant. It is deployed up front so its magic value
	// is known; deploying does not affect the network until the swap below.
	impl, deployReceipt, err := tests.DeployContract(net, magic_value_priority.DeployMagicValuePriority)
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, deployReceipt.Status)
	magicValue, err := impl.MAGICVALUE(nil)
	require.NoError(err)

	buildRegisteredSenderTxs := func() []*types.Transaction {
		nonce, err := client.PendingNonceAt(t.Context(), registered.Address())
		require.NoError(err)
		txs := make([]*types.Transaction, 5)
		for i := range txs {
			txs[i] = newSignedTx(t, net,
				registered, nonce+uint64(i), 1, 21_000, nil)
		}
		return txs
	}
	buildMagicValueTxs := func() []*types.Transaction {
		nonce, err := client.PendingNonceAt(t.Context(), unregistered.Address())
		require.NoError(err)
		txs := make([]*types.Transaction, 5)
		for i := range txs {
			txs[i] = newSignedTx(t, net,
				unregistered, nonce+uint64(i), magicValue.Uint64(), 21_000, nil)
		}
		return txs
	}
	txHasRegisteredSender := func(tx *types.Transaction) bool {
		from, err := types.Sender(signer, tx)
		require.NoError(err)
		return from == registered.Address()
	}
	txHasMagicValue := func(tx *types.Transaction) bool { return tx.Value().Cmp(magicValue) == 0 }

	// --- default registry - prioritization depends only on the sender ---
	requirePrioritized(t, net, registered.Address(), 1, true)
	requirePrioritized(t, net, registered.Address(), magicValue.Uint64(), true)
	requirePrioritized(t, net, unregistered.Address(), 1, false)
	requirePrioritized(t, net, unregistered.Address(), magicValue.Uint64(), false)

	requirePriorityHasEffect(t, net, buildRegisteredSenderTxs(), true, txHasRegisteredSender)
	requirePriorityHasEffect(t, net, buildMagicValueTxs(), false, txHasMagicValue)

	// --- MagicValuePriority registry - prioritization depends only on the tx value ---
	switchPriorityRegistry(t, net, deployReceipt.ContractAddress)

	requirePrioritized(t, net, registered.Address(), 1, false)
	requirePrioritized(t, net, registered.Address(), magicValue.Uint64(), true)
	requirePrioritized(t, net, unregistered.Address(), 1, false)
	requirePrioritized(t, net, unregistered.Address(), magicValue.Uint64(), true)

	requirePriorityHasEffect(t, net, buildRegisteredSenderTxs(), false, txHasRegisteredSender)
	requirePriorityHasEffect(t, net, buildMagicValueTxs(), true, txHasMagicValue)
}
