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
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// echoRouter answers a call with its parameters, and an error for a method
// named fail.
type echoRouter struct{}

func (echoRouter) Serve(_ context.Context, _ Kind, call dispatch.Call) (json.RawMessage, error) {
	if call.Method == "fail" {
		return nil, errors.New("fail")
	}
	if len(call.Params) == 0 {
		return json.RawMessage("null"), nil
	}
	return call.Params, nil
}

type collector struct {
	mu   sync.Mutex
	msgs [][]byte
}

func (c *collector) write(msg []byte, _ bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.msgs = append(c.msgs, msg)
}

// FuzzFrame is the codec fuzz target of Q5: any frame is answered without a
// panic, with at most one write, and every write is valid JSON.
func FuzzFrame(f *testing.F) {
	for _, seed := range []string{
		`{"jsonrpc":"2.0","id":1,"method":"m","params":[1]}`,
		`{"jsonrpc":"2.0","method":"m"}`,
		`[{"jsonrpc":"2.0","id":"a","method":"fail"},{"jsonrpc":"2.0","id":2,"method":"m","params":{"a":1}}]`,
		`[]`, `[1,2,3]`, `{}`, `null`, `"x"`, `{"jsonrpc":"1.0","id":1,"method":"m"}`,
		`{"jsonrpc":"2.0","id":{"a":1},"method":"m"}`, `{"jsonrpc":"2.0","id":1,"result":1}`,
	} {
		f.Add([]byte(seed))
	}
	frames := frames{router: echoRouter{}, kind: HTTP, limits: Limits{BatchRequestLimit: 10, BatchResponseMaxSize: 1 << 16}}
	f.Fuzz(func(t *testing.T, data []byte) {
		if !json.Valid(data) {
			return // the transports answer an undecodable frame before here
		}
		var c collector
		frames.serveFrame(context.Background(), json.RawMessage(data), &c, time.Minute, nil)
		if len(c.msgs) > 1 {
			t.Fatalf("%d writes for one frame", len(c.msgs))
		}
		for _, msg := range c.msgs {
			if !json.Valid(msg) {
				t.Fatalf("invalid answer %q to %q", msg, data)
			}
		}
	})
}
