// Copyright 2025 Sonic Operations Ltd
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
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestGasPrice_SuggestedGasPricesApproximateActualBaseFees(t *testing.T) {
	require := require.New(t)

	session := getIntegrationTestNetSession(t, opera.GetAllegroUpgrades())

	client, err := session.GetClient()
	require.NoError(err, "failed to get client")
	defer client.Close()

	fees := []uint64{}
	suggestions := []uint64{}
	for i := 0; i < 10; i++ {
		suggestedPrice, err := client.SuggestGasPrice(t.Context())
		require.NoError(err)

		// new block
		receipt, err := session.EndowAccount(common.Address{42}, big.NewInt(100))
		require.NoError(err)
		require.Equal(receipt.Status, types.ReceiptStatusSuccessful, "receipt status should be successful")

		lastBlock, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
		require.NoError(err)

		// store suggested and actual prices.
		suggestions = append(suggestions, suggestedPrice.Uint64())
		fees = append(fees, lastBlock.BaseFee().Uint64())
	}

	// Suggestions should over-estimate the actual prices by ~10%
	for i := 1; i < int(len(suggestions)); i++ {
		ratio := float64(suggestions[i]) / float64(fees[i-1])
		require.Less(1.09, ratio, "step %d, suggestion %d, fees %d", i, suggestions[i], fees[i-1])
		require.Less(ratio, 1.11, "step %d, suggestion %d, fees %d", i, suggestions[i], fees[i-1])
	}
}

func TestGasPrice_UnderpricedTransactionsAreRejected(t *testing.T) {
	require := require.New(t)

	session := getIntegrationTestNetSession(t, opera.GetAllegroUpgrades())

	client, err := session.GetClient()
	require.NoError(err, "failed to get client")
	defer client.Close()

	send := func(tx *types.Transaction) error {
		return client.SendTransaction(t.Context(), tx)
	}

	chainId, err := client.ChainID(t.Context())
	require.NoError(err, "failed to get chain ID::")

	// SetCode transactions are restricted to a max of one in-flight transaction
	// per address, so we need to use a different account.
	setCodeAccount := makeAccountWithBalance(t, session, big.NewInt(1e18))
	setCodeFactory := &txFactory{
		senderKey: setCodeAccount.PrivateKey,
		chainId:   chainId,
	}

	nonce, err := client.NonceAt(t.Context(), session.GetSessionSponsor().Address(), nil)
	require.NoError(err, "failed to get nonce:")

	factory := &txFactory{
		senderKey: session.GetSessionSponsor().PrivateKey,
		chainId:   chainId,
	}

	lastBlock, err := client.BlockByNumber(t.Context(), nil)
	require.NoError(err)

	// Everything below ~5% above the base fee should be rejected.
	baseFee := int(lastBlock.BaseFee().Uint64())
	for _, extra := range []int{-10, 0, baseFee / 100, 4 * baseFee / 100} {
		feeCap := int64(baseFee + extra)

		err = send(factory.makeLegacyTransactionWithPrice(t, nonce, feeCap, 0))
		require.ErrorContains(err, "transaction underpriced")

		err = send(factory.makeAccessListTransactionWithPrice(t, nonce, feeCap, 0))
		require.ErrorContains(err, "transaction underpriced")

		err = send(factory.makeDynamicFeeTransactionWithPrice(t, nonce, feeCap, 0))
		require.ErrorContains(err, "transaction underpriced")

		err = send(factory.makeBlobTransactionWithPrice(t, nonce, feeCap, 0))
		require.ErrorContains(err, "transaction underpriced")

		// SetCode transactions are restricted to a max of one in-flight transaction
		// per address, so we need to use a different account.
		err = send(setCodeFactory.makeSetCodeTransactionWithPrice(t, chainId, 0, feeCap, 0))
		require.ErrorContains(err, "transaction underpriced")
	}

	// Everything over ~5% above the base fee should be accepted.
	feeCap := int64(baseFee + 7*baseFee/100)
	require.NoError(send(factory.makeLegacyTransactionWithPrice(t, nonce, feeCap, 0)))
	require.NoError(send(factory.makeAccessListTransactionWithPrice(t, nonce+1, feeCap, 0)))
	require.NoError(send(factory.makeDynamicFeeTransactionWithPrice(t, nonce+2, feeCap, 0)))
	require.NoError(send(factory.makeBlobTransactionWithPrice(t, nonce+3, feeCap, 0)))
	// SetCode transactions are restricted to a max of one in-flight transaction
	// per address, so we need to use a different account.
	require.NoError(send(setCodeFactory.makeSetCodeTransactionWithPrice(t, chainId, 0, feeCap, 0)))
}

