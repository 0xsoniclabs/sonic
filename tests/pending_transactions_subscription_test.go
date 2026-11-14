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

package tests

import (
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// pendingTxNotificationTimeout bounds the wait for a newPendingTransactions
// notification. The notification is emitted when the transaction is accepted
// into the pool rather than when it is included in a block, so this only has
// to cover pool acceptance plus delivery over the WebSocket. It matches the
// 60s RPC timeout the nodes of the integration test net are started with.
const pendingTxNotificationTimeout = 60 * time.Second

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

	pendingTxs := make(chan *RPCTransaction)
	defer close(pendingTxs)

	trueBool := true
	subs, err := client.Client().EthSubscribe(t.Context(), pendingTxs, "newPendingTransactions", &trueBool)
	require.NoError(t, err, "failed to subscribe to pending transactions ", err)
	defer subs.Unsubscribe()

	err = client.SendTransaction(t.Context(), tx)
	require.NoError(t, err, "failed to send transaction ", err)

	var got *RPCTransaction
	select {
	case got = <-pendingTxs:
	case err := <-subs.Err():
		t.Fatalf("subscription failed before a pending transaction was received: %v", err)
	case <-time.After(pendingTxNotificationTimeout):
		t.Fatalf("no pending-transaction notification received within %v", pendingTxNotificationTimeout)
	}

	require.Equal(t, pendingForm(tx, session.GetSessionSponsor().Address()), got, "transaction from address does not match")

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

	var got common.Hash
	select {
	case got = <-pendingTxs:
	case err := <-subs.Err():
		t.Fatalf("subscription failed before a pending transaction was received: %v", err)
	case <-time.After(pendingTxNotificationTimeout):
		t.Fatalf("no pending-transaction notification received within %v", pendingTxNotificationTimeout)
	}

	require.Equal(t, tx.Hash(), got, "transaction hash does not match")
}

// pendingForm is how the RPC answers a legacy transaction that is in no
// block: priced by its gas price, with its signature and chain id.
func pendingForm(tx *types.Transaction, from common.Address) *RPCTransaction {
	v, r, s := tx.RawSignatureValues()
	return &RPCTransaction{
		From:     from,
		Gas:      hexutil.Uint64(tx.Gas()),
		GasPrice: (*hexutil.Big)(tx.GasPrice()),
		Hash:     tx.Hash(),
		Input:    tx.Data(),
		Nonce:    hexutil.Uint64(tx.Nonce()),
		To:       tx.To(),
		Value:    (*hexutil.Big)(tx.Value()),
		Type:     hexutil.Uint64(tx.Type()),
		ChainID:  (*hexutil.Big)(tx.ChainId()),
		V:        (*hexutil.Big)(v),
		R:        (*hexutil.Big)(r),
		S:        (*hexutil.Big)(s),
	}
}
