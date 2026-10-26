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
// the same ones, so they are one struct; DefaultConfig is the one definition
// of each default.
type Config struct {
	// GasCap bounds the gas of a call, an estimate and a traced call
	// (--rpc.gascap); zero means no cap.
	GasCap uint64
	// EVMTimeout cancels an EVM run where the interpreter polls for it
	// (--rpc.evmtimeout); Tosca's interpreters never do.
	EVMTimeout time.Duration
	// TxFeeCap refuses a send whose fee is above it, in FTM (--rpc.txfeecap);
	// zero means no cap.
	TxFeeCap float64
	// AllowUnprotectedTxs accepts sends without EIP-155 replay protection.
	AllowUnprotectedTxs bool
	// TxSyncDefaultTimeout and TxSyncMaxTimeout bound the wait of
	// eth_sendRawTransactionSync.
	TxSyncDefaultTimeout time.Duration
	TxSyncMaxTimeout     time.Duration
	// MaxResponseSize bounds a response that collects traces, in bytes
	// (--rpc.maxresponsesize); zero means no bound.
	MaxResponseSize int
	// StructLogLimit bounds the output of the default tracer; go-ethereum's
	// struct logger counts it in bytes.
	StructLogLimit int
	// AllowJSTracers lets debug_trace* run any tracer, not only the built-in
	// ones.
	AllowJSTracers bool
	// MaxTraceFilterRange bounds the block span of trace_filter
	// (--rpc.tracefilterrange); zero means no bound.
	MaxTraceFilterRange uint64
	// LogQueryParameterLimit bounds the addresses and topics of a log query
	// on every filter path; zero means no bound.
	LogQueryParameterLimit uint
	// LogQueryResultLimit bounds the logs of a query over more than one
	// block; zero means no bound.
	LogQueryResultLimit uint
	// IndexedLogsBlockRangeLimit and UnindexedLogsBlockRangeLimit bound the
	// block span of an indexed and of a block-scan log query.
	IndexedLogsBlockRangeLimit   uint64
	UnindexedLogsBlockRangeLimit uint64
	// FilterTimeout drops a polled filter that was not polled for so long.
	FilterTimeout time.Duration
}

// DefaultConfig is the one definition of the methods' limits.
func DefaultConfig() Config {
	return Config{
		GasCap:                       50000000,
		EVMTimeout:                   5 * time.Second,
		TxFeeCap:                     100,
		TxSyncDefaultTimeout:         5 * time.Second,
		TxSyncMaxTimeout:             30 * time.Second,
		MaxResponseSize:              25 * 1024 * 1024,
		StructLogLimit:               2000,
		MaxTraceFilterRange:          1000,
		LogQueryParameterLimit:       1000,
		LogQueryResultLimit:          5000,
		IndexedLogsBlockRangeLimit:   999999999999999999,
		UnindexedLogsBlockRangeLimit: 100,
		FilterTimeout:                5 * time.Minute,
	}
}
