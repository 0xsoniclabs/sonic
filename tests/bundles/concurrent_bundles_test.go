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
	"context"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/revert"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBundles_RunBundlesInParallel(t *testing.T) {
	// Create a list of successful and failing bundles.
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	t.Run("succeeding bundles", func(t *testing.T) {
		testSucceedingConcurrentBundles(t, net)
	})

	t.Run("randomly failing bundles", func(t *testing.T) {
		testRandomlyFailingBundles(t, net)
	})
}

func testSucceedingConcurrentBundles(
	t *testing.T,
	net *tests.IntegrationTestNet,
) {
	const N = 100 // Number of bundles to process
	const W = 3   // Number of transactions per bundle

	require := require.New(t)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	// Create all needed accounts and endow in parallel.
	accounts := tests.NewAccounts(N * W)
	addresses := make([]common.Address, len(accounts))
	for i, cur := range accounts {
		addresses[i] = cur.Address()
	}
	_, err = net.EndowAccounts(addresses, big.NewInt(1e18))
	require.NoError(err)

	envelopes := make([]*types.Transaction, N)
	planHashes := make([]common.Hash, N)
	for i := range N {
		envelope, plan := bundle.NewBuilder().With(
			Step(t, net, accounts[i*W+0], newBurnMoneyTransaction()),
			Step(t, net, accounts[i*W+1], newBurnMoneyTransaction()),
			Step(t, net, accounts[i*W+2], newBurnMoneyTransaction()),
		).BuildEnvelopeAndPlan()

		envelopes[i] = envelope
		planHashes[i] = plan.Hash()
	}

	// Submit all envelops to be processed in parallel.
	_, err = net.SendAll(envelopes)
	require.NoError(err)

	// Wait for all bundles to be completed.
	infos, err := WaitForBundlesExecution(t.Context(), client.Client(), planHashes)
	require.NoError(err)

	// Check that all bundles have been accepted.
	minBlock := uint64(math.MaxUint64)
	maxBlock := uint64(0)
	for _, info := range infos {
		if info.Block != nil {
			minBlock = min(minBlock, uint64(*info.Block))
			maxBlock = max(maxBlock, uint64(*info.Block))
		}
	}

	// Check that all obtained infos match the respective transactions.
	for i, info := range infos {
		bundle, err := bundle.OpenEnvelope(envelopes[i])
		require.NoError(err)
		require.EqualValues(*info.Count, len(bundle.Transactions))

		for i, tx := range bundle.Transactions {
			receipt, err := client.TransactionReceipt(t.Context(), tx.Hash())
			require.NoError(err)
			require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
			require.EqualValues(int(*info.Position)+i, receipt.TransactionIndex)
		}
	}
}

func testRandomlyFailingBundles(
	t *testing.T,
	net *tests.IntegrationTestNet,
) {
	const N = 200 // Number of bundles to process
	const W = 3   // Number of transactions per bundle

	require := require.New(t)

	_, receipt, err := tests.DeployContract(net, revert.DeployRevert)
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
	revertContractAddress := receipt.ContractAddress

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	// Create all needed accounts and endow in parallel.
	accounts := tests.NewAccounts(N * W)
	addresses := make([]common.Address, len(accounts))
	for i, cur := range accounts {
		addresses[i] = cur.Address()
	}
	_, err = net.EndowAccounts(addresses, big.NewInt(1e18))
	require.NoError(err)

	envelopes := make([]*types.Transaction, N)
	planHashes := make([]common.Hash, N)
	for i := range N {
		envelope, plan := bundle.NewBuilder().With(
			Step(t, net, accounts[i*W+0], newBurnMoneyTransaction()),
			Step(t, net, accounts[i*W+1], newBurnMoneyTransaction()),
			Step(t, net, accounts[i*W+2], newRandomlyRevertingTransaction(revertContractAddress)),
		).BuildEnvelopeAndPlan()

		envelopes[i] = envelope
		planHashes[i] = plan.Hash()
	}

	// Send all envelopes in parallel, but ignore rejected bundles.
	err = tests.RunParallelWithClient(net, len(envelopes),
		func(client *tests.PooledEhtClient, i int) error {
			err := client.SendTransaction(t.Context(), envelopes[i])
			if err != nil {
				require.ErrorContains(err, "permanently blocked")
			}
			return nil
		},
	)
	require.NoError(err)

	// Wait for execution
	timeout, timeoutCancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer timeoutCancel()
	infos, err := WaitForBundlesExecution(timeout, client.Client(), planHashes)
	if err != nil {
		// This test may have envelopes which were never admitted into the pool,
		// and other envelopes which were dropped while waiting for execution.
		// WaitForBundlesExecution will timeout waiting for execution in these cases.
		require.ErrorIs(err, context.DeadlineExceeded)
	}

	// For those bundles that got executed, check that the obtained infos match
	// the respective transactions.
	for i, info := range infos {

		bundle, err := bundle.OpenEnvelope(envelopes[i])
		require.NoError(err)

		if info != nil && *info.Count > 0 {
			// bundle produced transactions, so we expect all transactions
			// to be included in a block.
			require.Len(bundle.Transactions, int(*info.Count))

			for i, tx := range bundle.Transactions {
				receipt, err := client.TransactionReceipt(t.Context(), tx.Hash())
				require.NoError(err)
				require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
				require.EqualValues(int(*info.Position)+i, receipt.TransactionIndex)
			}
		} else {
			// bundle got reverted or dropped from the pool, in either case
			// we expect no transaction to be included in a block.
			for _, tx := range bundle.Transactions {
				receipt, err := client.TransactionReceipt(t.Context(), tx.Hash())
				require.ErrorIs(err, ethereum.NotFound, "got receipt: %v, info: %+v", receipt, info)
			}
		}
	}
}

func Step[T types.TxData](
	t *testing.T,
	net tests.IntegrationTestNetSession,
	account *tests.Account,
	txData T,
) bundle.BundleStep {
	return bundle.Step(
		account.PrivateKey,
		tests.SetTransactionDefaults(t, net, txData, account),
	)
}

func newBurnMoneyTransaction() *types.AccessListTx {
	zero := common.Address{}
	return &types.AccessListTx{
		To:    &zero,
		Value: big.NewInt(1),
		Gas:   21000,
	}
}

func newRandomlyRevertingTransaction(
	revertContractAddress common.Address,
) *types.AccessListTx {

	// This calls a function on the revert contract that reverts
	// probabilistically based on the sender and transaction history, so it
	// should not be reliably statically predictable whether this transaction
	// will revert or not.

	parsed, err := revert.RevertMetaData.GetAbi()
	if err != nil {
		panic("could not parse revert contract ABI")
	}

	return &types.AccessListTx{
		To:   &revertContractAddress,
		Data: parsed.Methods["probabilisticRevert"].ID,
		Gas:  100_000,
	}

}
