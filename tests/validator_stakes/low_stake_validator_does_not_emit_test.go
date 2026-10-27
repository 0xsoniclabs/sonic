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

package many

import (
	"fmt"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

// emitInterval is the interval each validator waits between two events.
// The integration test network defaults to 1ms, which is far below the 11ms
// period in which an emitter checks whether to emit. The emission rate is then
// bound by how often each node's emitter gets to run rather than by the
// interval, and on a busy machine the two validators drift apart by more than
// the tolerance asserted below. An interval well above the check period makes
// both of them emit at the same, clock-bound rate, while still being short
// enough to keep the test fast.
const emitInterval = 50 * time.Millisecond

func TestEventThrottler_NonDominantValidatorsProduceLessEvents_WhenEventThrottlerIsEnabled(t *testing.T) {
	// this test checks that when event throttler is enabled,
	// it collects events from the dag of the first validator after some time,
	// and ensures that the validator with low stake produces significantly less events
	// compared to the dominant validator if the feature is enabled, equally when disabled.
	// This test only queries the first node's DAG, as it is sufficient to verify the behavior.

	for _, throttlerEnabled := range []bool{true, false} {
		t.Run(fmt.Sprintf("emitter_throttle_events=%v", throttlerEnabled), func(t *testing.T) {

			// Start a network with many nodes where one node has very low stake
			initialStake := []uint64{
				1600, // 80% of stake: validatorId 1
				400,  // 20% of stake: validatorId 2
			}

			extraArguments := []string{
				fmt.Sprintf("--event-throttler=%t", throttlerEnabled),
			}

			net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
				ValidatorsStake:      initialStake,
				ClientExtraArguments: extraArguments,
				ModifyConfig: func(config *config.Config) {
					config.Emitter.EmitIntervals.Min = emitInterval
					config.Emitter.EmitIntervals.Confirming = emitInterval
				},
			})

			net.AdvanceEpoch(t, 1)

			// Poll until enough events are collected for statistical stability.
			// Only events created once every validator has resumed emitting in
			// the new epoch are considered, see eventsAfterAllValidatorsResumed.
			// Each poll fetches the full epoch DAG event by event, so it must
			// not run so often that it slows down the node it measures.
			const minEvents = 120
			var eventsInWindow eventMap
			require.Eventually(t, func() bool {
				eventsInWindow = eventsAfterAllValidatorsResumed(
					getEventsInEpoch(t, net), len(initialStake))
				return len(eventsInWindow) >= minEvents
			}, 100*time.Second, 250*time.Millisecond,
				"timed out waiting for at least %d events created while all validators were emitting", minEvents)

			percentages := calculateValidatorEmissionPercentages(eventsInWindow)

			if throttlerEnabled {
				require.GreaterOrEqual(t, percentages[1], 0.9,
					"High stake validator should dominate event creation")
				require.LessOrEqual(t, percentages[2], 0.1,
					"Low stake validator should create very few events")
			} else {
				// Without emitter throttling, both validators should create the same amount of events
				require.InDelta(t, percentages[1], percentages[2], 0.1,
					"Both validators should create equal amount of events")
			}
		})
	}
}

type eventMap map[hash.Event]testEvent

type testEvent struct {
	Epoch        idx.Block
	Id           hash.Event
	Creator      idx.ValidatorID
	CreationTime inter.Timestamp
	Parents      []hash.Event
}

// eventsAfterAllValidatorsResumed reduces the given events to those created
// after the last of the numValidators validators has created its first event
// of the epoch.
//
// After an epoch change validators do not resume emitting at the same instant;
// one of them can be several hundred milliseconds ahead of the others. At an
// emitInterval of 50ms such a delay is worth a dozen events, a tenth of the
// events this test samples, and would distort any measurement of the relative
// emission rates. The events created before all validators are back are
// therefore not representative and get discarded.
//
// An empty map is returned while some validator has not emitted any event yet.
func eventsAfterAllValidatorsResumed(events eventMap, numValidators int) eventMap {
	firstEventTime := map[idx.ValidatorID]inter.Timestamp{}
	for _, event := range events {
		cur, found := firstEventTime[event.Creator]
		if !found || event.CreationTime < cur {
			firstEventTime[event.Creator] = event.CreationTime
		}
	}
	if len(firstEventTime) < numValidators {
		return eventMap{}
	}

	// The window starts when the last validator created its first event.
	var windowStart inter.Timestamp
	for _, first := range firstEventTime {
		windowStart = max(windowStart, first)
	}

	inWindow := eventMap{}
	for id, event := range events {
		if event.CreationTime >= windowStart {
			inWindow[id] = event
		}
	}
	return inWindow
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
	event.CreationTime = inter.Timestamp(toUint64(result["creationTime"].(string)))
	event.Id = hash.Event(common.HexToHash(result["id"].(string)))
	event.Parents = make([]hash.Event, 0)
	for _, parent := range result["parents"].([]any) {
		event.Parents = append(event.Parents, hash.Event(common.HexToHash(parent.(string))))
	}

	return event
}

func calculateValidatorEmissionPercentages(
	allEvents eventMap,
) map[idx.ValidatorID]float64 {

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
