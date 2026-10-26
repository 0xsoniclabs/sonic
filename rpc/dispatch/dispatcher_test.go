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
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

// value reads a metric of a registry: the sum over the series whose labels
// include the given pairs, or the count of observations of a histogram.
func value(t *testing.T, reg *prometheus.Registry, name string, labels ...string) float64 {
	t.Helper()
	families, err := reg.Gather()
	require.NoError(t, err)
	var sum float64
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
	metrics:
		for _, m := range family.GetMetric() {
			for i := 0; i+1 < len(labels); i += 2 {
				found := false
				for _, l := range m.GetLabel() {
					found = found || (l.GetName() == labels[i] && l.GetValue() == labels[i+1])
				}
				if !found {
					continue metrics
				}
			}
			sum += m.GetCounter().GetValue() + m.GetGauge().GetValue() + float64(m.GetHistogram().GetSampleCount())
		}
	}
	return sum
}

// series counts the series of a metric.
func series(t *testing.T, reg *prometheus.Registry, name string) int {
	t.Helper()
	families, err := reg.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() == name {
			return len(family.GetMetric())
		}
	}
	return 0
}

func newTestDispatcher(t *testing.T, cfg Config) (*Dispatcher, *prometheus.Registry) {
	t.Helper()
	reg := prometheus.NewRegistry()
	d := New(cfg, NewMetrics(reg))
	t.Cleanup(func() { d.Stop(time.Second) })
	return d, reg
}

func serve(d *Dispatcher, method, params string) (json.RawMessage, ErrorObject, bool) {
	res, err := d.Serve(context.Background(), Call{Method: method, Params: json.RawMessage(params)})
	if err != nil {
		return nil, ToErrorObject(err), false
	}
	return res, ErrorObject{}, true
}

type twoParams struct {
	Name  string `rpc:"name"`
	Count *int   `rpc:"count"`
}

func echo(_ context.Context, p twoParams) (any, error) {
	count := -1
	if p.Count != nil {
		count = *p.Count
	}
	return fmt.Sprintf("%s:%d", p.Name, count), nil
}

func TestServe_DecodesAndCountsParameters(t *testing.T) {
	d, _ := newTestDispatcher(t, DefaultConfig())
	d.Add("test_echo", Func(echo))

	tests := map[string]struct {
		params string
		result string
		code   int
		msg    string
	}{
		"all":                {params: `["a",2]`, result: `"a:2"`},
		"optional absent":    {params: `["a"]`, result: `"a:-1"`},
		"optional null":      {params: `["a",null]`, result: `"a:-1"`},
		"required missing":   {params: `[]`, code: CodeInvalidParams, msg: "parameter 0 (name): missing value"},
		"no params":          {params: ``, code: CodeInvalidParams, msg: "parameter 0 (name): missing value"},
		"null params":        {params: `null`, code: CodeInvalidParams, msg: "parameter 0 (name): missing value"},
		"too many":           {params: `["a",1,2]`, code: CodeInvalidParams, msg: "too many arguments, want at most 2"},
		"not an array":       {params: `{"name":"a"}`, code: CodeInvalidParams, msg: "non-array args"},
		"undecodable":        {params: `[1]`, code: CodeInvalidParams, msg: "parameter 0 (name): json: cannot unmarshal number"},
		"undecodable second": {params: `["a","x"]`, code: CodeInvalidParams, msg: "parameter 1 (count): json: cannot unmarshal string"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			res, errObj, ok := serve(d, "test_echo", tc.params)
			if tc.code == 0 {
				require.True(t, ok, errObj.Message)
				require.JSONEq(t, tc.result, string(res))
				return
			}
			require.False(t, ok)
			require.Equal(t, tc.code, errObj.Code)
			require.Contains(t, errObj.Message, tc.msg)
		})
	}
}

func TestServe_DecodingFailureNeverReachesTheHandler(t *testing.T) {
	d, _ := newTestDispatcher(t, DefaultConfig())
	var calls atomic.Int32
	d.Add("test_count", Func(func(context.Context, twoParams) (any, error) {
		calls.Add(1)
		return nil, nil
	}))
	_, _, ok := serve(d, "test_count", `[1]`)
	require.False(t, ok)
	require.Zero(t, calls.Load())
}

