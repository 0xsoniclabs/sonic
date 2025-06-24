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

package evmstore

import (
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

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
