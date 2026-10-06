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

package evmstore

import (
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/logger"
)

func equalStorageReceipts(t *testing.T, expect, got []*types.ReceiptForStorage) {
	assert.EqualValues(t, len(expect), len(got))
	for i := range expect {
		assert.EqualValues(t, expect[i].CumulativeGasUsed, got[i].CumulativeGasUsed)
		assert.EqualValues(t, expect[i].Logs, got[i].Logs)
		assert.EqualValues(t, expect[i].Status, got[i].Status)
	}
}

func TestStoreGetCachedReceipts(t *testing.T) {
	logger.SetTestMode(t)

	block, expect := fakeReceipts()
	store := cachedStore()
	store.SetRawReceipts(block, expect)

	got, _ := store.GetRawReceipts(block)
	assert.EqualValues(t, expect, got)
}

func TestStoreGetNonCachedReceipts(t *testing.T) {
	logger.SetTestMode(t)

	block, expect := fakeReceipts()
	store := nonCachedStore()
	store.SetRawReceipts(block, expect)

	got, _ := store.GetRawReceipts(block)
	equalStorageReceipts(t, expect, got)
}

func BenchmarkStoreGetRawReceipts(b *testing.B) {
	logger.SetTestMode(b)

	b.Run("cache on", func(b *testing.B) {
		benchStoreGetRawReceipts(b, cachedStore())
	})
	b.Run("cache off", func(b *testing.B) {
		benchStoreGetRawReceipts(b, nonCachedStore())
	})
}

func benchStoreGetRawReceipts(b *testing.B, store *Store) {
	block, receipt := fakeReceipts()

	store.SetRawReceipts(block, receipt)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if v, _ := store.GetRawReceipts(block); v == nil {
			b.Fatal("invalid result")
		}
	}
}

func BenchmarkStoreSetRawReceipts(b *testing.B) {
	logger.SetTestMode(b)

	b.Run("cache on", func(b *testing.B) {
		benchStoreSetRawReceipts(b, cachedStore())
	})
	b.Run("cache off", func(b *testing.B) {
		benchStoreSetRawReceipts(b, nonCachedStore())
	})
}

func benchStoreSetRawReceipts(b *testing.B, store *Store) {
	block, receipt := fakeReceipts()

	for i := 0; i < b.N; i++ {
		store.SetRawReceipts(block, receipt)
	}
}

func fakeReceipts() (idx.Block, []*types.ReceiptForStorage) {
	return idx.Block(1),
		[]*types.ReceiptForStorage{
			{
				PostState:         nil,
				Status:            0,
				CumulativeGasUsed: 0,
				Bloom:             types.Bloom{},
				Logs:              []*types.Log{},
				TxHash:            common.Hash{},
				ContractAddress:   common.Address{},
				GasUsed:           0,
				BlockHash:         common.Hash{},
				BlockNumber:       nil,
				TransactionIndex:  0,
			},
		}
}

func TestFindGasUsedOverrides(t *testing.T) {
	tests := map[string]struct {
		receipts types.Receipts
		expected []gasUsedOverride
	}{
		"no receipts": {},
		"consistent gas": {
			receipts: types.Receipts{
				{GasUsed: 10, CumulativeGasUsed: 10},
				{GasUsed: 20, CumulativeGasUsed: 30},
			},
		},
		"surplus gas before first receipt": {
			receipts: types.Receipts{
				{GasUsed: 10, CumulativeGasUsed: 15},
				{GasUsed: 20, CumulativeGasUsed: 35},
			},
			expected: []gasUsedOverride{{Index: 0, GasUsed: 10}},
		},
		"surplus gas between receipts": {
			receipts: types.Receipts{
				{GasUsed: 10, CumulativeGasUsed: 10},
				{GasUsed: 20, CumulativeGasUsed: 35},
				{GasUsed: 5, CumulativeGasUsed: 40},
				{GasUsed: 5, CumulativeGasUsed: 55},
			},
			expected: []gasUsedOverride{{Index: 1, GasUsed: 20}, {Index: 3, GasUsed: 5}},
		},
		"decreasing cumulative gas is ignored": {
			receipts: types.Receipts{
				{GasUsed: 10, CumulativeGasUsed: 10},
				{GasUsed: 20, CumulativeGasUsed: 5},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expected, findGasUsedOverrides(test.receipts))
		})
	}
}

func TestStoreGetReceipts_RestoresGasUsedAfterGasSurplus(t *testing.T) {
	logger.SetTestMode(t)

	for name, store := range map[string]*Store{
		"cached":    cachedStore(),
		"nonCached": nonCachedStore(),
	} {
		t.Run(name, func(t *testing.T) {
			block := idx.Block(1)
			to := common.Address{1}
			txs := types.Transactions{
				types.NewTx(&types.LegacyTx{To: &to}),
				types.NewTx(&types.LegacyTx{To: &to}),
			}
			// 100 units of surplus gas precede the second receipt.
			receipts := types.Receipts{
				{Type: types.LegacyTxType, GasUsed: 10, CumulativeGasUsed: 10},
				{Type: types.LegacyTxType, GasUsed: 20, CumulativeGasUsed: 130},
			}
			store.SetReceipts(block, receipts)

			// Evict the in-memory copy to force loading from the database.
			store.cache.Receipts.Remove(block)

			got := store.GetReceipts(block, params.TestChainConfig, common.Hash{}, 0, nil, nil, txs)
			require.Len(t, got, 2)
			assert.Equal(t, uint64(10), got[0].GasUsed)
			assert.Equal(t, uint64(20), got[1].GasUsed)
			assert.Equal(t, uint64(10), got[0].CumulativeGasUsed)
			assert.Equal(t, uint64(130), got[1].CumulativeGasUsed)
		})
	}
}

