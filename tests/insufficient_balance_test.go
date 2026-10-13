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
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestInsufficientBalance_TransactionsNotCoveringTheirValueAreSkippedStartingWithCanto(t *testing.T) {
	tests := map[string]struct {
		to *common.Address
		// Signs a new transaction with the sender's current nonce for every
		// attempt instead of proposing the same signed transaction again.
		resignWithCurrentNonce bool
		// Before Canto, the transaction is charged as failed. Contract
		// creations fail before their nonce is incremented, so the same
		// signed transaction can be included and charged again.
		includedBeforeCanto []bool
	}{
		"transfer": {
			to:                  &common.Address{0x42},
			includedBeforeCanto: []bool{true, false},
		},
		"transfer re-signed with the current nonce": {
			to:                     &common.Address{0x42},
			resignWithCurrentNonce: true,
			includedBeforeCanto:    []bool{true, true},
		},
		"contract creation": {
			to:                  nil,
			includedBeforeCanto: []bool{true, true},
		},
	}

	for upgradeName, upgrades := range opera.GetAllHardForksInOrder() {
		t.Run(upgradeName, func(t *testing.T) {
			session := getIntegrationTestNetSession(t, upgrades)
			t.Parallel()

			for name, test := range tests {
				t.Run(name, func(t *testing.T) {
					require := require.New(t)
					client, err := session.GetClient()
					require.NoError(err)
					defer client.Close()

					balance := big.NewInt(1e18)
					accounts := MakeAccountsWithBalance(t, session, 2, balance)
					sender, marker := accounts[0], accounts[1]

					// The balance covers the fee, but not the fee and the value.
					// Proposing the transaction directly bypasses the pool,
					// which would reject it.
					makeTxWithNonce := func(nonce uint64) *types.Transaction {
						return CreateTransaction(t, session, &types.LegacyTx{
							Nonce: nonce,
							To:    test.to,
							Gas:   100_000,
							Value: balance,
						}, sender)
					}
					tx := makeTxWithNonce(0)

					wantBalance := new(big.Int).Set(balance)
					wantNonce := uint64(0)
					for _, includedBeforeCanto := range test.includedBeforeCanto {
						if test.resignWithCurrentNonce {
							tx = makeTxWithNonce(wantNonce)
						}
						// The marker is proposed in the same event to know the
						// block the transaction is processed in.
						markerTx := CreateTransaction(t, session, &types.LegacyTx{
							To: &common.Address{},
						}, marker)
						_, err := session.ForceEmitAll(t.Context(), []*types.Transaction{tx, markerTx})
						require.NoError(err)
						markerReceipt, err := session.GetReceipt(markerTx.Hash())
						require.NoError(err)
						require.Equal(types.ReceiptStatusSuccessful, markerReceipt.Status)
						block := markerReceipt.BlockNumber

						receipts, err := client.BlockReceipts(t.Context(),
							rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(block.Int64())))
						require.NoError(err)
						var receipt *types.Receipt
						for _, r := range receipts {
							if r.TxHash == tx.Hash() {
								receipt = r
							}
						}

						if includedBeforeCanto && !upgrades.Canto {
							require.NotNil(receipt, "transaction should be included")
							require.Equal(types.ReceiptStatusFailed, receipt.Status)
							fee := new(big.Int).Mul(
								new(big.Int).SetUint64(receipt.GasUsed),
								receipt.EffectiveGasPrice,
							)
							wantBalance.Sub(wantBalance, fee)
							if test.to != nil {
								wantNonce++
							}
						} else {
							require.Nil(receipt, "transaction should be skipped")
						}

						gotBalance, err := client.BalanceAt(t.Context(), sender.Address(), block)
						require.NoError(err)
						require.Equal(wantBalance, gotBalance)
						gotNonce, err := client.NonceAt(t.Context(), sender.Address(), block)
						require.NoError(err)
						require.Equal(wantNonce, gotNonce)
					}
				})
			}
		})
	}
}
