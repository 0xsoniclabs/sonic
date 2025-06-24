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
	"testing"

	"github.com/0xsoniclabs/sonic/tests/contracts/transientstorage"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestBlockInArchive checks that the block is in archive
// and that the block number notified is in order
// For failure on empty fakenet archive it was needed to modify
// carmen database to slow down async writes to archive db.
func TestBlockInArchive(t *testing.T) {

	require := require.New(t)
	net := StartIntegrationTestNetWithJsonGenesis(t)
	defer net.Stop()

	client, err := net.GetWebSocketClient()
	require.NoError(err, "failed to get client ", err)
	defer client.Close()
	done := make(chan struct{})

	go func() {

		defer close(done)
		rpcClient := client.Client()
		headChannel := make(chan *types.Header)
		subscription, err := client.SubscribeNewHead(t.Context(), headChannel)
		require.NoError(err, "failed to subscribe to new head %v", err)
		lastBlockNumber := uint64(0)

		for {
			select {
			case blockHeader := <-headChannel:

				// Check if block is in archive
				var res interface{}
				err := rpcClient.Call(&res, "eth_getBalance", net.account.Address().String(), hexutil.EncodeUint64(blockHeader.Number.Uint64()))
				if err != nil {
					require.NoError(err, "failed to call eth_getBalance %v", err)
				}

				// Check that the block number is in order
				if lastBlockNumber == 0 || blockHeader.Number.Uint64() == lastBlockNumber+1 {
					lastBlockNumber = blockHeader.Number.Uint64()
				} else {
					subscription.Unsubscribe()
					require.NoError(fmt.Errorf("received block number is not in correct order, expected %v, got %v", (lastBlockNumber + 1), blockHeader.Number.Uint64()))
				}

				if blockHeader.Number.Uint64() == 20 {
					return
				}
			case err = <-subscription.Err():
				require.NoError(err, "subscription error %v", err)
			}
		}
	}()

	contract, _, err := DeployContract(net, transientstorage.DeployTransientstorage)
	if err != nil {
		t.Errorf("failed to deploy contract %v", err)
	}

	for {
		select {
		case <-done:
			return
		default:
			txOptions, err := net.GetTransactOptions(net.GetSessionSponsor())
			require.NoError(err, "failed to get transaction options %v", err)
			txOptions.Nonce = nil
			txOptions.GasLimit = 0

			_, err = contract.StoreValue(txOptions)
			require.NoError(err, "failed to send transaction %v", err)
		}
	}
}
