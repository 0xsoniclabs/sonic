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

package gas_subsidies

import (
	"context"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/proxy"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/0xsoniclabs/sonic/tests/contracts/failing_post_tx_registry"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestGasSubsidies_Brio_FailedPostTx_RollsBackSponsoredTransaction verifies
// that under the Brio rules, a sponsored transaction whose post-transaction
// (track) always reverts is fully rolled back: the state change made by the
// sponsored transaction is undone and no receipt is ever produced.
func TestGasSubsidies_Brio_FailedPostTx_RollsBackSponsoredTransaction(t *testing.T) {
	net := startNetWithSubsidies(t)
	installFailingRegistry(t, net)

	_, counterReceipt, err := tests.DeployContract(net, counter.DeployCounter)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, counterReceipt.Status)
	counterAddr := counterReceipt.ContractAddress

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// Build the sponsored transaction: sponsee calls counter.IncrementCounter().
	calldata, err := buildIncrementCounterCalldata()
	require.NoError(t, err)

	sponsee := tests.NewAccount()
	sponsoredTx := makeSponsorRequestTransaction(t, &types.LegacyTx{
		To:   &counterAddr,
		Gas:  100_000,
		Data: calldata,
	}, net.GetChainId(), sponsee)

	// Submit the sponsored transaction; the txpool accepts it because
	// IsCovered returns a valid sponsorship (the failing registry only
	// fails during the post-transaction execution, not during chooseFund).
	require.NoError(t, client.SendTransaction(t.Context(), sponsoredTx))

	// Advance one epoch to guarantee that blocks were produced after the
	// sponsored transaction entered the mempool.
	net.AdvanceEpoch(t, 1)

	// The state change (counter increment) must be absent: the Brio rollback
	// undoes both the sponsored transaction and the failing post-transaction.
	boundCounter, err := counter.NewCounter(counterAddr, client)
	require.NoError(t, err)
	count, err := boundCounter.GetCount(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, int64(0), count.Int64(),
		"counter must remain at 0: the sponsored tx was rolled back due to the failing post-tx")

	// No receipt should ever be produced for the sponsored transaction.
	_, err = net.TryGetReceipt(3*time.Second, sponsoredTx.Hash())
	require.ErrorIs(t, err, context.DeadlineExceeded,
		"sponsored tx must not have a receipt: it was rolled back")
}

// TestGasSubsidies_PreBrio_FailedPostTx_SponsoredTransactionIsIncluded verifies
// that before the Brio upgrade a sponsored transaction is still included in the
// block even when its post-transaction fails: no rollback occurs and the state
// change from the sponsored transaction persists.
func TestGasSubsidies_PreBrio_FailedPostTx_SponsoredTransactionIsIncluded(t *testing.T) {
	// Start a pre-Brio network with gas subsidies enabled.
	upgrades := opera.GetAllegroUpgrades()
	upgrades.GasSubsidies = true
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})
	installFailingRegistry(t, net)

	_, counterReceipt, err := tests.DeployContract(net, counter.DeployCounter)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, counterReceipt.Status)
	counterAddr := counterReceipt.ContractAddress

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	calldata, err := buildIncrementCounterCalldata()
	require.NoError(t, err)

	sponsee := tests.NewAccount()
	sponsoredTx := makeSponsorRequestTransaction(t, &types.LegacyTx{
		To:   &counterAddr,
		Gas:  100_000,
		Data: calldata,
	}, net.GetChainId(), sponsee)

	require.NoError(t, client.SendTransaction(t.Context(), sponsoredTx))

	// Wait for the sponsored transaction to be included in a block.
	// Pre-Brio: the sponsored tx is committed despite the failing post-tx.
	receipt, err := net.GetReceipt(sponsoredTx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status,
		"sponsored tx should be included and succeed pre-Brio")

	// The counter increment must have persisted (no rollback pre-Brio).
	boundCounter, err := counter.NewCounter(counterAddr, client)
	require.NoError(t, err)
	count, err := boundCounter.GetCount(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, int64(1), count.Int64(),
		"counter must be 1: pre-Brio does not roll back on a failing post-tx")

	// Verify that the post-transaction (track) is present in the block and
	// has a failed receipt.
	block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
	require.NoError(t, err)

	found := false
	for i, tx := range block.Transactions() {
		if tx.Hash() != sponsoredTx.Hash() {
			continue
		}
		found = true
		require.Less(t, i+1, len(block.Transactions()),
			"track post-tx must follow the sponsored tx in the block")
		trackTx := block.Transactions()[i+1]
		require.True(t, subsidies.IsTrackTransaction(trackTx),
			"next tx must be an IsTrackTransaction post-tx")
		trackReceipt, err := net.GetReceipt(trackTx.Hash())
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusFailed, trackReceipt.Status,
			"pre-Brio: failing track post-tx has a failed receipt but the sponsored tx is not rolled back")
		break
	}
	require.True(t, found, "sponsored tx must be present in the block pre-Brio")
}

// installFailingRegistry deploys a FailingPostTxRegistry and sets it as the
// active implementation in the subsidies proxy. The failing registry approves
// all sponsorships (mode 3, network-with-tracking) so the txpool pre-check
// passes, but its track() function always reverts, making every post-tx fail.
func installFailingRegistry(t *testing.T, net *tests.IntegrationTestNet) {
	t.Helper()

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	_, receipt, err := tests.DeployContract(net, failing_post_tx_registry.DeployFailingPostTxRegistry)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	contractAddr := receipt.ContractAddress

	proxyContract, err := proxy.NewProxy(registry.GetAddress(), client)
	require.NoError(t, err)

	updateReceipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return proxyContract.Update(opts, contractAddr)
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, updateReceipt.Status)
}

// buildIncrementCounterCalldata returns the ABI-encoded calldata for Counter.incrementCounter().
func buildIncrementCounterCalldata() ([]byte, error) {
	abi, err := counter.CounterMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return abi.Pack("incrementCounter")
}
