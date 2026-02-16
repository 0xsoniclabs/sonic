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

package bundle

// func TestBundleTransactions(t *testing.T) {

// 	net := tests.StartIntegrationTestNet(t,
// 		tests.IntegrationTestNetOptions{
// 			Upgrades: tests.AsPointer(opera.GetBrioUpgrades()),
// 		},
// 	)

// 	t.Run("successful bundle", func(t *testing.T) {
// 		testSuccessfulBundle(t, net)
// 	})

// 	t.Run("reverted on failed transaction", func(t *testing.T) {
// 		testRevertedBundle(t, net)
// 	})

// 	t.Run("ignore failed transaction", func(t *testing.T) {
// 		testIgnoreFailedTransaction(t, net)
// 	})

// 	t.Run("ignore invalid transaction", func(t *testing.T) {
// 		testIgnoreInvalidTransactions(t, net)
// 	})

// 	t.Run("at most one", func(t *testing.T) {
// 		testAtMostOneBundle(t, net)
// 	})

// 	t.Run("at most one, ignore failures", func(t *testing.T) {
// 		testAtMostOneIgnoreFailures(t, net)
// 	})
// }

// func testSuccessfulBundle(t *testing.T, net *tests.IntegrationTestNet) {

// 	counter, counterABI, counterAddress := prepareContract(t, net, counter.CounterMetaData.GetAbi, counter.DeployCounter)
// 	countBefore, err := counter.GetCount(nil)
// 	require.NoError(t, err, "failed to get count from contract")

// 	sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

// 	plan := bundle.ExecutionPlan{
// 		Flags: 0,
// 		Transactions: []bundle.MetaTransaction{
// 			{
// 				To:   &counterAddress,
// 				From: sender.Address(),
// 				Data: generateCallData(t, counterABI, "incrementCounter"),
// 			},
// 		},
// 	}

// 	txs := makeTransactionsFromPlan(t, net, plan, sender)
// 	bundleTx, paymentHash := makeBundleTransaction(t, net, txs, plan, net.GetSessionSponsor())
// 	receipt := sendAndWaitBundleTx(t, net, bundleTx, paymentHash)

// 	hashes := getTransactionsInBlock(t, net, receipt.BlockNumber)
// 	require.ElementsMatch(t, hashes,
// 		[]common.Hash{paymentHash, txs[0].Hash()})

// 	// verify that the counter was incremented
// 	count, err := counter.GetCount(nil)
// 	require.NoError(t, err, "failed to get count from contract")
// 	require.Equal(t, countBefore.Uint64()+1, count.Uint64(), "counter was not incremented")
// }

// func testRevertedBundle(t *testing.T, net *tests.IntegrationTestNet) {

// 	counter, counterABI, counterAddress := prepareContract(t, net, counter.CounterMetaData.GetAbi, counter.DeployCounter)
// 	countBefore, err := counter.GetCount(nil)
// 	require.NoError(t, err, "failed to get count from contract")

// 	_, revertABI, revertAddress := prepareContract(t, net, revert.RevertMetaData.GetAbi, revert.DeployRevert)

// 	sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

// 	plan := bundle.ExecutionPlan{
// 		Flags: 0,
// 		Transactions: []bundle.MetaTransaction{
// 			{
// 				To:   &counterAddress,
// 				From: sender.Address(),
// 				Data: generateCallData(t, counterABI, "incrementCounter"),
// 			},
// 			{
// 				To:   &revertAddress,
// 				From: sender.Address(),
// 				Data: generateCallData(t, revertABI, "doCrash"),
// 			},
// 		},
// 	}
// 	txs := makeTransactionsFromPlan(t, net, plan, sender)
// 	bundleTx, paymentHash := makeBundleTransaction(t, net, txs, plan, net.GetSessionSponsor())
// 	receipt := sendAndWaitBundleTx(t, net, bundleTx, paymentHash)

// 	hashes := getTransactionsInBlock(t, net, receipt.BlockNumber)
// 	require.ElementsMatch(t, hashes, []common.Hash{paymentHash})