type txFactory struct {
	senderKey *ecdsa.PrivateKey
	chainId   *big.Int
}

func (f *txFactory) makeLegacyTransactionWithPrice(
	t *testing.T,
	nonce uint64,
	price int64,
	value int64,
) *types.Transaction {
	transaction, err := types.SignTx(types.NewTx(&types.LegacyTx{
		Gas:      21_000,
		GasPrice: big.NewInt(price),
		To:       &common.Address{},
		Nonce:    nonce,
		Value:    big.NewInt(value),
	}), types.NewEIP155Signer(f.chainId), f.senderKey)
	require.NoError(t, err, "failed to sign transaction")
	return transaction
}

func (f *txFactory) makeAccessListTransactionWithPrice(
	t *testing.T,
	nonce uint64,
	price int64,
	value int64,
) *types.Transaction {
	transaction, err := types.SignTx(types.NewTx(&types.AccessListTx{
		ChainID:  f.chainId,
		Gas:      21_000,
		GasPrice: big.NewInt(price),
		To:       &common.Address{},
		Nonce:    nonce,
		Value:    big.NewInt(value),
	}), types.NewEIP2930Signer(f.chainId), f.senderKey)
	require.NoError(t, err, "failed to sign transaction:")
	return transaction
}

func (f *txFactory) makeDynamicFeeTransactionWithPrice(
	t *testing.T,
	nonce uint64,
	price int64,
	value int64,
) *types.Transaction {
	transaction, err := types.SignTx(types.NewTx(&types.DynamicFeeTx{
		ChainID:   f.chainId,
		Gas:       21_000,
		GasFeeCap: big.NewInt(price),
		GasTipCap: big.NewInt(0),
		To:        &common.Address{},
		Nonce:     nonce,
		Value:     big.NewInt(value),
	}), types.NewLondonSigner(f.chainId), f.senderKey)
	require.NoError(t, err, "failed to sign transaction:")
	return transaction
}

func (f *txFactory) makeBlobTransactionWithPrice(
	t *testing.T,
	nonce uint64,
	price int64,
	value int64,
) *types.Transaction {
	transaction, err := types.SignTx(types.NewTx(&types.BlobTx{
		ChainID:    uint256.MustFromBig(f.chainId),
		Gas:        21_000,
		GasFeeCap:  uint256.MustFromBig(big.NewInt(price)),
		GasTipCap:  uint256.MustFromBig(big.NewInt(0)),
		Nonce:      nonce,
		Value:      uint256.MustFromBig(big.NewInt(value)),
		BlobFeeCap: uint256.NewInt(3e10), // fee cap for the blob data
		BlobHashes: nil,                  // blob hashes in the transaction
		Sidecar:    nil,                  // sidecar data in the transaction
	}), types.NewCancunSigner(f.chainId), f.senderKey)
	require.NoError(t, err, "failed to sign transaction:")
	return transaction
}

func (f *txFactory) makeSetCodeTransactionWithPrice(
	t *testing.T,
	chainId *big.Int,
	nonce uint64,
	price int64,
	value int64,
) *types.Transaction {
	auths, err := types.SignSetCode(f.senderKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(chainId),
		Address: common.Address{44},
		Nonce:   1,
	})
	require.NoError(t, err, "failed to sign SetCode authorization")

	transaction, err := types.SignTx(types.NewTx(&types.SetCodeTx{
		ChainID:   uint256.MustFromBig(chainId),
		Gas:       58_000,
		GasFeeCap: uint256.MustFromBig(big.NewInt(price)),
		GasTipCap: uint256.MustFromBig(big.NewInt(0)),
		Nonce:     nonce,
		Value:     uint256.MustFromBig(big.NewInt(value)),
		AuthList:  []types.SetCodeAuthorization{auths},
	}), types.NewPragueSigner(chainId), f.senderKey)
	require.NoError(t, err, "failed to sign transaction:")
	return transaction
}
