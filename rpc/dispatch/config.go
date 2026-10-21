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

package dispatch

import (
	"math/big"
	"time"
)

// Config holds the limits and options the module's methods apply. Every knob
// keeps the name, default and effect it has in the node's configuration.
type Config struct {
	// Timeout is the deadline the dispatcher attaches to every request (RPCTimeout).
	Timeout time.Duration
	// EVMTimeout cancels the geth interpreter (RPCEVMTimeout).
	EVMTimeout time.Duration
	// GasCap caps the gas of eth_call, estimates and traces (RPCGasCap), 0 = uncapped.
	GasCap uint64
	// TxFeeCap refuses sends that pay more than this many native tokens (RPCTxFeeCap), 0 = uncapped.
	TxFeeCap float64
	// UnprotectedAllowed accepts transactions without EIP-155 replay protection.
	UnprotectedAllowed bool
	// TxSyncDefaultTimeout and TxSyncMaxTimeout bound eth_sendRawTransactionSync.
	TxSyncDefaultTimeout time.Duration
	TxSyncMaxTimeout     time.Duration
	// MaxResponseSize caps the response of debug_* and trace_* (bytes).
	MaxResponseSize int
	// StructLogLimit caps the number of struct logs of the default tracer.
	StructLogLimit int
	// MaxTraceFilterRange is the widest block span of trace_filter, 0 = unlimited.
	MaxTraceFilterRange uint64
	// AllowJSTracers permits user supplied JavaScript tracers.
	AllowJSTracers bool

	// Log filter limits.
	LogQueryParameterLimit       uint
	LogQueryResultLimit          uint
	IndexedLogsBlockRangeLimit   uint64
	UnindexedLogsBlockRangeLimit uint64
	// FilterTimeout is how long an unpolled filter lives.
	FilterTimeout time.Duration

	// Gas price oracle. A nil OracleMaxGasPrice is the oracle's default.
	OracleMaxGasPrice      *big.Int
	OracleMinGasPrice      *big.Int
	OracleDefaultCertainty uint64

	// HistoryPruningCutoff is the first block that is not pruned; "earliest".
	HistoryPruningCutoff uint64
}

// DefaultConfig returns the defaults of the reference release.
func DefaultConfig() Config {
	return Config{
		Timeout:              5 * time.Second,
		EVMTimeout:           5 * time.Second,
		GasCap:               50_000_000,
		TxFeeCap:             100,
		TxSyncDefaultTimeout: 5 * time.Second,
		TxSyncMaxTimeout:     30 * time.Second,
		MaxResponseSize:      25 * 1024 * 1024,
		StructLogLimit:       2000,
		MaxTraceFilterRange:  1000,

		LogQueryParameterLimit:       1000,
		LogQueryResultLimit:          5000,
		IndexedLogsBlockRangeLimit:   999999999999999999,
		UnindexedLogsBlockRangeLimit: 100,
		FilterTimeout:                5 * time.Minute,

		OracleMinGasPrice:      new(big.Int),
		OracleDefaultCertainty: 500_000, // 0.5 * piecefunc.DecimalUnit
	}
}
