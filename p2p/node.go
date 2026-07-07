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

// Package p2p implements Sonic's libp2p-based peer-to-peer communication layer.
//
// The package provides the transport, identity, framing, rate limiting, and
// protocol-registration machinery on top of which the higher-level networks
// (the validator mesh, the gossip dissemination network, and archive
// discovery) and future consensus protocols are built. It follows an
// open/closed design: new protocols are added by registering a StreamProtocol
// or GossipTopic; the core is never modified. See ARCHITECTURE.md for the
// design and HANDOFF.md for the node-integration follow-up.
package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/p2p/guard"
	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/prometheus/client_golang/prometheus"
)

// Node is the P2P layer facade: a libp2p host plus a gossipsub router and the
// adversarial-network guards, exposing an open/closed registry for protocols.
// Its Start/Stop methods have the shape required by go-ethereum's node
// lifecycle so a future service wrapper can register it on the node stack.
type Node struct {
	logger  logger.Logger
	config  Config
	host    host.Host
	pubsub  *pubsub.PubSub
	gater   *guard.Gater
	limiter *guard.RateLimiter
	metrics *Metrics

	streamProtocols []StreamProtocol
	gossipTopics    []GossipTopic
	topics          map[string]*pubsub.Topic

	// now is the time source used for ban cooldowns, injectable for tests.
	now func() time.Time

	mutex   sync.Mutex
	started bool
	quit    chan struct{}
	wg      sync.WaitGroup
}

// New creates a Node from the given configuration. It generates or loads the
// host key, assembles the QUIC-primary (TCP+Noise fallback) transport, installs
// the resource manager, connection gater, and connection manager, and starts a
// gossipsub router with peer scoring. It does not begin serving protocols until
// Start is called. A nil registerer uses the default Prometheus registerer.
func New(config Config, log logger.Logger, registerer prometheus.Registerer) (*Node, error) {
	key, err := loadOrCreateHostKey(config.HostKeyPath)
	if err != nil {
		return nil, err
	}

	gater := guard.NewGater()
	resources, err := guard.NewResourceManager(config.Resources)
	if err != nil {
		return nil, fmt.Errorf("p2p: failed to create resource manager: %w", err)
	}
	connectionManager, err := connmgr.NewConnManager(
		config.ConnectionManager.LowWater,
		config.ConnectionManager.HighWater,
	)
	if err != nil {
		return nil, fmt.Errorf("p2p: failed to create connection manager: %w", err)
	}

	h, err := libp2p.New(
		libp2p.Identity(key),
		libp2p.ListenAddrStrings(config.ListenAddresses...),
		libp2p.Transport(libp2pquic.NewTransport),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Security(noise.ID, noise.New),
		libp2p.ConnectionGater(gater),
		libp2p.ResourceManager(resources),
		libp2p.ConnectionManager(connectionManager),
	)
	if err != nil {
		return nil, fmt.Errorf("p2p: failed to create libp2p host: %w", err)
	}

	scoreParams, scoreThresholds := guard.GossipScoreParams()
	router, err := pubsub.NewGossipSub(context.Background(), h,
		pubsub.WithPeerScore(scoreParams, scoreThresholds),
	)
	if err != nil {
		_ = h.Close()
		return nil, fmt.Errorf("p2p: failed to create gossipsub router: %w", err)
	}

	return &Node{
		logger:  log,
		config:  config,
		host:    h,
		pubsub:  router,
		gater:   gater,
		limiter: guard.NewRateLimiter(config.RateLimit),
		metrics: NewMetrics(registerer),
		topics:  make(map[string]*pubsub.Topic),
		now:     time.Now,
		quit:    make(chan struct{}),
	}, nil
}

// RegisterStreamProtocol registers a stream protocol. It must be called before
// Start and panics otherwise, since handlers are installed at Start.
func (n *Node) RegisterStreamProtocol(p StreamProtocol) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if n.started {
		panic("p2p: RegisterStreamProtocol called after Start")
	}
	n.streamProtocols = append(n.streamProtocols, p)
}

