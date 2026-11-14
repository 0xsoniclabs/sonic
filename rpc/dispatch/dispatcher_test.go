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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

type codeError struct{}

func (codeError) Error() string  { return "reverted" }
func (codeError) ErrorCode() int { return 3 }
func (codeError) ErrorData() any { return "0x01" }

type panicsOnDecode struct{}

func (*panicsOnDecode) UnmarshalJSON([]byte) error { panic("decode") }

type panicsOnEncode struct{}

func (panicsOnEncode) MarshalJSON() ([]byte, error) { panic("encode") }

type failsToEncode struct{}

func (failsToEncode) MarshalJSON() ([]byte, error) { return nil, errors.New("no form") }

func TestServe_AnswersEachStage(t *testing.T) {
	tests := map[string]struct {
		register func(*Dispatcher)
		params   string
		result   string
		error    *Error
	}{
		"result": {
			register: func(d *Dispatcher) { Add(d, "test_a", ok[struct{}]) },
			result:   `"ok"`,
		},
		"unknown method": {
			register: func(d *Dispatcher) {},
			error:    &Error{Code: CodeMethodNotFound, Message: "the method test_a does not exist/is not available"},
		},
		"invalid params": {
			register: func(d *Dispatcher) { Add(d, "test_a", ok[struct{}]) },
			params:   "[1]",
			error:    &Error{Code: CodeInvalidParams, Message: "too many arguments, want at most 0"},
		},
		"plain error": {
			register: func(d *Dispatcher) {
				Add(d, "test_a", func(context.Context, struct{}) (string, error) { return "", errors.New("no") })
			},
			error: &Error{Code: CodeDefault, Message: "no"},
		},
		"error with code and data": {
			register: func(d *Dispatcher) {
				Add(d, "test_a", func(context.Context, struct{}) (string, error) { return "", codeError{} })
			},
			error: &Error{Code: 3, Message: "reverted", Data: "0x01"},
		},
		"wrapped code is not the error's own": {
			register: func(d *Dispatcher) {
				Add(d, "test_a", func(context.Context, struct{}) (string, error) {
					return "", errors.Join(codeError{})
				})
			},
			error: &Error{Code: CodeDefault, Message: "reverted"},
		},
		"nil result": {
			register: func(d *Dispatcher) {
				Add(d, "test_a", func(context.Context, struct{}) (*string, error) { return nil, nil })
			},
			result: "null",
		},
		"encoded result unchanged": {
			register: func(d *Dispatcher) {
				Add(d, "test_a", func(context.Context, struct{}) (Encoded, error) { return Encoded(`{"a": 1}`), nil })
			},
			result: `{"a": 1}`,
		},
		"encoding error": {
			register: func(d *Dispatcher) {
				Add(d, "test_a", func(context.Context, struct{}) (failsToEncode, error) { return failsToEncode{}, nil })
			},
			error: &Error{Code: CodeInternal, Message: "json: error calling MarshalJSON for type *dispatch.failsToEncode: no form"},
		},
		"panic in handler": {
			register: func(d *Dispatcher) {
				Add(d, "test_a", func(context.Context, struct{}) (string, error) { panic("boom") })
			},
			error: errHandlerCrashed,
		},
		"panic in decoding": {
			register: func(d *Dispatcher) {
				type params struct{ Value panicsOnDecode }
				Add(d, "test_a", ok[params])
			},
			params: "[1]",
			error:  errHandlerCrashed,
		},
		"panic in encoding": {
			register: func(d *Dispatcher) {
				Add(d, "test_a", func(context.Context, struct{}) (panicsOnEncode, error) { return panicsOnEncode{}, nil })
			},
			error: errHandlerCrashed,
		},
		"panic in a child": {
			register: func(d *Dispatcher) {
				Add(d, "test_a", func(ctx context.Context, _ struct{}) (string, error) {
					group, _ := NewGroup(ctx)
					group.Go(func() error { panic("child") })
					return "", group.Wait()
				})
			},
			error: errHandlerCrashed,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			dispatcher := New(DefaultConfig(), prometheus.NewRegistry())
			test.register(dispatcher)
			response := dispatcher.Serve(context.Background(), "test_a", json.RawMessage(test.params))
			require.Equal(t, test.error, response.Error)
			require.Equal(t, test.result, string(response.Result))
		})
	}
}

func TestServe_AppliesTheRequestTimeout(t *testing.T) {
	dispatcher := New(Config{RequestTimeout: time.Minute}, prometheus.NewRegistry())
	var remaining time.Duration
	Add(dispatcher, "test_a", func(ctx context.Context, _ struct{}) (string, error) {
		deadline, _ := ctx.Deadline()
		remaining = time.Until(deadline)
		return "ok", nil
	})
	dispatcher.Serve(context.Background(), "test_a", nil)
	require.InDelta(t, time.Minute, remaining, float64(time.Second))
}

