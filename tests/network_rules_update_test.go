package tests

import (
	"encoding/json"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/contract/driverauth100"
	"github.com/0xsoniclabs/sonic/opera/contracts/driverauth"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

func TestNetworkRule_Update_RulesChangeDuringEpochHasNoEffect(t *testing.T) {
	require := require.New(t)
	net := StartIntegrationTestNetWithFakeGenesis(t)

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

	blockBefore, err := client.BlockByNumber(t.Context(), nil)
	require.NoError(err)

	require.Less(blockBefore.BaseFee.ToInt().Int64(), newMinBaseFee, "BaseFee should not reflect new MinBaseFee")

	// apply epoch change
	advanceEpoch(t, net)

	// rule should be effective
	err = client.Client().Call(&updatedRules, "eth_getRules", "latest")
	require.NoError(err)

	require.Equal(newMinBaseFee, updatedRules.Economy.MinBaseFee.Int64(),
		"Network rules should become effective after epoch change")

	var blockAfter evmcore.EvmBlockJson
	err = client.Client().Call(&blockAfter, "eth_getBlockByNumber", "latest", false)
	require.NoError(err)

	require.GreaterOrEqual(blockAfter.BaseFee.ToInt().Int64(), newMinBaseFee, "BaseFee should reflect new MinBaseFee")
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

	newMinBaseFee := 1e3 * originalRules.Economy.MinBaseFee.Int64()
	updateRequest := rulesType{}
	updateRequest.Economy.MinBaseFee = new(big.Int).SetInt64(newMinBaseFee)

	// Update network rules
	updateNetworkRules(t, net, updateRequest)

	// Restart the network, since the rules happened withing a current epoch
	// it should not be applied immediately but persisted to be applied at the end of the epoch.
	err = net.RestartWithExportImport()
	require.NoError(err)

	client2, err := net.GetClient()
	require.NoError(err)
	defer client2.Close()

	// Network rule should not change - it must be an epoch bound
	var updatedRules rulesType
	err = client2.Client().Call(&updatedRules, "eth_getRules", "latest")
	require.NoError(err)

	require.Equal(originalRules.Economy.MinBaseFee, updatedRules.Economy.MinBaseFee,
		"Network rules should not change - it must be an epoch bound")

	// apply epoch change
	advanceEpoch(t, net)

	// rule change should be effective
	err = client2.Client().Call(&updatedRules, "eth_getRules", "latest")
	require.NoError(err)

	require.Equal(newMinBaseFee, updatedRules.Economy.MinBaseFee.Int64(),
		"Network rules should become effective after epoch change")

	var blockAfter evmcore.EvmBlockJson
	err = client2.Client().Call(&blockAfter, "eth_getBlockByNumber", "latest", false)
	require.NoError(err)

	require.GreaterOrEqual(blockAfter.BaseFee.ToInt().Int64(), newMinBaseFee, "BaseFee should reflect new MinBaseFee")
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

func advanceEpoch(t *testing.T, net IntegrationTestNetSession) {
	t.Helper()
	require := require.New(t)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	var currentEpoch hexutil.Uint64
	err = client.Client().Call(&currentEpoch, "eth_currentEpoch")
	require.NoError(err)

	contract, err := driverauth100.NewContract(driverauth.ContractAddress, client)
	require.NoError(err)

	receipt, err := net.Apply(func(ops *bind.TransactOpts) (*types.Transaction, error) {
		return contract.AdvanceEpochs(ops, big.NewInt(1))
	})

	require.NoError(err)
	require.Equal(receipt.Status, types.ReceiptStatusSuccessful)

	// wait until the epoch is advanced
	for {
		var newEpoch hexutil.Uint64
		err = client.Client().Call(&newEpoch, "eth_currentEpoch")
		require.NoError(err)
		if newEpoch > currentEpoch {
			break
		}
	}

	var currentBlock evmcore.EvmBlockJson
	err = client.Client().Call(&currentBlock, "eth_getBlockByNumber", "latest", false)
	require.NoError(err)

	// wait the next two blocks as the fee is applied to the next block after
	//the epoch change becomes effective
	for {
		var newBlock evmcore.EvmBlockJson
		err = client.Client().Call(&newBlock, "eth_getBlockByNumber", "latest", false)
		require.NoError(err)
		if newBlock.Number.ToInt().Int64() > currentBlock.Number.ToInt().Int64()+1 {
			break
		}
	}
}
