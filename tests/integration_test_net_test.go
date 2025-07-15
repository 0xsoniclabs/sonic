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

package tests

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/0xsoniclabs/tosca/go/tosca/vm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestIntegrationTestNet_CanStartRestartAndStopIntegrationTestNet(t *testing.T) {
	net := StartIntegrationTestNet(t)
	require.NoError(t, net.Restart(), "Failed to restart the test network")

	net.Stop()
}

func TestIntegrationTestNet_CanRestartWithGenesisExportAndImport(t *testing.T) {
	for _, numNodes := range []int{1, 2} {
		t.Run(fmt.Sprintf("NumNodes=%d", numNodes), func(t *testing.T) {
			t.Parallel()
			net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
				NumNodes: numNodes,
			})
			require.NoError(t, net.RestartWithExportImport(),
				"Failed to restart the test network with export and import")

			net.Stop()
		})
	}
}

func TestIntegrationTestNet_CanStartMultipleConsecutiveInstances(t *testing.T) {
	for range 2 {
		net := StartIntegrationTestNet(t)
		net.Stop()
	}
}

func TestIntegrationTestNet_Can(t *testing.T) {
	net := StartIntegrationTestNet(t)
	// by default, the integration test network starts with a single node
	require.Equal(t, 1, net.NumNodes())

	session1 := net.SpawnSession(t)
	session2 := net.SpawnSession(t)
	session3 := net.SpawnSession(t)

	t.Run("EndowAccountsWithTokens", func(t *testing.T) {
		t.Parallel()
		testIntegrationTestNet_CanEndowAccountsWithTokens(t, session1)
	})

	t.Run("DeployContracts", func(t *testing.T) {
		t.Parallel()
		testIntegrationTestNet_CanDeployContracts(t, session2)
	})

	t.Run("InteractWithContract", func(t *testing.T) {
		t.Parallel()
		testIntegrationTestNet_CanInteractWithContract(t, session3)
	})

	t.Run("FetchInformationFromTheNetwork", func(t *testing.T) {
		t.Parallel()
		testIntegrationTestNet_CanFetchInformationFromTheNetwork(t, net)
	})

	t.Run("SpawnParallelSessions", func(t *testing.T) {
		t.Parallel()
		testIntegrationTestNet_CanSpawnParallelSessions(t, net)
	})

	t.Run("AdvanceEpoch", func(t *testing.T) {
		t.Parallel()
		testIntegrationTestNet_AdvanceEpoch(t, net)
	})

}

func testIntegrationTestNet_CanFetchInformationFromTheNetwork(t *testing.T, net *IntegrationTestNet) {
	client, err := net.GetClient()
	require.NoError(t, err, "Failed to connect to the integration test network")
	defer client.Close()

	block, err := client.BlockNumber(t.Context())
	require.NoError(t, err, "Failed to get block number")

	require.NotZero(t, block, "Block number should not be zero")
	require.LessOrEqual(t, block, uint64(1000), "Block number should not exceed 1000")
}

// testIntegrationTestNet_CanEndowAccountsWithTokens needs its own session because it
// modifies the state of the network by endowing an account with tokens, otherwise
// it can trigger a transaction replacement with a transaction from another test.
func testIntegrationTestNet_CanEndowAccountsWithTokens(t *testing.T, session IntegrationTestNetSession) {
	client, err := session.GetClient()
	require.NoError(t, err, "Failed to connect to the integration test network")
	defer client.Close()

	address := common.Address{0x01}
	balance, err := client.BalanceAt(t.Context(), address, nil)
	require.NoError(t, err, "Failed to get balance for account")

	for i := 0; i < 10; i++ {
		increment := int64(1000)

		receipt, err := session.EndowAccount(address, big.NewInt(increment))
		require.NoError(t, err, "Failed to endow account 1")
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

		want := balance.Add(balance, big.NewInt(int64(increment)))
		balance, err = client.BalanceAt(t.Context(), address, nil)
		require.NoError(t, err, "Failed to get balance for account")
		require.Equal(t, want.Uint64(), balance.Uint64(), "Unexpected balance for account after endowment")

		balance = want
	}
}

// testIntegrationTestNet_CanDeployContracts needs its own session because it
// deploys a counter contract, and this should not overlap with other tests that
// might also deploy contracts.
func testIntegrationTestNet_CanDeployContracts(t *testing.T, session IntegrationTestNetSession) {
	_, receipt, err := DeployContract(session, counter.DeployCounter)
	require.NoError(t, err, "Failed to deploy contract")
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "Contract deployment failed")
}

// testIntegrationTestNet_CanInteractWithContract needs its own session because it
// deploys and interacts with a counter contract, and this should not overlap
// with other tests that might also deploy or interact with contracts.
func testIntegrationTestNet_CanInteractWithContract(t *testing.T, session IntegrationTestNetSession) {
	contract, _, err := DeployContract(session, counter.DeployCounter)
	require.NoError(t, err, "Failed to deploy contract")

	receipt, err := session.Apply(contract.IncrementCounter)
	require.NoError(t, err, "Failed to increment counter")
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "Counter increment failed")
}

