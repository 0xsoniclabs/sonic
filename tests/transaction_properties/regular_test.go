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

package transaction_properties

import (
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/regular"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// TestTransactionProperties_ForcedTransactionsPreserveConsensusInvariants draws batches of
// transactions and injects them straight into consensus past the pool, which is the point: with the
// component that normally refuses malformed transactions out of the way, the pressure falls on event
// validation, the block formation filter and the state processor, where being wrong means a chain
// split rather than a rejected request. Each fork is tested separately because the answers differ.
func TestTransactionProperties_ForcedTransactionsPreserveConsensusInvariants(t *testing.T) {
	for name, upgrades := range opera.GetAllHardForksInOrder() {
		t.Run(name, func(t *testing.T) {
			require.False(t, upgrades.SingleProposerBlockFormation,
				"forced emission is unsupported in single proposer mode")

			network := core.StartNetwork(t, upgrades)
			session := network.Net.SpawnSession(t)

			client, err := session.GetClient()
			require.NoError(t, err)
			defer client.Close()

			runner := &core.Runner{
				Session: session,
				Client:  client,
				Network: network,
				Cfg:     genConfig(network),
				Domain:  regular.Domain{Cfg: genConfig(network)},
			}
			runner.Init(t)

			rapid.Check(t, runner.Run)

			runner.Report(t, name)
			runner.VerifyChainReplays(t)
			runner.ExportGenesis(t)
		})
	}
}
