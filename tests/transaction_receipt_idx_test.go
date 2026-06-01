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
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/contract/driverauth100"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/contracts/driverauth"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestReceipt_InternalTransactionsHaveZeroEffectiveGasPrice(t *testing.T) {
	net := StartIntegrationTestNet(t)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// Run one transaction to not interfere with still pending delayed genesis.
	receipt, err := net.EndowAccount(common.Address{}, big.NewInt(1e18))
	require.NoError(t, err)
	before := receipt.BlockNumber.Uint64()

	// Advance one epoch to trigger internal sealing transactions.
	net.AdvanceEpoch(t, 1)

	after, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	// Find a block containing internal sealing transactions (sender = zero address).
	var sealingBlock *types.Block
	for cur := before; cur <= after; cur++ {
		block, err := client.BlockByNumber(t.Context(), big.NewInt(int64(cur)))
		require.NoError(t, err)
		if len(block.Transactions()) == 0 {
			continue
		}
		sender, err := getSenderOfTransaction(client, block.Transactions()[0].Hash())
		require.NoError(t, err)
		if sender == (common.Address{}) {
			sealingBlock = block
			break
		}
	}
	require.NotNil(t, sealingBlock, "No block found with internal transactions")

	// For every internal transaction, verify that the RPC-reported effectiveGasPrice is zero.
	foundInternal := false
	for _, blockTx := range sealingBlock.Transactions() {
		sender, err := getSenderOfTransaction(client, blockTx.Hash())
		require.NoError(t, err)
		if sender != (common.Address{}) {
			continue
		}
		foundInternal = true
		txReceipt, err := client.TransactionReceipt(t.Context(), blockTx.Hash())
		require.NoError(t, err)
		require.NotNil(t, txReceipt, "receipt must exist for internal tx")
		require.NotNil(t, txReceipt.EffectiveGasPrice, "effectiveGasPrice must be set")
		require.Equal(t, int64(0), txReceipt.EffectiveGasPrice.Int64(),
			"effectiveGasPrice must be zero for internal transactions")
	}
	require.True(t, foundInternal, "No internal transactions found in sealing block")
}

func TestReceipt_InternalTransactionsDoNotChangeReceiptIndex(t *testing.T) {
	for hardfork, upgrades := range opera.GetAllHardForksInOrder() {
		t.Run(hardfork, func(t *testing.T) {
			testInternalTransactionsDoNotChangeReceiptIndex(t, upgrades)
		})
	}
}

func testInternalTransactionsDoNotChangeReceiptIndex(t *testing.T, upgrades opera.Upgrades) {
	net := StartIntegrationTestNetWithJsonGenesis(t, IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// Run one transaction to not interfere with still pending delayed genesis.
	receipt, err := net.EndowAccount(common.Address{}, big.NewInt(1e18))
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Send transaction instructing the network to advance one epoch.
	contract, err := driverauth100.NewContract(driverauth.ContractAddress, client)
	require.NoError(t, err)

	// Create a separate user account to avoid nonce conflicts with the account
	// used for epoch advancement.
	userAccount := MakeAccountWithBalance(t, net, big.NewInt(1e18))

	// Retry until we find a sealing block that contains both internal and user
	// transactions. The timing of when the sealing block is created relative to
	// when user transactions arrive is inherently racy, so we retry up to N times.
	var sealingBlock *types.Block
	const maxRetries = 10
	for attempt := 0; attempt < maxRetries && sealingBlock == nil; attempt++ {
		// Send a user transaction (non-blocking), then immediately advance the epoch.
		// This gives the user transaction a chance to be included in the epoch-sealing block.
		userTx := CreateTransaction(t, net, &types.LegacyTx{
			To:  &common.Address{0x42},
			Gas: 21_000,
		}, userAccount)
		err = client.SendTransaction(t.Context(), userTx)
		require.NoError(t, err)

		// This test interacts directly with the drive contract to avoid
		// overheads of the usual testing tools which make it difficult
		// to schedule the previous sponsored transaction within the same block
		receipt, err := net.Apply(func(ops *bind.TransactOpts) (*types.Transaction, error) {
			return contract.AdvanceEpochs(ops, big.NewInt(1))
		})
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status,
			"AdvanceEpochs transaction failed")

		// Wait for the user transaction to be executed.
		userReceipt, err := net.GetReceipt(userTx.Hash())
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, userReceipt.Status)

		// Check if the user transaction landed in the sealing block.
		block, err := client.BlockByNumber(t.Context(), userReceipt.BlockNumber)
		require.NoError(t, err)

		if len(block.Transactions()) > 2 {
			sender, err := getSenderOfTransaction(client, block.Transactions()[0].Hash())
			require.NoError(t, err)
			if sender == (common.Address{}) {
				sealingBlock = block
			}
		}
	}
	require.NotNil(t, sealingBlock, "No block found with internal transactions")

	// There should be at least 2 internal transactions + one extra transaction.
	// Internal transactions are send by the zero address.
	require.Greater(t, len(sealingBlock.Transactions()), 2)

	transactions := sealingBlock.Transactions()
	sender, err := getSenderOfTransaction(client, transactions[0].Hash())
	require.NoError(t, err)
	require.Equal(t, common.Address{}, sender)

	sender, err = getSenderOfTransaction(client, transactions[1].Hash())
	require.NoError(t, err)
	require.Equal(t, common.Address{}, sender)

	sender, err = getSenderOfTransaction(client, transactions[2].Hash())
	require.NoError(t, err)
	require.NotEqual(t, common.Address{}, sender)

	// Check that the index numbers of the receipts match the transaction index.
	for i, tx := range transactions {
		receipt, err := client.TransactionReceipt(t.Context(), tx.Hash())
		require.NoError(t, err)
		require.Equal(t, uint(i), receipt.TransactionIndex,
			"Receipt index does not match transaction index for tx %d", i,
		)
	}
}

