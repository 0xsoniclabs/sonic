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
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
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
	checker := func(tx *types.Transaction) bool {
		return tx.Nonce()%2 == 0
	}

	removed, _ := list.Filter(big.NewInt(1e18), 1_000_000, checker, nil)

	// All non-sponsored transactions should be removed.
	require.Len(removed, 5)
	for _, tx := range removed {
		require.True(tx.Nonce()%2 == 1, "removed tx with nonce %d", tx.Nonce())
	}
}

func TestTxList_Filter_WithBundleTransactions_RetainsPending(t *testing.T) {
	require := require.New(t)
	key, err := crypto.GenerateKey()
	require.NoError(err)

	txs := make([]*types.Transaction, 10)
	for i := range txs {
		txs[i] = bundleTx(uint64(i), key)
	}

	list := newTxList(true)
	for _, tx := range txs {
		list.Add(tx, DefaultTxPoolConfig.PriceBump)
	}

	// Each bundle transaction should be checked.
	callCount := 0
	checker := func(tx *types.Transaction) bundlePoolStatus {
		callCount++
		return bundlePending
	}

	costLimit := big.NewInt(1e18)
	removed, invalidated := list.Filter(costLimit, 1_000_000, nil, checker)
	require.Equal(10, callCount, "unexpected number of calls to checker")
	require.Len(removed, 0)
	require.Len(invalidated, 0)
}

func TestTxList_Filter_WithBundleTransactions_RemovesRejected(t *testing.T) {
	require := require.New(t)
	key, err := crypto.GenerateKey()
	require.NoError(err)

	txs := make([]*types.Transaction, 10)
	for i := range txs {
		txs[i] = bundleTx(uint64(i), key)
	}

	list := newTxList(false)
	for _, tx := range txs {
		list.Add(tx, DefaultTxPoolConfig.PriceBump)
	}

	// Each bundle transaction should be checked.
	callCount := 0
	checker := func(tx *types.Transaction) bundlePoolStatus {
		callCount++
		if tx.Nonce()%2 == 0 {
			return bundlePending
		}
		return bundleRejected
	}

	costLimit := big.NewInt(1e18)
	removed, invalidated := list.Filter(costLimit, 1_000_000, nil, checker)
	require.Equal(10, callCount, "unexpected number of calls to checker")
	require.Len(removed, 5, "odd nonce transactions should be removed")
	require.Len(invalidated, 0, "list is not strict, so no transactions should be invalidated")
}

func TestTxList_Strict_Filter_WithBundleTransactions_InvalidatesTemporarilyBlocked(t *testing.T) {
	require := require.New(t)
	key, err := crypto.GenerateKey()
	require.NoError(err)

	txs := make([]*types.Transaction, 10)
	for i := range txs {
		txs[i] = bundleTx(uint64(i), key)
	}

	list := newTxList(true)
	for _, tx := range txs {
		list.Add(tx, DefaultTxPoolConfig.PriceBump)
	}

	// Each bundle transaction should be checked.
	checker := func(tx *types.Transaction) bundlePoolStatus {
		if tx.Nonce() < 5 {
			return bundlePending
		}
		return bundleQueued
	}

	costLimit := big.NewInt(1e18)
	removed, invalidated := list.Filter(costLimit, 1_000_000, nil, checker)
	require.Len(removed, 0, "no transactions should be removed, just invalidated")
	require.Len(invalidated, 5, "list is strict, so temporarily blocked transactions should be invalidated")
}

func TestTxList_Strict_Filter_WithBundleTransactions_InvalidatesGappedNonces_AfterBundleIsInvalidated(t *testing.T) {
	require := require.New(t)
	key, err := crypto.GenerateKey()
	require.NoError(err)

	txs := make([]*types.Transaction, 10)
	txs[0] = bundleTx(uint64(0), key)
	for i := 1; i < len(txs); i++ {
		txs[i] = transaction(uint64(i), 0, key)
	}

	list := newTxList(true)
	for _, tx := range txs {
		list.Add(tx, DefaultTxPoolConfig.PriceBump)
	}

	// Each bundle transaction should be checked.
	checker := func(tx *types.Transaction) bundlePoolStatus {
		return bundleQueued
	}

	costLimit := big.NewInt(1e18)
	removed, invalidated := list.Filter(costLimit, 1_000_000, nil, checker)
	require.Len(removed, 0, "no transactions should be removed, just invalidated")
	require.Len(invalidated, 10, "invalid bundle with nonce 0 invalidates all transactions with nonce > 0")
}

