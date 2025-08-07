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
	session := getIntegrationTestNetSession(t, opera.GetAllegroUpgrades())
	t.Parallel()

	// create a client
	client, err := session.GetClient()
	require.NoError(t, err, "failed to get client")
	defer client.Close()

	chainId, err := client.ChainID(t.Context())
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
					Address: session.GetSessionSponsor().Address(),
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
				txData = SetTransactionDefaults(t, session, txData, account)
				tx := SignTransaction(t, chainId, txData, account)
				cost := tx.Cost()

				//  endow account with less than the cost of the transaction
				receipt, err := session.EndowAccount(account.Address(), new(big.Int).Sub(cost, big.NewInt(1)))
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

				_, err = session.Run(tx)
				require.ErrorContains(t, err, "insufficient funds")
			})

			t.Run("is rejected with nonce too low", func(t *testing.T) {
				account := NewAccount()

				txData := txFactory(t, account)
				txData = SetTransactionDefaults(t, session, txData, account)
				tx := SignTransaction(t, chainId, txData, account)

				// provide enough funds for successful execution
				receipt, err := session.EndowAccount(account.Address(), big.NewInt(1e18))
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

				// submit transaction once
				receipt, err = session.Run(tx)
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

				for {
					// submit transaction again
					_, err := session.Run(tx)
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
