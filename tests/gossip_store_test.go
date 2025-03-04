package tests

import (
	"context"
	"math/big"
	"reflect"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestGossipStore_CanTransactionsBeRetrievedFromBlocksAfterRestart(t *testing.T) {

	// This test will execute a series of transactions.
	// After restarting the network, it will query the block where each transaction
	// was executed and check if the transaction is present in the block and the
	// values match, by comparing the hashes.

	net, err := StartIntegrationTestNet(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to start the fake network: %v", err)
	}
	defer net.Stop()

	client, err := net.GetClient()
	require.NoError(t, err)

	chainId, err := client.ChainID(context.Background())
	require.NoError(t, err)

	sender := makeAccountWithBalance(t, net, 1e18)
	senderAddress := sender.Address()

	// launch one transaction from each type
	txs := make([]*types.Transaction, 0)

	// Type 0: legacy transaction
	txs = append(txs, signTransaction(t, chainId,
		&types.LegacyTx{
			Nonce:    0,
			To:       &senderAddress,
			Gas:      1e6,
			GasPrice: big.NewInt(500e9),
		},
		sender))

	// Type 1: AccessList transaction
	txs = append(txs, signTransaction(t, chainId,
		&types.AccessListTx{
			Nonce:    1,
			To:       &senderAddress,
			Gas:      1e6,
			GasPrice: big.NewInt(500e9),
			AccessList: types.AccessList{
				{Address: senderAddress, StorageKeys: []common.Hash{{0x01}}},
			},
		},
		sender))

	// Type 2: DynamicFee transaction
	txs = append(txs, signTransaction(t, chainId,
		&types.DynamicFeeTx{
			Nonce:     2,
			To:        &senderAddress,
			Gas:       1e6,
			GasFeeCap: big.NewInt(500e9),
			GasTipCap: big.NewInt(500e9),
		},
		sender))

	// Type 3: Blob transaction
	txs = append(txs, signTransaction(t, chainId,
		&types.BlobTx{
			Nonce:     3,
			Gas:       1e6,
			GasFeeCap: uint256.NewInt(500e9),
			GasTipCap: uint256.NewInt(500e9),
		},
		sender))

	// Type 4: SetCode transaction
	authorization, err := types.SignSetCode(sender.PrivateKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(chainId),
		Address: common.Address{42},
		Nonce:   5,
	})
	require.NoError(t, err, "failed to sign SetCode authorization")
	txs = append(txs, signTransaction(t, chainId,
		&types.SetCodeTx{
			Nonce:     4,
			To:        senderAddress,
			Gas:       1e6,
			GasFeeCap: uint256.NewInt(500e9),
			GasTipCap: uint256.NewInt(500e9),
			AuthList:  []types.SetCodeAuthorization{authorization},
		},
		sender))

	for _, tx := range txs {
		err := client.SendTransaction(context.Background(), tx)
		require.NoError(t, err)
	}

	executedIn := make(map[*types.Transaction]*big.Int, len(txs))
	for i, tx := range txs {
		receipt, err := net.GetReceipt(tx.Hash())
		require.NoError(t, err, "failed to get receipt for tx%d", i)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "tx%d failed", i)
		require.NotNil(t, receipt.BlockNumber)
		executedIn[tx] = receipt.BlockNumber
	}

	err = net.Restart()
	require.NoError(t, err, "failed to restart network; %v", err)

	// query last block, retrieve executed transactions
	client, err = net.GetClient()
	require.NoError(t, err)

	for tx, blockNumber := range executedIn {
		block, err := client.BlockByNumber(context.Background(), blockNumber)
		require.NoError(t, err, "failed to get block %v", blockNumber)

		require.True(t,
			slices.ContainsFunc(block.Transactions(), func(received *types.Transaction) bool {
				return received.Hash() == tx.Hash()
			}))
	}
}

func signTransaction(
	t *testing.T,
	chainId *big.Int,
	payload any,
	from *Account,
) *types.Transaction {
	t.Helper()

	switch tx := payload.(type) {
	case *types.LegacyTx:
		res, err := types.SignTx(
			types.NewTx(tx),
			types.NewEIP155Signer(chainId),
			from.PrivateKey)
		require.NoError(t, err)
		return res
	case *types.AccessListTx:
		res, err := types.SignTx(
			types.NewTx(tx),
			types.NewEIP2930Signer(chainId),
			from.PrivateKey)
		require.NoError(t, err)
		return res
	case *types.DynamicFeeTx:
		res, err := types.SignTx(
			types.NewTx(tx),
			types.NewLondonSigner(chainId),
			from.PrivateKey)
		require.NoError(t, err)
		return res
	case *types.BlobTx:
		res, err := types.SignTx(
			types.NewTx(tx),
			types.NewCancunSigner(chainId),
			from.PrivateKey)
		require.NoError(t, err)
		return res
	case *types.SetCodeTx:
		res, err := types.SignTx(
			types.NewTx(tx),
			types.NewPragueSigner(chainId),
			from.PrivateKey)
		require.NoError(t, err)
		return res
	default:
		t.Error("unsupported transaction type ", reflect.TypeOf(payload))
		return nil
	}
}
