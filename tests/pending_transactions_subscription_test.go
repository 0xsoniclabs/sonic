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
	"fmt"
	"math/big"
	"strings"
	"testing"

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

	tx := CreateTransaction(t, session, &types.LegacyTx{To: &common.Address{0x42}, Value: big.NewInt(2)}, session.GetSessionSponsor())

	v, r, s := tx.RawSignatureValues()

	expectedTx := map[string]any{
		"blockHash":           nil,
		"gas":                 fmt.Sprintf("0x%x", tx.Gas()),
		"gasPrice":            fmt.Sprintf("0x%x", tx.GasPrice()),
		"input":               "0x",
		"to":                  "0x4200000000000000000000000000000000000000",
		"transactionIndex":    nil,
		"chainId":             "0xfa3",
		"v":                   fmt.Sprintf("0x%x", v),
		"nonce":               "0x0",
		"value":               "0x2",
		"r":                   fmt.Sprintf("0x%x", r),
		"blobVersionedHashes": nil,
		"blockNumber":         nil,
		"from":                strings.ToLower(session.GetSessionSponsor().Address().Hex()),
		"hash":                tx.Hash().Hex(),
		"type":                "0x0",
		"s":                   fmt.Sprintf("0x%x", s),
		"maxFeePerBlobGas":    nil,
	}

	subscribeAndVerifyPendingTx(t, client, tx, expectedTx)
}

func TestPendingTransactionSubscription_ReturnsHashes(t *testing.T) {

	session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())
	// This test cannot be parallel because it expects only the specific transaction it sends

	client, err := session.GetWebSocketClient()
	require.NoError(t, err, "failed to get client ", err)
	defer client.Close()

	tx := CreateTransaction(t, session, &types.LegacyTx{To: &common.Address{0x42}, Value: big.NewInt(2)}, session.GetSessionSponsor())
	subscribeAndVerifyPendingTx(t, client, tx, nil)
}

func subscribeAndVerifyPendingTx(t *testing.T, client *ethClient, originalTx *types.Transaction, expectedTx map[string]any) {
	pendingTxs := make(chan any)
	defer close(pendingTxs)

	subs, err := client.Client().EthSubscribe(t.Context(), pendingTxs, "newPendingTransactions", expectedTx != nil)
	require.NoError(t, err, "failed to subscribe to pending transactions ", err)
	defer subs.Unsubscribe()

	err = client.SendTransaction(t.Context(), originalTx)
	require.NoError(t, err, "failed to send transaction ", err)

	got := <-pendingTxs
	if expectedTx != nil {
		tx, ok := got.(map[string]any)
		require.True(t, ok, "expected full transaction but got different type")
		require.Equal(t, expectedTx, tx, "transaction from address does not match")
	} else {
		hashStr, ok := got.(string)
		require.True(t, ok, "expected transaction hash string but got different type")
		hash := common.HexToHash(hashStr)
		require.Equal(t, originalTx.Hash(), hash, "transaction hash does not match")
	}

}
