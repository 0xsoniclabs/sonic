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

package proclogger

import (
	"testing"
	"time"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/logger"
)

func TestEventConnectionStarted_Emitted(t *testing.T) {
	logger.SetTestMode(t)
	l := NewLogger()

	me := &inter.MutableEventPayload{}
	me.SetEpoch(1)
	me.SetCreator(idx.ValidatorID(1))
	me.SetSeq(1)
	me.SetLamport(1)
	me.SetCreationTime(inter.Timestamp(time.Now().UnixNano()))
	e := me.Build()

	done := l.EventConnectionStarted(e, true)
	if done == nil {
		t.Fatal("expected non-nil cleanup function")
	}

	// noSummary should be true during processing.
	if !l.noSummary {
		t.Error("noSummary should be true during processing")
	}
	if !l.emitting {
		t.Error("emitting should be true")
	}

	done()

	// After done, noSummary and emitting should be false.
	if l.noSummary {
		t.Error("noSummary should be false after done")
	}
	if l.emitting {
		t.Error("emitting should be false after done")
	}

	// dagSum should have been incremented.
	// (summary may have reset it if nextLogging was in the past)
}

func TestEventConnectionStarted_NotEmitted(t *testing.T) {
	logger.SetTestMode(t)
	l := NewLogger()

	me := &inter.MutableEventPayload{}
	me.SetEpoch(1)
	me.SetCreator(idx.ValidatorID(1))
	me.SetSeq(1)
	me.SetLamport(1)
	me.SetCreationTime(inter.Timestamp(time.Now().UnixNano()))
	e := me.Build()

	done := l.EventConnectionStarted(e, false)
	if l.emitting {
		t.Error("emitting should be false for non-emitted event")
	}
	done()
}

func TestEventConnectionStarted_MultipleCalls(t *testing.T) {
	logger.SetTestMode(t)
	l := NewLogger()
	l.nextLogging = time.Now().Add(time.Hour) // prevent summary from firing

	for i := 0; i < 5; i++ {
		me := &inter.MutableEventPayload{}
		me.SetEpoch(1)
		me.SetCreator(idx.ValidatorID(1))
		me.SetSeq(idx.Event(i + 1))
		me.SetLamport(idx.Lamport(i + 1))
		me.SetCreationTime(inter.Timestamp(time.Now().UnixNano()))
		e := me.Build()

		done := l.EventConnectionStarted(e, false)
		done()
	}

	// dagSum should have accumulated 5 connected events, but summary may have reset.
	// Just ensure no panics.
}

func TestSummary_WithBothDagAndLlr(t *testing.T) {
	logger.SetTestMode(t)
	l := NewLogger()
	l.noSummary = false
	l.nextLogging = time.Time{} // zero time, so summary always fires

	l.dagSum.connected = 3
	l.dagSum.totalProcessing = 500 * time.Millisecond
	l.lastID = hash.FakeEvent()
	l.lastEventTime = inter.Timestamp(time.Now().Add(-10 * time.Second).UnixNano())

	l.llrSum.bvs = 10
	l.llrSum.brs = 5
	l.llrSum.evs = 2
	l.llrSum.ers = 1
	l.lastLlrTime = inter.Timestamp(time.Now().UnixNano())
	l.lastEpoch = 5
	l.lastBlock = 100

	l.summary(time.Now())

	// Both sums should be reset.
	if l.dagSum.connected != 0 {
		t.Error("dagSum should be reset")
	}
	if l.llrSum.bvs != 0 {
		t.Error("llrSum should be reset")
	}
}

func TestSummary_NextLoggingUpdated(t *testing.T) {
	logger.SetTestMode(t)
	l := NewLogger()
	l.nextLogging = time.Time{}
	l.dagSum.connected = 1
	l.lastEventTime = inter.Timestamp(time.Now().UnixNano())

	now := time.Now()
	l.summary(now)

	if l.nextLogging.Before(now) {
		t.Error("nextLogging should be updated to future")
	}
}

func TestSummary_EmptyDagAndLlr(t *testing.T) {
	logger.SetTestMode(t)
	l := NewLogger()
	l.nextLogging = time.Time{}

	// Both sums are zero - should be a no-op without panic.
	l.summary(time.Now())
}
