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
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestRpc_ChainViewIncludesTransactionAsSoonAsItsReceiptIsServed checks the
// invariant every Ethereum client relies on: once eth_getTransactionReceipt
// answers for a transaction, the node presents a chain view that includes it.
// The head must cover the block of the receipt, and calls resolving "latest"
// must observe the effects of the transaction.
//
// A node holds this invariant by publishing a block only after its state, its
// transaction index and its receipts are in place: block processing waits for
// the archive commit before storing the block, and advances the latest block
// index last. The guard against serving a block ahead of that index lives in
// the state reader and is covered by unit tests; this test pins the resulting
// behaviour down at the RPC level, where a client observes it.
//
// The archive commit is only awaited for blocks less than an hour old, see
// evmmodule.OperaEVMProcessor.Finalize. While catching up, older blocks are
// committed asynchronously, so the archive can trail the latest block index and
// state queries resolving "latest" may answer from a block preceding a receipt
// that was already served. This test runs against a live network and therefore
// only covers the steady state.
//
// It does not reproduce a race on its own — the window it guards is currently
// too narrow to hit. Its purpose is to fail loudly if a later change widens it,
// for instance by moving the archive commit off the block processing path.
//
// The receipt is polled with a plain RPC call rather than through the session
// helpers, which wait for the node to become consistent on their own and would
// therefore mask exactly what is under test.
func TestRpc_ChainViewIncludesTransactionAsSoonAsItsReceiptIsServed(t *testing.T) {
	const numRepetitions = 25

	net := StartIntegrationTestNet(t)
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	value := big.NewInt(1e15)
	for range numRepetitions {
		address := NewAccount().Address()
		tx := CreateTransaction(t, net, &types.LegacyTx{
			To:    &address,
			Value: value,
		}, net.GetSessionSponsor())
		require.NoError(t, client.SendTransaction(t.Context(), tx))

		var receipt *types.Receipt
		require.NoError(t, WaitFor(t.Context(), func(ctx context.Context) (bool, error) {
			res, err := client.TransactionReceipt(ctx, tx.Hash())
			if errors.Is(err, ethereum.NotFound) {
				return false, nil
			}
			if err != nil {
				return false, err
			}
			receipt = res
			return true, nil
		}))
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

		// No further waiting beyond this point: the node has committed to the
		// transaction being part of the chain and must present it as such.
		head, err := client.BlockNumber(t.Context())
		require.NoError(t, err)
		require.GreaterOrEqual(t, head, receipt.BlockNumber.Uint64(),
			"the head must cover the block of a receipt that was served")

		balance, err := client.BalanceAt(t.Context(), address, nil)
		require.NoError(t, err)
		require.Equal(t, value, balance,
			"the state at latest must include the transaction of the receipt")

		gasPrice, err := client.SuggestGasPrice(t.Context())
		require.NoError(t, err)
		_, err = client.EstimateGas(t.Context(), ethereum.CallMsg{
			From:     address,
			To:       &common.Address{},
			GasPrice: gasPrice,
			Value:    big.NewInt(1),
		})
		require.NoError(t, err,
			"gas estimation at latest must see the funds of the new account")
	}
}
