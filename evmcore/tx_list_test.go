// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package evmcore

import (
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// Tests that transactions can be added to strict lists and list contents and
// nonce boundaries are correctly maintained.
func TestStrictTxListAdd(t *testing.T) {
	// Generate a list of transactions to insert
	key, _ := crypto.GenerateKey()

	txs := make(types.Transactions, 1024)
	for i := 0; i < len(txs); i++ {
		txs[i] = transaction(uint64(i), 0, key)
	}
	// Insert the transactions in a random order
	list := newTxList(true)
	for _, v := range rand.Perm(len(txs)) {
		list.Add(txs[v], 10)
	}
	// Verify internal state
	if len(list.txs.items) != len(txs) {
		t.Errorf("transaction count mismatch: have %d, want %d", len(list.txs.items), len(txs))
	}
	for i, tx := range txs {
		if list.txs.items[tx.Nonce()] != tx {
			t.Errorf("item %d: transaction mismatch: have %v, want %v", i, list.txs.items[tx.Nonce()], tx)
		}
	}
}

func BenchmarkTxListAdd(t *testing.B) {
	// Generate a list of transactions to insert
	key, _ := crypto.GenerateKey()

	txs := make(types.Transactions, 100000)
	for i := 0; i < len(txs); i++ {
		txs[i] = transaction(uint64(i), 0, key)
	}
	// Insert the transactions in a random order
	list := newTxList(true)
	priceLimit := big.NewInt(int64(DefaultTxPoolConfig.MinimumTip))
	t.ResetTimer()
	for _, v := range rand.Perm(len(txs)) {
		list.Add(txs[v], DefaultTxPoolConfig.PriceBump)
		list.Filter(priceLimit, DefaultTxPoolConfig.MinimumTip)
	}
}

func TestTxList_Replacements(t *testing.T) {
	key, _ := crypto.GenerateKey()
	list := newTxList(false)

	tx := pricedTransaction(0, 0, big.NewInt(1000), key)
	inserted, replacedTx := list.Add(tx, DefaultTxPoolConfig.PriceBump)
	require.True(t, inserted, "transaction was not inserted")
	require.Nil(t, replacedTx, "replaced transaction should be nil")

	t.Run("transaction replacement with insufficient tipCap is rejected",
		func(t *testing.T) {
			tx := dynamicFeeTx(tx.Nonce(), 0, tx.GasFeeCap(), tx.GasTipCap(), key)
			replaced, replacedTx := list.Add(tx, DefaultTxPoolConfig.PriceBump)
			require.False(t, replaced, "transaction was replaced")
			require.Nil(t, replacedTx, "replaced transaction should be nil")
		})

	t.Run("transaction replacement with sufficient gasTip increment but insufficient gasFeeCap is rejected",
		func(t *testing.T) {
			newGasTip := new(big.Int).Add(tx.GasTipCap(), big.NewInt(100))
			tx := dynamicFeeTx(tx.Nonce(), 0, tx.GasFeeCap(), newGasTip, key)
			replaced, _ := list.Add(tx, DefaultTxPoolConfig.PriceBump)
			require.False(t, replaced, "transaction wasn't replaced")
		})

	t.Run("transaction replacement with sufficient gasTip increment is accepted",
		func(t *testing.T) {
			newGasTip := new(big.Int).Add(tx.GasTipCap(), big.NewInt(100))
			newGasFeeCap := new(big.Int).Set(newGasTip)
			tx := dynamicFeeTx(tx.Nonce(), 0, newGasFeeCap, newGasTip, key)
			replaced, replacedTx := list.Add(tx, DefaultTxPoolConfig.PriceBump)
			require.True(t, replaced, "transaction wasn't replaced")
			require.NotNil(t, replacedTx, "replaced transaction should't be nil")
		})
}
