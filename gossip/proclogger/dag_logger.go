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

package proclogger

import (
	"time"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/utils"
)

func NewLogger() *Logger {
	return &Logger{
		Instance: logger.New(),
	}
}

// EventConnectionStarted starts the event logging
// Not safe for concurrent use
func (l *Logger) EventConnectionStarted(e inter.EventPayloadI, emitted bool) func() {
	l.dagSum.connected++

	start := time.Now()
	l.emitting = emitted
	l.noSummary = true // print summary after the whole event is processed
	l.lastID = e.ID()
	l.lastEventTime = e.CreationTime()

	return func() {
		now := time.Now()
		// logging for the individual item
		msg := "New event"
		logType := l.Log.Debug
		if emitted {
			msg = "New event emitted"
			logType = l.Log.Info
		}
		logType(msg, "id", e.ID(), "parents", len(e.Parents()), "by", e.Creator(),
			"frame", e.Frame(), "txs", e.Transactions().Len(),
			"age", utils.PrettyDuration(now.Sub(e.CreationTime().Time())), "t", utils.PrettyDuration(now.Sub(start)))
		// logging for the summary
		l.dagSum.totalProcessing += now.Sub(start)
		l.emitting = false
		l.noSummary = false
		l.summary(now)
	}
}
