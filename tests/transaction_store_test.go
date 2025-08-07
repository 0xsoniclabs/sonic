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
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestTransactionStore_CanTransactionsBeRetrievedFromBlocksAfterRestart(t *testing.T) {

	// This test will execute a series of transactions.
	// After restarting the network, it will query the block where each transaction
	// was executed and check if the transaction is present in the block and the
	// values match, by comparing the hashes.

	net := StartIntegrationTestNet(t,
		IntegrationTestNetOptions{
			Upgrades: AsPointer(opera.GetAllegroUpgrades()),
		})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	chainId, err := client.ChainID(t.Context())
	require.NoError(t, err)

	sender := MakeAccountWithBalance(t, net, big.NewInt(1e18))
	senderAddress := sender.Address()

	// launch one transaction from each type
	txs := make([]*types.Transaction, 0)

	// Type 0: legacy transaction
	txs = append(txs, SignTransaction(t, chainId,
		&types.LegacyTx{
			Nonce:    0,
			To:       &senderAddress,
			Gas:      1e6,
			GasPrice: big.NewInt(500e9),
		},
		sender))

	// Type 1: AccessList transaction
	txs = append(txs, SignTransaction(t, chainId,
		&types.AccessListTx{
			Nonce:    1,
			To:       &senderAddress,
			Gas:      1e6,
			GasPrice: big.NewInt(500e9),
			AccessList: types.AccessList{
				{Address: senderAddress, StorageKeys: []common.Hash{{0x01}}},
			},
		},
		sender))

	// Type 2: DynamicFee transaction
	txs = append(txs, SignTransaction(t, chainId,
		&types.DynamicFeeTx{
			Nonce:     2,
			To:        &senderAddress,
			Gas:       1e6,
			GasFeeCap: big.NewInt(500e9),
			GasTipCap: big.NewInt(500e9),
		},
		sender))

	// Type 3: Blob transaction
	txs = append(txs, SignTransaction(t, chainId,
		&types.BlobTx{
			Nonce:     3,
			Gas:       1e6,
			GasFeeCap: uint256.NewInt(500e9),
			GasTipCap: uint256.NewInt(500e9),
		},
		sender))

	// Type 4: SetCode transaction
	authority := NewAccount()
	authorization, err := types.SignSetCode(authority.PrivateKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(chainId),
		Address: common.Address{42},
		Nonce:   5,
	})
	require.NoError(t, err, "failed to sign SetCode authorization")
	txs = append(txs, SignTransaction(t, chainId,
		&types.SetCodeTx{
			Nonce:     4,
			To:        senderAddress,
			Gas:       1e6,
			GasFeeCap: uint256.NewInt(500e9),
			GasTipCap: uint256.NewInt(500e9),
			AuthList:  []types.SetCodeAuthorization{authorization},
		},
		sender))

	for _, tx := range txs {
		err := client.SendTransaction(t.Context(), tx)
		require.NoError(t, err)
	}

	executedIn := make(map[*types.Transaction]*big.Int, len(txs))
	for i, tx := range txs {
		receipt, err := net.GetReceipt(tx.Hash())
		require.NoError(t, err, "failed to get receipt for tx%d", i)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "tx%d failed", i)
		require.NotNil(t, receipt.BlockNumber)
		executedIn[tx] = receipt.BlockNumber
	}

	err = net.Restart()
	require.NoError(t, err, "failed to restart network; %v", err)

	// query last block, retrieve executed transactions
	client, err = net.GetClient()
	require.NoError(t, err)

	for tx, blockNumber := range executedIn {
		block, err := client.BlockByNumber(t.Context(), blockNumber)
		require.NoError(t, err, "failed to get block %v", blockNumber)

		require.True(t,
			slices.ContainsFunc(block.Transactions(), func(received *types.Transaction) bool {
				return received.Hash() == tx.Hash()
			}))
	}
}
