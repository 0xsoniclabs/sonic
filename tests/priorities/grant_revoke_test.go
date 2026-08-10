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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestPriorities_SenderPriorityCanBeGrantedAndRevoked(t *testing.T) {
	require := require.New(t)

	net, client, signer := netClientSignerWithPriorities(t, nil)
	defer client.Close()

	account := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	buildTxs := func() []*types.Transaction {
		nonce, err := client.PendingNonceAt(t.Context(), account.Address())
		require.NoError(err)
		txs := make([]*types.Transaction, 5)
		for i := range txs {
			txs[i] = newSignedTx(t, net, account, nonce+uint64(i), 1, 21_000, nil)
		}
		return txs
	}
	txHasRegisteredSender := func(tx *types.Transaction) bool {
		from, err := types.Sender(signer, tx)
		require.NoError(err)
		return from == account.Address()
	}

	// --- unregistered ---
	requirePrioritized(t, net, account.Address(), 1, false)
	requirePriorityHasEffect(t, net, buildTxs(), false, txHasRegisteredSender)

	// --- granted ---
	setPrioritized(t, net, account.Address(), 1, 1, 1)
	requirePrioritized(t, net, account.Address(), 1, true)
	requirePriorityHasEffect(t, net, buildTxs(), true, txHasRegisteredSender)

	// --- revoked ---
	setPrioritized(t, net, account.Address(), 0, 0, 0)
	requirePrioritized(t, net, account.Address(), 1, false)
	requirePriorityHasEffect(t, net, buildTxs(), false, txHasRegisteredSender)
}
