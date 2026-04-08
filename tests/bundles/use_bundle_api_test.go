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
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/increasingly_expensive"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
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
	contractAddress, input := prepareContract(t, session, increasingly_expensive.IncreasinglyExpensiveMetaData.GetAbi, increasingly_expensive.DeployIncreasinglyExpensive, "incrementAndLoop")

	sender1 := session.GetSessionSponsor()

	// 1) Create a list of transactions to be executed in order atomically.
	const bundledTxCount = 15
	txsToBeBundled := make([]ethereum.CallMsg, bundledTxCount)

	for i := range txsToBeBundled {
		tx := ethereum.CallMsg{
			From: sender1.Address(),
			To:   &contractAddress,
			Data: input,
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
	bundleHash, err := SubmitBundle(client, txs, preparedBundle.ExecutionPlan)
	require.NoError(t, err, "failed to submit bundle")

	_, err = waitForBundleExecution(t.Context(), client.Client(), bundleHash)
	require.NoError(t, err, "failed to wait for bundle execution")

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
