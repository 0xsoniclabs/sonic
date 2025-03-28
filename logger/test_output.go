package logger

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/ethereum/go-ethereum/log"
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
