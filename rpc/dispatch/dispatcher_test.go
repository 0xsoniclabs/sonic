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

package dispatch

import (
	"context"
	"encoding/json"
	"errors"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func newTestDispatcher(t *testing.T, cfg Config) *Dispatcher {
	t.Helper()
	d := New(cfg, NewMetrics(nil))
	t.Cleanup(func() { d.Stop(time.Second) })
	return d
}

// requireNoModuleGoroutines fails if a goroutine started by the dispatcher is
// still alive after a second.
func requireNoModuleGoroutines(t *testing.T) {
	t.Helper()
	alive := func() string {
		buf := make([]byte, 1<<20)
		buf = buf[:runtime.Stack(buf, true)]
		for _, g := range strings.Split(string(buf), "\n\n") {
			if strings.Contains(g, "testing.tRunner") || strings.Contains(g, "runtime.Stack") {
				continue
			}
			if strings.Contains(g, "rpc/dispatch.") {
				return g
			}
		}
		return ""
	}
	deadline := time.Now().Add(time.Second)
	for alive() != "" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	require.Empty(t, alive(), "goroutine left")
}

func counter(c prometheus.Counter) float64 {
	var m dto.Metric
	_ = c.Write(&m)
	return m.GetCounter().GetValue()
}

func raw(s string) json.RawMessage {
	if s == "" {
		return nil
	}
	return json.RawMessage(s)
}

func TestDispatcher_UnknownMethod(t *testing.T) {
	d := newTestDispatcher(t, DefaultConfig())
	res, err := d.Serve(context.Background(), "eth_nope", nil)
	require.Nil(t, res)
	require.Equal(t, CodeMethodNotFound, err.Code)
	require.Equal(t, "the method eth_nope does not exist/is not available", err.Message)
	// Unknown names are counted under "other": the label set is bounded.
	require.EqualValues(t, 1, counter(d.metrics.Requests.WithLabelValues("other", ResultNotFound)))
}

func TestDispatcher_Stages_ParameterErrorsNeverReachTheHandler(t *testing.T) {
	d := newTestDispatcher(t, DefaultConfig())
	var called atomic.Int32
	Add2(d, "t_add", func(_ context.Context, addr common.Address, n *hexutil.Uint64) (string, error) {
		called.Add(1)
		return "ok", nil
	}, "address", "count?")

	tests := map[string]struct {
		params  string
		code    int
		message string
	}{
		"not an array":     {`{"a":1}`, CodeInvalidParams, "non-array args"},
		"missing required": {`[]`, CodeInvalidParams, "missing value for required argument 0"},
		"too many":         {`["0x0000000000000000000000000000000000000001","0x1",3]`, CodeInvalidParams, "too many arguments, want at most 2"},
		"bad first":        {`["zz"]`, CodeInvalidParams, "parameter 0 (address): "},
		"bad second":       {`["0x0000000000000000000000000000000000000001","x"]`, CodeInvalidParams, "parameter 1 (count): "},
		// A decoding error of a parameter that is there beats a count error.
		"bad before too many": {`["zz",1,2]`, CodeInvalidParams, "parameter 0 (address): "},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			res, err := d.Serve(context.Background(), "t_add", raw(test.params))
			require.Nil(t, res)
			require.NotNil(t, err)
			require.Equal(t, test.code, err.Code)
			require.Contains(t, err.Message, test.message)
		})
	}
	require.Zero(t, called.Load())

	for _, ok := range []string{
		`["0x0000000000000000000000000000000000000001"]`,
		`["0x0000000000000000000000000000000000000001", null]`,
		`["0x0000000000000000000000000000000000000001", "0x7"]`,
	} {
		res, err := d.Serve(context.Background(), "t_add", raw(ok))
		require.Nil(t, err)
		require.JSONEq(t, `"ok"`, string(res))
	}
	require.EqualValues(t, 3, called.Load())
}

func TestDispatcher_NoParams(t *testing.T) {
	d := newTestDispatcher(t, DefaultConfig())
	Add0(d, "t_zero", func(context.Context) (int, error) { return 7, nil })
	for _, params := range []string{"", "null", "[]"} {
		res, err := d.Serve(context.Background(), "t_zero", raw(params))
		require.Nil(t, err, params)
		require.JSONEq(t, `7`, string(res))
	}
	_, err := d.Serve(context.Background(), "t_zero", raw(`[1]`))
	require.Equal(t, "too many arguments, want at most 0", err.Message)
}

