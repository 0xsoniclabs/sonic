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

package blockbyhash

import (
	"math/big"
	"strconv"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter_event_emitter"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestRPCGetLogs_BlockWithSkippedTransaction_HasCorrectTxIndexes(t *testing.T) {

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: tests.AsPointer(opera.GetAllegroUpgrades()),
		ClientExtraArguments: []string{
			"--disable-txPool-validation",
		},
	})

	contract, receipt, err := tests.DeployContract(net, counter_event_emitter.DeployCounterEventEmitter)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// This test depends on how the scramblers schedules the 2 transaction sent.
	// There is a 50% chance of triggering the issue, hence we run it multiple times
	// as to have a fair chance of stimulating the issue but also not take too long in CI.
	for range 5 {

		// Make a transaction to be skipped.
		accountSkipped := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
		initCode := make([]byte, 50000)
		txSkip := tests.CreateTransaction(t, net, &types.LegacyTx{
			Gas:  10_000_000,
			To:   nil, // address 0x00 for contract creation
			Data: initCode,
		}, accountSkipped)

		// Send the transaction
		err = client.SendTransaction(t.Context(), txSkip)
		require.NoError(t, err)

		// make a transaction to log something
		receipt, err = net.Apply(contract.Increment)
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

		// get the block from the receipt
		block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
		require.NoError(t, err)

		// the test needs to call BlockByHash to reproduce the issue
		blockByHash, err := client.BlockByHash(t.Context(), block.Hash())
		require.NoError(t, err)
		require.NotNil(t, blockByHash)

		// get logs
		var logs []map[string]any
		err = client.Client().Call(&logs, "eth_getLogs", map[string]any{
			"blockHash": block.Hash(),
		})
		require.NoError(t, err)
		require.Greater(t, len(logs), 0)

		for _, log := range logs {
			txIndexHex := log["transactionIndex"].(string)
			txIndex, err := strconv.ParseUint(txIndexHex[2:], 16, 64)
			require.NoError(t, err)

			require.Less(t, txIndex, uint64(len(blockByHash.Transactions())), "tx index out of range")

			tx := blockByHash.Transactions()[txIndex]
			require.Equal(t, tx.Hash().Hex(), log["transactionHash"].(string))
		}
	}
}
