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
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/revert"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBundles_RunBundlesInParallel(t *testing.T) {
	// Create a list of successful and failing bundles.
	net := GetIntegrationTestNetWithBundlesEnabled(t)

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
	accounts := tests.MakeAccountsWithBalance(t, net, N*W, big.NewInt(1e18))

	envelopes := make([]*types.Transaction, N)
	planHashes := make([]common.Hash, N)
	signer := types.LatestSignerForChainID(net.GetChainId())
	for i := range N {
		envelope, plan := bundle.NewBuilder().WithSigner(signer).AllOf(
			Step(t, net, accounts[i*W+0], newBurnMoneyTransaction()),
			Step(t, net, accounts[i*W+1], newBurnMoneyTransaction()),
			Step(t, net, accounts[i*W+2], newBurnMoneyTransaction()),
		).BuildEnvelopeAndPlan()

		envelopes[i] = envelope
		planHashes[i] = plan.Hash()
	}

	// Submit all envelopes to be processed in parallel.
	_, err = net.SendAll(envelopes)
	require.NoError(err)

	// Wait for all bundles to be completed.
	infos, err := WaitForBundleExecutions(t.Context(), client.Client(), planHashes)
	require.NoError(err)

	// Check that all obtained infos match the respective transactions.
	for i, info := range infos {
		bundle, err := bundle.OpenEnvelope(signer, envelopes[i])
		require.NoError(err)
		require.EqualValues(info.Count, len(bundle.Transactions))

		for i, tx := range bundle.GetTransactionsInReferencedOrder() {
			receipt, err := client.TransactionReceipt(t.Context(), tx.Hash())
			require.NoError(err)
			require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
			require.Equal(int(receipt.BlockNumber.Uint64()), int(info.Block))
			require.Equal(int(receipt.TransactionIndex), int(info.Position)+i)
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

	revertContractAddress := tests.MustDeployContract(t, net, revert.DeployRevert)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	// Create all needed accounts and endow in parallel.
	accounts := tests.MakeAccountsWithBalance(t, net, N*W, big.NewInt(1e18))

	envelopes := make([]*types.Transaction, N)
	planHashes := make([]common.Hash, N)
	signer := types.LatestSignerForChainID(net.GetChainId())
	for i := range N {
		envelope, plan := bundle.NewBuilder().WithSigner(signer).AllOf(
			Step(t, net, accounts[i*W+0], newBurnMoneyTransaction()),
			Step(t, net, accounts[i*W+1], newBurnMoneyTransaction()),
			Step(t, net, accounts[i*W+2], newRandomlyRevertingTransaction(revertContractAddress)),
		).BuildEnvelopeAndPlan()

		envelopes[i] = envelope
		planHashes[i] = plan.Hash()
	}

	// Send all envelopes in parallel, but ignore rejected bundles.
	accepted, rejected, err := net.TrySendAll(envelopes)
	require.NoError(err)
	require.NotEmpty(accepted)
	for _, issue := range rejected {
		require.ErrorContains(issue, "trial-run failed")
	}

	// Wait for execution
	timeout, timeoutCancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer timeoutCancel()
	infos, err := WaitForBundleExecutions(timeout, client.Client(), planHashes)
	if err != nil {
		// This test may have envelopes which were never admitted into the pool,
		// and other envelopes which were dropped while waiting for execution.
		// WaitForBundleExecutions will timeout waiting for execution in these cases.
		require.ErrorIs(err, context.DeadlineExceeded)
	}

	// For those bundles that got executed, check that the obtained infos match
	// the respective transactions.
	for i, info := range infos {

		bundle, err := bundle.OpenEnvelope(signer, envelopes[i])
		require.NoError(err)

		if info != nil && info.Count > 0 {
			// bundle produced transactions, so we expect all transactions
			// to be included in a block.
			require.Len(bundle.Transactions, int(info.Count))

			for i, tx := range bundle.GetTransactionsInReferencedOrder() {
				receipt, err := client.TransactionReceipt(t.Context(), tx.Hash())
				require.NoError(err)
				require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
				require.Equal(int(receipt.BlockNumber.Uint64()), int(info.Block))
				require.Equal(int(receipt.TransactionIndex), int(info.Position)+i)
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

// newBurnMoneyTransaction creates transaction data for a transaction burning
// one unit of value by sending it to address zero. As long as the signer has
// enough balance for the value transfer and the gas costs, it will always
// succeed.
func newBurnMoneyTransaction() *types.AccessListTx {
	zero := common.Address{}
	return &types.AccessListTx{
		To:    &zero,
		Value: big.NewInt(1),
		Gas:   21000,
	}
}

// newRandomlyRevertingTransaction creates the transaction data for a call to
// a contract randomly reverting. The reverting of the transaction depends on
// the sequencing of other calls to this contract in the same block. It can
// thus not be predicted by the emitter without knowing the contributions of
// other validators. Reverts are only decided during block-execution after a
// consensus on the transaction order has been reached.
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
