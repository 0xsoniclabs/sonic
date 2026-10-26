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
	"bytes"
	"context"
	"errors"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// eventually waits for the condition, failing the test after ten seconds.
func eventually(t *testing.T, condition func() bool, description string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", description)
		}
		time.Sleep(time.Millisecond)
	}
}

// requireNoLeftGoroutines checks that the goroutines return to the number
// counted before the test started its work.
func requireNoLeftGoroutines(t *testing.T, before int) {
	t.Helper()
	eventually(t, func() bool { return runtime.NumGoroutine() <= before }, "the goroutines of the dispatcher to end")
}

func TestGroup_APanickingChildCancelsItsSiblings(t *testing.T) {
	dispatcher := newTestDispatcher(t)
	group, ctx := dispatcher.Group(context.Background())
	siblingCancelled := make(chan struct{})
	group.Go(ctx, func(ctx context.Context) error {
		<-ctx.Done()
		close(siblingCancelled)
		return ctx.Err()
	})
	group.Go(ctx, func(context.Context) error { panic("child") })
	err := group.Wait()
	require.ErrorIs(t, err, ErrPanic)
	select {
	case <-siblingCancelled:
	default:
		t.Fatal("Wait returned before the sibling ended")
	}
	require.Equal(t, 1.0, dispatcher.scrape(t, "rpc_panics_total"))
}

func TestGroup_WaitReturnsTheFirstError(t *testing.T) {
	dispatcher := newTestDispatcher(t)
	group, ctx := dispatcher.Group(context.Background())
	first := errors.New("first")
	group.Go(ctx, func(context.Context) error { return first })
	group.Go(ctx, func(ctx context.Context) error {
		<-ctx.Done()
		return errors.New("second")
	})
	require.Equal(t, first, group.Wait())
}

func TestGroup_ChildrenOfARequestAreCountedUnderItsMethod(t *testing.T) {
	dispatcher := newTestDispatcher(t)
	dispatcher.Add("test_children", func(ctx context.Context, _ Arguments) (any, error) {
		group, ctx := dispatcher.Group(ctx)
		group.Go(ctx, func(context.Context) error { panic("child") })
		return nil, group.Wait()
	})
	_, err := serve(t, dispatcher, "test_children", "[]")
	require.Equal(t, ErrPanic, err)
	require.Equal(t, 1.0, dispatcher.scrape(t, "rpc_panics_total", "method", "test_children"))
}

func TestLoop_RestartsAfterAPanicAndRunsAgain(t *testing.T) {
	dispatcher := newTestDispatcher(t)
	var runs atomic.Int32
	working := make(chan struct{})
	dispatcher.Loop("test", func(ctx context.Context) error {
		if runs.Add(1) == 1 {
			panic("loop")
		}
		close(working)
		<-ctx.Done()
		return nil
	})
	select {
	case <-working:
	case <-time.After(10 * time.Second):
		t.Fatal("the loop did not run again after its panic")
	}
	require.Equal(t, 1.0, dispatcher.scrape(t, "rpc_loop_restarts_total", "loop", "test"))
	require.Equal(t, 1.0, dispatcher.scrape(t, "rpc_panics_total", "method", "test"))
}

func TestLoop_ThatReturnsIsRestarted(t *testing.T) {
	dispatcher := newTestDispatcher(t)
	var runs atomic.Int32
	dispatcher.Loop("test", func(ctx context.Context) error {
		if runs.Add(1) < 3 {
			return errors.New("failed")
		}
		<-ctx.Done()
		return nil
	})
	eventually(t, func() bool { return runs.Load() == 3 }, "the third run of the loop")
	require.Equal(t, 2.0, dispatcher.scrape(t, "rpc_loop_restarts_total", "loop", "test"))
}

