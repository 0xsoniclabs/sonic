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
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// TestIntrinsicGas_CantoTransactionsBelowTheLegacyMinimumAreEstimatedAndIncluded
// checks that transactions EIP-2780 prices below 21000 are estimated at their
// price and included, rather than stalling the validators whose own event
// checks used to reject them.
func TestIntrinsicGas_CantoTransactionsBelowTheLegacyMinimumAreEstimatedAndIncluded(t *testing.T) {
	for _, singleProposer := range []bool{false, true} {
		t.Run(fmt.Sprintf("singleProposer=%t", singleProposer), func(t *testing.T) {
			upgrades := opera.GetCantoUpgrades()
			upgrades.SingleProposerBlockFormation = singleProposer
			session := getIntegrationTestNetSession(t, upgrades)
			sender := session.GetSessionSponsor()

			client, err := session.GetClient()
			require.NoError(t, err)
			defer client.Close()

			tests := map[string]struct {
				to    common.Address
				value *big.Int
				gas   uint64
			}{
				"zero-value call": {
					to:  common.Address{0x42},
					gas: params.TxBaseCost2780 + params.ColdAccountAccessAmsterdam,
				},
				"transfer to self": {
					to:    sender.Address(),
					value: big.NewInt(1),
					gas:   params.TxBaseCost2780,
				},
			}
			for name, test := range tests {
				t.Run(name, func(t *testing.T) {
					tx := CreateTransaction(t, session, &types.LegacyTx{
						To:    &test.to,
						Value: test.value,
						Gas:   test.gas,
					}, sender)
					receipt, err := session.Run(tx)
					require.NoError(t, err)
					require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
					require.Equal(t, test.gas, receipt.GasUsed)

					estimate, err := client.EstimateGas(t.Context(), ethereum.CallMsg{
						From:  sender.Address(),
						To:    &test.to,
						Value: test.value,
					})
					require.NoError(t, err)
					require.Equal(t, test.gas, estimate)
				})
			}
		})
	}
}