func TestDispatcher_HandlerErrors(t *testing.T) {
	d := newTestDispatcher(t, DefaultConfig())
	Add0(d, "t_plain", func(context.Context) (int, error) { return 0, errors.New("boom") })
	Add0(d, "t_coded", func(context.Context) (int, error) { return 0, &Error{Code: 3, Message: "revert", Data: "0x01"} })
	Add0(d, "t_wrapped", func(context.Context) (int, error) {
		return 0, wrapErr{&Error{Code: 3, Message: "inner"}}
	})
	Add0(d, "t_nilptr", func(context.Context) (*int, error) { return nil, nil })

	_, err := d.Serve(context.Background(), "t_plain", nil)
	require.Equal(t, &Error{Code: CodeDefault, Message: "boom"}, err)

	_, err = d.Serve(context.Background(), "t_coded", nil)
	require.Equal(t, &Error{Code: 3, Message: "revert", Data: "0x01"}, err)

	// An error that wraps a coded error is a plain error, as in go-ethereum.
	_, err = d.Serve(context.Background(), "t_wrapped", nil)
	require.Equal(t, CodeDefault, err.Code)

	res, err := d.Serve(context.Background(), "t_nilptr", nil)
	require.Nil(t, err)
	require.JSONEq(t, `null`, string(res))
}

type wrapErr struct{ error }

func (w wrapErr) Unwrap() error { return w.error }

func TestDispatcher_PanicIsRecoveredCountedAndLoggedOnce(t *testing.T) {
	d := newTestDispatcher(t, DefaultConfig())
	Add0(d, "t_panic", func(context.Context) (int, error) { panic("kaboom") })
	Add0(d, "t_nilmap", func(context.Context) (int, error) {
		var m map[string]int
		m["x"] = 1
		return 0, nil
	})

	for i := 1; i <= 2; i++ {
		_, err := d.Serve(context.Background(), "t_panic", nil)
		require.Equal(t, &Error{Code: CodeInternal, Message: "method handler crashed"}, err)
		require.EqualValues(t, i, counter(d.metrics.Panics.WithLabelValues("t_panic")))
	}
	_, err := d.Serve(context.Background(), "t_nilmap", nil)
	require.Equal(t, CodeInternal, err.Code)
	require.EqualValues(t, 1, counter(d.metrics.Panics.WithLabelValues("t_nilmap")))
	require.EqualValues(t, 3, counter(d.metrics.Requests.WithLabelValues("t_panic", ResultPanic))+
		counter(d.metrics.Requests.WithLabelValues("t_nilmap", ResultPanic)))
}

func TestDispatcher_DeadlineIsAttachedToEveryRequest(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Timeout = 50 * time.Millisecond
	d := newTestDispatcher(t, cfg)
	Add0(d, "t_slow", func(ctx context.Context) (int, error) {
		<-ctx.Done()
		return 0, ctx.Err()
	})
	start := time.Now()
	_, err := d.Serve(context.Background(), "t_slow", nil)
	require.Less(t, time.Since(start), 2*time.Second)
	require.Equal(t, "context deadline exceeded", err.Message)
	require.EqualValues(t, 1, counter(d.metrics.Requests.WithLabelValues("t_slow", ResultTimeout)))
}

func TestDispatcher_CancellationOfTheCaller(t *testing.T) {
	d := newTestDispatcher(t, DefaultConfig())
	Add0(d, "t_slow", func(ctx context.Context) (int, error) {
		<-ctx.Done()
		return 0, ctx.Err()
	})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	_, err := d.Serve(ctx, "t_slow", nil)
	require.Equal(t, "context canceled", err.Message)
	require.EqualValues(t, 1, counter(d.metrics.Requests.WithLabelValues("t_slow", ResultCancelled)))
}

func TestDispatcher_MetricsPerMethod(t *testing.T) {
	d := newTestDispatcher(t, DefaultConfig())
	Add0(d, "t_ok", func(context.Context) (string, error) { return "abc", nil })
	Add0(d, "t_err", func(context.Context) (string, error) { return "", errors.New("x") })
	_, _ = d.Serve(context.Background(), "t_ok", nil)
	_, _ = d.Serve(context.Background(), "t_ok", nil)
	_, _ = d.Serve(context.Background(), "t_err", nil)
	require.EqualValues(t, 2, counter(d.metrics.Requests.WithLabelValues("t_ok", ResultOK)))
	require.EqualValues(t, 1, counter(d.metrics.Requests.WithLabelValues("t_err", ResultError)))
}

func TestDispatcher_DuplicateRegistrationPanics(t *testing.T) {
	d := newTestDispatcher(t, DefaultConfig())
	Add0(d, "t_dup", func(context.Context) (int, error) { return 0, nil })
	require.Panics(t, func() {
		Add0(d, "t_dup", func(context.Context) (int, error) { return 0, nil })
	})
}

func TestDispatcher_ParameterNameCountMustMatch(t *testing.T) {
	d := newTestDispatcher(t, DefaultConfig())
	require.Panics(t, func() {
		Add2(d, "t_bad", func(context.Context, int, int) (int, error) { return 0, nil }, "only-one")
	})
	require.Panics(t, func() {
		Add2(d, "t_bad2", func(context.Context, int, int) (int, error) { return 0, nil }, "a?", "b")
	})
}