func TestServe_DecodesBeforeTheDeadlineStarts(t *testing.T) {
	dispatcher := New(Config{RequestTimeout: time.Nanosecond}, prometheus.NewRegistry())
	Add(dispatcher, "test_a", ok[mixedParams])
	response := dispatcher.Serve(context.Background(), "test_a", json.RawMessage(`[]`))
	require.Equal(t, "missing value for required argument 0", response.Error.Message)
}

func TestServe_CountsCallsPanicsAndUnknownMethods(t *testing.T) {
	registry := prometheus.NewRegistry()
	dispatcher := New(DefaultConfig(), registry)
	Add(dispatcher, "test_ok", ok[struct{}])
	Add(dispatcher, "test_panic", func(context.Context, struct{}) (string, error) { panic("boom") })

	before := scrape(t, registry, "rpc_requests_total", "method", "test_ok", "result", "success")
	dispatcher.Serve(context.Background(), "test_ok", nil)
	dispatcher.Serve(context.Background(), "test_panic", nil)
	dispatcher.Serve(context.Background(), "test_unknown", nil)

	require.Equal(t, before+1, scrape(t, registry, "rpc_requests_total", "method", "test_ok", "result", "success"))
	require.Equal(t, 1.0, scrape(t, registry, "rpc_requests_total", "method", "test_panic", "result", "error"))
	require.Equal(t, 1.0, scrape(t, registry, "rpc_panics_total", "method", "test_panic"))
	require.Equal(t, 1.0, scrape(t, registry, "rpc_requests_total", "method", unknownMethod, "result", "error"))
	require.Equal(t, 2.0, scrape(t, registry, "rpc_duration_seconds", "method", "test_ok")+
		scrape(t, registry, "rpc_duration_seconds", "method", "test_panic"))
	require.Equal(t, 0.0, scrape(t, registry, "rpc_inflight"))
}

func TestRegister_SharesTheCollectorsOfAnEarlierRegistration(t *testing.T) {
	registry := prometheus.NewRegistry()
	first := newMetrics(registry)
	second := newMetrics(registry)
	require.Same(t, first.inflight, second.inflight)
}

func TestStop_LetsCallsInFlightFinish(t *testing.T) {
	dispatcher := New(Config{RequestTimeout: time.Minute}, prometheus.NewRegistry())
	started, finish := make(chan struct{}), make(chan struct{})
	Add(dispatcher, "test_a", func(context.Context, struct{}) (string, error) {
		close(started)
		<-finish
		return "ok", nil
	})
	var response Response
	served := make(chan struct{})
	go func() {
		response = dispatcher.Serve(context.Background(), "test_a", nil)
		close(served)
	}()
	<-started
	stopped := make(chan struct{})
	go func() {
		dispatcher.Stop()
		close(stopped)
	}()
	require.Eventually(t, func() bool { return dispatcher.Serve(context.Background(), "test_a", nil).Error == errShuttingDown },
		time.Second, time.Millisecond)
	close(finish)
	<-served
	<-stopped
	require.Equal(t, `"ok"`, string(response.Result))
}

func TestStop_CancelsCallsThatOutliveTheTimeout(t *testing.T) {
	dispatcher := New(Config{RequestTimeout: 10 * time.Millisecond}, prometheus.NewRegistry())
	Add(dispatcher, "test_a", func(ctx context.Context, _ struct{}) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	})
	work := make(chan struct{})
	require.True(t, dispatcher.Go("test", func(ctx context.Context) {
		close(work)
		<-ctx.Done()
	}))
	<-work
	go dispatcher.Serve(context.Background(), "test_a", nil)
	dispatcher.Stop()
	require.False(t, dispatcher.Go("late", func(context.Context) {}))
}

func TestStop_LeavesNoGoroutineBehind(t *testing.T) {
	before := runtime.NumGoroutine()
	dispatcher := New(Config{RequestTimeout: 10 * time.Millisecond}, prometheus.NewRegistry())
	Add(dispatcher, "test_a", func(ctx context.Context, _ struct{}) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	})
	for range 10 {
		go dispatcher.Serve(context.Background(), "test_a", nil)
		dispatcher.Go("test", func(ctx context.Context) { <-ctx.Done() })
		dispatcher.Loop("test", func(ctx context.Context) { <-ctx.Done() })
	}
	dispatcher.Stop()
	// Polled by hand: require.Eventually runs its condition in a goroutine.
	for deadline := time.Now().Add(time.Second); runtime.NumGoroutine() > before; time.Sleep(time.Millisecond) {
		require.True(t, time.Now().Before(deadline), "goroutines left after Stop")
	}
}

