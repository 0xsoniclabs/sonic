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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// Before the Canto upgrade, a failed bundle with a bare execution plan root is
// not rolled back. The gas used by the failed transaction stays in the block's
// used gas and in the cumulative gas used of all following receipts, while the
// transaction and its receipt are not part of the block. Those values are part
// of the block hash and must be preserved, but the gas used reported for the
// receipt following the lost gas must still be the one of its own execution.
func TestBundle_GasSurplusDoesNotAffectGasUsedOfFollowingReceipts(t *testing.T) {
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true // < Canto is not enabled

	// The transaction pool would reject a bundle failing at the head state, so
	// the envelope is proposed directly, like a validator not pre-checking it.
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	signer := types.LatestSignerForChainID(net.GetChainId())
	revertAddress, revertInput := tests.MustDeployRevertContractAndGetMethodCallParameters(t, net)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	const transferGas = 21_000

	// The transaction order within a block is scrambled by the transaction
	// hashes, except for transactions of the same sender, which are ordered by
	// nonce and, for equal nonces, by descending gas price. The envelope is thus
	// sent from the transfer's sender with the same nonce and a higher gas price,
	// which makes the bundle precede the transfer in the block. An envelope does
	// not consume the nonce of its sender.
	accounts := tests.MakeAccountsWithBalance(t, net, 2, big.NewInt(1e18))
	bundleSender, transferSender := accounts[0], accounts[1]

	blockNumber, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	nonce, err := client.PendingNonceAt(t.Context(), transferSender.Address())
	require.NoError(t, err)

	envelope, txBundle, plan := bundle.NewBuilder().
		WithSigner(signer).
		SetEnvelopeSenderKey(transferSender.PrivateKey).
		SetEnvelopeNonce(nonce).
		SetEnvelopeGasPrice(big.NewInt(1e12)).
		SetEarliest(blockNumber).
		With(Step(t, net, bundleSender, &types.AccessListTx{
			To:   &revertAddress,
			Gas:  1_000_000,
			Data: revertInput,
		})).
		BuildEnvelopeBundleAndPlan()

	transfer := tests.CreateTransaction(t, net, &types.LegacyTx{
		Nonce: nonce,
		To:    &common.Address{0x42},
		Gas:   transferGas,
	}, transferSender)

	// Both are proposed in a single event, and thus end up in the same block.
	_, err = net.ForceEmitAll(t.Context(), []*types.Transaction{envelope, transfer})
	require.NoError(t, err)

	info, err := WaitForBundleExecution(t.Context(), client.Client(), plan.Hash())
	require.NoError(t, err)
	require.Zero(t, int(info.Count))

	receipt, err := net.GetReceipt(transfer.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	require.Equal(t, int64(info.Block), receipt.BlockNumber.Int64(),
		"the bundle and the transfer must be in the same block")

	// The failed transaction of the bundle is not part of the block.
	_, err = client.TransactionReceipt(t.Context(), txBundle.GetTransactionsInReferencedOrder()[0].Hash())
	require.Error(t, err)

	// The gas of the failed transaction is part of the cumulative gas used.
	previous := uint64(0)
	if receipt.TransactionIndex > 0 {
		receipts := getBlockReceipts(t, client, receipt.BlockNumber)
		previous = receipts[receipt.TransactionIndex-1].CumulativeGasUsed
	}
	gasSurplus := receipt.CumulativeGasUsed - previous - transferGas
	require.NotZero(t, gasSurplus, "the transfer must follow the failed bundle")
	transferHash := transfer.Hash()

	// Reading from the database after a restart must not report the surplus
	// gas as part of the transfer's gas used.
	require.NoError(t, net.Restart())
	client2, err := net.GetClient()
	require.NoError(t, err)
	defer client2.Close()

	receipt, err = client2.TransactionReceipt(t.Context(), transferHash)
	require.NoError(t, err)
	require.Equal(t, uint64(transferGas), receipt.GasUsed)
	require.GreaterOrEqual(t, receipt.CumulativeGasUsed, receipt.GasUsed+gasSurplus)

	// The block level view is consistent: all receipts report their own gas.
	receipts := getBlockReceipts(t, client2, receipt.BlockNumber)
	block, err := client2.BlockByNumber(t.Context(), receipt.BlockNumber)
	require.NoError(t, err)
	reported := uint64(0)
	for _, r := range receipts {
		reported += r.GasUsed
	}
	require.Equal(t, reported+gasSurplus, block.GasUsed())
}

func getBlockReceipts(
	t *testing.T,
	client *tests.PooledEhtClient,
	blockNumber *big.Int,
) []*types.Receipt {
	t.Helper()
	block, err := client.BlockByNumber(t.Context(), blockNumber)
	require.NoError(t, err)
	receipts := make([]*types.Receipt, 0, len(block.Transactions()))
	for _, tx := range block.Transactions() {
		receipt, err := client.TransactionReceipt(t.Context(), tx.Hash())
		require.NoError(t, err)
		receipts = append(receipts, receipt)
	}
	return receipts
}
