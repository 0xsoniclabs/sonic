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
