// Copyright 2025 Sonic Operations Ltd
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

package network_rules_update

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/contract/driverauth100"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/contracts/driverauth"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestNetworkRule_Update_RulesChangeIsDelayedUntilNextEpochStart(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	net := tests.StartIntegrationTestNetWithFakeGenesis(t,
		tests.IntegrationTestNetOptions{
			Upgrades: tests.AsPointer(opera.GetAllegroUpgrades()),
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
	tests.UpdateNetworkRules(t, net, updateRequest)

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
	tests.AdvanceEpochAndWaitForBlocks(t, net)

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
	t.Parallel()

	require := require.New(t)
	net := tests.StartIntegrationTestNetWithFakeGenesis(t,
		tests.IntegrationTestNetOptions{
			Upgrades: tests.AsPointer(opera.GetSonicUpgrades()),
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
	tests.UpdateNetworkRules(t, net, updateRequest)

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
	t.Parallel()

	require := require.New(t)
	net := tests.StartIntegrationTestNetWithFakeGenesis(t,
		tests.IntegrationTestNetOptions{
			Upgrades: tests.AsPointer(opera.GetAllegroUpgrades()),
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
	tests.UpdateNetworkRules(t, net, updateRequest)

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
	tests.AdvanceEpochAndWaitForBlocks(t, net)

	// rule change should be effective
	err = client2.Client().Call(&updatedRules, "eth_getRules", "latest")
	require.NoError(err)

	require.Equal(newMinBaseFee, updatedRules.Economy.MinBaseFee.Int64(),
		"Network rules should become effective after epoch change")

	blockAfter, err := client2.BlockByNumber(t.Context(), nil)
	require.NoError(err)

	require.GreaterOrEqual(blockAfter.BaseFee().Int64(), newMinBaseFee, "BaseFee should reflect new MinBaseFee")
}

func TestNetworkRules_UpdateMaxEventGas_DropsLargeGasTxs(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	net := tests.StartIntegrationTestNetWithFakeGenesis(t,
		tests.IntegrationTestNetOptions{
			Upgrades: tests.AsPointer(opera.GetAllegroUpgrades()),
		})

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	newAccount := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	// make a transaction with over 20M gas
	tx := tests.CreateTransaction(t, net, &types.LegacyTx{
		To:    &common.Address{1},
		Gas:   21_000_000,
		Nonce: 1, // High nonce that cannot be executed yet but will not be dropped from the txpool
	}, newAccount)

	err = client.SendTransaction(t.Context(), tx)
	require.NoError(err, "failed to send high gas transaction")

	var content map[string]map[string]map[string]*ethapi.RPCTransaction
	err = client.Client().Call(&content, "txpool_content")
	require.NoError(err, "failed to get tx pool status")
	require.Equal(1, len(content["queued"]), "expected the high gas tx to be in the queued section of the tx pool")

	type rulesType struct {
		Economy  struct{ Gas opera.GasRules }
		Upgrades struct{ Brio bool }
	}

	var originalRules rulesType
	err = client.Client().Call(&originalRules, "eth_getRules", "latest")
	require.NoError(err)
	require.NotNil(originalRules.Economy.Gas, "GasRules should be filled")
	require.NotEqual(0, originalRules.Economy.Gas.MaxEventGas, "GasRules should be filled")

	updatedRules := originalRules
	defaultGasRules := opera.DefaultGasRules()
	defaultGasRules.MaxEventGas = 16_777_216 // inspired by params.MaxTxGas
	updatedRules.Economy.Gas = defaultGasRules

	// Update network rules
	tests.UpdateNetworkRules(t, net, updatedRules)

	err = client.Client().Call(&content, "txpool_content")
	require.NoError(err, "failed to get tx pool status")
	require.Equal(1, len(content["queued"]), "expected the high gas tx to be in the queued section of the tx pool")

	// reach epoch ceiling to apply the new rules
	tests.AdvanceEpochAndWaitForBlocks(t, net)

	err = client.Client().Call(&content, "txpool_content")
	require.NoError(err, "failed to get tx pool status")
	require.Equal(0, len(content["queued"]))
}

func TestNetworkRule_MinEventGas_AllowsChangingRules(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	net := tests.StartIntegrationTestNetWithFakeGenesis(t,
		tests.IntegrationTestNetOptions{
			Upgrades: tests.AsPointer(opera.GetSonicUpgrades()),
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
		From:     net.GetSessionSponsor().Address(),
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

func TestNetworkRules_PragueFeaturesBecomeAvailableWithAllegroUpgrade(t *testing.T) {
	t.Parallel()

	net := tests.StartIntegrationTestNetWithFakeGenesis(t,
		tests.IntegrationTestNetOptions{
			// Explicitly set the network to use the Sonic Hard Fork
			Upgrades: tests.AsPointer(opera.GetSonicUpgrades()),
			// Use 2 nodes to test the rules update propagation
			NumNodes: 2,
		},
	)

	account := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	t.Run("expectations before sonic-allegro hardfork", func(t *testing.T) {
		forEachClientInNet(t, net, func(t *testing.T, client *tests.PooledEhtClient) {
			tx := makeSetCodeTx(t, net, account)
			err := client.SendTransaction(t.Context(), tx)
			require.ErrorContains(t,
				err, evmcore.ErrTxTypeNotSupported.Error(),
				"SetCodeTx cannot be accepted before Prague hard fork")
		})
	})

	// Update network rules to enable the Allegro Hard Fork
	type rulesType struct {
		Upgrades struct{ Allegro bool }
	}
	rulesDiff := rulesType{
		Upgrades: struct{ Allegro bool }{Allegro: true},
	}
	tests.UpdateNetworkRules(t, net, rulesDiff)

	// reach epoch ceiling to apply the new rules
	tests.AdvanceEpochAndWaitForBlocks(t, net)

	// Wait for another block, this is time for the tx_pool to tick, run reorg,
	// and implement the new rules.
	receipt, err := net.EndowAccount(account.Address(), big.NewInt(1e18))
	require.NoError(t, err, "failed to endow account with balance")
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	t.Run("expectations before sonic-allegro hardfork", func(t *testing.T) {

		// Submit a transaction that requires the new behavior
		tx := makeSetCodeTx(t, net, account)
		receipt, err := net.Run(tx)
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

		delegationIndicator :=
			hexutil.MustDecode("0xEF01002A00000000000000000000000000000000000000")

		forEachClientInNet(t, net, func(t *testing.T, client *tests.PooledEhtClient) {

			// make sure that this client has already processed the transaction
			_, err := net.GetReceipt(tx.Hash())
			require.NoError(t, err, "failed to get receipt for the transaction")

			code, err := client.CodeAt(t.Context(), account.Address(), nil)
			require.NoError(t, err)
			require.Equal(t, code, delegationIndicator)
		})
	})
}

func forEachClientInNet(
	t *testing.T,
	net *tests.IntegrationTestNet,
	fn func(t *testing.T, client *tests.PooledEhtClient),
) {
	for i := 0; i < net.NumNodes(); i++ {
		t.Run(fmt.Sprintf("client%d", i), func(t *testing.T) {
			client, err := net.GetClientConnectedToNode(i)
			require.NoError(t, err)
			defer client.Close()
			fn(t, client)
		})
	}
}

func makeSetCodeTx(
	t *testing.T,
	net *tests.IntegrationTestNet,
	account *tests.Account,
) *types.Transaction {
	chainID := net.GetChainId()
	client, err := net.GetClient()
	require.NoError(t, err, "failed to get client for the network")
	nonce, err := client.PendingNonceAt(t.Context(), account.Address())
	require.NoError(t, err, "failed to get nonce for the account")
	authorization, err := types.SignSetCode(account.PrivateKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(chainID),
		Address: common.Address{42},
		Nonce:   nonce + 1,
	})
	require.NoError(t, err, "failed to sign SetCode authorization")

	txData := &types.SetCodeTx{
		AuthList: []types.SetCodeAuthorization{authorization},
	}
	return tests.CreateTransaction(t, net, txData, account)
}