func TestServe_UnknownMethodIsMethodNotFound(t *testing.T) {
	d, reg := newTestDispatcher(t, DefaultConfig())
	_, errObj, ok := serve(d, "nope_method", `[]`)
	require.False(t, ok)
	require.Equal(t, CodeMethodNotFound, errObj.Code)
	require.Equal(t, "the method nope_method does not exist/is not available", errObj.Message)
	require.Equal(t, 1.0, value(t, reg, "rpc_requests_total", "method", otherMethod, "result", resultNotFound))
}

func TestServe_MethodNameTooLongIsInvalidRequest(t *testing.T) {
	d, _ := newTestDispatcher(t, DefaultConfig())
	_, errObj, ok := serve(d, strings.Repeat("x", maxMethodNameLength+1), `[]`)
	require.False(t, ok)
	require.Equal(t, CodeInvalidRequest, errObj.Code)
}

type codedErr struct{ code int }

func (e codedErr) Error() string  { return "coded" }
func (e codedErr) ErrorCode() int { return e.code }
func (e codedErr) ErrorData() any { return "0xdata" }

func TestServe_ErrorsKeepTheirCodeOnlyWhenNotWrapped(t *testing.T) {
	d, _ := newTestDispatcher(t, DefaultConfig())
	d.Add("test_plain", Func(func(context.Context, None) (any, error) { return nil, errors.New("plain") }))
	d.Add("test_coded", Func(func(context.Context, None) (any, error) { return nil, codedErr{3} }))
	d.Add("test_wrapped", Func(func(context.Context, None) (any, error) { return nil, fmt.Errorf("w: %w", codedErr{3}) }))

	_, e, _ := serve(d, "test_plain", ``)
	require.Equal(t, ErrorObject{Code: CodeDefault, Message: "plain"}, e)
	_, e, _ = serve(d, "test_coded", ``)
	require.Equal(t, ErrorObject{Code: 3, Message: "coded", Data: "0xdata"}, e)
	_, e, _ = serve(d, "test_wrapped", ``)
	require.Equal(t, CodeDefault, e.Code, "a wrapped error is answered -32000, as by go-ethereum")
}

func TestServe_PanicIsRecoveredAnsweredAndCounted(t *testing.T) {
	d, reg := newTestDispatcher(t, DefaultConfig())
	d.Add("test_panic", Func(func(context.Context, None) (any, error) { panic("boom") }))
	before := value(t, reg, "rpc_panics_total", "method", "test_panic")
	_, e, ok := serve(d, "test_panic", ``)
	require.False(t, ok)
	require.Equal(t, ErrorObject{Code: CodeInternal, Message: "method handler crashed"}, e)
	require.Equal(t, before+1, value(t, reg, "rpc_panics_total", "method", "test_panic"))
	require.Equal(t, 1.0, value(t, reg, "rpc_requests_total", "method", "test_panic", "result", resultPanic))
}

func TestServe_AttachesTheRequestDeadline(t *testing.T) {
	d, _ := newTestDispatcher(t, Config{RequestTimeout: time.Minute})
	d.Add("test_deadline", Func(func(ctx context.Context, _ None) (any, error) {
		deadline, ok := ctx.Deadline()
		return ok && time.Until(deadline) <= time.Minute, nil
	}))
	res, _, ok := serve(d, "test_deadline", ``)
	require.True(t, ok)
	require.Equal(t, "true", string(res))

	d2, _ := newTestDispatcher(t, Config{})
	d2.Add("test_deadline", Func(func(ctx context.Context, _ None) (any, error) {
		_, ok := ctx.Deadline()
		return ok, nil
	}))
	res, _, _ = serve(d2, "test_deadline", ``)
	require.Equal(t, "false", string(res), "no deadline when the timeout is zero")
}

