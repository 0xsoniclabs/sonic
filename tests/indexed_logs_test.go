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
		if current.BlockNumber > next.BlockNumber {
			t.Errorf("BlockNumber out of order: %d > %d", current.BlockNumber, next.BlockNumber)
		}

		// If BlockNumbers are equal, check if Index is non-decreasing
		if current.BlockNumber == next.BlockNumber && current.Index >= next.Index {
			t.Errorf("Index out of order for BlockNumber %d: current log %d > next log %d", current.BlockNumber, current.Index, next.Index)
		}
	}
}
