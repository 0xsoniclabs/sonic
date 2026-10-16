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
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// Harness serves methods through a real dispatcher over a core, the way a
// transport would.
type Harness struct {
	t     testing.TB
	D     *dispatch.Dispatcher
	Chain *dispatch.Chain
	Core  core.Core
}

// NewHarness creates a dispatcher and a chain view over the core, lets the
// register function add namespaces, starts the chain and stops everything with
// the test.
func NewHarness(t testing.TB, c core.Core, cfg dispatch.Config, register func(*dispatch.Dispatcher, *dispatch.Chain)) *Harness {
	t.Helper()
	d := dispatch.New(cfg, dispatch.NewMetrics(nil))
	chain := dispatch.NewChain(c, cfg, d.Metrics())
	if register != nil {
		register(d, chain)
	}
	require.NoError(t, chain.Start(d))
	t.Cleanup(func() { d.Stop(time.Second) })
	return &Harness{t: t, D: d, Chain: chain, Core: c}
}

// Call serves a method with positional parameters.
func (h *Harness) Call(method string, params ...any) (json.RawMessage, *dispatch.Error) {
	return h.CallContext(context.Background(), method, params...)
}

// CallContext is Call with a context.
func (h *Harness) CallContext(ctx context.Context, method string, params ...any) (json.RawMessage, *dispatch.Error) {
	h.t.Helper()
	raw, err := json.Marshal(params)
	require.NoError(h.t, err)
	return h.D.Serve(ctx, method, raw)
}

// CallRaw serves a method with the given raw params member.
func (h *Harness) CallRaw(method string, params string) (json.RawMessage, *dispatch.Error) {
	var raw json.RawMessage
	if params != "" {
		raw = json.RawMessage(params)
	}
	return h.D.Serve(context.Background(), method, raw)
}

// Result calls a method that must succeed and returns its encoded result.
func (h *Harness) Result(method string, params ...any) json.RawMessage {
	h.t.Helper()
	res, err := h.Call(method, params...)
	require.Nil(h.t, err, "calling %s", method)
	return res
}

// Into calls a method that must succeed and decodes its result.
func (h *Harness) Into(out any, method string, params ...any) {
	h.t.Helper()
	require.NoError(h.t, json.Unmarshal(h.Result(method, params...), out))
}

// Fails calls a method that must fail and returns the error.
func (h *Harness) Fails(method string, params ...any) *dispatch.Error {
	h.t.Helper()
	res, err := h.Call(method, params...)
	require.NotNil(h.t, err, "calling %s returned %s", method, res)
	return err
}

// Default returns the default configuration of the module.
func Default() dispatch.Config { return dispatch.DefaultConfig() }

// Panics returns how many panics the harness's dispatcher recovered.
func (h *Harness) Panics() float64 {
	h.t.Helper()
	var total float64
	for _, method := range h.D.Methods() {
		total += testutilCounter(h.D.Metrics().Panics.WithLabelValues(method))
	}
	return total
}

func testutilCounter(c prometheus.Counter) float64 {
	var m dto.Metric
	if err := c.Write(&m); err != nil {
		return 0
	}
	return m.GetCounter().GetValue()
}
