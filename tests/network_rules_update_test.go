package tests

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/contract/driverauth100"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/contracts/driverauth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestNetworkRule_Update_RulesChangeIsDelayedUntilNextEpochStart(t *testing.T) {
	require := require.New(t)
	net := StartIntegrationTestNetWithFakeGenesis(t,
		IntegrationTestNetOptions{
			Upgrades: AsPointer(opera.GetAllegroUpgrades()),
		})

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

	newMinBaseFee := 1e3 * originalRules.Economy.MinBaseFee.Int64()
	updateRequest := rulesType{}
	updateRequest.Economy.MinBaseFee = new(big.Int).SetInt64(newMinBaseFee)

	// Update network rules
	updateNetworkRules(t, net, updateRequest)

	// Network rule should not change - it must be an epoch bound
	var updatedRules rulesType
	err = client.Client().Call(&updatedRules, "eth_getRules", "latest")
	require.NoError(err)

	require.Equal(originalRules.Economy.MinBaseFee, updatedRules.Economy.MinBaseFee,
		"Network rules should not change - it must be an epoch bound")

	// produce a block to make sure the rule is not applied
	_, err = net.EndowAccount(common.Address{}, big.NewInt(1))
	require.NoError(err)

	blockBefore, err := client.BlockByNumber(t.Context(), nil)
	require.NoError(err)

	require.Less(blockBefore.BaseFee().Int64(), newMinBaseFee, "BaseFee should not reflect new MinBaseFee")

	// apply epoch change
	advanceEpochAndWaitForBlocks(t, net)

	// rule should be effective
	err = client.Client().Call(&updatedRules, "eth_getRules", "latest")
	require.NoError(err)

	require.Equal(newMinBaseFee, updatedRules.Economy.MinBaseFee.Int64(),
		"Network rules should become effective after epoch change")

	blockAfter, err := client.BlockByNumber(t.Context(), nil)
	require.NoError(err)

	require.GreaterOrEqual(blockAfter.BaseFee().Int64(), newMinBaseFee, "BaseFee should reflect new MinBaseFee")
}

func TestNetworkRule_Update_RulesChangeDuringEpoch_PreAllegro(t *testing.T) {
	require := require.New(t)
	net := StartIntegrationTestNetWithFakeGenesis(t,
		IntegrationTestNetOptions{
			Upgrades: AsPointer(opera.GetSonicUpgrades()),
		})

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

	newMinBaseFee := 10 * originalRules.Economy.MinBaseFee.Int64()
	updateRequest := rulesType{}
	updateRequest.Economy.MinBaseFee = new(big.Int).SetInt64(newMinBaseFee)

	// Update network rules
	updateNetworkRules(t, net, updateRequest)

	// Network rule applied immediately - only for pre-Allegro versions
	var updatedRules rulesType
	err = client.Client().Call(&updatedRules, "eth_getRules", "latest")
	require.NoError(err)

	require.Equal(updatedRules.Economy.MinBaseFee.Int64(), newMinBaseFee,
		"Network rules not changed")

	// produce a block to make sure the rule is applied
	_, err = net.EndowAccount(common.Address{}, big.NewInt(1))
	require.NoError(err)

	blockAfter, err := client.BlockByNumber(t.Context(), nil)
	require.NoError(err)

	require.GreaterOrEqual(blockAfter.BaseFee().Int64(), newMinBaseFee, "BaseFee should reflect new MinBaseFee")
}

