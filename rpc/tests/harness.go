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

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

// Settings are the configurations a harness starts with.
type Settings struct {
	Dispatch dispatch.Config
	Chain    chain.Config
	Exec     exec.Config
	Oracle   oracle.Config
}

// Harness serves the methods of a namespace over a core as a transport would:
// a dispatcher, the served chain and the environment of the namespaces.
type Harness struct {
	Core        *FakeCore
	Dispatcher  *dispatch.Dispatcher
	Chain       *chain.Chain
	Environment *exec.Environment
	registry    *prometheus.Registry
}

// NewHarness starts a harness over the fake core, with the methods register
// adds and the default settings changed by the mutate functions. It stops
// with the test.
func NewHarness(t testing.TB, fake *FakeCore, register func(*Harness), mutate ...func(*Settings)) *Harness {
	t.Helper()
	settings := Settings{Dispatch: dispatch.DefaultConfig(), Exec: exec.DefaultConfig(), Oracle: oracle.DefaultConfig()}
	for _, change := range mutate {
		change(&settings)
	}
	registry := prometheus.NewRegistry()
	harness := &Harness{
		Core:       fake,
		Dispatcher: dispatch.New(settings.Dispatch, registry),
		Chain:      chain.New(settings.Chain, fake, registry),
		registry:   registry,
	}
	suggester := oracle.New(settings.Oracle, fake, func() uint64 { return harness.Chain.Head().NumberU64() })
	harness.Environment = exec.NewEnvironment(settings.Exec, harness.Chain, fake, suggester)
	register(harness)
	require.NoError(t, harness.Chain.Start(harness.Dispatcher))
	t.Cleanup(harness.Dispatcher.Stop)
	// The chain's loops subscribe to the node's feeds; until they have, an
	// event would be lost.
	require.Eventually(t, func() bool {
		return fake.Calls("SubscribeHead") > 0 && fake.Calls("SubscribeTransactions") > 0
	}, time.Second, time.Millisecond)
	return harness
}

// Serve calls a method with the given parameters, which are encoded as the
// positional parameters of a request.
func (h *Harness) Serve(t testing.TB, method string, params ...any) dispatch.Response {
	t.Helper()
	encoded, err := json.Marshal(params)
	require.NoError(t, err)
	return h.Dispatcher.Serve(context.Background(), method, encoded)
}

// Call calls a method that must succeed, and decodes its result into result.
func (h *Harness) Call(t testing.TB, result any, method string, params ...any) {
	t.Helper()
	response := h.Serve(t, method, params...)
	require.Nil(t, response.Error, "%s failed", method)
	require.NoError(t, json.Unmarshal(response.Result, result))
}

// Result calls a method that must succeed, and answers its encoded result.
func (h *Harness) Result(t testing.TB, method string, params ...any) string {
	t.Helper()
	response := h.Serve(t, method, params...)
	require.Nil(t, response.Error, "%s failed", method)
	return string(response.Result)
}

// Fail calls a method that must fail, and answers its error.
func (h *Harness) Fail(t testing.TB, method string, params ...any) *dispatch.Error {
	t.Helper()
	response := h.Serve(t, method, params...)
	require.NotNil(t, response.Error, "%s answered %s", method, response.Result)
	return response.Error
}

// Panics is the number of panics the dispatcher has recovered.
func (h *Harness) Panics(t testing.TB) float64 {
	t.Helper()
	families, err := h.registry.Gather()
	require.NoError(t, err)
	total := 0.0
	for _, family := range families {
		if family.GetName() != "rpc_panics_total" {
			continue
		}
		for _, metric := range family.GetMetric() {
			total += metric.GetCounter().GetValue()
		}
	}
	return total
}

// SetHead serves a new head and waits until the chain serves it.
func (h *Harness) SetHead(t testing.TB, number uint64) {
	t.Helper()
	h.Core.SetHead(number)
	require.Eventually(t, func() bool { return h.Chain.Head().NumberU64() == number },
		5*time.Second, time.Millisecond, "the chain does not serve block %d", number)
}
