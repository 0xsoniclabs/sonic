package tests

import (
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/tests/contracts/counter_event_emitter"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestLogSubscription_CanGetCallBacksForLogEvents(t *testing.T) {
	const NumEvents = 3
	require := require.New(t)
	net := StartIntegrationTestNet(t)

	contract, _, err := DeployContract(net, counter_event_emitter.DeployCounterEventEmitter)
	require.NoError(err)

	client, err := net.GetWebSocketClient()
	require.NoError(err, "failed to get client; ", err)
	defer client.Close()

	allLogs := make(chan types.Log, NumEvents)
	subscription, err := client.SubscribeFilterLogs(
		t.Context(),
		ethereum.FilterQuery{},
		allLogs,
	)
	require.NoError(err, "failed to subscribe to logs; ", err)
	defer subscription.Unsubscribe()

	for range NumEvents {
		_, err = net.Apply(contract.Increment)
		require.NoError(err)
	}

	for i := range NumEvents {
		select {
		case log := <-allLogs:
			event, err := contract.ParseCount(log)
			require.NoError(err)
			require.Equal(uint64(i+1), event.TotalCount.Uint64())
		case <-time.After(5 * time.Second):
			require.Fail("expected log event not received")
		}
	}
}
