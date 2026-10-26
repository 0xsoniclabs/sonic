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

package tests

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
)

// waitTimeout bounds every wait of the harness; a wait that reaches it fails
// the test.
const waitTimeout = 10 * time.Second

// Configs are the configurations of the packages a harness wires.
type Configs struct {
	Dispatch dispatch.Config
	Chain    chain.Config
	Exec     exec.Config
	Oracle   oracle.Config
}

// Mutate changes the default configurations for one test.
type Mutate func(*Configs)

// Harness is a dispatcher, a chain and an environment over a core, with
// helpers that call methods the way a transport does.
type Harness struct {
	t        testing.TB
	D        *dispatch.Dispatcher
	Chain    *chain.Chain
	Env      *exec.Env
	Registry *prometheus.Registry
}

// NewHarness wires the module's runtime over a core, lets register add the
// methods under test, and starts it; it stops with the test.
func NewHarness(t testing.TB, c core.Core, register func(h *Harness), mutate ...Mutate) *Harness {
	t.Helper()
	cfgs := Configs{
		Dispatch: dispatch.DefaultConfig(),
		Chain:    chain.DefaultConfig(),
		Exec:     exec.DefaultConfig(),
		Oracle:   oracle.DefaultConfig(),
	}
	for _, m := range mutate {
		m(&cfgs)
	}
	reg := prometheus.NewRegistry()
	d := dispatch.New(cfgs.Dispatch, dispatch.NewMetrics(reg))
	ch := chain.New(c, cfgs.Chain, d)
	o := oracle.New(cfgs.Oracle, c, func() (*opera.Rules, error) {
		return c.Rules(ch.Latest().Number() + 1)
	})
	h := &Harness{t: t, D: d, Chain: ch, Env: exec.NewEnv(c, ch, o, d, cfgs.Exec), Registry: reg}
	if register != nil {
		register(h)
	}
	if err := ch.Start(); err != nil {
		t.Fatalf("starting the chain: %v", err)
	}
	d.Start()
	t.Cleanup(func() { d.Stop(time.Second) })
	return h
}

// Call serves one call, as a transport hands it to the dispatcher.
func (h *Harness) Call(method string, params ...any) (json.RawMessage, error) {
	return h.CallContext(context.Background(), method, params...)
}

// CallContext serves one call under a context.
func (h *Harness) CallContext(ctx context.Context, method string, params ...any) (json.RawMessage, error) {
	raw, err := json.Marshal(params)
	if err != nil {
		h.t.Fatalf("encoding the parameters of %s: %v", method, err)
	}
	return h.D.Serve(ctx, dispatch.Call{Method: method, Params: raw})
}

// CallRaw serves one call with parameters as the client sent them.
func (h *Harness) CallRaw(method string, params string) (json.RawMessage, error) {
	return h.D.Serve(context.Background(), dispatch.Call{Method: method, Params: json.RawMessage(params)})
}

// MustCall serves one call, fails the test on an error, and decodes the result
// into result unless it is nil.
func (h *Harness) MustCall(result any, method string, params ...any) json.RawMessage {
	h.t.Helper()
	raw, err := h.Call(method, params...)
	if err != nil {
		h.t.Fatalf("%s failed: %v", method, err)
	}
	if result != nil {
		if err := json.Unmarshal(raw, result); err != nil {
			h.t.Fatalf("decoding the answer of %s: %v (%s)", method, err, raw)
		}
	}
	return raw
}

// Fails serves one call and returns its error object; the test fails if the
// call succeeds.
func (h *Harness) Fails(method string, params ...any) dispatch.ErrorObject {
	h.t.Helper()
	raw, err := h.Call(method, params...)
	if err == nil {
		h.t.Fatalf("%s succeeded with %s, want an error", method, raw)
	}
	return dispatch.ToErrorObject(err)
}

// Panics is the number of panics the dispatcher recovered.
func (h *Harness) Panics() float64 {
	return Scrape(h.t, h.Registry, "rpc_panics_total", nil)
}

// WaitHead waits until the served head reaches a block; it fails the test at
// the wait timeout.
func (h *Harness) WaitHead(number uint64) {
	h.t.Helper()
	WaitFor(h.t, func() bool { return h.Chain.Latest().Number() >= number })
}

// WaitFor waits for a condition; it fails the test at the wait timeout.
func WaitFor(t testing.TB, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(waitTimeout)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatalf("condition not reached within %v", waitTimeout)
		}
		time.Sleep(time.Millisecond)
	}
}

// Scrape sums the values of a metric of a registry whose labels include the
// given ones.
func Scrape(t testing.TB, reg *prometheus.Registry, name string, labels map[string]string) float64 {
	t.Helper()
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("gathering metrics: %v", err)
	}
	var sum float64
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
	metrics:
		for _, m := range family.GetMetric() {
			for k, v := range labels {
				found := false
				for _, l := range m.GetLabel() {
					found = found || (l.GetName() == k && l.GetValue() == v)
				}
				if !found {
					continue metrics
				}
			}
			switch {
			case m.Counter != nil:
				sum += m.Counter.GetValue()
			case m.Gauge != nil:
				sum += m.Gauge.GetValue()
			case m.Histogram != nil:
				sum += float64(m.Histogram.GetSampleCount())
			}
		}
	}
	return sum
}

// Stream is a stream connection of the harness: calls on it may subscribe,
// and its notifications arrive on Notifications.
type Stream struct {
	h             *Harness
	state         *dispatch.Stream
	Notifications chan json.RawMessage
	Closed        chan string
}

// Stream opens a stream connection that closes with the test.
func (h *Harness) Stream(limits dispatch.StreamLimits) *Stream {
	s := &Stream{h: h, Notifications: make(chan json.RawMessage, 1024), Closed: make(chan string, 1)}
	s.state = h.D.NewStream(context.Background(), limits,
		func(msg []byte) error {
			select {
			case s.Notifications <- json.RawMessage(msg):
				return nil
			default:
				return errors.New("test stream full")
			}
		},
		func(reason string) {
			select {
			case s.Closed <- reason:
			default:
			}
		})
	h.t.Cleanup(s.state.Close)
	return s
}

// Call serves a call on the stream; the notifications of a subscription it
// opens are released once it answered.
func (s *Stream) Call(method string, params ...any) (json.RawMessage, error) {
	raw, err := json.Marshal(params)
	if err != nil {
		s.h.t.Fatalf("encoding the parameters of %s: %v", method, err)
	}
	n := s.state.Request()
	res, err := s.h.D.Serve(dispatch.WithNotifier(context.Background(), n), dispatch.Call{Method: method, Params: raw})
	n.Finish(err != nil)
	return res, err
}

// Next waits for the next notification; it fails the test at the wait
// timeout.
func (s *Stream) Next() json.RawMessage {
	s.h.t.Helper()
	select {
	case msg := <-s.Notifications:
		return msg
	case <-time.After(waitTimeout):
		s.h.t.Fatalf("no notification within %v", waitTimeout)
		return nil
	}
}

// Close closes the connection.
func (s *Stream) Close() {
	s.state.Close()
}
