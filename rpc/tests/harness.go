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

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

// Configs are the configurations of the packages a harness wires, at their
// defaults until a Mutate changes them.
type Configs struct {
	Dispatch dispatch.Config
	Chain    chain.Config
	Exec     exec.Config
}

// Mutate changes the default configurations for one harness.
type Mutate func(configs *Configs)

// Register adds the methods of the namespaces under test.
type Register func(dispatcher *dispatch.Dispatcher, environment *exec.Environment)

// Harness is a dispatcher and a served chain over a core, the way the module
// wires them, with a registry of its own.
type Harness struct {
	t           testing.TB
	Dispatcher  *dispatch.Dispatcher
	Chain       *chain.Chain
	Environment *exec.Environment
	Registry    *prometheus.Registry
}

// NewHarness starts a harness over the core, with the methods register adds.
// It stops with the test.
func NewHarness(t testing.TB, nodeCore core.Core, register Register, mutations ...Mutate) *Harness {
	t.Helper()
	configs := Configs{
		Dispatch: dispatch.DefaultConfig(),
		Chain:    chain.DefaultConfig(),
		Exec:     exec.DefaultConfig(),
	}
	for _, mutate := range mutations {
		mutate(&configs)
	}
	registry := prometheus.NewRegistry()
	metrics := dispatch.NewMetrics(registry)
	dispatcher := dispatch.New(configs.Dispatch, metrics)
	served := chain.New(configs.Chain, nodeCore, metrics)
	environment := exec.NewEnvironment(nodeCore, served, configs.Exec)
	register(dispatcher, environment)
	if err := served.Start(dispatcher); err != nil {
		t.Fatalf("failed to start the chain: %v", err)
	}
	t.Cleanup(dispatcher.Stop)
	return &Harness{t: t, Dispatcher: dispatcher, Chain: served, Environment: environment, Registry: registry}
}

// Call serves a method with the given parameters, each encoded as JSON.
func (harness *Harness) Call(method string, parameters ...any) (json.RawMessage, *dispatch.Error) {
	harness.t.Helper()
	return harness.CallContext(context.Background(), method, parameters...)
}

// CallContext serves a method under the given context.
func (harness *Harness) CallContext(ctx context.Context, method string, parameters ...any) (json.RawMessage, *dispatch.Error) {
	harness.t.Helper()
	if parameters == nil {
		parameters = []any{}
	}
	encoded, err := json.Marshal(parameters)
	if err != nil {
		harness.t.Fatalf("failed to encode the parameters of %s: %v", method, err)
	}
	return harness.Dispatcher.Serve(ctx, method, encoded)
}

// Result serves a method and fails the test if it answers an error.
func (harness *Harness) Result(method string, parameters ...any) json.RawMessage {
	harness.t.Helper()
	result, err := harness.Call(method, parameters...)
	if err != nil {
		harness.t.Fatalf("%s answered %d %q", method, err.Code, err.Message)
	}
	return result
}

// Failure serves a method and fails the test unless it answers an error.
func (harness *Harness) Failure(method string, parameters ...any) *dispatch.Error {
	harness.t.Helper()
	result, err := harness.Call(method, parameters...)
	if err == nil {
		harness.t.Fatalf("%s answered %s, want an error", method, result)
	}
	return err
}

// Decode decodes a result into a value of the given type.
func Decode[Value any](t testing.TB, raw json.RawMessage) Value {
	t.Helper()
	var value Value
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("failed to decode %s: %v", raw, err)
	}
	return value
}

// Panics is the number of panics the dispatcher recovered.
func (harness *Harness) Panics() float64 {
	harness.t.Helper()
	return Sum(harness.t, harness.Registry, "rpc_panics_total")
}

// Sum adds up the values of the series of a metric in a registry: counters
// and gauges by value, histograms by their number of samples.
func Sum(t testing.TB, registry *prometheus.Registry, name string) float64 {
	return Labeled(t, registry, name, nil)
}

// Labeled adds up the values of the series of a metric that carry the given
// labels.
func Labeled(t testing.TB, registry *prometheus.Registry, name string, labels map[string]string) float64 {
	t.Helper()
	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}
	var total float64
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			matched := 0
			for _, pair := range metric.GetLabel() {
				if labels[pair.GetName()] == pair.GetValue() {
					matched++
				}
			}
			if matched < len(labels) {
				continue
			}
			total += metric.GetCounter().GetValue() + metric.GetGauge().GetValue() + float64(metric.GetHistogram().GetSampleCount())
		}
	}
	return total
}

// WaitForHead waits until the harness serves at least the given block, and
// fails the test after a deadline.
func (harness *Harness) WaitForHead(number uint64) {
	harness.t.Helper()
	Eventually(harness.t, func() bool {
		head, err := harness.Chain.Latest()
		return err == nil && head.Number >= number
	}, "the served head to reach block %d", number)
}

// Deadline is how long a wait of the test support waits before it fails the
// test.
const Deadline = 10 * time.Second

// Eventually waits until the condition holds and fails the test with the
// description after Deadline.
func Eventually(t testing.TB, condition func() bool, description string, values ...any) {
	t.Helper()
	deadline := time.Now().Add(Deadline)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for "+description, values...)
		}
		time.Sleep(time.Millisecond)
	}
}
