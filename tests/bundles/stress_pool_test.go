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

package bundles

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/stress"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func Test_StresTxPool_WithBundles(t *testing.T) {
	// the purpose of this test is to be profiled and evaluate the usage of
	// CPU when re-evaluating bundles in the pool.

	keys := make([]*ecdsa.PrivateKey, 100_000)
	for i := range keys {
		keys[i], _ = crypto.GenerateKey()
	}
	const numberOfEnvelopes = 50

	updates := opera.GetBrioUpgrades()
	updates.TransactionBundles = true
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &updates,
		ClientExtraArguments: []string{
			// this test is not interested in initial validation but later
			// promote/demotion of bundles
			"--disable-txPool-validation",
		},
	})

	contract, receipt, err := tests.DeployContract(net, stress.DeployStress)
	require.NoError(t, err, "failed to deploy contract; %v", err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	fmt.Println("make accounts")
	accounts := tests.MakeAccountsWithBalance(t, net, numberOfEnvelopes, big.NewInt(1e18))

	client, err := net.GetClient()
	require.NoError(t, err, "failed to get client")
	defer client.Close()

	// setting gas limit of the transaction the maximum allowed
	// forces them to not be included in the same block, and extends
	// their lifetime in the pool, which allows to better observe the behavior of the pool with bundles
	gasLimit := CurrentMaxGasLimit(t, net)

	fmt.Println("make bundles")
	wg := sync.WaitGroup{}
	envelopes := make([]*types.Transaction, numberOfEnvelopes)
	for i := range numberOfEnvelopes {
		wg.Go(func() {
			opts, err := net.GetTransactOptions(accounts[0])
			require.NoError(t, err)
			opts.NoSend = true
			opts.GasLimit = gasLimit
			// number of iterations is empirically chosen to fill the
			// available gas with the
			tx, err := contract.ComputeHeavySum(opts, big.NewInt(12500))
			require.NoError(t, err)

			envelope := bundle.NewBuilder().
				With(bundle.Step(accounts[i].PrivateKey, tx)).
				Build()

			require.GreaterOrEqual(t, envelope.Gas(), gasLimit)
			envelopes[i] = envelope
		})
	}
	wg.Wait()

	_, err = net.SendAll(envelopes)
	require.NoError(t, err, "failed to send bundles")

	// Because of the gas limit for each transaction, it will take a long
	// time to execute all of them, just wait a little, user shall monitor the CPU usage
	// of the txPool reorg.
	time.Sleep(5 * time.Second)
}

func Test_StressContract_FindMaxNumberOfRounds(t *testing.T) {
	updates := opera.GetBrioUpgrades()
	updates.TransactionBundles = true
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &updates,
	})

	contract, receipt, err := tests.DeployContract(net, stress.DeployStress)
	require.NoError(t, err, "failed to deploy contract; %v", err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	gasLimit := CurrentMaxGasLimit(t, net)
	rounds := 100_000
	for {
		receipt, err = net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
			opts.GasLimit = gasLimit
			return contract.ComputeHeavySum(opts, big.NewInt(int64(rounds)))
		})
		require.NoError(t, err, "failed to call ComputeHeavySum without bundles")

		if receipt.Status == types.ReceiptStatusSuccessful {
			break
		}
		rounds /= 2
	}
	// the stress test uses this value as maximum number of iteration for the maximium gas limit
	require.Equal(t, 12500, rounds)
}

// CurrentMaxGasLimit returns the maximum gas limit that can be used for a
// transaction in the current network configuration.
//
// It duplicates the max gas limit calculation for
func CurrentMaxGasLimit(t testing.TB, net *tests.IntegrationTestNet) uint64 {
	t.Helper()

	client, err := net.GetClient()
	require.NoError(t, err, "failed to get client")
	defer client.Close()

	var rules opera.Rules
	err = client.Client().Call(&rules, "eth_getRules", "latest")
	require.NoError(t, err, "failed to get rules")

	return gossip.ComputeMaxGasLimit(rules)
}
