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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// gethService is a service of go-ethereum's rpc.Server. The same methods are
// registered with the module's dispatcher; the envelope tests send both the
// same requests and require the same answers (Q5).
type gethService struct{}

func (gethService) Echo(a int, b *string) map[string]any {
	res := map[string]any{"a": a}
	if b != nil {
		res["b"] = *b
	}
	return res
}

func (gethService) Fail() error { return errors.New("boom") }

type dataErr struct{}

func (dataErr) Error() string  { return "revert" }
func (dataErr) ErrorCode() int { return 3 }
func (dataErr) ErrorData() any { return "0x08c379a0" }

func (gethService) Coded() error { return dataErr{} }

func (gethService) Nil() *int { return nil }

func (gethService) Two(a, b int) int { return a + b }

func (gethService) Big() string { return strings.Repeat("x", 100) }

// slowCall is how long t_slow takes; it ignores its context, as a method that
// cannot be cancelled does.
const slowCall = 1500 * time.Millisecond

func (gethService) Slow() int {
	time.Sleep(slowCall)
	return 1
}

func newGethServer(t *testing.T, batchItems, batchBytes int) *httptest.Server {
	t.Helper()
	return newGethServerWithWriteTimeout(t, batchItems, batchBytes, 0)
}

func newGethServerWithWriteTimeout(t *testing.T, batchItems, batchBytes int, writeTimeout time.Duration) *httptest.Server {
	t.Helper()
	srv := rpc.NewServer()
	srv.SetBatchLimits(batchItems, batchBytes)
	require.NoError(t, srv.RegisterName("t", gethService{}))
	ts := httptest.NewUnstartedServer(srv)
	ts.Config.WriteTimeout = writeTimeout
	ts.Start()
	t.Cleanup(func() { ts.Close(); srv.Stop() })
	return ts
}

func registerTestMethods(d *dispatch.Dispatcher) {
	dispatch.Add2(d, "t_echo", func(_ context.Context, a int, b *string) (map[string]any, error) {
		res := map[string]any{"a": a}
		if b != nil {
			res["b"] = *b
		}
		return res, nil
	}, "a", "b?")
	dispatch.Add0(d, "t_fail", func(context.Context) (int, error) { return 0, errors.New("boom") })
	dispatch.Add0(d, "t_coded", func(context.Context) (int, error) { return 0, dataErr{} })
	dispatch.Add0(d, "t_nil", func(context.Context) (*int, error) { return nil, nil })
	dispatch.Add2(d, "t_two", func(_ context.Context, a, b int) (int, error) { return a + b, nil }, "a", "b")
	dispatch.Add0(d, "t_big", func(context.Context) (string, error) { return strings.Repeat("x", 100), nil })
	dispatch.Add0(d, "t_slow", func(context.Context) (int, error) {
		time.Sleep(slowCall)
		return 1, nil
	})
}

func newModuleServer(t *testing.T, batchItems, batchBytes int) string {
	t.Helper()
	return newModuleServerWithTimeouts(t, HTTPConfig{BatchLimit: batchItems, BatchResponseMax: batchBytes})
}

func newModuleServerWithTimeouts(t *testing.T, cfg HTTPConfig) string {
	t.Helper()
	d := dispatch.New(dispatch.DefaultConfig(), dispatch.NewMetrics(nil))
	registerTestMethods(d)
	cfg.Host = "127.0.0.1"
	srv, err := NewHTTPServer(cfg, NewFront(d))
	require.NoError(t, err)
	require.NoError(t, srv.Start())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second+slowCall)
		defer cancel()
		_ = srv.Stop(ctx)
		d.Stop(time.Second)
	})
	return "http://" + srv.Addr()
}

type answer struct {
	status int
	body   string
}

func post(t *testing.T, url, body string) answer {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return answer{resp.StatusCode, strings.TrimSpace(string(data))}
}

