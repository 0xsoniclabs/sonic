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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAnalyze_DetectsGapAndCumulativeJump(t *testing.T) {
	s := &snapshot{
		Hash:     "0xb",
		GasUsed:  100,
		TxHashes: []string{"0x1", "0x2"},
		Receipts: []rpcReceipt{
			{TransactionHash: "0x1", TransactionIndex: 0, BlockHash: "0xb", GasUsed: 30, CumulativeGasUsed: 30, Status: 1},
			{TransactionHash: "0x2", TransactionIndex: 1, BlockHash: "0xb", GasUsed: 40, CumulativeGasUsed: 90, Status: 0},
		},
	}
	a := analyze(s)
	if a.SumGasUsed != 70 || a.LastCumulative != 90 || a.Gap != 30 || a.cumulativeGap() != 20 {
		t.Fatalf("unexpected analysis: %+v", a)
	}
	if len(a.CumulativeIssues) != 1 || !strings.Contains(a.CumulativeIssues[0], "+20 not covered by any receipt") {
		t.Fatalf("unexpected cumulative issues: %v", a.CumulativeIssues)
	}
	if a.hasOtherIssue() {
		t.Fatalf("unexpected other issues: %v", a.OtherIssues)
	}
}

func TestAnalyze_ConsistentBlockIsClean(t *testing.T) {
	s := &snapshot{
		Hash:     "0xb",
		GasUsed:  70,
		TxHashes: []string{"0x1", "0x2"},
		Receipts: []rpcReceipt{
			{TransactionHash: "0x1", TransactionIndex: 0, GasUsed: 30, CumulativeGasUsed: 30, Status: 1},
			{TransactionHash: "0x2", TransactionIndex: 1, GasUsed: 40, CumulativeGasUsed: 70, Status: 1},
		},
	}
	if a := analyze(s); !a.clean() {
		t.Fatalf("expected clean analysis, got %+v", a)
	}
}

// fakeChain serves blocks 1..head, all with a single 21000 gas transaction.
// Some values differ per query round (one round is a block + receipts query):
// block 3 reports a higher block gasUsed on its head query only, block 4 on
// its second query only, and block 5's receipt reports more gas from its
// second query on.
var (
	fakeBlockGasPerRound   = map[uint64][]uint64{3: {50000, 21000, 21000}, 4: {21000, 60000, 21000}}
	fakeReceiptGasPerRound = map[uint64][]uint64{5: {21000, 30000, 30000}}
)

func fakeValue(perRound map[uint64][]uint64, block uint64, round int) uint64 {
	if rounds := perRound[block]; round < len(rounds) {
		return rounds[round]
	}
	return 21000
}

type fakeChain struct {
	mu      sync.Mutex
	head    uint64
	queries map[uint64]int
}

func (c *fakeChain) handle(req rpcRequest) rpcResponse {
	c.mu.Lock()
	defer c.mu.Unlock()
	resp := rpcResponse{ID: req.ID}
	var result any
	switch req.Method {
	case "eth_blockNumber":
		c.head++ // the chain grows by one block per head poll
		result = toHex(c.head)
	case "eth_getBlockByNumber", "eth_getBlockReceipts":
		var n hexUint64
		_ = n.UnmarshalJSON([]byte(`"` + req.Params[0].(string) + `"`))
		round := c.queries[uint64(n)] / 2
		blockGas := fakeValue(fakeBlockGasPerRound, uint64(n), round)
		receiptGas := toHex(fakeValue(fakeReceiptGasPerRound, uint64(n), round))
		c.queries[uint64(n)]++
		hash := fmt.Sprintf("0x%064x", uint64(n))
		tx := fmt.Sprintf("0x%064x", uint64(n)+1000)
		if req.Method == "eth_getBlockByNumber" {
			result = map[string]any{"number": toHex(uint64(n)), "hash": hash, "gasUsed": toHex(blockGas), "timestamp": "0x1", "transactions": []string{tx}}
		} else {
			result = []map[string]any{{"transactionHash": tx, "transactionIndex": "0x0", "blockHash": hash,
				"gasUsed": receiptGas, "cumulativeGasUsed": receiptGas, "status": "0x1"}}
		}
	default:
		resp.Error = &rpcError{Code: -32601, Message: "method not found"}
		return resp
	}
	resp.Result, _ = json.Marshal(result)
	return resp
}

func (c *fakeChain) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	if strings.HasPrefix(strings.TrimSpace(string(body)), "[") {
		var reqs []rpcRequest
		_ = json.Unmarshal(body, &reqs)
		resps := make([]rpcResponse, len(reqs))
		for i, req := range reqs {
			resps[i] = c.handle(req)
		}
		_ = json.NewEncoder(w).Encode(resps)
		return
	}
	var req rpcRequest
	_ = json.Unmarshal(body, &req)
	_ = json.NewEncoder(w).Encode(c.handle(req))
}

