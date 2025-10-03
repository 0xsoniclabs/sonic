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
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/0xsoniclabs/sonic/tests/contracts/revert"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestGasSubsidies_SubsidizedTransaction_DeductsSubsidyFunds(t *testing.T) {
	upgrades := []struct {
		name    string
		upgrade opera.Upgrades
	}{
		{name: "sonic", upgrade: opera.GetSonicUpgrades()},
		{name: "allegro", upgrade: opera.GetAllegroUpgrades()},
		// TODO: add brio once it supports internal transactions
	}
	singleProposerOption := map[string]bool{
		"singleProposer": true,
		"distributed":    false,
	}

	for _, test := range upgrades {
		for mode, enabled := range singleProposerOption {
			t.Run(fmt.Sprintf("%s/%v", test.name, mode), func(t *testing.T) {

				test.upgrade.GasSubsidies = true
				test.upgrade.SingleProposerBlockFormation = enabled
				net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
					Upgrades: &test.upgrade,
				})

				testGasSubsidies_SubsidizedTransaction_DeductsSubsidyFunds(t, net)
			})
		}
	}
}

func testGasSubsidies_SubsidizedTransaction_DeductsSubsidyFunds(t *testing.T, net *tests.IntegrationTestNet) {

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	revertContract, receipt, err := tests.DeployContract(net, revert.DeployRevert)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	counterContract, receipt, err := tests.DeployContract(net, counter.DeployCounter)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	cases := map[string]struct {
		runTransactions func(t *testing.T, net *tests.IntegrationTestNet, sender *tests.Account)
	}{
		"sponsored transaction calls contract": {
			runTransactions: func(t *testing.T, net *tests.IntegrationTestNet, sender *tests.Account) {
				opts, err := net.GetTransactOptions(sender)
				require.NoError(t, err)

				opts.GasPrice = big.NewInt(0)
				tx, err := counterContract.IncrementCounter(opts)
				require.NoError(t, err)
				receipt, err := net.GetReceipt(tx.Hash())
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
			},
		},
		"sponsored transaction transfers balance (gas is paid by subsidies, value by sender)": {
			runTransactions: func(t *testing.T, net *tests.IntegrationTestNet, sender *tests.Account) {
				nonce, err := client.PendingNonceAt(t.Context(), sender.Address())
				require.NoError(t, err)
				tx := tests.SignTransaction(t, net.GetChainId(), &types.LegacyTx{
					Nonce: nonce,
					To:    &common.Address{},
					Gas:   21000,
					Value: big.NewInt(1),
				}, sender)
				receipt, err := net.Run(tx)
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
			},
		},
		"sponsored transaction does nothing": {
			runTransactions: func(t *testing.T, net *tests.IntegrationTestNet, sender *tests.Account) {
				nonce, err := client.PendingNonceAt(t.Context(), sender.Address())
				require.NoError(t, err)
				tx := tests.SignTransaction(t, net.GetChainId(), &types.LegacyTx{
					Nonce: nonce,
					To:    &common.Address{},
					Gas:   21000,
				}, sender)
				receipt, err := net.Run(tx)
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
			},
		},
		"sponsored transaction calls contract which reverts": {
			runTransactions: func(t *testing.T, net *tests.IntegrationTestNet, sender *tests.Account) {
				opts, err := net.GetTransactOptions(sender)
				require.NoError(t, err)

				opts.GasPrice = big.NewInt(0)
				tx, err := revertContract.DoRevert(opts)
				require.NoError(t, err)
				receipt, err := net.GetReceipt(tx.Hash())
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusFailed, receipt.Status)
			},
		},
		"multiple sponsored transactions": {
			runTransactions: func(t *testing.T, net *tests.IntegrationTestNet, sender *tests.Account) {
				nonce, err := client.PendingNonceAt(t.Context(), sender.Address())
				require.NoError(t, err)
				tx1 := tests.SignTransaction(t, net.GetChainId(), &types.LegacyTx{
					Nonce: nonce,
					To:    &common.Address{},
					Gas:   21000,
				}, sender)

				require.NoError(t, err)
				tx2 := tests.SignTransaction(t, net.GetChainId(), &types.LegacyTx{
					Nonce: nonce + 1,
					To:    &common.Address{},
					Gas:   21000,
				}, sender)

				err = client.SendTransaction(t.Context(), tx1)
				require.NoError(t, err)
				err = client.SendTransaction(t.Context(), tx2)
				require.NoError(t, err)

				receipts, err := net.GetReceipts([]common.Hash{tx1.Hash(), tx2.Hash()})
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipts[0].Status)
				require.Equal(t, types.ReceiptStatusSuccessful, receipts[1].Status)
			},
		},
		"multiple mixed transactions, sponsored first": {
			runTransactions: func(t *testing.T, net *tests.IntegrationTestNet, sender *tests.Account) {
				nonce, err := client.PendingNonceAt(t.Context(), sender.Address())
				require.NoError(t, err)
				sponsoredTx := tests.SignTransaction(t, net.GetChainId(),
					&types.LegacyTx{
						Nonce: nonce,
						To:    &common.Address{},
						Gas:   21000,
					}, sender)

				require.NoError(t, err)
				normalTx := tests.CreateTransaction(t, net,
					&types.LegacyTx{
						To: &common.Address{},
					}, net.GetSessionSponsor())

				err = client.SendTransaction(t.Context(), sponsoredTx)
				require.NoError(t, err)
				err = client.SendTransaction(t.Context(), normalTx)
				require.NoError(t, err)

				receipts, err := net.GetReceipts([]common.Hash{sponsoredTx.Hash(), normalTx.Hash()})
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipts[0].Status)
				require.Equal(t, types.ReceiptStatusSuccessful, receipts[1].Status)
			},
		},
		"multiple mixed transactions, sponsored last": {
			runTransactions: func(t *testing.T, net *tests.IntegrationTestNet, sender *tests.Account) {
				nonce, err := client.PendingNonceAt(t.Context(), sender.Address())
				require.NoError(t, err)
				sponsoredTx := tests.SignTransaction(t, net.GetChainId(),
					&types.LegacyTx{
						Nonce: nonce,
						To:    &common.Address{},
						Gas:   21000,
					}, sender)

				require.NoError(t, err)
				normalTx := tests.CreateTransaction(t, net,
					&types.LegacyTx{
						To: &common.Address{},
					}, net.GetSessionSponsor())

				err = client.SendTransaction(t.Context(), normalTx)
				require.NoError(t, err)
				err = client.SendTransaction(t.Context(), sponsoredTx)
				require.NoError(t, err)

				receipts, err := net.GetReceipts([]common.Hash{sponsoredTx.Hash(), normalTx.Hash()})
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipts[0].Status)
				require.Equal(t, types.ReceiptStatusSuccessful, receipts[1].Status)
			},
		},
		"sponsored transaction calling a contract which aborts": {
			runTransactions: func(t *testing.T, net *tests.IntegrationTestNet, sender *tests.Account) {

				opts, err := net.GetTransactOptions(sender)
				require.NoError(t, err)

				opts.GasLimit = 300_000
				opts.GasPrice = big.NewInt(0)
				tx, err := revertContract.DoCrash(opts)
				require.NoError(t, err)
				receipt, err := net.GetReceipt(tx.Hash())
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusFailed, receipt.Status)
			},
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {

			donation := big.NewInt(1e18)

			sponsoredSender := tests.MakeAccountWithBalance(t, net, big.NewInt(1))
			sponsorshipRegistry := Fund(t, net, sponsoredSender.Address(), donation)

			subsidiesRegistryBalanceBefore, err := client.BalanceAt(t.Context(), registry.GetAddress(), nil)
			require.NoError(t, err)

			blockBefore, err := client.BlockByNumber(t.Context(), nil)
			require.NoError(t, err)

			// Here the scenario is ran.
			test.runTransactions(t, net, sponsoredSender)

			blockAfter, err := client.BlockByNumber(t.Context(), nil)
			require.NoError(t, err)

			config, err := sponsorshipRegistry.GetGasConfig(nil)
			require.NoError(t, err)

			// Scenarios may produce multiple blocks, so we need to
			// iterate through all of them to find all sponsored transactions.
			var fundsDelta uint64

			// For every block created during test scenario
			for blockNumber := blockBefore.NumberU64() + 1; blockNumber <= blockAfter.NumberU64(); blockNumber++ {

				block, err := client.BlockByNumber(t.Context(), big.NewInt(int64(blockNumber)))
				require.NoError(t, err)

				var totalGasUsed uint64

				for i, tx := range block.Transactions() {
					receipt, err := net.GetReceipt(tx.Hash())
					require.NoError(t, err)

					totalGasUsed += receipt.GasUsed
					require.EqualValues(t, totalGasUsed, receipt.CumulativeGasUsed)

					if subsidies.IsSponsorshipRequest(tx) {
						fundsUsed := (receipt.GasUsed +
							config.OverheadCharge.Uint64()) * block.BaseFee().Uint64()
						require.Greater(t, fundsUsed, uint64(0),
							"sponsored tx must have a non-zero funds cost",
						)
						fundsDelta += fundsUsed

						require.Less(t, i, len(block.Transactions())-1,
							"sponsored tx should not be the last tx in the block")
						internalTx := block.Transactions()[i+1]

						deduceFundsReceipt, err := net.GetReceipt(internalTx.Hash())
						require.NoError(t, err)
						require.Equal(t, types.ReceiptStatusSuccessful, deduceFundsReceipt.Status)

						validateSponsoredTxInBlock(t, net, tx.Hash())

					}

				}
			}

			_, fundId, err := sponsorshipRegistry.AccountSponsorshipFundId(nil, sponsoredSender.Address())
			require.NoError(t, err)
			sponsorship, err := sponsorshipRegistry.Sponsorships(nil, fundId)
			require.NoError(t, err)
			fundsAfter := sponsorship.Funds.Uint64()

			require.Equal(t,
				donation.Uint64()-fundsDelta,
				fundsAfter,
				"the sponsorship fund should be deducted by the expected amount",
			)

			subsidiesRegistryBalanceAfter, err := client.BalanceAt(t.Context(), registry.GetAddress(), nil)
			require.NoError(t, err)

			require.Equal(t,
				subsidiesRegistryBalanceBefore.Uint64()-(fundsDelta),
				subsidiesRegistryBalanceAfter.Uint64(),
				"the subsidies registry balance should be deducted by the expected amount",
			)
		})
	}
}

