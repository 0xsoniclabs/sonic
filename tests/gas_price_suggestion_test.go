package tests

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestGasPrice_SuggestedGasPricesApproximateActualBaseFees(t *testing.T) {
	require := require.New(t)
	net, client := makeNetAndClient(t)

	fees := []uint64{}
	suggestions := []uint64{}
	ctxt := context.Background()
	for i := 0; i < 10; i++ {
		suggestedPrice, err := client.SuggestGasPrice(ctxt)
		require.NoError(err)

		// new block
		receipt, err := net.EndowAccount(common.Address{42}, big.NewInt(100))
		require.NoError(err)

		lastBlock, err := client.BlockByNumber(ctxt, receipt.BlockNumber)
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

	net, client := makeNetAndClient(t)
	send := func(tx *types.Transaction) error {
		return client.SendTransaction(context.Background(), tx)
	}

	chainId, err := client.ChainID(context.Background())
	require.NoError(err, "failed to get chain ID::")

	nonce, err := client.NonceAt(context.Background(), net.GetSessionSponsor().Address(), nil)
	require.NoError(err, "failed to get nonce:")

	factory := &txFactory{
		senderKey: net.GetSessionSponsor().PrivateKey,
		chainId:   chainId,
	}

	lastBlock, err := client.BlockByNumber(context.Background(), nil)
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
	}

	// Everything over ~5% above the base fee should be accepted.
	feeCap := int64(baseFee + 7*baseFee/100)
	require.NoError(send(factory.makeLegacyTransactionWithPrice(t, nonce, feeCap, 0)))
	require.NoError(send(factory.makeAccessListTransactionWithPrice(t, nonce+1, feeCap, 0)))
	require.NoError(send(factory.makeDynamicFeeTransactionWithPrice(t, nonce+2, feeCap, 0)))
	require.NoError(send(factory.makeBlobTransactionWithPrice(t, nonce+3, feeCap, 0)))
}

func makeNetAndClient(t *testing.T) (*IntegrationTestNet, *ethclient.Client) {
	net := StartIntegrationTestNet(t)

	client, err := net.GetClient()
	require.NoError(t, err)
	t.Cleanup(func() { client.Close() })

	return net, client
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
