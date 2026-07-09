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

package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/0xsoniclabs/sonic/p2p"
	"github.com/0xsoniclabs/sonic/p2p/networks"
	"github.com/0xsoniclabs/sonic/p2p/protocols"
)

// demoServiceTag scopes mDNS discovery to this demo, so its nodes only find each
// other and not unrelated libp2p services on the network.
const demoServiceTag = "sonic-p2p-demo"

// ephemeralListenAddresses binds OS-assigned ports so many demo nodes can run on
// one host at once. mDNS makes the resulting addresses discoverable, so no fixed
// ports are needed.
var ephemeralListenAddresses = []string{
	"/ip4/0.0.0.0/udp/0/quic-v1",
	"/ip4/0.0.0.0/tcp/0",
}

// runNode starts a validator or archive node, connects it to the network via
// mDNS (and any explicit --peer addresses), periodically reports peer health, and
// runs until ctx is cancelled. The id is used only for validators.
func runNode(ctx context.Context, role networks.Role, id int, peers []string, reportEvery time.Duration) error {
	out := log.Root()

	node, err := newNode(out, peers)
	if err != nil {
		return err
	}

	status := newDemoStatus(role, roleSeed(role, id, node.ID()))
	node.RegisterStreamProtocol(protocols.NewPingProtocol(status))
	node.RegisterStreamProtocol(protocols.NewScanProtocol(status, nodePeerSource{node: node}))

	var validatorNetwork *networks.ValidatorNetwork
	if role == networks.RoleValidator {
		validatorNetwork = networks.NewValidatorNetwork(
			node,
			staticMembership{},
			networks.NewSecp256k1Signer(validatorKey(id)),
			networks.NewSecp256k1Verifier(),
			uint32(id),
			networks.ValidatorNetworkConfig{},
		)
	}

	if err := node.Start(); err != nil {
		return fmt.Errorf("failed to start node: %w", err)
	}
	defer func() { _ = node.Stop() }()

	if validatorNetwork != nil {
		validatorNetwork.Start(ctx)
		defer validatorNetwork.Stop()
	}

	discovery, err := startDiscovery(ctx, node, out)
	if err != nil {
		return err
	}
	defer func() { _ = discovery.Close() }()

	monitor := protocols.NewHealthMonitor(node, protocols.HealthMonitorConfig{Interval: reportEvery}, prometheus.NewRegistry())
	monitor.Start(ctx)
	defer monitor.Stop()

	printBanner(out, role, id, node)
	go reportLoop(ctx, out, monitor, reportEvery)

	<-ctx.Done()
	out.Info("shutting down")
	return nil
}

// newNode creates (but does not start) a demo p2p node listening on ephemeral
// ports and bootstrapping to any explicit peer addresses.
func newNode(out log.Logger, peers []string) (*p2p.Node, error) {
	config := p2p.DefaultConfig()
	config.ListenAddresses = ephemeralListenAddresses
	config.BootstrapPeers = peers
	node, err := p2p.New(config, out, prometheus.NewRegistry())
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}
	return node, nil
}

// startDiscovery runs mDNS on the node, connecting to every demo peer it finds.
func startDiscovery(ctx context.Context, node *p2p.Node, out log.Logger) (io.Closer, error) {
	service := mdns.NewMdnsService(node.Host(), demoServiceTag, &mdnsNotifee{ctx: ctx, node: node, out: out})
	if err := service.Start(); err != nil {
		return nil, fmt.Errorf("failed to start mDNS discovery: %w", err)
	}
	return service, nil
}

// mdnsNotifee connects the node to peers discovered on the local network.
type mdnsNotifee struct {
	ctx  context.Context
	node *p2p.Node
	out  log.Logger
}

// HandlePeerFound implements mdns.Notifee.
func (n *mdnsNotifee) HandlePeerFound(info peer.AddrInfo) {
	if info.ID == n.node.ID() {
		return
	}
	n.out.Debug("discovered peer via mDNS", "peer", info.ID)
	if err := n.node.Connect(n.ctx, info); err != nil {
		n.out.Debug("failed to connect to discovered peer", "peer", info.ID, "err", err)
	}
}

// roleSeed returns a stable seed identifying this node, used to derive its faked
// block-height drift. Validators use their ID (stable across runs); archives use
// their ephemeral peer ID.
func roleSeed(role networks.Role, id int, self p2p.PeerID) string {
	if role == networks.RoleValidator {
		return fmt.Sprintf("validator-%d", id)
	}
	return self.String()
}

// printBanner prints a short, human-readable summary of the node so a first-time
// user understands what is running and how it will find peers.
func printBanner(out log.Logger, role networks.Role, id int, node *p2p.Node) {
	out.Info("Sonic p2p demo node started",
		"role", roleName(role),
		"validator_id", validatorIDField(role, id),
		"peer_id", node.ID(),
		"validators", numValidators,
	)
	for _, address := range node.Addresses() {
		out.Info("listening", "address", fmt.Sprintf("%s/p2p/%s", address, node.ID()))
	}
	out.Info("discovering peers automatically via mDNS; start more nodes to form the network")
}

func validatorIDField(role networks.Role, id int) string {
	if role == networks.RoleValidator {
		return fmt.Sprintf("%d", id)
	}
	return "-"
}
