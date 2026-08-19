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
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// TestStateGasReservoir_CantoDeploysCodeWhoseStateGasExceedsTheEventGasCap
// checks that, from Canto on, the gas limit above the execution cap funds the
// EIP-8037 state-gas reservoir, so a deployment whose code deposit alone costs
// more than MaxEventGas is estimated, included and executed.
func TestStateGasReservoir_CantoDeploysCodeWhoseStateGasExceedsTheEventGasCap(t *testing.T) {
	const codeSize = 30_000
	// Returns codeSize bytes of zeroed memory as the contract's code.
	initCode := []byte{
		byte(vm.PUSH2), byte(codeSize >> 8), byte(codeSize & 0xff),
		byte(vm.PUSH1), 0,
		byte(vm.RETURN),
	}

	for _, singleProposer := range []bool{false, true} {
		t.Run(fmt.Sprintf("singleProposer=%t", singleProposer), func(t *testing.T) {
			upgrades := opera.GetCantoUpgrades()
			upgrades.SingleProposerBlockFormation = singleProposer
			session := getIntegrationTestNetSession(t, upgrades)
			sender := session.GetSessionSponsor()

			client, err := session.GetClient()
			require.NoError(t, err)
			defer client.Close()

			estimate, err := client.EstimateGas(t.Context(), ethereum.CallMsg{
				From: sender.Address(),
				Data: initCode,
			})
			require.NoError(t, err)
			require.Greater(t, estimate, uint64(codeSize*params.CostPerStateByte))
			require.Greater(t, estimate, GetNetworkRules(t, session).Economy.Gas.MaxEventGas)

			tx := CreateTransaction(t, session, &types.LegacyTx{
				Gas:  estimate,
				Data: initCode,
			}, sender)
			receipt, err := session.Run(tx)
			require.NoError(t, err)
			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

			code, err := client.CodeAt(t.Context(), receipt.ContractAddress, receipt.BlockNumber)
			require.NoError(t, err)
			require.Len(t, code, codeSize)
		})
	}
}
