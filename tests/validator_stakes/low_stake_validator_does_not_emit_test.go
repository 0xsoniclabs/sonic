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
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

type eventIdSet map[string]struct{}
type eventMap map[string]map[string]any
type jsonEvent map[string]any

func TestLowStakeValidator_DoesNotEmitMoreThanStakeProportion(t *testing.T) {

	for _, emitterEnabled := range []bool{true, false} {
		t.Run(fmt.Sprintf("emitter_throttle_events=%v", emitterEnabled), func(t *testing.T) {

			// Start a network with many nodes where one node has very low stake
			initialStake := []uint64{
				1600, // 80% of stake
				400,  // 20% of stake
			}

			clientExtraArgs := []string{}
			if emitterEnabled {
				clientExtraArgs = []string{"--emitter.throttle-events"}
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
			epochEvents := eventMap{}
			heads := eventIdSet{}

			// wait until some events are generated
			time.Sleep(10 * time.Second)

			getEventIdsForCurrentEpoch(t, net, currentEpoch, heads)
			for eventID := range heads {
				event := fetchEvent(t, client, eventID)
				epochEvents[eventID] = event
			}

			for _, event := range epochEvents {
				fetchAncestry(t, client, event["id"].(string), currentEpoch, epochEvents)
			}

			eventCreatorCount := make(map[string]int)
			for _, event := range epochEvents {
				creator := event["creator"].(string)
				eventCreatorCount[creator]++
			}

			percentages, stakePercentages := calculatePercentages(t, eventCreatorCount, initialStake)

			if emitterEnabled {
				require.GreaterOrEqual(t, percentages[0], stakePercentages[0],
					"High stake validator should create at least its stake proportion of events")
				require.LessOrEqual(t, percentages[1], stakePercentages[1],
					"Low stake validator should not create more than its stake proportion of events")
			} else {
				// Without emitter throttling, both validators should create events roughly in proportion to their stake
				require.InDelta(t, percentages[0], percentages[1], 0.05,
					"Both validators should create events roughly in proportion to their stake")
			}
		})
	}
}

// getEventIdsForCurrentEpoch populates the provided eventIdSet with event IDs from the target epoch.
func getEventIdsForCurrentEpoch(
	t *testing.T,
	net *tests.IntegrationTestNet,
	targetEpoch uint64,
	events eventIdSet) {
	t.Helper()

	for i := range net.NumNodes() {
		client, err := net.GetClientConnectedToNode(i)
		require.NoError(t, err)
		defer client.Close()

		currentEpoch := getCurrentEpoch(t, client)
		if currentEpoch != targetEpoch {
			return
		}

		// Get the current epoch.
		eventIDs := tests.GetEventsFromEpoch(t, client, int(currentEpoch))
		for _, eventID := range eventIDs {
			events[eventID.String()] = struct{}{}
		}
	}
}

// getCurrentEpoch retrieves the current epoch number from the blockchain.
func getCurrentEpoch(t *testing.T, client *tests.PooledEhtClient) uint64 {
	t.Helper()

	block := struct {
		Number hexutil.Uint64
		Epoch  hexutil.Uint64
	}{}
	err := client.Client().Call(&block, "eth_getBlockByNumber", rpc.BlockNumber(-1), false)
	require.NoError(t, err)
	return uint64(block.Epoch)
}

// fetchAncestry recursively fetches the ancestry of the given event ID and populates the ancestry map.
func fetchAncestry(t *testing.T, client *tests.PooledEhtClient, eventID string, currentEpoch uint64, ancestry eventMap) {
	t.Helper()

	olderEpoch := false
	event := ancestry[eventID]
	eventParent := event["parents"].([]any)

	for _, parentID := range eventParent {
		parentIDStr := parentID.(string)
		if _, exists := ancestry[parentIDStr]; !exists {
			event := fetchEvent(t, client, parentIDStr)
			if eventEpoch, ok := ancestry[parentIDStr]["epoch"].(hexutil.Uint64); ok {
				if uint64(eventEpoch) < currentEpoch {
					olderEpoch = true
					break
				}
			}
			ancestry[parentIDStr] = event
			fetchAncestry(t, client, parentIDStr, currentEpoch, ancestry)
		}
	}
	if olderEpoch {
		return
	}
}

// fetchEvent retrieves the event details for the given event ID.
func fetchEvent(t *testing.T, client *tests.PooledEhtClient, eventID string) jsonEvent {
	event := jsonEvent{}
	err := client.Client().Call(&event, "dag_getEvent", eventID)
	require.NoError(t, err)
	return event
}

// calculatePercentages computes the percentage of events created by each validator.
func calculatePercentages(
	t *testing.T,
	eventCreatorCount map[string]int,
	initialStake []uint64,
) ([]float64, []float64) {
	t.Helper()

	percentages := make([]float64, 0)
	stakePercentages := make([]float64, 0)
	totalEvents := 0
	for _, count := range eventCreatorCount {
		totalEvents += count
	}
	totalStake := uint64(0)
	for _, stake := range initialStake {
		totalStake += stake
	}

	for i, stake := range initialStake {
		creator := fmt.Sprintf("0x%v", i+1)
		count := eventCreatorCount[creator]
		percentage := float64(count) / float64(totalEvents)
		stakePercentage := float64(stake) / float64(totalStake)

		percentages = append(percentages, percentage)
		stakePercentages = append(stakePercentages, stakePercentage)
	}
	return percentages, stakePercentages
}
