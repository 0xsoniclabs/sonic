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

package gas_subsidies

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/indexed_logs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestGasSubsidies_ProperlyAssignTxIndexToLogsInThePresenceOfSponsoredTransactions(t *testing.T) {
	require := require.New(t)

	upgrade := opera.GetAllegroUpgrades()
	upgrade.GasSubsidies = true
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrade,
	})

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	sponsee := tests.NewAccount()
	donation := big.NewInt(1e18)
	Fund(t, net, sponsee.Address(), donation)

	contract, receipt, err := tests.DeployContract(net, indexed_logs.DeployIndexedLogs)
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	numTxs := 5
	hashes := make([]common.Hash, 0, numTxs)
	for range numTxs {
		txOpts, err := net.GetTransactOptions(sponsee)
		require.NoError(err)

		txOpts.GasPrice = big.NewInt(0)
		tx, err := contract.EmitEvents(txOpts)
		require.NoError(err)

		hashes = append(hashes, tx.Hash())
	}

	receipts, err := net.GetReceipts(hashes)
	require.NoError(err)

	// Check receipts and find first and last block numbers
	firstBlockNumber := receipts[0].BlockNumber
	lastBlockNumber := receipts[0].BlockNumber
	for _, receipt := range receipts {
		require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

		if receipt.BlockNumber.Cmp(firstBlockNumber) < 0 {
			firstBlockNumber = receipt.BlockNumber
		}
		if receipt.BlockNumber.Cmp(lastBlockNumber) > 0 {
			lastBlockNumber = receipt.BlockNumber
		}
	}

	// Ensure that the logs are in the correct order
	for blockNumber := firstBlockNumber.Uint64(); blockNumber <= lastBlockNumber.Uint64(); blockNumber++ {
		block, err := client.BlockByNumber(t.Context(), big.NewInt(int64(blockNumber)))
		require.NoError(err)

		seenLogs := 0
		var totalGasUsed uint64

		for i, tx := range block.Transactions() {
			receipt, err := client.TransactionReceipt(t.Context(), tx.Hash())
			require.NoError(err)

			totalGasUsed += receipt.GasUsed
			require.EqualValues(totalGasUsed, receipt.CumulativeGasUsed)

			require.EqualValues(i, receipt.TransactionIndex, "transaction index should match")
			for _, log := range receipt.Logs {
				require.Equal(tx.Hash(), log.TxHash, "log tx hash should match transaction hash")
				require.EqualValues(i, log.TxIndex, "log tx index should match transaction index")
				require.EqualValues(seenLogs, int(log.Index), "log index should match seen logs count")
				seenLogs++
			}
		}

		require.Greater(seenLogs, 0, "should have seen some logs")
	}
}