func TestStoreSetReceipts_ReplacesOverridesOfPreviousExecution(t *testing.T) {
	logger.SetTestMode(t)

	store := nonCachedStore()
	block := idx.Block(1)
	to := common.Address{1}
	txs := types.Transactions{types.NewTx(&types.LegacyTx{To: &to})}

	store.SetReceipts(block, types.Receipts{
		{Type: types.LegacyTxType, GasUsed: 10, CumulativeGasUsed: 110},
	})
	store.SetReceipts(block, types.Receipts{
		{Type: types.LegacyTxType, GasUsed: 10, CumulativeGasUsed: 10},
	})

	got := store.GetReceipts(block, params.TestChainConfig, common.Hash{}, 0, nil, nil, txs)
	require.Len(t, got, 1)
	assert.Equal(t, uint64(10), got[0].GasUsed)
}

func TestStoreGetReceipts_RestoresGasUsedAfterMultipleGasSurpluses(t *testing.T) {
	logger.SetTestMode(t)

	store := nonCachedStore()
	block := idx.Block(1)
	to := common.Address{1}
	txs := make(types.Transactions, 4)
	for i := range txs {
		txs[i] = types.NewTx(&types.LegacyTx{To: &to})
	}
	receipts := types.Receipts{
		{Type: types.LegacyTxType, GasUsed: 10, CumulativeGasUsed: 10},
		{Type: types.LegacyTxType, GasUsed: 20, CumulativeGasUsed: 80}, // 50 surplus
		{Type: types.LegacyTxType, GasUsed: 30, CumulativeGasUsed: 110},
		{Type: types.LegacyTxType, GasUsed: 40, CumulativeGasUsed: 190}, // 40 surplus
	}
	store.SetReceipts(block, receipts)

	got := store.GetReceipts(block, params.TestChainConfig, common.Hash{}, 0, nil, nil, txs)
	require.Len(t, got, 4)
	for i, want := range []uint64{10, 20, 30, 40} {
		assert.Equal(t, want, got[i].GasUsed, "receipt %d", i)
	}
	for i, want := range []uint64{10, 80, 110, 190} {
		assert.Equal(t, want, got[i].CumulativeGasUsed, "receipt %d", i)
	}
}

func TestStoreGetReceipts_NoOverrideForGasSurplusAfterLastReceipt(t *testing.T) {
	logger.SetTestMode(t)

	store := nonCachedStore()
	block := idx.Block(1)
	to := common.Address{1}
	txs := types.Transactions{types.NewTx(&types.LegacyTx{To: &to})}

	// Surplus gas after the last receipt is only visible in the block header
	// and does not affect any receipt.
	store.SetReceipts(block, types.Receipts{
		{Type: types.LegacyTxType, GasUsed: 10, CumulativeGasUsed: 10},
	})

	has, err := store.table.ReceiptGasUsedOverrides.Has(block.Bytes())
	require.NoError(t, err)
	assert.False(t, has)

	got := store.GetReceipts(block, params.TestChainConfig, common.Hash{}, 0, nil, nil, txs)
	require.Len(t, got, 1)
	assert.Equal(t, uint64(10), got[0].GasUsed)
}

func TestStoreGetReceipts_IgnoresOutOfRangeOverride(t *testing.T) {
	logger.SetTestMode(t)

	store := nonCachedStore()
	block := idx.Block(1)
	to := common.Address{1}
	txs := types.Transactions{types.NewTx(&types.LegacyTx{To: &to})}

	store.SetReceipts(block, types.Receipts{
		{Type: types.LegacyTxType, GasUsed: 10, CumulativeGasUsed: 10},
	})
	store.setGasUsedOverrides(block, []gasUsedOverride{{Index: 5, GasUsed: 1}})

	got := store.GetReceipts(block, params.TestChainConfig, common.Hash{}, 0, nil, nil, txs)
	require.Len(t, got, 1)
	assert.Equal(t, uint64(10), got[0].GasUsed)
}

func TestStoreSetReceipts_EmptyReceiptsStoreNoOverrides(t *testing.T) {
	logger.SetTestMode(t)

	store := nonCachedStore()
	block := idx.Block(1)

	store.SetReceipts(block, types.Receipts{})

	has, err := store.table.ReceiptGasUsedOverrides.Has(block.Bytes())
	require.NoError(t, err)
	assert.False(t, has)

	got := store.GetReceipts(block, params.TestChainConfig, common.Hash{}, 0, nil, nil, types.Transactions{})
	assert.Empty(t, got)
}
