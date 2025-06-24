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

package topology

import (
	"slices"
	"testing"
	"time"
)

func TestConnectionAdvisor_SuggestToConnectToPeerOfPeer(t *testing.T) {
	// Connection pattern: 1 <-> 2 <-> 3
	advisor := newAdvisor(1)
	advisor.UpdatePeers(2, []int{1, 3})
	if want, got := 3, advisor.GetNewPeerSuggestion(); want != got {
		t.Fatalf("unexpected suggestion: want=%d got=%d", want, got)
	}
}

func TestConnectionAdvisor_SuggestsNothingIfThereIsNoPeerOfAPeer(t *testing.T) {
	// Connection pattern: 1
	advisor := newAdvisor(1)
	if want, got := 0, advisor.GetNewPeerSuggestion(); want != got {
		t.Fatalf("unexpected suggestion: want=%d got=%d", want, got)
	}

	// Connection pattern: 1 <-> 2
	advisor.UpdatePeers(2, []int{1})
	if want, got := 0, advisor.GetNewPeerSuggestion(); want != got {
		t.Fatalf("unexpected suggestion: want=%d got=%d", want, got)
	}
}

func TestConnectionAdvisor_SuggestToRemoveRedundantConnection(t *testing.T) {
	// Connection pattern: (1,2), (1,3), (1,4), (2,3), (3,4)
	// The connection (1,3) is redundant.
	advisor := newAdvisor(1)
	advisor.UpdatePeers(2, []int{1, 3})
	advisor.UpdatePeers(3, []int{1})
	advisor.UpdatePeers(4, []int{1, 3})
	if want, got := 3, advisor.GetRedundantPeerSuggestion(); got == nil || want != *got {
		t.Fatalf("unexpected suggestion: want=%d got=%v", want, got)
	}
}

func TestConnectionAdvisor_DoesNotSuggestToRemovePeerIfNoneIsRedundant(t *testing.T) {
	// Connection pattern: 1
	advisor := newAdvisor(1)
	if got := advisor.GetRedundantPeerSuggestion(); got != nil {
		t.Fatalf("unexpected suggestion: got=%v", got)
	}

	// Connection pattern: 1 <-> 2
	advisor.UpdatePeers(2, []int{1})
	if got := advisor.GetRedundantPeerSuggestion(); got != nil {
		t.Fatalf("unexpected suggestion: got=%v", got)
	}

	// Connection pattern: 1 <-> 2, 1 <-> 3
	advisor.UpdatePeers(3, []int{1})
	if got := advisor.GetRedundantPeerSuggestion(); got != nil {
		t.Fatalf("unexpected suggestion: got=%v", got)
	}
}

func TestConnectionAdvisor_NewPeerSuggestionPrunesOutdatedPeerInformation(t *testing.T) {
	// Connection pattern: 1 <-> 2 <-> 3
	advisor := newAdvisor(1)
	advisor.UpdatePeers(2, []int{1, 3})
	if want, got := 3, advisor.GetNewPeerSuggestion(); want != got {
		t.Fatalf("unexpected suggestion: want=%d got=%d", want, got)
	}

	if len(advisor.neighborhood) == 0 {
		t.Fatalf("unexpected neighborhood: got=%v", advisor.neighborhood)
	}

	// Set the maximum age of peer information to 0.
	advisor.maxPeerInfoAge = 0

	// All knowledge about the network should have been forgotten.
	if want, got := 0, advisor.GetNewPeerSuggestion(); want != got {
		t.Fatalf("unexpected suggestion: want=%d got=%d", want, got)
	}

	if len(advisor.neighborhood) != 0 {
		t.Fatalf("unexpected neighborhood: got=%v", advisor.neighborhood)
	}
}

func TestConnectionAdvisor_RedundantPeerSuggestionPrunesOutdatedPeerInformation(t *testing.T) {
	advisor := newAdvisor(1)
	advisor.UpdatePeers(2, []int{1, 3})
	advisor.UpdatePeers(3, []int{1})
	advisor.UpdatePeers(4, []int{1, 3})
	if got := advisor.GetRedundantPeerSuggestion(); got == nil {
		t.Fatalf("unexpected suggestion: got=%v", got)
	}

	if len(advisor.neighborhood) == 0 {
		t.Fatalf("unexpected neighborhood: got=%v", advisor.neighborhood)
	}

	// Set the maximum age of peer information to 0.
	advisor.maxPeerInfoAge = 0

	// All knowledge about the network should have been forgotten.
	if got := advisor.GetRedundantPeerSuggestion(); got != nil {
		t.Fatalf("unexpected suggestion: got=%v", got)
	}

	if len(advisor.neighborhood) != 0 {
		t.Fatalf("unexpected neighborhood: got=%v", advisor.neighborhood)
	}
}