func TestNetworkRule_Update_Restart_Recovers_Original_Value(t *testing.T) {
	require := require.New(t)
	net := StartIntegrationTestNetWithFakeGenesis(t,
		IntegrationTestNetOptions{
			Upgrades: AsPointer(opera.GetAllegroUpgrades()),
		})

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

	newMinBaseFee := 1e3 * originalRules.Economy.MinBaseFee.Int64()
	updateRequest := rulesType{}
	updateRequest.Economy.MinBaseFee = new(big.Int).SetInt64(newMinBaseFee)

	// Update network rules
	updateNetworkRules(t, net, updateRequest)

	// Restart the network, since the rules happened within a current epoch
	// it should not be applied immediately but persisted to be applied at the end of the epoch.
	err = net.RestartWithExportImport()
	require.NoError(err)

	client2, err := net.GetClient()
	require.NoError(err)
	defer client2.Close()

	// produce a block to make sure the rule is not applied
	_, err = net.EndowAccount(common.Address{}, big.NewInt(1))
	require.NoError(err)

	blockAfterRestart, err := client2.BlockByNumber(t.Context(), nil)
	require.NoError(err)

	require.Less(blockAfterRestart.BaseFee().Int64(), newMinBaseFee, "BaseFee should not reflect new MinBaseFee")

	// Network rule should not change - it must be an epoch bound
	var updatedRules rulesType
	err = client2.Client().Call(&updatedRules, "eth_getRules", "latest")
	require.NoError(err)

	require.Equal(originalRules.Economy.MinBaseFee, updatedRules.Economy.MinBaseFee,
		"Network rules should not change - it must be an epoch bound")

	// apply epoch change
	advanceEpochAndWaitForBlocks(t, net)

	// rule change should be effective
	err = client2.Client().Call(&updatedRules, "eth_getRules", "latest")
	require.NoError(err)

	require.Equal(newMinBaseFee, updatedRules.Economy.MinBaseFee.Int64(),
		"Network rules should become effective after epoch change")

	blockAfter, err := client2.BlockByNumber(t.Context(), nil)
	require.NoError(err)

	require.GreaterOrEqual(blockAfter.BaseFee().Int64(), newMinBaseFee, "BaseFee should reflect new MinBaseFee")
}

func TestNetworkRule_MinEventGas_AllowsChangingRules(t *testing.T) {
	require := require.New(t)
	net := StartIntegrationTestNetWithFakeGenesis(t,
		IntegrationTestNetOptions{
			Upgrades: AsPointer(opera.GetSonicUpgrades()),
		})

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	var rules opera.Rules
	data, err := json.Marshal(rules)
	require.NoError(err)

	abi, err := driverauth100.ContractMetaData.GetAbi()
	require.NoError(err)

	input, err := abi.Pack("updateNetworkRules", data)
	require.NoError(err)

	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(err)

	msg := ethereum.CallMsg{
		From:     net.account.Address(),
		To:       &driverauth.ContractAddress,
		GasPrice: gasPrice.Mul(gasPrice, big.NewInt(10)),
		Data:     input,
	}

	gas, err := client.EstimateGas(t.Context(), msg)
	require.NoError(err)

	defaultGasRules := opera.DefaultGasRules()

	require.Less(gas, defaultGasRules.MaxEventGas, "Gas should be less than MaxEventGas")
	require.Less(gas, opera.UpperBoundForRuleChangeGasCosts(), "Gas should be less than upper bound for rule change gas costs")

	require.Less(gas, opera.UpperBoundForRuleChangeGasCosts()/10, "There should be a factor of 10 head room for gas costs")

	// Check that these two properties do not contradict each other
	require.Less(opera.UpperBoundForRuleChangeGasCosts(), defaultGasRules.MaxEventGas, "Upper bound for rule change gas costs should be less than MaxEventGas")
}

// updateNetworkRules sends a transaction to update the network rules.
func updateNetworkRules(t *testing.T, net IntegrationTestNetSession, rulesChange any) {
	t.Helper()
	require := require.New(t)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	b, err := json.Marshal(rulesChange)
	require.NoError(err)

	contract, err := driverauth100.NewContract(driverauth.ContractAddress, client)
	require.NoError(err)

	receipt, err := net.Apply(func(ops *bind.TransactOpts) (*types.Transaction, error) {
		return contract.UpdateNetworkRules(ops, b)
	})

	require.NoError(err)
	require.Equal(receipt.Status, types.ReceiptStatusSuccessful)
}

// advanceEpochAndWaitForBlocks sends a transaction to advance to the next epoch.
// It also waits until the new epoch is really reached and the next two blocks are produced.
// It is useful to test a situation when the rule change is applied to the next block after the epoch change.
func advanceEpochAndWaitForBlocks(t *testing.T, net IntegrationTestNetSession) {
	t.Helper()

	require := require.New(t)

	err := net.AdvanceEpoch(1)
	require.NoError(err)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	currentBlock, err := client.BlockByNumber(t.Context(), nil)
	require.NoError(err)

	// wait the next two blocks as some rules (such as min base fee) are applied
	// to the next block after the epoch change becomes effective
	for {
		newBlock, err := client.BlockByNumber(t.Context(), nil)
		require.NoError(err)
		if newBlock.Number().Int64() > currentBlock.Number().Int64()+1 {
			break
		}
	}
}
