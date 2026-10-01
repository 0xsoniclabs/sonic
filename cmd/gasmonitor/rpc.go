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

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// errNotFound is returned when the node answers with a null result,
// e.g. for a block or receipts it does not have yet.
var errNotFound = errors.New("not found (null result)")

// hexUint64 decodes a 0x-prefixed hex quantity.
type hexUint64 uint64

func (h *hexUint64) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	v, err := strconv.ParseUint(strings.TrimPrefix(s, "0x"), 16, 64)
	if err != nil {
		return fmt.Errorf("invalid hex quantity %q: %w", s, err)
	}
	*h = hexUint64(v)
	return nil
}

func toHex(v uint64) string { return "0x" + strconv.FormatUint(v, 16) }

type rpcBlock struct {
	Number       hexUint64 `json:"number"`
	Hash         string    `json:"hash"`
	GasUsed      hexUint64 `json:"gasUsed"`
	Timestamp    hexUint64 `json:"timestamp"`
	Transactions []string  `json:"transactions"`
}

type rpcReceipt struct {
	TransactionHash   string    `json:"transactionHash"`
	TransactionIndex  hexUint64 `json:"transactionIndex"`
	BlockHash         string    `json:"blockHash"`
	GasUsed           hexUint64 `json:"gasUsed"`
	CumulativeGasUsed hexUint64 `json:"cumulativeGasUsed"`
	Status            hexUint64 `json:"status"`
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      uint64 `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	ID     uint64          `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

func (r *rpcResponse) decode(method string, out any) error {
	if r.Error != nil {
		return fmt.Errorf("%s: rpc error %d: %s", method, r.Error.Code, r.Error.Message)
	}
	if len(r.Result) == 0 || string(r.Result) == "null" {
		return fmt.Errorf("%s: %w", method, errNotFound)
	}
	if err := json.Unmarshal(r.Result, out); err != nil {
		return fmt.Errorf("%s: decoding result: %w", method, err)
	}
	return nil
}

type rpcClient struct {
	url     string
	http    *http.Client
	nextID  atomic.Uint64
	noBatch atomic.Bool // set once the endpoint rejects batch requests
}

func newRPCClient(url string, timeout time.Duration) *rpcClient {
	return &rpcClient{url: url, http: &http.Client{Timeout: timeout}}
}

func (c *rpcClient) post(ctx context.Context, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(data))
	if err != nil {
		return redactErr(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return redactErr(err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decoding response: %w (body: %s)", err, truncate(string(raw), 200))
	}
	return nil
}

func (c *rpcClient) call(ctx context.Context, out any, method string, params ...any) error {
	var resp rpcResponse
	req := rpcRequest{JSONRPC: "2.0", ID: c.nextID.Add(1), Method: method, Params: params}
	if err := c.post(ctx, req, &resp); err != nil {
		return fmt.Errorf("%s: %w", method, err)
	}
	return resp.decode(method, out)
}

type batchCall struct {
	method string
	params []any
	out    any
}

// batch sends all calls in one HTTP request, so that a load-balanced
// endpoint is likely to serve them from the same backend node. If the
// endpoint does not support batching, calls are sent one by one.
func (c *rpcClient) batch(ctx context.Context, calls ...batchCall) error {
	if !c.noBatch.Load() {
		reqs := make([]rpcRequest, len(calls))
		for i, call := range calls {
			reqs[i] = rpcRequest{JSONRPC: "2.0", ID: c.nextID.Add(1), Method: call.method, Params: call.params}
		}
		var resps []rpcResponse
		err := c.post(ctx, reqs, &resps)
		if err == nil && len(resps) == len(calls) {
			byID := make(map[uint64]*rpcResponse, len(resps))
			for i := range resps {
				byID[resps[i].ID] = &resps[i]
			}
			for i, call := range calls {
				resp, ok := byID[reqs[i].ID]
				if !ok {
					return fmt.Errorf("%s: missing response in batch", call.method)
				}
				if err := resp.decode(call.method, call.out); err != nil {
					return err
				}
			}
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// Anything else than a well-formed array answer is taken as
		// "batching not supported"; fall back to single calls for good.
		c.noBatch.Store(true)
	}
	for _, call := range calls {
		if err := c.call(ctx, call.out, call.method, call.params...); err != nil {
			return err
		}
	}
	return nil
}

func (c *rpcClient) blockNumber(ctx context.Context) (uint64, error) {
	var n hexUint64
	err := c.call(ctx, &n, "eth_blockNumber")
	return uint64(n), err
}

// fetchBlock retrieves the block header (with tx hashes) and all receipts of
// the given block.
func (c *rpcClient) fetchBlock(ctx context.Context, number uint64) (*rpcBlock, []rpcReceipt, error) {
	var block rpcBlock
	var receipts []rpcReceipt
	err := c.batch(ctx,
		batchCall{method: "eth_getBlockByNumber", params: []any{toHex(number), false}, out: &block},
		batchCall{method: "eth_getBlockReceipts", params: []any{toHex(number)}, out: &receipts},
	)
	if err != nil {
		return nil, nil, err
	}
	return &block, receipts, nil
}

// redactURL hides everything after the host of an endpoint URL (path, query,
// credentials), which may carry an API key.
func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "<redacted URL>"
	}
	redacted := u.Scheme + "://" + u.Host
	if (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		redacted += "/***"
	}
	return redacted
}

// redactErr redacts the URL that net/http includes in its errors.
func redactErr(err error) error {
	if ue, ok := errors.AsType[*url.Error](err); ok {
		ue.URL = redactURL(ue.URL)
	}
	return err
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
