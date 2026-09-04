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

package rpcs

import (
	"encoding/json"
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

// vmTraceCallResult captures the vmTrace part of a trace_call response.
type vmTraceCallResult struct {
	VmTrace json.RawMessage `json:"vmTrace"`
}

// vmTraceOps mirrors the minimal vmTrace structure needed for assertions.
type vmTraceOps struct {
	Ops []struct {
		Op string          `json:"op"`
		Ex json.RawMessage `json:"ex"`
	} `json:"ops"`
}

// TestVmTrace_TraceCall_HugeMemoryOperandsDoNotExhaustMemory verifies that a
// trace_call with the vmTrace option survives bytecode whose memory operands
// are far beyond any allocatable size. The traced code is executed as init
// code of a contract creation:
//
//	PUSH1 0x00; PUSH1 0x00; PUSH9 0xff..ff; CALLDATACOPY
//	  — zero-length copy to an offset near 2^72: executes successfully
//	    without expanding memory, so the tracer sees a huge (truncated)
//	    offset but must not report or allocate any region for it.
//	PUSH1 0x01; PUSH9 0xff..ff; MSTORE
//	  — memory expansion to ~2^72 overflows the gas calculation and the
//	    opcode faults with out-of-gas; the tracer must not allocate the
//	    32-byte region at that offset either.
//
// The call itself fails with out-of-gas, but the RPC must return a bounded
// vmTrace (no error, no node crash, no multi-gigabyte allocation).
func TestVmTrace_TraceCall_HugeMemoryOperandsDoNotExhaustMemory(t *testing.T) {
	net := tests.StartIntegrationTestNet(t)

	sender := tests.NewAccount()

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	code := []byte{
		0x60, 0x00, // PUSH1 0x00 (size)
		0x60, 0x00, // PUSH1 0x00 (source offset)
		0x68, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // PUSH9 dest offset ~2^72
		0x37,       // CALLDATACOPY (zero-size, succeeds)
		0x60, 0x01, // PUSH1 0x01 (value)
		0x68, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // PUSH9 offset ~2^72
		0x52, // MSTORE (faults with out-of-gas)
	}

	txArgs := map[string]any{
		"from": sender.Address(),
		"gas":  hexutil.Uint64(500_000),
		"data": hexutil.Bytes(code),
	}

	var result vmTraceCallResult
	err = client.Client().Call(&result, "trace_call", txArgs, []string{"vmTrace"}, "latest")
	require.NoError(t, err, "trace_call must not fail on huge memory operands")
	require.NotEmpty(t, result.VmTrace, "vmTrace must be present in the response")

	// The response must stay bounded: no memory region may have been dumped
	// for the huge operands. The full trace of this short program is tiny.
	require.Less(t, len(result.VmTrace), 1<<20,
		"vmTrace response unexpectedly large — a huge memory region may have been recorded")

	var trace vmTraceOps
	require.NoError(t, json.Unmarshal(result.VmTrace, &trace))
	require.NotEmpty(t, trace.Ops, "vmTrace must record the executed opcodes")

	// The faulting MSTORE is the last recorded op and must have no execution
	// result (Ex == null) — in particular no memory region.
	last := trace.Ops[len(trace.Ops)-1]
	require.Equal(t, "MSTORE", last.Op)
	require.Equal(t, "null", string(last.Ex), "faulting MSTORE must have null ex")

	// The node must still be responsive after serving the hostile trace.
	var blockNumber hexutil.Uint64
	require.NoError(t, client.Client().Call(&blockNumber, "eth_blockNumber"),
		"node must remain responsive after the trace")
}
