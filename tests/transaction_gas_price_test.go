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
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// Constant large enough to satisfied tx validation checks
// deduced from net rules minimum gas price
const enoughGasPrice = 150_000_000_000

func TestTransactionGasPrice(t *testing.T) {

	session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())
	t.Parallel()

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// use a fresh account to send transactions from
	account := MakeAccountWithBalance(t, session, big.NewInt(1e18))

	t.Run("Legacy transaction, effectivePrice is equal to requested price", func(t *testing.T) {

		// This test:
		// 1. Compute a valid gas price
		// 2. Sends a Legacy transaction with specified gas price
		// 3. Checks
		//    - effective gas price is greater or equal than basefee
		//    - subtracted balance equals gas price * gas used + value transferred
		//    - effective gas price equals the specified price

		balanceBefore := getBalance(t, client, account.Address())

		// 1: gas price
		var specifiedPrice int64 = enoughGasPrice

		// 2: make & execute transaction
		tx := makeLegacyTx(t, client, specifiedPrice, account, &common.Address{}, nil)

		receipt, err := session.Run(tx)
		require.NoError(t, err)
		require.Equal(t,
			receipt.Status,
			types.ReceiptStatusSuccessful,
			"transaction execution failed",
		)

		// 3: checks
		t.Run("Transaction gas price >= BaseFee  ", func(t *testing.T) {
			basefeeAfter := getBaseFeeAt(t, receipt.BlockNumber, client)
			require.GreaterOrEqual(t,
				receipt.EffectiveGasPrice.Int64(), basefeeAfter,
				"effective gas price is less than base fee")
		})

		t.Run("Account is charged gas price * gas used + balance transfer", func(t *testing.T) {
			costCharged := receipt.EffectiveGasPrice.Uint64()*receipt.GasUsed + tx.Value().Uint64()
			balance := getBalance(t, client, account.Address())
			balanceDifference := balanceBefore - balance
			require.Equal(t,
				balanceDifference,
				int64(costCharged),
				"changed wrong balance amount",
			)
		})

		t.Run("effective price equals specified price", func(t *testing.T) {
			require.Equal(t,
				specifiedPrice,
				receipt.EffectiveGasPrice.Int64(),
				"gas price does not match specified price",
			)
		})
	})

	t.Run("EIP-1559 transaction no tip", func(t *testing.T) {

		// This test:
		// 1. Compute a valid maximum gas price
		// 2. Sends an EIP-1559 transaction with specified gas price (max fee)
		// 3. Checks:
		//    - effective gas price is equal to the basefee
		//    - subtracted balance equals gas price * gas used + value transferred

		balanceBefore := getBalance(t, client, account.Address())

		// 1: gas price
		const maxGasPrice int64 = enoughGasPrice

		// 2: make & execute transaction
		tx := makeEip1559Transaction(t, client, maxGasPrice, 0, account, &common.Address{}, nil)

		receipt, err := session.Run(tx)
		require.NoError(t, err)
		require.Equal(t,
			receipt.Status,
			types.ReceiptStatusSuccessful,
			"transaction execution failed",
		)
		basefeeAfter := getBaseFeeAt(t, receipt.BlockNumber, client)

		// 3: checks
		t.Run("BaseFee <= EffectiveGasPrice <= maxGasPrice", func(t *testing.T) {

			require.LessOrEqual(t,
				basefeeAfter, receipt.EffectiveGasPrice.Int64(),
				"effective gas price is less than base fee")
			require.LessOrEqual(t,
				receipt.EffectiveGasPrice.Int64(), maxGasPrice,
				"effective gas price is greater than maximum requested price")
		})

		t.Run("Account is charged gas price * gas used + balance transfer", func(t *testing.T) {

			costCharged := receipt.EffectiveGasPrice.Uint64()*receipt.GasUsed + tx.Value().Uint64()
			balance := getBalance(t, client, account.Address())
			balanceDifference := balanceBefore - balance
			require.Equal(t,
				balanceDifference,
				int64(costCharged),
				"changed wrong balance amount",
			)
		})

		t.Run("effective price equals basefee", func(t *testing.T) {
			require.Equal(t,
				basefeeAfter, receipt.EffectiveGasPrice.Int64(),
				"gas price does not match specified price",
			)
		})

	})

	t.Run("EIP-1559 transaction with exact tip", func(t *testing.T) {

		// This test:
		// 1. Compute a valid maximum gas price
		// 2. Sends an EIP-1559 transaction with specified gas price (max fee) and some priority fee
		//    that can be charged whole. (basefee + tip <= maxGasPrice)
		// 3. checks:
		//    - effective gas price P; basefee <= P <= maxGasPrice
		//    - subtracted balance equals gas price * gas used + value transferred
		//    - effective gas price is equal to basefee + tip

		balanceBefore := getBalance(t, client, account.Address())

		// 1: gas price
		// transaction will be accepted with maxGasPrice == minGasPrice
		// but there will not be any room to charge the for the tip
		const maxGasPrice int64 = enoughGasPrice
		const tip = 17

		// 2: make & execute transaction
		tx := makeEip1559Transaction(t, client, maxGasPrice, tip, account, &common.Address{}, nil)

		receipt, err := session.Run(tx)
		require.NoError(t, err)
		require.Equal(t,
			receipt.Status,
			types.ReceiptStatusSuccessful,
			"transaction execution failed",
		)
		basefeeAfter := getBaseFeeAt(t, receipt.BlockNumber, client)

		// 3: checks
		t.Run("BaseFee <= EffectiveGasPrice <= maxGasPrice  ", func(t *testing.T) {

			require.LessOrEqual(t,
				basefeeAfter, receipt.EffectiveGasPrice.Int64(),
				"effective gas price is less than base fee")
			require.LessOrEqual(t,
				receipt.EffectiveGasPrice.Int64(), maxGasPrice,
				"effective gas price is greater than maximum requested price")
		})

		t.Run("Account is charged gas price * gas used + balance transfer", func(t *testing.T) {

			costCharged := receipt.EffectiveGasPrice.Uint64()*receipt.GasUsed + tx.Value().Uint64()
			balance := getBalance(t, client, account.Address())
			balanceDifference := balanceBefore - balance
			require.Equal(t,
				balanceDifference,
				int64(costCharged),
				"changed wrong balance amount",
			)
		})

		t.Run("effective price equals basefee + tip", func(t *testing.T) {
			require.Equal(t,
				basefeeAfter+tip, receipt.EffectiveGasPrice.Int64(),
				"gas price does not match basefee + tip",
			)
		})
	})

	t.Run("EIP-1559 transaction with excessive tip", func(t *testing.T) {

		// This test:
		// 1. Compute a valid maximum gas price
		// 2. Sends an EIP-1559 transaction with specified gas price (max fee) and some priority fee
		//    that cannot be charged whole. (tip > maxGasPrice - basefee)
		// 3. checks:
		//    - effective gas price P; basefee <= P <= maxGasPrice
		//    - subtracted balance equals gas price * gas used + value transferred
		//    - effective gas price is equal to maxGasPrice (consumed by tip)

		balanceBefore := getBalance(t, client, account.Address())

		// 1: gas price
		const maxGasPrice int64 = enoughGasPrice
		const tip = maxGasPrice // tip cannot be larger than max gas price

		// 2: make & execute transaction
		tx := makeEip1559Transaction(t, client, maxGasPrice, tip, account, &common.Address{}, nil)

		receipt, err := session.Run(tx)
		require.NoError(t, err)
		require.Equal(t,
			receipt.Status,
			types.ReceiptStatusSuccessful,
			"transaction execution failed",
		)
		basefeeAfter := getBaseFeeAt(t, receipt.BlockNumber, client)

		// 3: checks
		t.Run("BaseFee <= EffectiveGasPrice <= maxGasPrice  ", func(t *testing.T) {

			require.LessOrEqual(t,
				basefeeAfter, receipt.EffectiveGasPrice.Int64(),
				"effective gas price is less than base fee")
			require.LessOrEqual(t,
				receipt.EffectiveGasPrice.Int64(), maxGasPrice,
				"effective gas price is greater than maximum requested price")
		})

		t.Run("Account is charged gas price * gas used + balance transfer", func(t *testing.T) {

			costCharged := receipt.EffectiveGasPrice.Uint64()*receipt.GasUsed + tx.Value().Uint64()
			balance := getBalance(t, client, account.Address())
			balanceDifference := balanceBefore - balance
			require.Equal(t,
				balanceDifference,
				int64(costCharged),
				"changed wrong balance amount",
			)
		})

		t.Run("effective price equals max gas price", func(t *testing.T) {
			require.Equal(t,
				maxGasPrice, receipt.EffectiveGasPrice.Int64(),
				"gas price does not match expected price",
			)
		})
	})
}