func TestGasSubsidies_SubsidizedTransaction_SkipTransactionIfDeduceFundsDoesNotFit(t *testing.T) {

	upgrades := opera.GetSonicUpgrades()
	upgrades.GasSubsidies = true
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	revertContract, receipt, err := tests.DeployContract(net, revert.DeployRevert)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	donation := big.NewInt(1e18)

	sponsoredSender := tests.NewAccount()
	registry := Fund(t, net, sponsoredSender.Address(), donation)

	config, err := registry.GetGasConfig(nil)
	require.NoError(t, err)

	rules := tests.GetNetworkRules(t, net)
	rules.Blocks.MaxBlockGas = 3_000_000
	tests.UpdateNetworkRules(t, net, rules)

	net.AdvanceEpoch(t, 1)

	tooLargeToFit := 3_000_000 - config.OverheadCharge.Uint64() + 1

	opts, err := net.GetTransactOptions(sponsoredSender)
	require.NoError(t, err)
	opts.GasPrice = big.NewInt(0)
	opts.GasLimit = tooLargeToFit
	tx, err := revertContract.DoCrash(opts)
	require.NoError(t, err)

	// wait three blocks
	_, _ = net.EndowAccount(common.Address{}, big.NewInt(1))
	_, _ = net.EndowAccount(common.Address{}, big.NewInt(1))
	_, _ = net.EndowAccount(common.Address{}, big.NewInt(1))

	// transaction was skipped
	_, err = client.TransactionReceipt(t.Context(), tx.Hash())
	require.ErrorContains(t, err, "not found")
}

