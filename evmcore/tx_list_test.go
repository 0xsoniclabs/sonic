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

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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

func TestTxSortedMap_ContainsFunc_LocatesMatchingTransactions(t *testing.T) {
	const N = 10
	require := require.New(t)
	key, err := crypto.GenerateKey()
	require.NoError(err)

	m := newTxSortedMap()

	for i := range N {
		m.Put(transaction(uint64(i), 0, key))
	}

	for i := range N {
		require.True(m.containsFunc(func(tx *types.Transaction) bool {
			return tx.Nonce() == uint64(i)
		}))
	}

	require.False(m.containsFunc(func(tx *types.Transaction) bool {
		return tx.Nonce() == N+1
	}))
}

func TestTxList_Filter_WithSponsoredTransactions_RetainsCovered(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	checker := NewMocksubsidiesChecker(ctrl)

	key, err := crypto.GenerateKey()
	require.NoError(err)

	txs := []*types.Transaction{}
	for i := range 10 {
		tx := pricedTransaction(uint64(i), 0, common.Big0, key)
		require.True(subsidies.IsSponsorshipRequest(tx))
		txs = append(txs, tx)
	}

	list := newTxList(true)
	for _, tx := range txs {
		list.Add(tx, DefaultTxPoolConfig.PriceBump)
	}

	// Each sponsored transaction should be checked.
	for _, tx := range txs {
		checker.EXPECT().isSponsored(tx).Return(tx.Nonce()%2 == 0)
	}

	removed, _ := list.Filter(big.NewInt(1e18), 1_000_000, checker)

	// All non-sponsored transactions should be removed.
	require.Len(removed, 5)
	for _, tx := range removed {
		require.True(tx.Nonce()%2 == 1, "removed tx with nonce %d", tx.Nonce())
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
	minimumTip := big.NewInt(int64(DefaultTxPoolConfig.MinimumTip))
	t.ResetTimer()
	for _, v := range rand.Perm(len(txs)) {
		list.Add(txs[v], DefaultTxPoolConfig.PriceBump)
		list.Filter(minimumTip, DefaultTxPoolConfig.MinimumTip, nil)
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
