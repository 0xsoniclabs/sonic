// Copyright 2024 The Sonic Authors
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

package gossip

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/0xsoniclabs/sonic/eventcheck"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/protocols/dag/dagstream"
	"github.com/0xsoniclabs/sonic/gossip/protocols/dag/dagstream/dagstreamleecher"
	"github.com/0xsoniclabs/sonic/gossip/protocols/dag/dagstream/dagstreamseeder"
	"github.com/0xsoniclabs/sonic/gossip/topology"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/utils/caution"

	"github.com/Fantom-foundation/lachesis-base/gossip/dagprocessor"
	"github.com/Fantom-foundation/lachesis-base/gossip/itemsfetcher"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/utils/datasemaphore"
	"github.com/ethereum/go-ethereum/common"
	notify "github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover/discfilter"
)

const (
	softResponseLimitSize = 2 * 1024 * 1024    // Target maximum size of returned events, or other data.
	softLimitItems        = 250                // Target maximum number of events or transactions to request/response
	hardLimitItems        = softLimitItems * 4 // Maximum number of events or transactions to request/response

	// txChanSize is the size of channel listening to NewTxsNotify.
	// The number is referenced from the size of tx pool.
	txChanSize = 4096
)

var (
	// broadcastedTxsCounter tracks the number of transactions broadcasted over p2p.
	broadcastedTxsCounter = metrics.GetOrRegisterCounter("p2p_txs_broadcasted", nil)

	// incompleteEventsSpilled counts events that could not be added to the DAG and were dropped from the incomplete events buffer.
	// Note: Despite the name, these events are not moved to another storage; they are simply discarded.
	incompleteEventsSpilled = metrics.GetOrRegisterCounter("p2p_incomplete_events_spilled", nil)
)

// errResp builds a formatted error response from the given error code and message.
func errResp(code errCode, format string, v ...interface{}) error {
	return fmt.Errorf("%v - %v", code, fmt.Sprintf(format, v...))
}

// checkLenLimits validates that the message item count is within acceptable bounds.
func checkLenLimits(size int, v interface{}) error {
	if size <= 0 {
		return errResp(ErrEmptyMessage, "%v", v)
	}
	if size > hardLimitItems {
		return errResp(ErrMsgTooLarge, "%v", v)
	}
	return nil
}

// dagNotifier provides subscriptions for new epoch and new emitted event notifications.
type dagNotifier interface {
	SubscribeNewEpoch(ch chan<- idx.Epoch) notify.Subscription
	SubscribeNewEmitted(ch chan<- *inter.EventPayload) notify.Subscription
}

// processCallback holds callback functions for processing events and switching epochs.
type processCallback struct {
	Event         func(*inter.EventPayload) error
	SwitchEpochTo func(idx.Epoch) error
}

// handlerConfig is the collection of initialization parameters to create a full
// node network handler.
type handlerConfig struct {
	config              Config
	notifier            dagNotifier
	txpool              TxPool
	engineMu            sync.Locker
	checkers            *eventcheck.Checkers
	s                   *Store
	process             processCallback
	localId             enode.ID
	localEndPointSource LocalEndPointSource
}

// LocalEndPointSource provides access to the local node's public endpoint.
type LocalEndPointSource interface {
	GetLocalEndPoint() *enode.Node
}

// handler is the main P2P protocol manager for the Sonic network.
// It manages peer connections, event/transaction propagation, DAG synchronization,
// and all sub-components including fetchers, processors, leechers, and seeders.
type handler struct {
	NetworkID uint64
	config    Config

	syncStatus syncStatus

	txpool   TxPool
	maxPeers int

	peers *peerSet

	txsCh  chan evmcore.NewTxsNotify
	txsSub notify.Subscription

	dagLeecher   *dagstreamleecher.Leecher
	dagSeeder    *dagstreamseeder.Seeder
	dagProcessor *dagprocessor.Processor
	dagFetcher   *itemsfetcher.Fetcher

	process processCallback

	txFetcher *itemsfetcher.Fetcher

	checkers *eventcheck.Checkers

	msgSemaphore *datasemaphore.DataSemaphore

	store    *Store
	engineMu sync.Locker

	notifier             dagNotifier
	emittedEventsCh      chan *inter.EventPayload
	emittedEventsSub     notify.Subscription
	newEpochsCh          chan idx.Epoch
	newEpochsSub         notify.Subscription
	quitProgressBradcast chan struct{}

	// channels for syncer, txsyncLoop
	txsyncCh chan *txsync
	quitSync chan struct{}

	// wait group is used for graceful shutdowns during downloading
	// and processing
	loopsWg sync.WaitGroup
	wg      sync.WaitGroup
	peerWG  sync.WaitGroup
	started sync.WaitGroup

	// channels for peer info collection loop
	peerInfoStop chan<- struct{}

	// suggests new peers to connect to by monitoring the neighborhood
	connectionAdvisor topology.ConnectionAdvisor
	nextSuggestedPeer chan *enode.Node

	localEndPointSource LocalEndPointSource

	logger.Instance
}

