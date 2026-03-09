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
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"

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
	sponsee, unsignedTx := makeSponsorshipRequestTx(t, net)

	// prepare the bundle with the sponsorship request transaction as payload,
	// but without a sponsorship transaction.
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

	// sign the payload transaction and verify it is a sponsorship request.
	signedTx := types.MustSignNewTx(sponsee.PrivateKey, signer, txToSign[0].Transaction)
	require.True(t, subsidies.IsSponsorshipRequest(signedTx))

	bundleTx := types.MustSignNewTx(
		net.GetSessionSponsor().PrivateKey, signer,
		makeBundle(types.Transactions{signedTx}, plan),
	)

	// verify bundle construction.
	checkBundleIntegrity(t, signer, bundleTx, plan)

	// send the bundle.
	// NOTE: once bundle trial-run is implemented this submition will fail.
	// which is what this test should verify.
	err = client.SendTransaction(t.Context(), bundleTx)
	require.NoError(t, err)

	// check that the bundle tx made it into the txpool, but the sponsored transaction did not.
	checkAccountTxsInTxPool(t, client, sponsee, 0)
	checkAccountTxsInTxPool(t, client, net.GetSessionSponsor(), 1)

	info, err := waitForBundleExecution(t.Context(), client.Client(), plan.Hash())
	require.NoError(t, err)

	// The bundle is expected to be executed because it reached the processor,
	// regardless of the fact that the sponsored transaction was skipped.
	require.Equal(t, ethapi.BundleStatusExecuted, info.Status)
	require.NotNil(t, info.Block)

	// verify the block where the bundle was executed has no transactions.
	block, err := client.BlockByNumber(t.Context(), big.NewInt(int64(*info.Block)))
	require.NoError(t, err)
	require.Empty(t, block.Transactions())
}

// checkAccountTxsInTxPool checks that there are `want“ transactions for the given account in the txpool.
func checkAccountTxsInTxPool(t *testing.T, client *tests.PooledEhtClient, account *tests.Account, want int) {
	var content map[string]map[string]map[string]*ethapi.RPCTransaction
	err := client.Client().Call(&content, "txpool_content")
	require.NoError(t, err, "Should get txpool content")

	txPoolSponsee := len(content["pending"][account.Address().String()]) +
		len(content["queued"][account.Address().String()])
	require.Equal(t, want, txPoolSponsee, "There should be %d transactions for the account in the txpool", want)
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

	// prepare sponsorship and sponsored transactions
	sponsee, unsignedTx := makeSponsorshipRequestTx(t, net)
	sponsor, txSponsorData := makeSponsorTx(t, net, sponsee)

	// prepare the bundle with both the sponsorship transaction and the sponsored transaction.
	blockNumber, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	signer := types.LatestSignerForChainID(net.GetChainId())
	chainId := net.GetChainId()

	txToSign, plan := prepareBundle(
		chainId, blockNumber,
		[]UnsignedTransaction{
			{
				Sender:      sponsor.Address(),
				Transaction: txSponsorData,
			},
			{
				Sender:      sponsee.Address(),
				Transaction: unsignedTx,
			}})

	signedSponsorTx := types.MustSignNewTx(sponsor.PrivateKey,
		signer, txToSign[0].Transaction)

	signedTx := types.MustSignNewTx(sponsee.PrivateKey, signer, txToSign[1].Transaction)
	require.True(t, subsidies.IsSponsorshipRequest(signedTx))

	// Create a bundle where the first transaction is a sponsorship and the
	// second transaction is a sponsored transaction.
	bundleTx := types.MustSignNewTx(
		sponsor.PrivateKey, signer,
		makeBundle(types.Transactions{signedSponsorTx, signedTx}, plan),
	)

	// Check bundle construction.
	checkBundleIntegrity(t, signer, bundleTx, plan)

	// Send the bundle to the network and check that it is processed successfully.
	err = client.SendTransaction(t.Context(), bundleTx)
	require.NoError(t, err)

	info, err := waitForBundleExecution(t.Context(), client.Client(), plan.Hash())
	require.NoError(t, err)

	require.Equal(t, ethapi.BundleStatusExecuted, info.Status)
	require.NotNil(t, info.Block)
	require.NotNil(t, info.Position)

	block, err := client.BlockByNumber(t.Context(), big.NewInt(int64(*info.Block)))
	require.NoError(t, err)

	// sponsored transaction introduce an internal transaction,
	// so we expect 3 transactions in the block:
	// 1. the sponsorship transaction
	// 2. the sponsored transaction
	// 3. the internal transaction that transfer the fee from the sponsee to the sponsor
	txs := block.Transactions()
	position := *info.Position
	require.GreaterOrEqual(t, uint32(len(txs)), position+3)
	require.Equal(t, txs[position].Hash(), signedSponsorTx.Hash())
	require.Equal(t, txs[position+1].Hash(), signedTx.Hash())
	require.True(t, internaltx.IsInternal(txs[position+2]))
}

func makeSponsorshipRequestTx(t *testing.T, net *tests.IntegrationTestNet) (*tests.Account, *types.AccessListTx) {
	// create a sponsorship request transaction.
	txData := &types.AccessListTx{
		To:  &common.Address{0x42},
		Gas: 26850,
	}

	sponsee := tests.NewAccount()
	unsignedTx := tests.SetTransactionDefaults(t, net, txData, sponsee)
	unsignedTx.GasPrice = big.NewInt(0)

	return sponsee, unsignedTx
}

func makeSponsorTx(t *testing.T, net *tests.IntegrationTestNet, sponsee *tests.Account) (*tests.Account, *types.AccessListTx) {

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// create sponsorship for the sponsee.
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

	// The sponsorship transaction returned by the registry is already signed.
	// But since it needs to be modified, it is dropped and a new sponsor transaction is created.
	sponsorshipTx, err := registry.Sponsor(opts, fundId)
	require.NoError(t, err)

	txSponsorData := &types.AccessListTx{
		To:       &registryAddr,
		Value:    donation,
		Gas:      sponsorshipTx.Gas(),
		GasPrice: sponsorshipTx.GasPrice(),
		Data:     sponsorshipTx.Data(),
	}

	return net.GetSessionSponsor(), txSponsorData
}

func checkBundleIntegrity(t *testing.T, signer types.Signer, bundleTx *types.Transaction, plan bundle.ExecutionPlan) {
	require.True(t, bundle.IsTransactionBundle(bundleTx))
	recoveredBundle, recoveredPlan, err := bundle.ValidateTransactionBundle(bundleTx, signer)
	require.NoError(t, err)
	require.NotNil(t, recoveredBundle)
	require.NotNil(t, recoveredPlan)
	require.Equal(t, plan, *recoveredPlan)
	require.EqualValues(t, 0, bundleTx.GasFeeCap().Uint64())
}