func TestTxList_Filter_RemovesAndInvalidatesTransactions(t *testing.T) {

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	gaslimit := uint64(1_000_000)

	allPending := func(tx *types.Transaction) bundlePoolStatus {
		return bundlePending
	}
	allRejected := func(tx *types.Transaction) bundlePoolStatus {
		return bundleRejected
	}
	allQueued := func(tx *types.Transaction) bundlePoolStatus {
		return bundleQueued
	}
	invalidateFromNonce := func(nonce uint64) func(tx *types.Transaction) bundlePoolStatus {
		return func(tx *types.Transaction) bundlePoolStatus {
			if tx.Nonce() >= nonce {
				return bundleQueued
			}
			return bundlePending
		}
	}
	rejectFromNonce := func(nonce uint64) func(tx *types.Transaction) bundlePoolStatus {
		return func(tx *types.Transaction) bundlePoolStatus {
			if tx.Nonce() >= nonce {
				return bundleRejected
			}
			return bundlePending
		}
	}

	tests := map[string]struct {
		strict          bool
		txs             types.Transactions
		checker         func(*types.Transaction) bundlePoolStatus
		wantRemoved     []uint64
		wantInvalidated []uint64
	}{
		"all pending bundles are keep": {
			txs: []*types.Transaction{
				bundleTx(0, key),
				transaction(1, gaslimit, key),
				bundleTx(2, key),
				transaction(3, gaslimit, key),
				bundleTx(4, key),
			},
			checker:         allPending,
			wantRemoved:     []uint64{},
			wantInvalidated: []uint64{},
		},
		"all pending bundles are keep, strict mode": {
			strict: true,
			txs: []*types.Transaction{
				bundleTx(0, key),
				transaction(1, gaslimit, key),
				bundleTx(2, key),
				transaction(3, gaslimit, key),
				bundleTx(4, key),
			},
			checker:         allPending,
			wantRemoved:     []uint64{},
			wantInvalidated: []uint64{},
		},
		"all queued bundles are kept, non-strict mode": {
			txs: []*types.Transaction{
				bundleTx(0, key),
				transaction(1, gaslimit, key),
				bundleTx(2, key),
				transaction(3, gaslimit, key),
				bundleTx(4, key),
			},
			checker:         allQueued,
			wantRemoved:     []uint64{},
			wantInvalidated: []uint64{},
		},
		"all queued bundles are invalidated, strict mode": {
			strict: true,
			txs: []*types.Transaction{
				bundleTx(0, key),
				bundleTx(1, key),
				bundleTx(2, key),
			},
			checker:         allQueued,
			wantRemoved:     []uint64{},
			wantInvalidated: []uint64{0, 1, 2},
		},
		"all rejected bundles are removed": {
			txs: []*types.Transaction{
				bundleTx(0, key),
				transaction(1, gaslimit, key),
				bundleTx(2, key),
				transaction(3, gaslimit, key),
				bundleTx(4, key),
			},
			checker:         allRejected,
			wantRemoved:     []uint64{0, 2, 4},
			wantInvalidated: []uint64{},
		},
		"all rejected bundles are removed, gapped nonces are invalidated": {
			strict: true,
			txs: []*types.Transaction{
				bundleTx(0, key),
				transaction(1, gaslimit, key),
				bundleTx(2, key),
				transaction(3, gaslimit, key),
				bundleTx(4, key),
			},
			checker:         allRejected,
			wantRemoved:     []uint64{0, 2, 4},
			wantInvalidated: []uint64{1, 3},
		},
		"all bundles with nonce >= 2 are queued, strict mode invalidates from nonce 2": {
			strict: true,
			txs: []*types.Transaction{
				bundleTx(0, key),
				transaction(1, gaslimit, key),
				bundleTx(2, key),
				transaction(3, gaslimit, key),
				bundleTx(4, key),
			},
			checker:         invalidateFromNonce(2),
			wantRemoved:     []uint64{},
			wantInvalidated: []uint64{2, 3, 4},
		},
		"all bundles with nonce >= 2 are queued, non-strict mode keeps all": {
			txs: []*types.Transaction{
				bundleTx(0, key),
				transaction(1, gaslimit, key),
				bundleTx(2, key),
				transaction(3, gaslimit, key),
				bundleTx(4, key),
			},
			checker:         invalidateFromNonce(2),
			wantRemoved:     []uint64{},
			wantInvalidated: []uint64{},
		},
		"all bundles with nonce >= 2 are rejected, strict mode removes from nonce 2 and invalidates gapped nonces": {
			strict: true,
			txs: []*types.Transaction{
				bundleTx(0, key),
				transaction(1, gaslimit, key),
				bundleTx(2, key),
				transaction(3, gaslimit, key),
				bundleTx(4, key),
			},
			checker:         rejectFromNonce(2),
			wantRemoved:     []uint64{2, 4},
			wantInvalidated: []uint64{3},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			list := newTxList(test.strict)
			for _, tx := range test.txs {
				list.Add(tx, DefaultTxPoolConfig.PriceBump)
			}

			costLimit := big.NewInt(1e18)
			gasLimit := uint64(1_000_000)
			removed, invalidated := list.Filter(costLimit, gasLimit, nil, test.checker)

			assert.Len(t, removed, len(test.wantRemoved))
			for _, nonce := range test.wantRemoved {
				assert.True(t, slices.ContainsFunc(removed, func(tx *types.Transaction) bool { return tx.Nonce() == nonce }))
			}

			assert.Len(t, invalidated, len(test.wantInvalidated))
			for _, nonce := range test.wantInvalidated {
				assert.True(t, slices.ContainsFunc(invalidated, func(tx *types.Transaction) bool { return tx.Nonce() == nonce }))
			}
		})
	}
}

