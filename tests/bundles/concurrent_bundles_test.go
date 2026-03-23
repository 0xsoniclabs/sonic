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
	"maps"
	"math"
	"math/big"
	"slices"
	"testing"

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
		NumNodes: 1,
	})

	t.Run("succeeding bundles", func(t *testing.T) {
		testSucceedingConcurrentBundles(t, net)
	})

	t.Run("concurrent failing bundles", func(t *testing.T) {
		testFailingConcurrentBundles(t, net)
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

	envelopes := make(map[common.Hash]*types.Transaction)
	for i := range N {
		envelope, plan := bundle.NewBuilder().With(
			Step(t, net, accounts[i*W+0], newBurnMoneyTransaction()),
			Step(t, net, accounts[i*W+1], newBurnMoneyTransaction()),
			Step(t, net, accounts[i*W+2], newBurnMoneyTransaction()),
		).BuildEnvelopeAndPlan()

		envelopes[plan.Hash()] = envelope
	}

	// Submit all envelops to be processed in parallel.
	_, err = net.SendAll(slices.Collect(maps.Values(envelopes)))
	require.NoError(err)

	// Wait for all bundles to be completed.
	infos, err := waitForBundlesExecution(t.Context(), client.Client(),
		slices.Collect(maps.Keys(envelopes)))
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
	for planHash, info := range infos {
		bundle, err := bundle.OpenEnvelope(envelopes[planHash])
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

func testFailingConcurrentBundles(
	t *testing.T,
	net *tests.IntegrationTestNet,
) {
	const N = 100 // Number of bundles to process
	const W = 3   // Number of transactions per bundle

	require := require.New(t)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	revertContract, lastReceipt, err := tests.DeployContract(net, revert.DeployRevert)
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, lastReceipt.Status)
	revertContractAddress := lastReceipt.ContractAddress

	// Create all needed accounts and endow in parallel.
	accounts := tests.NewAccounts(N * W)
	addresses := make([]common.Address, len(accounts))
	for i, cur := range accounts {
		addresses[i] = cur.Address()
	}
	_, err = net.EndowAccounts(addresses, big.NewInt(1e18))
	require.NoError(err)

	envelopes := make(map[common.Hash]*types.Transaction)
	for i := range N {
		envelope, plan := bundle.NewBuilder().With(
			Step(t, net, accounts[i*W+0], newBurnMoneyTransaction()),
			Step(t, net, accounts[i*W+1], newBurnMoneyTransaction()),
			Step(t, net, accounts[i*W+2], newStateDependentRevertingTransaction(revertContractAddress)),
		).BuildEnvelopeAndPlan()

		envelopes[plan.Hash()] = envelope
	}

	// Submit all envelops to be processed in parallel.
	// there is no reason for the pool not to accept them at this point
	_, err = net.SendAll(slices.Collect(maps.Values(envelopes)))
	require.NoError(err, "expect no error, as the contract state does not revert execution yet")

	// Submit a transaction which forces bundles executed afterward to revert
	lastReceipt, err = net.Apply(revertContract.ToggleRevert)
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, lastReceipt.Status)

	bundlesExecuted := 0
	for planHash := range envelopes {
		info, err := getBundleInfo(t.Context(), client.Client(), planHash)
		if err != nil {
			// bundles dropped in the pool before they were emitted will not produce info.
			// this is ok, any other error should not happen in this test
			require.ErrorIs(err, ethereum.NotFound)
			continue
		}
		require.NotNil(info, "if error was not ethereum.NotFound, info should not be nil")

		if int64(*info.Block) < lastReceipt.BlockNumber.Int64() {
			// bundles executed in blocks before the revert flag was enabled
			require.EqualValues(3, uint(*info.Count),
				"bundles executed before the revert flag was enabled shall be complete")
		} else if int64(*info.Block) == lastReceipt.BlockNumber.Int64() {
			// bundles executed in the same block as the revert flag transaction
			if uint(*info.Position) >= lastReceipt.TransactionIndex {
				require.Zero(*info.Count,
					"bundles executed after the revert flag was enabled shall be empty")
			} else {
				require.EqualValues(W, *info.Count,
					"bundles executed before the revert flag was enabled shall be complete")
			}
		} else {
			// bundles executed in blocks after the revert flag was enabled
			require.Zero(*info.Count,
				"bundles executed after the revert flag was enabled shall be empty")
		}

		bundlesExecuted++
	}
	require.NotZero(bundlesExecuted)
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
		Gas:   25300,
	}
}

func newStateDependentRevertingTransaction(
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
		Data: parsed.Methods["conditionalRevert"].ID,
		Gas:  100_000,
	}
}
