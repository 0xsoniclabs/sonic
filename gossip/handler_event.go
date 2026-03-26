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
	"math"
	"slices"
	"time"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/utils/txtime"
)

// interfacesToEventIDs converts a slice of interface values to event IDs.
func interfacesToEventIDs(ids []interface{}) hash.Events {
	res := make(hash.Events, len(ids))
	for i, id := range ids {
		res[i] = id.(hash.Event)
	}
	return res
}

// eventIDsToInterfaces converts a slice of event IDs to interface values.
func eventIDsToInterfaces(ids hash.Events) []interface{} {
	res := make([]interface{}, len(ids))
	for i, id := range ids {
		res[i] = id
	}
	return res
}

// handleEventsMsg processes full events received from a peer.
func (h *handler) handleEventsMsg(p *peer, raw []byte) error {
	events, err := unmarshalEvents(raw)
	if err != nil {
		return errResp(ErrDecode, "%v", err)
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
	return nil
}

// handleNewEventIDsMsg processes event ID announcements from a peer.
func (h *handler) handleNewEventIDsMsg(p *peer, raw []byte) error {
	announces, err := unmarshalEventIDs(raw)
	if err != nil {
		return errResp(ErrDecode, "%v", err)
	}
	if err := checkLenLimits(len(announces), announces); err != nil {
		return err
	}
	h.handleEventHashes(p, announces)
	return nil
}

// handleGetEventsMsg processes a request for events by ID.
func (h *handler) handleGetEventsMsg(p *peer, raw []byte) error {
	requests, err := unmarshalEventIDs(raw)
	if err != nil {
		return errResp(ErrDecode, "%v", err)
	}
	if err := checkLenLimits(len(requests), requests); err != nil {
		return err
	}
	rawEvents := make([][]byte, 0, len(requests))
	ids := make(hash.Events, 0, len(requests))
	size := 0
	for _, id := range requests {
		if eventBytes := h.store.GetEventPayloadRLP(id); eventBytes != nil {
			rawEvents = append(rawEvents, eventBytes)
			ids = append(ids, id)
			size += len(eventBytes)
		} else {
			h.Log.Debug("requested event not found", "hash", id)
		}
		if size >= softResponseLimitSize {
			break
		}
	}
	if len(rawEvents) != 0 {
		p.EnqueueSendEventsRaw(rawEvents, ids, p.queue)
	}
	return nil
}

// handleEventHashes processes event ID announcements from broadcast (NewEventIDsMsg).
// These are fed to dagFetcher which batches and retrieves full events via
// GetEventsMsg/EventsMsg. This is separate from the streaming path (dagLeecher)
// which delivers full events directly during bulk epoch sync.
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

// handleEvents processes received events, filters by lamport height, and enqueues them to the DAG processor.
func (h *handler) handleEvents(peer *peer, events dag.Events, ordered bool) {
	// Reward peer for delivering valid events
	peer.AddScore(1)
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

// decideBroadcastAggressiveness computes the number of full-broadcast recipients
// based on event size, age, and the latency/throughput tradeoff configuration.
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

	// Sort peers by quality score (descending) so high-quality peers get full events first
	slices.SortFunc(peers, func(a, b *peer) int {
		return int(b.Score() - a.Score())
	})

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

// emittedBroadcastLoop broadcasts locally emitted events immediately upon creation.
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