// 	// verify that the counter has NOT been incremented
// 	count, err := counter.GetCount(nil)
// 	require.NoError(t, err, "failed to get count from contract")
// 	require.Equal(t, countBefore.Uint64(), count.Uint64(), "counter changed")
// }

// func testIgnoreFailedTransaction(t *testing.T, net *tests.IntegrationTestNet) {
// 	counter, counterABI, counterAddress := prepareContract(t, net, counter.CounterMetaData.GetAbi, counter.DeployCounter)
// 	countBefore, err := counter.GetCount(nil)
// 	require.NoError(t, err, "failed to get count from contract")

// 	_, revertABI, revertAddress := prepareContract(t, net, revert.RevertMetaData.GetAbi, revert.DeployRevert)

// 	sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

// 	plan := bundle.ExecutionPlan{
// 		Flags: bundle.FlagIgnoreReverts,
// 		Transactions: []bundle.MetaTransaction{
// 			{
// 				To:   &counterAddress,
// 				From: sender.Address(),
// 				Data: generateCallData(t, counterABI, "incrementCounter"),
// 			},
// 			{
// 				To:   &revertAddress,
// 				From: sender.Address(),
// 				Data: generateCallData(t, revertABI, "doCrash"),
// 			},
// 		},
// 	}
// 	txs := makeTransactionsFromPlan(t, net, plan, sender)
// 	bundleTx, paymentHash := makeBundleTransaction(t, net, txs, plan, net.GetSessionSponsor())
// 	receipt := sendAndWaitBundleTx(t, net, bundleTx, paymentHash)

// 	hashes := getTransactionsInBlock(t, net, receipt.BlockNumber)
// 	require.ElementsMatch(t, hashes, []common.Hash{paymentHash, txs[0].Hash(), txs[1].Hash()})

// 	// verify that the counter has been incremented
// 	count, err := counter.GetCount(nil)
// 	require.NoError(t, err, "failed to get count from contract")
// 	require.Equal(t, countBefore.Uint64()+1, count.Uint64(), "counter changed")
// }

// func testIgnoreInvalidTransactions(t *testing.T, net *tests.IntegrationTestNet) {

// 	// This test causes an invalid transaction because lack of funds to cover the value transfer
// 	// Note: two senders are used to avoid having to manually set the nonces correctly
// 	sender1 := tests.MakeAccountWithBalance(t, net, big.NewInt(10_000_000))
// 	sender2 := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
// 	expectedTransfer := big.NewInt(1000)

// 	plan := bundle.ExecutionPlan{
// 		Flags: bundle.FlagIgnoreInvalid,
// 		Transactions: []bundle.MetaTransaction{
// 			{
// 				To:   &common.Address{0x42},
// 				From: sender1.Address(),
// 				// value transfer is above sender balance, transaction will be invalid
// 				Value: big.NewInt(20_000_000),
// 			},
// 			{
// 				To:   &common.Address{0x42},
// 				From: sender2.Address(),
// 				// this transfer is can be executed
// 				Value: expectedTransfer,
// 			},
// 		},
// 	}
// 	txs := makeTransactionsFromPlan(t, net, plan, sender1, sender2)
// 	bundleTx, paymentHash := makeBundleTransaction(t, net, txs, plan, net.GetSessionSponsor())
// 	receipt := sendAndWaitBundleTx(t, net, bundleTx, paymentHash)

// 	hashes := getTransactionsInBlock(t, net, receipt.BlockNumber)
// 	require.ElementsMatch(t, hashes, []common.Hash{paymentHash, txs[1].Hash()})

// 	client, err := net.GetClient()
// 	require.NoError(t, err, "failed to get client; %v", err)
// 	defer client.Close()

// 	// verify that the first transaction was executed and the second was not
// 	balance0, err := client.BalanceAt(t.Context(), common.Address{0x42}, nil)
// 	require.NoError(t, err, "failed to get balance at address 0x42")
// 	require.Equal(t, expectedTransfer, balance0, "balance at address 0x42 should be the expected transfer amount")
// }

// func testAtMostOneBundle(t *testing.T, net *tests.IntegrationTestNet) {
// 	counter, counterABI, counterAddress := prepareContract(t, net, counter.CounterMetaData.GetAbi, counter.DeployCounter)
// 	countBefore, err := counter.GetCount(nil)
// 	require.NoError(t, err, "failed to get count from contract")

