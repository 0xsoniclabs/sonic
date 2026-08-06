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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRingDialEdges_ConnectsPairsFromExactlyOneEnd(t *testing.T) {
	// Dialing a pair from both ends can close both connections, see
	// ringDialEdges. Two nodes are the case at risk: there the ring degenerates
	// into a single pair.
	for _, numNodes := range []int{2, 3, 4, 7} {
		t.Run(fmt.Sprintf("nodes=%d", numNodes), func(t *testing.T) {
			require := require.New(t)

			seen := map[[2]int]bool{}
			for _, edge := range ringDialEdges(numNodes) {
				dialer, target := edge[0], edge[1]
				require.NotEqual(dialer, target, "node must not dial itself")
				require.Less(dialer, numNodes)
				require.Less(target, numNodes)

				pair := [2]int{min(dialer, target), max(dialer, target)}
				require.False(seen[pair], "pair %v is dialed from both ends", pair)
				seen[pair] = true
			}
		})
	}
}

func TestRingDialEdges_FormsConnectedRing(t *testing.T) {
	for _, numNodes := range []int{2, 3, 4, 7} {
		t.Run(fmt.Sprintf("nodes=%d", numNodes), func(t *testing.T) {
			require := require.New(t)

			edges := ringDialEdges(numNodes)
			// A ring needs one edge per node, except for two nodes, whose two
			// ring edges collapse into a single connection.
			want := numNodes
			if numNodes == 2 {
				want = 1
			}
			require.Len(edges, want)

			// The connections must span all nodes.
			neighbors := map[int][]int{}
			for _, edge := range edges {
				neighbors[edge[0]] = append(neighbors[edge[0]], edge[1])
				neighbors[edge[1]] = append(neighbors[edge[1]], edge[0])
			}
			visited := map[int]bool{0: true}
			queue := []int{0}
			for len(queue) > 0 {
				cur := queue[0]
				queue = queue[1:]
				for _, next := range neighbors[cur] {
					if !visited[next] {
						visited[next] = true
						queue = append(queue, next)
					}
				}
			}
			require.Len(visited, numNodes, "the network is not fully connected")
		})
	}
}

func TestRingDialEdges_SingleNodeNeedsNoConnection(t *testing.T) {
	require.Empty(t, ringDialEdges(1))
	require.Empty(t, ringDialEdges(0))
}
