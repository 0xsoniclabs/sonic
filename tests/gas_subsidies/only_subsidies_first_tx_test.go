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
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestGasSubsidies_SkippedTransactionsAreNotSubsidized(t *testing.T) {
	sponsor := tests.NewAccount()
	sponsee := tests.NewAccount()
	receiver := tests.NewAccount()

	upgrades := opera.GetAllegroUpgrades()
	upgrades.GasSubsidies = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
		ClientExtraArguments: []string{
			"--disable-txPool-validation",
		},
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	donation := big.NewInt(1e16)
	sponsorRegistry := createRegistryWithDonation(t, client, net, sponsor, sponsee, receiver, donation)

	// --- send a subsidized transaction ---

	receiverAddress := receiver.Address()
	txData := &types.LegacyTx{
		Nonce:    0,
		GasPrice: nil,
		Gas:      21000,
		To:       &receiverAddress,
	}

	signer := types.LatestSignerForChainID(net.GetChainId())
	signedTx, err := types.SignNewTx(sponsee.PrivateKey, signer, txData)
	require.NoError(t, err)

	require.NoError(t, client.SendTransaction(t.Context(), signedTx))

	for range 5 {
		require.Error(t, client.SendTransaction(t.Context(), signedTx))
	}
	// Wait for the sponsored transaction to be executed.
	receipt, err := net.GetReceipt(signedTx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
	require.NoError(t, err)

	// Make sure only one instance of the transaction is in the block and the skipped
	// transaction is not included.
	found := false
	for _, tx := range block.Transactions() {
		if tx.Hash() == signedTx.Hash() {
			require.False(t, found, "transaction found multiple times in the block")
			found = true
		}
	}
	require.True(t, found, "transaction not found in the block")

	// check that the sponsorship funds got deducted only once
	ops := &bind.CallOpts{
		BlockNumber: big.NewInt(0).Sub(receipt.BlockNumber, big.NewInt(1)),
	}
	sponsorship, err := sponsorRegistry.UserSponsorships(ops, sponsee.Address(), receiver.Address())
	require.NoError(t, err)

	fundsBefore := sponsorship.Funds.Uint64()
	require.Equal(t, donation.Uint64(), fundsBefore)

	ops = &bind.CallOpts{
		BlockNumber: receipt.BlockNumber,
	}
	sponsorship, err = sponsorRegistry.UserSponsorships(ops, sponsee.Address(), receiver.Address())
	require.NoError(t, err)
	fundsAfter := sponsorship.Funds.Uint64()

	header, err := client.HeaderByHash(t.Context(), receipt.BlockHash)
	require.NoError(t, err)

	cost := (txData.Gas + subsidies.SponsorshipOverheadGasCost) * header.BaseFee.Uint64()
	require.Equal(t, fundsBefore-cost, fundsAfter)
}
