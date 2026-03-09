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
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func Test_CreateBundlesWithRPC(t *testing.T) {

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

	sender1 := net.GetSessionSponsor()

	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(t, err, "failed to suggest gas price")

	targetAddress := common.Address{0x42}
	transferAmount := big.NewInt(2)

	// 1) Create a list of transactions to be executed in order atomically.
	const bundledTxCount = 15
	txsToBeBundled := make([]ethereum.CallMsg, bundledTxCount)

	for i := range txsToBeBundled {
		tx := ethereum.CallMsg{
			From:     sender1.Address(),
			To:       &targetAddress,
			Value:    transferAmount,
			GasPrice: gasPrice,
		}
		// TODO: this should be done for the complete bundle
		tx.Gas = estimateGasForBundledTransaction(t, client, tx)
		txsToBeBundled[i] = tx
	}

	// 2) Define bundle execution parameters: when the bundle should be executed
	// and what flags it should have.
	earliest, err := client.BlockNumber(t.Context())
	require.NoError(t, err, "failed to get block number")
	latest := earliest + 10
	flags := uint8(0)

	// 3) Prepare bundle:
	preparedBundle, err := PrepareBundle(
		t, client,
		flags, earliest, latest,
		txsToBeBundled)
	require.NoError(t, err, "failed to prepare bundle")

	// 4) Sign prepared transactions
	signer := types.LatestSignerForChainID(net.GetChainId())
	txs := make([]*types.Transaction, len(preparedBundle.Transactions))
	for i, txArgs := range preparedBundle.Transactions {
		txs[i], err = types.SignTx(txArgs.ToTransaction(), signer, sender1.PrivateKey)
		require.NoError(t, err, "failed to sign transaction")
	}

	checkCompatWithMetaMask(t, client, txs)

	// 5) Submit the bundle to the network
	bundleHash, err := SubmitBundle(client, txs, preparedBundle.Plan)
	require.NoError(t, err, "failed to submit bundle")

	info, err := waitForBundleExecution(t.Context(), client.Client(), bundleHash)
	require.NoError(t, err, "failed to wait for bundle execution")
	require.Equal(t, ethapi.BundleStatusExecuted, info.Status)

	for _, tx := range txs {
		receipt, err := net.GetReceipt(tx.Hash())
		require.NoError(t, err, "failed to get receipt")
		require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful)
	}

	balance, err := client.BalanceAt(t.Context(), targetAddress, nil)
	require.NoError(t, err, "failed to get balance")
	require.Equal(t, transferAmount.Uint64()*bundledTxCount, balance.Uint64(),
		"unexpected balance of target address")
}

// checkCompatWithMetaMask checks that the signed bundle-only transactions can
// be submitted to the network and retrieved as binary blobs from the mempool.
// When using Metamask, transactions are automatically signed and submitted to
// the network, users only receive the transaction hash. For bundles to
// work correctly, the transactions have to be fetched from the mempool.
func checkCompatWithMetaMask(t *testing.T, client *tests.PooledEhtClient, txs []*types.Transaction) {
	t.Helper()

	for _, tx := range txs {
		err := client.SendTransaction(t.Context(), tx)
		require.NoError(t, err, "failed to send transaction to the network")

		// Check that the transaction can be retrieved from the mempool as a binary blob
		var mempoolTx hexutil.Bytes
		err = client.Client().Call(&mempoolTx, "eth_getRawTransactionByHash", tx.Hash())
		require.NoError(t, err, "failed to get transaction from mempool")

		var retrievedTx types.Transaction
		err = retrievedTx.UnmarshalBinary(mempoolTx)
		require.NoError(t, err, "failed to unmarshal transaction from mempool")

		require.Equal(t, tx.Hash(), retrievedTx.Hash(), "transaction hash mismatch")
	}
}

