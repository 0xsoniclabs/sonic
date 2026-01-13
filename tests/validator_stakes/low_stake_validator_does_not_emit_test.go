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

package many

import (
	"fmt"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

type eventIdSet map[hash.Event]struct{}
type eventMap map[hash.Event]testEvent

type testEvent struct {
	Epoch   idx.Block
	Id      hash.Event
	Creator idx.ValidatorID
	Parents []hash.Event
}

func TestEventThrottler_NonDominantValidatorsProduceLessEvents_WhenEventThrottlerIsEnabled(t *testing.T) {

	for _, eventThrottler := range []bool{true, false} {
		t.Run(fmt.Sprintf("emitter_throttle_events=%v", eventThrottler), func(t *testing.T) {

			// Start a network with many nodes where one node has very low stake
			initialStake := []uint64{
				1600, // 80% of stake: validatorId 1
				400,  // 20% of stake: validatorId 2
			}

			clientExtraArgs := []string{}
			if eventThrottler {
				clientExtraArgs = []string{"--event-throttler.enable"}
			}

			net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
				ValidatorsStake:      initialStake,
				ClientExtraArguments: clientExtraArgs,
			})

			client, err := net.GetClient()
			require.NoError(t, err)
			defer client.Close()

			var currentRules opera.Rules
			err = client.Client().Call(&currentRules, "eth_getRules", "latest")
			require.NoError(t, err)

			tests.AdvanceEpochAndWaitForBlocks(t, net)
			currentEpoch := getCurrentEpoch(t, client)
			eventsInEpoch := eventMap{}

			// wait until some events are generated
			time.Sleep(1 * time.Second)

			heads := getEventIdsForCurrentEpoch(t, net, currentEpoch)
			for eventID := range heads {
				event := fetchEvent(t, client, eventID)
				eventsInEpoch[eventID] = event
			}

			for _, event := range eventsInEpoch {
				fetchAncestry(t, client, event, eventsInEpoch)
			}

			percentages := calculatePercentages(t, eventsInEpoch)

			if eventThrottler {
				require.GreaterOrEqual(t, percentages[1], 0.9,
					"High stake validator should create at least its stake proportion of events")
				require.LessOrEqual(t, percentages[2], 0.1,
					"Low stake validator should not create more than its stake proportion of events")
			} else {
				// Without emitter throttling, both validators should create the same amount of events
				require.InDelta(t, percentages[1], percentages[2], 0.05,
					"Both validators should create events roughly in proportion to their stake")
			}
		})
	}
}

// getEventIdsForCurrentEpoch populates the provided eventIdSet with event IDs from the target epoch.
func getEventIdsForCurrentEpoch(
	t *testing.T,
	net *tests.IntegrationTestNet,
	targetEpoch uint64) eventIdSet {
	t.Helper()

	events := eventIdSet{}

	for i := range net.NumNodes() {
		client, err := net.GetClientConnectedToNode(i)
		require.NoError(t, err)
		defer client.Close()

		currentEpoch := getCurrentEpoch(t, client)
		if currentEpoch != targetEpoch {
			// If the test fails here, is because the test time is close to the duration of an epoch
			// and one of the nodes already moved into the next epoch, because of the rpc only serving
			// events within the current epoch, the test cannot longer be conducted.
			require.Fail(t, "node %d moved to the next epoch, and the test cannot longer be conducted", i)
		}

		// Get the current epoch.
		eventIDs := tests.GetEventsFromEpoch(t, client, int(currentEpoch))
		for _, eventID := range eventIDs {
			events[eventID] = struct{}{}
		}
	}
	return events
}

// getCurrentEpoch retrieves the current epoch number from the blockchain.
func getCurrentEpoch(t *testing.T, client *tests.PooledEhtClient) uint64 {
	t.Helper()

	var epoch hexutil.Uint64
	err := client.Client().Call(&epoch, "eth_currentEpoch")
	require.NoError(t, err)
	return uint64(epoch)
}

// fetchAncestry recursively fetches the ancestry of the given event ID and populates the ancestry map.
func fetchAncestry(
	t *testing.T,
	client *tests.PooledEhtClient,
	event testEvent,
	ancestry eventMap) {
	t.Helper()

	for _, parentHash := range event.Parents {
		if _, exists := ancestry[parentHash]; exists {
			continue
		}
		event := fetchEvent(t, client, parentHash)
		ancestry[parentHash] = event
		fetchAncestry(t, client, event, ancestry)
	}
}

// fetchEvent retrieves the event details for the given event ID.
func fetchEvent(t *testing.T, client *tests.PooledEhtClient, eventID hash.Event) testEvent {
	var result map[string]any
	err := client.Client().Call(&result, "dag_getEvent", eventID.Hex())
	require.NoError(t, err)

	var event testEvent

	toUint64 := func(encoded string) uint64 {
		var unmarshal hexutil.Uint64
		err := unmarshal.UnmarshalText([]byte(encoded))
		require.NoError(t, err)
		return uint64(unmarshal)
	}

	event.Epoch = idx.Block(toUint64(result["epoch"].(string)))
	event.Creator = idx.ValidatorID(toUint64(result["creator"].(string)))
	event.Id = hash.Event(common.HexToHash(result["id"].(string)))
	event.Parents = make([]hash.Event, 0)
	for _, parent := range result["parents"].([]any) {
		event.Parents = append(event.Parents, hash.Event(common.HexToHash(parent.(string))))
	}

	return event
}

// calculatePercentages computes the percentage of events created by each validator.
func calculatePercentages(
	t *testing.T,
	allEvents eventMap,
) map[idx.ValidatorID]float64 {
	t.Helper()

	counts := map[idx.ValidatorID]float64{}
	for _, event := range allEvents {
		creator := event.Creator
		counts[creator]++
	}

	for id, count := range counts {
		counts[id] = count / float64(len(allEvents))
	}
	return counts
}