func getBaseFeeAt(t *testing.T, blockNumber *big.Int, client *PooledEhtClient) int64 {
	t.Helper()
	block, err := client.BlockByNumber(t.Context(), blockNumber)
	require.NoError(t, err)
	basefee := block.BaseFee()
	return basefee.Int64()
}

func getBalance(t *testing.T, client *PooledEhtClient, account common.Address) int64 {
	t.Helper()
	balance, err := client.BalanceAt(t.Context(), account, nil)
	require.NoError(t, err)
	return balance.Int64()
}

// makeLegacyTx creates a legacy transaction from a CallMsg, filling in the nonce
// and gas limit.
func makeLegacyTx(t *testing.T,
	client *PooledEhtClient,
	gasPrice int64,
	sender *Account,
	to *common.Address,
	data []byte,
) *types.Transaction {
	t.Helper()

	nonce, err := client.NonceAt(t.Context(), sender.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", sender.Address())

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       to,
		Value:    big.NewInt(1),
		Gas:      1e6,
		GasPrice: big.NewInt(gasPrice),
		Data:     data,
	})

	chainId, err := client.ChainID(t.Context())
	require.NoError(t, err, "failed to get chain ID")

	signer := types.NewEIP155Signer(chainId)
	tx, err = types.SignTx(tx, signer, sender.PrivateKey)
	require.NoError(t, err, "failed to sign transaction")
	return tx
}

// makeLegacyTx creates a legacy transaction from a CallMsg, filling in the nonce
// and gas limit.
func makeEip1559Transaction(t *testing.T,
	client *PooledEhtClient,
	maxFeeCap int64,
	maxGasTip int64,
	sender *Account,
	to *common.Address,
	data []byte,
) *types.Transaction {
	t.Helper()

	nonce, err := client.NonceAt(t.Context(), sender.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", sender.Address())

	tx := types.NewTx(&types.DynamicFeeTx{
		Nonce:     nonce,
		To:        to,
		Value:     big.NewInt(1),
		Gas:       1e6,
		GasFeeCap: big.NewInt(maxFeeCap),
		GasTipCap: big.NewInt(maxGasTip),
		Data:      data,
	})

	chainId, err := client.ChainID(t.Context())
	require.NoError(t, err, "failed to get chain ID")

	signer := types.NewLondonSigner(chainId)
	tx, err = types.SignTx(tx, signer, sender.PrivateKey)
	require.NoError(t, err, "failed to sign transaction")
	return tx
}