func TestServe_EncodedPassesThroughAndRawMessageIsCompacted(t *testing.T) {
	d, _ := newTestDispatcher(t, DefaultConfig())
	d.Add("test_encoded", Func(func(context.Context, None) (any, error) { return Encoded(`{ "a" : 1 }`), nil }))
	d.Add("test_raw", Func(func(context.Context, None) (any, error) { return json.RawMessage(`{ "a" : 1 }`), nil }))
	res, _, _ := serve(d, "test_encoded", ``)
	require.Equal(t, `{ "a" : 1 }`, string(res))
	res, _, _ = serve(d, "test_raw", ``)
	require.Equal(t, `{"a":1}`, string(res))
}

func TestServe_SubscriptionsNeedAStream(t *testing.T) {
	d, _ := newTestDispatcher(t, DefaultConfig())
	d.AddSubscription("eth", "newHeads", Func(func(context.Context, None) (any, error) { return "0x1", nil }))

	_, e, _ := serve(d, "eth_subscribe", `["newHeads"]`)
	require.Equal(t, ErrorObject{Code: CodeMethodNotFound, Message: "notifications not supported"}, e)
	_, e, _ = serve(d, "foo_subscribe", `["x"]`)
	require.Equal(t, "notifications not supported", e.Message, "every namespace, as in go-ethereum")
	_, e, _ = serve(d, "eth_unsubscribe", `["0x1"]`)
	require.Equal(t, ErrorObject{Code: CodeDefault, Message: "subscription not found"}, e)
}

func TestServe_SubscriptionKindsByNamespace(t *testing.T) {
	d, _ := newTestDispatcher(t, DefaultConfig())
	d.AddSubscription("eth", "newHeads", Func(func(ctx context.Context, _ None) (any, error) {
		n, _ := NotifierFrom(ctx)
		sub, err := n.Subscribe("eth", "newHeads")
		if err != nil {
			return nil, err
		}
		return sub.ID(), nil
	}))
	s := d.NewStream(context.Background(), StreamLimits{MaxSubscriptions: 2, Buffer: 4}, func([]byte) error { return nil }, func(string) {})
	defer s.Close()
	call := func(method, params string) (json.RawMessage, error) {
		n := s.Request()
		res, err := d.Serve(WithNotifier(context.Background(), n), Call{Method: method, Params: json.RawMessage(params)})
		n.Finish(err != nil)
		return res, err
	}
	_, err := call("eth_subscribe", `["logs"]`)
	require.Equal(t, `no "logs" subscription in eth namespace`, err.Error())
	_, err = call("eth_subscribe", `[1]`)
	require.Equal(t, CodeInvalidParams, ToErrorObject(err).Code)
	_, err = call("eth_subscribe", `["newHeads", 1]`)
	require.Equal(t, "too many arguments, want at most 1", err.Error())

	res, err := call("eth_subscribe", `["newHeads"]`)
	require.NoError(t, err)
	_, err = call("eth_subscribe", `["newHeads"]`)
	require.NoError(t, err)
	_, err = call("eth_subscribe", `["newHeads"]`)
	require.ErrorContains(t, err, "too many subscriptions, the limit is 2")

	var id string
	require.NoError(t, json.Unmarshal(res, &id))
	res, err = call("eth_unsubscribe", fmt.Sprintf(`[%q]`, id))
	require.NoError(t, err)
	require.Equal(t, "true", string(res))
	_, err = call("eth_unsubscribe", fmt.Sprintf(`[%q]`, id))
	require.Equal(t, "subscription not found", err.Error())
}

