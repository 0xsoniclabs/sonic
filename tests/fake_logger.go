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

package tests

import (
	"context"
	"log/slog"
	"runtime/debug"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/log"
)

type fakeLogger struct {
	l log.Logger
	t *testing.T

	expectedErrorsLock sync.Mutex          // lock to protect the expected errors slice
	expectedErrors     map[string]struct{} // expected errors in the logs of the network
}

func (f *fakeLogger) With(ctx ...interface{}) log.Logger {
	return f.l.With(ctx...)
}

func (f *fakeLogger) New(ctx ...interface{}) log.Logger {
	return f.l.With(ctx...)
}

func (f *fakeLogger) Log(level slog.Level, msg string, ctx ...interface{}) {
	if (level == log.LevelCrit || level == log.LevelError) && !f.isExpectedError(msg) {
		f.t.Errorf("Log at level '%s': '%s' with context '%v'\n stack: %v", log.LevelString(level), msg, ctx, debug.Stack())
	}
	f.l.Log(level, msg, ctx...)
}

func (f *fakeLogger) Trace(msg string, ctx ...interface{}) {
	f.l.Trace(msg, ctx...)
}

func (f *fakeLogger) Debug(msg string, ctx ...interface{}) {
	f.l.Debug(msg, ctx...)
}

func (f *fakeLogger) Info(msg string, ctx ...interface{}) {
	f.l.Info(msg, ctx...)
}

func (f *fakeLogger) Warn(msg string, ctx ...interface{}) {
	f.l.Warn(msg, ctx...)
}

func (f *fakeLogger) Error(msg string, ctx ...interface{}) {
	if !f.isExpectedError(msg) {
		f.t.Errorf("Error '%s' - with context '%v'\n stack: %v", msg, ctx, debug.Stack())
	}
}

func (f *fakeLogger) Crit(msg string, ctx ...interface{}) {
	if !f.isExpectedError(msg) {
		f.t.Errorf("Critical error '%s' - with context '%v'\n stack: %v", msg, ctx, debug.Stack())
	}
}

func (f *fakeLogger) Write(level slog.Level, msg string, ctx ...interface{}) {
	if (level == log.LevelCrit || level == log.LevelError) && !f.isExpectedError(msg) {
		f.t.Errorf("Write at level '%v' %v with context '%v' \n stack: %v", log.LevelString(level), msg, ctx, debug.Stack())
	}
	f.l.Log(level, msg, ctx...)
}

func (f *fakeLogger) Enabled(ctx context.Context, level slog.Level) bool {
	return f.l.Enabled(ctx, level)
}

func (f *fakeLogger) Handler() slog.Handler {
	return f.l.Handler()
}

// addExpectedError adds an error message to the list of expected errors,
// which will not trigger a test failure when logged.
func (f *fakeLogger) addExpectedError(err string) {
	f.expectedErrorsLock.Lock()
	defer f.expectedErrorsLock.Unlock()
	if f.expectedErrors == nil {
		f.expectedErrors = make(map[string]struct{})
	}
	f.expectedErrors[err] = struct{}{}
}

// isExpectedError checks if the given error message is among the known expected messages.
func (f *fakeLogger) isExpectedError(err string) bool {
	f.expectedErrorsLock.Lock()
	defer f.expectedErrorsLock.Unlock()
	if f.expectedErrors == nil {
		return false
	}
	_, ok := f.expectedErrors[err]
	return ok
}
