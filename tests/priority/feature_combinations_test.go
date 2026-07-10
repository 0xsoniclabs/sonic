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

package priority

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	testbundles "github.com/0xsoniclabs/sonic/tests/bundles"
	"github.com/0xsoniclabs/sonic/tests/gas_subsidies"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestPriority_PriorityAndBundles(t *testing.T) {
	require := require.New(t)

	net, client, signer := netClientSignerWithPriorities(t, func(u *opera.Upgrades) {
		u.TransactionBundles = true
	})
	defer client.Close()

	// Register the bundle envelope sender as prioritized.
	envelopeSender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
	setPrioritized(t, net, envelopeSender.Address(), 1, 1, common.Hash{0x01})

	block, err := client.BlockNumber(t.Context())
	require.NoError(err)

	// Build a bundle whose envelope is signed by the prioritized sender and
	// whose inner txs are ordinary funds transfers signed by the same account.
	innerTx0 := tests.SetTransactionDefaults(t, net, &types.AccessListTx{
		To:    &common.Address{0xa1},
		Value: big.NewInt(1),
	}, envelopeSender)
	innerTx1 := *innerTx0
	innerTx1.To = &common.Address{0xa2}
	innerTx1.Value = big.NewInt(2)
	innerTx1.Nonce++

	envelope, plan := bundle.NewBuilder().
		WithSigner(signer).
		SetEarliest(block).
		SetEnvelopeSenderKey(envelopeSender.PrivateKey).
		AllOf(
			bundle.Step(envelopeSender.PrivateKey, innerTx0),
			bundle.Step(envelopeSender.PrivateKey, &innerTx1),
		).
		BuildEnvelopeAndPlan()

	// Pre-build ordinary traffic.
	ordinaryTxs := buildOrdinaryTraffic(t, net, signer, 4, 4)

	firstBlock, err := client.BlockNumber(t.Context())
	require.NoError(err)

	// Submit the sponsored tx and every ordinary tx in one batch.
	batch := append([]*types.Transaction{envelope}, ordinaryTxs...)
	hashes, err := net.SendAll(batch)
	require.NoError(err)

	// Wait for atomic bundle execution.
	info, err := testbundles.WaitForBundleExecution(t.Context(), client.Client(), plan.Hash())
	require.NoError(err)
	require.EqualValues(2, info.Count, "bundle must execute atomically (2 inner txs)")

	waitForReceipts(t, net, hashes[1:])

	requirePriorityAppliedSince(t, net, signer, firstBlock, true,
		func(a common.Address) bool { return a == envelopeSender.Address() })
}

func TestPriority_PriorityAndSponsorship(t *testing.T) {
	require := require.New(t)

	net, client, signer := netClientSignerWithPriorities(t, func(u *opera.Upgrades) {
		u.GasSubsidies = true
	})
	defer client.Close()

	// Create a sponsee, fund its sponsorship, and register it as prioritized.
	sponsee := tests.NewAccount()
	gas_subsidies.Fund(t, net, sponsee.Address(), big.NewInt(1e18))
	setPrioritized(t, net, sponsee.Address(), 1, 1, common.Hash{0x02})

	// Build a sponsorship request (GasPrice=0) from the sponsee.
	sponsoredTx := types.MustSignNewTx(sponsee.PrivateKey, signer, &types.LegacyTx{
		Nonce:    0,
		To:       &common.Address{0xb1},
		Value:    big.NewInt(0),
		Gas:      21000,
		GasPrice: big.NewInt(0),
	})
	require.True(subsidies.IsSponsorshipRequest(sponsoredTx))

	// Pre-build ordinary traffic.
	ordinaryTxs := buildOrdinaryTraffic(t, net, signer, 4, 4)

	firstBlock, err := client.BlockNumber(t.Context())
	require.NoError(err)

	// Submit the sponsored tx and every ordinary tx in one batch.
	batch := append([]*types.Transaction{sponsoredTx}, ordinaryTxs...)
	hashes, err := net.SendAll(batch)
	require.NoError(err)

	// Wait for the sponsored tx to be executed.
	receipt, err := net.GetReceipt(sponsoredTx.Hash())
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Verify the sponsor payment tx is emitted right after the sponsored one.
	sponsoredBlock, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
	require.NoError(err)
	txs := sponsoredBlock.Transactions()
	require.Less(int(receipt.TransactionIndex)+1, len(txs),
		"sponsored tx must be followed by a payment tx")
	payment := txs[receipt.TransactionIndex+1]
	require.True(internaltx.IsInternal(payment),
		"tx immediately after sponsored tx must be an internal payment")
	paymentReceipt, err := net.GetReceipt(payment.Hash())
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, paymentReceipt.Status)

	waitForReceipts(t, net, hashes[1:])

	requirePriorityAppliedSince(t, net, signer, firstBlock, true,
		func(a common.Address) bool { return a == sponsee.Address() })
}
