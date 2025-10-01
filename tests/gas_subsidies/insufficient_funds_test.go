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

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
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

	sponsor := tests.NewAccount()
	sponsee := tests.NewAccount()
	receiver := tests.NewAccount()

	chainId := net.GetChainId()
	receiverAddress := receiver.Address()
	signer := types.LatestSignerForChainID(chainId)
	tx, err := types.SignNewTx(sponsee.PrivateKey, signer, &types.LegacyTx{
		To:       &receiverAddress,
		Gas:      21000,
		GasPrice: big.NewInt(0),
	})
	require.NoError(t, err)

	// Transfer funds to the sponsor account
	receipt, err := net.EndowAccount(sponsor.Address(), big.NewInt(1e18))
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Create a new sponsorship registry
	sponsorRegistry, err := registry.NewRegistry(registry.GetAddress(), client)
	require.NoError(t, err)

	// Calculate the price for the sponsored transaction
	cost := tx.Gas() + subsidies.SponsorshipOverheadGasCost

	// Create a new sponsorship with insufficient funds
	header, err := client.HeaderByHash(t.Context(), receipt.BlockHash)
	require.NoError(t, err)
	donation := big.NewInt(int64(cost) * header.BaseFee.Int64() / 2) // only half the required funds
	receipt, err = net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		opts.Value = donation
		return sponsorRegistry.SponsorUser(opts, sponsee.Address(), receiver.Address())
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Try to send the sponsored transaction
	require.Error(t, client.SendTransaction(t.Context(), tx), "transaction should be rejected due to insufficient funds")

	// Add the second half of the required funds
	receipt, err = net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		opts.Value = donation
		return sponsorRegistry.SponsorUser(opts, sponsee.Address(), receiver.Address())
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// get the funds before resending the transaction
	ops := &bind.CallOpts{
		BlockNumber: receipt.BlockNumber,
	}
	sponsorship, err := sponsorRegistry.UserSponsorships(ops, sponsee.Address(), receiver.Address())
	require.NoError(t, err)
	fundsBefore := sponsorship.Funds.Uint64()

	// Send the sponsored transaction
	require.NoError(t, client.SendTransaction(t.Context(), tx))
	receipt, err = net.GetReceipt(tx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// get the baseFee of the block where the tx was included
	header, err = client.HeaderByHash(t.Context(), receipt.BlockHash)
	require.NoError(t, err)
	baseFee := header.BaseFee

	ops = &bind.CallOpts{
		BlockNumber: receipt.BlockNumber,
	}
	sponsorship, err = sponsorRegistry.UserSponsorships(ops, sponsee.Address(), receiver.Address())
	require.NoError(t, err)
	fundsAfter := sponsorship.Funds.Uint64()

	require.Equal(t, fundsBefore-cost*baseFee.Uint64(), fundsAfter)
}
