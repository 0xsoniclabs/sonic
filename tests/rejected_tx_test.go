package tests

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore"
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

	testTransactions := map[string]func(testing.TB, *Account) types.TxData{
		"legacy tx": func(testing.TB, *Account) types.TxData {
			return &types.LegacyTx{}
		},
		"access list no entries ": func(testing.TB, *Account) types.TxData {
			return &types.AccessListTx{}
		},
		"access list tx with one entry": func(testing.TB, *Account) types.TxData {
			return &types.AccessListTx{
				AccessList: []types.AccessTuple{
					{Address: common.Address{0x42}, StorageKeys: []common.Hash{{0x42}}},
				},
			}
		},
		"dynamic fee tx": func(testing.TB, *Account) types.TxData {
			return &types.DynamicFeeTx{}
		},
		"blob tx": func(testing.TB, *Account) types.TxData {
			return &types.BlobTx{}
		},
		"set code tx": func(t testing.TB, account *Account) types.TxData {

			// One authorization is required for SetCodeTx
			auth, err := types.SignSetCode(account.PrivateKey,
				types.SetCodeAuthorization{
					ChainID: *uint256.MustFromBig(chainId),
					Address: net.GetSessionSponsor().Address(),
					Nonce:   0,
				})
			require.NoError(t, err)

			return &types.SetCodeTx{
				AuthList: []types.SetCodeAuthorization{auth},
			}
		},
	}

	for name, txFactory := range testTransactions {
		t.Run(name, func(t *testing.T) {

			t.Run("is rejected with insufficient balance", func(t *testing.T) {
				account := NewAccount()

				txData := txFactory(t, account)
				txData = setTransactionDefaults(t, net, txData, account)
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

				txData := txFactory(t, account)
				txData = setTransactionDefaults(t, net, txData, account)
				tx := signTransaction(t, chainId, txData, account)

				// provide enough funds for successful execution
				receipt, err := net.EndowAccount(account.Address(), big.NewInt(1e18))
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

				// submit transaction once
				receipt, err = net.Run(tx)
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

				for {
					// submit transaction again
					_, err := net.Run(tx)
					require.Error(t, err)

					// Pool may take longer to purge the transaction after its execution.
					// If the transaction is still in the pool, try again
					if strings.Contains(err.Error(), evmcore.ErrAlreadyKnown.Error()) {
						continue
					}

					// eventually the transaction has been purged from the pool
					// and any subsequent submission with the same nonce is rejected
					require.ErrorContains(t, err, evmcore.ErrNonceTooLow.Error())
					break
				}
			})
		})
	}
}
