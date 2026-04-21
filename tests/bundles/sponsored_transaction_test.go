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
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/0xsoniclabs/sonic/tests/gas_subsidies"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBundle_SponsoredTransactionsAreAcceptedAndExecuted(t *testing.T) {
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true
	upgrades.GasSubsidies = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	signer := types.LatestSignerForChainID(net.GetChainId())
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// Deploy the counter contract.
	counterContract, receipt, err := tests.DeployContract(net, counter.DeployCounter)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	counterAddr := receipt.ContractAddress

	// Create a sponsored sender and fund its sponsorship.
	sponsoredSender := tests.NewAccount()
	gas_subsidies.Fund(t, net, sponsoredSender.Address(), big.NewInt(1e18))

	block, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	// Build the sponsored transaction data with gas price = 0.
	sponsoredTxData0 := tests.SetTransactionDefaults(t, net, &types.AccessListTx{
		To:   &counterAddr,
		Data: tests.MustGetMethodParameters(t, counter.CounterMetaData, "incrementCounter"),
		Gas:  50_000,
	}, sponsoredSender)
	sponsoredTxData0.GasPrice = big.NewInt(0)

	sponsoredTxData1 := *sponsoredTxData0
	sponsoredTxData1.Nonce += 1

	// Create a bundle with a sponsored txs.
	envelope, bundleTx, plan := bundle.NewBuilder().
		WithSigner(signer).
		SetEarliest(block).
		AllOf(
			bundle.Step(
				sponsoredSender.PrivateKey,
				sponsoredTxData0,
			),
			bundle.Step(
				sponsoredSender.PrivateKey,
				sponsoredTxData1,
			),
		).
		BuildEnvelopeBundleAndPlan()

	// Run the bundle.
	require.NoError(t, client.SendTransaction(t.Context(), envelope))

	// Wait for the bundle to be processed.
	info, err := WaitForBundleExecution(t.Context(), client.Client(), plan.Hash())
	require.NoError(t, err)
	require.EqualValues(t, 4, info.Count)

	// The payment transactions are not returned by the bundle API.
	txs := bundleTx.GetTransactionsInReferencedOrder()
	receipts, err := net.GetReceipts([]common.Hash{txs[0].Hash(), txs[1].Hash()})
	require.NoError(t, err)
	require.Len(t, receipts, 2)
	for _, receipt := range receipts {
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
		require.EqualValues(t, info.Block, receipt.BlockNumber.Uint64())
	}

	// The sponsored transactions should be at position 0 and 2 in the block,
	// with the payment transaction in between.
	require.EqualValues(t, info.Position, receipts[0].TransactionIndex)
	require.EqualValues(t, info.Position+2, receipts[1].TransactionIndex)

	// Ensure the sponsored transaction was successful and the counter has been incremented.
	got, err := counterContract.GetCount(&bind.CallOpts{BlockNumber: big.NewInt(info.Block.Int64())})
	require.NoError(t, err, "failed to get counter value")
	require.Equal(t, int64(2), got.Int64(), "unexpected counter value")
}

func TestBundle_RevertedBundleDoesNotConsumeSponsoredFunds(t *testing.T) {
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true
	upgrades.GasSubsidies = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	signer := types.LatestSignerForChainID(net.GetChainId())
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// Deploy the counter contract.
	counterContract, receipt, err := tests.DeployContract(net, counter.DeployCounter)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	counterAddr := receipt.ContractAddress

	// Create a sponsored sender and fund its sponsorship.
	sponsoredSender := tests.NewAccount()
	fundsForOneExecution := big.NewInt(5e15)
	gas_subsidies.Fund(t, net, sponsoredSender.Address(), fundsForOneExecution)

	// Create a regular sender with funds.
	sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	blockNumber, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	// Build the sponsored transaction data with gas price = 0.
	sponsoredTxData := tests.SetTransactionDefaults(t, net, &types.AccessListTx{
		To:   &counterAddr,
		Data: tests.MustGetMethodParameters(t, counter.CounterMetaData, "incrementCounter"),
		Gas:  50_000,
	}, sponsoredSender)
	sponsoredTxData.GasPrice = big.NewInt(0)

	// Create a bundle with a sponsored tx followed by a reverting tx.
	envelope, _, plan := bundle.NewBuilder().
		WithSigner(signer).
		SetEarliest(blockNumber).
		AllOf(
			bundle.Step(
				sponsoredSender.PrivateKey,
				sponsoredTxData,
			),
			// This transaction will revert due to insufficient gas
			Step(t, net, sender, &types.AccessListTx{Gas: 1}),
		).
		BuildEnvelopeBundleAndPlan()

	// Run the bundle.
	require.NoError(t, client.SendTransaction(t.Context(), envelope))

	// Wait for the bundle to be processed.
	info, err := WaitForBundleExecution(t.Context(), client.Client(), plan.Hash())
	require.NoError(t, err)

	// Ensure the bundle was reverted and the counter has not been incremented.
	number := big.NewInt(info.Block.Int64())
	got, err := counterContract.GetCount(&bind.CallOpts{BlockNumber: number})
	require.NoError(t, err, "failed to get counter value")
	require.Equal(t, int64(0), got.Int64(), "unexpected counter value")

	// Ensure there are still sufficient funds
	transaction := types.MustSignNewTx(sponsoredSender.PrivateKey, signer, sponsoredTxData)
	require.NoError(t, client.SendTransaction(t.Context(), transaction))

	// Wait for the transaction to be processed.
	receipt, err = net.GetReceipt(transaction.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Ensure the sponsored transaction was successful and the counter has been incremented.
	got, err = counterContract.GetCount(&bind.CallOpts{BlockNumber: receipt.BlockNumber})
	require.NoError(t, err, "failed to get counter value")
	require.Equal(t, int64(1), got.Int64(), "unexpected counter value")

	// Ensure there are no more funds
	sponsoredTxData.Nonce += 1
	transaction = types.MustSignNewTx(sponsoredSender.PrivateKey, signer, sponsoredTxData)
	err = client.SendTransaction(t.Context(), transaction)
	require.ErrorContains(t, err, "transaction sponsorship rejected")
}
