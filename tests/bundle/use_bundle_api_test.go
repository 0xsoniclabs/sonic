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

package bundle

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func Test_MakeBundle_WithRPC_API(t *testing.T) {

	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true
	net := tests.StartIntegrationTestNet(t,
		tests.IntegrationTestNetOptions{
			Upgrades: &upgrades,
		},
	)
	client, err := net.GetClient()
	require.NoError(t, err, "failed to get client")
	defer client.Close()

	sender1 := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	nonce, err := client.PendingNonceAt(t.Context(), sender1.Address())
	require.NoError(t, err, "failed to get pending nonce")

	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(t, err, "failed to suggest gas price")
	fmt.Println("gas price:", gasPrice.Uint64())

	counterContract, counterAbi, counterAddress := prepareContract(t, net, counter.CounterMetaData.GetAbi, counter.DeployCounter)
	contractCallData := generateCallData(t, counterAbi, "incrementCounter")

	// 1) Create a list of transactions to be bundled together.
	tx1 := ethapi.TransactionArgs{
		From:     (*common.Address)(asPointer(sender1.Address())),
		To:       &counterAddress,
		GasPrice: (*hexutil.Big)(gasPrice),
		Nonce:    asPointer(hexutil.Uint64(nonce + 1)), // bundled transaction is executed by the same sender as the bundle,
		Data:     asPointer(hexutil.Bytes(contractCallData)),
	}
	tx2 := ethapi.TransactionArgs{
		From:     (*common.Address)(asPointer(sender1.Address())),
		To:       &counterAddress,
		GasPrice: (*hexutil.Big)(gasPrice),
		Nonce:    asPointer(hexutil.Uint64(nonce + 2)), // bundled transaction is executed by the same sender as the bundle,
		Data:     asPointer(hexutil.Bytes(contractCallData)),
	}
	// TODO: decide whenever we want to estimate gas for bundles inside of 'bundle_prepare'
	err = client.Client().Call(&tx1.Gas, "eth_estimateGas", tx1)
	require.NoError(t, err, "failed to call eth_estimateGas")
	err = client.Client().Call(&tx2.Gas, "eth_estimateGas", tx2)
	require.NoError(t, err, "failed to call eth_estimateGas")

	// 2) Prepare bundle:
	var preparedBundle ethapi.BundleArgs
	err = client.Client().Call(&preparedBundle, "bundle_prepare", []any{tx1, tx2}, sender1.Address(), uint16(0))
	require.NoError(t, err, "failed to call bundle_prepare")

	// 3) Sign prepared transactions
	signer := types.LatestSignerForChainID(net.GetChainId())
	txs := make([]*types.Transaction, len(preparedBundle.Transactions))
	for i, txArgs := range preparedBundle.Transactions {
		txs[i], err = types.SignTx(asTransaction(txArgs), signer, sender1.PrivateKey)
		require.NoError(t, err, "failed to sign transaction")
	}

	// 4) Sign payment transaction
	paymentArgs := preparedBundle.Payment
	paymentArgs.Nonce = asPointer(hexutil.Uint64(nonce))
	paymentTx, err := types.SignTx(asTransaction(paymentArgs), signer, sender1.PrivateKey)
	require.NoError(t, err, "failed to sign transaction")

	// return ec.c.CallContext(ctx, nil, "eth_sendRawTransaction", hexutil.Encode(data))
	encodedTransactions := make([]string, len(txs))
	for i, tx := range txs {
		data, err := tx.MarshalBinary()
		require.NoError(t, err, "failed to marshal transaction")
		encodedTransactions[i] = hexutil.Encode(data)
	}
	data, err := paymentTx.MarshalBinary()
	require.NoError(t, err, "failed to marshal payment transaction")
	encodedPayment := hexutil.Encode(data)

	// 5) Finalize, sign, and send transaction bundle to the network
	btx := ethapi.TransactionArgs{}
	err = client.Client().Call(&btx, "bundle_finalize", encodedTransactions, encodedPayment, sender1.Address(), uint8(0))
	require.NoError(t, err, "failed to call bundle_finalize")

	tx, err := types.SignTx(asTransaction(btx), signer, sender1.PrivateKey)
	require.NoError(t, err, "failed to sign transaction")

	err = client.SendTransaction(t.Context(), tx)
	require.NoError(t, err, "failed to send transaction")

	// 6) Wait for the bundle to be executed and check the results
	receipt, err := net.GetReceipt(paymentTx.Hash())
	require.NoError(t, err, "failed to get receipt")
	require.Equal(t, uint64(1), receipt.Status, "transaction failed")

	// check that the counter was incremented twice
	count, err := counterContract.GetCount(nil)
	require.NoError(t, err, "failed to call GetCount")
	require.Equal(t, big.NewInt(2), count, "unexpected counter value")
}

func asTransaction(txArgs ethapi.TransactionArgs) *types.Transaction {
	if txArgs.MaxFeePerGas != nil {
		res := types.DynamicFeeTx{}
		if txArgs.Nonce != nil {
			res.Nonce = uint64(*txArgs.Nonce)
		}
		if txArgs.MaxFeePerGas != nil {
			res.GasFeeCap = txArgs.MaxFeePerGas.ToInt()
		}
		if txArgs.MaxPriorityFeePerGas != nil {
			res.GasTipCap = txArgs.MaxPriorityFeePerGas.ToInt()
		}
		if txArgs.Gas != nil {
			res.Gas = uint64(*txArgs.Gas)
		}
		res.To = txArgs.To
		if txArgs.Value != nil {
			res.Value = txArgs.Value.ToInt()
		}
		if txArgs.Data != nil {
			res.Data = (*txArgs.Data)
		}
		if txArgs.AccessList != nil {
			res.AccessList = *txArgs.AccessList
		}
		return types.NewTx(&res)
	}
	res := types.AccessListTx{}
	if txArgs.Nonce != nil {
		res.Nonce = uint64(*txArgs.Nonce)
	}
	if txArgs.GasPrice != nil {
		res.GasPrice = txArgs.GasPrice.ToInt()
	}
	if txArgs.Gas != nil {
		res.Gas = uint64(*txArgs.Gas)
	}
	res.To = txArgs.To
	if txArgs.Value != nil {
		res.Value = txArgs.Value.ToInt()
	}
	if txArgs.Data != nil {
		res.Data = (*txArgs.Data)
	}
	if txArgs.AccessList != nil {
		res.AccessList = *txArgs.AccessList
	}
	return types.NewTx(&res)
}

func asPointer[T any](v T) *T {
	return &v
}
