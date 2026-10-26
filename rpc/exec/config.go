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

package exec

import "time"

// Config holds the semantic limits of the methods. Several namespaces apply
// the same ones, so they are one struct. The knobs keep the names of the
// node configuration they come from.
type Config struct {
	// RPCGasCap caps the gas of eth_call, eth_estimateGas and the traces.
	RPCGasCap uint64
	// RPCEVMTimeout cancels an EVM run; only go-ethereum's interpreter polls
	// it, so on the chain's interpreters the bound is RPCGasCap.
	RPCEVMTimeout time.Duration
	// RPCTxFeeCap refuses a transaction whose fee exceeds it, in FTM.
	RPCTxFeeCap float64
	// AllowUnprotectedTxs accepts transactions without EIP-155 protection.
	AllowUnprotectedTxs bool

	// MaxResponseSize bounds the responses that collect traces, in bytes.
	MaxResponseSize int
	// StructLogLimit bounds the entries of the default tracer.
	StructLogLimit int
	// MaxTraceFilterRange is the widest block span of trace_filter, 0 for
	// none.
	MaxTraceFilterRange uint64
	// AllowJSTracers permits tracers that are not built in.
	AllowJSTracers bool

	// LogQueryParameterLimit bounds the addresses and topics of a log query,
	// 0 for none.
	LogQueryParameterLimit uint
	// LogQueryResultLimit bounds the logs of a query over several blocks, 0
	// for none.
	LogQueryResultLimit uint
	// IndexedLogsBlockRangeLimit bounds the span of an indexed log query.
	IndexedLogsBlockRangeLimit uint64
	// UnindexedLogsBlockRangeLimit bounds the span of a log query that scans
	// blocks.
	UnindexedLogsBlockRangeLimit uint64
	// FilterTimeout removes a polled filter not polled for this long.
	FilterTimeout time.Duration

	// TxSyncDefaultTimeout and TxSyncMaxTimeout bound the wait of
	// eth_sendRawTransactionSync.
	TxSyncDefaultTimeout time.Duration
	TxSyncMaxTimeout     time.Duration
}

// DefaultConfig returns the default limits, those of the reference. The node
// configuration refers to these values and repeats none of them.
func DefaultConfig() Config {
	return Config{
		RPCGasCap:                    50_000_000,
		RPCEVMTimeout:                5 * time.Second,
		RPCTxFeeCap:                  100,
		MaxResponseSize:              25 * 1024 * 1024,
		StructLogLimit:               2000,
		MaxTraceFilterRange:          1000,
		LogQueryParameterLimit:       1000,
		LogQueryResultLimit:          5000,
		IndexedLogsBlockRangeLimit:   999_999_999_999_999_999,
		UnindexedLogsBlockRangeLimit: 100,
		FilterTimeout:                5 * time.Minute,
		TxSyncDefaultTimeout:         5 * time.Second,
		TxSyncMaxTimeout:             30 * time.Second,
	}
}
