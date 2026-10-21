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
	rpceth "github.com/0xsoniclabs/sonic/rpc/eth"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestEthConfig_ProducesReadableConfig(t *testing.T) {

	net := StartIntegrationTestNet(t,
		IntegrationTestNetOptions{
			Upgrades: AsPointer(opera.GetBrioUpgrades()),
		})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	response := map[string]map[string]any{}
	err = client.Client().Call(&response, "eth_config")
	require.NoError(t, err, "eth_config failed")
	response["current"]["activationTime"] = uint64(response["current"]["activationTime"].(float64))

	// get time from block one because genesis block has time zero
	// and upgrades gets updated in block 1
	block1, err := client.BlockByNumber(t.Context(), big.NewInt(1))
	require.NoError(t, err, "could not get block 1 to determine expected ActivationTime")

	upgradeHeight := opera.UpgradeHeight{
		Upgrades: opera.GetBrioUpgrades(),
		Height:   1,
	}
	genesisId := net.GetGenesisId()
	expectedForkId, err := rpceth.MakeForkId(upgradeHeight, genesisId)
	require.NoError(t, err, "could not make expected fork ID")

	want := map[string]map[string]any{
		"current": {
			"activationTime": block1.Time(),
			"blockHeight":    fmt.Sprintf("0x%x", block1.Number().Uint64()),
			"blobSchedule":   nil,
			"chainId":        "0xfa3",
			"forkId":         fmt.Sprintf("0x%x", expectedForkId),
			"precompiles": map[string]any{
				"BLAKE2F":              "0x0000000000000000000000000000000000000009",
				"BN254_ADD":            "0x0000000000000000000000000000000000000006",
				"BN254_MUL":            "0x0000000000000000000000000000000000000007",
				"BN254_PAIRING":        "0x0000000000000000000000000000000000000008",
				"ECREC":                "0x0000000000000000000000000000000000000001",
				"ID":                   "0x0000000000000000000000000000000000000004",
				"KZG_POINT_EVALUATION": "0x000000000000000000000000000000000000000a",
				"MODEXP":               "0x0000000000000000000000000000000000000005",
				"RIPEMD160":            "0x0000000000000000000000000000000000000003",
				"SHA256":               "0x0000000000000000000000000000000000000002",
				"BLS12_G1ADD":          "0x000000000000000000000000000000000000000b",
				"BLS12_G1MSM":          "0x000000000000000000000000000000000000000c",
				"BLS12_G2ADD":          "0x000000000000000000000000000000000000000d",
				"BLS12_G2MSM":          "0x000000000000000000000000000000000000000e",
				"BLS12_MAP_FP2_TO_G2":  "0x0000000000000000000000000000000000000011",
				"BLS12_MAP_FP_TO_G1":   "0x0000000000000000000000000000000000000010",
				"BLS12_PAIRING_CHECK":  "0x000000000000000000000000000000000000000f",
				"P256VERIFY":           "0x0000000000000000000000000000000000000100",
			},
			"systemContracts": map[string]any{"HISTORY_STORAGE_ADDRESS": "0x0000f90827f1c53a10cb7a02335b175320002935"},
		},
		"next": nil,
		"last": nil,
	}
	require.Equal(t, want, response, "eth_config returned unexpected result")

	originalForkId := response["current"]["forkId"]
	require.NotZero(t, originalForkId, "forkId should not be zero")

	// get current rules and change to single proposer mode
	var originalRules opera.Rules
	err = client.Client().Call(&originalRules, "eth_getRules", "latest")
	require.NoError(t, err, "eth_getRules failed")

	originalRules.Upgrades.GasSubsidies = true
	UpdateNetworkRules(t, net, originalRules)
	net.AdvanceEpoch(t, 1)

	// get new config
	err = client.Client().Call(&response, "eth_config")
	require.NoError(t, err, "eth_config failed")

	// again, activation time needs to be reinterpreted as a uint64
	response["last"]["activationTime"] = uint64(response["last"]["activationTime"].(float64))
	require.Equal(t, want["current"], response["last"],
		"original config should be in 'last' field")

	require.Contains(t, response["current"], "systemContracts")
	require.Contains(t, response["current"]["systemContracts"], "GAS_SUBSIDY_REGISTRY_ADDRESS")

	// reinterpret block height from string to uint64
	heightString := response["current"]["blockHeight"].(string)
	heightUint, err := hexutil.DecodeUint64(heightString)
	require.NoError(t, err, "could not decode block height from hex string")

	// get current block to confirm epoch advancement
	activationBlock, err := client.HeaderByNumber(t.Context(), big.NewInt(int64(heightUint)))
	require.NoError(t, err)
	require.Equal(t, activationBlock.Time, uint64(response["current"]["activationTime"].(float64)),
		"activation block time should be greater than the activation time in config")

	upgradeHeight = opera.UpgradeHeight{
		Upgrades: originalRules.Upgrades,
		Height:   idx.Block(heightUint),
	}
	expectedForkId, err = rpceth.MakeForkId(upgradeHeight, genesisId)
	require.NoError(t, err, "could not make expected fork ID after upgrade")

	require.Equal(t, fmt.Sprintf("0x%x", expectedForkId), response["current"]["forkId"],
		"fork ID after upgrade is incorrect")
}
