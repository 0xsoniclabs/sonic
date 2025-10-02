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
	"math/big"
	"testing"

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

func TestGasSubsidies_InternalTransaction_HaveConsistentNonces(t *testing.T) {
	// internal transactions shall respect the incrementing nonce order without gaps

	upgrades := opera.GetAllegroUpgrades()
	upgrades.GasSubsidies = true
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	sponsoredSender := tests.NewAccount()
	totalValueForSponsorships := new(big.Int).Mul(
		big.NewInt(1e18),
		big.NewInt(100),
	)

	// fund any transaction sent by the sponsored sender
	Fund(t, net, sponsoredSender.Address(), totalValueForSponsorships)

	// mount the driver auth contract to be able to advance epochs
	contract, err := driverauth100.NewContract(driverauth.ContractAddress, client)
	require.NoError(t, err, "failed to create contract instance")

	tests := map[string]struct {
		scenario func(t *testing.T, net *tests.IntegrationTestNet)
	}{
		"single sponsored transaction": {
			scenario: func(t *testing.T, net *tests.IntegrationTestNet) {
				nonce, err := client.PendingNonceAt(t.Context(), sponsoredSender.Address())
				require.NoError(t, err)
				txData := &types.LegacyTx{
					Nonce: nonce,
					To:    &common.Address{0x1}, // not a contract creation, contract creation cannot be sponsored
					Gas:   21_000,
				}
				sponsoredTx := makeSponsorRequestTransaction(t, txData, net.GetChainId(), sponsoredSender)
				receipt, err := net.Run(sponsoredTx)
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

				validateSponsoredTxInBlock(t, net, sponsoredTx.Hash())
			},
		},
		"multiple sponsored transactions are executed within the same block": {
			scenario: func(t *testing.T, net *tests.IntegrationTestNet) {
				// This scenario issues multiple sponsored transactions asynchronously
				// to facilitate their inclusion in the same block.

				const numTxs = 10
				txHashes := make([]common.Hash, 0, numTxs)

				StartingNonce, err := client.PendingNonceAt(t.Context(), sponsoredSender.Address())
				require.NoError(t, err)

				for i := 0; i < numTxs; i++ {

					txData := &types.LegacyTx{
						Nonce: StartingNonce + uint64(i),
						To:    &common.Address{0x1}, // not a contract creation, contract creation cannot be sponsored
						Gas:   21_000,
					}
					sponsoredTx := makeSponsorRequestTransaction(t, txData, net.GetChainId(), sponsoredSender)

					err = client.SendTransaction(t.Context(), sponsoredTx)
					require.NoError(t, err)
					txHashes = append(txHashes, sponsoredTx.Hash())
				}

				// wait for the last one to be executed
				receipt, err := net.GetReceipt(txHashes[len(txHashes)-1])
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

				for _, txHash := range txHashes {
					validateSponsoredTxInBlock(t, net, txHash)
				}
			},
		},
		"an sponsored transaction right at epoch change": {
			scenario: func(t *testing.T, net *tests.IntegrationTestNet) {
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
				for i := 0; !inSameBlock && i < retries; i++ {
					nonce, err := client.PendingNonceAt(t.Context(), sponsoredSender.Address())
					require.NoError(t, err)
					txData := &types.LegacyTx{
						Nonce: nonce,
						To:    &common.Address{0x1}, // not a contract creation, contract creation cannot be sponsored
						Gas:   21_000,
					}
					tx := makeSponsorRequestTransaction(t, txData, net.GetChainId(), sponsoredSender)
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
				}

				require.True(t, inSameBlock, "could not include the sponsored transaction in an epoch change block after %d retries", retries)
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			startBefore, err := client.BlockByNumber(t.Context(), nil)
			require.NoError(t, err)
			internalTxNonce, err := client.PendingNonceAt(t.Context(), common.Address{})
			require.NoError(t, err)

			test.scenario(t, net)

			blockAfter, err := client.BlockByNumber(t.Context(), nil)
			require.NoError(t, err)

			// after each scenario, validate all new internal transactions nonces
			// created during the scenario execution

			for i := startBefore.NumberU64() + 1; i <= blockAfter.NumberU64(); i++ {
				block, err := client.BlockByNumber(t.Context(), big.NewInt(int64(i)))
				require.NoError(t, err)

				for _, tx := range block.Transactions() {
					if !internaltx.IsInternal(tx) {
						continue
					}
					require.Equalf(t, internalTxNonce, tx.Nonce(),
						"internal transaction nonce mismatch")
					internalTxNonce++
				}
			}
		})
	}
}