// normalize parses a body and drops what differs by design: the wording of the
// -32602 errors that name a parameter, and the code of a crashed handler.
func normalize(t *testing.T, body string) any {
	t.Helper()
	if body == "" {
		return nil
	}
	var v any
	require.NoError(t, json.Unmarshal([]byte(body), &v), body)
	var fix func(any)
	fix = func(v any) {
		switch v := v.(type) {
		case []any:
			for _, e := range v {
				fix(e)
			}
		case map[string]any:
			if e, ok := v["error"].(map[string]any); ok {
				msg, _ := e["message"].(string)
				if e["code"] == float64(-32602) && (strings.HasPrefix(msg, "invalid argument") || strings.HasPrefix(msg, "parameter")) {
					e["message"] = "<invalid parameter>"
				}
			}
		}
	}
	fix(v)
	return v
}

func TestEnvelope_MatchesGeth(t *testing.T) {
	geth := newGethServer(t, 0, 0).URL
	mine := newModuleServer(t, 0, 0)

	requests := map[string]string{
		"call":                     `{"jsonrpc":"2.0","id":1,"method":"t_echo","params":[1,"x"]}`,
		"optional absent":          `{"jsonrpc":"2.0","id":1,"method":"t_echo","params":[1]}`,
		"optional null":            `{"jsonrpc":"2.0","id":1,"method":"t_echo","params":[1,null]}`,
		"no params member":         `{"jsonrpc":"2.0","id":1,"method":"t_nil"}`,
		"params null":              `{"jsonrpc":"2.0","id":1,"method":"t_nil","params":null}`,
		"params empty":             `{"jsonrpc":"2.0","id":1,"method":"t_nil","params":[]}`,
		"params object":            `{"jsonrpc":"2.0","id":1,"method":"t_nil","params":{}}`,
		"params string":            `{"jsonrpc":"2.0","id":1,"method":"t_nil","params":"x"}`,
		"too many":                 `{"jsonrpc":"2.0","id":1,"method":"t_two","params":[1,2,3]}`,
		"missing":                  `{"jsonrpc":"2.0","id":1,"method":"t_two","params":[1]}`,
		"missing all":              `{"jsonrpc":"2.0","id":1,"method":"t_two"}`,
		"wrong type":               `{"jsonrpc":"2.0","id":1,"method":"t_two","params":["a",2]}`,
		"null for required":        `{"jsonrpc":"2.0","id":1,"method":"t_two","params":[null,2]}`,
		"unknown method":           `{"jsonrpc":"2.0","id":1,"method":"t_nope"}`,
		"unknown namespace":        `{"jsonrpc":"2.0","id":1,"method":"x_nope"}`,
		"no underscore":            `{"jsonrpc":"2.0","id":1,"method":"nope"}`,
		"plain error":              `{"jsonrpc":"2.0","id":1,"method":"t_fail"}`,
		"coded error with data":    `{"jsonrpc":"2.0","id":1,"method":"t_coded"}`,
		"string id":                `{"jsonrpc":"2.0","id":"abc","method":"t_nil"}`,
		"float id":                 `{"jsonrpc":"2.0","id":1.5,"method":"t_nil"}`,
		"null id":                  `{"jsonrpc":"2.0","id":null,"method":"t_nil"}`,
		"object id":                `{"jsonrpc":"2.0","id":{},"method":"t_nil"}`,
		"array id":                 `{"jsonrpc":"2.0","id":[],"method":"t_nil"}`,
		"notification":             `{"jsonrpc":"2.0","method":"t_nil"}`,
		"notification unknown":     `{"jsonrpc":"2.0","method":"t_nope"}`,
		"no version":               `{"id":1,"method":"t_nil"}`,
		"wrong version":            `{"jsonrpc":"1.0","id":1,"method":"t_nil"}`,
		"no method":                `{"jsonrpc":"2.0","id":1}`,
		"empty object":             `{}`,
		"number":                   `5`,
		"string":                   `"hello"`,
		"null":                     `null`,
		"response object":          `{"jsonrpc":"2.0","id":1,"result":1}`,
		"subscribe on http":        `{"jsonrpc":"2.0","id":1,"method":"eth_subscribe","params":["newHeads"]}`,
		"subscribe unknown ns":     `{"jsonrpc":"2.0","id":1,"method":"zzz_subscribe","params":["x"]}`,
		"unsubscribe":              `{"jsonrpc":"2.0","id":1,"method":"eth_unsubscribe","params":["0x1"]}`,
		"unsubscribe no param":     `{"jsonrpc":"2.0","id":1,"method":"eth_unsubscribe","params":[]}`,
		"unsubscribe bad":          `{"jsonrpc":"2.0","id":1,"method":"eth_unsubscribe","params":[5]}`,
		"long method":              `{"jsonrpc":"2.0","id":1,"method":"` + strings.Repeat("a", 3000) + `"}`,
		"batch":                    `[{"jsonrpc":"2.0","id":1,"method":"t_nil"},{"jsonrpc":"2.0","id":2,"method":"t_two","params":[1,2]}]`,
		"batch with notification":  `[{"jsonrpc":"2.0","method":"t_nil"},{"jsonrpc":"2.0","id":2,"method":"t_two","params":[1,2]}]`,
		"batch notifications only": `[{"jsonrpc":"2.0","method":"t_nil"},{"jsonrpc":"2.0","method":"t_nil"}]`,
		"batch with garbage":       `[1,{"jsonrpc":"2.0","id":2,"method":"t_nil"},null,"x"]`,
		"batch with errors":        `[{"jsonrpc":"2.0","id":1,"method":"t_fail"},{"jsonrpc":"2.0","id":2,"method":"t_nope"}]`,
		"empty batch":              `[]`,
		"nested batch":             `[[{"jsonrpc":"2.0","id":1,"method":"t_nil"}]]`,
		"malformed":                `{"jsonrpc":"2.0","id":1,"method":`,
		"garbage":                  `not json`,
		"empty body":               ``,
		"whitespace body":          "  \n ",
		"truncated batch":          `[{"jsonrpc":"2.0","id":1,"method":"t_nil"},`,
		"big result":               `{"jsonrpc":"2.0","id":1,"method":"t_big"}`,
	}
	for name, body := range requests {
		t.Run(name, func(t *testing.T) {
			want := post(t, geth, body)
			got := post(t, mine, body)
			require.Equal(t, want.status, got.status)
			require.Equal(t, normalize(t, want.body), normalize(t, got.body), "geth: %s\nmine: %s", want.body, got.body)
		})
	}
}

