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
	"testing"

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/increasingly_expensive"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func Test_CreateBundlesWithRPC(t *testing.T) {
	t.Parallel()
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true
	session := sharedNetwork.GetIntegrationTestNetSession(t, upgrades)

	client, err := session.GetClient()
	require.NoError(t, err, "failed to get client")
	defer client.Close()

	// Deploy the increasingly expensive contract.
	_, receipt, err := tests.DeployContract(session, increasingly_expensive.DeployIncreasinglyExpensive)
	require.NoError(t, err, "failed to deploy contract; %v", err)
	require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful)

	abi, err := increasingly_expensive.IncreasinglyExpensiveMetaData.GetAbi()
	require.NoError(t, err, "failed to get abi")
	input := generateCallData(t, abi, "incrementAndLoop")

	sender1 := session.GetSessionSponsor()

	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(t, err, "failed to suggest gas price")

	// 1) Create a list of transactions to be executed in order atomically.
	const bundledTxCount = 15
	txsToBeBundled := make([]ethereum.CallMsg, bundledTxCount)

	for i := range txsToBeBundled {
		tx := ethereum.CallMsg{
			From:     sender1.Address(),
			To:       &receipt.ContractAddress,
			GasPrice: gasPrice,
			Data:     input,
		}
		txsToBeBundled[i] = tx
	}

	// 2) Prepare bundle:
	preparedBundle, err := PrepareBundle(t, client, txsToBeBundled, nil, nil)
	require.NoError(t, err, "failed to prepare bundle")

	// 3) Sign prepared transactions
	signer := types.LatestSignerForChainID(session.GetChainId())
	txs := make([]*types.Transaction, len(preparedBundle.Transactions))
	for i, txArgs := range preparedBundle.Transactions {
		txs[i], err = types.SignTx(txArgs.ToTransaction(), signer, sender1.PrivateKey)
		require.NoError(t, err, "failed to sign transaction")
	}

	checkCompatWithMetaMask(t, client, txs)

	// 4) Submit the bundle to the network
	bundleHash, err := SubmitBundle(client, txs, preparedBundle.Plan)
	require.NoError(t, err, "failed to submit bundle")

	info, err := waitForBundleExecution(t.Context(), client.Client(), bundleHash)
	require.NoError(t, err, "failed to wait for bundle execution")
	require.Equal(t, ethapi.BundleStatusExecuted, info.Status)

	for _, tx := range txs {
		receipt, err := session.GetReceipt(tx.Hash())
		require.NoError(t, err, "failed to get receipt")
		require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful)
	}
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
// This function also estimates gas for each transaction and fills in the Gas field,
// as it is required by sonic_prepareBundle.
// if earliest and latest block numbers are not provided, it will set earliest to the next block after submission
// and latest to 1024 blocks after earliest.
//
// This function should be part of the go-ethereum client object, being the entry
// point to the api from go programs.
func PrepareBundle(
	t *testing.T, client *tests.PooledEhtClient,
	txs []ethereum.CallMsg,
	earliest, latest *int64,
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
			GasPrice: (*hexutil.Big)(tx.GasPrice),
			Value:    (*hexutil.Big)(tx.Value),
			Data:     (*hexutil.Bytes)(&tx.Data),
		}
		txsArgs[i] = txArgs
	}

	var gasLimits ethapi.BundleGasLimits
	err := client.Client().Call(&gasLimits, "sonic_estimateGasForTransactions", txsArgs, "latest", nil, nil)
	require.NoError(t, err, "failed to estimate gas for bundle")

	for i := range txsArgs {
		txsArgs[i].Gas = (*hexutil.Uint64)(&gasLimits.GasLimits[i])
	}

	var earliestBlock, latestBlock *rpc.BlockNumber
	if earliest != nil {
		earliestBlock = (*rpc.BlockNumber)(earliest)
	}
	if latest != nil {
		latestBlock = (*rpc.BlockNumber)(latest)
	}

	// Call sonic_prepareBundle to get a bundle with all fields properly filled in and encoded
	var preparedBundle ethapi.PreparedBundle
	err = client.Client().Call(&preparedBundle, "sonic_prepareBundle",
		ethapi.PrepareBundleArgs{
			Transactions:  txsArgs,
			EarliestBlock: earliestBlock,
			LatestBlock:   latestBlock,
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