// Prepare bundle is a wrapper around the rpc method sonic_prepareBundle, which
// prepares a bundle for execution by filling in all necessary fields and
// encoding them properly.
//
// It accepts transactions in the form of CallMsg to keep compatibility with
// standard go-ethereum client methods like EstimateGas.
// CallMsg is a more convenient type to prepare transactions,
// it does not encode fields into hex and is compatible with standard
// go-ethereum client methods like EstimateGas.
// Unfortunately, it does not include nonce, therefore this function needs
// to assign a fitting value.
//
// This function should be part of the go-ethereum client object, being the entry
// point to the api from go programs.
func PrepareBundle(
	t *testing.T, client *tests.PooledEhtClient,
	flags uint8, earliest, latest uint64,
	txs []ethereum.CallMsg,
) (ethapi.PreparedBundle, error) {

	nonces := make(map[common.Address]uint64)
	for _, tx := range txs {
		if _, ok := nonces[tx.From]; !ok {
			nonce, err := client.PendingNonceAt(t.Context(), tx.From)
			require.NoError(t, err, "failed to get pending nonce")
			nonces[tx.From] = nonce
		}
	}

	// Convert CallMsg without nonce into TransactionArgs with nonce and hex encoding of fields
	txsArgs := make([]ethapi.TransactionArgs, len(txs))
	for i, tx := range txs {
		nonce := nonces[tx.From]
		nonces[tx.From] = nonce + 1
		txArgs := ethapi.TransactionArgs{
			From:     &tx.From,
			To:       tx.To,
			Nonce:    (*hexutil.Uint64)(&nonce),
			Gas:      (*hexutil.Uint64)(&tx.Gas),
			GasPrice: (*hexutil.Big)(tx.GasPrice),
			Value:    (*hexutil.Big)(tx.Value),
			Data:     (*hexutil.Bytes)(&tx.Data),
		}
		txsArgs[i] = txArgs
	}

	// Call sonic_prepareBundle to get a bundle with all fields properly filled in and encoded
	var preparedBundle ethapi.PreparedBundle
	err := client.Client().Call(&preparedBundle, "sonic_prepareBundle",
		ethapi.PrepareBundleArgs{
			Transactions:   txsArgs,
			ExecutionFlags: hexutil.Uint(flags),
			EarliestBlock:  rpc.BlockNumber(earliest),
			LatestBlock:    rpc.BlockNumber(latest),
		})
	require.NoError(t, err, "failed to call sonic_prepareBundle")
	return preparedBundle, nil
}

// SubmitBundle is a wrapper around the rpc method sonic_submitBundle, which
// submits a prepared bundle for execution.
// It uses types.Transaction just like the method SendTransaction.
// This function should be part of the go-ethereum client object, being the entry
// point to the api from go programs.
func SubmitBundle(client *tests.PooledEhtClient,
	txs []*types.Transaction,
	plan bundle.ExecutionPlan,
) (common.Hash, error) {
	encodedTransactions := make([]hexutil.Bytes, len(txs))
	for i, tx := range txs {
		data, err := tx.MarshalBinary()
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to marshal transaction: %w", err)
		}
		encodedTransactions[i] = hexutil.Bytes(data)
	}

	var bundleHash common.Hash
	err := client.Client().Call(&bundleHash, "sonic_submitBundle",
		ethapi.SubmitBundleArgs{
			SignedTransactions: encodedTransactions,
			ExecutionPlan:      plan,
		})
	return bundleHash, err
}

func estimateGasForBundledTransaction(t *testing.T, client *tests.PooledEhtClient, tx ethereum.CallMsg) uint64 {
	t.Helper()

	tx.AccessList = append(tx.AccessList,
		types.AccessTuple{
			Address:     bundle.BundleOnly,
			StorageKeys: []common.Hash{{}},
		},
	)
	gas, err := client.EstimateGas(t.Context(), tx)
	require.NoError(t, err, "failed to estimate gas")
	return gas
}
