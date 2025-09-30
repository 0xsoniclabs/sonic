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

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

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
		// Brio is commented out until the gas cap is properly handled for internal transactions.
		//{name: "brio", upgrade: opera.GetBrioUpgrades()},
	}
	for _, test := range upgrades {
		t.Run(test.name, func(t *testing.T) {
			client, err := net.GetClient()
			require.NoError(err)
			defer client.Close()

			// enforce the current upgrade
			testRules := tests.GetNetworkRules(t, net)
			testRules.Upgrades = test.upgrade
			tests.UpdateNetworkRules(t, net, testRules)
			// Advance the epoch by one to apply the change.
			tests.AdvanceEpochAndWaitForBlocks(t, net)

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

func TestGasSubsidies_InternalTransaction_HaveConsistentNonces(t *testing.T) {
	require := require.New(t)

	upgrade := opera.GetAllegroUpgrades()
	upgrade.GasSubsidies = true
	// this test needs its own network instance because it needs to check
	// what that the nonces of internal payment transactions are consistent
	// with the nonces of the seal epoch internal transactions.
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrade,
	})

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	sponsor := tests.NewAccount()
	sponsee := tests.NewAccount()
	receiver := tests.NewAccount()
	donation := big.NewInt(1e16)

	// set up sponsorship
	Fund(t, net, sponsor, sponsee, receiver, donation)

	internalNonce, err := client.PendingNonceAt(t.Context(), common.Address{})
	require.NoError(err)

	tx := makeSponsoredTransactionWithNonce(t, net, receiver.Address(), sponsee, 0)
	receipt, err := net.Run(tx)
	require.NoError(err)
	require.Equal(receipt.Status, types.ReceiptStatusSuccessful)

	txIndex, block := getTransactionIndexInBlock(t, client, receipt)
	require.Less(txIndex, len(block.Transactions())-1,
		"the sponsored transaction should not be the last transaction in the block",
	)
	// check that the payment transaction has the same nonce as the internal transaction
	payment := block.Transactions()[txIndex+1]
	require.True(internaltx.IsInternal(payment), "payment transaction should not be internal")
	require.Equal(internalNonce, payment.Nonce(),
		"the payment transaction should have the same nonce as the internal transaction",
	)

	tx = makeSponsoredTransactionWithNonce(t, net, receiver.Address(), sponsee, 1)
	receipt, err = net.Run(tx)
	require.NoError(err)
	require.Equal(receipt.Status, types.ReceiptStatusSuccessful)

	txIndex, block = getTransactionIndexInBlock(t, client, receipt)

	require.Less(txIndex, len(block.Transactions())-1,
		"the sponsored transaction should not be the last transaction in the block",
	)
	payment = block.Transactions()[txIndex+1]
	require.True(internaltx.IsInternal(payment), "payment transaction should not be internal")
	require.Equal(internalNonce+1, payment.Nonce(),
		"the payment transaction should have nonce incremented by 1",
	)

	net.AdvanceEpoch(t, 1)

	tx = makeSponsoredTransactionWithNonce(t, net, receiver.Address(), sponsee, 2)
	receipt, err = net.Run(tx)
	require.NoError(err)
	require.Equal(receipt.Status, types.ReceiptStatusSuccessful)

	txIndex, block = getTransactionIndexInBlock(t, client, receipt)
	require.Less(txIndex, len(block.Transactions())-1,
		"the sponsored transaction should not be the last transaction in the block",
	)
	payment = block.Transactions()[txIndex+1]
	require.True(internaltx.IsInternal(payment), "payment transaction should not be internal")
	internalNonce += 2 + // for the 2 sponsored transactions
		2 // for the 2 internal transactions in the seal epoch
	require.Equal(internalNonce, payment.Nonce(),
		"the payment transaction should have nonce incremented by 1",
	)

}

func TestGasSubsidies(t *testing.T) {

	upgrades := []struct {
		name    string
		upgrade opera.Upgrades
	}{
		{name: "sonic", upgrade: opera.GetSonicUpgrades()},
		{name: "allegro", upgrade: opera.GetAllegroUpgrades()},
		//{name: "brio", upgrade: opera.GetBrioUpgrades()},
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

				t.Run("Receipts_HaveConsistentTransactionNonces", func(t *testing.T) {
					session := net.SpawnSession(t)
					t.Parallel()
					testGasSubsidies_Receipts_HaveConsistentTransactionIndices(t, session)
				})

				t.Run("SubsidizedTransactionDeductsSubsidyFunds", func(t *testing.T) {
					session := net.SpawnSession(t)
					t.Parallel()
					testGasSubsidies_SubsidizedTransaction_DeductsSubsidyFunds(t, session)
				})
			})
		}
	}
}

