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
	"math"
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestGasSubsidies_CumulativeGasUsageOfSponsoredTransactions(t *testing.T) {
	upgrades := opera.GetSonicUpgrades()
	upgrades.GasSubsidies = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// -------------------------------------------------------------------------

	sponsor := tests.NewAccount()
	sponsee := tests.NewAccount()
	receiver := tests.NewAccount()

	// Before the sponsorship is set up, a transaction from the sponsee
	// to the receiver should fail due to lack of funds.
	chainId := net.GetChainId()
	receiverAddress := receiver.Address()
	signer := types.LatestSignerForChainID(chainId)

	// --- deposit sponsorship funds ---

	donation := big.NewInt(1e16)
	createRegistryWithDonation(t, client, net, sponsor, sponsee, receiver, donation)

	burnedBefore, err := client.BalanceAt(t.Context(), common.Address{}, nil)
	require.NoError(t, err)

	// --- submit a sponsored transaction ---
	numTransactions := 5
	transactionCost := 21000
	transactions := make([]*types.Transaction, 0, numTransactions)
	for i := range numTransactions {
		tx, err := types.SignNewTx(sponsee.PrivateKey, signer, &types.LegacyTx{
			To:       &receiverAddress,
			Nonce:    uint64(i),
			Gas:      uint64(transactionCost),
			GasPrice: big.NewInt(0),
		})
		require.NoError(t, err)
		transactions = append(transactions, tx)
		require.NoError(t, client.SendTransaction(t.Context(), tx))
	}

	cumulativeGasCost := uint64(0)

	var receipt *types.Receipt
	firstBlockNumber := uint64(math.MaxUint64)
	lastBlockNumber := uint64(0)
	for _, tx := range transactions {
		// Wait for the sponsored transaction to be executed.
		receipt, err = net.GetReceipt(tx.Hash())
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

		if receipt.BlockNumber.Uint64() < firstBlockNumber {
			firstBlockNumber = receipt.BlockNumber.Uint64()
		}
		if receipt.BlockNumber.Uint64() > lastBlockNumber {
			lastBlockNumber = receipt.BlockNumber.Uint64()
		}

		block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
		require.NoError(t, err)
		index := slices.IndexFunc(block.Transactions(), func(t *types.Transaction) bool {
			return t.Hash() == tx.Hash()
		})
		require.Greater(t, index, -1, "transaction not found in block")

		// Check that the next transaction in the block is the payment transaction.
		require.Less(t, index+1, len(block.Transactions()), "no payment transaction found")
		payment := block.Transactions()[index+1]
		paymentReceipt, err := net.GetReceipt(payment.Hash())
		require.NoError(t, err)

		// Accumulate the gas used by the sponsored transaction.
		cumulativeGasCost += receipt.GasUsed + paymentReceipt.GasUsed
	}

	for i := firstBlockNumber; i <= lastBlockNumber; i++ {
		cumulativeGas := uint64(0)
		block, err := client.BlockByNumber(t.Context(), big.NewInt(int64(i)))
		require.NoError(t, err)
		for _, tx := range block.Transactions() {
			receipt, err := net.GetReceipt(tx.Hash())
			require.NoError(t, err)
			cumulativeGas += receipt.GasUsed
			require.Equal(t, cumulativeGas, receipt.CumulativeGasUsed,
				"cumulative gas used in the block should equal the sum of the gas used by the sponsored transactions")
		}
	}

	header, err := client.HeaderByHash(t.Context(), receipt.BlockHash)
	require.NoError(t, err)

	// the difference in the sponsorship funds should have been burned
	burnedAfter, err := client.BalanceAt(t.Context(), common.Address{}, nil)
	require.NoError(t, err)

	require.Greater(t,
		burnedAfter.Uint64(),
		burnedBefore.Uint64()+cumulativeGasCost*header.BaseFee.Uint64(),
		"the burned amount should equal the cost of the sponsored transactions",
	)
}
