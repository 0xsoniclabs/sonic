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

package tests

import (
	"maps"
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/emitter"
	"github.com/0xsoniclabs/sonic/version"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestEventsContainExtraData(t *testing.T) {
	const numNodes = 2
	net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		NumNodes: numNodes,
	})

	// test for multiple epochs
	coveredBlocks := make(map[uint64]struct{})
	for range 3 {
		// Produce a bit of load ..
		for i := range 10 {
			_, err := net.EndowAccount(common.Address{byte(i)}, big.NewInt(1))
			require.NoError(t, err)
		}

		// Get all events fo the current epoch.
		events := getAllEventsOfCurrentEpoch(t, net)
		require.NotEmpty(t, events)

		// Fetch all headers to check reported data.
		headers, err := net.GetHeaders()
		require.NoError(t, err)
		require.NotEmpty(t, headers)

		maxBlock := uint64(0)
		hashIndex := map[uint64]common.Hash{}
		for _, header := range headers {
			blockNum := header.Number.Uint64()
			hashIndex[blockNum] = header.Hash()
			maxBlock = max(maxBlock, blockNum)
		}

		// Check that the extra data in the events is reasonable.
		numGenesisEvents := 0
		for _, event := range events {
			reportedVersion, blockInfo, err := emitter.DecodeExtraData(event.ExtraData)
			require.NoError(t, err)

			// Check that the client version is included in genesis events.
			if event.SeqNumber == 1 {
				require.NotNil(t, reportedVersion)
				require.Equal(t, version.Get().String(), *reportedVersion)
				numGenesisEvents++
			} else {
				require.Nil(t, reportedVersion)
			}

			// Check that reported block hashes are reasonable.
			require.NotNil(t, blockInfo)
			coveredBlocks[blockInfo.Number] = struct{}{}
			if blockInfo.Number <= maxBlock {
				wantedHash, knownBlock := hashIndex[blockInfo.Number]
				require.True(t, knownBlock)
				require.Equal(t, wantedHash, blockInfo.Hash)
			}
		}
		require.Equal(t, numNodes, numGenesisEvents)
		require.Greater(t, len(coveredBlocks), len(headers)*3/4)

		// Move to next epoch.
		net.AdvanceEpoch(t, 1)
	}
}

type EventData struct {
	SeqNumber uint64
	Parents   []hash.Event
	ExtraData []byte
}

func getAllEventsOfCurrentEpoch(t *testing.T, net *IntegrationTestNet) []EventData {
	t.Helper()

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// Start with the heads
	heads := GetEventHeads(t, client)

	missingEvents := heads
	resolved := map[hash.Event]EventData{}
	for len(missingEvents) > 0 {
		eventID := missingEvents[len(missingEvents)-1]
		missingEvents = missingEvents[:len(missingEvents)-1]

		if _, resolved := resolved[eventID]; resolved {
			continue
		}

		var result struct {
			Seq       hexutil.Uint64  `json:"seq"`
			Parents   []hexutil.Bytes `json:"parents"`
			ExtraData hexutil.Bytes   `json:"extraData"`
			// other fields are ignored
		}
		err := client.Client().Call(&result, "dag_getEvent", eventID.Hex())
		require.NoError(t, err)

		event := EventData{
			SeqNumber: uint64(result.Seq),
			ExtraData: result.ExtraData,
		}
		for _, parent := range result.Parents {
			event.Parents = append(event.Parents, hash.Event(common.HexToHash(parent.String())))
		}

		// store event as resolved and add parents to missing list
		resolved[eventID] = event
		missingEvents = append(missingEvents, event.Parents...)
	}

	return slices.Collect(maps.Values(resolved))
}