func testGasSubsidies_Receipts_HaveConsistentTransactionIndices(t *testing.T,
	session tests.IntegrationTestNetSession) {
	require := require.New(t)

	client, err := session.GetClient()
	require.NoError(err)
	defer client.Close()

	sponsor := tests.NewAccount()
	sponsee := tests.NewAccount()
	receiver := tests.NewAccount()
	receiverAddress := receiver.Address()
	donation := big.NewInt(1e16)

	anotherAccount := tests.MakeAccountWithBalance(t, session, big.NewInt(1e18))

	// set up sponsorship
	Fund(t, session, sponsor, sponsee, receiver, donation)
	suggestedGasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(err)

	numTxs := 5
	hashes := []common.Hash{}
	for i := uint64(0); i < uint64(numTxs); i++ {
		tx := tests.CreateTransaction(t, session, &types.LegacyTx{
			To:       &receiverAddress,
			Gas:      21000,
			GasPrice: suggestedGasPrice,
			Nonce:    i,
		}, anotherAccount)
		hashes = append(hashes, tx.Hash())
		require.NoError(client.SendTransaction(t.Context(), tx), "failed to send transaction %v", i)

		tx = makeSponsoredTransactionWithNonce(t, session, receiverAddress, sponsee, i)
		hashes = append(hashes, tx.Hash())
		require.NoError(client.SendTransaction(t.Context(), tx), "failed to send transaction %v", i)
	}

	// wait for all of them to be processed
	receipts, err := session.GetReceipts(hashes)
	require.NoError(err)

	block, err := client.BlockByNumber(t.Context(), receipts[0].BlockNumber)
	require.NoError(err)

	blockReceipts := []*types.Receipt{}
	err = client.Client().Call(&blockReceipts, "eth_getBlockReceipts", fmt.Sprintf("0x%v", block.Number().String()))
	require.NoError(err)

	for i, tx := range block.Transactions() {

		receipt, err := client.TransactionReceipt(t.Context(), tx.Hash())
		require.NoError(err, "failed to get receipt for tx %d", i)
		require.Equal(uint(i), receipt.TransactionIndex,
			"receipt index does not match transaction index for tx %d", i,
		)

		// if this is a payment transaction, the one before must be the sponsored tx
		if internaltx.IsInternal(tx) {
			require.False(internaltx.IsInternal(block.Transactions()[i-1]),
				"payment transaction at index %d must be preceded by a sponsored transaction", i,
			)
		}

		// verify that transaction index in the block is the one reported by the receipt
		// for internal payment as well as for non-sponsored transactions
		require.EqualValues(i, receipt.TransactionIndex)
		require.Equal(receipt.TxHash, tx.Hash(),
			"receipt tx hash does not match transaction hash for tx %d", i,
		)

		// verify that the receipt obtained from eth_getBlockReceipts matches
		// the one obtained from eth_getTransactionReceipt
		require.Equal(blockReceipts[i].TxHash, receipt.TxHash,
			"receipt tx hash does not match eth_getBlockReceipts for tx %d", i,
		)
		require.Equal(blockReceipts[i].TransactionIndex, receipt.TransactionIndex,
			"receipt index does not match eth_getBlockReceipts for tx %d", i,
		)
	}
}

func testGasSubsidies_SubsidizedTransaction_DeductsSubsidyFunds(t *testing.T, session tests.IntegrationTestNetSession) {
	require := require.New(t)

	client, err := session.GetClient()
	require.NoError(err)
	defer client.Close()

	sponsor := tests.NewAccount()
	sponsee := tests.NewAccount()
	receiver := tests.NewAccount()

	// Before the sponsorship is set up, a transaction from the sponsee
	// to the receiver should fail due to lack of funds.
	chainId := session.GetChainId()
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
	ledger := Fund(t, session, sponsor, sponsee, receiver, donation)

	burnedBefore, err := client.BalanceAt(t.Context(), common.Address{}, nil)
	require.NoError(err)

	// --- submit a sponsored transaction ---

	receipt, err := session.Run(tx)
	require.NoError(err)
	validateSponsoredTxInBlock(t, session, receipt.TxHash)

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
