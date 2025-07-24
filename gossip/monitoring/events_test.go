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
	"testing"
	"time"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestNewEventMonitor_CreatesAnEventMonitor(t *testing.T) {
	// Checks that eventMonitor implements the EventMonitor interface.
	var monitor EventMonitor = NewEventMonitor()

	// smoke test for the monitor
	monitor.OnIncomingEvent(hash.Event{})
	monitor.OnOutgoingEvent(hash.Event{})
}

func TestEventMonitor_TracksForwardingTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	incoming := NewMockcounterMetric(ctrl)
	outgoing := NewMockcounterMetric(ctrl)
	latency := NewMockvalueMetric(ctrl)

	delay := time.Millisecond * 100

	incoming.EXPECT().Inc(int64(1))
	outgoing.EXPECT().Inc(int64(1))

	latency.EXPECT().Update(gomock.Any()).Do(func(value int64) {
		require.GreaterOrEqual(t, value, int64(delay.Milliseconds()), "Forwarding time should be non-negative")
		require.LessOrEqual(t, value, int64(2*delay.Milliseconds()), "Forwarding time should be within acceptable range")
	})

	monitor := newEventMonitor(incoming, outgoing, latency)

	eventId := hash.Event{0x01, 0x02, 0x03}
	monitor.OnIncomingEvent(eventId)
	time.Sleep(delay)
	monitor.OnOutgoingEvent(eventId)
}

func TestEventMonitor_ForwardingTimeIsOnlyReportedForFirstOutgoingEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	incoming := NewMockcounterMetric(ctrl)
	outgoing := NewMockcounterMetric(ctrl)
	latency := NewMockvalueMetric(ctrl)

	incoming.EXPECT().Inc(int64(1))
	outgoing.EXPECT().Inc(int64(1)).Times(3)

	latency.EXPECT().Update(gomock.Any()).Times(1)

	monitor := newEventMonitor(incoming, outgoing, latency)

	eventId := hash.Event{0x01, 0x02, 0x03}
	monitor.OnIncomingEvent(eventId)
	monitor.OnOutgoingEvent(eventId)
	monitor.OnOutgoingEvent(eventId)
	monitor.OnOutgoingEvent(eventId)
}

func TestEventMonitor_ReportsOfUnknownOutgoingEventsAreIgnoredInLatencyMonitoring(t *testing.T) {
	ctrl := gomock.NewController(t)
	incoming := NewMockcounterMetric(ctrl)
	outgoing := NewMockcounterMetric(ctrl)
	latency := NewMockvalueMetric(ctrl)

	outgoing.EXPECT().Inc(int64(1))

	monitor := newEventMonitor(incoming, outgoing, latency)
	monitor.OnOutgoingEvent(hash.Event{}) // Does not cause any metric updates
}

func TestEventMonitor_MultipleIncomingReportsAreIgnoredInLatencyMonitoring(t *testing.T) {
	ctrl := gomock.NewController(t)
	incoming := NewMockcounterMetric(ctrl)
	outgoing := NewMockcounterMetric(ctrl)
	latency := NewMockvalueMetric(ctrl)

	delay := time.Millisecond * 100

	incoming.EXPECT().Inc(int64(1)).Times(2)
	outgoing.EXPECT().Inc(int64(1))

	latency.EXPECT().Update(gomock.Any()).Do(func(value int64) {
		require.GreaterOrEqual(t, value, int64(delay.Milliseconds()), "Forwarding time should be non-negative")
		require.LessOrEqual(t, value, int64(2*delay.Milliseconds()), "Forwarding time should be within acceptable range")
	})

	monitor := newEventMonitor(incoming, outgoing, latency)

	eventId := hash.Event{0x01, 0x02, 0x03}
	monitor.OnIncomingEvent(eventId)
	time.Sleep(delay)
	monitor.OnIncomingEvent(eventId) // does not restart the timer
	monitor.OnOutgoingEvent(eventId)
}