// 	sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

// 	plan := bundle.ExecutionPlan{
// 		Flags: bundle.FlagAtMostOne,
// 		Transactions: []bundle.MetaTransaction{
// 			{
// 				To:   &counterAddress,
// 				From: sender.Address(),
// 				Data: generateCallData(t, counterABI, "incrementCounter"),
// 			},
// 			{
// 				To:   &counterAddress,
// 				From: sender.Address(),
// 				Data: generateCallData(t, counterABI, "incrementCounter"),
// 			},
// 		},
// 	}

// 	txs := makeTransactionsFromPlan(t, net, plan, sender)
// 	bundleTx, paymentHash := makeBundleTransaction(t, net, txs, plan, net.GetSessionSponsor())
// 	receipt := sendAndWaitBundleTx(t, net, bundleTx, paymentHash)

// 	hashes := getTransactionsInBlock(t, net, receipt.BlockNumber)
// 	require.ElementsMatch(t, hashes,
// 		[]common.Hash{paymentHash, txs[0].Hash()})

// 	// verify that the counter was incremented only once
// 	count, err := counter.GetCount(nil)
// 	require.NoError(t, err, "failed to get count from contract")
// 	require.Equal(t, countBefore.Uint64()+1, count.Uint64(), "counter was not incremented")
// }

// func testAtMostOneIgnoreFailures(
// 	t *testing.T,
// 	net *tests.IntegrationTestNet,
// ) {
// 	counter, counterABI, counterAddress := prepareContract(t, net, counter.CounterMetaData.GetAbi, counter.DeployCounter)
// 	countBefore, err := counter.GetCount(nil)
// 	require.NoError(t, err, "failed to get count from contract")

// 	_, revertABI, revertAddress := prepareContract(t, net, revert.RevertMetaData.GetAbi, revert.DeployRevert)

// 	sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

// 	plan := bundle.ExecutionPlan{
// 		Flags: bundle.FlagAtMostOne | bundle.FlagIgnoreReverts,
// 		Transactions: []bundle.MetaTransaction{
// 			{
// 				To:   &revertAddress,
// 				From: sender.Address(),
// 				Data: generateCallData(t, revertABI, "doCrash"),
// 			},
// 			{
// 				To:   &counterAddress,
// 				From: sender.Address(),
// 				Data: generateCallData(t, counterABI, "incrementCounter"),
// 			},
// 		},
// 	}

// 	txs := makeTransactionsFromPlan(t, net, plan, sender)
// 	bundleTx, paymentHash := makeBundleTransaction(t, net, txs, plan, net.GetSessionSponsor())
// 	receipt := sendAndWaitBundleTx(t, net, bundleTx, paymentHash)

// 	hashes := getTransactionsInBlock(t, net, receipt.BlockNumber)
// 	require.ElementsMatch(t, hashes,
// 		[]common.Hash{paymentHash, txs[0].Hash(), txs[1].Hash()})

// 	// verify that the counter was incremented only once
// 	count, err := counter.GetCount(nil)
// 	require.NoError(t, err, "failed to get count from contract")
// 	require.Equal(t, countBefore.Uint64()+1, count.Uint64(), "counter was not incremented")
// }

// func sendAndWaitBundleTx(t *testing.T, net *tests.IntegrationTestNet, bundleTx *types.Transaction, paymentHash common.Hash) *types.Receipt {
// 	t.Helper()

// 	// Note, run will wait for the execution of the bundle tx. This transaction
// 	// will never be included in a block, but its parts. No receipt will be present
// 	// and net.Run will timeout

// 	client, err := net.GetClient()
// 	require.NoError(t, err)
// 	defer client.Close()

// 	err = client.SendTransaction(t.Context(), bundleTx)
// 	require.NoError(t, err, "failed to send bundle tx")

// 	receipt, err := net.GetReceipt(paymentHash)
// 	require.NoError(t, err, "failed to get payment tx receipt")
// 	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status,
// 		"payment tx failed")
// 	return receipt
// }
