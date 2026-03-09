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

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/stretchr/testify/require"
)

func TestBundle_RejectsBundle_WithPayloadSponsorRequest_WithoutSponsorship(t *testing.T) {

	upgrade := opera.GetBrioUpgrades()
	upgrade.TransactionBundles = true
	upgrade.GasSubsidies = true
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrade,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// make a sponsorship request transaction
	txData := &types.AccessListTx{
		To:  &common.Address{0x42},
		Gas: 26850,
	}

	coordinator := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
	sponsee := tests.NewAccount()
	unsignedTx := tests.SetTransactionDefaults(t, net, txData, sponsee)
	unsignedTx.GasPrice = big.NewInt(0)

	signer := types.LatestSignerForChainID(net.GetChainId())
	chainId := net.GetChainId()

	blockNumber, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	txToSign, plan := prepareBundle(
		chainId, blockNumber,
		[]UnsignedTransaction{
			{
				Sender:      sponsee.Address(),
				Transaction: unsignedTx,
			}})

	signedTx := types.MustSignNewTx(sponsee.PrivateKey, signer, txToSign[0].Transaction)
	require.True(t, subsidies.IsSponsorshipRequest(signedTx))

	// Create the bundle transaction
	bundleTx := types.MustSignNewTx(
		coordinator.PrivateKey, signer,
		makeBundle(types.Transactions{signedTx}, plan),
	)

	// Check bundle construction.
	require.True(t, bundle.IsTransactionBundle(bundleTx))
	recoveredBundle, recoveredPlan, err := bundle.ValidateTransactionBundle(bundleTx, signer)
	require.NoError(t, err)
	require.NotNil(t, recoveredBundle)
	require.NotNil(t, recoveredPlan)
	require.Equal(t, plan, *recoveredPlan)
	require.EqualValues(t, 0, bundleTx.GasFeeCap().Uint64())

	// Run the bundle.
	err = client.SendTransaction(t.Context(), bundleTx)
	require.NoError(t, err)

	// Check that the sponsored transaction is not in the txpool.
	var content map[string]map[string]map[string]*ethapi.RPCTransaction
	err = client.Client().Call(&content, "txpool_content")
	require.NoError(t, err, "Should get txpool content")

	txPoolSponsee := len(content["pending"][sponsee.Address().String()]) +
		len(content["queued"][sponsee.Address().String()])
	require.Zero(t, txPoolSponsee, "There should be no transactions for the sponsee in the txpool")

	txPoolCoordinator := len(content["pending"][coordinator.Address().String()]) +
		len(content["queued"][coordinator.Address().String()])
	require.Equal(t, 1, txPoolCoordinator, "There should be no transactions for the sponsee in the txpool")

	info, err := waitForBundleExecution(t.Context(), client.Client(), plan.Hash())
	require.NoError(t, err)

	// The bundle is expected to be executed because it reached the processor,
	// regardless of the fact that the sponsored transaction was skipped.
	require.Equal(t, ethapi.BundleStatusExecuted, info.Status)
	require.NotNil(t, info.Block)
	block, err := client.BlockByNumber(t.Context(), big.NewInt(int64(*info.Block)))
	require.NoError(t, err)
	require.Empty(t, block.Transactions())
}

func TestBundle_CanRunSponsorshipAndSponsored(t *testing.T) {

	upgrade := opera.GetBrioUpgrades()
	upgrade.TransactionBundles = true
	upgrade.GasSubsidies = true
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrade,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// Make a sponsorship and a sponsored transaction.
	txData := &types.AccessListTx{
		To:  &common.Address{0x42},
		Gas: 26850,
	}

	sponsee := tests.NewAccount()
	unsignedTx := tests.SetTransactionDefaults(t, net, txData, sponsee)
	unsignedTx.GasPrice = big.NewInt(0)

	signer := types.LatestSignerForChainID(net.GetChainId())
	chainId := net.GetChainId()

	// create sponsorship - extract into a function.
	donation := big.NewInt(1e18)
	registryAddr := registry.GetAddress()
	registry, err := registry.NewRegistry(registryAddr, client)
	require.NoError(t, err)

	ok, fundId, err := registry.AccountSponsorshipFundId(nil, sponsee.Address())
	require.NoError(t, err)
	require.True(t, ok)

	opts, err := net.GetTransactOptions(net.GetSessionSponsor())
	require.NoError(t, err)

	opts.NoSend = true
	opts.Value = donation
	// this transaction is already singed and it need to be dropped.
	sponsorshipTx, err := registry.Sponsor(opts, fundId)
	require.NoError(t, err)

	txSponsorData := &types.AccessListTx{
		To:       &registryAddr,
		Value:    donation,
		Gas:      sponsorshipTx.Gas(),
		GasPrice: sponsorshipTx.GasPrice(),
		Data:     sponsorshipTx.Data(),
	}
	//// return sponsorship tx.

	blockNumber, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	txToSign, plan := prepareBundle(
		chainId, blockNumber,
		[]UnsignedTransaction{
			{
				Sender:      net.GetSessionSponsor().Address(),
				Transaction: txSponsorData,
			},
			{
				Sender:      sponsee.Address(),
				Transaction: unsignedTx,
			}})

	signedSponsorTx := types.MustSignNewTx(net.GetSessionSponsor().PrivateKey,
		signer, txToSign[0].Transaction)

	signedTx := types.MustSignNewTx(sponsee.PrivateKey, signer, txToSign[1].Transaction)
	require.True(t, subsidies.IsSponsorshipRequest(signedTx))

	// Create a bundle where the first transaction is a sponsorship and the
	// second transaction is a sponsored transaction.
	bundleTx := types.MustSignNewTx(
		net.GetSessionSponsor().PrivateKey, signer,
		makeBundle(types.Transactions{signedSponsorTx, signedTx}, plan),
	)

	// Check bundle construction.
	require.True(t, bundle.IsTransactionBundle(bundleTx))
	recoveredBundle, recoveredPlan, err := bundle.ValidateTransactionBundle(bundleTx, signer)
	require.NoError(t, err)
	require.NotNil(t, recoveredBundle)
	require.NotNil(t, recoveredPlan)
	require.Equal(t, plan, *recoveredPlan)
	require.EqualValues(t, 0, bundleTx.GasFeeCap().Uint64())

	// Send the bundle to the network and check that it is processed successfully.
	err = client.SendTransaction(t.Context(), bundleTx)
	require.NoError(t, err)

	info, err := waitForBundleExecution(t.Context(), client.Client(), plan.Hash())
	require.NoError(t, err)

	// The bundle is expected to be executed because it reached the processor,
	// regardless of the fact that the sponsored transaction was skipped.
	require.Equal(t, ethapi.BundleStatusExecuted, info.Status)
	require.NotNil(t, info.Block)

	block, err := client.BlockByNumber(t.Context(), big.NewInt(int64(*info.Block)))
	require.NoError(t, err)

	// sponsored transaction introduce an internal transaction,
	// so we expect 3 transactions in the block:
	// 1. the sponsorship transaction
	// 2. the sponsored transaction
	// 3. the internal transaction that transfer the fee from the sponsee to the sponsor
	require.Equal(t, len(block.Transactions()), 3)
	require.Equal(t, block.Transactions()[0].Hash(), signedSponsorTx.Hash())
	require.Equal(t, block.Transactions()[1].Hash(), signedTx.Hash())
}
