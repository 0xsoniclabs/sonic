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

package protocols

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/0xsoniclabs/sonic/p2p"
	"github.com/0xsoniclabs/sonic/p2p/pb"
)

const drainProtocolID = protocol.ID("/sonic/test/drain/1")

// TestRateAbuse_SustainedFlood_DisconnectsPeer stands up a server with a tiny
// per-peer message budget and a drain protocol that keeps reading, then has a
// client flood rate-violating messages. It asserts the server disconnects the
// abusive client.
//
// Until sustained-abuse handling is implemented, rate violations only cause the
// individual message to be dropped and never disconnect the peer, so this test
// fails - demonstrating the gap.
func TestRateAbuse_SustainedFlood_DisconnectsPeer(t *testing.T) {
	server := newLimitedNode(t, 1, 1) // 1 msg/sec, burst 1: a flood is nearly all violations
	server.RegisterStreamProtocol(drainProtocol{})
	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	t.Cleanup(func() { _ = server.Stop() })

	client := newLimitedNode(t, 1000, 1000)
	if err := client.Start(); err != nil {
		t.Fatalf("failed to start client: %v", err)
	}
	t.Cleanup(func() { _ = client.Stop() })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := client.Connect(ctx, addrInfoOf(server)); err != nil {
		t.Fatalf("client failed to connect: %v", err)
	}
	stream, err := client.OpenStream(ctx, server.ID(), drainProtocolID)
	if err != nil {
		t.Fatalf("client failed to open stream: %v", err)
	}

	// Flood the server until the connection is torn down.
	go func() {
		for ctx.Err() == nil {
			if err := stream.WriteMessage(&pb.ScanStatusRequest{}, 1024); err != nil {
				return
			}
		}
	}()

	if !waitForConnectedness(server, client.ID(), network.NotConnected, 15*time.Second) {
		t.Fatal("expected server to disconnect the flooding client, but it stayed connected")
	}
}

// drainProtocol reads messages from a stream in a loop, continuing past
// rate-limit rejections so sustained abuse can accumulate.
type drainProtocol struct{}

func (drainProtocol) ProtocolID() protocol.ID { return drainProtocolID }

func (drainProtocol) Handle(stream p2p.Stream) {
	defer func() { _ = stream.Close() }()
	for {
		var message pb.ScanStatusRequest
		if err := stream.ReadMessage(&message, 1024); err != nil {
			if errors.Is(err, p2p.ErrRateLimited) {
				continue
			}
			return
		}
	}
}

func newLimitedNode(t *testing.T, messagesPerSecond float64, messageBurst int) *p2p.Node {
	t.Helper()
	config := p2p.DefaultConfig()
	config.ListenAddresses = []string{
		"/ip4/127.0.0.1/udp/0/quic-v1",
		"/ip4/127.0.0.1/tcp/0",
	}
	config.RateLimit.MessagesPerSecond = messagesPerSecond
	config.RateLimit.MessageBurst = messageBurst
	node, err := p2p.New(config, log.Root(), prometheus.NewRegistry())
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}
	return node
}

func waitForConnectedness(node *p2p.Node, target peer.ID, want network.Connectedness, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if node.Host().Network().Connectedness(target) == want {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}
