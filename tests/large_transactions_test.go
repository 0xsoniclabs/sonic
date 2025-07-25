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
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestLargeTransactions_CanHandleLargeTransactions(t *testing.T) {
	require := require.New(t)
	net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		Upgrades: AsPointer(opera.GetAllegroUpgrades()),
	})

	account := NewAccount()
	_, err := net.EndowAccount(account.Address(), big.NewInt(1e18))
	require.NoError(err)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	chainId, err := client.ChainID(t.Context())
	require.NoError(err, "failed to get chain ID")

	target := common.Address{}

	// The smallest possible transaction passes.
	receipt, err := net.Run(types.MustSignNewTx(
		account.PrivateKey,
		types.NewCancunSigner(chainId),
		&types.AccessListTx{
			ChainID:  chainId,
			Gas:      21_000,
			GasPrice: big.NewInt(1e11),
			To:       &target,
			Nonce:    0,
		},
	))
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// A large transaction passes as well.
	receipt, err = net.Run(types.MustSignNewTx(
		account.PrivateKey,
		types.NewCancunSigner(chainId),
		&types.AccessListTx{
			ChainID:  chainId,
			Gas:      2_000_000,
			GasPrice: big.NewInt(1e11),
			To:       &target,
			Nonce:    1,
			Data:     make([]byte, 100_000), // 100 KB of data
		},
	))
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// A too large transaction fails to be accepted by the pool.
	_, err = net.Run(types.MustSignNewTx(
		account.PrivateKey,
		types.NewCancunSigner(chainId),
		&types.AccessListTx{
			ChainID:  chainId,
			Gas:      4_000_000,
			GasPrice: big.NewInt(1e11),
			To:       &target,
			Nonce:    1,
			Data:     make([]byte, 200_000), // 200 KB of data
		},
	))
	require.ErrorContains(err, "oversized data")
}

func TestLargeTransactions_LargeTransactionLoadTest(t *testing.T) {

	if isDataRaceDetectionEnabled() {
		t.Skip(`Due to the concurrency requirements of this test, 
		it becomes unstable when running with enabled data race detection.`)
	}

	hardForks := map[string]opera.Upgrades{
		"Sonic":   opera.GetSonicUpgrades(),
		"Allegro": opera.GetAllegroUpgrades(),
	}

	modes := map[string]bool{
		"DistributedProposer": false,
		"SingleProposer":      true,
	}

	for name, upgrades := range hardForks {
		for mode, singleProposer := range modes {
			t.Run(fmt.Sprintf("%s/%s", name, mode), func(t *testing.T) {
				effectiveUpgrades := upgrades
				effectiveUpgrades.SingleProposerBlockFormation = singleProposer
				testLargeTransactionLoadTest(t, &effectiveUpgrades)
			})
		}
	}
}

func testLargeTransactionLoadTest(
	t *testing.T,
	upgrades *opera.Upgrades,
) {
	// The aim of this test is to flood the network with large transactions to
	// trigger the production of messages exceeding the maximum limit of 10 MB.
	// If this happens, events are not forwarded between nodes, leading to a
	// network stall -- observable by the fact that the transactions are not
	// processed and no receipts are produced. This test ensures that the
	// network can handle such a load without stalling.
	const (
		numAccounts = 50
		numRounds   = 10
	)
	require := require.New(t)
	net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		Upgrades: upgrades,
		NumNodes: 3,
	})

	// Increase the gas limit to allow for larger transactions in blocks. These
	// limits are beyond safe limits acceptable for production.
	current := getNetworkRules(t, net)

	modified := current.Copy()
	modified.Economy.Gas.MaxEventGas = 1_000_000_000
	modified.Economy.ShortGasPower.AllocPerSec = 20_000_000_000
	modified.Economy.ShortGasPower.MaxAllocPeriod = 50_000_000_000
	modified.Economy.LongGasPower = modified.Economy.ShortGasPower
	modified.Emitter.Interval = 200_000_000 // low a bit down to provoke larger events
	updateNetworkRules(t, net, modified)
	require.NoError(net.AdvanceEpoch(1))

	// Check that the modification was applied.
	current = getNetworkRules(t, net)
	require.Equal(modified, current)

	// Create accounts and provide them with funds to run the load test.
	accounts := make([]*Account, numAccounts)
	addresses := make([]common.Address, len(accounts))
	for i := range accounts {
		accounts[i] = NewAccount()
		addresses[i] = accounts[i].Address()
	}
	endowment := new(big.Int).Mul(big.NewInt(100), big.NewInt(1e18))
	_, err := net.EndowAccounts(addresses, endowment)
	require.NoError(err)

	chainId := net.GetChainId()
	signer := types.NewCancunSigner(chainId)

	// Create a list of large transactions to flood the network.
	transactions := []*types.Transaction{}
	data := make([]byte, 125_000)
	gas := uint64(len(data))*params.TxCostFloorPerToken + 21_000
	for nonce := range uint64(numRounds) {
		for i := range accounts {
			tx := types.MustSignNewTx(
				accounts[i].PrivateKey,
				signer,
				&types.AccessListTx{
					ChainID:  chainId,
					Gas:      gas,
					GasPrice: big.NewInt(1e11),
					To:       &common.Address{0x42},
					Nonce:    nonce,
					Data:     data,
				},
			)
			transactions = append(transactions, tx)
		}
	}

	// Send the enabling transactions with the low nonces last to maximize the
	// load peak.
	slices.Reverse(transactions)

	receipts, err := net.RunAll(transactions)
	require.NoError(err, "failed to run transactions")
	for _, receipt := range receipts {
		require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
	}
}
