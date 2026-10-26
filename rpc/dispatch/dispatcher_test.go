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
	"math"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

// testDispatcher is a dispatcher with a registry of its own.
type testDispatcher struct {
	*Dispatcher
	registry *prometheus.Registry
}

func newTestDispatcher(t *testing.T, mutate ...func(*Config)) testDispatcher {
	t.Helper()
	config := DefaultConfig()
	for _, change := range mutate {
		change(&config)
	}
	registry := prometheus.NewRegistry()
	dispatcher := New(config, NewMetrics(registry))
	t.Cleanup(dispatcher.Stop)
	return testDispatcher{Dispatcher: dispatcher, registry: registry}
}

// scrape adds up the series of a metric of the dispatcher that carry the
// given label values, given as name and value pairs.
func (dispatcher testDispatcher) scrape(t *testing.T, name string, labels ...string) float64 {
	t.Helper()
	families, err := dispatcher.registry.Gather()
	require.NoError(t, err)
	var total float64
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			if matchesLabels(metric.GetLabel(), labels) {
				total += metric.GetCounter().GetValue() + metric.GetGauge().GetValue() + float64(metric.GetHistogram().GetSampleCount())
			}
		}
	}
	return total
}

// series counts the series of a metric.
func (dispatcher testDispatcher) series(t *testing.T, name string) int {
	t.Helper()
	families, err := dispatcher.registry.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() == name {
			return len(family.GetMetric())
		}
	}
	return 0
}

