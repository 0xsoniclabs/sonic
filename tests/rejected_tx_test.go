package tests

import (
	"context"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestRejectedTx_TransactionsAreRejectedBecauseOfAccountState(t *testing.T) {

	// start network
	net := StartIntegrationTestNet(t,
		IntegrationTestNetOptions{
			FeatureSet: opera.AllegroFeatures,
		})

	// create a client
	client, err := net.GetClient()
	require.NoError(t, err, "failed to get client")
	defer client.Close()

	chainId, err := client.ChainID(context.Background())
	require.NoError(t, err, "failed to get chain ID::")

	// One authorization is required for SetCodeTx
	auth, err := types.SignSetCode(net.GetSessionSponsor().PrivateKey,
		types.SetCodeAuthorization{
			ChainID: *uint256.MustFromBig(chainId),
			Address: net.GetSessionSponsor().Address(),
			Nonce:   0,
		})
	require.NoError(t, err)

	testTransactions := map[string]types.TxData{
		"legacy tx":               &types.LegacyTx{},
		"access list no entries ": &types.AccessListTx{},
		"access list tx with one entry": &types.AccessListTx{
			AccessList: []types.AccessTuple{
				{Address: common.Address{0x42}, StorageKeys: []common.Hash{common.Hash{0x42}}},
			},
		},
		"dynamic fee tx": &types.DynamicFeeTx{},
		"blob tx":        &types.BlobTx{},
		"set code tx": &types.SetCodeTx{
			AuthList: []types.SetCodeAuthorization{auth, auth, auth},
		},
	}

	for name, tx := range testTransactions {
		t.Run(name, func(t *testing.T) {

			t.Run("is rejected with insufficient balance", func(t *testing.T) {
				account := NewAccount()

				txData := setTransactionDefaults(t, net, tx, account)
				tx := signTransaction(t, chainId, txData, account)
				cost := tx.Cost()

				//  endow account with less than the cost of the transaction
				receipt, err := net.EndowAccount(account.Address(), new(big.Int).Sub(cost, big.NewInt(1)))
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

				_, err = net.Run(tx)
				require.ErrorContains(t, err, "insufficient funds")
			})

			t.Run("is rejected with nonce too low", func(t *testing.T) {

				account := NewAccount()

				txData := setTransactionDefaults(t, net, tx, account)
				tx := signTransaction(t, chainId, txData, account)

				// provide enough funds for successful execution
				receipt, err := net.EndowAccount(account.Address(), big.NewInt(1e18))
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

				// execute once to increment nonce
				receipt, err = net.Run(tx)
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status,
					"first execution should be successful")

				// transaction has been executed, this is not a replacement
				// but a new submission with nonce too low
				_, err = net.Run(tx)
				require.ErrorContains(t, err, "nonce too low")
			})
		})
	}
}