// RegisterGossipTopic registers a gossip topic. It must be called before Start.
func (n *Node) RegisterGossipTopic(t GossipTopic) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if n.started {
		panic("p2p: RegisterGossipTopic called after Start")
	}
	n.gossipTopics = append(n.gossipTopics, t)
}

// Start installs the registered protocol handlers, joins and subscribes to the
// registered gossip topics, wires connection-lifecycle logging, and dials the
// configured bootstrap peers. It satisfies the node-lifecycle Start signature.
func (n *Node) Start() error {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if n.started {
		return nil
	}

	n.host.Network().Notify(n.connectionNotifiee())

	for _, streamProtocol := range n.streamProtocols {
		n.installStreamHandler(streamProtocol)
	}
	for _, topic := range n.gossipTopics {
		if err := n.joinTopic(topic); err != nil {
			return err
		}
	}

	n.started = true
	n.wg.Add(1)
	go n.dialBootstrapPeers()

	n.logger.Info("p2p node started", "id", n.host.ID(), "addrs", n.host.Addrs())
	return nil
}

// Stop tears the node down: it stops accepting new work, waits for background
// goroutines, and closes the host. It satisfies the node-lifecycle Stop
// signature.
func (n *Node) Stop() error {
	n.mutex.Lock()
	if !n.started {
		n.mutex.Unlock()
		return nil
	}
	n.started = false
	close(n.quit)
	n.mutex.Unlock()

	n.wg.Wait()
	n.logger.Info("p2p node stopping", "id", n.host.ID())
	return n.host.Close()
}

// ID returns this node's peer identity.
func (n *Node) ID() PeerID { return n.host.ID() }

// Host exposes the underlying libp2p host for advanced integrations.
func (n *Node) Host() host.Host { return n.host }

// Addresses returns the multiaddrs the node is currently listening on.
func (n *Node) Addresses() []ma.Multiaddr { return n.host.Addrs() }

// Logger returns the injected logger, so components built on the node share it.
func (n *Node) Logger() logger.Logger { return n.logger }

// Gater exposes the connection gater so higher layers can ban misbehaving peers.
func (n *Node) Gater() *guard.Gater { return n.gater }

// Connect dials a peer, logging the attempt. It is used by the validator mesh
// to establish direct connections.
func (n *Node) Connect(ctx context.Context, info peer.AddrInfo) error {
	n.logger.Debug("dialing peer", "peer", info.ID)
	if err := n.host.Connect(ctx, info); err != nil {
		n.logger.Debug("dial failed", "peer", info.ID, "err", err)
		return err
	}
	return nil
}

// OpenStream opens an outbound, framed, rate-limited stream to a peer on the
// given protocol.
func (n *Node) OpenStream(ctx context.Context, target PeerID, id protocol.ID) (Stream, error) {
	raw, err := n.host.NewStream(ctx, target, id)
	if err != nil {
		return nil, err
	}
	return newStream(raw, n.limiter, n.metrics, n.penalizePeer), nil
}

// ClosePeer closes all connections to a peer, logging the reason.
func (n *Node) ClosePeer(target PeerID, reason string) error {
	n.logger.Debug("closing connection", "peer", target, "reason", reason)
	return n.host.Network().ClosePeer(target)
}

// Publish publishes a message to a registered gossip topic.
func (n *Node) Publish(ctx context.Context, topic string, message []byte) error {
	n.mutex.Lock()
	handle, ok := n.topics[topic]
	n.mutex.Unlock()
	if !ok {
		return fmt.Errorf("p2p: topic %q is not registered", topic)
	}
	return handle.Publish(ctx, message)
}

func (n *Node) installStreamHandler(streamProtocol StreamProtocol) {
	n.host.SetStreamHandler(streamProtocol.ProtocolID(), func(raw network.Stream) {
		streamProtocol.Handle(newStream(raw, n.limiter, n.metrics, n.penalizePeer))
	})
}

