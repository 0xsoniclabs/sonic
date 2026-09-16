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
	"context"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/gossip/contract/sfc100"
	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/contracts/sfc"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/0xsoniclabs/tosca/go/tosca/vm"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
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

	t.Run("EndowAccountsWithTokens", func(t *testing.T) {
		session := net.SpawnSession(t)
		t.Parallel()
		testIntegrationTestNet_CanEndowAccountsWithTokens(t, session)
	})

	t.Run("DeployContracts", func(t *testing.T) {
		session := net.SpawnSession(t)
		t.Parallel()
		testIntegrationTestNet_CanDeployContracts(t, session)
	})

	t.Run("InteractWithContract", func(t *testing.T) {
		session := net.SpawnSession(t)
		t.Parallel()
		testIntegrationTestNet_CanInteractWithContract(t, session)
	})

	t.Run("FetchInformationFromTheNetwork", func(t *testing.T) {
		t.Parallel()
		testIntegrationTestNet_CanFetchInformationFromTheNetwork(t, net)
	})

	t.Run("SpawnParallelSessions", func(t *testing.T) {
		session := net.SpawnSession(t)
		t.Parallel()
		testIntegrationTestNet_CanSpawnParallelSessions(t, session)
	})

	t.Run("ForceEmitTransactions", func(t *testing.T) {
		session := net.SpawnSession(t)
		testIntegrationTestNet_CanForceEmitTransactions(t, session)
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

		// The state of the block containing the transaction must be
		// available as soon as the receipt is returned; no extra waiting.
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

func testIntegrationTestNet_CanSpawnParallelSessions(t *testing.T, session IntegrationTestNetSession) {
	for i := range 15 {
		t.Run(fmt.Sprint("SpawnSession", i), func(t *testing.T) {
			receipt, err := session.EndowAccount(common.Address{0x42}, big.NewInt(1000))
			require.NoError(t, err)
			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
		})
	}
}

func testIntegrationTestNet_CanForceEmitTransactions(t *testing.T, session IntegrationTestNetSession) {
	account := MakeAccountWithBalance(t, session, big.NewInt(1e18))

	signer := types.LatestSignerForChainID(session.GetChainId())
	tx := types.MustSignNewTx(account.PrivateKey, signer, &types.LegacyTx{
		Nonce:    0,
		To:       &common.Address{},
		Value:    big.NewInt(0),
		Gas:      21000,
		GasPrice: big.NewInt(enoughGasPrice),
	})

	hash, err := session.ForceEmit(t.Context(), tx)
	require.NoError(t, err, "Failed to force emit transaction")

	receipt, err := session.GetReceipt(hash)
	require.NoError(t, err, "Failed to get receipt for forced transaction")
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
}

func TestIntegrationTestNet_AdvanceEpoch(t *testing.T) {

	net := StartIntegrationTestNet(t)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	var epochBefore hexutil.Uint64
	err = client.Client().Call(&epochBefore, "eth_currentEpoch")
	require.NoError(t, err)

	net.AdvanceEpoch(t, 13)

	var epochAfter hexutil.Uint64
	err = client.Client().Call(&epochAfter, "eth_currentEpoch")
	require.NoError(t, err)

	require.Equal(t, epochBefore+13, epochAfter)
}

func TestIntegrationTestNet_CanRunMultipleNodes(t *testing.T) {
	for _, numNodes := range []int{1, 2, 3} {
		t.Run(fmt.Sprintf("NumNodes%d", numNodes), func(t *testing.T) {
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

	sender := MakeAccountWithBalance(t, net, big.NewInt(1e18))

	chainId, err := client.ChainID(t.Context())
	require.NoError(t, err)

	gp, err := client.SuggestGasPrice(t.Context())
	require.NoError(t, err)

	gas, err := core.IntrinsicGas(nil, nil, nil, true, true, true, true)
	require.NoError(t, err)

	tx := SignTransaction(t, chainId, &types.DynamicFeeTx{
		Nonce:     0,
		Value:     big.NewInt(100),
		Gas:       gas,
		GasFeeCap: gp,
		GasTipCap: big.NewInt(9),
	}, sender)
	err = client.SendTransaction(t.Context(), tx)
	require.ErrorContains(t, err, "transaction underpriced")

	tx = SignTransaction(t, chainId, &types.DynamicFeeTx{
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

	sender := MakeAccountWithBalance(t, net, big.NewInt(1e18))

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
	tx := SignTransaction(t, chainId, txData, sender)

	receipt, err := net.Run(tx)
	require.NoError(t, err)

	require.Equal(t, topic, receipt.Logs[0].Topics[0])
}

func TestIntegrationTestNet_CanDefineValidatorsStakes(t *testing.T) {

	tests := map[string]struct {
		stakes         []uint64
		expectedStakes []uint64
	}{
		"default unspecified stakes": {
			stakes:         nil,
			expectedStakes: []uint64{5_000_000},
		},
		"multiple validators with different stakes": {
			stakes:         []uint64{50, 20, 11, 1},
			expectedStakes: []uint64{50, 20, 11, 1},
		},
		"multiple validators with equal stakes": {
			stakes:         makefakegenesis.CreateEqualValidatorStake(3),
			expectedStakes: []uint64{5_000_000, 5_000_000, 5_000_000},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
				ValidatorsStake: test.stakes,
			})

			client, err := net.GetClient()
			require.NoError(t, err)
			defer client.Close()

			require.Equal(t, len(test.expectedStakes), net.NumNodes(),
				"The number of nodes in the network does not match the expected number of validators")

			sfc, err := sfc100.NewContract(sfc.ContractAddress, client)
			require.NoError(t, err)

			epoch, err := sfc.CurrentEpoch(nil)
			require.NoError(t, err)

			validatorIDs, err := sfc.GetEpochValidatorIDs(nil, epoch)
			require.NoError(t, err)
			require.Len(t, validatorIDs, len(test.expectedStakes),
				"The number of validators with stakes in the SFC does not match the expected number of validators")
			for i, validatorID := range validatorIDs {
				stake, err := sfc.GetSelfStake(nil, validatorID)
				require.NoError(t, err)

				expectedStake := utils.ToFtm(test.expectedStakes[i])
				require.Conditionf(t,
					func() bool {
						return expectedStake.Cmp(stake) == 0
					},
					"unexpected stake for validator %d: expected %v, got %v",
					i,
					expectedStake,
					stake,
				)
			}
		})
	}
}

func TestIntegrationTestNet_ValidateAndSanitizeOptions(t *testing.T) {

	tests := map[string]struct {
		options         []IntegrationTestNetOptions
		expectedOptions IntegrationTestNetOptions
		expectError     string
	}{
		"when multiple options are provided, error is returned": {
			options: []IntegrationTestNetOptions{
				{},
				{},
			},
			expectError: "expected at most one option, got 2",
		},
		"if upgrades is defined, it is preserved": {
			options: []IntegrationTestNetOptions{
				{
					Upgrades: AsPointer(opera.GetAllegroUpgrades()),
				},
			},
			expectedOptions: IntegrationTestNetOptions{
				Upgrades:        AsPointer(opera.GetAllegroUpgrades()),
				NumNodes:        1,
				ValidatorsStake: []uint64{5_000_000},
			},
		},
		"when left empty, it defaults to the default options": {
			options: []IntegrationTestNetOptions{},
			expectedOptions: IntegrationTestNetOptions{
				Upgrades:        AsPointer(opera.GetSonicUpgrades()),
				NumNodes:        1,
				ValidatorsStake: []uint64{5_000_000},
			},
		},
		"when NumNodes is defined, ValidatorsStake is initialized to be uniform": {
			options: []IntegrationTestNetOptions{
				{
					NumNodes: 3,
				},
			},
			expectedOptions: IntegrationTestNetOptions{
				Upgrades:        AsPointer(opera.GetSonicUpgrades()),
				NumNodes:        3,
				ValidatorsStake: []uint64{5_000_000, 5_000_000, 5_000_000},
			},
		},
		"when ValidatorsStake length does not match NumNodes, an error is returned": {
			options: []IntegrationTestNetOptions{
				{
					NumNodes:        2,
					ValidatorsStake: []uint64{5_000_000},
				},
			},
			expectError: "number of nodes (2) does not match number of validator stakes (1)",
		},
		"when only ValidatorsStake is defined, NumNodes is set accordingly": {
			options: []IntegrationTestNetOptions{
				{
					ValidatorsStake: []uint64{10, 20, 30},
				},
			},
			expectedOptions: IntegrationTestNetOptions{
				Upgrades:        AsPointer(opera.GetSonicUpgrades()),
				NumNodes:        3,
				ValidatorsStake: []uint64{10, 20, 30},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			options, err := validateAndSanitizeOptions(test.options...)
			if len(test.expectError) > 0 {
				require.ErrorContains(t, err, test.expectError)
				return
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, test.expectedOptions, options)
		})
	}

}

// TestIntegrationTestNet_StateIsAvailableWhenReceiptIsReturned checks that all
// helpers returning receipts only return once the state of the block containing
// the transaction is served by state-dependent RPC calls at "latest". Receipts
// are served from the block store, which is updated before the archive state
// becomes available. Without waiting for the archive, calls like
// eth_estimateGas from a freshly funded account would fail with
// "insufficient funds for transfer".
func TestIntegrationTestNet_StateIsAvailableWhenReceiptIsReturned(t *testing.T) {
	net := StartIntegrationTestNet(t)

	// Each session is funded with MaxUint64 wei; keep the total of all
	// repetitions well below that budget.
	const numRepetitions = 20
	value := big.NewInt(1e17)

	// Each case funds the given account and returns the receipt of the funding
	// transaction, using a different helper each time.
	testCases := map[string]struct {
		fund func(t *testing.T, session IntegrationTestNetSession, address common.Address) *types.Receipt
	}{
		"EndowAccount": {
			fund: func(t *testing.T, session IntegrationTestNetSession, address common.Address) *types.Receipt {
				receipt, err := session.EndowAccount(address, value)
				require.NoError(t, err)
				return receipt
			},
		},
		"EndowAccounts": {
			fund: func(t *testing.T, session IntegrationTestNetSession, address common.Address) *types.Receipt {
				receipts, err := session.EndowAccounts([]common.Address{address}, value)
				require.NoError(t, err)
				require.Len(t, receipts, 1)
				return receipts[0]
			},
		},
		"Run": {
			fund: func(t *testing.T, session IntegrationTestNetSession, address common.Address) *types.Receipt {
				receipt, err := session.Run(makeTransferTx(t, session, address, value))
				require.NoError(t, err)
				return receipt
			},
		},
		"RunAll": {
			fund: func(t *testing.T, session IntegrationTestNetSession, address common.Address) *types.Receipt {
				receipts, err := session.RunAll([]*types.Transaction{makeTransferTx(t, session, address, value)})
				require.NoError(t, err)
				require.Len(t, receipts, 1)
				return receipts[0]
			},
		},
		"Send and GetReceipt": {
			fund: func(t *testing.T, session IntegrationTestNetSession, address common.Address) *types.Receipt {
				hash, err := session.Send(makeTransferTx(t, session, address, value))
				require.NoError(t, err)
				receipt, err := session.GetReceipt(hash)
				require.NoError(t, err)
				return receipt
			},
		},
		"Send and TryGetReceipt": {
			fund: func(t *testing.T, session IntegrationTestNetSession, address common.Address) *types.Receipt {
				hash, err := session.Send(makeTransferTx(t, session, address, value))
				require.NoError(t, err)
				receipt, err := session.TryGetReceipt(100*time.Second, hash)
				require.NoError(t, err)
				return receipt
			},
		},
		"MakeAccountWithBalance": {
			fund: func(t *testing.T, session IntegrationTestNetSession, address common.Address) *types.Receipt {
				// MakeAccountWithBalance creates its own account; the address
				// parameter is ignored and the receipt is not exposed. The
				// state checks below are performed on the created account.
				return nil
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			session := net.SpawnSession(t)
			t.Parallel()

			client, err := session.GetClient()
			require.NoError(t, err)
			defer client.Close()

			for range numRepetitions {
				account := NewAccount()
				address := account.Address()

				receipt := test.fund(t, session, address)
				if receipt == nil {
					account = MakeAccountWithBalance(t, session, value)
					address = account.Address()
				} else {
					require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
					// The state of the block of the receipt must be available.
					balance, err := client.BalanceAt(t.Context(), address, receipt.BlockNumber)
					require.NoError(t, err, "state of block %v not available", receipt.BlockNumber)
					require.Equal(t, value, balance)
				}

				// The latest state must include the funding transaction.
				balance, err := client.BalanceAt(t.Context(), address, nil)
				require.NoError(t, err)
				require.Equal(t, value, balance, "latest state does not include the funding")

				// Gas estimation must see the funds of the new account. This
				// is the operation that failed before the receipt helpers
				// started to wait for the archive state.
				gasPrice, err := client.SuggestGasPrice(t.Context())
				require.NoError(t, err)
				_, err = client.EstimateGas(t.Context(), ethereum.CallMsg{
					From:     address,
					To:       &common.Address{},
					GasPrice: gasPrice,
					Value:    big.NewInt(1),
				})
				require.NoError(t, err, "gas estimation does not see the funds of the new account")
			}
		})
	}
}

// makeTransferTx creates a signed transaction transferring the given value
// from the session sponsor to the given address.
func makeTransferTx(t *testing.T, session IntegrationTestNetSession, to common.Address, value *big.Int) *types.Transaction {
	t.Helper()
	return CreateTransaction(t, session, &types.LegacyTx{
		To:    &to,
		Value: value,
	}, session.GetSessionSponsor())
}

// latestBlockProbeService is a fake implementation of the eth_blockNumber and
// eth_getBalance RPC methods. It simulates a node whose latest block number
// lags behind for a configurable number of calls, and whose state for the
// probed block is unavailable for a configurable number of calls.
type latestBlockProbeService struct {
	mu sync.Mutex

	block           uint64 // the block the probe is waiting for
	behindCalls     int    // number of eth_blockNumber calls reporting block-1
	noStateCalls    int    // number of eth_getBalance calls failing
	blockNumberSeen int    // number of eth_blockNumber calls received
	getBalanceSeen  int    // number of eth_getBalance calls received
}

func (s *latestBlockProbeService) BlockNumber() hexutil.Uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blockNumberSeen++
	if s.blockNumberSeen <= s.behindCalls {
		return hexutil.Uint64(s.block - 1)
	}
	return hexutil.Uint64(s.block)
}

func (s *latestBlockProbeService) GetBalance(_ context.Context, _ common.Address, block rpc.BlockNumberOrHash) (*hexutil.Big, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getBalanceSeen++
	if s.getBalanceSeen <= s.noStateCalls {
		number, _ := block.Number()
		return nil, fmt.Errorf("block %d is not present in the archive", number)
	}
	return (*hexutil.Big)(big.NewInt(0)), nil
}

func (s *latestBlockProbeService) calls() (blockNumber, getBalance int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.blockNumberSeen, s.getBalanceSeen
}

// newLatestBlockProbeClient creates a client connected to an in-process RPC
// server serving the given fake eth service.
func newLatestBlockProbeClient(t *testing.T, service *latestBlockProbeService) *PooledEhtClient {
	t.Helper()
	server := rpc.NewServer()
	t.Cleanup(server.Stop)
	require.NoError(t, server.RegisterName("eth", service))
	rpcClient := rpc.DialInProc(server)
	t.Cleanup(rpcClient.Close)
	return &PooledEhtClient{ethClient: *ethclient.NewClient(rpcClient)}
}

func TestWaitUntilBlockStateIsLatest_ReturnsOnceLatestBlockAndStateAreAvailable(t *testing.T) {
	const block = 42
	testCases := map[string]struct {
		behindCalls  int
		noStateCalls int
		// expected number of calls to the respective RPC methods
		blockNumberCalls int
		getBalanceCalls  int
	}{
		"available immediately": {
			blockNumberCalls: 1,
			getBalanceCalls:  1,
		},
		"latest block behind for one probe": {
			behindCalls:      1,
			blockNumberCalls: 2,
			getBalanceCalls:  1,
		},
		"latest block behind for several probes": {
			behindCalls:      4,
			blockNumberCalls: 5,
			getBalanceCalls:  1,
		},
		"state unavailable for one probe": {
			noStateCalls:     1,
			blockNumberCalls: 2,
			getBalanceCalls:  2,
		},
		"state unavailable for several probes": {
			noStateCalls:     3,
			blockNumberCalls: 4,
			getBalanceCalls:  4,
		},
		"latest block behind, then state unavailable": {
			behindCalls:      2,
			noStateCalls:     2,
			blockNumberCalls: 5,
			getBalanceCalls:  3,
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			service := &latestBlockProbeService{
				block:        block,
				behindCalls:  test.behindCalls,
				noStateCalls: test.noStateCalls,
			}
			client := newLatestBlockProbeClient(t, service)

			err := waitUntilBlockStateIsLatest(t.Context(), client, big.NewInt(block))
			require.NoError(t, err)
			blockNumberCalls, getBalanceCalls := service.calls()
			require.Equal(t, test.blockNumberCalls, blockNumberCalls, "unexpected number of eth_blockNumber calls")
			require.Equal(t, test.getBalanceCalls, getBalanceCalls, "unexpected number of eth_getBalance calls")
		})
	}
}

func TestWaitUntilBlockStateIsLatest_ReportsContextAndLastProbeError(t *testing.T) {
	const block = 42
	never := int(^uint(0) >> 1)

	deadlineContext := func(t *testing.T) context.Context {
		ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
		t.Cleanup(cancel)
		return ctx
	}
	canceledContext := func(t *testing.T) context.Context {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		return ctx
	}

	testCases := map[string]struct {
		makeContext   func(t *testing.T) context.Context
		behindCalls   int
		noStateCalls  int
		expectedError error
		expectedText  string
	}{
		"latest block never advances": {
			makeContext:   deadlineContext,
			behindCalls:   never,
			expectedError: context.DeadlineExceeded,
			expectedText:  "latest block 41 is behind block 42",
		},
		"state never available": {
			makeContext:   deadlineContext,
			noStateCalls:  never,
			expectedError: context.DeadlineExceeded,
			expectedText:  "state of block 42 is not available: block 42 is not present in the archive",
		},
		"context canceled before the first probe": {
			makeContext:   canceledContext,
			behindCalls:   never,
			expectedError: context.Canceled,
			// the RPC call itself fails with the context error
			expectedText: "failed to get latest block number",
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			service := &latestBlockProbeService{
				block:        block,
				behindCalls:  test.behindCalls,
				noStateCalls: test.noStateCalls,
			}
			client := newLatestBlockProbeClient(t, service)

			err := waitUntilBlockStateIsLatest(test.makeContext(t), client, big.NewInt(block))
			require.ErrorIs(t, err, test.expectedError)
			require.ErrorContains(t, err, test.expectedText,
				"last probe error should be reported")
		})
	}
}

func BenchmarkIntegrationTestNet_StartAndStop(b *testing.B) {
	for i := 0; i < b.N; i++ {
		net := StartIntegrationTestNet(b)
		b.StopTimer()
		net.Stop()
	}
}
