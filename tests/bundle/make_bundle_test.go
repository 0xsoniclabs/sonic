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

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func Test_MakeBundle(t *testing.T) {

	net := tests.StartIntegrationTestNet(t,
		tests.IntegrationTestNetOptions{
			Upgrades: tests.AsPointer(opera.GetBrioUpgrades()),
		},
	)
	client, err := net.GetClient()
	require.NoError(t, err, "failed to get client; %v", err)
	defer client.Close()

	sender1 := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
	sender2 := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
	bundler := net.GetSessionSponsor()

	_, counterAbi, counterAddress := prepareContract(t, net, counter.CounterMetaData.GetAbi, counter.DeployCounter)
	input := generateCallData(t, counterAbi, "incrementCounter")

	signer := types.NewCancunSigner(net.GetChainId())

	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(t, err, "failed to suggest gas price; %v", err)

	gasLimit, err := client.EstimateGas(t.Context(), ethereum.CallMsg{
		From:     sender1.Address(),
		To:       &counterAddress,
		Data:     input,
		GasPrice: gasPrice,
		AccessList: types.AccessList{
			// add one entry to the estimation, to allocate gas for the bundle-only marker
			{Address: bundle.BundleOnly, StorageKeys: []common.Hash{{}}},
		},
	})
	require.NoError(t, err, "failed to estimate gas")
	fmt.Printf("gasLimit: %d (%x)\n", gasLimit, gasLimit)

	// 1)  make transactions
	tx1 := types.NewTx(tests.SetTransactionDefaults(t, net, &types.AccessListTx{
		To:       &counterAddress,
		Gas:      gasLimit,
		Data:     input,
		GasPrice: gasPrice,
	}, sender1))
	tx2 := types.NewTx(tests.SetTransactionDefaults(t, net, &types.AccessListTx{
		To:       &counterAddress,
		Gas:      gasLimit,
		Data:     input,
		GasPrice: gasPrice,
	}, sender2))

	plan := bundle.ExecutionPlan{
		Flags: 0,
		Steps: []bundle.ExecutionStep{
			{From: sender1.Address(), Hash: signer.Hash(tx1)},
			{From: sender2.Address(), Hash: signer.Hash(tx2)},
		},
	}

	// 2) redo transactions, now with bundle-only access list item, and sign them with the corresponding sender account
	tx1 = tests.SignTransaction(t, net.GetChainId(), &types.AccessListTx{
		Nonce:    tx1.Nonce(),
		GasPrice: tx1.GasPrice(),
		Gas:      tx1.Gas(),
		To:       tx1.To(),
		Value:    tx1.Value(),
		Data:     tx1.Data(),
		AccessList: append(tx1.AccessList(),
			types.AccessTuple{Address: bundle.BundleOnly, StorageKeys: []common.Hash{plan.Hash()}},
		),
	}, sender1)
	tx2 = tests.SignTransaction(t, net.GetChainId(), &types.AccessListTx{
		Nonce:    tx2.Nonce(),
		GasPrice: tx2.GasPrice(),
		Gas:      tx2.Gas(),
		To:       tx2.To(),
		Value:    tx2.Value(),
		Data:     tx2.Data(),
		AccessList: append(tx2.AccessList(),
			types.AccessTuple{Address: bundle.BundleOnly, StorageKeys: []common.Hash{plan.Hash()}},
		),
	}, sender2)

	transactions := types.Transactions{tx1, tx2}

	bundleTx, paymentTxHash := makeBundleTransaction(t, net, transactions, plan, bundler)
	require.NotNil(t, bundleTx)
	require.NotZero(t, paymentTxHash)

	err = client.SendTransaction(t.Context(), bundleTx)
	require.NoError(t, err)

	receipt, err := net.GetReceipt(paymentTxHash)
	require.NoError(t, err, "failed to get payment tx receipt; %v", err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "payment transaction failed")

	// Check all transactions have been executed and the order is correct
	receipt, err = net.GetReceipt(transactions[0].Hash())
	require.NoError(t, err, "failed to get transaction tx 0 receipt; %v", err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "tx1 failed")

	receipt, err = net.GetReceipt(transactions[1].Hash())
	require.NoError(t, err, "failed to get transaction tx 1 receipt; %v", err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "tx2 failed")

	transactionHashes := getTransactionsInBlock(t, net, receipt.BlockNumber)
	require.ElementsMatch(t, transactionHashes, []common.Hash{
		paymentTxHash, transactions[0].Hash(), transactions[1].Hash(),
	})
}