func getSenderOfTransaction(
	client *PooledEhtClient,
	txHash common.Hash,
) (common.Address, error) {
	details := struct {
		From common.Address
	}{}
	err := client.Client().Call(&details, "eth_getTransactionByHash", txHash)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get transaction details: %w", err)
	}
	return details.From, nil
}

func TestReceipt_SkippedTransactionsDoNotChangeReceiptIndexOrCumulativeGasUsed(t *testing.T) {
	for hardfork, upgrades := range opera.GetAllHardForksInOrder() {
		t.Run(hardfork, func(t *testing.T) {
			testSkippedTransactionsDoNotChangeReceiptIndexOrCumulativeGasUsed(t, upgrades)
		})
	}
}

func testSkippedTransactionsDoNotChangeReceiptIndexOrCumulativeGasUsed(t *testing.T, upgrades opera.Upgrades) {
	net := StartIntegrationTestNetWithJsonGenesis(t, IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	chainId := net.GetChainId()
	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(t, err)
	sender := MakeAccountWithBalance(t, net, big.NewInt(1e18))
	senderSkipped := MakeAccountWithBalance(t, net, big.NewInt(1e18))

	numSimpleTxs := 10
	// Create simple transactions
	transactions := make([]*types.Transaction, numSimpleTxs)
	for nonce := range numSimpleTxs {
		txData := &types.LegacyTx{
			Nonce:    uint64(nonce),
			Gas:      100000,
			GasPrice: gasPrice,
			To:       &common.Address{0x42},
			Value:    big.NewInt(1),
		}

		tx := SignTransaction(t, chainId, txData, sender)
		transactions[nonce] = tx
	}

	// Create skipped transaction
	initCode := make([]byte, 50000)
	if upgrades.Brio {
		initCode = make([]byte, 50000*2)
	}
	txData := &types.LegacyTx{
		Nonce:    uint64(0),
		Gas:      10000000,
		GasPrice: gasPrice,
		To:       nil, // address 0x00 for contract creation
		Value:    big.NewInt(0),
		Data:     initCode,
	}
	skippedTx := SignTransaction(t, chainId, txData, senderSkipped)

	// Run one transaction to not interfere with any still pending transactions.
	receipt, err := net.EndowAccount(common.Address{}, big.NewInt(1e18))
	require.NoError(t, err)
	before := receipt.BlockNumber.Uint64()

	// Send first half of simple transactions
	for i := range numSimpleTxs / 2 {
		err = client.SendTransaction(t.Context(), transactions[i])
		require.NoError(t, err)
	}

	// Send skipped transaction
	_, err = net.ForceEmit(t.Context(), skippedTx)
	require.NoError(t, err)

	// Send second half of simple transactions
	for i := range numSimpleTxs / 2 {
		err = client.SendTransaction(t.Context(), transactions[numSimpleTxs/2+i])
		require.NoError(t, err)
	}

	// Wait for receipts of transactions
	after := before
	for _, transaction := range transactions {
		receipt, err := net.GetReceipt(transaction.Hash())
		require.NoError(t, err)
		require.NotNil(t, receipt)
		if receipt.BlockNumber.Uint64() > after {
			after = receipt.BlockNumber.Uint64()
		}
	}

	// Make sure the skipped transaction was included in the block by checking the balance of the sender.
	balanceBefore, err := client.BalanceAt(t.Context(), senderSkipped.Address(), big.NewInt(int64(before)))
	require.NoError(t, err)
	balanceAfter, err := client.BalanceAt(t.Context(), senderSkipped.Address(), big.NewInt(int64(after)))
	require.NoError(t, err)

	if !upgrades.Allegro {
		require.Greater(t, balanceBefore.Uint64(), balanceAfter.Uint64(),
			"Balance should have decreased",
		)
	} else {
		require.Equal(t, balanceBefore.Uint64(), balanceAfter.Uint64(),
			"Balance should remain unchanged",
		)
	}

	require.Greater(t, after, before, "Block number should have increased")

	// Get the receipts of all blocks between before and after
	for number := before + 1; number <= after; number++ {
		block, err := client.BlockByNumber(t.Context(), big.NewInt(int64(number)))
		require.NoError(t, err)

		cumulativeGas := uint64(0)
		for idx, tx := range block.Transactions() {
			receipt, err := client.TransactionReceipt(t.Context(), tx.Hash())
			require.NoError(t, err)
			cumulativeGas += receipt.GasUsed

			// Check that the receipt index is equal to the transaction index
			require.Equal(t, uint(idx), receipt.TransactionIndex,
				"Receipt index does not match transaction index for tx %d", idx,
			)

			// Check that sum of gas used by each transaction matches the cumulative gas used
			require.Equal(t, cumulativeGas, receipt.CumulativeGasUsed,
				"Cumulative gas used does not match for tx %d", idx,
			)
		}
	}
}