func TestStop_WaitsForTheRequestsInFlight(t *testing.T) {
	dispatcher := newTestDispatcher(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	dispatcher.Add("test_slow", func(context.Context, Arguments) (any, error) {
		close(entered)
		<-release
		return "done", nil
	})
	go func() {
		defer close(finished)
		result, err := serve(t, dispatcher, "test_slow", "[]")
		require.Nil(t, err)
		require.Equal(t, `"done"`, result)
	}()
	<-entered
	stopped := make(chan struct{})
	go func() {
		dispatcher.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
		t.Fatal("Stop returned while a request was in flight")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	<-stopped
	<-finished
}

func TestStop_CancelsAfterTheGracePeriodAndWaits(t *testing.T) {
	dispatcher := newTestDispatcher(t, func(config *Config) { config.RequestTimeout = 100 * time.Millisecond })
	entered := make(chan struct{})
	var cancelledAt atomic.Int64
	dispatcher.Go(context.Background(), "test_background", func(ctx context.Context) {
		close(entered)
		<-ctx.Done()
		cancelledAt.Store(time.Now().UnixNano())
		// Work that ignores cancellation for a while: Stop must wait for it.
		time.Sleep(100 * time.Millisecond)
	})
	<-entered
	start := time.Now()
	dispatcher.Stop()
	require.NotZero(t, cancelledAt.Load(), "Stop returned before cancelling the work")
	require.GreaterOrEqual(t, time.Since(start), 200*time.Millisecond, "Stop did not wait for the work after cancelling it")
}

func TestStop_LogsTheWorkItWaitsFor(t *testing.T) {
	var output bytes.Buffer
	previous := log.Root()
	log.SetDefault(log.NewLogger(slog.NewTextHandler(&output, nil)))
	defer log.SetDefault(previous)

	dispatcher := newTestDispatcher(t, func(config *Config) { config.RequestTimeout = 10 * time.Millisecond })
	entered := make(chan struct{})
	dispatcher.Go(context.Background(), "eth_estimateGas", func(context.Context) {
		close(entered)
		time.Sleep(50 * time.Millisecond)
	})
	<-entered
	dispatcher.Stop()
	require.Contains(t, output.String(), "RPC stop waits for a request that ignores cancellation")
	require.Contains(t, output.String(), "method=eth_estimateGas")
	require.Contains(t, output.String(), "age=")
}

func TestStop_RefusesNewWork(t *testing.T) {
	dispatcher := newTestDispatcher(t)
	dispatcher.Add("test_echo", echo)
	dispatcher.Stop()

	_, err := serve(t, dispatcher, "test_echo", "[]")
	require.Equal(t, ErrStopping, err)
	require.False(t, dispatcher.Go(context.Background(), "test", func(context.Context) {}))
	require.False(t, dispatcher.Loop("test", func(context.Context) error { return nil }))
	require.False(t, dispatcher.Run("test", func(context.Context) {}))
	require.True(t, dispatcher.Stopping())
	require.Equal(t, 1.0, dispatcher.scrape(t, "rpc_requests_total", "method", "test_echo", "result", ResultRejected))
}

func TestStop_LeavesNoGoroutineBehind(t *testing.T) {
	before := runtime.NumGoroutine()
	dispatcher := newTestDispatcher(t)
	dispatcher.Add("test_echo", echo)
	for range 10 {
		dispatcher.Go(context.Background(), "test", func(ctx context.Context) { <-ctx.Done() })
	}
	dispatcher.Loop("test", func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	})
	dispatcher.Run("test", func(ctx context.Context) { <-ctx.Done() })
	var callers sync.WaitGroup
	for range 10 {
		callers.Go(func() { _, _ = serve(t, dispatcher, "test_echo", "[]") })
	}
	callers.Wait()
	dispatcher.Stop()
	requireNoLeftGoroutines(t, before)
}

// TestStop_AdmissionRacesStop holds P7: no work starts after Stop returned,
// however requests, goroutines and Stop interleave.
func TestStop_AdmissionRacesStop(t *testing.T) {
	for range 20 {
		dispatcher := newTestDispatcher(t)
		var stopped atomic.Bool
		var lateWork atomic.Int32
		work := func(context.Context) {
			if stopped.Load() {
				lateWork.Add(1)
			}
		}
		dispatcher.Add("test_work", func(ctx context.Context, _ Arguments) (any, error) {
			work(ctx)
			return nil, nil
		})
		var callers sync.WaitGroup
		for range 8 {
			callers.Go(func() {
				for range 50 {
					_, _ = serve(t, dispatcher, "test_work", "[]")
					dispatcher.Go(context.Background(), "test", work)
				}
			})
		}
		dispatcher.Stop()
		stopped.Store(true)
		callers.Wait()
		require.Zero(t, lateWork.Load(), "work started after Stop returned")
	}
}

func TestRun_RecoversAPanic(t *testing.T) {
	dispatcher := newTestDispatcher(t)
	done := make(chan struct{})
	dispatcher.Run("test", func(context.Context) {
		defer close(done)
		panic("run")
	})
	<-done
	eventually(t, func() bool { return dispatcher.scrape(t, "rpc_panics_total", "method", "test") == 1 }, "the panic to be counted")
}
