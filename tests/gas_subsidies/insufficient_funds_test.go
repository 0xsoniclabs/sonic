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

package gas_subsidies

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
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

	tx := &types.LegacyTx{
		To:  &common.Address{0x42},
		Gas: 21000,
	}
	sponsee := tests.NewAccount()
	signedTx := makeSponsorRequestTransaction(t, tx, net.GetChainId(), sponsee)

	// Create a sponsorship fund with 0 initial funds
	sponsorRegistry := Fund(t, net, sponsee.Address(), big.NewInt(0))
	gasConfig, err := sponsorRegistry.GetGasConfig(nil)
	require.NoError(t, err)

	// The cost of the sponsored transaction is the gas used by the tx
	// plus the overhead of the sponsorship itself
	cost := tx.Gas + gasConfig.OverheadCharge.Uint64()

	// Get the current baseFee to calculate the required funds
	header, err := client.HeaderByNumber(t.Context(), big.NewInt(0))
	require.NoError(t, err)
	baseFee := header.BaseFee

	// Only add half the required funds
	sponsorshipAmount := big.NewInt(int64(cost) * baseFee.Int64() / 2)
	ok, fundId, err := sponsorRegistry.AccountSponsorshipFundId(nil, sponsee.Address())
	require.NoError(t, err)
	require.True(t, ok, "registry should have a fund ID")
	receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		opts.Value = sponsorshipAmount
		return sponsorRegistry.Sponsor(opts, fundId)
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Try to send the sponsored transaction
	require.ErrorContains(t, client.SendTransaction(t.Context(), signedTx),
		"transaction sponsorship rejected")

	// Add the second half of the required funds
	receipt, err = net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		opts.Value = sponsorshipAmount
		return sponsorRegistry.Sponsor(opts, fundId)
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	tests.WaitForProofOf(t, client, int(receipt.BlockNumber.Int64()))

	// Get the funds before resending the transaction
	ops := &bind.CallOpts{
		BlockNumber: receipt.BlockNumber,
	}
	sponsorship, err := sponsorRegistry.Sponsorships(ops, fundId)
	require.NoError(t, err)
	fundsBefore := sponsorship.Funds.Uint64()

	// Check that the funds were not decreased during the failed attempt
	require.Equal(t, fundsBefore, uint64(sponsorshipAmount.Uint64()*2))

	// Send the sponsored transaction
	receipt, err = net.Run(signedTx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Get the baseFee of the block which included the tx
	header, err = client.HeaderByHash(t.Context(), receipt.BlockHash)
	require.NoError(t, err)
	baseFee = header.BaseFee

	tests.WaitForProofOf(t, client, int(receipt.BlockNumber.Int64()))

	ops = &bind.CallOpts{
		BlockNumber: receipt.BlockNumber,
	}
	sponsorship, err = sponsorRegistry.Sponsorships(ops, fundId)
	require.NoError(t, err)
	fundsAfter := sponsorship.Funds.Uint64()

	// Check that the funds were reduced by the expected amount
	require.Equal(t, fundsBefore-cost*baseFee.Uint64(), fundsAfter)
}
