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
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/proxy"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/legacy_registry"
	"github.com/0xsoniclabs/sonic/tests/contracts/network_sponsor"
	"github.com/0xsoniclabs/sonic/tests/contracts/network_sponsor_tracking"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestNetworkSponsorship_NetworkSponsored_NoSecondTransaction verifies that a network-sponsored
// transaction produces exactly one receipt in the block and no deduction/track tx.
func TestNetworkSponsorship_NetworkSponsored_NoSecondTransaction(t *testing.T) {
	net := startNetWithSubsidies(t)

	// Install the network-sponsored registry (no tracking).
	installRegistry(t, net, network_sponsor.DeployNetworkSponsor)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	sender := tests.NewAccount()

	// Submit a sponsored transaction (gasPrice = 0).
	tx := makeSponsorRequestTransaction(t, &types.LegacyTx{
		To:  &common.Address{0x42},
		Gas: 21000,
	}, net.GetChainId(), sender)
	require.NoError(t, client.SendTransaction(t.Context(), tx))

	receipt, err := net.GetReceipt(tx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
	require.NoError(t, err)

	// Find the sponsored tx in the block.
	found := false
	for i, blockTx := range block.Transactions() {
		if blockTx.Hash() != tx.Hash() {
			continue
		}
		found = true

		// Network-sponsored: must be the only tx or, if there are more, the next one
		// must NOT be an internal registry call.
		if i+1 < len(block.Transactions()) {
			next := block.Transactions()[i+1]
			require.False(t,
				internaltx.IsInternal(next) &&
					next.To() != nil && *next.To() == registry.GetAddress(),
				"network-sponsored tx should not insert a registry call after the sponsored tx",
			)
		}
		break
	}
	require.True(t, found, "sponsored tx not found in block")
}

// TestNetworkSponsorship_NetworkSponsoredTracked_TrackTransactionInserted verifies that a
// network-sponsored-with-tracking transaction produces two receipts: the sponsored tx and a
// track tx, and that the track tx carries the correct tracking ID and fee.
func TestNetworkSponsorship_NetworkSponsoredTracked_TrackTransactionInserted(t *testing.T) {
	net := startNetWithSubsidies(t)

	// Install the network-sponsored-with-tracking registry.
	_, trackingRegistry := installRegistry(t, net, network_sponsor_tracking.DeployNetworkSponsorTracking)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// Fetch the constant tracking ID from the contract.
	expectedTrackingId, err := trackingRegistry.TRACKINGID(nil)
	require.NoError(t, err)

	sender := tests.NewAccount()

	tx := makeSponsorRequestTransaction(t, &types.LegacyTx{
		To:  &common.Address{0x42},
		Gas: 21000,
	}, net.GetChainId(), sender)
	require.NoError(t, client.SendTransaction(t.Context(), tx))

	receipt, err := net.GetReceipt(tx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
	require.NoError(t, err)

	// Find the sponsored tx and check the following tx is a track tx.
	found := false
	for i, blockTx := range block.Transactions() {
		if blockTx.Hash() != tx.Hash() {
			continue
		}
		found = true
		require.Less(t, i+1, len(block.Transactions()), "track tx should follow the sponsored tx")
		trackTx := block.Transactions()[i+1]

		require.True(t, subsidies.IsTrackTransaction(trackTx), "second tx must be a track transaction")
		require.Equal(t, registry.GetAddress(), *trackTx.To())

		// Verify the tracking ID embedded in the track tx's calldata.
		trackTxData := trackTx.Data()
		require.Equal(t, expectedTrackingId[:], trackTxData[4:36], "tracking ID in track tx must match registry's TRACKING_ID")

		// Verify the reported fee: (gasUsed + overhead) * baseFee
		reportedFee, err := subsidies.ParseTrackAmount(trackTx)
		require.NoError(t, err)
		require.Greater(t, reportedFee.Sign(), 0, "track tx fee must be non-zero")

		trackReceipt, err := net.GetReceipt(trackTx.Hash())
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, trackReceipt.Status)

		break
	}
	require.True(t, found, "sponsored tx not found in block")
}

// TestNetworkSponsorship_LegacyChooseFund_32ByteResponse_FundBackedBehavior verifies
// that a registry returning a legacy 32-byte chooseFund response is treated as
// fund-backed, with a deductFees tx appended after execution.
func TestNetworkSponsorship_LegacyChooseFund_32ByteResponse_FundBackedBehavior(t *testing.T) {
	net := startNetWithSubsidies(t)

	// Install the legacy registry (delegates all calls via proxy to legacy code).
	installRegistry(t, net, legacy_registry.DeployLegacyRegistry)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// Fund the fixed fund ID (bytes32(uint256(1))) through the PROXY address so
	// that the storage update lands in the proxy's own storage (delegatecall
	// preserves caller storage).
	legacyRegViaProxy, err := legacy_registry.NewLegacyRegistry(registry.GetAddress(), client)
	require.NoError(t, err)
	fixedFundId := [32]byte{}
	fixedFundId[31] = 1
	fundReceipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		opts.Value = big.NewInt(1e18)
		return legacyRegViaProxy.Sponsor(opts, fixedFundId)
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, fundReceipt.Status)

	sender := tests.NewAccount()

	tx := makeSponsorRequestTransaction(t, &types.LegacyTx{
		To:  &common.Address{0x42},
		Gas: 21000,
	}, net.GetChainId(), sender)
	require.NoError(t, client.SendTransaction(t.Context(), tx))

	receipt, err := net.GetReceipt(tx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
	require.NoError(t, err)

	// Check that a deductFees tx follows the sponsored tx (fund-backed behavior).
	found := false
	for i, blockTx := range block.Transactions() {
		if blockTx.Hash() != tx.Hash() {
			continue
		}
		found = true
		require.Less(t, i+1, len(block.Transactions()), "deductFees tx should follow the sponsored tx")
		deductTx := block.Transactions()[i+1]
		require.True(t, subsidies.IsFeeChargeTransaction(deductTx),
			"second tx must be a fee charge (deductFees) transaction for fund-backed sponsorship")
		break
	}
	require.True(t, found, "sponsored tx not found in block")
}

func startNetWithSubsidies(t *testing.T) *tests.IntegrationTestNet {
	t.Helper()
	upgrades := opera.GetBrioUpgrades()
	upgrades.GasSubsidies = true
	return tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})
}

// installRegistry deploys a new registry contract and updates the proxy to
// point to it. Returns the deployed contract address.
func installRegistry[T any](
	t *testing.T,
	net *tests.IntegrationTestNet,
	deploy func(*bind.TransactOpts, bind.ContractBackend) (common.Address, *types.Transaction, *T, error),
) (common.Address, *T) {
	t.Helper()
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	instance, receipt, err := tests.DeployContract(net, deploy)
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

	return contractAddr, instance
}
