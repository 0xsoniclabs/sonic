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

package rpc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

// allocatedBy returns the bytes the process allocated while f ran.
func allocatedBy(f func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	f()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// V9: no request allocates in proportion to a number the client chose before
// that number is checked against its bound.
func TestValidation_NeverAllocatesForClientChosenNumbers(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(3))
	module, addr, _ := startModule(t, fc, func(c *rpc.Config) {
		c.HTTP.BatchLimit = 1000
		c.HTTP.BatchResponseMax = 25 << 20
	})
	const budget = 8 << 20

	hugeNumber := `"0xffffffffffffffff"`
	hugeCount := `4294967295`
	calls := map[string]string{
		"eth_feeHistory":                          `[` + hugeNumber + `,"latest",[]]`,
		"eth_getLogs":                             `[{"fromBlock":"0x0","toBlock":` + hugeNumber + `}]`,
		"eth_newFilter":                           `[{"fromBlock":"0x0","toBlock":` + hugeNumber + `}]`,
		"eth_getBlockByNumber":                    `[` + hugeNumber + `,true]`,
		"eth_getBlockReceipts":                    `[` + hugeNumber + `]`,
		"eth_getBalance":                          `["0x0000000000000000000000000000000000000001",` + hugeNumber + `]`,
		"eth_getStorageAt":                        `["0x0000000000000000000000000000000000000001","0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","latest"]`,
		"eth_estimateGas":                         `[{"gas":` + hugeNumber + `}]`,
		"eth_call":                                `[{"gas":` + hugeNumber + `,"input":"0x"}]`,
		"eth_createAccessList":                    `[{"gas":` + hugeNumber + `}]`,
		"eth_simulateV1":                          `[{"blockStateCalls":[{"blockOverrides":{"number":` + hugeNumber + `}}]}]`,
		"trace_filter":                            `[{"fromBlock":"0x0","toBlock":` + hugeNumber + `,"count":` + hugeCount + `,"after":` + hugeCount + `}]`,
		"trace_block":                             `[` + hugeNumber + `]`,
		"debug_getRawBlock":                       `[` + hugeNumber + `]`,
		"debug_traceBlockByNumber":                `[` + hugeNumber + `]`,
		"debug_getBadBlocks":                      `[]`,
		"txpool_contentFrom":                      `["0x0000000000000000000000000000000000000001"]`,
		"eth_getTransactionByBlockNumberAndIndex": `[` + hugeNumber + `,` + hugeNumber + `]`,
		"eth_getBlockTransactionCountByNumber":    `[` + hugeNumber + `]`,
	}
	for method, params := range calls {
		if !module.Dispatcher().Has(method) {
			continue
		}
		var err error
		bytesAllocated := allocatedBy(func() {
			_, rerr := module.Dispatcher().Serve(context.Background(), method, json.RawMessage(params))
			if rerr != nil {
				err = rerr
			}
		})
		_ = err // an error is the expected outcome; only the allocation is checked
		require.Less(t, bytesAllocated, uint64(budget), "%s allocated %d bytes", method, bytesAllocated)
	}

	// Batches are counted before any item is parsed, bodies before the batch.
	var batch bytes.Buffer
	batch.WriteByte('[')
	for i := range 100_000 {
		if i > 0 {
			batch.WriteByte(',')
		}
		batch.WriteString(`{"jsonrpc":"2.0","id":1,"method":"eth_chainId"}`)
	}
	batch.WriteByte(']')
	var status int
	var body string
	bytesAllocated := allocatedBy(func() {
		resp, err := http.Post("http://"+addr, "application/json", bytes.NewReader(batch.Bytes()))
		require.NoError(t, err)
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		status, body = resp.StatusCode, string(data)
	})
	require.True(t, status == http.StatusOK || status == http.StatusRequestEntityTooLarge, "status %d", status)
	require.True(t, strings.Contains(body, "batch too large") || status == http.StatusRequestEntityTooLarge, body[:min(len(body), 200)])
	// The allocation follows the bytes the client sent, not the number of items
	// it claims to be entitled to; the race detector multiplies it.
	require.Less(t, bytesAllocated, uint64(64*batch.Len()))
}
