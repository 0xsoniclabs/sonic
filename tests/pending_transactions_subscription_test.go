// Copyright 2025 Sonic Operations Ltd
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

package tests

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestPendingTransactionSubscription_ReturnsFullTransaction(t *testing.T) {

	session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())
	t.Parallel()

	client, err := session.GetWebSocketClient()
	require.NoError(t, err, "failed to get client ", err)
	defer client.Close()

	tx := CreateTransaction(t, session, &types.LegacyTx{To: &common.Address{0x42}, Value: big.NewInt(2)}, session.GetSessionSponsor())
	subscribeAndVerifyPendingTx(t, client, tx, true)
}

func TestPendingTransactionSubscription_ReturnsHashes(t *testing.T) {

	session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())
	t.Parallel()

	client, err := session.GetWebSocketClient()
	require.NoError(t, err, "failed to get client ", err)
	defer client.Close()

	tx := CreateTransaction(t, session, &types.LegacyTx{To: &common.Address{0x42}, Value: big.NewInt(2)}, session.GetSessionSponsor())
	subscribeAndVerifyPendingTx(t, client, tx, false)
}

func subscribeAndVerifyPendingTx(t *testing.T, client *ethClient, originalTx *types.Transaction, expectFullTx bool) {
	pendingTxs := make(chan any)
	defer close(pendingTxs)

	subs, err := client.Client().EthSubscribe(t.Context(), pendingTxs, "newPendingTransactions", expectFullTx)
	require.NoError(t, err, "failed to subscribe to pending transactions ", err)
	defer subs.Unsubscribe()

	err = client.SendTransaction(t.Context(), originalTx)
	require.NoError(t, err, "failed to send transaction ", err)

	// wait for a pending transaction
	select {
	case got := <-pendingTxs:
		if expectFullTx {
			tx, ok := got.(*types.Transaction)
			require.True(t, ok, "expected full transaction but got different type")
			require.Equal(t, originalTx.Hash(), tx.Hash(), "transaction from address does not match")
		} else {
			hash, ok := got.(common.Hash)
			require.True(t, ok, "expected transaction hash but got different type")
			require.Equal(t, originalTx.Hash(), hash, "transaction hash does not match")
		}
	case err := <-subs.Err():
		// Err returns the subscription error channel. The intended use of Err is to schedule
		// resubscription when the client connection is closed unexpectedly.
		//
		// The error channel receives a value when the subscription has ended due to an error. The
		// received error is nil if Close has been called on the underlying client and no other
		// error has occurred.
		//
		// During this test this channel should not receive any value, so if either the connection is closed
		// or an error is received, this test should fail.
		require.Fail(t, "unexpected subscription error: %v", err)
	}
}
