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

package blockbyhash

import (
	"errors"
	"math/big"
	"strconv"
	"sync"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter_event_emitter"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestRPCGetLogs_BlockWithSkippedTransaction_HasCorrectTxIndexes(t *testing.T) {

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: tests.AsPointer(opera.GetBrioUpgrades()),
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
		_, err = net.ForceEmit(t.Context(), txSkip)
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

// Logs of the by-hash and unindexed queries are served from the receipts
// cache and therefore shared between requests. Run with -race to detect
// requests mutating them.
func TestRPCGetLogs_ConcurrentQueries_HaveCorrectTxIndexes(t *testing.T) {
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: tests.AsPointer(opera.GetBrioUpgrades()),
	})

	contract, receipt, err := tests.DeployContract(net, counter_event_emitter.DeployCounterEventEmitter)
	require.NoError(t, err)
	address := receipt.ContractAddress

	// Propose the transactions together so that the block holds logs of
	// several transaction indexes.
	accounts := tests.MakeAccountsWithBalance(t, net, 5, big.NewInt(1e18))
	txs := make([]*types.Transaction, len(accounts))
	for i, account := range accounts {
		opts, err := net.GetTransactOptions(account)
		require.NoError(t, err)
		opts.NoSend = true
		txs[i], err = contract.Increment(opts)
		require.NoError(t, err)
	}
	hashes, err := net.ForceEmitAll(t.Context(), txs)
	require.NoError(t, err)
	receipts, err := net.GetReceipts(hashes)
	require.NoError(t, err)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	block, err := client.BlockByNumber(t.Context(), receipts[0].BlockNumber)
	require.NoError(t, err)
	require.Len(t, block.Transactions(), len(txs))
	hash, number := block.Hash(), block.Number()

	queries := []func() ([]types.Log, error){
		func() ([]types.Log, error) {
			return client.FilterLogs(t.Context(), ethereum.FilterQuery{BlockHash: &hash})
		},
		func() ([]types.Log, error) {
			return client.FilterLogs(t.Context(), ethereum.FilterQuery{FromBlock: number, ToBlock: number})
		},
		func() ([]types.Log, error) {
			return client.FilterLogs(t.Context(), ethereum.FilterQuery{
				FromBlock: number, ToBlock: number, Addresses: []common.Address{address},
			})
		},
		func() ([]types.Log, error) {
			blockReceipts, err := client.BlockReceipts(t.Context(), rpc.BlockNumberOrHashWithHash(hash, false))
			var logs []types.Log
			for _, r := range blockReceipts {
				for _, l := range r.Logs {
					logs = append(logs, *l)
				}
			}
			return logs, err
		},
	}

	const numWorkers, numQueries = 16, 40
	results := make([][]types.Log, numWorkers)
	errs := make([]error, numWorkers)
	var wg sync.WaitGroup
	for w := range numWorkers {
		wg.Go(func() {
			for q := range numQueries {
				logs, err := queries[(w+q)%len(queries)]()
				results[w] = append(results[w], logs...)
				errs[w] = errors.Join(errs[w], err)
			}
		})
	}
	wg.Wait()
	require.NoError(t, errors.Join(errs...))

	for _, logs := range results {
		for _, log := range logs {
			require.Equal(t, number.Uint64(), log.BlockNumber)
			require.Less(t, log.TxIndex, uint(len(block.Transactions())))
			require.Equal(t, block.Transactions()[log.TxIndex].Hash(), log.TxHash)
		}
	}
}
