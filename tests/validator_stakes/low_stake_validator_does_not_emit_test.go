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

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestEventThrottler_NonDominantValidatorsProduceLessEvents_WhenEventThrottlerIsEnabled(t *testing.T) {
	// this test checks that when event throttler is enabled,
	// it collects events from the dag of the first validator after some time,
	// and ensures that the validator with low stake produces significantly less events
	// compared to the dominant validator if the feature is enabled, equally when disabled.
	// This test only queries the first node's DAG, as it is sufficient to verify the behavior.

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

			tests.AdvanceEpochAndWaitForBlocks(t, net)

			// wait until some events are generated
			time.Sleep(1 * time.Second)

			eventsInEpoch := getEventsInEpoch(t, net)

			percentages := calculateValidatorEmissionPercentages(t, eventsInEpoch)

			if eventThrottler {
				require.GreaterOrEqual(t, percentages[1], 0.9,
					"High stake validator should dominate event creation")
				require.LessOrEqual(t, percentages[2], 0.1,
					"Low stake validator should create very few events")
			} else {
				// Without emitter throttling, both validators should create the same amount of events
				require.InDelta(t, percentages[1], percentages[2], 0.05,
					"Both validators should create equal amount of events")
			}
		})
	}
}

type eventMap map[hash.Event]testEvent

type testEvent struct {
	Epoch   idx.Block
	Id      hash.Event
	Creator idx.ValidatorID
	Parents []hash.Event
}

// getEventsInEpoch returns the events created in the current epoch up to the latest event heads.
// these events are collected from default client
func getEventsInEpoch(t *testing.T, net *tests.IntegrationTestNet) eventMap {
	t.Helper()

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	eventsInEpoch := eventMap{}
	eventIDs := tests.GetEventHeads(t, client)

	for _, eventID := range eventIDs {
		event := fetchEvent(t, client, eventID)
		eventsInEpoch[eventID] = event
	}

	for _, event := range eventsInEpoch {
		collectEventsAncestry(t, client, event, eventsInEpoch)
	}
	return eventsInEpoch
}

// collectEventsAncestry recursively collects all ancestor events of the given event
func collectEventsAncestry(
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
		collectEventsAncestry(t, client, event, ancestry)
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

func calculateValidatorEmissionPercentages(
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
