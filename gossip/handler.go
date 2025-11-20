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

package gossip

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/0xsoniclabs/sonic/eventcheck"
	"github.com/0xsoniclabs/sonic/eventcheck/epochcheck"
	"github.com/0xsoniclabs/sonic/eventcheck/heavycheck"
	"github.com/0xsoniclabs/sonic/eventcheck/parentlesscheck"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/protocols/dag/dagstream"
	"github.com/0xsoniclabs/sonic/gossip/protocols/dag/dagstream/dagstreamleecher"
	"github.com/0xsoniclabs/sonic/gossip/protocols/dag/dagstream/dagstreamseeder"
	"github.com/0xsoniclabs/sonic/gossip/topology"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/utils/caution"
	"github.com/0xsoniclabs/sonic/utils/txtime"
	"github.com/Fantom-foundation/lachesis-base/gossip/dagprocessor"
	"github.com/Fantom-foundation/lachesis-base/gossip/itemsfetcher"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/utils/datasemaphore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	notify "github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover/discfilter"
	"github.com/ethereum/go-ethereum/rlp"
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

func errResp(code errCode, format string, v ...interface{}) error {
	return fmt.Errorf("%v - %v", code, fmt.Sprintf(format, v...))
}

func checkLenLimits(size int, v interface{}) error {
	if size <= 0 {
		return errResp(ErrEmptyMessage, "%v", v)
	}
	if size > hardLimitItems {
		return errResp(ErrMsgTooLarge, "%v", v)
	}
	return nil
}

type dagNotifier interface {
	SubscribeNewEpoch(ch chan<- idx.Epoch) notify.Subscription
	SubscribeNewEmitted(ch chan<- *inter.EventPayload) notify.Subscription
}

type processCallback struct {
	Event         func(*inter.EventPayload) error
	SwitchEpochTo func(idx.Epoch) error
}

// handlerConfig is the collection of initialization parameters to create a full
// node network handler.
type handlerConfig struct {
	config   Config
	notifier dagNotifier
	txpool   TxPool
	engineMu sync.Locker
	checkers *eventcheck.Checkers
	s        *Store
	process  processCallback
	localId  enode.ID

	localEndPointSource LocalEndPointSource
}

type LocalEndPointSource interface {
	GetLocalEndPoint() *enode.Node
}

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

		localEndPointSource: c.localEndPointSource,

		Instance: logger.New("PM"),
	}
	h.started.Add(1)

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

func (h *handler) peerMisbehaviour(peer string, err error) bool {
	if eventcheck.IsBan(err) {
		log.Warn("Dropping peer due to a misbehaviour", "peer", peer, "err", err)
		h.removePeer(peer)
		return true
	}
	return false
}

func (h *handler) makeDagProcessor(checkers *eventcheck.Checkers) *dagprocessor.Processor {
	// checkers
	lightCheck := func(e dag.Event) error {
		if h.store.GetEpoch() != e.ID().Epoch() {
			return epochcheck.ErrNotRelevant
		}
		if h.dagProcessor.IsBuffered(e.ID()) {
			return eventcheck.ErrDuplicateEvent
		}
		if h.store.HasEvent(e.ID()) {
			return eventcheck.ErrAlreadyConnectedEvent
		}
		if err := checkers.Basiccheck.Validate(e.(inter.EventPayloadI)); err != nil {
			return err
		}
		if err := checkers.Epochcheck.Validate(e.(inter.EventPayloadI)); err != nil {
			return err
		}
		return nil
	}
	bufferedCheck := func(_e dag.Event, _parents dag.Events) error {
		e := _e.(inter.EventPayloadI)
		parents := make(inter.EventIs, len(_parents))
		for i := range _parents {
			parents[i] = _parents[i].(inter.EventI)
		}
		return validateEventPropertiesDependingOnParents(checkers, e, parents)
	}
	parentlessChecker := parentlesscheck.Checker{
		HeavyCheck: &heavycheck.EventsOnly{Checker: checkers.Heavycheck},
		LightCheck: lightCheck,
	}
	newProcessor := dagprocessor.New(datasemaphore.New(h.config.Protocol.EventsSemaphoreLimit, getSemaphoreWarningFn("DAG events")), h.config.Protocol.DagProcessor, dagprocessor.Callback{
		// DAG callbacks
		Event: dagprocessor.EventCallback{
			Process: func(_e dag.Event) error {
				e := _e.(*inter.EventPayload)
				preStart := time.Now()
				h.engineMu.Lock()
				defer h.engineMu.Unlock()

				err := h.process.Event(e)
				if err != nil {
					return err
				}

				// event is connected, announce it
				passedSinceEvent := preStart.Sub(e.CreationTime().Time())
				h.BroadcastEvent(e, passedSinceEvent)

				return nil
			},
			Released: func(e dag.Event, peer string, err error) {
				if eventcheck.IsBan(err) {
					log.Warn("Incoming event rejected", "event", e.ID().String(), "creator", e.Creator(), "err", err)
					h.removePeer(peer)
				}

				if errors.Is(err, eventcheck.ErrSpilledEvent) {
					incompleteEventsSpilled.Inc(1)
				}
			},

			Exists: func(id hash.Event) bool {
				return h.store.HasEvent(id)
			},

			Get: func(id hash.Event) dag.Event {
				e := h.store.GetEventPayload(id)
				if e == nil {
					return nil
				}
				return e
			},

			CheckParents:    bufferedCheck,
			CheckParentless: parentlessChecker.Enqueue,
		},
		HighestLamport: h.store.GetHighestLamport,
	})

	return newProcessor
}

