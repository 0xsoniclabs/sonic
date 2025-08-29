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
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter_event_emitter"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogSubscription_CanGetCallBacksForLogEvents(t *testing.T) {
	t.Parallel()

	const NumEvents = 3
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
	const NumBatches = 60
	require := require.New(t)

	// This test relies on transactions included in the block to be only
	// the transactions generated in the test itself, test may fail if other
	// blocks are generated in the background.
	//
	// For this reason this test uses a dedicated network.
	net := StartIntegrationTestNetWithJsonGenesis(t)

	contract, _, err := DeployContract(net, counter_event_emitter.DeployCounterEventEmitter)
	require.NoError(err)

	wsClient, err := net.GetWebSocketClient()
	require.NoError(err, "failed to get client; ", err)
	defer wsClient.Close()

	newHeadChannel := make(chan *types.Header)
	subscription, err := wsClient.SubscribeNewHead(t.Context(), newHeadChannel)
	require.NoError(err)

	shutdownTestRoutine := make(chan struct{})
	testRoutineDone := make(chan struct{})
	testRoutine := sync.Once{}

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		client, err := net.GetClient()
		require.NoError(err, "failed to get client; ", err)
		defer client.Close()

		for {
			select {
			case <-stop:
				return
			case <-time.After(100 * time.Millisecond):
			default:
			}

			block, err := client.BlockByNumber(t.Context(), nil)
			require.NoError(err)
			if len(block.Transactions()) == 0 || block.NumberU64() <= 2 {
				continue
			}
			if (types.Bloom{} == block.Bloom()) {
				t.Errorf("expected non-empty bloom in block %d", block.NumberU64())
			}
		}
	}()

	tasks := sync.WaitGroup{}
	tasks.Add(1)
	go func() {
		defer tasks.Done()
		defer subscription.Unsubscribe()
		defer close(newHeadChannel)

		client, err := net.GetClient()
		require.NoError(err, "failed to get client; ", err)
		defer client.Close()

		for range NumBatches {
			opts, err := net.GetTransactOptions(net.GetSessionSponsor())
			require.NoError(err)

			// accumulate 10 txs per block
			batch := []*types.Transaction{}
			for range 10 {
				opts.NoSend = true
				tx, err := contract.Increment(opts)
				require.NoError(err)
				batch = append(batch, tx)
				opts.Nonce.Add(opts.Nonce, big.NewInt(1))
			}

			for _, tx := range batch {
				err := client.SendTransaction(t.Context(), tx)
				require.NoError(err)
			}

			// wait for this batch before starting a new one
			// (this waits for block creation)
			hashes := make([]common.Hash, len(batch))
			for i, tx := range batch {
				hashes[i] = tx.Hash()
			}
			receipts, err := net.GetReceipts(hashes)
			require.NoError(err)

			for _, receipt := range receipts {
				require.NotEqual(
					types.Bloom{},
					receipt.Bloom,
					"expected non-empty bloom filter",
				)
			}

			blockNum := uint64(0)
			for _, receipt := range receipts {
				blockNum = max(blockNum, receipt.BlockNumber.Uint64())
			}

			// start test after first generation of blocks with logs and Bloom
			testRoutine.Do(func() {
				go pollHeadAndCheckBloom(t, net, shutdownTestRoutine, testRoutineDone)
			})
		}

	}()

	tasks.Add(1)
	go func() {
		defer tasks.Done()

		for head := range newHeadChannel {

			if head.Bloom == (types.Bloom{}) {
				assert.Fail(t, fmt.Sprintf("expected non-empty bloom filter in head for block %d", head.Number.Uint64()))
			}
		}
	}()

	tasks.Wait()

	close(stop)
	<-done

	close(shutdownTestRoutine)
	<-testRoutineDone
}

func pollHeadAndCheckBloom(t *testing.T,
	session IntegrationTestNetSession,
	finalized <-chan struct{}, done chan<- struct{},
) {
	defer close(done)

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	for {
		select {
		case <-finalized:
			return
		default:
		}

		lastHead, err := client.BlockByNumber(t.Context(), nil)
		require.NoError(t, err)

		if lastHead.Bloom() == (types.Bloom{}) {
			assert.Fail(t, fmt.Sprintf("expected non-empty bloom filter in head for block %d", lastHead.NumberU64()))
		}
	}
}

func TestLogBloom_query2(t *testing.T) {
	const NumBatches = 60
	require := require.New(t)

	// This test relies on transactions included in the block to be only
	// the transactions generated in the test itself, test may fail if other
	// blocks are generated in the background.
	//
	// For this reason this test uses a dedicated network.
	net := StartIntegrationTestNetWithJsonGenesis(t)

	contract, _, err := DeployContract(net, counter_event_emitter.DeployCounterEventEmitter)
	require.NoError(err)

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		client, err := net.GetClient()
		require.NoError(err, "failed to get client; ", err)
		defer client.Close()

		next := int64(0)
		for {
			select {
			case <-stop:
				return
			case <-time.After(time.Millisecond):
			default:
			}

			//header, err := client.HeaderByNumber(t.Context(), nil)
			//header, err := client.HeaderByNumber(t.Context(), big.NewInt(next))
			block, err := client.BlockByNumber(t.Context(), big.NewInt(next))
			if errors.Is(err, ethereum.NotFound) {
				continue
			}
			require.NoError(err)
			next++
			if block.NumberU64() <= 2 {
				continue
			}
			//fmt.Printf("Fetched header for block %d with bloom %x\n", header.Number.Uint64(), header.Bloom)
			if (types.Bloom{} == block.Bloom()) {
				t.Errorf("expected non-empty bloom in block %d", block.NumberU64())
			}
		}
	}()

	tasks := sync.WaitGroup{}
	tasks.Add(1)
	go func() {
		defer tasks.Done()

		client, err := net.GetClient()
		require.NoError(err, "failed to get client; ", err)
		defer client.Close()

		for i := range NumBatches {
			fmt.Printf("Running batch %d\n", i)
			opts, err := net.GetTransactOptions(net.GetSessionSponsor())
			require.NoError(err)

			// accumulate 10 txs per block
			batch := []*types.Transaction{}
			for range 10 {
				opts.NoSend = true
				tx, err := contract.Increment(opts)
				require.NoError(err)
				batch = append(batch, tx)
				opts.Nonce.Add(opts.Nonce, big.NewInt(1))
			}

			for _, tx := range batch {
				err := client.SendTransaction(t.Context(), tx)
				require.NoError(err)
			}

			// wait for this batch before starting a new one
			// (this waits for block creation)
			hashes := make([]common.Hash, len(batch))
			for i, tx := range batch {
				hashes[i] = tx.Hash()
			}

			receipts, err := net.GetReceipts(hashes)
			require.NoError(err)

			for _, receipt := range receipts {
				require.NotEqual(
					types.Bloom{},
					receipt.Bloom,
					"expected non-empty bloom filter",
				)
			}

			blockNum := uint64(0)
			for _, receipt := range receipts {
				blockNum = max(blockNum, receipt.BlockNumber.Uint64())
			}
		}

	}()

	tasks.Wait()

	close(stop)
	<-done
}