func TestMonitor_ReportsGapAndChangesBetweenDepths(t *testing.T) {
	chain := &fakeChain{queries: map[uint64]int{}}
	server := httptest.NewServer(chain)
	defer server.Close()

	reportPath := filepath.Join(t.TempDir(), "report.md")
	cfg := config{
		rpcURL:     server.URL + "/secret-key",
		reportPath: reportPath,
		depths:     []uint64{2, 4},
		workers:    2,
		poll:       20 * time.Millisecond, // re-checks of one block must not overlap
		timeout:    time.Second,
		retries:    1,
		from:       1,
		maxBlocks:  5,
	}
	var out strings.Builder
	m := newMonitor(cfg, &console{out: &out})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := m.run(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if ctx.Err() != nil {
		t.Fatal("monitor did not finish draining re-checks")
	}

	st := m.stats
	if st.observed != 5 || st.gapBlocks != 1 || st.rechecks != 10 || st.rechecksChanged != 6 || st.changedBlocks != 3 ||
		st.blockGasChangedBlocks != 2 || st.receiptSumChangedBlocks != 1 || st.cumulativeChangedBlocks != 1 || st.otherChangedBlocks != 0 {
		t.Fatalf("unexpected stats: %+v", st)
	}
	wantDepths := map[uint64]depthStats{
		2: {rechecks: 5, vsPrevious: changeCounts{same: 2, blockGas: 2, receiptSum: 1, cumulative: 1},
			vsHead: changeCounts{same: 2, blockGas: 2, receiptSum: 1, cumulative: 1}},
		4: {rechecks: 5, vsPrevious: changeCounts{same: 4, blockGas: 1},
			vsHead: changeCounts{same: 3, blockGas: 1, receiptSum: 1, cumulative: 1}},
	}
	for d, want := range wantDepths {
		if got := *st.byDepth[d]; got != want {
			t.Errorf("depth %d stats: got %+v, want %+v", d, got, want)
		}
	}
	if len(m.records) != 3 || !m.records[3].Changed || !m.records[4].Changed || !m.records[5].Changed {
		t.Fatalf("expected blocks 3, 4 and 5 to be kept as changed, got %v", m.records)
	}

	// Console events list all queries of the block, each comparison on its
	// own indented lines.
	for _, want := range []string{
		"block 4: CHANGED at re-check depth 4\n" +
			"    head (lag ",
		"    re-check depth 2    receipts Σ 21000 | block gasUsed 60000 (gap +39000) | last cumulativeGasUsed 21000 (gap +0)\n" +
			"        vs previous query:\n" +
			"            block gasUsed            CHANGED    21000 -> 60000 (+39000)\n" +
			"            receipts Σ               unchanged  21000\n" +
			"            last cumulativeGasUsed   unchanged  21000\n" +
			"    re-check depth 4    receipts Σ 21000 | block gasUsed 21000 (gap +0) | last cumulativeGasUsed 21000 (gap +0)\n" +
			"        vs previous query:\n" +
			"            block gasUsed            CHANGED    60000 -> 21000 (-39000)\n" +
			"            receipts Σ               unchanged  21000\n" +
			"            last cumulativeGasUsed   unchanged  21000\n" +
			"        vs head query: same values\n",
		"block 5: CHANGED at re-check depth 2\n",
		"    re-check depth 2    receipts Σ 30000 | block gasUsed 21000 (gap -9000) | last cumulativeGasUsed 30000 (gap +0)\n" +
			"        vs previous query:\n" +
			"            block gasUsed            unchanged  21000\n" +
			"            receipts Σ               CHANGED    21000 -> 30000 (+9000)\n" +
			"            last cumulativeGasUsed   CHANGED    21000 -> 30000 (+9000)\n" +
			"            receipt #0 gasUsed 21000 -> 30000\n" +
			"            receipt #0 cumulativeGasUsed 21000 -> 30000\n",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("console output does not contain %q:\n%s", want, out.String())
		}
	}

	if strings.Contains(out.String(), "secret-key") {
		t.Errorf("console output leaks the URL secret:\n%s", out.String())
	}

	if err := m.writeReport(); err != nil {
		t.Fatal(err)
	}
	report, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Change deeper in history: YES.** 3 block(s)",
		"  - reported block `gasUsed`: **2 block(s)**",
		"  - receipts Σ: **1 block(s)**",
		"#### Block 3: changed reported block `gasUsed`",
		"#### Block 5: changed receipts Σ, last `cumulativeGasUsed`",
		"  - vs previous query: same values\n" +
			"  - vs head query:\n" +
			"    - block gasUsed **CHANGED** 50000 → 21000 (-29000)\n" +
			"    - receipts Σ unchanged (21000)\n",
		"| Block | Lag | Receipts Σ |",
		"| 2 | 5 | 2 | **2** | **1** | **1** | 0 | 0 |", // depth 2, vs previous and vs head
		"| 4 | 5 | 4 | **1** | 0 | 0 | 0 | 0 |",         // depth 4, vs previous
		"| 4 | 5 | 3 | **1** | **1** | **1** | 0 | 0 |", // depth 4, vs head
		"+29000",
		"| RPC endpoint | `" + server.URL + "/***` |",
	} {
		if !strings.Contains(string(report), want) {
			t.Errorf("report does not contain %q:\n%s", want, report)
		}
	}
	if strings.Contains(string(report), "secret-key") {
		t.Errorf("report leaks the URL secret:\n%s", report)
	}
}