func TestTxList_Filter_ShortCircuitIfAllTransacitonsAreBellowThresshold(t *testing.T) {
	require := require.New(t)
	key, err := crypto.GenerateKey()
	require.NoError(err)

	list := newTxList(true)
	for i := range 5 {
		list.Add(transaction(uint64(i), 1_000, key), DefaultTxPoolConfig.PriceBump)
	}

	// Cost limit and gas limit are above all transactions, so Filter short-circuits.
	removed, invalidated := list.Filter(big.NewInt(1e18), 1_000_000, nil, nil)
	require.Len(removed, 0)
	require.Len(invalidated, 0)
	require.Equal(5, list.Len(), "all transactions should remain in the list")
}

func TestTxList_Ready_PromotesAllConsecutiveNonces(t *testing.T) {
	require := require.New(t)
	key, err := crypto.GenerateKey()
	require.NoError(err)

	list := newTxList(true)
	for i := range 5 {
		list.Add(transaction(uint64(i), 1_000, key), DefaultTxPoolConfig.PriceBump)
	}

	// All transactions are executable, so all consecutive nonces are promoted.
	ready := list.Ready(0, func(tx *types.Transaction) bool {
		return true
	})
	require.Len(ready, 5)
	for i, tx := range ready {
		require.Equal(uint64(i), tx.Nonce())
	}
	require.True(list.Empty(), "all transactions should have been promoted")
}

func TestTxList_Ready_StopsWithFirstNonExecutableTransaction(t *testing.T) {
	require := require.New(t)
	key, err := crypto.GenerateKey()
	require.NoError(err)

	list := newTxList(true)
	for i := range 5 {
		list.Add(transaction(uint64(i), 1_000, key), DefaultTxPoolConfig.PriceBump)
	}

	// Only transactions with nonce < 3 are executable.
	ready := list.Ready(0, func(tx *types.Transaction) bool {
		return tx.Nonce() < 3
	})
	require.Len(ready, 3)
	for i, tx := range ready {
		require.Equal(uint64(i), tx.Nonce())
	}
	require.Equal(2, list.Len(), "nonce 3 and 4 should remain in the list")
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
		list.Filter(minimumTip, DefaultTxPoolConfig.MinimumTip, nil, nil)
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
