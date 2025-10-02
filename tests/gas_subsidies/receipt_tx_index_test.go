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

package gas_subsidies

import (
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestGasSubsidies_Receipts_HaveConsistentTransactionIndices(t *testing.T) {
	require := require.New(t)
	upgrades := []struct {
		name    string
		upgrade opera.Upgrades
	}{
		{name: "sonic", upgrade: opera.GetSonicUpgrades()},
		{name: "allegro", upgrade: opera.GetAllegroUpgrades()},
		// Brio is commented out until the gas cap is properly handled for internal transactions.
		//{name: "brio", upgrade: opera.GetBrioUpgrades()},
	}
	singleProposerOption := map[string]bool{
		"singleProposer": true,
		"distributed":    false,
	}

	for _, test := range upgrades {
		for mode, enabled := range singleProposerOption {
			t.Run(fmt.Sprintf("%s/%v", test.name, mode), func(t *testing.T) {

				test.upgrade.GasSubsidies = true
				test.upgrade.SingleProposerBlockFormation = enabled
				net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
					Upgrades: &test.upgrade,
				})

				client, err := net.GetClient()
				require.NoError(err)
				defer client.Close()

				sponsee := tests.NewAccount()
				receiverAddress := tests.NewAccount().Address()
				donation := big.NewInt(math.MaxInt64)

				nonSponsoredAccount := tests.MakeAccountWithBalance(t, net, big.NewInt(math.MaxInt64))

				// set up sponsorship, but drop the returned registry since it is not needed
				_ = Fund(t, net, sponsee.Address(), donation)
				suggestedGasPrice, err := client.SuggestGasPrice(t.Context())
				require.NoError(err)

				numTxs := uint64(5)
				hashes := []common.Hash{}

				// send interleaved non-sponsored and sponsored transactions
				// so that the effect on index of both kinds of transactions
				// can be verified
				for i := range numTxs {
					tx := &types.LegacyTx{
						To:       &receiverAddress,
						Gas:      21000,
						GasPrice: suggestedGasPrice,
						Nonce:    i,
					}
					// send a non-sponsored transaction
					signedTx := tests.CreateTransaction(t, net, tx, nonSponsoredAccount)
					hashes = append(hashes, signedTx.Hash())
					require.NoError(client.SendTransaction(t.Context(), signedTx), "failed to send  transaction %v", i)

					// send a sponsored transaction
					sponsoredTx := makeSponsorRequestTransaction(t, tx, net.GetChainId(), sponsee)
					hashes = append(hashes, sponsoredTx.Hash())
					require.NoError(client.SendTransaction(t.Context(), sponsoredTx), "failed to send sponsored transaction %v", i)
				}

				// wait for all of them to be processed
				// note that this list of receipts does not contain the receipts for the
				// internal payment transactions
				receipts, err := net.GetReceipts(hashes)
				require.NoError(err)

				// get the block with all the executed transactions.
				block, err := client.BlockByNumber(t.Context(), receipts[0].BlockNumber)
				require.NoError(err)
				require.Equal(len(block.Transactions()), len(receipts)+int(numTxs),
					"number of transactions in the block does not match number of receipts",
				)

				// get the receipts for all transactions in the block
				blockReceipts := []*types.Receipt{}
				err = client.Client().Call(&blockReceipts, "eth_getBlockReceipts", fmt.Sprintf("0x%v", block.Number().String()))
				require.NoError(err)

				for i, tx := range block.Transactions() {

					receipt, err := client.TransactionReceipt(t.Context(), tx.Hash())
					require.NoError(err, "failed to get receipt for tx %d", i)
					require.Equal(uint(i), receipt.TransactionIndex,
						"receipt index does not match transaction index for tx %d", i,
					)

					require.Equal(receipt, blockReceipts[i],
						"receipt does not match eth_getBlockReceipts for tx %d", i,
					)

					// verify that transaction index in the block is the one reported by the receipt
					// for internal payment as well as for non-sponsored transactionsn
					require.EqualValues(i, receipt.TransactionIndex)
					require.Equal(receipt.TxHash, tx.Hash(),
						"receipt tx hash does not match transaction hash for tx %d", i,
					)

					// verify that the receipt obtained from eth_getBlockReceipts matches
					// the one obtained from eth_getTransactionReceipt
					require.Equal(blockReceipts[i].TxHash, receipt.TxHash,
						"receipt tx hash does not match eth_getBlockReceipts for tx %d", i,
					)
					require.Equal(blockReceipts[i].TransactionIndex, receipt.TransactionIndex,
						"receipt index does not match eth_getBlockReceipts for tx %d", i,
					)

					// if this is a payment transaction, the one before must be the sponsored tx
					if internaltx.IsInternal(tx) {
						require.False(internaltx.IsInternal(block.Transactions()[i-1]),
							"payment transaction at index %d must be preceded by a sponsored transaction", i,
						)
						require.True(subsidies.IsSponsorshipRequest(block.Transactions()[i-1]),
							"transaction at index %d must be a sponsored transaction", i-1,
						)
					}

				}
			})
		}
	}
}
