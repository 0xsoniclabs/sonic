package tests

import (
	"encoding/json"
	"github.com/0xsoniclabs/sonic/gossip/contract/driverauth100"
	"github.com/0xsoniclabs/sonic/opera/contracts/driverauth"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

func TestNetworkRule_Update_RulesChangeDuringEpochHasNoEffect(t *testing.T) {
	require := require.New(t)
	net := StartIntegrationTestNetWithFakeGenesis(t)
	defer net.Stop()

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	type rulesType struct {
		Economy struct {
			MinBaseFee *big.Int
		}
	}

	var originalRules rulesType
	err = client.Client().Call(&originalRules, "eth_getRules", "latest")
	require.NoError(err)
	require.NotEqual(0, originalRules.Economy.MinBaseFee.Int64(), "MinBaseFee should be filled")

	updateRequest := rulesType{}
	updateRequest.Economy.MinBaseFee = new(big.Int).SetInt64(2 * originalRules.Economy.MinBaseFee.Int64())

	// Update network rules
	updateNetworkRules(t, require, net, updateRequest)

	// Network rule should not change - it must be an epoch bound
	var updatedRules rulesType
	err = client.Client().Call(&updatedRules, "eth_getRules", "latest")
	require.NoError(err)

	require.Equal(originalRules.Economy.MinBaseFee, updatedRules.Economy.MinBaseFee,
		"Network rules should not change - it must be an epoch bound")
}

func TestNetworkRule_Update_Restart_Recovers_Original_Value(t *testing.T) {
	require := require.New(t)
	net := StartIntegrationTestNetWithFakeGenesis(t)
	defer net.Stop()

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	type rulesType struct {
		Economy struct {
			MinBaseFee *big.Int
		}
	}

	var originalRules rulesType
	err = client.Client().Call(&originalRules, "eth_getRules", "latest")
	require.NoError(err)
	require.NotEqual(0, originalRules.Economy.MinBaseFee.Int64(), "MinBaseFee should be filled")

	updateRequest := rulesType{}
	updateRequest.Economy.MinBaseFee = new(big.Int).SetInt64(2 * originalRules.Economy.MinBaseFee.Int64())

	// Update network rules
	updateNetworkRules(t, require, net, updateRequest)

	// Restart the network, since the rules happened withing a current epoch
	// it should not be applied and persisted.
	err = net.RestartWithExportImport()
	require.NoError(err)

	client2, err := net.GetClient()
	require.NoError(err)
	defer client2.Close()

	// Network rule should not change - it must be an epoch bound
	var updatedRules rulesType
	err = client.Client().Call(&updatedRules, "eth_getRules", "latest")
	require.NoError(err)

	require.Equal(originalRules.Economy.MinBaseFee, updatedRules.Economy.MinBaseFee,
		"Network rules should not change - it must be an epoch bound")
}

// updateNetworkRules sends a transaction to update the network rules.
func updateNetworkRules(t *testing.T, require *require.Assertions, net IntegrationTestNetSession, rulesChange any) {
	t.Helper()

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	b, err := json.Marshal(rulesChange)
	require.NoError(err)

	contract, err := driverauth100.NewContract(driverauth.ContractAddress, client)
	receipt, err := net.Apply(func(ops *bind.TransactOpts) (*types.Transaction, error) {
		return contract.UpdateNetworkRules(ops, b)
	})

	require.NoError(err)
	require.Equal(receipt.Status, types.ReceiptStatusSuccessful)
}
