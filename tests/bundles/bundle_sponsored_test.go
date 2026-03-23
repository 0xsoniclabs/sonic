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
	session := sharedNetwork.GetIntegrationTestNetSession(t, upgrade)
	t.Parallel()

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// make a sponsorship request transaction
	sponsee, unsignedTx := makeSponsorshipRequestTx(t, session)

	// prepare the bundle with the sponsorship request transaction as payload,
	// but without a sponsorship transaction.
	blockNumber, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	bundleTx := bundle.NewBuilder().
		Earliest(blockNumber).
		With(bundle.Step(sponsee.PrivateKey, unsignedTx)).
		Build()

	// send the bundle.
	// NOTE: once bundle trial-run is implemented this submition will fail.
	// which is what this test should verify.
	err = client.SendTransaction(t.Context(), bundleTx)
	require.ErrorContains(t, err, "bundle is permanently blocked")
}

func TestBundle_CanRunSponsorshipAndSponsored(t *testing.T) {
	upgrade := opera.GetBrioUpgrades()
	upgrade.TransactionBundles = true
	upgrade.GasSubsidies = true
	session := sharedNetwork.GetIntegrationTestNetSession(t, upgrade)
	t.Parallel()

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// prepare sponsorship and sponsored transactions
	sponsee, unsignedTx := makeSponsorshipRequestTx(t, session)
	sponsor, txSponsorData := makeSponsorTx(t, session, sponsee)

	// prepare the bundle with both the sponsorship transaction and the sponsored transaction.
	blockNumber, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	envelope, bundle, plan := bundle.NewBuilder().
		Earliest(blockNumber).
		With(
			bundle.Step(sponsor.PrivateKey, txSponsorData),
			bundle.Step(sponsee.PrivateKey, unsignedTx),
		).
		BuildEnvelopeBundleAndPlan()

	// Send the bundle to the network and check that it is processed successfully.
	err = client.SendTransaction(t.Context(), envelope)
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
	position := uint(*info.Position)
	require.GreaterOrEqual(t, uint(len(txs)), position+3)
	require.Equal(t, txs[position].Hash(), bundle.Transactions[0].Hash())
	require.Equal(t, txs[position+1].Hash(), bundle.Transactions[1].Hash())
	require.True(t, internaltx.IsInternal(txs[position+2]))
}

func makeSponsorshipRequestTx(t *testing.T, session tests.IntegrationTestNetSession) (*tests.Account, *types.AccessListTx) {
	// create a sponsorship request transaction.
	txData := &types.AccessListTx{
		To:  &common.Address{0x42},
		Gas: 26850,
	}

	sponsee := tests.NewAccount()
	unsignedTx := tests.SetTransactionDefaults(t, session, txData, sponsee)
	unsignedTx.GasPrice = big.NewInt(0)

	return sponsee, unsignedTx
}

func makeSponsorTx(t *testing.T, session tests.IntegrationTestNetSession, sponsee *tests.Account) (*tests.Account, *types.AccessListTx) {

	client, err := session.GetClient()
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

	opts, err := session.GetTransactOptions(session.GetSessionSponsor())
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

	return session.GetSessionSponsor(), txSponsorData
}
