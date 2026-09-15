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

// Package transaction_properties draws transactions with pgregory.net/rapid and injects them
// straight into consensus through the test-only API, bypassing the transaction pool, then checks
// what the network did with them against an independent model and a set of block invariants.
// README.md describes the design.
//
// It all runs on one chain: each fork is a subtest, and each workload within it another, so the
// history the run leaves behind carries every fork and every feature in the order they were switched
// on.
//
// This package only wires the domains together and holds the property test itself: the harness is in
// core, the ordinary transaction types are in regular, and everything about gas subsidies,
// transaction bundles and transaction priorities is in subsidies, bundles and priorities
// respectively.
//
// A reported failure names the seed that produced it, so it can be replayed:
//
//	go test ./tests/transaction_properties/ -rapid.seed=1436157575314665811
package transaction_properties

import (
	"fmt"
	"os"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
)

// How much work a run does. A handful of transactions from a handful of senders is what reaches the
// interesting interactions: rival nonces, one malformed transaction taking its batch-mates down, a
// sender's sequence advancing mid-batch.
const (
	checksPerWorkload   = "200"
	maxTxsPerBatch      = 4
	maxAccountsPerBatch = 3
)

func TestMain(m *testing.M) {
	if err := core.ApplyTestFlags(checksPerWorkload); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// genConfig is the generation budget for one workload, read from the rules the network is running
// under at the time, which the fork it is in has just moved on. The gas budget stays clearly below MaxEventGas,
// since a batch rejected for exceeding it starves every other case of coverage.
func genConfig(network *core.Network) core.GenConfig {
	return core.GenConfig{
		Network:             network.Cfg,
		MaxTxsPerBatch:      maxTxsPerBatch,
		MaxAccountsPerBatch: maxAccountsPerBatch,
		GasBudget:           network.Cfg.MaxEventGas / 10,
		Contracts:           network.Contracts,
	}
}