func TestStop_IsAtomicTowardsAdmission(t *testing.T) {
	for range 50 {
		dispatcher := New(Config{RequestTimeout: time.Millisecond}, prometheus.NewRegistry())
		Add(dispatcher, "test_a", ok[struct{}])
		var running atomic.Int32
		var wg sync.WaitGroup
		for range 4 {
			wg.Go(func() {
				dispatcher.Serve(context.Background(), "test_a", nil)
				dispatcher.Go("test", func(context.Context) {
					running.Add(1)
					defer running.Add(-1)
				})
			})
		}
		dispatcher.Stop()
		require.Zero(t, running.Load(), "work runs after Stop returned")
		wg.Wait()
	}
}

func TestStop_LogsTheWorkItWaitsFor(t *testing.T) {
	interval := stopLogInterval
	stopLogInterval = time.Millisecond
	t.Cleanup(func() { stopLogInterval = interval })

	dispatcher := New(Config{RequestTimeout: time.Millisecond}, prometheus.NewRegistry())
	release := make(chan struct{})
	dispatcher.Go("stuck", func(context.Context) { <-release })
	time.AfterFunc(20*time.Millisecond, func() { close(release) })
	dispatcher.Stop()
}

func TestGo_RecoversAPanicOfTheWork(t *testing.T) {
	dispatcher := New(DefaultConfig(), prometheus.NewRegistry())
	require.True(t, dispatcher.Go("test", func(context.Context) { panic("boom") }))
	dispatcher.Stop()
}

func TestLoop_RestartsAfterAPanic(t *testing.T) {
	first, maximum := loopFirstBackoff, loopMaxBackoff
	loopFirstBackoff, loopMaxBackoff = time.Millisecond, 2*time.Millisecond
	t.Cleanup(func() { loopFirstBackoff, loopMaxBackoff = first, maximum })

	registry := prometheus.NewRegistry()
	dispatcher := New(DefaultConfig(), registry)
	var runs atomic.Int32
	ran := make(chan struct{})
	dispatcher.Loop("test", func(ctx context.Context) {
		if runs.Add(1) <= 3 {
			panic("boom")
		}
		close(ran)
		<-ctx.Done()
	})
	<-ran
	require.Equal(t, 3.0, scrape(t, registry, "rpc_loop_restarts_total", "loop", "test"))
	dispatcher.Stop()
	require.False(t, dispatcher.Loop("late", func(context.Context) {}))
}

func TestGroup_CancelsSiblingsOfAFailingChild(t *testing.T) {
	group, ctx := NewGroup(context.Background())
	group.Go(func() error { return errors.New("failed") })
	group.Go(func() error {
		<-ctx.Done()
		return nil
	})
	require.EqualError(t, group.Wait(), "failed")
}

func TestToError_KeepsCodeAndData(t *testing.T) {
	require.Equal(t, &Error{Code: 3, Message: "reverted", Data: "0x01"}, ToError(codeError{}))
	original := &Error{Code: CodeInvalidParams, Message: "bad"}
	require.Equal(t, original, ToError(original))
}

func TestMethods_ListsTheRegisteredNames(t *testing.T) {
	dispatcher := newShapesDispatcher()
	require.Equal(t, []string{"test_hex", "test_identified", "test_mixed", "test_none", "test_slice"}, dispatcher.Methods())
	require.True(t, dispatcher.Serves("test_hex"))
	require.False(t, dispatcher.Serves("test_other"))
}

// scrape reads the value of a metric of a registry: a counter or gauge, or
// the sample count of a histogram. Labels are name and value pairs.
func scrape(t *testing.T, registry *prometheus.Registry, name string, labels ...string) float64 {
	t.Helper()
	families, err := registry.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			if hasLabels(metric, labels) {
				return metricValue(metric)
			}
		}
	}
	return 0
}

func hasLabels(metric *dto.Metric, labels []string) bool {
	want := map[string]string{}
	for i := 0; i+1 < len(labels); i += 2 {
		want[labels[i]] = labels[i+1]
	}
	for _, pair := range metric.GetLabel() {
		if value, named := want[pair.GetName()]; named && value != pair.GetValue() {
			return false
		}
	}
	return true
}

func metricValue(metric *dto.Metric) float64 {
	switch {
	case metric.Counter != nil:
		return metric.GetCounter().GetValue()
	case metric.Gauge != nil:
		return metric.GetGauge().GetValue()
	default:
		return float64(metric.GetHistogram().GetSampleCount())
	}
}
