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
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
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

// TestGasSubsidies_SimulateWithValidation_RegistryQueryIsBoundedAndLeavesNoTrace
// checks that the registry query eth_simulateV1 runs under validation is bounded
// on a node without a gas cap and does not shift the log index of the calls, for
// a registry that a state override replaces.
func TestGasSubsidies_SimulateWithValidation_RegistryQueryIsBoundedAndLeavesNoTrace(t *testing.T) {
	upgrades := opera.GetAllegroUpgrades()
	upgrades.GasSubsidies = true
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades:             &upgrades,
		ClientExtraArguments: []string{"--rpc.gascap", "0"},
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	target := common.Address{0x42}
	zeroFeeCall := map[string]any{
		"from":                 common.Address{0x51},
		"to":                   target,
		"gas":                  hexutil.Uint64(100_000),
		"maxFeePerGas":         (*hexutil.Big)(big.NewInt(0)),
		"maxPriorityFeePerGas": (*hexutil.Big)(big.NewInt(0)),
	}

	t.Run("looping registry runs out of gas", func(t *testing.T) {
		// getGasConfig (4 bytes of input) grants chooseFund the largest gas
		// limit, and chooseFund loops forever.
		code := hexutil.MustDecode("0x" +
			"60043614600b" + "57" + // if CALLDATASIZE == 4, jump to getGasConfig
			"5b600756" + // chooseFund: loop
			"5b" + "67ffffffffffffffff600052" + "620186a0602052" + "61c350" + "604052" +
			"60606000f3") // getGasConfig: return the gas config
		opts := map[string]any{
			"blockStateCalls": []any{map[string]any{
				"stateOverrides": map[string]any{
					registry.GetAddress().Hex(): map[string]any{"code": hexutil.Bytes(code)},
				},
				"calls": []any{zeroFeeCall},
			}},
			"validation": true,
		}
		ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
		defer cancel()
		var result any
		err := client.Client().CallContext(ctx, &result, "eth_simulateV1", opts, "latest")
		require.ErrorContains(t, err, "failed to query subsidies registry")
		require.ErrorContains(t, err, "out of gas")
	})

	t.Run("logs of the registry query do not shift the log index", func(t *testing.T) {
		// Every registry query emits a log; getGasConfig (4 bytes of input)
		// returns a gas config and chooseFund reports that no fund covers the
		// call. With a base fee of zero, the uncovered call still runs.
		registryCode := hexutil.MustDecode("0x" +
			"60006000a0" + // LOG0
			"600436146011" + "57" + // if CALLDATASIZE == 4, jump to getGasConfig
			"60206000f3" + // chooseFund: not covered
			"5b" + "620186a0600052" + "620186a0602052" + "61c350" + "604052" +
			"60606000f3") // getGasConfig: return the gas config
		targetCode := hexutil.MustDecode("0x60006000a000") // LOG0, STOP
		opts := map[string]any{
			"blockStateCalls": []any{map[string]any{
				"blockOverrides": map[string]any{"baseFeePerGas": (*hexutil.Big)(big.NewInt(0))},
				"stateOverrides": map[string]any{
					registry.GetAddress().Hex(): map[string]any{"code": hexutil.Bytes(registryCode)},
					target.Hex():                map[string]any{"code": hexutil.Bytes(targetCode)},
				},
				"calls": []any{zeroFeeCall, zeroFeeCall},
			}},
			"validation": true,
		}
		var result []struct {
			Calls []struct {
				Status hexutil.Uint64 `json:"status"`
				Logs   []struct {
					Address  common.Address `json:"address"`
					LogIndex hexutil.Uint   `json:"logIndex"`
				} `json:"logs"`
			} `json:"calls"`
		}
		require.NoError(t, client.Client().Call(&result, "eth_simulateV1", opts, "latest"))
		require.Len(t, result, 1)
		require.Len(t, result[0].Calls, 2)
		for i, call := range result[0].Calls {
			require.Equal(t, hexutil.Uint64(1), call.Status)
			require.Len(t, call.Logs, 1)
			require.Equal(t, target, call.Logs[0].Address)
			require.Equal(t, hexutil.Uint(i), call.Logs[0].LogIndex)
		}
	})
}
