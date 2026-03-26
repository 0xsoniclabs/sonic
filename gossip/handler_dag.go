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
	"time"

	"github.com/Fantom-foundation/lachesis-base/gossip/dagprocessor"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/utils/datasemaphore"
	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/eventcheck"
	"github.com/0xsoniclabs/sonic/eventcheck/epochcheck"
	"github.com/0xsoniclabs/sonic/eventcheck/heavycheck"
	"github.com/0xsoniclabs/sonic/eventcheck/parentlesscheck"
	"github.com/0xsoniclabs/sonic/inter"
)

// getSemaphoreWarningFn returns a warning logger for semaphore inconsistencies.
func getSemaphoreWarningFn(name string) func(dag.Metric, dag.Metric, dag.Metric) {
	return func(received dag.Metric, processing dag.Metric, releasing dag.Metric) {
		log.Warn(fmt.Sprintf("%s semaphore inconsistency", name),
			"receivedNum", received.Num, "receivedSize", received.Size,
			"processingNum", processing.Num, "processingSize", processing.Size,
			"releasingNum", releasing.Num, "releasingSize", releasing.Size)
	}
}

// makeDagProcessor builds the DAG event processor with light, buffered, and parentless checks.
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
				} else if err != nil {
					if p := h.peers.Peer(peer); p != nil {
						p.AddScore(-10)
					}
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

// validateEventPropertiesDependingOnParents runs parent-dependent validation checks on an event.
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

// isEventInterested reports whether the event is needed for the current epoch and not already known.
func (h *handler) isEventInterested(id hash.Event, epoch idx.Epoch) bool {
	if id.Epoch() != epoch {
		return false
	}
	if h.dagProcessor.IsBuffered(id) || h.store.HasEvent(id) {
		return false
	}
	return true
}

// onlyInterestedEventsI filters a list of event IDs to only those the node is interested in.
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
