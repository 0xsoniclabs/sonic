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

package transport

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// recorder collects what an endpoint writes.
type recorder struct{ written []any }

func (r *recorder) write(_ context.Context, v any, _ bool) error {
	r.written = append(r.written, v)
	return nil
}

// FuzzEndpoint feeds arbitrary bytes to the message layer: whatever arrives, it
// answers or ignores it without panicking, and every answer encodes as JSON.
func FuzzEndpoint(f *testing.F) {
	for _, seed := range []string{
		`{"jsonrpc":"2.0","id":1,"method":"t_echo","params":[1,"x"]}`,
		`[{"jsonrpc":"2.0","id":1,"method":"t_nil"},{"jsonrpc":"2.0","method":"t_nil"}]`,
		`[]`, `{}`, `null`, `5`, `[[]]`, `{"jsonrpc":"2.0","id":{},"method":"x"}`,
		`{"jsonrpc":"2.0","id":1,"method":"eth_subscribe","params":["newHeads"]}`,
		`{"jsonrpc":"2.0","id":1,"method":"rpc_modules"}`,
		`{"jsonrpc":"2.0","id":"a","method":"t_two","params":[1`,
	} {
		f.Add([]byte(seed))
	}
	d := dispatch.New(dispatch.DefaultConfig(), dispatch.NewMetrics(nil))
	registerTestMethods(d)
	front := NewFront(d)
	endpoint := NewEndpoint(EndpointConfig{Name: "fuzz", BatchLimit: 8, BatchResponseMax: 1 << 16}, front)

	f.Fuzz(func(t *testing.T, data []byte) {
		if !json.Valid(data) {
			return // the transports answer a parse error before the message layer
		}
		msgs, batch := parseMessages(data)
		rec := &recorder{}
		ctx := context.Background()
		if batch {
			endpoint.handleBatch(ctx, msgs, rec)
		} else {
			endpoint.handleMsg(ctx, msgs[0], rec)
		}
		for _, v := range rec.written {
			if _, err := json.Marshal(v); err != nil {
				t.Fatalf("answer does not encode: %v", err)
			}
		}
	})
}

// FuzzSplitParams checks the parameter splitter against encoding/json.
func FuzzSplitParams(f *testing.F) {
	for _, seed := range []string{``, `null`, `[]`, `[1,2]`, `{}`, `"x"`, `[null,{"a":[1]}]`, `[1,`} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if !json.Valid(data) && len(data) > 0 {
			return // the params of a request are a valid JSON value
		}
		raw, err := dispatch.SplitParams(data)
		if err != nil {
			return
		}
		for _, item := range raw {
			if !json.Valid(item) {
				t.Fatalf("item %q is not JSON", item)
			}
		}
		if len(raw) > 0 {
			var arr []json.RawMessage
			if json.Unmarshal(data, &arr) != nil || len(arr) != len(raw) {
				t.Fatalf("split %q into %d items, json says %v", data, len(raw), len(arr))
			}
		}
	})
}
