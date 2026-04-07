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
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discover/discfilter"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/stretchr/testify/require"
)

func TestConnectP2PNetwork_TrustedPeersBypassBan(t *testing.T) {
	// This test verifies that peers marked as trusted via admin_addTrustedPeer
	// can reconnect even when their node IDs have been banned by discfilter.
	//
	// This test would consistently fail if connectP2PNetwork did not call
	// admin_addTrustedPeer, because a restart after banning would be unable
	// to re-establish P2P connections.
	require := require.New(t)

	net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		NumNodes: 2,
	})

	// Get the enode IDs of all nodes in the running network.
	nodeIDs := getNodeIDs(t, net)

	// Ban all node IDs in the process-global discfilter.
	for _, id := range nodeIDs {
		discfilter.Ban(id)
	}

	// Verify the bans are in effect.
	for _, id := range nodeIDs {
		require.True(discfilter.BannedDynamic(id), "node ID should be banned")
	}

	// Restart the network. On restart, connectP2PNetwork is called with
	// fresh enode URLs (same node IDs, new ports). Despite the bans,
	// this should succeed because connectP2PNetwork adds peers as
	// trusted, which bypasses the discfilter ban check in the P2P
	// server's postHandshakeChecks.
	errCh := make(chan error, 1)
	go func() {
		errCh <- net.Restart()
	}()

	select {
	case err := <-errCh:
		require.NoError(err, "network restart should succeed")
	case <-time.After(30 * time.Second):
		t.Fatal("network restart timed out, likely due to discfilter bans")
	}

	// Verify both nodes are connected after the restart.
	for i := range net.NumNodes() {
		client, err := net.GetClientConnectedToNode(i)
		require.NoError(err)
		defer client.Close()

		var peers []map[string]any
		require.NoError(client.Client().Call(&peers, "admin_peers"))
		require.NotEmpty(peers, "node %d should have peers after restart", i)
	}
}

// getNodeIDs returns the enode.ID for each node in the network.
func getNodeIDs(t *testing.T, net *IntegrationTestNet) []enode.ID {
	t.Helper()
	ids := make([]enode.ID, net.NumNodes())
	for i := range net.NumNodes() {
		client, err := net.GetClientConnectedToNode(i)
		require.NoError(t, err)
		defer client.Close()

		var info struct {
			Enode string `json:"enode"`
		}
		require.NoError(t, client.Client().Call(&info, "admin_nodeInfo"))

		node, err := enode.ParseV4(info.Enode)
		require.NoError(t, err)
		ids[i] = node.ID()
	}
	return ids
}
