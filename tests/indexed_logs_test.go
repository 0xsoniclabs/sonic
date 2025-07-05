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
	"testing"

	"github.com/0xsoniclabs/sonic/tests/contracts/indexed_logs"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestClient_IndexedLogsAreInOrder(t *testing.T) {
	net := StartIntegrationTestNetWithFakeGenesis(t)

	contract, receipt, err := DeployContract(net, indexed_logs.DeployIndexedLogs)
	require.NoError(t, err)
	contractAddress := receipt.ContractAddress

	// Create logs
	txReceiptBlock1, err := net.net.Apply(contract.EmitEvents)
	require.NoError(t, err)
	logs := txReceiptBlock1.Logs

	// Create logs in another block
	txReceiptBlock2, err := net.net.Apply(contract.EmitEvents)
	require.NoError(t, err)

	require.NotEqual(t, txReceiptBlock1.BlockNumber, txReceiptBlock2.BlockNumber, "Logs should be in different blocks")

	client, err := net.net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// Search in indexed logs in parallel
	blockLogs, err := client.FilterLogs(t.Context(), ethereum.FilterQuery{
		FromBlock: txReceiptBlock1.BlockNumber,
		ToBlock:   txReceiptBlock2.BlockNumber,
		Addresses: []common.Address{contractAddress},
		Topics:    [][]common.Hash{{logs[0].Topics[0], logs[1].Topics[0], logs[2].Topics[0]}},
	})
	require.NoError(t, err)
	// EmitEvents is called twice and contract produces 3 logs 5 times
	require.Len(t, blockLogs, 2*5*3)

	for i := 0; i < len(blockLogs)-1; i++ {
		current := blockLogs[i]
		next := blockLogs[i+1]

		// Check if BlockNumber is non-decreasing
		require.LessOrEqual(t, current.BlockNumber, next.BlockNumber)

		// If BlockNumbers are equal, check if Index is non-decreasing
		if current.BlockNumber == next.BlockNumber && current.Index >= next.Index {
			t.Errorf("Index out of order for BlockNumber %d: current log %d > next log %d", current.BlockNumber, current.Index, next.Index)
		}
	}
}
