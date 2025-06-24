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

package logger

import (
	"time"
)

// Periodic is the same as logger.Instance, but writes only once in a period
type Periodic struct {
	Instance
	prevLogTime time.Time
}

// Info is timed log.Info
func (l *Periodic) Info(period time.Duration, msg string, ctx ...interface{}) {
	if time.Since(l.prevLogTime) > period {
		l.Log.Info(msg, ctx...)
		l.prevLogTime = time.Now()
	}
}

// Warn is timed log.Warn
func (l *Periodic) Warn(period time.Duration, msg string, ctx ...interface{}) {
	if time.Since(l.prevLogTime) > period {
		l.Log.Warn(msg, ctx...)
		l.prevLogTime = time.Now()
	}
}

// Error is timed log.Error
func (l *Periodic) Error(period time.Duration, msg string, ctx ...interface{}) {
	if time.Since(l.prevLogTime) > period {
		l.Log.Error(msg, ctx...)
		l.prevLogTime = time.Now()
	}
}

// Debug is timed log.Debug
func (l *Periodic) Debug(period time.Duration, msg string, ctx ...interface{}) {
	if time.Since(l.prevLogTime) > period {
		l.Log.Debug(msg, ctx...)
		l.prevLogTime = time.Now()
	}
}

// Trace is timed log.Trace
func (l *Periodic) Trace(period time.Duration, msg string, ctx ...interface{}) {
	if time.Since(l.prevLogTime) > period {
		l.Log.Trace(msg, ctx...)
		l.prevLogTime = time.Now()
	}
}