func TestEnvelope_BatchLimitsMatchGeth(t *testing.T) {
	geth := newGethServer(t, 2, 150).URL
	mine := newModuleServer(t, 2, 150)

	requests := map[string]string{
		"within limits":  `[{"jsonrpc":"2.0","id":1,"method":"t_nil"},{"jsonrpc":"2.0","id":2,"method":"t_nil"}]`,
		"too many items": `[{"jsonrpc":"2.0","id":1,"method":"t_nil"},{"jsonrpc":"2.0","id":2,"method":"t_nil"},{"jsonrpc":"2.0","id":3,"method":"t_nil"}]`,
		"too many items, first is a notification": `[{"jsonrpc":"2.0","method":"t_nil"},{"jsonrpc":"2.0","id":2,"method":"t_nil"},{"jsonrpc":"2.0","id":3,"method":"t_nil"}]`,
		"response too large":                      `[{"jsonrpc":"2.0","id":1,"method":"t_big"},{"jsonrpc":"2.0","id":2,"method":"t_big"}]`,
	}
	for name, body := range requests {
		t.Run(name, func(t *testing.T) {
			want := post(t, geth, body)
			got := post(t, mine, body)
			require.Equal(t, want.status, got.status)
			require.Equal(t, normalize(t, want.body), normalize(t, got.body), "geth: %s\nmine: %s", want.body, got.body)
		})
	}
}