// penalizePeer disconnects and temporarily bans a peer that has committed
// sustained rate-limit abuse. The ban keeps the connection gater rejecting the
// peer's reconnection attempts until the cooldown elapses. scope is the
// protocol ID or topic the abuse occurred on, for logging.
func (n *Node) penalizePeer(peer PeerID, scope string) {
	n.metrics.peerDisconnects.WithLabelValues("rate-abuse").Inc()
	n.logger.Info("disconnecting abusive peer",
		"peer", peer, "scope", scope, "reason", "rate-limit-abuse")
	if duration := n.config.RateLimit.BanDuration; duration > 0 {
		n.gater.BanUntil(peer, n.now().Add(duration))
	}
	_ = n.host.Network().ClosePeer(peer)
	n.limiter.Forget(peer.String())
}

func (n *Node) joinTopic(topic GossipTopic) error {
	name := topic.Topic()
	if err := n.pubsub.RegisterTopicValidator(name, n.topicValidator(topic)); err != nil {
		return fmt.Errorf("p2p: failed to register validator for %q: %w", name, err)
	}
	handle, err := n.pubsub.Join(name)
	if err != nil {
		return fmt.Errorf("p2p: failed to join topic %q: %w", name, err)
	}
	subscription, err := handle.Subscribe()
	if err != nil {
		return fmt.Errorf("p2p: failed to subscribe to topic %q: %w", name, err)
	}
	n.topics[name] = handle

	n.wg.Add(1)
	go n.consumeTopic(topic, subscription)
	return nil
}

func (n *Node) topicValidator(topic GossipTopic) pubsub.ValidatorEx {
	name := topic.Topic()
	return func(_ context.Context, from peer.ID, message *pubsub.Message) pubsub.ValidationResult {
		if decision := n.limiter.Check(from.String(), len(message.Data)); !decision.Allowed {
			n.metrics.rateDropped.WithLabelValues(name, "traffic").Inc()
			n.metrics.gossip.WithLabelValues(name, "rate_limited").Inc()
			if decision.Abusive {
				n.penalizePeer(from, name)
				return pubsub.ValidationReject
			}
			return pubsub.ValidationIgnore
		}
		switch topic.Validate(from, message.Data) {
		case ValidationAccept:
			n.metrics.gossip.WithLabelValues(name, "accept").Inc()
			return pubsub.ValidationAccept
		case ValidationReject:
			n.metrics.gossip.WithLabelValues(name, "reject").Inc()
			return pubsub.ValidationReject
		default:
			n.metrics.gossip.WithLabelValues(name, "ignore").Inc()
			return pubsub.ValidationIgnore
		}
	}
}

func (n *Node) consumeTopic(topic GossipTopic, subscription *pubsub.Subscription) {
	defer n.wg.Done()
	defer subscription.Cancel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-n.quit:
			cancel()
		case <-ctx.Done():
		}
	}()

	self := n.host.ID()
	for {
		message, err := subscription.Next(ctx)
		if err != nil {
			return // context cancelled on shutdown
		}
		if message.ReceivedFrom == self {
			continue
		}
		topic.Deliver(message.ReceivedFrom, message.Data)
	}
}

func (n *Node) connectionNotifiee() network.Notifiee {
	return &network.NotifyBundle{
		ConnectedF: func(_ network.Network, conn network.Conn) {
			direction := conn.Stat().Direction.String()
			n.metrics.connections.WithLabelValues(direction).Inc()
			n.logger.Debug("connection opened", "peer", conn.RemotePeer(), "direction", direction)
		},
		DisconnectedF: func(_ network.Network, conn network.Conn) {
			direction := conn.Stat().Direction.String()
			n.metrics.connections.WithLabelValues(direction).Dec()
			n.limiter.Forget(conn.RemotePeer().String())
			n.logger.Debug("connection closed", "peer", conn.RemotePeer(), "direction", direction)
		},
	}
}

func (n *Node) dialBootstrapPeers() {
	defer n.wg.Done()
	if len(n.config.BootstrapPeers) == 0 {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-n.quit
		cancel()
	}()

	for _, address := range n.config.BootstrapPeers {
		info, err := peer.AddrInfoFromString(address)
		if err != nil {
			n.logger.Warn("invalid bootstrap peer", "addr", address, "err", err)
			continue
		}
		if err := n.Connect(ctx, *info); err != nil {
			n.logger.Warn("failed to dial bootstrap peer", "peer", info.ID, "err", err)
		}
	}
}
