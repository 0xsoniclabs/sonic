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
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestGasSubsidies_RequestIsRejectedInCaseOfInsufficientFunds(t *testing.T) {
	upgrades := opera.GetSonicUpgrades()
	upgrades.GasSubsidies = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// -------------------------------------------------------------------------

	sponsor := tests.NewAccount()
	sponsee := tests.NewAccount()
	receiver := tests.NewAccount()

	// --- send transaction although there are no sponsorship funds yet ---

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
	require.NoError(t, err)
	require.Error(t,
		client.SendTransaction(t.Context(), tx),
		"should be rejected due to lack of funds and no sponsorship",
	)

	receipt, err := net.EndowAccount(sponsor.Address(), big.NewInt(1e18))
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// --- deposit sponsorship funds ---

	registry, err := registry.NewRegistry(registry.GetAddress(), client)
	require.NoError(t, err)

	// Create a sponsorship with insufficient funds (1e15 wei).
	// The gas usage of the sponsored transaction is 134095 gas,
	// with a gas price greater than 1e10, the transaction will fail.
	donation := big.NewInt(1e15)
	receipt, err = net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		opts.Value = donation
		return registry.SponsorUser(opts, sponsee.Address(), receiver.Address())
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	require.Error(t,
		client.SendTransaction(t.Context(), tx),
		"should be rejected due to lack of funds",
	)

	// Increase the sponsorship funds by another 1e15 to 2e15 wei.
	donation = big.NewInt(1e15)
	receipt, err = net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		opts.Value = donation
		return registry.SponsorUser(opts, sponsee.Address(), receiver.Address())
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// --- submit a sponsored transaction ---

	receipt = sendSponsoredTransaction(t, client, net, tx)

	// check that the sponsorship funds got deducted
	ops := &bind.CallOpts{
		BlockNumber: receipt.BlockNumber,
	}
	sponsorship, err := registry.UserSponsorships(ops, sponsee.Address(), receiver.Address())
	require.NoError(t, err)
	require.Less(t, sponsorship.Funds.Uint64(), donation.Uint64())

}
