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

package gas_subsidies

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

// TestGasSubsidies_SimulateWithValidation_TreatsSponsorshipRequestsAsBlockProcessingDoes
// checks that eth_simulateV1 with validation runs a zero-fee call that a subsidy
// covers, as block processing does, and still refuses one that no subsidy covers.
func TestGasSubsidies_SimulateWithValidation_TreatsSponsorshipRequestsAsBlockProcessingDoes(t *testing.T) {
	upgrades := opera.GetAllegroUpgrades()
	upgrades.GasSubsidies = true
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	sponsored := tests.NewAccount()
	Fund(t, net, sponsored.Address(), big.NewInt(1e16))
	unsponsored := tests.NewAccount()

	zeroFeeCall := func(from common.Address) map[string]any {
		return map[string]any{
			"from":                 from,
			"to":                   common.Address{0x42},
			"gas":                  hexutil.Uint64(21000),
			"maxFeePerGas":         (*hexutil.Big)(big.NewInt(0)),
			"maxPriorityFeePerGas": (*hexutil.Big)(big.NewInt(0)),
		}
	}
	type callResult struct {
		Status  hexutil.Uint64 `json:"status"`
		GasUsed hexutil.Uint64 `json:"gasUsed"`
	}
	simulate := func(validation bool, calls ...map[string]any) ([]callResult, error) {
		var result []struct {
			Calls []callResult `json:"calls"`
		}
		opts := map[string]any{
			"blockStateCalls": []any{map[string]any{"calls": calls}},
			"validation":      validation,
		}
		if err := client.Client().Call(&result, "eth_simulateV1", opts, "latest"); err != nil {
			return nil, err
		}
		require.Len(t, result, 1)
		return result[0].Calls, nil
	}
	const refused = "max fee per gas less than block base fee"

	t.Run("covered call runs with validation", func(t *testing.T) {
		calls, err := simulate(true, zeroFeeCall(sponsored.Address()))
		require.NoError(t, err)
		require.Equal(t, []callResult{{Status: 1, GasUsed: 21000}}, calls)
	})

	t.Run("uncovered call is refused with validation", func(t *testing.T) {
		_, err := simulate(true, zeroFeeCall(unsponsored.Address()))
		require.ErrorContains(t, err, refused)
	})

	t.Run("uncovered call after a covered one is refused with validation", func(t *testing.T) {
		_, err := simulate(true, zeroFeeCall(sponsored.Address()), zeroFeeCall(unsponsored.Address()))
		require.ErrorContains(t, err, refused)
	})

	t.Run("uncovered call runs without validation", func(t *testing.T) {
		calls, err := simulate(false, zeroFeeCall(unsponsored.Address()))
		require.NoError(t, err)
		require.Equal(t, []callResult{{Status: 1, GasUsed: 21000}}, calls)
	})
}
