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

package tests

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestBlobTransactions_AreRejectedStartingWithCanto(t *testing.T) {
	for name, upgrades := range map[string]opera.Upgrades{
		"Brio":  opera.GetBrioUpgrades(),
		"Canto": opera.GetCantoUpgrades(),
	} {
		t.Run(name, func(t *testing.T) {
			upgrades.TransactionBundles = true
			session := getIntegrationTestNetSession(t, upgrades)
			t.Parallel()

			t.Run("submitted to the pool", func(t *testing.T) {
				account := MakeAccountWithBalance(t, session, big.NewInt(1e18))
				tx := makeEmptyBlobTx(t, session, account, 0)

				_, err := session.Send(tx)
				if upgrades.Canto {
					require.ErrorContains(t, err, types.ErrTxTypeNotSupported.Error())
				} else {
					require.NoError(t, err)
					requireSuccessfulReceipt(t, session, tx)
				}
			})

			t.Run("force emitted", func(t *testing.T) {
				account := MakeAccountWithBalance(t, session, big.NewInt(1e18))
				tx := makeEmptyBlobTx(t, session, account, 0)

				block := forceEmitWithMarker(t, session, tx)
				if upgrades.Canto {
					requireNotPresent(t, session, tx, account, block)
				} else {
					requireSuccessfulReceipt(t, session, tx)
				}
			})

			t.Run("bundled and submitted to the pool", func(t *testing.T) {
				account := MakeAccountWithBalance(t, session, big.NewInt(1e18))
				envelope, tx := makeBlobBundle(t, session, account, 0)

				_, err := session.Send(envelope)
				if upgrades.Canto {
					require.ErrorContains(t, err, "not supported since Canto")
				} else {
					require.NoError(t, err)
					requireSuccessfulReceipt(t, session, tx)
				}
			})

			t.Run("bundled and force emitted", func(t *testing.T) {
				account := MakeAccountWithBalance(t, session, big.NewInt(1e18))
				envelope, tx := makeBlobBundle(t, session, account, 0)

				block := forceEmitWithMarker(t, session, envelope)
				if upgrades.Canto {
					requireNotPresent(t, session, tx, account, block)
				} else {
					requireSuccessfulReceipt(t, session, tx)
				}
			})
		})
	}
}

func TestBlobTransactions_PooledBeforeCantoAreRemovedWhenCantoActivates(t *testing.T) {
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true
	net := StartIntegrationTestNet(t, IntegrationTestNetOptions{Upgrades: &upgrades})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// Nonce gaps keep both transactions queued in the pool until Canto.
	accounts := MakeAccountsWithBalance(t, net, 2, big.NewInt(1e18))
	blobTx := makeEmptyBlobTx(t, net, accounts[0], 1)
	envelope, _ := makeBlobBundle(t, net, accounts[1], 1)

	pooled := []*types.Transaction{blobTx, envelope}
	for _, tx := range pooled {
		_, err := net.Send(tx)
		require.NoError(t, err)
		_, _, err = client.TransactionByHash(t.Context(), tx.Hash())
		require.NoError(t, err, "transaction should be in the pool")
	}

	type rulesDiff[T any] struct{ Upgrades T }
	type cantoFlag struct{ Canto bool }
	UpdateNetworkRules(t, net, rulesDiff[cantoFlag]{Upgrades: cantoFlag{Canto: true}})
	net.AdvanceEpoch(t, 1)
	require.True(t, GetNetworkRules(t, net).Upgrades.Canto)

	for _, tx := range pooled {
		err := WaitFor(t.Context(), func(ctx context.Context) (bool, error) {
			_, _, err := client.TransactionByHash(ctx, tx.Hash())
			return errors.Is(err, ethereum.NotFound), nil
		})
		require.NoError(t, err, "transaction should be removed from the pool")
	}

	// Closing the nonce gap must not bring the blob transaction back.
	gapFiller := CreateTransaction(t, net, &types.LegacyTx{To: &common.Address{}, Nonce: 0}, accounts[0])
	receipt, err := net.Run(gapFiller)
	require.NoError(t, err)
	requireNotPresent(t, net, blobTx, accounts[0], receipt.BlockNumber)
}

// makeEmptyBlobTx creates a blob transaction without blobs, the only kind
// accepted by Sonic before Canto. A zero nonce is replaced by the sender's
// pending nonce.
func makeEmptyBlobTx(t *testing.T, session IntegrationTestNetSession, account *Account, nonce uint64) *types.Transaction {
	return CreateTransaction(t, session, blobTxData(nonce), account)
}

func blobTxData(nonce uint64) *types.BlobTx {
	return &types.BlobTx{
		Nonce:      nonce,
		Gas:        21_000,
		To:         common.Address{0x42},
		Value:      uint256.NewInt(0),
		GasTipCap:  uint256.NewInt(0),
		BlobFeeCap: uint256.NewInt(1),
	}
}

// makeBlobBundle wraps a single blob transaction of the account into a bundle,
// sending the envelope with the given nonce from a fresh account.
func makeBlobBundle(
	t *testing.T,
	session IntegrationTestNetSession,
	account *Account,
	envelopeNonce uint64,
) (envelope, blobTx *types.Transaction) {
	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()
	block, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	envelope, txBundle, _ := bundle.NewBuilder().
		WithSigner(types.LatestSignerForChainID(session.GetChainId())).
		SetEarliest(block).
		SetEnvelopeNonce(envelopeNonce).
		AllOf(bundle.Step(account.PrivateKey, SetTransactionDefaults(t, session, blobTxData(0), account))).
		BuildEnvelopeBundleAndPlan()
	return envelope, txBundle.GetTransactionsInReferencedOrder()[0]
}

// forceEmitWithMarker proposes tx together with a marker transaction and
// returns the block the marker was included in.
func forceEmitWithMarker(t *testing.T, session IntegrationTestNetSession, tx *types.Transaction) *big.Int {
	marker := CreateTransaction(t, session, &types.LegacyTx{To: &common.Address{}},
		MakeAccountWithBalance(t, session, big.NewInt(1e18)))
	_, err := session.ForceEmitAll(t.Context(), []*types.Transaction{tx, marker})
	require.NoError(t, err)
	receipt, err := session.GetReceipt(marker.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	return receipt.BlockNumber
}

func requireSuccessfulReceipt(t *testing.T, session IntegrationTestNetSession, tx *types.Transaction) {
	t.Helper()
	receipt, err := session.GetReceipt(tx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
}

// requireNotPresent checks that tx was not executed up to the given block.
func requireNotPresent(t *testing.T, session IntegrationTestNetSession, tx *types.Transaction, sender *Account, block *big.Int) {
	t.Helper()
	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	_, err = client.TransactionReceipt(t.Context(), tx.Hash())
	require.ErrorIs(t, err, ethereum.NotFound, "transaction should be skipped")
	nonce, err := client.NonceAt(t.Context(), sender.Address(), block)
	require.NoError(t, err)
	require.Equal(t, tx.Nonce(), nonce, "sender nonce should not advance")
}