func TestConnectionAdvisor_LinearTopologyTurnsIntoMesh(t *testing.T) {
	const N = 10

	// Create a linear topology
	net := newNetwork(N)
	for i := 0; i < N-1; i++ {
		net.connect(i, i+1)
	}

	if want, got := N-1, net.getDiameter(); want != got {
		t.Fatalf("unexpected diameter: want=%d got=%d", want, got)
	}

	// Add connections in turns as suggested by the connection advisor.
	last := net.getDiameter()
	for range N {
		newPeers := make([]int, N)
		for i, node := range net.nodes {
			advisor := newAdvisor(node.id)
			for _, peer := range node.peers {
				advisor.UpdatePeers(peer, net.nodes[peer].peers)
			}
			newPeers[i] = advisor.GetNewPeerSuggestion()
		}

		for i, newPeer := range newPeers {
			if newPeer == 0 {
				continue
			}
			net.connect(i, newPeer)
		}

		diameter := net.getDiameter()
		if diameter > last {
			t.Errorf("diameter increased: had=%d new=%d", last, diameter)
		}
		last = diameter
	}

	// The linear topology should have turned into a mesh.
	if want, got := 1, net.getDiameter(); want != got {
		t.Errorf("unexpected diameter: want=%d got=%d", want, got)
	}
}

func TestConnectionAdvisor_GradualRefinementCanReachOptimum(t *testing.T) {
	const N = 16 // Number of nodes
	const C = 4  // Maximum number of connections per node; enough to reach 16 nodes in 2 hops.

	// Create a linear topology
	net := newNetwork(N)
	for i := 0; i < N-1; i++ {
		net.connect(i, i+1)
	}

	if want, got := N-1, net.getDiameter(); want != got {
		t.Fatalf("unexpected diameter: want=%d got=%d", want, got)
	}

	// Updating connections should gradually turn the linear topology into a mesh.
	for range 10 * N {
		// Remove suggested redundant peers.
		for i, node := range net.nodes {
			if len(node.peers) <= C {
				continue
			}
			advisor := newAdvisor(node.id)
			for _, peer := range node.peers {
				advisor.UpdatePeers(peer, net.nodes[peer].peers)
			}
			if toRemove := advisor.GetRedundantPeerSuggestion(); toRemove != nil {
				net.disconnect(i, *toRemove)
			}
		}

		// Add new connections.
		newPeers := make([]int, N)
		for i, node := range net.nodes {
			advisor := newAdvisor(node.id)
			for _, peer := range node.peers {
				advisor.UpdatePeers(peer, net.nodes[peer].peers)
			}
			newPeers[i] = advisor.GetNewPeerSuggestion()
		}

		for i, newPeer := range newPeers {
			if newPeer == 0 {
				continue
			}
			net.connect(i, newPeer)
		}

		diameter := net.getDiameter()
		if diameter == 2 {
			break
		}
	}

	// Connections should be effectively used to reach all nodes in 2 hops.
	if want, got := 2, net.getDiameter(); want != got {
		t.Errorf("unexpected diameter: want=%d got=%d", want, got)
	}
}

type node struct {
	id    int
	peers []int
}

type network struct {
	nodes []node
}

func newNetwork(n int) *network {
	nodes := make([]node, n)
	for i := range nodes {
		nodes[i].id = i
	}
	return &network{nodes: nodes}
}

func (n *network) connect(i, j int) {
	if !slices.Contains(n.nodes[i].peers, j) {
		n.nodes[i].peers = append(n.nodes[i].peers, j)
		n.nodes[j].peers = append(n.nodes[j].peers, i)
	}
}

func (n *network) disconnect(i, j int) {
	n.nodes[i].peers = slices.DeleteFunc(n.nodes[i].peers, func(peer int) bool { return peer == j })
	n.nodes[j].peers = slices.DeleteFunc(n.nodes[j].peers, func(peer int) bool { return peer == i })
}

func (n *network) getDiameter() int {
	// Floyd-Warshall algorithm
	const INF = 1e9
	dist := make([][]int, len(n.nodes))
	for i := range dist {
		dist[i] = make([]int, len(n.nodes))
		for j := range dist[i] {
			dist[i][j] = INF
		}
		dist[i][i] = 0
	}

	for i, node := range n.nodes {
		for _, peer := range node.peers {
			dist[i][peer] = 1
			dist[peer][i] = 1
		}
	}

	for k := range n.nodes {
		for i := range n.nodes {
			for j := range n.nodes {
				if dist[i][j] > dist[i][k]+dist[k][j] {
					dist[i][j] = dist[i][k] + dist[k][j]
				}
			}
		}
	}

	diameter := 0
	for i := range n.nodes {
		for j := range n.nodes {
			if dist[i][j] > diameter {
				diameter = dist[i][j]
			}
		}
	}

	return diameter
}

func newAdvisor(localId int) *connectionAdvisor[int, int] {
	return newConnectionAdvisor(localId, time.Second, func(i int) int { return i })
}
