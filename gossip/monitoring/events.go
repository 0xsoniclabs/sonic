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

package monitoring

import (
	"time"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/utils/wlru"
	"github.com/ethereum/go-ethereum/metrics"
)

//go:generate mockgen -source=events.go -destination=events_mock.go -package=monitoring

// EventMonitor is a component that tracks the forwarding of events.
type EventMonitor interface {
	// OnIncomingEvent is called when an event is received by a node.
	OnIncomingEvent(hash.Event)
	// OnOutgoingEvent is called when an event is forwarded to a peer.
	OnOutgoingEvent(hash.Event)
}

var (
	// incomingEventCounter is a metric tracking the number of incoming events
	incomingEventCounter = metrics.GetOrRegisterCounter("p2p_incoming_event", nil)

	// outgoingEventCounter is a metric tracking the number of outgoing events
	outgoingEventCounter = metrics.GetOrRegisterCounter("p2p_outgoing_event", nil)

	// eventForwardingDelay is a metric tracking the latency distribution of
	// forwarding events to peers. It tracks the time from receiving an event
	// until the first time an event is being forwarded to any peer. If it is
	// not forwarded, no data point is recorded.
	eventForwardingDelay = metrics.GetOrRegisterHistogramLazy("p2p_event_forwarding_delay", nil, func() metrics.Sample {
		return metrics.ResettingSample(
			metrics.NewExpDecaySample(1028, 0.015),
		)
	})
)

// NewEventMonitor creates a new EventMonitor instance tracking the number of
// incoming and outgoing events as well as the delay between receiving an event
// and forwarding it to a peer. All metrics are tracked using prometheus metrics.
func NewEventMonitor() *eventMonitor {
	return newEventMonitor(
		incomingEventCounter,
		outgoingEventCounter,
		eventForwardingDelay,
	)
}

func newEventMonitor(
	incoming counterMetric,
	outgoing counterMetric,
	delay valueMetric,
) *eventMonitor {
	const cacheSize = 64 * 1024                // 64k entries, ~2.7 MiB
	cache, _ := wlru.New(cacheSize, cacheSize) // Only results in an error if size is negative.
	return &eventMonitor{
		incoming: incoming,
		outgoing: outgoing,
		delay:    delay,
		seen:     cache,
	}
}

type eventMonitor struct {
	incoming counterMetric
	outgoing counterMetric
	delay    valueMetric
	seen     *wlru.Cache
}

func (m *eventMonitor) OnIncomingEvent(event hash.Event) {
	m.incoming.Inc(1)
	if m.seen.Contains(event) {
		return // Already seen this event, no need to record incoming time.
	}
	now := time.Now()
	m.seen.Add(event, &eventInfo{firstIncoming: now}, 1)
}

func (m *eventMonitor) OnOutgoingEvent(event hash.Event) {
	m.outgoing.Inc(1)
	entry, found := m.seen.Get(event)
	if !found {
		return // This event was never seen, so there is no way to record forwarding time.
	}
	info := entry.(*eventInfo)
	if info.forwarded {
		return // This event was already forwarded, no need to record again.
	}
	info.forwarded = true
	m.delay.Update(time.Since(info.firstIncoming).Milliseconds())
}

type counterMetric interface {
	Inc(int64)
}

type valueMetric interface {
	Update(int64)
}

// eventInfo holds information about an event's forwarding state.
type eventInfo struct {
	firstIncoming time.Time // Time when the event was first received.
	forwarded     bool      // false if not yet forwarded, true otherwise
}