func validateEventPropertiesDependingOnParents(
	checkers *eventcheck.Checkers,
	event inter.EventPayloadI,
	parents inter.EventIs,
) error {
	var selfParent inter.EventI
	if event.SelfParent() != nil {
		selfParent = parents[0]
	}
	if err := checkers.Parentscheck.Validate(event, parents); err != nil {
		return err
	}
	if err := checkers.Gaspowercheck.Validate(event, selfParent); err != nil {
		return err
	}
	if err := checkers.Proposalcheck.Validate(event); err != nil {
		return err
	}
	return nil
}

func (h *handler) isEventInterested(id hash.Event, epoch idx.Epoch) bool {
	if id.Epoch() != epoch {
		return false
	}

	if h.dagProcessor.IsBuffered(id) || h.store.HasEvent(id) {
		return false
	}
	return true
}

func (h *handler) onlyInterestedEventsI(ids []interface{}) []interface{} {
	if len(ids) == 0 {
		return ids
	}
	epoch := h.store.GetEpoch()
	interested := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		if h.isEventInterested(id.(hash.Event), epoch) {
			interested = append(interested, id)
		}
	}
	return interested
}

func (h *handler) removePeer(id string) {
	peer := h.peers.Peer(id)
	if peer != nil {
		peer.Disconnect(p2p.DiscUselessPeer)
	}
}

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

func (h *handler) myProgress() PeerProgress {
	bs := h.store.GetBlockState()
	epoch := h.store.GetEpoch()
	return PeerProgress{
		Epoch:            epoch,
		LastBlockIdx:     bs.LastBlock.Idx,
		LastBlockAtropos: bs.LastBlock.Atropos,
	}
}