func TestStream_NotificationsWaitForTheAnswer(t *testing.T) {
	d, _ := newTestDispatcher(t, DefaultConfig())
	written := make(chan []byte, 10)
	s := d.NewStream(context.Background(), StreamLimits{MaxSubscriptions: 10, Buffer: 10}, func(msg []byte) error {
		written <- msg
		return nil
	}, func(string) {})
	defer s.Close()

	n := s.Request()
	sub, err := n.Subscribe("eth", "newHeads")
	require.NoError(t, err)
	sub.Notify(1)
	select {
	case <-written:
		t.Fatal("a notification was written before the answer")
	case <-time.After(20 * time.Millisecond):
	}
	n.Finish(false)
	select {
	case msg := <-written:
		require.JSONEq(t, fmt.Sprintf(`{"jsonrpc":"2.0","method":"eth_subscription","params":{"subscription":%q,"result":1}}`, sub.ID()), string(msg))
	case <-time.After(time.Second):
		t.Fatal("no notification after the answer")
	}

	failing := s.Request()
	sub2, err := failing.Subscribe("eth", "newHeads")
	require.NoError(t, err)
	failing.Finish(true)
	select {
	case <-sub2.Done():
	case <-time.After(time.Second):
		t.Fatal("the subscription of a failed request lives on")
	}
}

func TestStream_ASubscriptionThatFallsBehindClosesItsConnection(t *testing.T) {
	d, reg := newTestDispatcher(t, DefaultConfig())
	closed := make(chan string, 1)
	block := make(chan struct{})
	s := d.NewStream(context.Background(), StreamLimits{MaxSubscriptions: 1, Buffer: 2}, func([]byte) error {
		<-block
		return nil
	}, func(reason string) { closed <- reason })
	defer func() {
		close(block)
		s.Close()
	}()
	n := s.Request()
	sub, err := n.Subscribe("eth", "logs")
	require.NoError(t, err)
	n.Finish(false)
	for i := 0; i < 10; i++ {
		sub.Notify(i)
	}
	select {
	case reason := <-closed:
		require.Contains(t, reason, "fell behind")
	case <-time.After(time.Second):
		t.Fatal("the connection of a subscription that fell behind stays open")
	}
	require.Equal(t, 1.0, value(t, reg, "rpc_subscription_events_total", "kind", "logs", "outcome", OutcomeDropped))
}

func TestStream_IdsBelongToTheirConnection(t *testing.T) {
	d, _ := newTestDispatcher(t, DefaultConfig())
	limits := StreamLimits{MaxSubscriptions: 1, Buffer: 1}
	a := d.NewStream(context.Background(), limits, func([]byte) error { return nil }, func(string) {})
	b := d.NewStream(context.Background(), limits, func([]byte) error { return nil }, func(string) {})
	defer a.Close()
	defer b.Close()
	n := a.Request()
	sub, err := n.Subscribe("eth", "logs")
	require.NoError(t, err)
	n.Finish(false)
	require.False(t, b.Request().Unsubscribe(sub.ID()))
	require.True(t, a.Request().Unsubscribe(sub.ID()))
}

func TestNewID_IsAHexQuantityOfRandomBytes(t *testing.T) {
	seen := map[string]bool{}
	for range 100 {
		id := NewID()
		require.True(t, strings.HasPrefix(id, "0x"))
		require.False(t, strings.HasPrefix(id, "0x0") && id != "0x0")
		require.LessOrEqual(t, len(id), 34)
		require.False(t, seen[id])
		seen[id] = true
	}
}

func TestGroup_APanickingChildCancelsItsSiblings(t *testing.T) {
	d, reg := newTestDispatcher(t, DefaultConfig())
	g, ctx := d.Group(context.WithValue(context.Background(), methodKey{}, "test_group"))
	var siblingDone atomic.Bool
	g.Go(func() error {
		<-ctx.Done()
		siblingDone.Store(true)
		return nil
	})
	g.Go(func() error { panic("child") })
	err := g.Wait()
	require.Equal(t, errPanic, err)
	require.True(t, siblingDone.Load(), "Wait returned before the sibling")
	require.Equal(t, 1.0, value(t, reg, "rpc_panics_total", "method", "test_group"))
}

func TestGroup_TheFirstFailureIsReturned(t *testing.T) {
	d, _ := newTestDispatcher(t, DefaultConfig())
	g, ctx := d.Group(context.Background())
	want := errors.New("first")
	g.Go(func() error { return want })
	g.Go(func() error {
		<-ctx.Done()
		return errors.New("second")
	})
	require.Equal(t, want, g.Wait())
}

