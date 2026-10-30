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
	"encoding/json"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/contracts/indexed_logs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/rpc"
	req "github.com/stretchr/testify/require"
)

// TestCallTracer_WithLog_ReportsBlockLevelLogIndexOfReceipts checks that the
// call tracer reports the same block-level log index as the receipts for every
// log, including logs of transactions that share a block with other
// transactions emitting logs.
func TestCallTracer_WithLog_ReportsBlockLevelLogIndexOfReceipts(t *testing.T) {
	session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())
	t.Parallel()

	client, err := session.GetClient()
	req.NoError(t, err)
	defer client.Close()
	rpcClient := client.Client()

	contract, receipt, err := DeployContract(session, indexed_logs.DeployIndexedLogs)
	req.NoError(t, err)
	req.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// All transactions are proposed together in a single event, which places
	// them in the same block. The number of logs per transaction varies,
	// including transactions without any log.
	logsPerTx := []int64{3, 0, 1, 5, 2, 0, 4}
	accounts := MakeAccountsWithBalance(t, session, len(logsPerTx), big.NewInt(1e18))
	txs := make([]*types.Transaction, len(logsPerTx))
	for i, n := range logsPerTx {
		opts, err := session.GetTransactOptions(accounts[i])
		req.NoError(t, err)
		opts.NoSend = true
		txs[i], err = contract.EmitLogs(opts, big.NewInt(n))
		req.NoError(t, err)
	}
	hashes, err := session.ForceEmitAll(t.Context(), txs)
	req.NoError(t, err)
	receipts, err := session.GetReceipts(hashes)
	req.NoError(t, err)

	wantLogs := map[common.Hash][]*types.Log{}
	hasOffsetLogIndex := false
	for i, receipt := range receipts {
		req.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
		req.Equal(t, receipts[0].BlockNumber, receipt.BlockNumber,
			"all transactions should be included in the same block")
		req.Len(t, receipt.Logs, int(logsPerTx[i]))
		wantLogs[receipt.TxHash] = receipt.Logs
		// Only some index in the block starting beyond 0 shows that the tracer
		// reports block-level and not transaction-level indexes.
		hasOffsetLogIndex = hasOffsetLogIndex ||
			(len(receipt.Logs) > 0 && receipt.Logs[0].Index > 0)
	}
	req.True(t, hasOffsetLogIndex,
		"expected at least one transaction sharing its block with earlier logs")
	blockNumber := receipts[0].BlockNumber.Int64()

	t.Run("debug_traceTransaction", func(t *testing.T) {
		for _, receipt := range receipts {
			trace := callRPC[json.RawMessage](t, rpcClient,
				"debug_traceTransaction", receipt.TxHash, callTracerWithLogConfig())
			checkTraceLogsMatchReceiptLogs(t, trace, wantLogs[receipt.TxHash])
		}
	})

	t.Run("debug_traceBlockByNumber", func(t *testing.T) {
		results := callRPC[[]blockTraceResult](t, rpcClient, "debug_traceBlockByNumber",
			rpc.BlockNumber(blockNumber), callTracerWithLogConfig())
		traced := 0
		for _, result := range results {
			want, found := wantLogs[result.TxHash]
			if !found {
				continue // transaction not issued by this test
			}
			req.Empty(t, result.Error)
			checkTraceLogsMatchReceiptLogs(t, result.Result, want)
			traced++
		}
		req.Equal(t, len(txs), traced)
	})
}

func callTracerWithLogConfig() *ethapi.TraceCallConfig {
	tracer := "callTracer"
	return &ethapi.TraceCallConfig{
		TraceConfig: tracers.TraceConfig{
			Tracer:       &tracer,
			TracerConfig: json.RawMessage(`{"withLog": true}`),
		},
	}
}

// tracedLog is the subset of the logs reported by the call tracer needed here.
type tracedLog struct {
	Address common.Address `json:"address"`
	Topics  []common.Hash  `json:"topics"`
	Data    hexutil.Bytes  `json:"data"`
	Index   hexutil.Uint   `json:"index"`
}

// tracedCall is the subset of a call frame reported by the call tracer needed
// here.
type tracedCall struct {
	Logs  []tracedLog  `json:"logs"`
	Calls []tracedCall `json:"calls"`
}

// allLogs returns the logs of the frame and its sub-calls in a depth-first order.
func (c tracedCall) allLogs() []tracedLog {
	res := append([]tracedLog(nil), c.Logs...)
	for _, call := range c.Calls {
		res = append(res, call.allLogs()...)
	}
	return res
}

func checkTraceLogsMatchReceiptLogs(t *testing.T, trace json.RawMessage, want []*types.Log) {
	t.Helper()
	require := req.New(t)

	var frame tracedCall
	require.NoError(json.Unmarshal(trace, &frame))

	got := frame.allLogs()
	require.Len(got, len(want))
	for i, log := range want {
		require.Equal(log.Address, got[i].Address, "address of log %d", i)
		require.Equal(log.Topics, got[i].Topics, "topics of log %d", i)
		require.Equal(log.Data, []byte(got[i].Data), "data of log %d", i)
		require.Equal(log.Index, uint(got[i].Index),
			"index of log %d differs from the one in the receipt", i)
	}
}
