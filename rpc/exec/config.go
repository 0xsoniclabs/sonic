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

// Config holds the semantic limits and switches of the RPC module, which its
// namespaces read through the environment.
type Config struct {
	// RPCGasCap caps the gas of calls, estimates and traces; zero for none.
	RPCGasCap uint64
	// RPCEVMTimeout bounds an EVM run of a call; only go-ethereum's
	// interpreter polls it.
	RPCEVMTimeout time.Duration
	// RPCTxFeeCap caps the fee, in FTM, of a sent transaction; zero for none.
	RPCTxFeeCap float64
	// AllowUnprotectedTxs accepts sent transactions without EIP-155 replay
	// protection.
	AllowUnprotectedTxs bool
	// TxSyncDefaultTimeout is how long eth_sendRawTransactionSync waits for
	// the transaction's block unless the request names a wait, which
	// TxSyncMaxTimeout caps unless it is zero.
	TxSyncDefaultTimeout time.Duration
	TxSyncMaxTimeout     time.Duration
	// EnableTestOnlyApi lets the methods of the test namespace do their work.
	EnableTestOnlyApi bool
	// LogQueryParameterLimit caps the addresses and topics of the criteria
	// of a polled filter or a subscription; zero for none.
	LogQueryParameterLimit uint
	// FilterTimeout is how long a polled filter lives after its last poll.
	FilterTimeout time.Duration
}

// DefaultConfig is the configuration unless the node sets one.
func DefaultConfig() Config {
	return Config{
		RPCGasCap:              50000000,
		RPCEVMTimeout:          5 * time.Second,
		RPCTxFeeCap:            100,
		TxSyncDefaultTimeout:   5 * time.Second,
		TxSyncMaxTimeout:       30 * time.Second,
		LogQueryParameterLimit: 1000,
		FilterTimeout:          5 * time.Minute,
	}
}