func (h *handler) highestPeerProgress() PeerProgress {
	peers := h.peers.List()
	max := h.myProgress()
	for _, peer := range peers {
		peerProgress := peer.GetProgress()
		if max.LastBlockIdx < peerProgress.LastBlockIdx {
			max = peerProgress
		}
	}
	return max
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

func interfacesToEventIDs(ids []interface{}) hash.Events {
	res := make(hash.Events, len(ids))
	for i, id := range ids {
		res[i] = id.(hash.Event)
	}
	return res
}

func eventIDsToInterfaces(ids hash.Events) []interface{} {
	res := make([]interface{}, len(ids))
	for i, id := range ids {
		res[i] = id
	}
	return res
}

func interfacesToTxids(ids []interface{}) []common.Hash {
	res := make([]common.Hash, len(ids))
	for i, id := range ids {
		res[i] = id.(common.Hash)
	}
	return res
}

func txidsToInterfaces(ids []common.Hash) []interface{} {
	res := make([]interface{}, len(ids))
	for i, id := range ids {
		res[i] = id
	}
	return res
}

func (h *handler) handleTxHashes(p *peer, announces []common.Hash) {
	// Mark the hashes as present at the remote node
	now := time.Now()
	for _, id := range announces {
		txtime.Saw(id, now)
		p.MarkTransaction(id)
	}
	// Schedule all the unknown hashes for retrieval
	requestTransactions := func(ids []interface{}) error {
		return p.RequestTransactions(interfacesToTxids(ids))
	}
	_ = h.txFetcher.NotifyAnnounces(p.id, txidsToInterfaces(announces), time.Now(), requestTransactions)
}

func (h *handler) handleTxs(p *peer, txs types.Transactions) {
	// Mark the hashes as present at the remote node
	now := time.Now()
	for _, tx := range txs {
		txid := tx.Hash()
		txtime.Saw(txid, now)
		p.MarkTransaction(txid)
	}
	h.txpool.AddRemotes(txs)
}

func (h *handler) handleEventHashes(p *peer, announces hash.Events) {
	// Mark the hashes as present at the remote node
	for _, id := range announces {
		p.MarkEvent(id)
	}
	// filter too high IDs
	notTooHigh := make(hash.Events, 0, len(announces))
	sessionCfg := h.config.Protocol.DagStreamLeecher.Session
	for _, id := range announces {
		maxLamport := h.store.GetHighestLamport() + idx.Lamport(sessionCfg.DefaultChunkItemsNum+1)*idx.Lamport(sessionCfg.ParallelChunksDownload)
		if id.Lamport() <= maxLamport {
			notTooHigh = append(notTooHigh, id)
		}
	}
	if len(announces) != len(notTooHigh) {
		h.dagLeecher.ForceSyncing()
	}
	if len(notTooHigh) == 0 {
		return
	}
	// Schedule all the unknown hashes for retrieval
	requestEvents := func(ids []interface{}) error {
		return p.RequestEvents(interfacesToEventIDs(ids))
	}
	_ = h.dagFetcher.NotifyAnnounces(p.id, eventIDsToInterfaces(notTooHigh), time.Now(), requestEvents)
}

func (h *handler) handleEvents(peer *peer, events dag.Events, ordered bool) {
	// Mark the hashes as present at the remote node
	now := time.Now()
	for _, e := range events {
		for _, tx := range e.(inter.EventPayloadI).Transactions() {
			txtime.Saw(tx.Hash(), now)
		}
		peer.MarkEvent(e.ID())
	}
	// filter too high events
	notTooHigh := make(dag.Events, 0, len(events))
	sessionCfg := h.config.Protocol.DagStreamLeecher.Session
	for _, e := range events {
		maxLamport := h.store.GetHighestLamport() + idx.Lamport(sessionCfg.DefaultChunkItemsNum+1)*idx.Lamport(sessionCfg.ParallelChunksDownload)
		if e.Lamport() <= maxLamport {
			notTooHigh = append(notTooHigh, e)
		}
		if now.Sub(e.(inter.EventI).CreationTime().Time()) < 10*time.Minute {
			h.syncStatus.MarkMaybeSynced()
		}
	}
	if len(events) != len(notTooHigh) {
		h.dagLeecher.ForceSyncing()
	}
	if len(notTooHigh) == 0 {
		return
	}
	// Schedule all the events for connection
	requestEvents := func(ids []interface{}) error {
		return peer.RequestEvents(interfacesToEventIDs(ids))
	}
	notifyAnnounces := func(ids hash.Events) {
		_ = h.dagFetcher.NotifyAnnounces(peer.id, eventIDsToInterfaces(ids), now, requestEvents)
	}
	err := h.dagProcessor.Enqueue(peer.id, notTooHigh, ordered, notifyAnnounces, nil)
	if err != nil {
		// This error typically occurs when the number of events exceeds the EventsSemaphoreLimit
		// or if the dagProcessor has been stopped. Since the specific error is internal to the package,
		// we cannot distinguish between these cases here.
		// The current shutdown process stops the handler after the dagProcessor, so this warning should not
		// appear during normal de-initialization.
		//
		// The EventsSemaphoreLimit should be at least twice the maximum number of allowed incomplete buffered events.
		// This ensures that the queue can handle a full load plus additional incoming events.
		// If this warning appears, it likely indicates a misconfiguration.
		log.Warn("Unable to enqueue events", "from", peer.id, "events count", len(notTooHigh), "error", err)
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

	// Handle the message depending on its contents
	switch msg.Code {
	case HandshakeMsg:
		// Status messages should never arrive after the handshake
		return errResp(ErrExtraStatusMsg, "uncontrolled status message")

	case ProgressMsg:
		var progress PeerProgress
		if err := msg.Decode(&progress); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		p.SetProgress(progress)

	case EvmTxsMsg:
		// Transactions arrived, make sure we have a valid and fresh graph to handle them
		if !h.syncStatus.AcceptTxs() {
			break
		}
		// Transactions can be processed, parse all of them and deliver to the pool
		var txs types.Transactions
		if err := msg.Decode(&txs); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		if err := checkLenLimits(len(txs), txs); err != nil {
			return err
		}
		txids := make([]interface{}, txs.Len())
		for i, tx := range txs {
			txids[i] = tx.Hash()
		}
		_ = h.txFetcher.NotifyReceived(txids)
		h.handleTxs(p, txs)

	case NewEvmTxHashesMsg:
		// Transactions arrived, make sure we have a valid and fresh graph to handle them
		if !h.syncStatus.AcceptTxs() {
			break
		}
		// Transactions can be processed, parse all of them and deliver to the pool
		var txHashes []common.Hash
		if err := msg.Decode(&txHashes); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		if err := checkLenLimits(len(txHashes), txHashes); err != nil {
			return err
		}
		h.handleTxHashes(p, txHashes)

	case GetEvmTxsMsg:
		var requests []common.Hash
		if err := msg.Decode(&requests); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		if err := checkLenLimits(len(requests), requests); err != nil {
			return err
		}

		txs := make(types.Transactions, 0, len(requests))
		for _, txid := range requests {
			tx := h.txpool.Get(txid)
			if tx == nil {
				continue
			}
			txs = append(txs, tx)
		}
		SplitTransactions(txs, func(batch types.Transactions) {
			p.EnqueueSendTransactions(batch, p.queue)
		})

	case EventsMsg:
		var events inter.EventPayloads
		if err := msg.Decode(&events); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		if err := checkLenLimits(len(events), events); err != nil {
			return err
		}

		// Replace transactions in event with the instances found in the pool
		// This allows to reuse instances of already known transactions, which
		// already have cached the sender. Saving the cost of resolving the
		// the signature again.
		for i := range events {
			txs := events[i].Transactions()
			for i := range txs {
				if tx := h.txpool.Get(txs[i].Hash()); tx != nil {
					txs[i] = tx
				}
			}
		}

		_ = h.dagFetcher.NotifyReceived(eventIDsToInterfaces(events.IDs()))
		h.handleEvents(p, events.Bases(), events.Len() > 1)

	case NewEventIDsMsg:
		var announces hash.Events
		if err := msg.Decode(&announces); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		if err := checkLenLimits(len(announces), announces); err != nil {
			return err
		}
		h.handleEventHashes(p, announces)

	case GetEventsMsg:
		var requests hash.Events
		if err := msg.Decode(&requests); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		if err := checkLenLimits(len(requests), requests); err != nil {
			return err
		}

		rawEvents := make([]rlp.RawValue, 0, len(requests))
		ids := make(hash.Events, 0, len(requests))
		size := 0
		for _, id := range requests {
			if raw := h.store.GetEventPayloadRLP(id); raw != nil {
				rawEvents = append(rawEvents, raw)
				ids = append(ids, id)
				size += len(raw)
			} else {
				h.Log.Debug("requested event not found", "hash", id)
			}
			if size >= softResponseLimitSize {
				break
			}
		}
		if len(rawEvents) != 0 {
			p.EnqueueSendEventsRLP(rawEvents, ids, p.queue)
		}

	case RequestEventsStream:
		var request dagstream.Request
		if err := msg.Decode(&request); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		if request.Limit.Num > hardLimitItems-1 {
			return errResp(ErrMsgTooLarge, "%v", msg)
		}
		if request.Limit.Size > protocolMaxMsgSize*2/3 {
			return errResp(ErrMsgTooLarge, "%v", msg)
		}

		pid := p.id
		_, peerErr := h.dagSeeder.NotifyRequestReceived(dagstreamseeder.Peer{
			ID:        pid,
			SendChunk: p.SendEventsStream,
			Misbehaviour: func(err error) {
				h.peerMisbehaviour(pid, err)
			},
		}, request)
		if peerErr != nil {
			return peerErr
		}

	case EventsStreamResponse:
		var chunk dagChunk
		if err := msg.Decode(&chunk); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		if err := checkLenLimits(len(chunk.Events)+len(chunk.IDs)+1, chunk); err != nil {
			return err
		}

		if (len(chunk.Events) != 0) && (len(chunk.IDs) != 0) {
			return errors.New("expected either events or event hashes")
		}
		var last hash.Event
		if len(chunk.IDs) != 0 {
			h.handleEventHashes(p, chunk.IDs)
			last = chunk.IDs[len(chunk.IDs)-1]
		}
		if len(chunk.Events) != 0 {
			h.handleEvents(p, chunk.Events.Bases(), true)
			last = chunk.Events[len(chunk.Events)-1].ID()
		}

		_ = h.dagLeecher.NotifyChunkReceived(chunk.SessionID, last, chunk.Done)

	case GetPeerInfosMsg:
		infos := []peerInfo{}
		for _, peer := range h.peers.List() {
			if peer.Useless() {
				continue
			}
			info := peer.endPoint.Load()
			if info == nil {
				continue
			}
			infos = append(infos, peerInfo{
				Enode: info.enode.String(),
			})
		}
		err := p2p.Send(p.rw, PeerInfosMsg, peerInfoMsg{
			Peers: infos,
		})
		if err != nil {
			return err
		}

	case PeerInfosMsg:
		var infos peerInfoMsg
		if err := msg.Decode(&infos); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}

		reportedPeers := []*enode.Node{}
		for _, info := range infos.Peers {
			var enode enode.Node
			if err := enode.UnmarshalText([]byte(info.Enode)); err != nil {
				h.Log.Warn("Failed to unmarshal enode", "enode", info.Enode, "err", err)
			} else {
				reportedPeers = append(reportedPeers, &enode)
			}
		}

		h.connectionAdvisor.UpdatePeers(p.ID(), reportedPeers)

	case GetEndPointMsg:
		source := h.localEndPointSource
		if source == nil {
			return nil
		}
		enode := source.GetLocalEndPoint()
		if enode == nil {
			return nil
		}
		if err := p2p.Send(p.rw, EndPointUpdateMsg, enode.String()); err != nil {
			return err
		}

	case EndPointUpdateMsg:
		var encoded string
		if err := msg.Decode(&encoded); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		var enode enode.Node
		if err := enode.UnmarshalText([]byte(encoded)); err != nil {
			h.Log.Warn("Failed to unmarshal enode", "enode", encoded, "err", err)
		} else {
			p.endPoint.Store(&peerEndPointInfo{
				enode:     enode,
				timestamp: time.Now(),
			})
		}

	default:
		return errResp(ErrInvalidMsgCode, "%v", msg.Code)
	}
	return nil
}

func (h *handler) decideBroadcastAggressiveness(size int, passed time.Duration, peersNum int) int {
	percents := 100
	maxPercents := 1000000 * percents
	latencyVsThroughputTradeoff := maxPercents
	cfg := h.config.Protocol
	if cfg.ThroughputImportance != 0 {
		latencyVsThroughputTradeoff = (cfg.LatencyImportance * percents) / cfg.ThroughputImportance
	}

	broadcastCost := passed * time.Duration(128+size) / 128
	broadcastAllCostTarget := time.Duration(latencyVsThroughputTradeoff) * (700 * time.Millisecond) / time.Duration(percents)
	broadcastSqrtCostTarget := broadcastAllCostTarget * 10

	fullRecipients := 0
	if latencyVsThroughputTradeoff >= maxPercents {
		// edge case
		fullRecipients = peersNum
	} else if latencyVsThroughputTradeoff <= 0 {
		// edge case
		fullRecipients = 0
	} else if broadcastCost <= broadcastAllCostTarget {
		// if event is small or was created recently, always send to everyone full event
		fullRecipients = peersNum
	} else if broadcastCost <= broadcastSqrtCostTarget || passed == 0 {
		// if event is big but was created recently, send full event to subset of peers
		fullRecipients = int(math.Sqrt(float64(peersNum)))
		if fullRecipients < 4 {
			fullRecipients = 4
		}
	}
	if fullRecipients > peersNum {
		fullRecipients = peersNum
	}
	return fullRecipients
}

// BroadcastEvent will either propagate a event to a subset of it's peers, or
// will only announce it's availability (depending what's requested).
func (h *handler) BroadcastEvent(event *inter.EventPayload, passed time.Duration) int {
	if passed < 0 {
		passed = 0
	}
	id := event.ID()
	peers := h.peers.PeersWithoutEvent(id)
	if len(peers) == 0 {
		log.Trace("Event is already known to all peers", "hash", id)
		return 0
	}

	fullRecipients := h.decideBroadcastAggressiveness(event.Size(), passed, len(peers))

	// Exclude low quality peers from fullBroadcast
	var fullBroadcast = make([]*peer, 0, fullRecipients)
	var hashBroadcast = make([]*peer, 0, len(peers))
	for _, p := range peers {
		if !p.Useless() && len(fullBroadcast) < fullRecipients {
			fullBroadcast = append(fullBroadcast, p)
		} else {
			hashBroadcast = append(hashBroadcast, p)
		}
	}
	for _, peer := range fullBroadcast {
		peer.AsyncSendEvents(inter.EventPayloads{event}, peer.queue)
	}
	// Broadcast of event hash to the rest peers
	for _, peer := range hashBroadcast {
		peer.AsyncSendEventIDs(hash.Events{event.ID()}, peer.queue)
	}
	log.Trace("Broadcast event", "hash", id, "fullRecipients", len(fullBroadcast), "hashRecipients", len(hashBroadcast))
	return len(peers)
}

// BroadcastTxs will propagate a batch of transactions to all peers which are not known to
// already have the given transaction.
func (h *handler) BroadcastTxs(txs types.Transactions) {
	broadcastedTxsCounter.Inc(int64(txs.Len()))
	var txset = make(map[*peer]types.Transactions)

	// Broadcast transactions to a batch of peers not knowing about it
	totalSize := common.StorageSize(0)
	for _, tx := range txs {
		peers := h.peers.PeersWithoutTx(tx.Hash())
		for _, peer := range peers {
			txset[peer] = append(txset[peer], tx)
		}
		totalSize += common.StorageSize(tx.Size())
		log.Trace("Broadcast transaction", "hash", tx.Hash(), "recipients", len(peers))
	}
	fullRecipients := h.decideBroadcastAggressiveness(int(totalSize), time.Second, len(txset))
	i := 0
	for peer, txs := range txset {
		SplitTransactions(txs, func(batch types.Transactions) {
			if i < fullRecipients {
				peer.AsyncSendTransactions(batch, peer.queue)
			} else {
				txids := make([]common.Hash, batch.Len())
				for i, tx := range batch {
					txids[i] = tx.Hash()
				}
				peer.AsyncSendTransactionHashes(txids, peer.queue)
			}
		})
		i++
	}
}

// Mined broadcast loop
func (h *handler) emittedBroadcastLoop() {
	defer h.loopsWg.Done()
	for {
		select {
		case emitted := <-h.emittedEventsCh:
			// If this node starts emitting events, it is considered synced and
			// can start accepting transactions.
			h.syncStatus.MarkMaybeSynced()
			h.BroadcastEvent(emitted, 0)
		// Err() channel will be closed when unsubscribing.
		case <-h.emittedEventsSub.Err():
			return
		}
	}
}

func (h *handler) broadcastProgress() {
	progress := h.myProgress()
	for _, peer := range h.peers.List() {
		peer.AsyncSendProgress(progress, peer.queue)
	}
}

// Progress broadcast loop
func (h *handler) progressBroadcastLoop() {
	ticker := time.NewTicker(h.config.Protocol.ProgressBroadcastPeriod)
	defer ticker.Stop()
	defer h.loopsWg.Done()
	// automatically stops if unsubscribe
	for {
		select {
		case <-ticker.C:
			h.broadcastProgress()
		case <-h.quitProgressBradcast:
			return
		}
	}
}

func (h *handler) onNewEpochLoop() {
	defer h.loopsWg.Done()
	for {
		select {
		case myEpoch := <-h.newEpochsCh:
			h.dagProcessor.Clear()
			h.dagLeecher.OnNewEpoch(myEpoch)
		// Err() channel will be closed when unsubscribing.
		case <-h.newEpochsSub.Err():
			return
		}
	}
}

func (h *handler) txBroadcastLoop() {
	ticker := time.NewTicker(h.config.Protocol.RandomTxHashesSendPeriod)
	defer ticker.Stop()
	defer h.loopsWg.Done()
	for {
		select {
		case notify := <-h.txsCh:
			h.BroadcastTxs(notify.Txs)

		// Err() channel will be closed when unsubscribing.
		case <-h.txsSub.Err():
			return

		case <-ticker.C:
			if !h.syncStatus.AcceptTxs() {
				break
			}
			peers := h.peers.List()
			if len(peers) == 0 {
				continue
			}
			randPeer := peers[rand.IntN(len(peers))]
			h.syncTransactions(randPeer, h.txpool.SampleHashes(h.config.Protocol.MaxRandomTxHashesSend))
		}
	}
}

func (h *handler) peerInfoCollectionLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(h.config.Protocol.PeerInfoCollectionPeriod)
	defer ticker.Stop()
	defer h.loopsWg.Done()
	for {
		select {
		case <-ticker.C:
			// Get a suggestion for a new peer.
			suggestion := h.connectionAdvisor.GetNewPeerSuggestion()
			if suggestion != nil {
				select {
				case h.nextSuggestedPeer <- suggestion:
				default:
				}
			}

			// Request updated peer information from current peers.
			peers := h.peers.List()
			for _, peer := range peers {
				// If we do not have the peer's end-point or it is too old, request it.
				if info := peer.endPoint.Load(); info == nil || time.Since(info.timestamp) > h.config.Protocol.PeerEndPointUpdatePeriod {
					if err := peer.SendEndPointUpdateRequest(); err != nil {
						log.Warn("Failed to send end-point update request", "peer", peer.id, "err", err)
						// If the end-point update request fails, do not send the peer info request.
						continue
					}
				}

				if err := peer.SendPeerInfoRequest(); err != nil {
					log.Warn("Failed to send peer info request", "peer", peer.id, "err", err)
				}
			}

			// Drop a redundant connection if there are too many connections.
			if suggestion != nil && len(peers) >= h.maxPeers-1 {
				redundant := h.connectionAdvisor.GetRedundantPeerSuggestion()
				if redundant != nil {
					for _, peer := range peers {
						if peer.Node().ID() == *redundant {
							peer.Disconnect(p2p.DiscTooManyPeers)
							break
						}
					}
				}
			}

		case <-stop:
			return
		}
	}
}