func TestLoop_RestartsAfterAPanicAndDoesItsJob(t *testing.T) {
	d, reg := newTestDispatcher(t, DefaultConfig())
	var runs atomic.Int32
	worked := make(chan struct{})
	d.Loop("test_loop", func(ctx context.Context) {
		if runs.Add(1) == 1 {
			panic("first run")
		}
		close(worked)
		<-ctx.Done()
	})
	d.Start()
	select {
	case <-worked:
	case <-time.After(5 * time.Second):
		t.Fatal("the loop did not run again after its panic")
	}
	require.Equal(t, 1.0, value(t, reg, "rpc_loop_restarts_total", "loop", "test_loop"))
}

func TestLoop_LoopsWaitForStart(t *testing.T) {
	d, _ := newTestDispatcher(t, DefaultConfig())
	var started atomic.Bool
	d.Loop("test_loop", func(ctx context.Context) {
		started.Store(true)
		<-ctx.Done()
	})
	time.Sleep(10 * time.Millisecond)
	require.False(t, started.Load())
	d.Start()
	require.Eventually(t, started.Load, time.Second, time.Millisecond)
}

func TestStop_LeavesNoGoroutineAndRefusesWork(t *testing.T) {
	before := runtime.NumGoroutine()
	d := New(DefaultConfig(), NewMetrics(prometheus.NewRegistry()))
	d.Add("test_wait", Func(func(ctx context.Context, _ None) (any, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}))
	d.Loop("test_loop", func(ctx context.Context) { <-ctx.Done() })
	d.Start()
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		d.Go("test_request", func(ctx context.Context) {
			defer wg.Done()
			_, _ = d.Serve(ctx, Call{Method: "test_wait"})
		})
	}
	d.Stop(10 * time.Millisecond)
	wg.Wait()
	if !assertEventually(func() bool { return runtime.NumGoroutine() <= before }) {
		buf := make([]byte, 1<<20)
		t.Fatalf("goroutines before %d, after %d:\n%s", before, runtime.NumGoroutine(), buf[:runtime.Stack(buf, true)])
	}

	_, err := d.Serve(context.Background(), Call{Method: "test_wait"})
	require.Equal(t, errStopped, err)
	require.False(t, d.Go("late", func(context.Context) {}))
	require.False(t, d.Loop("late", func(context.Context) {}))
}

// TestStop_AdmissionRacesStop runs requests and goroutines against Stop; under
// -race, an admission after the wait returned shows.
func TestStop_AdmissionRacesStop(t *testing.T) {
	for range 20 {
		d := New(DefaultConfig(), NewMetrics(prometheus.NewRegistry()))
		d.Add("test_ok", Func(func(context.Context, None) (any, error) { return 1, nil }))
		var after atomic.Bool
		var wg sync.WaitGroup
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range 50 {
					if _, err := d.Serve(context.Background(), Call{Method: "test_ok"}); err == nil && after.Load() {
						t.Error("a request was served after Stop returned")
					}
					d.Go("test_go", func(context.Context) {
						if after.Load() {
							t.Error("a goroutine ran after Stop returned")
						}
					})
				}
			}()
		}
		d.Stop(time.Second)
		after.Store(true)
		wg.Wait()
	}
}

// TestStop_WaitsForWorkThatIgnoresCancellation checks that Stop does not
// return while a request still runs past the grace period, as an EVM run on
// Tosca or a Core call would.
func TestStop_WaitsForWorkThatIgnoresCancellation(t *testing.T) {
	d := New(DefaultConfig(), NewMetrics(prometheus.NewRegistry()))
	release := make(chan struct{})
	running := make(chan struct{})
	d.Add("test_stuck", Func(func(context.Context, None) (any, error) {
		close(running)
		<-release // ignores its context
		return nil, nil
	}))
	served := make(chan struct{})
	go func() {
		defer close(served)
		_, _ = d.Serve(context.Background(), Call{Method: "test_stuck"})
	}()
	<-running
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		d.Stop(10 * time.Millisecond)
	}()
	select {
	case <-stopped:
		t.Fatal("Stop returned while a request still ran")
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	<-served
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop did not return after the request ended")
	}
}