// newHandler returns a new Sonic sub protocol manager. The Sonic sub protocol manages peers capable
// with the Sonic network.
func newHandler(
	c handlerConfig,
) (
	*handler,
	error,
) {
	// Create the protocol manager with the base fields
	h := &handler{
		NetworkID:            c.s.GetRules().NetworkID,
		config:               c.config,
		notifier:             c.notifier,
		txpool:               c.txpool,
		msgSemaphore:         datasemaphore.New(c.config.Protocol.MsgsSemaphoreLimit, getSemaphoreWarningFn("P2P messages")),
		store:                c.s,
		process:              c.process,
		checkers:             c.checkers,
		peers:                newPeerSet(),
		engineMu:             c.engineMu,
		txsyncCh:             make(chan *txsync),
		quitSync:             make(chan struct{}),
		quitProgressBradcast: make(chan struct{}),
		connectionAdvisor:    topology.NewConnectionAdvisor(c.localId),
		nextSuggestedPeer:    make(chan *enode.Node, 1),
		localEndPointSource:  c.localEndPointSource,
		Instance:             logger.New("PM"),
	}

	h.started.Add(1)

	// dagFetcher handles the non-streaming event fetch path: when peers announce
	// event IDs via NewEventIDsMsg (broadcast), dagFetcher batches and retrieves
	// them via GetEventsMsg/EventsMsg. The streaming path (dagLeecher) handles
	// bulk epoch sync and delivers full events directly to dagProcessor.
	h.dagFetcher = itemsfetcher.New(h.config.Protocol.DagFetcher, itemsfetcher.Callback{
		OnlyInterested: func(ids []interface{}) []interface{} {
			return h.onlyInterestedEventsI(ids)
		},
		Suspend: func() bool {
			return false
		},
	})
	h.txFetcher = itemsfetcher.New(h.config.Protocol.TxFetcher, itemsfetcher.Callback{
		OnlyInterested: func(txids []interface{}) []interface{} {
			return txidsToInterfaces(h.txpool.OnlyNotExisting(interfacesToTxids(txids)))
		},
		Suspend: func() bool {
			return false
		},
	})

	h.dagProcessor = h.makeDagProcessor(c.checkers)

	h.dagLeecher = dagstreamleecher.New(h.store.GetEpoch(), h.store.GetHighestLamport() == 0, h.config.Protocol.DagStreamLeecher, dagstreamleecher.Callbacks{
		IsProcessed: h.store.HasEvent,
		RequestChunk: func(peer string, r dagstream.Request) error {
			p := h.peers.Peer(peer)
			if p == nil {
				return errNotRegistered
			}
			return p.RequestEventsStream(r)
		},
		Suspend: func(_ string) bool {
			return h.dagFetcher.Overloaded() || h.dagProcessor.Overloaded()
		},
		PeerEpoch: func(peer string) idx.Epoch {
			p := h.peers.Peer(peer)
			if p == nil || p.Useless() {
				return 0
			}
			return p.GetProgress().Epoch
		},
	})

	h.dagSeeder = dagstreamseeder.New(h.config.Protocol.DagStreamSeeder, dagstreamseeder.Callbacks{
		ForEachEvent: c.s.ForEachEventRLP,
	})

	return h, nil
}

// peerMisbehaviour checks whether the error warrants banning the peer and removes it if so.
func (h *handler) peerMisbehaviour(peer string, err error) bool {
	if eventcheck.IsBan(err) {
		log.Warn("Dropping peer due to a misbehaviour", "peer", peer, "err", err)
		h.removePeer(peer)
		return true
	}
	return false
}

// removePeer disconnects the peer with the given ID.
func (h *handler) removePeer(id string) {
	peer := h.peers.Peer(id)
	if peer != nil {
		peer.Disconnect(p2p.DiscUselessPeer)
	}
}

