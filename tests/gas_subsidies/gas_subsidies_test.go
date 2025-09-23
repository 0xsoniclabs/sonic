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
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestGasSubsidies_CanRunSubsidizedTransactions(t *testing.T) {
	t.Parallel()
	t.Run("single proposer", func(t *testing.T) {
		t.Parallel()
		testCanRunSubsidizedTransactions(t, true)
	})
	t.Run("distributed proposer", func(t *testing.T) {
		t.Parallel()
		testCanRunSubsidizedTransactions(t, false)
	})
}

func testCanRunSubsidizedTransactions(t *testing.T, singleProposer bool) {
	require := require.New(t)

	upgrades := opera.GetSonicUpgrades()
	upgrades.SingleProposerBlockFormation = singleProposer
	upgrades.GasSubsidies = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		//NumNodes: 3,
		ModifyConfig: func(config *config.Config) {
			// The transaction to deploy the subsidies registry contract has
			// chain id 0, and is thus not replay protected. To be able to
			// submit it, we need to allow unprotected transactions in the
			// transaction pool.
			config.Opera.AllowUnprotectedTxs = true
		},
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	sponsor := tests.NewAccount()
	sponsee := tests.NewAccount()
	receiver := tests.NewAccount()

	// Before the sponsorship is set up, a transaction from the sponsee
	// to the receiver should fail due to lack of funds.
	chainId := net.GetChainId()
	receiverAddress := receiver.Address()
	signer := types.LatestSignerForChainID(chainId)
	tx, err := types.SignNewTx(sponsee.PrivateKey, signer, &types.LegacyTx{
		To:       &receiverAddress,
		Gas:      21000,
		GasPrice: big.NewInt(0),
	})
	require.NoError(err)
	require.Error(
		client.SendTransaction(t.Context(), tx),
		"should be rejected due to lack of funds and no sponsorship",
	)

	// --- deposit sponsorship funds ---

	donation := big.NewInt(1e16)
	ledger := createRegistryWithDonation(t, client, net, sponsor, sponsee, receiver, donation)

	burnedBefore, err := client.BalanceAt(t.Context(), common.Address{}, nil)
	require.NoError(err)

	// --- submit a sponsored transaction ---

	receipt := sendSponsoredTransaction(t, client, net, tx)

	// check that the sponsorship funds got deducted
	ops := &bind.CallOpts{
		BlockNumber: receipt.BlockNumber,
	}
	sponsorship, err := ledger.UserSponsorships(ops, sponsee.Address(), receiver.Address())
	require.NoError(err)
	require.Less(sponsorship.Funds.Uint64(), donation.Uint64())

	// the difference in the sponsorship funds should have been burned
	burnedAfter, err := client.BalanceAt(t.Context(), common.Address{}, nil)
	require.NoError(err)
	require.Greater(burnedAfter.Uint64(), burnedBefore.Uint64())

	// the sponsorship difference and the increase in burned funds should be equal
	diff := new(big.Int).Sub(burnedAfter, burnedBefore)
	reduced := new(big.Int).Sub(donation, sponsorship.Funds)
	require.Equal(0, diff.Cmp(reduced),
		"the burned amount should equal the reduction of the sponsorship funds",
	)
}

func createRegistryWithDonation(t *testing.T, client *tests.PooledEhtClient, net *tests.IntegrationTestNet,
	sponsor, sponsee, receiver *tests.Account, donation *big.Int) *registry.Registry {
	registry, err := registry.NewRegistry(registry.GetAddress(), client)
	require.NoError(t, err)

	receipt, err := net.EndowAccount(sponsor.Address(), big.NewInt(1e18))
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	receipt, err = net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		opts.Value = big.NewInt(1e16)
		return registry.SponsorUser(opts, sponsee.Address(), receiver.Address())
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// check that the sponsorship funds got deposited
	sponsorship, err := registry.UserSponsorships(nil, sponsee.Address(), receiver.Address())
	require.NoError(t, err)
	require.Equal(t, donation, sponsorship.Funds)

	return registry
}