func TestMetrics_CountRequestsByMethodAndResult(t *testing.T) {
	d, reg := newTestDispatcher(t, DefaultConfig())
	d.Add("test_ok", Func(func(context.Context, None) (any, error) { return "x", nil }))
	d.Add("test_err", Func(func(context.Context, None) (any, error) { return nil, errors.New("e") }))
	serve(d, "test_ok", ``)
	serve(d, "test_err", ``)
	serve(d, "test_ok", `[1]`)
	for i := range 50 {
		serve(d, fmt.Sprintf("random_%d", i), ``)
	}
	require.Equal(t, 1.0, value(t, reg, "rpc_requests_total", "method", "test_ok", "result", resultOK))
	require.Equal(t, 1.0, value(t, reg, "rpc_requests_total", "method", "test_err", "result", resultError))
	require.Equal(t, 1.0, value(t, reg, "rpc_requests_total", "method", "test_ok", "result", resultInvalidParams))
	require.Equal(t, 50.0, value(t, reg, "rpc_requests_total", "method", otherMethod, "result", resultNotFound))
	// The label set stays bounded whatever names clients send.
	require.Equal(t, 4, series(t, reg, "rpc_requests_total"))
	require.Equal(t, 0.0, value(t, reg, "rpc_inflight"))
	require.Equal(t, 53.0, value(t, reg, "rpc_duration_seconds"))
	require.Equal(t, 1.0, value(t, reg, "rpc_response_bytes", "method", "test_ok"))
}

func TestRegister_SharesACollectorRegisteredAlready(t *testing.T) {
	reg := prometheus.NewRegistry()
	a := NewMetrics(reg)
	b := NewMetrics(reg)
	require.Same(t, a.requests, b.requests)
}

func TestDispatcher_ListsMethodsAndNamespaces(t *testing.T) {
	d, _ := newTestDispatcher(t, DefaultConfig())
	d.Add("eth_a", Func(func(context.Context, None) (any, error) { return nil, nil }))
	d.Add("web3_b", Func(func(context.Context, None) (any, error) { return nil, nil }))
	d.AddSubscription("eth", "newHeads", Func(func(context.Context, None) (any, error) { return nil, nil }))
	require.Equal(t, []string{"eth_a", "eth_subscribe", "eth_unsubscribe", "web3_b"}, d.Methods())
	require.Equal(t, []string{"eth", "web3"}, d.Namespaces())
	require.True(t, d.Has("eth_a"))
	require.True(t, d.Has("eth_subscribe"))
	require.False(t, d.Has("ftm_subscribe"))
	require.Panics(t, func() { d.Add("eth_a", Func(func(context.Context, None) (any, error) { return nil, nil })) })
}

func TestForward_RunsUnderTheDispatcher(t *testing.T) {
	d, reg := newTestDispatcher(t, DefaultConfig())
	res, err := d.Forward(context.Background(), Call{Method: "admin_x"}, func(context.Context, Call) (json.RawMessage, error) {
		return json.RawMessage(`"ok"`), nil
	})
	require.NoError(t, err)
	require.Equal(t, `"ok"`, string(res))
	_, err = d.Forward(context.Background(), Call{Method: "admin_x"}, func(context.Context, Call) (json.RawMessage, error) {
		panic("node")
	})
	require.Equal(t, errPanic, err)
	require.Equal(t, 1.0, value(t, reg, "rpc_requests_total", "method", forwardedMethod, "result", resultPanic))
}

func TestFunc_RefusesParametersThatAreNotAStruct(t *testing.T) {
	require.Panics(t, func() { Func(func(context.Context, int) (any, error) { return nil, nil }) })
	type private struct{ x int }
	_ = private{x: 1}
	require.Panics(t, func() { Func(func(context.Context, private) (any, error) { return nil, nil }) })
}

func assertEventually(condition func() bool) bool {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return condition()
}