func testIntegrationTestNet_CanSpawnParallelSessions(t *testing.T, net *IntegrationTestNet) {
	for i := range 15 {
		t.Run(fmt.Sprint("SpawnSession", i), func(t *testing.T) {
			t.Parallel()
			session := net.SpawnSession(t)

			receipt, err := session.EndowAccount(common.Address{0x42}, big.NewInt(1000))
			require.NoError(t, err)
			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
		})
	}
}

func testIntegrationTestNet_AdvanceEpoch(t *testing.T, net *IntegrationTestNet) {
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	var epochBefore hexutil.Uint64
	err = client.Client().Call(&epochBefore, "eth_currentEpoch")
	require.NoError(t, err)

	err = net.AdvanceEpoch(13)
	require.NoError(t, err)

	var epochAfter hexutil.Uint64
	err = client.Client().Call(&epochAfter, "eth_currentEpoch")
	require.NoError(t, err)

	require.Equal(t, epochBefore+13, epochAfter)
}

func TestIntegrationTestNet_CanRunMultipleNodes(t *testing.T) {
	for _, numNodes := range []int{1, 2, 3} {
		t.Run(fmt.Sprintf("NumNodes%d", numNodes), func(t *testing.T) {
			t.Parallel()
			net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
				NumNodes: numNodes,
			})
			require.Equal(t, numNodes, net.NumNodes())

			// send one transaction to check that transactions can be processed
			_, err := net.EndowAccount(common.Address{0x42}, big.NewInt(1000))
			require.NoError(t, err)

			// check that a connection to all nodes can be established and that
			// the connected nodes are indeed different nodes
			accounts := make([]string, numNodes)
			for i := range numNodes {
				client, err := net.GetClientConnectedToNode(i)
				require.NoError(t, err)
				defer client.Close()

				// by asking for the managed accounts, nodes can be identified
				res := []string{}
				require.NoError(t, client.Client().Call(&res, "eth_accounts"))
				require.NotEmpty(t, res)
				accounts[i] = res[0]
			}

			// check that all accounts are different
			seen := make(map[string]struct{})
			for _, account := range accounts {
				if _, found := seen[account]; found {
					t.Fatalf("Duplicate account %v", account)
				}
				seen[account] = struct{}{}
			}
		})
	}
}

func TestIntegrationTestNet_CanStartWithCustomConfig(t *testing.T) {

	// This test checks that configuration changes are applied to the network
	// by modifying the tx_pool configuration and checking that the transaction
	// validation behaves as expected.
	net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		ModifyConfig: func(config *config.Config) {
			// enable minimum tip check for local tx submission
			config.TxPool.NoLocals = true
			// increase minimum tip, default is 1
			config.TxPool.MinimumTip = 10
		},
	})
	client, err := net.GetClient()
	require.NoError(t, err)

	sender := makeAccountWithBalance(t, net, big.NewInt(1e18))

	chainId, err := client.ChainID(t.Context())
	require.NoError(t, err)

	gp, err := client.SuggestGasPrice(t.Context())
	require.NoError(t, err)

	gas, err := core.IntrinsicGas(nil, nil, nil, true, true, true, true)
	require.NoError(t, err)

	tx := signTransaction(t, chainId, &types.DynamicFeeTx{
		Nonce:     0,
		Value:     big.NewInt(100),
		Gas:       gas,
		GasFeeCap: gp,
		GasTipCap: big.NewInt(9),
	}, sender)
	err = client.SendTransaction(t.Context(), tx)
	require.ErrorContains(t, err, "transaction underpriced")

	tx = signTransaction(t, chainId, &types.DynamicFeeTx{
		Nonce:     1,
		Value:     big.NewInt(100),
		Gas:       gas,
		GasFeeCap: gp,
		GasTipCap: big.NewInt(10),
	}, sender)
	err = client.SendTransaction(t.Context(), tx)
	require.NoError(t, err)
}

func TestIntegrationTestNet_AccountsToBeDeployedWithGenesisCanBeCalled(t *testing.T) {
	address := common.HexToAddress("0x42")
	topic := common.Hash{0x24}
	code := []byte{byte(vm.PUSH32)}
	code = append(code, topic.Bytes()...) // topic
	code = append(code, []byte{
		byte(vm.PUSH1), 0x00, // size
		byte(vm.PUSH1), 0x00, // offset
		byte(vm.LOG1), // log
		byte(vm.STOP), // stop
	}...)
	accounts := []makefakegenesis.Account{
		{
			Name:    "account",
			Address: address,
			Code:    code,
			Nonce:   1,
		},
	}
	net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		Accounts: accounts,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	sender := makeAccountWithBalance(t, net, big.NewInt(1e18))

	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(t, err)

	chainId, err := client.ChainID(t.Context())
	require.NoError(t, err)

	txData := &types.LegacyTx{
		Nonce:    0,
		GasPrice: gasPrice,
		Gas:      50000,
		To:       &address,
		Value:    big.NewInt(0),
		Data:     []byte{},
	}
	tx := signTransaction(t, chainId, txData, sender)

	receipt, err := net.Run(tx)
	require.NoError(t, err)

	require.Equal(t, topic, receipt.Logs[0].Topics[0])

}
