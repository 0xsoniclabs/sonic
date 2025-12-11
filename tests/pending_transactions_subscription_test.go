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

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestPendingTransactionSubscription_ReturnsFullTransaction(t *testing.T) {

	session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())
	// This test cannot be parallel because it expects only the specific transaction it sends

	client, err := session.GetWebSocketClient()
	require.NoError(t, err, "failed to get client ", err)
	defer client.Close()

	tx := CreateTransaction(t, session,
		&types.LegacyTx{
			To:    &common.Address{0x42},
			Value: big.NewInt(2),
			// needed because the tx from the channel comes with an empty Data field instead of nil
			Data: []byte{},
		},
		session.GetSessionSponsor())

	pendingTxs := make(chan *ethapi.RPCTransaction)
	defer close(pendingTxs)

	trueBool := true
	subs, err := client.Client().EthSubscribe(t.Context(), pendingTxs, "newPendingTransactions", &trueBool)
	require.NoError(t, err, "failed to subscribe to pending transactions ", err)
	defer subs.Unsubscribe()

	err = client.SendTransaction(t.Context(), tx)
	require.NoError(t, err, "failed to send transaction ", err)

	got := <-pendingTxs
	want := ethapi.NewRPCPendingTransaction(tx, tx.GasPrice(), session.GetChainId())
	require.Equal(t, want, got, "transaction from address does not match")

}

func TestPendingTransactionSubscription_ReturnsHashes(t *testing.T) {
	session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())
	// This test cannot be parallel because it expects only the specific transaction it sends

	client, err := session.GetWebSocketClient()
	require.NoError(t, err, "failed to get client ", err)
	defer client.Close()

	tx := CreateTransaction(t, session, &types.LegacyTx{To: &common.Address{0x42}, Value: big.NewInt(2)}, session.GetSessionSponsor())

	pendingTxs := make(chan common.Hash)
	defer close(pendingTxs)

	subs, err := client.Client().EthSubscribe(t.Context(), pendingTxs, "newPendingTransactions", nil)
	require.NoError(t, err, "failed to subscribe to pending transactions ", err)
	defer subs.Unsubscribe()

	err = client.SendTransaction(t.Context(), tx)
	require.NoError(t, err, "failed to send transaction ", err)

	got := <-pendingTxs
	require.Equal(t, tx.Hash(), got, "transaction hash does not match")
}
