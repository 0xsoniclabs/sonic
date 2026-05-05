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

package gasfee

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestTxFeeAccounting_TransactionFeesAreChargedAsStatedInReceipts(t *testing.T) {
	upgrades := opera.GetBrioUpgrades()
	upgrades.GasSubsidies = true
	upgrades.TransactionBundles = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	// run a mix of transactions
	txs := createTransactionMix(t, net, 100)
	_, err := net.SendAll(txs)
	require.NoError(t, err)
	waitForTransactionMixToBeComplete(t, net, txs)

	// Test for each transaction that fees have been deducted correctly. This
	// check exploits the fact that all transactions in the mix use independent
	// accounts and that there are no balances increased on those accounts.
	blocks, err := net.GetBlocks(t.Context())
	require.NoError(t, err)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())
	for _, b := range blocks {

		// Collect all balance changes of senders in this block.
		expectedDiffs := map[common.Address]*big.Int{}
		for _, tx := range b.Transactions() {

			receipt, err := net.GetReceipt(tx.Hash())
			require.NoError(t, err)

			expectedCharges := new(big.Int).Mul(
				receipt.EffectiveGasPrice,
				big.NewInt(int64(receipt.GasUsed)),
			)

			// The expected change is the sum of the fee charges and the
			// transferred value.
			expectedChange := new(big.Int).Add(expectedCharges, tx.Value())

			// Internal transactions should have zero changes.
			if internaltx.IsInternal(tx) {
				require.Equal(t, big.NewInt(0), expectedChange)
				continue
			}

			// Keep track of the expected change in the balance.
			sender, err := types.Sender(signer, tx)
			require.NoError(t, err)
			if old, found := expectedDiffs[sender]; found {
				expectedDiffs[sender] = new(big.Int).Add(old, expectedChange)
			} else {
				expectedDiffs[sender] = expectedChange
			}
		}

		// Check the aggregated changes of the full block.
		for sender, expectedChange := range expectedDiffs {
			before := new(big.Int).Sub(b.Number(), big.NewInt(1))
			balanceBefore, err := client.BalanceAt(t.Context(), sender, before)
			require.NoError(t, err)

			balanceAfter, err := client.BalanceAt(t.Context(), sender, b.Number())
			require.NoError(t, err)

			seenDiff := new(big.Int).Sub(balanceBefore, balanceAfter)
			require.True(t, expectedChange.Cmp(seenDiff) == 0,
				"expected: %v, seen: %v for account %v", expectedChange, seenDiff, sender,
			)
		}
	}
}