func TestGroup_PanickingChildBecomesAnErrorAndCancelsSiblings(t *testing.T) {
	d := newTestDispatcher(t, DefaultConfig())
	Add0(d, "t_group", func(ctx context.Context) (int, error) {
		g, gctx := d.Group(ctx)
		g.Go(func(context.Context) error { panic("child") })
		g.Go(func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		})
		_ = gctx
		return 0, g.Wait()
	})
	_, err := d.Serve(context.Background(), "t_group", nil)
	require.Equal(t, CodeInternal, err.Code)
	require.Equal(t, "method handler crashed", err.Message)
	require.EqualValues(t, 1, counter(d.metrics.Panics.WithLabelValues("t_group")))
	requireNoModuleGoroutines(t)
}

func TestGroup_WaitsForAllChildren(t *testing.T) {
	d := newTestDispatcher(t, DefaultConfig())
	var done atomic.Int32
	Add0(d, "t_group", func(ctx context.Context) (int, error) {
		g, _ := d.Group(ctx)
		for range 5 {
			g.Go(func(context.Context) error {
				time.Sleep(10 * time.Millisecond)
				done.Add(1)
				return nil
			})
		}
		err := g.Wait()
		return int(done.Load()), err
	})
	res, err := d.Serve(context.Background(), "t_group", nil)
	require.Nil(t, err)
	require.JSONEq(t, `5`, string(res))
}

func TestLoop_RestartsAfterPanicWithBackoffAndCounts(t *testing.T) {
	d := newTestDispatcher(t, DefaultConfig())
	var runs atomic.Int32
	require.True(t, d.Loop("test", func(ctx context.Context) {
		if runs.Add(1) < 3 {
			panic("loop")
		}
		<-ctx.Done()
	}))
	require.Eventually(t, func() bool { return runs.Load() == 3 }, 2*time.Second, time.Millisecond)
	require.EqualValues(t, 2, counter(d.metrics.LoopRestarts.WithLabelValues("test")))
	require.EqualValues(t, 2, counter(d.metrics.Panics.WithLabelValues("loop:test")))
}

func TestDispatcher_StopDrainsRequestsAndJoinsGoroutines(t *testing.T) {
	d := New(DefaultConfig(), NewMetrics(nil))

	started := make(chan struct{})
	Add0(d, "t_wait", func(ctx context.Context) (string, error) {
		close(started)
		time.Sleep(50 * time.Millisecond) // finishes within the grace period
		return "drained", nil
	})
	d.Loop("idle", func(ctx context.Context) { <-ctx.Done() })
	d.Go("task", func(ctx context.Context) { <-ctx.Done() })

	result := make(chan json.RawMessage, 1)
	go func() {
		res, _ := d.Serve(context.Background(), "t_wait", nil)
		result <- res
	}()
	<-started
	d.Stop(2 * time.Second)

	require.JSONEq(t, `"drained"`, string(<-result))
	// New requests are refused and nothing new may start.
	_, err := d.Serve(context.Background(), "t_wait", nil)
	require.Equal(t, "server is shutting down", err.Message)
	require.False(t, d.Go("late", func(context.Context) {}))
	require.False(t, d.Loop("late", func(context.Context) {}))
	requireNoModuleGoroutines(t)
}

func TestDispatcher_NoRequestRunsAfterStopReturns(t *testing.T) {
	for range 200 {
		d := New(DefaultConfig(), NewMetrics(nil))
		var stopped atomic.Bool
		var late atomic.Int32
		Add0(d, "t_work", func(context.Context) (int, error) {
			if stopped.Load() {
				late.Add(1)
			}
			return 0, nil
		})
		var wg sync.WaitGroup
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for !stopped.Load() {
					_, _ = d.Serve(context.Background(), "t_work", nil)
					d.Go("t_task", func(context.Context) {
						if stopped.Load() {
							late.Add(1)
						}
					})
				}
			}()
		}
		d.Stop(time.Second)
		stopped.Store(true)
		wg.Wait()
		require.Zero(t, late.Load(), "work admitted after Stop returned")
	}
}

func TestDispatcher_StopCancelsRequestsAfterTheGracePeriod(t *testing.T) {
	d := New(DefaultConfig(), NewMetrics(nil))
	started := make(chan struct{})
	Add0(d, "t_hang", func(ctx context.Context) (string, error) {
		close(started)
		<-ctx.Done()
		return "", ctx.Err()
	})
	done := make(chan *Error, 1)
	go func() {
		_, err := d.Serve(context.Background(), "t_hang", nil)
		done <- err
	}()
	<-started
	start := time.Now()
	d.Stop(50 * time.Millisecond)
	require.Less(t, time.Since(start), 2*time.Second)
	require.Equal(t, "context canceled", (<-done).Message)
}