// NodeInfo represents a short summary of the sub-protocol metadata
// known about the host peer.
type NodeInfo struct {
	Network     uint64      `json:"network"` // network ID
	Genesis     common.Hash `json:"genesis"` // SHA3 hash of the host's genesis object
	Epoch       idx.Epoch   `json:"epoch"`
	NumOfBlocks idx.Block   `json:"blocks"`
	//Config  *params.ChainConfig `json:"config"`  // Chain configuration for the fork rules
}

// NodeInfo retrieves some protocol metadata about the running host node.
func (h *handler) NodeInfo() *NodeInfo {
	numOfBlocks := h.store.GetLatestBlockIndex()
	return &NodeInfo{
		Network:     h.NetworkID,
		Genesis:     common.Hash(h.store.GetGenesisID()),
		Epoch:       h.store.GetEpoch(),
		NumOfBlocks: numOfBlocks,
	}
}

func getSemaphoreWarningFn(name string) func(dag.Metric, dag.Metric, dag.Metric) {
	return func(received dag.Metric, processing dag.Metric, releasing dag.Metric) {
		log.Warn(fmt.Sprintf("%s semaphore inconsistency", name),
			"receivedNum", received.Num, "receivedSize", received.Size,
			"processingNum", processing.Num, "processingSize", processing.Size,
			"releasingNum", releasing.Num, "releasingSize", releasing.Size)
	}
}

func (h *handler) GetSuggestedPeerIterator() enode.Iterator {
	return &suggestedPeerIterator{
		handler: h,
		close:   make(chan struct{}),
	}
}

type suggestedPeerIterator struct {
	handler *handler
	next    *enode.Node
	close   chan struct{}
}

func (i *suggestedPeerIterator) Next() bool {
	select {
	case i.next = <-i.handler.nextSuggestedPeer:
		return true
	case <-i.close:
		return false
	}
}

func (i *suggestedPeerIterator) Node() *enode.Node {
	return i.next
}

func (i *suggestedPeerIterator) Close() {
	close(i.close)
}