// unregisterPeer removes the peer from all sub-components and the peer set.
func (h *handler) unregisterPeer(id string) {
	// Short circuit if the peer was already removed
	peer := h.peers.Peer(id)
	if peer == nil {
		return
	}
	log.Debug("Removing peer", "peer", id)

	// Unregister the peer from the leecher's and seeder's and peer sets
	_ = h.dagLeecher.UnregisterPeer(id)
	_ = h.dagSeeder.UnregisterPeer(id)
	if err := h.peers.UnregisterPeer(id); err != nil {
		log.Error("Peer removal failed", "peer", id, "err", err)
	}
}

// Start launches all broadcast loops, fetchers, processors, and sync handlers.
func (h *handler) Start(maxPeers int) {
	h.maxPeers = maxPeers

	// broadcast transactions
	h.txsCh = make(chan evmcore.NewTxsNotify, txChanSize)
	h.txsSub = h.txpool.SubscribeNewTxsNotify(h.txsCh)
	h.loopsWg.Add(1)
	go h.txBroadcastLoop()

	h.loopsWg.Add(1)
	peerInfoStopChannel := make(chan struct{})
	h.peerInfoStop = peerInfoStopChannel
	go h.peerInfoCollectionLoop(peerInfoStopChannel)

	if h.notifier != nil {
		// broadcast mined events
		h.emittedEventsCh = make(chan *inter.EventPayload, 4)
		h.emittedEventsSub = h.notifier.SubscribeNewEmitted(h.emittedEventsCh)
		// epoch changes
		h.newEpochsCh = make(chan idx.Epoch, 4)
		h.newEpochsSub = h.notifier.SubscribeNewEpoch(h.newEpochsCh)
		h.loopsWg.Add(3)
		go h.emittedBroadcastLoop()
		go h.progressBroadcastLoop()
		go h.onNewEpochLoop()
	}

	// start sync handlers
	go h.txsyncLoop()
	h.dagFetcher.Start()
	h.txFetcher.Start()
	h.checkers.Heavycheck.Start()
	h.dagProcessor.Start()
	h.dagSeeder.Start()
	h.dagLeecher.Start()
	h.started.Done()
}

// Stop shuts down all sub-components, broadcast loops, and peer connections.
func (h *handler) Stop() {
	log.Info("Stopping Sonic protocol")

	h.dagLeecher.Stop()
	h.dagSeeder.Stop()
	h.dagProcessor.Stop()
	h.checkers.Heavycheck.Stop()
	h.txFetcher.Stop()
	h.dagFetcher.Stop()

	close(h.quitProgressBradcast)
	h.txsSub.Unsubscribe() // quits txBroadcastLoop
	if h.notifier != nil {
		h.emittedEventsSub.Unsubscribe() // quits eventBroadcastLoop
		h.newEpochsSub.Unsubscribe()     // quits onNewEpochLoop
	}
	close(h.peerInfoStop)
	h.peerInfoStop = nil

	// Wait for the subscription loops to come down.
	h.loopsWg.Wait()
	h.msgSemaphore.Terminate()

	// Quit the sync loop.
	// After this send has completed, no new peers will be accepted.
	close(h.quitSync)

	// Disconnect existing sessions.
	// This also closes the gate for any new registrations on the peer set.
	// sessions which are already established but not added to h.peers yet
	// will exit when they try to register.
	h.peers.Close()

	// Wait for all peer handler goroutines to come down.
	h.wg.Wait()
	h.peerWG.Wait()

	log.Info("Sonic protocol stopped")
}

// isUseless checks if the peer is banned from discovery and ban it if it should be
func isUseless(node *enode.Node, name string) bool {
	useless := discfilter.Banned(node.ID(), node.Record())
	lowerName := strings.ToLower(name)
	if !useless && !strings.Contains(lowerName, "opera") && !strings.Contains(lowerName, "sonic") {
		useless = true
		discfilter.Ban(node.ID())
	}
	return useless
}

