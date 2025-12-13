package longsocket

import (
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestLongWebSocket_DoesNotHang(t *testing.T) {
	// This is a placeholder for a test that ensures long WebSocket connections
	// do not hang indefinitely. The actual implementation would depend on the
	// specifics of the long WebSocket functionality being tested.

	net := tests.StartIntegrationTestNet(t)

	client, err := net.GetWebSocketClient()
	require.NoError(t, err)
	defer client.Close()

	newBlocks := make(chan *evmcore.EvmBlockJson)
	subs, err := client.Client().EthSubscribe(t.Context(), newBlocks, "newHeads")
	require.NoError(t, err)
	defer subs.Unsubscribe()

	startTime := time.Now()
	newBlockCounter := 0
	newBlockInLastMinute := false
	timeSinceLastBlock := time.Now()
	ticker := time.NewTicker(1 * time.Minute)
	sendTxTicker := time.NewTicker(1 * time.Second)
	for time.Since(startTime) < 24*time.Hour+5*time.Minute {
		select {
		case <-newBlocks:
			newBlockCounter++
			newBlockInLastMinute = true
			t.Logf("Received block %v after %v.",
				newBlockCounter, time.Since(timeSinceLastBlock))
			timeSinceLastBlock = time.Now()
		case <-sendTxTicker.C:
			basicTx := tests.CreateTransaction(t, net, &types.LegacyTx{}, net.GetSessionSponsor())
			err := client.SendTransaction(t.Context(), basicTx)
			require.NoError(t, err, "failed to send transaction to keep blocks coming")
		case <-ticker.C:
			if newBlockInLastMinute {
				newBlockInLastMinute = false
			} else {
				t.Fatalf("No new blocks received in the last minute, WebSocket might be hanging")
			}
		case err := <-subs.Err():
			require.NoError(t, err, "subscription error")
		}
	}
}
