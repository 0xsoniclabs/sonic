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

package rpcs

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// TestSendRawTransactionSync tests the eth_sendRawTransactionSync RPC endpoint (EIP-7966).
func TestSendRawTransactionSync(t *testing.T) {
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{})

	t.Run("happy_path_returns_receipt", func(t *testing.T) {
		session := net.SpawnSession(t)

		// Create and fund a sender account.
		sender := tests.NewAccount()
		_, err := session.EndowAccount(sender.Address(), big.NewInt(1e18))
		require.NoError(t, err)

		client, err := net.GetClient()
		require.NoError(t, err)
		defer client.Close()

		chainID, err := client.ChainID(t.Context())
		require.NoError(t, err)

		nonce, err := client.PendingNonceAt(t.Context(), sender.Address())
		require.NoError(t, err)

		gasPrice, err := client.SuggestGasPrice(t.Context())
		require.NoError(t, err)

		receiver := tests.NewAccount()
		tx := tests.SignTransaction(t, chainID, &types.LegacyTx{
			Nonce:    nonce,
			Gas:      21000,
			GasPrice: gasPrice,
			To:       addrPtr(receiver.Address()),
			Value:    big.NewInt(1000),
		}, sender)

		encoded, err := tx.MarshalBinary()
		require.NoError(t, err)

		var receipt map[string]interface{}
		err = client.Client().Call(&receipt, "eth_sendRawTransactionSync", hexutil.Bytes(encoded))
		require.NoError(t, err, "eth_sendRawTransactionSync must succeed for a valid transaction")
		require.NotNil(t, receipt, "receipt must not be nil")
		require.Equal(t, "0x1", receipt["status"], "transaction must succeed")
		require.Equal(t, tx.Hash().Hex(), receipt["transactionHash"], "receipt must match submitted tx hash")

		// Verify receipt matches eth_getTransactionReceipt.
		var expectedReceipt map[string]interface{}
		err = client.Client().Call(&expectedReceipt, "eth_getTransactionReceipt", tx.Hash())
		require.NoError(t, err)
		require.Equal(t, expectedReceipt["transactionHash"], receipt["transactionHash"])
		require.Equal(t, expectedReceipt["blockNumber"], receipt["blockNumber"])
		require.Equal(t, expectedReceipt["status"], receipt["status"])
	})

	t.Run("nonce_gap_returns_code6_error", func(t *testing.T) {
		// Fresh account — pool expects nonce=0, but we send nonce=5.
		sender := tests.NewAccount()

		client, err := net.GetClient()
		require.NoError(t, err)
		defer client.Close()

		chainID, err := client.ChainID(t.Context())
		require.NoError(t, err)

		gasPrice, err := client.SuggestGasPrice(t.Context())
		require.NoError(t, err)

		receiver := tests.NewAccount()
		tx := tests.SignTransaction(t, chainID, &types.LegacyTx{
			Nonce:    5,
			Gas:      21000,
			GasPrice: gasPrice,
			To:       addrPtr(receiver.Address()),
			Value:    big.NewInt(0),
		}, sender)

		encoded, err := tx.MarshalBinary()
		require.NoError(t, err)

		var result map[string]interface{}
		err = client.Client().Call(&result, "eth_sendRawTransactionSync", hexutil.Bytes(encoded))
		require.Error(t, err, "nonce gap must return an error")

		rpcErr, ok := err.(rpc.Error)
		require.True(t, ok, "error must be an RPC error, got %T: %v", err, err)
		require.Equal(t, 6, rpcErr.ErrorCode(), "nonce gap must return error code 6")
	})

	t.Run("short_timeout_returns_timeout_error", func(t *testing.T) {
		session := net.SpawnSession(t)

		sender := tests.NewAccount()
		_, err := session.EndowAccount(sender.Address(), big.NewInt(1e18))
		require.NoError(t, err)

		client, err := net.GetClient()
		require.NoError(t, err)
		defer client.Close()

		chainID, err := client.ChainID(t.Context())
		require.NoError(t, err)

		nonce, err := client.PendingNonceAt(t.Context(), sender.Address())
		require.NoError(t, err)

		gasPrice, err := client.SuggestGasPrice(t.Context())
		require.NoError(t, err)

		receiver := tests.NewAccount()
		tx := tests.SignTransaction(t, chainID, &types.LegacyTx{
			Nonce:    nonce,
			Gas:      21000,
			GasPrice: gasPrice,
			To:       addrPtr(receiver.Address()),
			Value:    big.NewInt(1000),
		}, sender)

		encoded, err := tx.MarshalBinary()
		require.NoError(t, err)

		// 1ms timeout — will almost certainly not confirm in time.
		timeoutMs := hexutil.Uint64(1)

		var result map[string]interface{}
		err = client.Client().Call(&result, "eth_sendRawTransactionSync", hexutil.Bytes(encoded), timeoutMs)
		require.Error(t, err, "very short timeout must return an error")

		rpcErr, ok := err.(rpc.Error)
		require.True(t, ok, "error must be an RPC error, got %T: %v", err, err)
		// Code 4 (timeout, not in pool) or 5 (queued, tx still in pool).
		require.True(t,
			rpcErr.ErrorCode() == 4 || rpcErr.ErrorCode() == 5,
			"short timeout must return error code 4 or 5, got %d", rpcErr.ErrorCode(),
		)
	})
}

// addrPtr returns a pointer to the given address value.
func addrPtr[T any](v T) *T { return &v }