func matchesLabels[Pair interface {
	GetName() string
	GetValue() string
}](pairs []Pair, labels []string) bool {
	for index := 0; index+1 < len(labels); index += 2 {
		found := false
		for _, pair := range pairs {
			if pair.GetName() == labels[index] && pair.GetValue() == labels[index+1] {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func echo(_ context.Context, arguments Arguments) (any, error) {
	return arguments, nil
}

func serve(t *testing.T, dispatcher testDispatcher, method, parameters string) (string, *Error) {
	t.Helper()
	result, err := dispatcher.Serve(context.Background(), method, json.RawMessage(parameters))
	return string(result), err
}

func TestServe_UnknownMethodAnswersMethodNotFound(t *testing.T) {
	dispatcher := newTestDispatcher(t)
	_, err := serve(t, dispatcher, "eth_nothing", "[]")
	require.Equal(t, CodeMethodNotFound, err.Code)
	require.Equal(t, "the method eth_nothing does not exist/is not available", err.Message)
	require.Equal(t, 1.0, dispatcher.scrape(t, "rpc_requests_total", "method", OtherMethod, "result", ResultInvalid))
}

func TestServe_DecodesParametersBeforeTheHandler(t *testing.T) {
	reached := 0
	dispatcher := newTestDispatcher(t)
	dispatcher.Add("test_echo", func(ctx context.Context, arguments Arguments) (any, error) {
		reached++
		return echo(ctx, arguments)
	}, Required[uint64]("number"), Optional[string]("label"))

	tests := map[string]struct {
		parameters string
		result     string
		code       int
		message    string
	}{
		"both":                {parameters: `[1, "x"]`, result: `[1,"x"]`},
		"optional absent":     {parameters: `[1]`, result: `[1,null]`},
		"optional null":       {parameters: `[1, null]`, result: `[1,null]`},
		"absent array":        {parameters: ``, code: CodeInvalidParameters, message: "missing value for required argument 0"},
		"null array":          {parameters: `null`, code: CodeInvalidParameters, message: "missing value for required argument 0"},
		"required null":       {parameters: `[null]`, result: `[0,null]`},
		"too many":            {parameters: `[1, "x", 2]`, code: CodeInvalidParameters, message: "too many arguments, want at most 2"},
		"not an array":        {parameters: `{"number": 1}`, code: CodeInvalidParameters, message: "non-array args"},
		"undecodable":         {parameters: `["one"]`, code: CodeInvalidParameters, message: "parameter 0 (number): json: cannot unmarshal string into Go value of type uint64"},
		"undecodable second":  {parameters: `[1, 2]`, code: CodeInvalidParameters, message: "parameter 1 (label): json: cannot unmarshal number into Go value of type string"},
		"empty array":         {parameters: `[]`, code: CodeInvalidParameters, message: "missing value for required argument 0"},
		"whitespace array":    {parameters: ` [ 1 ] `, result: `[1,null]`},
		"required and absent": {parameters: `[]`, code: CodeInvalidParameters, message: "missing value for required argument 0"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			before := reached
			result, err := serve(t, dispatcher, "test_echo", test.parameters)
			if test.code != 0 {
				require.NotNil(t, err)
				require.Equal(t, test.code, err.Code)
				require.Equal(t, test.message, err.Message)
				require.Equal(t, before, reached, "an undecodable call reached the handler")
				return
			}
			require.Nil(t, err)
			require.Equal(t, test.result, result)
			require.Equal(t, before+1, reached)
		})
	}
}

func TestServe_RefusesTooManyParametersBeforeDecodingThem(t *testing.T) {
	dispatcher := newTestDispatcher(t)
	dispatcher.Add("test_one", echo, Required[uint64]("number"))
	// The second element is malformed only after the limit; the count fails first.
	_, err := serve(t, dispatcher, "test_one", `[1, "x"]`)
	require.Equal(t, "too many arguments, want at most 1", err.Message)
}

func TestServe_AppliesTheRequestDeadline(t *testing.T) {
	dispatcher := newTestDispatcher(t, func(config *Config) { config.RequestTimeout = 3 * time.Second })
	var deadline time.Time
	dispatcher.Add("test_deadline", func(ctx context.Context, _ Arguments) (any, error) {
		deadline, _ = ctx.Deadline()
		return nil, nil
	})
	start := time.Now()
	_, err := serve(t, dispatcher, "test_deadline", "[]")
	require.Nil(t, err)
	require.WithinDuration(t, start.Add(3*time.Second), deadline, time.Second)
}

func TestServe_RecoversAPanicAndCountsIt(t *testing.T) {
	dispatcher := newTestDispatcher(t)
	dispatcher.Add("test_panic", func(context.Context, Arguments) (any, error) {
		panic("boom")
	})
	dispatcher.Add("test_fine", func(context.Context, Arguments) (any, error) { return "fine", nil })
	before := dispatcher.scrape(t, "rpc_panics_total", "method", "test_panic")

	_, err := serve(t, dispatcher, "test_panic", "[]")
	require.Equal(t, CodeInternalError, err.Code)
	require.Equal(t, "method handler crashed", err.Message)
	require.Equal(t, before+1, dispatcher.scrape(t, "rpc_panics_total", "method", "test_panic"))
	require.Equal(t, 1.0, dispatcher.scrape(t, "rpc_requests_total", "method", "test_panic", "result", ResultPanic))

	result, err := serve(t, dispatcher, "test_fine", "[]")
	require.Nil(t, err)
	require.Equal(t, `"fine"`, result)
}

func TestServe_RecoversAPanicWhileEncoding(t *testing.T) {
	dispatcher := newTestDispatcher(t)
	dispatcher.Add("test_marshal", func(context.Context, Arguments) (any, error) { return panickingMarshaler{}, nil })
	_, err := serve(t, dispatcher, "test_marshal", "[]")
	require.Equal(t, ErrPanic, err)
}

type panickingMarshaler struct{}

func (panickingMarshaler) MarshalJSON() ([]byte, error) { panic("encoding") }

func TestServe_EncodedResultPassesUnchanged(t *testing.T) {
	dispatcher := newTestDispatcher(t)
	dispatcher.Add("test_encoded", func(context.Context, Arguments) (any, error) {
		return Encoded(`{ "spaced" : 1 }`), nil
	})
	dispatcher.Add("test_raw", func(context.Context, Arguments) (any, error) {
		return json.RawMessage(`{ "spaced" : 1 }`), nil
	})
	result, err := serve(t, dispatcher, "test_encoded", "[]")
	require.Nil(t, err)
	require.Equal(t, `{ "spaced" : 1 }`, result)

	result, err = serve(t, dispatcher, "test_raw", "[]")
	require.Nil(t, err)
	require.Equal(t, `{"spaced":1}`, result, "a json.RawMessage is compacted again")
}

type codedError struct{}

func (codedError) Error() string  { return "coded" }
func (codedError) ErrorCode() int { return 3 }
func (codedError) ErrorData() any { return "0x01" }

func TestServe_KeepsTheCodeAndDataOfAHandlerError(t *testing.T) {
	dispatcher := newTestDispatcher(t)
	dispatcher.Add("test_coded", func(context.Context, Arguments) (any, error) { return nil, codedError{} })
	dispatcher.Add("test_plain", func(context.Context, Arguments) (any, error) { return nil, errors.New("plain") })

	_, err := serve(t, dispatcher, "test_coded", "[]")
	require.Equal(t, &Error{Code: 3, Message: "coded", Data: "0x01"}, err)
	_, err = serve(t, dispatcher, "test_plain", "[]")
	require.Equal(t, &Error{Code: CodeDefault, Message: "plain"}, err)
}

func TestServe_AnUnencodableResultIsAnInternalError(t *testing.T) {
	dispatcher := newTestDispatcher(t)
	dispatcher.Add("test_nan", func(context.Context, Arguments) (any, error) { return math.NaN(), nil })
	_, err := serve(t, dispatcher, "test_nan", "[]")
	require.Equal(t, CodeInternalError, err.Code)
	require.Contains(t, err.Message, "unsupported value")
}

func TestServe_CountsEveryCallWithItsResult(t *testing.T) {
	dispatcher := newTestDispatcher(t)
	dispatcher.Add("test_echo", echo, Required[uint64]("number"))
	dispatcher.Add("test_fail", func(context.Context, Arguments) (any, error) { return nil, errors.New("no") })

	for _, call := range [][2]string{
		{"test_echo", "[1]"}, {"test_echo", `["x"]`}, {"test_fail", "[]"},
		{"client_chosen_name", "[]"}, {"another_client_chosen_name", "[]"},
	} {
		_, _ = serve(t, dispatcher, call[0], call[1])
	}

	requests := "rpc_requests_total"
	require.Equal(t, 1.0, dispatcher.scrape(t, requests, "method", "test_echo", "result", ResultSuccess))
	require.Equal(t, 1.0, dispatcher.scrape(t, requests, "method", "test_echo", "result", ResultInvalid))
	require.Equal(t, 1.0, dispatcher.scrape(t, requests, "method", "test_fail", "result", ResultError))
	require.Equal(t, 2.0, dispatcher.scrape(t, requests, "method", OtherMethod, "result", ResultInvalid))
	require.Equal(t, 4, dispatcher.series(t, requests), "an unknown name adds no label value")
	require.Equal(t, 1, dispatcher.series(t, "rpc_response_bytes"))
}

func TestForward_IsCountedUnderOneLabel(t *testing.T) {
	dispatcher := newTestDispatcher(t)
	var forwarded []string
	forward := func(_ context.Context, name string, parameters json.RawMessage) (json.RawMessage, error) {
		forwarded = append(forwarded, name+string(parameters))
		return json.RawMessage(`"ok"`), nil
	}
	result, err := dispatcher.Forward(context.Background(), "dag_getHeads", json.RawMessage(`["latest"]`), forward)
	require.Nil(t, err)
	require.Equal(t, `"ok"`, string(result))
	require.Equal(t, []string{`dag_getHeads["latest"]`}, forwarded)
	require.Equal(t, 1.0, dispatcher.scrape(t, "rpc_requests_total", "method", ForwardedMethod, "result", ResultSuccess))
}

func TestAdd_RefusesAMethodTwice(t *testing.T) {
	dispatcher := newTestDispatcher(t)
	dispatcher.Add("test_echo", echo)
	require.Panics(t, func() { dispatcher.Add("test_echo", echo) })
	require.True(t, dispatcher.Has("test_echo"))
	require.Equal(t, []string{"test_echo"}, dispatcher.Methods())
}

func TestNewMetrics_SharesMetricsRegisteredBefore(t *testing.T) {
	dispatcher := newTestDispatcher(t)
	second := NewMetrics(dispatcher.registry)
	second.Inflight.Inc()
	require.Equal(t, 1.0, dispatcher.scrape(t, "rpc_inflight"))
}

func TestToError_ConvertsLikeGoEthereum(t *testing.T) {
	require.Equal(t, &Error{Code: CodeDefault, Message: "plain"}, ToError(errors.New("plain")))
	known := &Error{Code: 1, Message: "known"}
	require.Same(t, known, ToError(known))
	wrapped := errors.Join(codedError{})
	require.Equal(t, CodeDefault, ToError(wrapped).Code, "a wrapped code is not unwrapped, as in go-ethereum")
}