func sendSponsoredTransaction(t *testing.T, client *tests.PooledEhtClient, net *tests.IntegrationTestNet, tx *types.Transaction) *types.Receipt {
	require.NoError(t, client.SendTransaction(t.Context(), tx))

	// Wait for the sponsored transaction to be executed.
	receipt, err := net.GetReceipt(tx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
	require.NoError(t, err)
	require.True(t, slices.ContainsFunc(
		block.Transactions(),
		func(cur *types.Transaction) bool {
			return cur.Hash() == tx.Hash()
		},
	))

	// Check that the payment transaction is included right after the sponsored
	// transaction and that it was successful and has a non-zero value.
	found := false
	for i, tx := range block.Transactions() {
		if tx.Hash() == receipt.TxHash {
			require.Less(t, i, len(block.Transactions()))
			payment := block.Transactions()[i+1]
			receipt, err := net.GetReceipt(payment.Hash())
			require.NoError(t, err)
			require.Less(t, receipt.GasUsed, uint64(100_000))
			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
			found = true
			break
		}
	}
	require.True(t, found, "sponsored transaction not found in the block")

	return receipt
}

// TODO: test the following properties
//  - sponsorship requests work with all types of transactions (legacy, dynamic fee, etc.)
//  - check the enforcement of the GasSponsorship flag in the network rules
//  - check that the sponsorship funds are correctly deducted after a sponsored tx
//  - check that the sponsorship request is rejected if there are insufficient funds
//  - check that the sponsorship request is rejected if the registry contract is not deployed
//  - test that fee charging transactions and sealing transactions use proper nonces (incrementally, no gaps)
//  - test cumulative gas usage of multiple sponsored transactions in a block
//  - test receipt to transaction mapping in blocks with (multiple) sponsored transactions and internal transactions
//  - test correct log message indexing in blocks with (multiple) sponsored transactions and internal transactions
//  - test correct nonce usage of internal transactions (pre-, post- and fee charging transactions

func TestGasSubsidies_CanBeEnabledAndDisabled(
	t *testing.T,
) {
	require := require.New(t)

	// The network is initially started using the distributed protocol.
	net := tests.StartIntegrationTestNet(t)
	// a sliced is used here to ensure the forks get updated in an acceptable order.
	upgrades := []struct {
		name    string
		upgrade opera.Upgrades
	}{
		{name: "sonic", upgrade: opera.GetSonicUpgrades()},
		{name: "allegro", upgrade: opera.GetAllegroUpgrades()},
		//{name: "brio", upgrade: opera.GetBrioUpgrades()},
	}
	for _, test := range upgrades {
		t.Run(test.name, func(t *testing.T) {
			client, err := net.GetClient()
			require.NoError(err)
			defer client.Close()

			// enforce the current upgrade
			tests.UpdateNetworkRules(t, net, test.upgrade)
			// Advance the epoch by one to apply the change.
			net.AdvanceEpoch(t, 1)

			// check original state
			type upgrades struct {
				GasSubsidies bool
			}
			type rulesType struct {
				Upgrades upgrades
			}

			var originalRules rulesType
			err = client.Client().Call(&originalRules, "eth_getRules", "latest")
			require.NoError(err)
			require.Equal(false, originalRules.Upgrades.GasSubsidies, "GasSubsidies should be disabled initially")

			// Enable gas subsidies.
			rulesDiff := rulesType{
				Upgrades: upgrades{GasSubsidies: true},
			}
			tests.UpdateNetworkRules(t, net, rulesDiff)

			// Advance the epoch by one to apply the change.
			net.AdvanceEpoch(t, 1)

			err = client.Client().Call(&originalRules, "eth_getRules", "latest")
			require.NoError(err)
			require.Equal(true, originalRules.Upgrades.GasSubsidies, "GasSubsidies should be enabled after the update")

			// Disable gas subsidies.
			rulesDiff = rulesType{
				Upgrades: upgrades{GasSubsidies: false},
			}
			tests.UpdateNetworkRules(t, net, rulesDiff)

			// Advance the epoch by one to apply the change.
			net.AdvanceEpoch(t, 1)

			err = client.Client().Call(&originalRules, "eth_getRules", "latest")
			require.NoError(err)
			require.Equal(false, originalRules.Upgrades.GasSubsidies, "GasSubsidies should be disabled after the update")
		})
	}
}