// handle is the callback invoked to manage the life cycle of a peer. When
// this function terminates, the peer is disconnected.
func (h *handler) handle(p *peer) error {
	p.Log().Trace("Connecting peer", "peer", p.ID(), "name", p.Name())
	useless := isUseless(p.Node(), p.Name())
	if !p.Peer.Info().Network.Trusted && useless && h.peers.UselessNum() >= h.maxPeers/10 {
		// don't allow more than 10% of useless peers
		p.Log().Trace("Rejecting peer as useless", "peer", p.ID(), "name", p.Name())
		return p2p.DiscUselessPeer
	}
	if !p.Peer.Info().Network.Trusted && useless {
		p.SetUseless()
	}

	h.peerWG.Add(1)
	defer h.peerWG.Done()

	// Execute the handshake
	var (
		genesis    = h.store.GetGenesisID()
		myProgress = h.myProgress()
	)
	if err := p.Handshake(h.NetworkID, myProgress, common.Hash(genesis)); err != nil {
		p.Log().Debug("Handshake failed", "err", err, "peer", p.ID(), "name", p.Name())
		if !useless {
			discfilter.Ban(p.ID())
		}
		return err
	}

	// Ignore maxPeers if this is a trusted peer
	if h.peers.Len() >= h.maxPeers && !p.Peer.Info().Network.Trusted {
		p.Log().Trace("Rejecting peer as maxPeers is exceeded")
		return p2p.DiscTooManyPeers
	}

	p.Log().Debug("Peer connected", "peer", p.ID(), "name", p.Name())

	// Register the peer locally
	if err := h.peers.RegisterPeer(p); err != nil {
		p.Log().Warn("Peer registration failed", "err", err)
		return err
	}
	if err := h.dagLeecher.RegisterPeer(p.id); err != nil {
		p.Log().Warn("Leecher peer registration failed", "err", err)
		return err
	}
	defer h.unregisterPeer(p.id)

	// Propagate existing transactions. new transactions appearing
	// after this will be sent via broadcasts.
	h.syncTransactions(p, h.txpool.SampleHashes(h.config.Protocol.MaxInitialTxHashesSend))

	// Handle incoming messages until the connection is torn down
	for {
		if err := h.handleMsg(p); err != nil {
			level := slog.LevelWarn
			if errors.Is(err, io.EOF) {
				level = slog.LevelDebug
			}
			p.Log().Log(level, "Message handling failed", "err", err, "peer", p.ID(), "name", p.Name())
			return err
		}
	}
}

// handleMsg is invoked whenever an inbound message is received from a remote
// peer. The remote connection is torn down upon returning any error.
func (h *handler) handleMsg(p *peer) (err error) {
	// Read the next message from the remote peer, and ensure it's fully consumed
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Size > protocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, protocolMaxMsgSize)
	}
	defer caution.ExecuteAndReportError(&err, msg.Discard, "failed to discard message")

	// Acquire semaphore for serialized messages
	eventsSizeEst := dag.Metric{
		Num:  1,
		Size: uint64(msg.Size),
	}
	if !h.msgSemaphore.Acquire(eventsSizeEst, h.config.Protocol.MsgsSemaphoreTimeout) {
		h.Log.Warn("Failed to acquire semaphore for p2p message", "size", msg.Size, "peer", p.id)
		return nil
	}
	defer h.msgSemaphore.Release(eventsSizeEst)

	// Read raw bytes for protobuf decoding
	raw, err := decodeBytes(msg)
	if err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}

	// Dispatch to per-message-type handler
	switch msg.Code {
	case HandshakeMsg:
		return h.handleHandshakeMsg(p, raw)
	case ProgressMsg:
		return h.handleProgressMsg(p, raw)
	case EvmTxsMsg:
		return h.handleEvmTxsMsg(p, raw)
	case NewEvmTxHashesMsg:
		return h.handleNewEvmTxHashesMsg(p, raw)
	case GetEvmTxsMsg:
		return h.handleGetEvmTxsMsg(p, raw)
	case EventsMsg:
		return h.handleEventsMsg(p, raw)
	case NewEventIDsMsg:
		return h.handleNewEventIDsMsg(p, raw)
	case GetEventsMsg:
		return h.handleGetEventsMsg(p, raw)
	case RequestEventsStream:
		return h.handleRequestEventsStream(p, raw)
	case EventsStreamResponse:
		return h.handleEventsStreamResponse(p, raw)
	case GetPeerInfosMsg:
		return h.handleGetPeerInfosMsg(p, raw)
	case PeerInfosMsg:
		return h.handlePeerInfosMsg(p, raw)
	case GetEndPointMsg:
		return h.handleGetEndPointMsg(p, raw)
	case EndPointUpdateMsg:
		return h.handleEndPointUpdateMsg(p, raw)
	default:
		return errResp(ErrInvalidMsgCode, "%v", msg.Code)
	}
}
