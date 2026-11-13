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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

// gethService serves the same parameter shapes as the methods registered in
// newShapesDispatcher, through go-ethereum's server, which is the reference
// for decoding and counting parameters.
type gethService struct{}

func (gethService) Mixed(ctx context.Context, number uint64, label *string) (string, error) {
	return "ok", nil
}

func (gethService) Hex(ctx context.Context, quantity hexutil.Uint64, address *common.Address) (string, error) {
	return "ok", nil
}

func (gethService) Slice(ctx context.Context, values []float64) (string, error) {
	return "ok", nil
}

func (gethService) None(ctx context.Context) (string, error) {
	return "ok", nil
}

type mixedParams struct {
	Number uint64
	Label  *string
}

type hexParams struct {
	Quantity hexutil.Uint64
	Address  *common.Address
}

type sliceParams struct {
	Values []float64
}

func ok[Params any](context.Context, Params) (string, error) {
	return "ok", nil
}

func newShapesDispatcher() *Dispatcher {
	dispatcher := New(DefaultConfig(), prometheus.NewRegistry())
	Add(dispatcher, "test_mixed", ok[mixedParams])
	Add(dispatcher, "test_hex", ok[hexParams])
	Add(dispatcher, "test_slice", ok[sliceParams])
	Add(dispatcher, "test_none", ok[struct{}])
	return dispatcher
}

func TestParams_DecodeAsGoEthereumServer(t *testing.T) {
	reference := rpc.NewServer()
	require.NoError(t, reference.RegisterName("test", gethService{}))
	dispatcher := newShapesDispatcher()

	tests := map[string]struct {
		method string
		params string // absent if empty
	}{
		"absent":                   {"test_mixed", ""},
		"null":                     {"test_mixed", "null"},
		"empty":                    {"test_mixed", "[]"},
		"required only":            {"test_mixed", "[1]"},
		"all":                      {"test_mixed", `[1,"a"]`},
		"too many":                 {"test_mixed", `[1,"a",2]`},
		"wrong type":               {"test_mixed", `["x"]`},
		"null required":            {"test_mixed", "[null]"},
		"null optional":            {"test_mixed", "[1,null]"},
		"object":                   {"test_mixed", "{}"},
		"number":                   {"test_mixed", "5"},
		"string":                   {"test_mixed", `"5"`},
		"hex quantity":             {"test_hex", `["0x1"]`},
		"hex null required":        {"test_hex", `[null]`},
		"hex not a string":         {"test_hex", `[1]`},
		"hex leading zero":         {"test_hex", `["0x01"]`},
		"hex bad address":          {"test_hex", `["0x1","0x12"]`},
		"slice missing":            {"test_slice", `[]`},
		"slice null":               {"test_slice", `[null]`},
		"slice given":              {"test_slice", `[[1.5]]`},
		"none with none":           {"test_none", `[]`},
		"none with one":            {"test_none", `[1]`},
		"nested error in element":  {"test_slice", `[["a"]]`},
		"second argument mismatch": {"test_hex", `["0x1",5]`},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			want := callGoEthereum(t, reference, test.method, test.params)
			got := dispatcher.Serve(context.Background(), test.method, json.RawMessage(test.params))
			require.Equal(t, want.Error, got.Error)
			require.Equal(t, string(want.Result), string(got.Result))
		})
	}
}

// callGoEthereum calls a method on go-ethereum's server and answers its
// response.
func callGoEthereum(t *testing.T, server *rpc.Server, method, params string) Response {
	t.Helper()
	body := `{"jsonrpc":"2.0","id":1,"method":"` + method + `"`
	if params != "" {
		body += `,"params":` + params
	}
	body += "}"
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	request.Header.Set("content-type", "application/json")
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	var response struct {
		Result json.RawMessage
		Error  *Error
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	return Response{Result: response.Result, Error: response.Error}
}

func TestAdd_RefusesShapesItCannotServe(t *testing.T) {
	type unexported struct {
		Number uint64
		label  *string //nolint:unused // the shape under test
	}
	type requiredAfterOptional struct {
		Label  *string
		Number uint64
	}
	tests := map[string]func(*Dispatcher){
		"not a struct":            func(d *Dispatcher) { Add(d, "test_a", ok[uint64]) },
		"unexported field":        func(d *Dispatcher) { Add(d, "test_a", ok[unexported]) },
		"required after optional": func(d *Dispatcher) { Add(d, "test_a", ok[requiredAfterOptional]) },
		"registered twice": func(d *Dispatcher) {
			Add(d, "test_a", ok[struct{}])
			Add(d, "test_a", ok[struct{}])
		},
	}
	for name, register := range tests {
		t.Run(name, func(t *testing.T) {
			dispatcher := New(DefaultConfig(), prometheus.NewRegistry())
			require.Panics(t, func() { register(dispatcher) })
		})
	}
}

func TestAdd_AcceptsTrailingOptionalFields(t *testing.T) {
	type params struct {
		Number uint64
		First  *string
		Second *uint64
	}
	dispatcher := New(DefaultConfig(), prometheus.NewRegistry())
	var got params
	Add(dispatcher, "test_a", func(ctx context.Context, p params) (string, error) {
		got = p
		return "ok", nil
	})
	response := dispatcher.Serve(context.Background(), "test_a", json.RawMessage(`[7,"x"]`))
	require.Nil(t, response.Error)
	require.Equal(t, uint64(7), got.Number)
	require.Equal(t, "x", *got.First)
	require.Nil(t, got.Second)
}

// The positional parameters of eth_getBalance, as a handler would take them.
type balanceParams struct {
	Address common.Address
	Block   *string
}

var balanceRequest = json.RawMessage(`["0x0000000000000000000000000000000000000001","latest"]`)

func BenchmarkParams_TypedDecoding(b *testing.B) {
	layout := newParamsLayout(reflect.TypeFor[balanceParams]())
	for b.Loop() {
		var params balanceParams
		if err := layout.decode(balanceRequest, &params); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParams_HandDecoding(b *testing.B) {
	for b.Loop() {
		var raw []json.RawMessage
		if err := json.Unmarshal(balanceRequest, &raw); err != nil {
			b.Fatal(err)
		}
		var params balanceParams
		if err := json.Unmarshal(raw[0], &params.Address); err != nil {
			b.Fatal(err)
		}
		if len(raw) > 1 {
			if err := json.Unmarshal(raw[1], &params.Block); err != nil {
				b.Fatal(err)
			}
		}
	}
}