func TestGasSubsidies_NonSponsoredTransactionsAreRejected(t *testing.T) {

	upgrades := opera.GetSonicUpgrades()
	upgrades.GasSubsidies = true
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	cases := map[string]struct {
		tx          types.TxData
		funds       uint64
		expectError error
	}{
		"contract creation cannot be sponsored": {
			tx: &types.LegacyTx{
				// contract creation cannot be sponsored
				Gas:      60000,
				GasPrice: big.NewInt(0),
			},
			expectError: evmcore.ErrUnderpriced,
		},
		"contract creation with gas price cannot be sponsored": {
			tx: &types.LegacyTx{
				// contract creation cannot be sponsored
				Gas:      60000,
				GasPrice: big.NewInt(1),
			},
			expectError: evmcore.ErrUnderpriced,
		},
		"Non-contract creation with gas price is not a valid sponsorship request": {
			tx: &types.LegacyTx{
				To:       &common.Address{0x1},
				Gas:      21000,
				GasPrice: big.NewInt(1),
			},
			expectError: evmcore.ErrUnderpriced,
		},
		"Non-contract creation with zero gas cannot be accepted without funds": {
			tx: &types.LegacyTx{
				To:       &common.Address{0x1},
				Gas:      21000,
				GasPrice: big.NewInt(0),
			},
			expectError: evmcore.ErrSponsorshipRejected,
		},
		"sanity check: valid sponsorship request is accepted": {
			tx: &types.LegacyTx{
				To:       &common.Address{0x1},
				Gas:      21000,
				GasPrice: big.NewInt(0),
			},
			funds:       1e18,
			expectError: nil,
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {

			sponsoredSender := tests.NewAccount()
			Fund(t, net, sponsoredSender.Address(), big.NewInt(int64(test.funds)))

			signer := types.LatestSignerForChainID(net.GetChainId())
			tx := types.MustSignNewTx(sponsoredSender.PrivateKey, signer, test.tx)

			err := client.SendTransaction(t.Context(), tx)
			if test.expectError == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.expectError.Error())
			}
		})
	}
}
