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
	"github.com/0xsoniclabs/sonic/gossip/contract/driverauth100"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/contracts/driverauth"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestGasSubsidies_Receipts_HaveConsistentTransactionIndices(t *testing.T) {

	upgrades := []struct {
		name    string
		upgrade opera.Upgrades
	}{
		{name: "sonic", upgrade: opera.GetSonicUpgrades()},
		{name: "allegro", upgrade: opera.GetAllegroUpgrades()},
		{name: "brio", upgrade: opera.GetBrioUpgrades()},
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
				require.NoError(t, err)
				defer client.Close()

				contract, err := driverauth100.NewContract(driverauth.ContractAddress, client)
				require.NoError(t, err, "failed to create contract instance")

				sponsee := tests.NewAccount()
				donation := big.NewInt(math.MaxInt64)

				// set up sponsorship, but drop the returned registry since it is not needed
				_ = Fund(t, net, sponsee.Address(), donation)

				transactionsLoad := map[string]struct {
					scenario func(t *testing.T, net *tests.IntegrationTestNet) []common.Hash
				}{
					"single sponsored transaction": {
						scenario: func(t *testing.T, net *tests.IntegrationTestNet) []common.Hash {
							nonce, err := client.PendingNonceAt(t.Context(), sponsee.Address())
							require.NoError(t, err)
							txData := &types.LegacyTx{
								Nonce: nonce,
								To:    &common.Address{0x1}, // not a contract creation, contract creation cannot be sponsored
								Gas:   21_000,
							}
							sponsoredTx := makeSponsorRequestTransaction(t, txData, net.GetChainId(), sponsee)
							hashes := []common.Hash{sponsoredTx.Hash()}

							client, err := net.GetClient()
							require.NoError(t, err)
							defer client.Close()

							require.NoError(t, client.SendTransaction(t.Context(), sponsoredTx), "failed to send single sponsored transaction %v")

							return hashes
						},
					},
					"multiple sponsored transactions are executed within the same block": {
						scenario: func(t *testing.T, net *tests.IntegrationTestNet) []common.Hash {
							// This scenario issues multiple sponsored transactions asynchronously
							// to facilitate their inclusion in the same block.

							const numTxs = 10
							txHashes := make([]common.Hash, 0, numTxs)

							StartingNonce, err := client.PendingNonceAt(t.Context(), sponsee.Address())
							require.NoError(t, err)

							for i := 0; i < numTxs; i++ {

								txData := &types.LegacyTx{
									Nonce: StartingNonce + uint64(i),
									To:    &common.Address{0x1}, // not a contract creation, contract creation cannot be sponsored
									Gas:   21_000,
								}
								sponsoredTx := makeSponsorRequestTransaction(t, txData, net.GetChainId(), sponsee)

								require.NoError(t, client.SendTransaction(t.Context(), sponsoredTx), "failed to send sponsored transaction %v", i)
								txHashes = append(txHashes, sponsoredTx.Hash())
							}
							return txHashes
						},
					},
					"multiple sponsored and non-sponsored transactions are executed within the same block": {
						scenario: func(t *testing.T, net *tests.IntegrationTestNet) []common.Hash {
							// This scenario issues multiple sponsored transactions asynchronously
							// to facilitate their inclusion in the same block.

							const numTxs = 10
							txHashes := make([]common.Hash, 0, numTxs*2)

							StartingNonce, err := client.PendingNonceAt(t.Context(), sponsee.Address())
							require.NoError(t, err)

							nonSponsoredAccount := tests.MakeAccountWithBalance(t, net, big.NewInt(math.MaxInt64))
							suggestedGasPrice, err := client.SuggestGasPrice(t.Context())
							require.NoError(t, err)

							for i := 0; i < numTxs; i++ {

								// sponsored transaction
								sponsoredTxData := &types.LegacyTx{
									Nonce: StartingNonce + uint64(i),
									To:    &common.Address{0x1}, // not a contract creation, contract creation cannot be sponsored
									Gas:   21_000,
								}
								sponsoredTx := makeSponsorRequestTransaction(t, sponsoredTxData, net.GetChainId(), sponsee)

								require.NoError(t, client.SendTransaction(t.Context(), sponsoredTx), "failed to send sponsored transaction %v", i)
								txHashes = append(txHashes, sponsoredTx.Hash())

								// non-sponsored transaction
								nonSponsoredTxData := &types.LegacyTx{
									Nonce:    uint64(i),            // nonce is ignored for non-sponsored transactions
									To:       &common.Address{0x1}, // not a contract creation, contract creation cannot be sponsored
									Gas:      21_000,
									GasPrice: suggestedGasPrice,
								}
								nonSponsoredTx := tests.SignTransaction(t, net.GetChainId(), nonSponsoredTxData, nonSponsoredAccount)

								require.NoError(t, client.SendTransaction(t.Context(), nonSponsoredTx), "failed to send non-sponsored transaction %v", i)
								txHashes = append(txHashes, nonSponsoredTx.Hash())
							}
							return txHashes
						},
					},
					"an sponsored transaction right at epoch change": {
						scenario: func(t *testing.T, net *tests.IntegrationTestNet) []common.Hash {
							// This test issues both a sponsored transaction and an epoch change transaction
							// asynchronously and attempts to have them included in the same block.
							//
							// Note: this test is somewhat flaky as it depends on fitting
							// two transactions in the same block, congested machines may make it fail.
							// Nevertheless, this test is necessary to ensure that internal transactions
							// nonces are correctly handled even in this edge case.

							// repeat until the sponsored transaction is included in an epoch change block
							var inSameBlock bool
							const retries = 10
							txHashes := []common.Hash{}
							for i := 0; !inSameBlock && i < retries; i++ {
								nonce, err := client.PendingNonceAt(t.Context(), sponsee.Address())
								require.NoError(t, err)
								txData := &types.LegacyTx{
									Nonce: nonce,
									To:    &common.Address{0x1}, // not a contract creation, contract creation cannot be sponsored
									Gas:   21_000,
								}
								tx := makeSponsorRequestTransaction(t, txData, net.GetChainId(), sponsee)

								err = client.SendTransaction(t.Context(), tx)
								require.NoError(t, err)

								// This test interacts directly with the drive contract to avoid
								// overheads of the usual testing tools which make it difficult
								// to schedule the previous sponsored transaction within the same block
								receipt, err := net.Apply(func(ops *bind.TransactOpts) (*types.Transaction, error) {
									return contract.AdvanceEpochs(ops, big.NewInt(1))
								})

								require.NoError(t, err, "failed to advance epoch")
								require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

								// wait for the transaction to be executed
								receipt, err = net.GetReceipt(tx.Hash())
								require.NoError(t, err)
								require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

								// the sponsored transaction is in an epoch change block iff there are
								// internal transactions at the beginning of the block transactions list
								block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
								require.NoError(t, err)
								inSameBlock = internaltx.IsInternal(block.Transactions()[0])
								if inSameBlock {
									txHashes = []common.Hash{block.Transactions()[0].Hash(), tx.Hash()}
									break
								}
							}

							require.True(t, inSameBlock, "could not include the sponsored transaction in an epoch change block after %d retries", retries)
							return txHashes
						},
					},
				}
				for name, test := range transactionsLoad {
					t.Run(name, func(t *testing.T) {

						// send all transactions asynchronously
						txHashes := test.scenario(t, net)

						// wait for all of them to be processed
						// note that this list of receipts does not contain the receipts for the
						// internal payment transactions
						receipts, err := net.GetReceipts(txHashes)
						require.NoError(t, err)

						for i, receipt := range receipts {

							// get the block with all the executed transactions.
							block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
							require.NoError(t, err)
							require.Greater(t, len(block.Transactions()), int(receipt.TransactionIndex),
								"block does not have enough transactions for tx %d", i,
							)
							tx := block.Transactions()[receipt.TransactionIndex]

							// verify that the transaction hash matches the one in the block
							require.Equal(t, tx.Hash(), receipt.TxHash,
								"receipt tx hash does not match block transaction hash for tx %d", i,
							)

							// get the receipts for all transactions in the block
							blockReceipts := []*types.Receipt{}
							err = client.Client().Call(&blockReceipts, "eth_getBlockReceipts", fmt.Sprintf("0x%x", block.Number().Uint64()))
							require.NoError(t, err)

							require.Greater(t, len(blockReceipts), int(receipt.TransactionIndex),
								"eth_getBlockReceipts returned too few receipts for tx %d", i,
							)
							require.Equal(t, len(block.Transactions()), len(blockReceipts),
								"eth_getBlockReceipts returned different number of receipts than block transactions for tx %d", i,
							)

							require.Equal(t, receipt, blockReceipts[receipt.TransactionIndex],
								"receipt does not match eth_getBlockReceipts for tx %d", i,
							)

							if subsidies.IsSponsorshipRequest(tx) {
								validateSponsoredTxInBlock(t, net, tx.Hash())
							}
						}
					})
				}
			})
		}
	}
}