func TestEnvelope_TimeoutsMatchGeth(t *testing.T) {
	const writeTimeout = time.Second
	geth := newGethServerWithWriteTimeout(t, 0, 0, writeTimeout).URL
	timeouts := DefaultTimeouts
	timeouts.Write = writeTimeout
	mine := newModuleServerWithTimeouts(t, HTTPConfig{Timeouts: timeouts})

	requests := map[string]string{
		// A slow notification is not compared: go-ethereum's server races
		// with its own timeout answer there.
		"slow call": `{"jsonrpc":"2.0","id":1,"method":"t_slow"}`,
		"batch with a slow call": `[{"jsonrpc":"2.0","id":1,"method":"t_two","params":[1,2]},` +
			`{"jsonrpc":"2.0","id":2,"method":"t_slow"},{"jsonrpc":"2.0","method":"t_nil"},` +
			`{"jsonrpc":"2.0","id":3,"method":"t_nil"}]`,
		"fast call": `{"jsonrpc":"2.0","id":1,"method":"t_two","params":[1,2]}`,
	}
	for name, body := range requests {
		t.Run(name, func(t *testing.T) {
			start := time.Now()
			want := post(t, geth, body)
			got := post(t, mine, body)
			require.Equal(t, want.status, got.status)
			require.Equal(t, normalize(t, want.body), normalize(t, got.body), "geth: %s\nmine: %s", want.body, got.body)
			// The answer does not wait for the slow method.
			require.Less(t, time.Since(start), 2*slowCall)
		})
	}

	// go-ethereum answers a slow notification with the timeout error too.
	got := post(t, mine, `{"jsonrpc":"2.0","method":"t_slow"}`)
	require.JSONEq(t, `{"jsonrpc":"2.0","error":{"code":-32002,"message":"request timed out"}}`, got.body)
}

func TestHTTP_RequestValidationMatchesGeth(t *testing.T) {
	geth := newGethServer(t, 0, 0).URL
	mine := newModuleServer(t, 0, 0)

	do := func(url, method, contentType string, body io.Reader) answer {
		req, err := http.NewRequest(method, url, body)
		require.NoError(t, err)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		return answer{resp.StatusCode, strings.TrimSpace(string(data))}
	}
	call := `{"jsonrpc":"2.0","id":1,"method":"t_nil"}`

	cases := map[string]func(url string) answer{
		"empty get":     func(u string) answer { return do(u, http.MethodGet, "", nil) },
		"put":           func(u string) answer { return do(u, http.MethodPut, "application/json", strings.NewReader(call)) },
		"delete":        func(u string) answer { return do(u, http.MethodDelete, "application/json", strings.NewReader(call)) },
		"wrong type":    func(u string) answer { return do(u, http.MethodPost, "text/plain", strings.NewReader(call)) },
		"no type":       func(u string) answer { return do(u, http.MethodPost, "", strings.NewReader(call)) },
		"json-rpc type": func(u string) answer { return do(u, http.MethodPost, "application/json-rpc", strings.NewReader(call)) },
		"charset": func(u string) answer {
			return do(u, http.MethodPost, "application/json; charset=utf-8", strings.NewReader(call))
		},
		"options": func(u string) answer { return do(u, http.MethodOptions, "", nil) },
		"oversize body": func(u string) answer {
			return do(u, http.MethodPost, "application/json", bytes.NewReader(bytes.Repeat([]byte(" "), 6*1024*1024)))
		},
		"get with body": func(u string) answer { return do(u, http.MethodGet, "application/json", strings.NewReader(call)) },
	}
	for name, fn := range cases {
		t.Run(name, func(t *testing.T) {
			want, got := fn(geth), fn(mine)
			require.Equal(t, want.status, got.status)
			require.Equal(t, normalize0(want.body), normalize0(got.body))
		})
	}
}

func normalize0(body string) string {
	// A JSON answer is compared by value elsewhere; here plain text and JSON
	// bodies are compared verbatim.
	return strings.TrimSpace(body)
}
