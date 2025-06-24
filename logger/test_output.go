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
	"bytes"
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"log/slog"
	"testing"
)

// SetTestMode sets default logger to log into the test output.
func SetTestMode(tb testing.TB) {
	log.SetDefault(log.NewLogger(&testLogHandler{tb, nil}))
}

type testLogHandler struct {
	tb    testing.TB
	attrs []slog.Attr
}

func (t testLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (t testLogHandler) Handle(ctx context.Context, r slog.Record) error {
	buf := &bytes.Buffer{}
	lvl := log.LevelAlignedString(r.Level)

	var attrs []slog.Attr
	r.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})
	attrs = append(attrs, t.attrs...)

	if _, err := fmt.Fprintf(buf, "%s %s", lvl, r.Message); err != nil {
		return err
	}
	for _, attr := range attrs {
		if _, err := fmt.Fprintf(buf, " %s=%s", attr.Key, string(log.FormatSlogValue(attr.Value, nil))); err != nil {
			return err
		}
	}
	t.tb.Log(buf.String())
	return nil
}

func (t testLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &testLogHandler{
		tb:    t.tb,
		attrs: append(t.attrs, attrs...),
	}
}

func (t testLogHandler) WithGroup(name string) slog.Handler {
	return t // ignored
}
