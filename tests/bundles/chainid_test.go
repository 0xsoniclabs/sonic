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
	"time"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestBundle_WrongChainIdIsRejectedByPoolAndSkippedInConsensus(t *testing.T) {
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// Use a chain ID that differs from the network's.
	wrongChainId := new(big.Int).Add(net.GetChainId(), big.NewInt(1))
	wrongSigner := types.LatestSignerForChainID(wrongChainId)
	correctSigner := types.LatestSignerForChainID(net.GetChainId())

	sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	block, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	t.Run("envelope_signed_with_wrong_chain_id", func(t *testing.T) {
		// Build a bundle where the envelope is signed with the wrong chain ID
		// but the inner transaction uses the correct chain ID.
		innerTx := tests.SetTransactionDefaults(t, net, &types.AccessListTx{}, sender)
		envelope, plan := bundle.NewBuilder().
			WithSigner(wrongSigner).
			SetEarliest(block).
			AllOf(bundle.Step(sender.PrivateKey, innerTx)).
			BuildEnvelopeAndPlan()

		// The pool should reject the envelope.
		_, err := net.Send(envelope)
		require.Error(t, err, "envelope with wrong chain ID should be rejected by pool")

		// Force-emit into consensus — the bundle should yield no results.
		_, err = net.ForceEmit(t.Context(), envelope)
		require.NoError(t, err, "force-emit should succeed")

		time.Sleep(3 * time.Second)

		_, err = GetBundleInfo(t.Context(), client.Client(), plan.Hash())
		require.ErrorIs(t, err, ethereum.NotFound,
			"bundle with wrong chain ID envelope should not be executed")
	})

	t.Run("bundled_transaction_signed_with_wrong_chain_id", func(t *testing.T) {
		// Build a bundle where the envelope uses the correct chain ID
		// but the inner transaction is signed with the wrong chain ID.
		innerTx := tests.SetTransactionDefaults(t, net, &types.AccessListTx{}, sender)

		envelope, plan := bundle.NewBuilder().
			WithSigner(wrongSigner).
			SetEarliest(block).
			AllOf(bundle.Step(sender.PrivateKey, innerTx)).
			BuildEnvelopeAndPlan()

		// sign the envelope again with the correct signer. this bypasses the
		// issue of the builder not allowing to have different signer for internal
		// tx and the envelope.
		key, err := crypto.GenerateKey()
		require.NoError(t, err)
		envelope = types.MustSignNewTx(key, correctSigner, &types.LegacyTx{
			// utils.GetTxData copies the correct signature,
			// and attempting to sign again panics. We need to copy the
			// payload without v,r,s values to sign again
			Nonce:    envelope.Nonce(),
			GasPrice: envelope.GasPrice(),
			Gas:      envelope.Gas(),
			To:       envelope.To(),
			Value:    envelope.Value(),
			Data:     envelope.Data(),
		})

		// The pool should reject the envelope.
		_, err = net.Send(envelope)
		require.Error(t, err, "envelope with wrong chain ID inner tx should be rejected by pool")

		// Force-emit into consensus — the bundle should yield no results.
		_, err = net.ForceEmit(t.Context(), envelope)
		require.NoError(t, err, "force-emit should succeed")

		time.Sleep(3 * time.Second)

		_, err = GetBundleInfo(t.Context(), client.Client(), plan.Hash())
		require.ErrorIs(t, err, ethereum.NotFound,
			"bundle with wrong chain ID inner tx should not be executed")
	})
}