// makeBundleTransaction creates a bundle transaction with the given transactions and execution plan
// This function will create the corresponding payment transaction. Both payment and the bundle transaction
// are signed by the bundler account.
// It returns the bundle transaction and the hash of the payment transaction, the later is used
// for waiting on the completion of the bundle execution, as the bundle transaction will not be included
// in a block.
func makeBundleTransaction(t *testing.T,
	net *tests.IntegrationTestNet,
	transactions types.Transactions,
	plan bundle.ExecutionPlan,
	bundler *tests.Account) (*types.Transaction, common.Hash) {
	t.Helper()

	client, err := net.GetClient()
	require.NoError(t, err, "failed to get client; %v", err)
	defer client.Close()

	sameNonceForBundleAndPayment, err := client.PendingNonceAt(t.Context(), bundler.Address())
	require.NoError(t, err, "failed to get nonce for bundler; %v", err)

	cost := big.NewInt(0)
	for _, tx := range transactions {
		txCost := new(big.Int).Mul(new(big.Int).SetUint64(tx.Gas()), tx.GasPrice())
		cost = new(big.Int).Add(cost, txCost)
	}

	// make payment transaction
	paymentTx := tests.CreateTransaction(t, net,
		&types.AccessListTx{Nonce: sameNonceForBundleAndPayment,
			To:    &common.Address{0x01},
			Value: cost,
			AccessList: types.AccessList{
				{Address: bundle.BundleOnly, StorageKeys: []common.Hash{plan.Hash()}},
			}}, bundler)

	var gas uint64
	for _, tx := range append(transactions, paymentTx) {
		gas += tx.Gas()
	}

	bundlePayload := bundle.TransactionBundle{
		Version: bundle.BundleV1,
		Bundle:  transactions,
		Payment: paymentTx,
		Flags:   plan.Flags,
	}

	// create the bundle transaction with the same nonce as the payment transaction
	bundleTx := tests.CreateTransaction(t, net,
		&types.LegacyTx{Nonce: sameNonceForBundleAndPayment,
			To:   &bundle.BundleAddress,
			Gas:  gas,
			Data: bundle.Encode(bundlePayload),
		}, bundler)

	// Sanity check the bundle before sending it to the mempool, if fails to validate before making
	// a bundle transaction, it will fail to be included in a block and waiting for payment receipt will timeout
	upgrades := net.GetUpgrades()
	signer := types.NewCancunSigner(net.GetChainId())
	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(t, err, "failed to suggest gas price; %v", err)
	require.NoError(t, bundle.ValidateTransactionBundle(bundleTx, bundlePayload, signer, gasPrice, upgrades))

	return bundleTx, paymentTx.Hash()
}

func prepareContract[T any](
	t testing.TB, net *tests.IntegrationTestNet,
	getABI func() (*abi.ABI, error),
	deployFunc tests.ContractDeployer[T],
) (*T, *abi.ABI, common.Address) {
	t.Helper()
	abi, err := getABI()
	require.NoError(t, err, "failed to get counter abi; %v", err)

	contract, receipt, err := tests.DeployContract(net, deployFunc)
	require.NoError(t, err, "failed to deploy contract; %v", err)
	require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful)
	return contract, abi, receipt.ContractAddress
}

func generateCallData(t testing.TB, abi *abi.ABI, methodName string, params ...any) []byte {
	t.Helper()
	input, err := abi.Pack(methodName, params...)
	require.NoError(t, err, "failed to pack input for method %s; %v", methodName, err)
	return input
}

func getTransactionsInBlock(t *testing.T, net *tests.IntegrationTestNet, blockNumber *big.Int) []common.Hash {
	t.Helper()

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()
	block, err := client.BlockByNumber(t.Context(), blockNumber)
	require.NoError(t, err, "failed to get block by number")

	hashes := make([]common.Hash, 0, len(block.Transactions()))
	for _, btx := range block.Transactions() {
		hashes = append(hashes, btx.Hash())
	}
	return hashes
}
