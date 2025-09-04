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
	"errors"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter_event_emitter"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestLogSubscription_CanGetCallBacksForLogEvents(t *testing.T) {

	const NumEvents = 10
	require := require.New(t)
	session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())

	contract, _, err := DeployContract(session, counter_event_emitter.DeployCounterEventEmitter)
	require.NoError(err)

	client, err := session.GetWebSocketClient()
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
		_, err = session.Apply(contract.Increment)
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

func TestLogBloom_query(t *testing.T) {
	// the number of blocks queried by the test
	const numBlocks = 6
	require := require.New(t)

	// This test can reuse an existing blockchain history, but should not
	// run concurrently to other tests. It requires logs to exist in at least one
	// transaction per block while testFunction is running.
	net := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())

	contract, _, err := DeployContract(net, counter_event_emitter.DeployCounterEventEmitter)
	require.NoError(err)

	stopTest := make(chan struct{})
	testDone := make(chan struct{})
	// testFunction monitors the latest available block, ensuring it has logs.
	testFunction := func(blockNumber uint64) {
		defer close(testDone)
		client, err := net.GetClient()
		require.NoError(err, "failed to get client; ", err)
		defer client.Close()

		for {
			select {
			case <-stopTest:
				return
			default:
			}

			block, err := client.BlockByNumber(t.Context(), big.NewInt(int64(blockNumber)))
			if errors.Is(err, ethereum.NotFound) {
				continue
			}
			require.NoError(err)

			if (types.Bloom{} == block.Bloom()) {
				t.Errorf("expected non-empty bloom in block %d, got empty", block.NumberU64())
			}

			blockNumber++
		}
	}
	launchTest := sync.Once{}

	client, err := net.GetClient()
	require.NoError(err, "failed to get client; ", err)
	defer client.Close()

	for range numBlocks {
		opts, err := net.GetTransactOptions(net.GetSessionSponsor())
		require.NoError(err)

		// accumulate 10 txs per block
		tx, err := contract.Increment(opts)
		require.NoError(err)
		receipt, err := net.GetReceipt(tx.Hash())
		require.NoError(err)

		require.NotEqual(
			types.Bloom{},
			receipt.Bloom,
			"expected non-empty bloom filter",
		)

		// start the test function in a goroutine as soon as the
		// first block has been generated (this way the test does not)
		// need to deal with previous blocks not having logs
		launchTest.Do(
			func() {
				go testFunction(receipt.BlockNumber.Uint64())
			})
	}

	close(stopTest)
	<-testDone
}
